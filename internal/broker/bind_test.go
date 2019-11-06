package broker_test

import (
	"context"
	"testing"
	"time"

	jsonhash "github.com/komkom/go-jsonhash"
	"github.com/kyma-project/helm-broker/internal"
	"github.com/kyma-project/helm-broker/internal/bind"
	osb "github.com/pmorie/go-open-service-broker-client/v2"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"k8s.io/helm/pkg/proto/hapi/chart"

	"github.com/kyma-project/helm-broker/internal/broker"
	"github.com/kyma-project/helm-broker/internal/broker/automock"
)

//type bindServiceTestCase struct {
//	addonStorageMock         *automock.AddonStorage
//	chartGetterMock          *automock.ChartGetter
//	instanceGetterMock       *automock.InstanceStorage
//	bindTemplateRendererMock *automock.BindTemplateRenderer
//	bindTemplateResolverMock *automock.BindTemplateResolver
//}
//
//func newBindTC() *bindServiceTestCase {
//	return &bindServiceTestCase{
//		addonStorageMock:         &automock.AddonStorage{},
//		chartGetterMock:          &automock.ChartGetter{},
//		instanceGetterMock:       &automock.InstanceStorage{},
//		bindTemplateRendererMock: &automock.BindTemplateRenderer{},
//		bindTemplateResolverMock: &automock.BindTemplateResolver{},
//	}
//}

func newBindServiceTestSuite(t *testing.T) *bindServiceTestSuite {
	return &bindServiceTestSuite{t: t}
}

type bindServiceTestSuite struct {
	t   *testing.T
	Exp expAll
}

func (ts *bindServiceTestSuite) SetUp() {
	ts.Exp.Populate()
}

func (ts *bindServiceTestSuite) FixAddon() internal.Addon {
	return *ts.Exp.NewAddon()
}

func (ts *bindServiceTestSuite) FixChart() chart.Chart {
	return *ts.Exp.NewChart()
}

func (ts *bindServiceTestSuite) FixInstanceWithInfo() internal.Instance {
	return *ts.Exp.NewInstanceWithInfo()
}

func (ts *bindServiceTestSuite) FixInstanceBindData(cr internal.InstanceCredentials) internal.InstanceBindData {
	return *ts.Exp.NewInstanceBindData(cr)
}

func (ts *bindServiceTestSuite) FixInstanceCredentials() internal.InstanceCredentials {
	return *ts.Exp.NewInstanceCredentials()
}

func (ts *bindServiceTestSuite) FixBindOperation() internal.BindOperation {
	return *ts.Exp.NewBindOperation(internal.OperationTypeCreate, internal.OperationStateInProgress)
}

func (ts *bindServiceTestSuite) FixBindRequest() osb.BindRequest {
	return osb.BindRequest{
		BindingID:  string(ts.Exp.BindingID),
		InstanceID: string(ts.Exp.InstanceID),
		ServiceID:  string(ts.Exp.Service.ID),
		PlanID:     string(ts.Exp.ServicePlan.ID),
		Parameters: make(internal.ChartValues),
		Context: map[string]interface{}{
			"namespace": string(ts.Exp.Namespace),
		},
		AcceptsIncomplete: true,
	}
}

func (ts *bindServiceTestSuite) FixLastBindOperationRequest(key *osb.OperationKey) *osb.BindingLastOperationRequest {
	return &osb.BindingLastOperationRequest{
		InstanceID:   string(ts.Exp.InstanceID),
		BindingID:    string(ts.Exp.BindingID),
		OperationKey: key,
	}
}

func (ts *bindServiceTestSuite) FixBindOpCollection() []*internal.BindOperation {
	return ts.Exp.NewBindOperationCollection()
}

