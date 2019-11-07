package broker_test

import (
	"context"
	"errors"
	"fmt"
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

func (ts *bindServiceTestSuite) FixBindOperation(tpe internal.OperationType, state internal.OperationState) internal.BindOperation {
	return *ts.Exp.NewBindOperation(tpe, state)
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

func (ts *bindServiceTestSuite) FixGetBindingRequest() osb.GetBindingRequest {
	return osb.GetBindingRequest{
		InstanceID: string(ts.Exp.InstanceID),
		BindingID:  string(ts.Exp.BindingID),
	}
}

func (ts *bindServiceTestSuite) FixBindingLastOperationRequest() *osb.BindingLastOperationRequest {
	opKey := osb.OperationKey(ts.Exp.OperationID)
	return &osb.BindingLastOperationRequest{
		InstanceID:   string(ts.Exp.InstanceID),
		BindingID:    string(ts.Exp.BindingID),
		OperationKey: &opKey,
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
	expBindOp := ts.FixBindOperation(internal.OperationTypeCreate, internal.OperationStateInProgress)
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

func TestBindServiceBindFailureWhenNotBindedOnIsBinded(t *testing.T) {
	//given
	ts := newBindServiceTestSuite(t)
	ts.SetUp()

	asMock := &automock.AddonStorage{}
	cgMock := &automock.ChartGetter{}
	isMock := &automock.InstanceStorage{}
	ibdsMock := &automock.InstanceBindDataStorage{}
	bosMock := &automock.BindOperationStorage{}
	rendererMock := &automock.BindTemplateRenderer{}
	resolverMock := &automock.BindTemplateResolver{}

	bsgMock := &automock.BindStateGetter{}
	defer bsgMock.AssertExpectations(t)
	expIsBindedError := errors.New("fake-is-binded-error")
	bsgMock.On("IsBinded", ts.Exp.InstanceID, ts.Exp.BindingID).Return(false, expIsBindedError).Once()

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
	assert.NotNil(t, err)
	assert.Nil(t, resp)

}

func TestBindServiceBindFailureAsyncWhenNotBindedOnChartGet(t *testing.T) {
	//given
	ts := newBindServiceTestSuite(t)
	ts.SetUp()

	asMock := &automock.AddonStorage{}
	defer asMock.AssertExpectations(t)
	expAddon := ts.FixAddon()
	asMock.On("GetByID", internal.ClusterWide, ts.Exp.Addon.ID).Return(&expAddon, nil).Once()

	cgMock := &automock.ChartGetter{}
	defer cgMock.AssertExpectations(t)
	expChartError := errors.New("fake-chart-error")
	cgMock.On("Get", internal.ClusterWide, ts.Exp.Chart.Name, ts.Exp.Chart.Version).Return(nil, expChartError).Once()

	isMock := &automock.InstanceStorage{}
	defer isMock.AssertExpectations(t)
	expInstance := ts.FixInstanceWithInfo()
	params := jsonhash.HashS(ts.FixBindRequest().Parameters)
	expInstance.ParamsHash = params
	isMock.On("Get", ts.Exp.InstanceID).Return(&expInstance, nil).Once()

	ibdsMock := &automock.InstanceBindDataStorage{}

	bosMock := &automock.BindOperationStorage{}
	defer bosMock.AssertExpectations(t)
	expBindOp := ts.FixBindOperation(internal.OperationTypeCreate, internal.OperationStateInProgress)
	expBindOp.ParamsHash = params
	bosMock.On("Insert", &expBindOp).Return(nil).Once()
	operationFailed := make(chan struct{})
	bosMock.On("UpdateStateDesc", ts.Exp.InstanceID, ts.Exp.BindingID, ts.Exp.OperationID, internal.OperationStateFailed, mock.Anything).Return(nil).Once().
		Run(func(mock.Arguments) { close(operationFailed) })

	rendererMock := &automock.BindTemplateRenderer{}
	resolverMock := &automock.BindTemplateResolver{}

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
	case <-operationFailed:
	case <-time.After(time.Millisecond * 100):
		t.Fatal("timeout on operation succeeded")
	}

	select {
	case <-testHookCalled:
	default:
		t.Fatal("async test hook not called")
	}
}

func TestBindServiceBindFailureWhenNotBindedAsyncOnRenderAndResolve(t *testing.T) {
	//given
	ts := newBindServiceTestSuite(t)
	ts.SetUp()

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

	bosMock := &automock.BindOperationStorage{}
	defer bosMock.AssertExpectations(t)
	expBindOp := ts.FixBindOperation(internal.OperationTypeCreate, internal.OperationStateInProgress)
	expBindOp.ParamsHash = params
	bosMock.On("Insert", &expBindOp).Return(nil).Once()
	operationFailed := make(chan struct{})
	bosMock.On("UpdateStateDesc", ts.Exp.InstanceID, ts.Exp.BindingID, ts.Exp.OperationID, internal.OperationStateFailed, mock.Anything).Return(nil).Once().
		Run(func(mock.Arguments) { close(operationFailed) })

	rendererMock := &automock.BindTemplateRenderer{}
	defer rendererMock.AssertExpectations(t)
	expRenError := errors.New("fake-renderer-error")
	rendererMock.On("Render", ts.Exp.AddonPlan.BindTemplate, &expInstance, &expChart).Return(nil, expRenError).Once()

	resolverMock := &automock.BindTemplateResolver{}

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
	case <-operationFailed:
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

func TestBindServiceBindFailureWhenBindedOnCompareBindParams(t *testing.T) {
	//given
	ts := newBindServiceTestSuite(t)
	ts.SetUp()

	asMock := &automock.AddonStorage{}
	cgMock := &automock.ChartGetter{}
	isMock := &automock.InstanceStorage{}

	ibdsMock := &automock.InstanceBindDataStorage{}

	bosMock := &automock.BindOperationStorage{}
	defer bosMock.AssertExpectations(t)
	expBindOpCollection := ts.FixBindOpCollection()

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
	req := ts.FixBindRequest()

	//when
	resp, err := svc.Bind(ctx, osbCtx, &req)

	//then
	assert.NotNil(t, err)
	assert.Nil(t, resp)
}

func TestBindServiceBindFailureWhenBindedOnGetIbd(t *testing.T) {
	//given
	ts := newBindServiceTestSuite(t)
	ts.SetUp()

	asMock := &automock.AddonStorage{}
	cgMock := &automock.ChartGetter{}
	isMock := &automock.InstanceStorage{}

	ibdsMock := &automock.InstanceBindDataStorage{}
	ibdsMock.AssertExpectations(t)
	expIbdGetError := errors.New("fake-ibd-get-error")
	ibdsMock.On("Get", ts.Exp.InstanceID).Return(nil, expIbdGetError).Once()

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
	assert.NotNil(t, err)
	assert.Nil(t, resp)
}

func TestBindServiceBindSuccessAsyncWhenBindingInProgress(t *testing.T) {
	//given
	ts := newBindServiceTestSuite(t)
	ts.SetUp()

	asMock := &automock.AddonStorage{}
	cgMock := &automock.ChartGetter{}
	isMock := &automock.InstanceStorage{}

	ibdsMock := &automock.InstanceBindDataStorage{}

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
	bsgMock.On("IsBinded", ts.Exp.InstanceID, ts.Exp.BindingID).Return(false, nil).Once()
	bsgMock.On("IsBindingInProgress", ts.Exp.InstanceID, ts.Exp.BindingID).Return(ts.Exp.OperationID, true, nil).Once()

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
	assert.True(t, resp.Async)
	assert.EqualValues(t, ts.Exp.OperationID, *resp.OperationKey)
}

func TestBindServiceBindFailureWhenBindingInProgressOnCheckingIsBindingInProgress(t *testing.T) {
	//given
	ts := newBindServiceTestSuite(t)
	ts.SetUp()

	asMock := &automock.AddonStorage{}
	cgMock := &automock.ChartGetter{}
	isMock := &automock.InstanceStorage{}
	ibdsMock := &automock.InstanceBindDataStorage{}
	bosMock := &automock.BindOperationStorage{}
	rendererMock := &automock.BindTemplateRenderer{}
	resolverMock := &automock.BindTemplateResolver{}

	bsgMock := &automock.BindStateGetter{}
	defer bsgMock.AssertExpectations(t)
	bsgMock.On("IsBinded", ts.Exp.InstanceID, ts.Exp.BindingID).Return(false, nil).Once()
	expIsBindInProgError := errors.New("fake-is-binding-in-progress-error")
	bsgMock.On("IsBindingInProgress", ts.Exp.InstanceID, ts.Exp.BindingID).Return(internal.OperationID(""), false, expIsBindInProgError).Once()

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
	req := ts.FixBindRequest()

	//when
	resp, err := svc.Bind(ctx, osbCtx, &req)

	//then
	assert.NotNil(t, err)
	assert.Nil(t, resp)
}

func TestBindServiceBindFailureWhenBindingInProgressOnCompareBindParams(t *testing.T) {
	//given
	ts := newBindServiceTestSuite(t)
	ts.SetUp()

	asMock := &automock.AddonStorage{}
	cgMock := &automock.ChartGetter{}
	isMock := &automock.InstanceStorage{}
	ibdsMock := &automock.InstanceBindDataStorage{}

	bosMock := &automock.BindOperationStorage{}
	defer bosMock.AssertExpectations(t)
	expBindOpCollection := ts.FixBindOpCollection()
	bosMock.On("GetAll", ts.Exp.InstanceID).Return(expBindOpCollection, nil).Once()

	rendererMock := &automock.BindTemplateRenderer{}
	resolverMock := &automock.BindTemplateResolver{}

	bsgMock := &automock.BindStateGetter{}
	defer bsgMock.AssertExpectations(t)
	bsgMock.On("IsBinded", ts.Exp.InstanceID, ts.Exp.BindingID).Return(false, nil).Once()
	bsgMock.On("IsBindingInProgress", ts.Exp.InstanceID, ts.Exp.BindingID).Return(ts.Exp.OperationID, true, nil).Once()

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
	req := ts.FixBindRequest()

	//when
	resp, err := svc.Bind(ctx, osbCtx, &req)

	//then
	assert.NotNil(t, err)
	assert.Nil(t, resp)
}

func TestBindServiceBindFailureWhenGivenIncorrectParameters(t *testing.T) {
	//given
	ts := newBindServiceTestSuite(t)
	ts.SetUp()
	asMock := &automock.AddonStorage{}
	cgMock := &automock.ChartGetter{}
	isMock := &automock.InstanceStorage{}
	ibdsMock := &automock.InstanceBindDataStorage{}
	bosMock := &automock.BindOperationStorage{}
	rendererMock := &automock.BindTemplateRenderer{}
	resolverMock := &automock.BindTemplateResolver{}
	bsgMock := &automock.BindStateGetter{}
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
	req.Parameters = map[string]interface{}{
		"param1": 121,
		"param2": 132,
	}

	//when
	resp, err := svc.Bind(ctx, osbCtx, &req)

	//then
	assert.NotNil(t, err)
	assert.EqualValues(t, "helm-broker does not support configuration options for the service binding", *err.ErrorMessage)
	assert.Nil(t, resp)

	//given
	req = ts.FixBindRequest()
	req.AcceptsIncomplete = false

	//when
	resp, err = svc.Bind(ctx, osbCtx, &req)

	//then
	assert.NotNil(t, err)

	assert.EqualValues(t, "asynchronous operation mode required", *err.ErrorMessage)
	assert.Nil(t, resp)
}

func TestBindServiceGetLastBindOperationSuccessWhenBinded(t *testing.T) {
	ts := newBindServiceTestSuite(t)
	ts.SetUp()

	asMock := &automock.AddonStorage{}
	cgMock := &automock.ChartGetter{}
	isMock := &automock.InstanceStorage{}
	ibdsMock := &automock.InstanceBindDataStorage{}

	bosMock := &automock.BindOperationStorage{}
	defer bosMock.AssertExpectations(t)
	expBindOp := ts.FixBindOperation(internal.OperationTypeCreate, internal.OperationStateSucceeded)
	bosMock.On("Get", ts.Exp.InstanceID, ts.Exp.BindingID, ts.Exp.OperationID).Return(&expBindOp, nil).Once()

	rendererMock := &automock.BindTemplateRenderer{}
	resolverMock := &automock.BindTemplateResolver{}
	bsgMock := &automock.BindStateGetter{}

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
	req := ts.FixBindingLastOperationRequest()

	//when
	resp, err := svc.GetLastBindOperation(ctx, osbCtx, req)

	//then
	expResp := osb.LastOperationResponse{
		State:       osb.LastOperationState(internal.OperationStateSucceeded),
		Description: nil,
	}
	assert.Nil(t, err)
	assert.EqualValues(t, expResp.State, resp.State)
	assert.EqualValues(t, expResp.Description, resp.Description)

}

func TestBindServiceGetLastBindOperationSuccessWhenBindingInProgress(t *testing.T) {
	ts := newBindServiceTestSuite(t)
	ts.SetUp()

	asMock := &automock.AddonStorage{}
	cgMock := &automock.ChartGetter{}
	isMock := &automock.InstanceStorage{}
	ibdsMock := &automock.InstanceBindDataStorage{}

	bosMock := &automock.BindOperationStorage{}
	defer bosMock.AssertExpectations(t)
	expBindOp := ts.FixBindOperation(internal.OperationTypeCreate, internal.OperationStateInProgress)
	bosMock.On("Get", ts.Exp.InstanceID, ts.Exp.BindingID, ts.Exp.OperationID).Return(&expBindOp, nil).Once()

	rendererMock := &automock.BindTemplateRenderer{}
	resolverMock := &automock.BindTemplateResolver{}
	bsgMock := &automock.BindStateGetter{}

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
	req := ts.FixBindingLastOperationRequest()

	//when
	resp, err := svc.GetLastBindOperation(ctx, osbCtx, req)

	//then
	expResp := osb.LastOperationResponse{
		State:       osb.LastOperationState(internal.OperationStateInProgress),
		Description: nil,
	}
	assert.Nil(t, err)
	assert.EqualValues(t, expResp.State, resp.State)
	assert.EqualValues(t, expResp.Description, resp.Description)

}

func TestBindServiceGetLastBindOperationFailureOnGetBindOp(t *testing.T) {
	ts := newBindServiceTestSuite(t)
	ts.SetUp()

	asMock := &automock.AddonStorage{}
	cgMock := &automock.ChartGetter{}
	isMock := &automock.InstanceStorage{}
	ibdsMock := &automock.InstanceBindDataStorage{}

	bosMock := &automock.BindOperationStorage{}
	defer bosMock.AssertExpectations(t)
	expBindOpError := errors.New("fake-get-bind-op-error")
	bosMock.On("Get", ts.Exp.InstanceID, ts.Exp.BindingID, ts.Exp.OperationID).Return(nil, expBindOpError).Once()

	rendererMock := &automock.BindTemplateRenderer{}
	resolverMock := &automock.BindTemplateResolver{}
	bsgMock := &automock.BindStateGetter{}

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
	req := ts.FixBindingLastOperationRequest()

	//when
	resp, err := svc.GetLastBindOperation(ctx, osbCtx, req)

	//then
	assert.NotNil(t, err)
	assert.Nil(t, resp)

}

func TestBindServiceGetServiceBindingSuccessWhenBinded(t *testing.T) {
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

	rendererMock := &automock.BindTemplateRenderer{}
	resolverMock := &automock.BindTemplateResolver{}

	bsgMock := &automock.BindStateGetter{}
	defer bsgMock.AssertExpectations(t)
	bsgMock.On("IsBindingInProgress", ts.Exp.InstanceID, ts.Exp.BindingID).Return(ts.Exp.OperationID, false, nil).Once()

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
	req := ts.FixGetBindingRequest()

	//when
	resp, err := svc.GetServiceBinding(ctx, osbCtx, &req)

	//then
	assert.Nil(t, err)
	assert.EqualValues(t, map[string]interface{}{
		"password": "secret",
	}, resp.Credentials)
}

func TestBindServiceGetServiceBindingSuccessWhenBindedOnIbdGet(t *testing.T) {
	ts := newBindServiceTestSuite(t)
	ts.SetUp()

	asMock := &automock.AddonStorage{}
	cgMock := &automock.ChartGetter{}
	isMock := &automock.InstanceStorage{}

	ibdsMock := &automock.InstanceBindDataStorage{}
	defer ibdsMock.AssertExpectations(t)
	expIbdGetErr := errors.New("fake-ibd-get-error")
	ibdsMock.On("Get", ts.Exp.InstanceID).Return(nil, expIbdGetErr).Once()

	bosMock := &automock.BindOperationStorage{}
	rendererMock := &automock.BindTemplateRenderer{}
	resolverMock := &automock.BindTemplateResolver{}

	bsgMock := &automock.BindStateGetter{}
	defer bsgMock.AssertExpectations(t)
	bsgMock.On("IsBindingInProgress", ts.Exp.InstanceID, ts.Exp.BindingID).Return(ts.Exp.OperationID, false, nil).Once()

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
	req := ts.FixGetBindingRequest()

	//when
	resp, err := svc.GetServiceBinding(ctx, osbCtx, &req)

	//then
	assert.NotNil(t, err)
	assert.Nil(t, resp)
}

func TestBindServiceGetServiceBindingFailureWhenBindingInProgress(t *testing.T) {
	ts := newBindServiceTestSuite(t)
	ts.SetUp()

	asMock := &automock.AddonStorage{}
	cgMock := &automock.ChartGetter{}
	isMock := &automock.InstanceStorage{}
	ibdsMock := &automock.InstanceBindDataStorage{}
	bosMock := &automock.BindOperationStorage{}
	rendererMock := &automock.BindTemplateRenderer{}
	resolverMock := &automock.BindTemplateResolver{}

	bsgMock := &automock.BindStateGetter{}
	defer bsgMock.AssertExpectations(t)
	bsgMock.On("IsBindingInProgress", ts.Exp.InstanceID, ts.Exp.BindingID).Return(ts.Exp.OperationID, true, nil).Once()

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
	req := ts.FixGetBindingRequest()

	//when
	resp, err := svc.GetServiceBinding(ctx, osbCtx, &req)

	//then
	assert.NotNil(t, err)
	expErrMsg := fmt.Sprintf("service binding id: %q is in progress", ts.Exp.OperationID)
	assert.EqualValues(t, expErrMsg, *err.ErrorMessage)
	assert.Nil(t, resp)
}

func TestBindServiceGetServiceBindingFailureOnIsBindingInProgressCheck(t *testing.T) {
	ts := newBindServiceTestSuite(t)
	ts.SetUp()

	asMock := &automock.AddonStorage{}
	cgMock := &automock.ChartGetter{}
	isMock := &automock.InstanceStorage{}
	ibdsMock := &automock.InstanceBindDataStorage{}
	bosMock := &automock.BindOperationStorage{}
	rendererMock := &automock.BindTemplateRenderer{}
	resolverMock := &automock.BindTemplateResolver{}

	bsgMock := &automock.BindStateGetter{}
	defer bsgMock.AssertExpectations(t)
	expIsBindInProgErr := errors.New("fake-is-binding-in-progress-error")
	bsgMock.On("IsBindingInProgress", ts.Exp.InstanceID, ts.Exp.BindingID).Return(internal.OperationID(""), false, expIsBindInProgErr).Once()

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
	req := ts.FixGetBindingRequest()

	//when
	resp, err := svc.GetServiceBinding(ctx, osbCtx, &req)

	//then
	assert.NotNil(t, err)
	assert.Nil(t, resp)
}
