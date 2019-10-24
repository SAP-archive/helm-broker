package broker

import (
	"context"
	"fmt"
	"net/http"
	"sync"

	"github.com/komkom/go-jsonhash"
	"github.com/pkg/errors"
	osb "github.com/pmorie/go-open-service-broker-client/v2"
	"github.com/sirupsen/logrus"
	"k8s.io/helm/pkg/proto/hapi/chart"

	"github.com/kyma-project/helm-broker/internal"
)

type bindService struct {
	addonIDGetter            addonIDGetter
	chartGetter              chartGetter
	instanceBindDataGetter instanceBindDataGetter
	instanceGetter           instanceGetter
	bindTemplateRenderer     bindTemplateRenderer
	bindTemplateResolver     bindTemplateResolver
	operationInserter        operationInserter
	operationUpdater         operationUpdater
	operationIDProvider      func() (internal.OperationID, error)
	mu                       sync.Mutex

	log *logrus.Entry

	testHookAsyncCalled func(internal.OperationID)
}

var resolvedBindData internal.InstanceBindData

func (svc *bindService) Bind(ctx context.Context, osbCtx OsbContext, req *osb.BindRequest) (*osb.BindResponse, error) {
	if len(req.Parameters) > 0 {
		return nil, fmt.Errorf("helm-broker does not support configuration options for the service binding")
	}

	if !req.AcceptsIncomplete {
		return nil, &osb.HTTPStatusCodeError{StatusCode: http.StatusBadRequest, ErrorMessage: strPtr("asynchronous operation mode required")}
	}

	svc.mu.Lock()
	defer svc.mu.Unlock()

	iID := internal.InstanceID(req.InstanceID)
	bID := internal.BindingID(req.BindingID)
	instance, err := svc.instanceGetter.Get(iID)

	opID, err := svc.operationIDProvider()
	if err != nil {
		return nil, &osb.HTTPStatusCodeError{StatusCode: http.StatusBadRequest, ErrorMessage: strPtr(fmt.Sprintf("while generating ID for operation: %v", err))}
	}

	// TODO: handle operation async
	paramHash := jsonhash.HashS(req.Parameters)
	op := internal.BindOperation{
		InstanceID:  iID,
		BindingID: bID,
		OperationID: opID,
		Type:        internal.OperationTypeCreate,
		State:       internal.OperationStateInProgress,
		ParamsHash:  paramHash,
	}

	if err := svc.operationInserter.InsertBindOperation(&op); err != nil {
		return nil, &osb.HTTPStatusCodeError{StatusCode: http.StatusBadRequest, ErrorMessage: strPtr(fmt.Sprintf("while inserting instance operation to storage: %v", err))}
	}

	// here ends operation

	svcID := internal.ServiceID(req.ServiceID)
	addonID := internal.AddonID(svcID)
	addon, err := svc.addonIDGetter.GetByID(osbCtx.BrokerNamespace, addonID)

	svcPlanID := internal.ServicePlanID(req.PlanID)
	// addonPlanID is in 1:1 match with servicePlanID (from service catalog)
	addonPlanID := internal.AddonPlanID(svcPlanID)
	addonPlan, found := addon.Plans[addonPlanID]
	if !found {
		return nil, &osb.HTTPStatusCodeError{StatusCode: http.StatusBadRequest, ErrorMessage: strPtr(fmt.Sprintf("addon does not contain requested plan (planID: %s): %v", err, addonPlanID))}
	}


	bindInput := bindingInput{
		instance: instance,
		addonPlan: addonPlan,
		isAddonBindable: addon.Bindable,

	}
	// TODO: ASYNC HERE
	svc.doAsync(ctx, bindInput)

	opKey := osb.OperationKey(op.OperationID)

	resp := &osb.BindResponse{
		OperationKey: &opKey,
		Async:        true,
		Credentials: svc.dtoFromModel(resolvedBindData.Credentials),
	}

	//out, err := svc.instanceBindDataGetter.Get(internal.InstanceID(req.InstanceID)) // [SECRETS-ISSUE] here we get credentials in exchange for instanceID - perhaps from etcd
	//if err != nil {
	//	return nil, errors.Wrapf(err, "while getting bind data from storage for instance id: %q", req.InstanceID)
	//}

	//return &osb.BindResponse{
	//	Credentials: svc.dtoFromModel(out.Credentials),
	//}, nil

	return resp, nil
}

type bindingInput struct {
	instance *internal.Instance
	operationID internal.OperationID
	addonPlan internal.AddonPlan
	isAddonBindable bool
}

func (svc *bindService) doAsync(ctx context.Context, input bindingInput)  {
	if svc.testHookAsyncCalled != nil {
		svc.testHookAsyncCalled(input.operationID)
	}
	go svc.do(ctx, input)
}

func (svc *bindService) do(ctx context.Context, input bindingInput)  {


	fDo := func() error {
		if svc.isBindable(input.addonPlan, input.isAddonBindable) {
			c, err := svc.chartGetter.Get(input.instance.Namespace, input.addonPlan.ChartRef.Name, input.addonPlan.ChartRef.Version)
			if err != nil {
				return errors.Wrap(err, "while getting chart from storage")
			}

			resolveErr := svc.renderAndResolveBindData(input.addonPlan, input.instance, c)
			if resolveErr != nil {
				opState := internal.OperationStateFailed
				opDesc := fmt.Sprintf("resolving bind data failed with error: %s", err.Error())

				if err := svc.operationUpdater.UpdateStateDesc(input.instance.ID, input.operationID, opState, &opDesc); err != nil {
					svc.log.Errorf("State description was not updated, got error: %v", err)
				}

				return errors.Wrap(resolveErr, "while resolving bind data")
			}
		}
		return nil
	}

	opState := internal.OperationStateSucceeded
	opDesc := "binding succeeded"

	err := fDo()
	if err != nil {
		opState = internal.OperationStateFailed
		opDesc = fmt.Sprintf("binding failed on error: %s", err.Error())
	}

	if err := svc.operationUpdater.UpdateStateDesc(input.instance.ID, input.operationID, opState, &opDesc); err != nil {
		svc.log.Errorf("State description was not updated, got error: %v", err)
	}
}

func (svc *bindService) isBindable(plan internal.AddonPlan, isAddonBindable bool) bool {
	return (plan.Bindable != nil && *plan.Bindable) || // if bindable field is set on plan it's override bindable field on addon
		(plan.Bindable == nil && isAddonBindable) // if bindable field is NOT set on plan that bindable field on addon is important
}

func (svc *bindService) renderAndResolveBindData(addonPlan internal.AddonPlan, instance *internal.Instance, ch *chart.Chart) error {
	rendered, err := svc.bindTemplateRenderer.RenderOnBind(addonPlan.BindTemplate, instance, ch)
	if err != nil {
		return errors.Wrap(err, "while rendering bind yaml template")
	}

	out, err := svc.bindTemplateResolver.Resolve(rendered, instance.Namespace)
	if err != nil {
		return errors.Wrap(err, "while resolving bind yaml values")
	}

	resolvedBindData = internal.InstanceBindData{
		InstanceID:  instance.ID,
		Credentials: out.Credentials,
	}


	return nil
}

func (*bindService) dtoFromModel(in internal.InstanceCredentials) map[string]interface{} {
	dto := map[string]interface{}{}
	for k, v := range in {
		dto[k] = v
	}
	return dto
}