func TestBindServiceBindSuccessAsyncWhenNotBinded(t *testing.T) {
	//given
	ts := newBindServiceTestSuite(t)
	ts.SetUp()

	expCreds := ts.FixInstanceCredentials()

	asMock := &automock.AddonStorage{}
	defer asMock.AssertExpectations(t)
	expAddon := ts.FixAddon()
	asMock.On("GetByID", internal.ClusterWide, ts.Exp.Addon.ID).Return(&expAddon, nil).Once()

	cgMock := &automock.ChartGetter{}
	defer cgMock.AssertExpectations(t)
	expChart := ts.FixChart()
	cgMock.On("Get", internal.ClusterWide, ts.Exp.Chart.Name, ts.Exp.Chart.Version).Return(&expChart, nil).Once()

	isMock := &automock.InstanceStorage{}
	defer isMock.AssertExpectations(t)
	expInstance := ts.FixInstanceWithInfo()
	params := jsonhash.HashS(ts.FixBindRequest().Parameters)
	expInstance.ParamsHash = params
	isMock.On("Get", ts.Exp.InstanceID).Return(&expInstance, nil).Once()

	ibdsMock := &automock.InstanceBindDataStorage{}
	defer ibdsMock.AssertExpectations(t)
	expIbd := ts.FixInstanceBindData(expCreds)
	ibdsMock.On("Insert", &expIbd).Return(nil).Once()

	bosMock := &automock.BindOperationStorage{}
	defer bosMock.AssertExpectations(t)
	expBindOp := ts.FixBindOperation()
	expBindOp.ParamsHash = params
	bosMock.On("Insert", &expBindOp).Return(nil).Once()
	operationSucceeded := make(chan struct{})
	bosMock.On("UpdateStateDesc", ts.Exp.InstanceID, ts.Exp.BindingID, ts.Exp.OperationID, internal.OperationStateSucceeded, mock.Anything).Return(nil).Once().
		Run(func(mock.Arguments) { close(operationSucceeded) })

	rendererMock := &automock.BindTemplateRenderer{}
	defer rendererMock.AssertExpectations(t)
	expRendered := bind.RenderedBindYAML{}
	rendererMock.On("Render", ts.Exp.AddonPlan.BindTemplate, &expInstance, &expChart).Return(expRendered, nil)

	resolverMock := &automock.BindTemplateResolver{}
	defer resolverMock.AssertExpectations(t)
	expResolved := bind.ResolveOutput{Credentials: expCreds}
	resolverMock.On("Resolve", expRendered, ts.Exp.Namespace).Return(&expResolved, nil).Once()

	bsgMock := &automock.BindStateGetter{}
	defer bsgMock.AssertExpectations(t)
	bsgMock.On("IsBinded", ts.Exp.InstanceID, ts.Exp.BindingID).Return(false, nil).Once()
	bsgMock.On("IsBindingInProgress", ts.Exp.InstanceID, ts.Exp.BindingID).Return(internal.OperationID(""), false, nil).Once()

	oipFake := func() (internal.OperationID, error) {
		return ts.Exp.OperationID, nil
	}

	testHookCalled := make(chan struct{})

	svc := broker.NewBindService(asMock, cgMock, isMock, ibdsMock, ibdsMock, rendererMock, resolverMock,
		bsgMock, bosMock, bosMock, bosMock, bosMock, oipFake).
		WithTestHookOnAsyncCalled(func(opID internal.OperationID) {
			assert.Equal(t, ts.Exp.OperationID, opID)
			close(testHookCalled)
		})

	ctx := context.Background()
	osbCtx := *broker.NewOSBContext("", "v1")
	req := ts.FixBindRequest()

	//when
	resp, err := svc.Bind(ctx, osbCtx, &req)

	//then
	assert.Nil(t, err)
	assert.True(t, resp.Async)
	assert.EqualValues(t, ts.Exp.OperationID, *resp.OperationKey)

	select {
	case <-operationSucceeded:
	case <-time.After(time.Millisecond * 100):
		t.Fatal("timeout on operation succeeded")
	}

	select {
	case <-testHookCalled:
	default:
		t.Fatal("async test hook not called")
	}
}

func TestBindServiceBindSuccessAsyncWhenBinded(t *testing.T) {
	//given
	ts := newBindServiceTestSuite(t)
	ts.SetUp()

	expCreds := ts.FixInstanceCredentials()

	asMock := &automock.AddonStorage{}
	cgMock := &automock.ChartGetter{}
	isMock := &automock.InstanceStorage{}

	ibdsMock := &automock.InstanceBindDataStorage{}
	defer ibdsMock.AssertExpectations(t)
	expIbd := ts.FixInstanceBindData(expCreds)
	ibdsMock.On("Get", ts.Exp.InstanceID).Return(&expIbd, nil).Once()

	bosMock := &automock.BindOperationStorage{}
	defer bosMock.AssertExpectations(t)
	expBindOpCollection := ts.FixBindOpCollection()

	req := ts.FixBindRequest()

	expBindOpCollection[0].ParamsHash = jsonhash.HashS(req.Parameters)
	bosMock.On("GetAll", ts.Exp.InstanceID).Return(expBindOpCollection, nil).Once()

	rendererMock := &automock.BindTemplateRenderer{}
	resolverMock := &automock.BindTemplateResolver{}

	bsgMock := &automock.BindStateGetter{}
	defer bsgMock.AssertExpectations(t)
	bsgMock.On("IsBinded", ts.Exp.InstanceID, ts.Exp.BindingID).Return(true, nil).Once()

	oipFake := func() (internal.OperationID, error) {
		return ts.Exp.OperationID, nil
	}

	testHookCalled := make(chan struct{})

	svc := broker.NewBindService(asMock, cgMock, isMock, ibdsMock, ibdsMock,
		rendererMock, resolverMock, bsgMock, bosMock, bosMock, bosMock, bosMock, oipFake).
		WithTestHookOnAsyncCalled(func(opID internal.OperationID) {
			assert.Equal(t, ts.Exp.OperationID, opID)
			close(testHookCalled)
		})

	ctx := context.Background()
	osbCtx := *broker.NewOSBContext("", "v1")

	//when
	resp, err := svc.Bind(ctx, osbCtx, &req)

	//then
	assert.Nil(t, err)
	assert.False(t, resp.Async)
	assert.EqualValues(t, map[string]interface{}{
		"password": "secret",
	}, resp.Credentials)
}

//func TestBindServiceBindSuccess(t *testing.T) {
//	// given
//	tc := newBindTC()
//	defer tc.AssertExpectations(t)
//	fixID := tc.FixBindRequest().InstanceID
//	expCreds := map[string]string{
//		"password": "secret",
//	}
//	tc.ExpectOnGet(fixID, expCreds)
//
//	addonGetter := &automock.AddonStorage{}
//	chartGetter := &automock.ChartGetter{}
//	bindTemplateRenderer := &automock.BindTemplateRenderer{}
//	bindTemplateResolver := &automock.BindTemplateResolver{}
//	instanceGetter := &automock.InstanceStorage{}
//
//	oipFake := func() (internal.OperationID, error) {
//		return "test-op-id", nil
//	}
//
//	svc := broker.NewBindService(addonGetter, chartGetter, instanceGetter, bindTemplateRenderer, bindTemplateResolver, oipFake)
//	osbCtx := broker.NewOSBContext("not", "important")
//
//	// when
//	resp, err := svc.Bind(context.Background(), *osbCtx, tc.FixBindRequest())
//
//	// then
//	require.NoError(t, err)
//	assert.Equal(t, map[string]interface{}{
//		"password": "secret",
//	}, resp.Credentials)
//	assert.Nil(t, resp.RouteServiceURL)
//	assert.Nil(t, resp.SyslogDrainURL)
//	assert.Nil(t, resp.VolumeMounts)
//}
//
//func TestBindServiceBindFailure(t *testing.T) {
//	t.Run("On service Get", func(t *testing.T) {
//		// given
//		tc := newBindTC()
//		defer tc.AssertExpectations(t)
//		fixID := tc.FixBindRequest().InstanceID
//		fixErr := errors.New("Get ERR")
//		tc.ExpectOnGetError(fixID, fixErr)
//
//		addonGetter := &automock.AddonStorage{}
//		chartGetter := &automock.ChartGetter{}
//		bindTemplateRenderer := &automock.BindTemplateRenderer{}
//		bindTemplateResolver := &automock.BindTemplateResolver{}
//		instanceGetter := &automock.InstanceStorage{}
//
//		oipFake := func() (internal.OperationID, error) {
//			return "test-op-id", nil
//		}
//
//		svc := broker.NewBindService(addonGetter, chartGetter, instanceGetter, bindTemplateRenderer, bindTemplateResolver, oipFake)
//		osbCtx := broker.NewOSBContext("not", "important")
//
//		// when
//		resp, err := svc.Bind(context.Background(), *osbCtx, tc.FixBindRequest())
//
//		// then
//		require.EqualError(t, err, fmt.Sprintf("while getting bind data from storage for instance id: %q: %v", fixID, fixErr.Error()))
//		assert.Nil(t, resp)
//	})
//
//	t.Run("On unexpected req params", func(t *testing.T) {
//		// given
//		tc := newBindTC()
//		fixReq := tc.FixBindRequest()
//		fixReq.Parameters = map[string]interface{}{
//			"some-key": "some-value",
//		}
//
//		addonGetter := &automock.AddonStorage{}
//		chartGetter := &automock.ChartGetter{}
//		bindTemplateRenderer := &automock.BindTemplateRenderer{}
//		bindTemplateResolver := &automock.BindTemplateResolver{}
//		instanceGetter := &automock.InstanceStorage{}
//
//		oipFake := func() (internal.OperationID, error) {
//			return "test-op-id", nil
//		}
//
//		svc := broker.NewBindService(addonGetter, chartGetter, instanceGetter, bindTemplateRenderer, bindTemplateResolver, oipFake)
//		osbCtx := broker.NewOSBContext("not", "important")
//
//		// when
//		resp, err := svc.Bind(context.Background(), *osbCtx, fixReq)
//
//		// then
//		assert.EqualError(t, err, "helm-broker does not support configuration options for the service binding")
//		assert.Zero(t, resp)
//	})
//}

//func (tc *bindServiceTestCase) ExpectOnGet(iID string, creds map[string]string) {
//	tc.InstanceBindDataGetter.On("Get", internal.InstanceID(iID)).
//		Return(&internal.InstanceBindData{
//			InstanceID:  internal.InstanceID(iID),
//			Credentials: internal.InstanceCredentials(creds),
//		}, nil).Once()
//}
//
//func (tc *bindServiceTestCase) ExpectOnGetError(iID string, err error) {
//	tc.InstanceBindDataGetter.On("Get", internal.InstanceID(iID)).
//		Return(nil, err).Once()
//}
