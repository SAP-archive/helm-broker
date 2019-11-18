package broker

import (
	"context"
	"fmt"
	"net/http"
	"sync"

	jsonhash "github.com/komkom/go-jsonhash"
	"github.com/pkg/errors"
	osb "github.com/pmorie/go-open-service-broker-client/v2"
	"github.com/sirupsen/logrus"
	"k8s.io/helm/pkg/proto/hapi/chart"

	"github.com/kyma-project/helm-broker/internal"
)

type bindService struct {
	addonIDGetter                 addonIDGetter
	chartGetter                   chartGetter
	instanceGetter                instanceGetter
	bindTemplateRenderer          bindTemplateRenderer
	bindTemplateResolver          bindTemplateResolver
	instanceBindDataInserter      instanceBindDataInserter
	instanceBindDataGetter        instanceBindDataGetter
	instanceBindDataRemover       instanceBindDataRemover
	bindStateGetter               bindStateBindingGetter
	bindOperationGetter           bindOperationGetter
	bindOperationCollectionGetter bindOperationCollectionGetter
	bindOperationInserter         bindOperationInserter
	bindOperationUpdater          bindOperationUpdater
	operationIDProvider           func() (internal.OperationID, error)
	mu                            sync.Mutex

	log *logrus.Entry

	testHookAsyncCalled func(internal.OperationID)
}

func (svc *bindService) Bind(ctx context.Context, osbCtx OsbContext, req *osb.BindRequest) (*osb.BindResponse, *osb.HTTPStatusCodeError) {
	if len(req.Parameters) > 0 {
		return nil, &osb.HTTPStatusCodeError{StatusCode: http.StatusBadRequest, ErrorMessage: strPtr("helm-broker does not support configuration options for the service binding")}
	}

	if !req.AcceptsIncomplete {
		return nil, &osb.HTTPStatusCodeError{StatusCode: http.StatusBadRequest, ErrorMessage: strPtr("asynchronous operation mode required")}
	}

	svc.mu.Lock()
	defer svc.mu.Unlock()

	iID := internal.InstanceID(req.InstanceID)
	bID := internal.BindingID(req.BindingID)
	svcID := internal.ServiceID(req.ServiceID)
	svcPlanID := internal.ServicePlanID(req.PlanID)
	paramHash := jsonhash.HashS(req.Parameters)

	switch opIDInProgress, inProgress, err := svc.bindStateGetter.IsBindingInProgress(iID, bID); true {
	case err != nil:
		return nil, &osb.HTTPStatusCodeError{StatusCode: http.StatusBadRequest, ErrorMessage: strPtr(fmt.Sprintf("while checking if service binding is being created: %v", err))}
	case inProgress:
		opKeyInProgress := osb.OperationKey(opIDInProgress)
		return &osb.BindResponse{Async: true, OperationKey: &opKeyInProgress}, nil
	}

	switch bindOp, state, err := svc.bindStateGetter.IsBound(iID, bID); true {
	case err != nil:
		return nil, &osb.HTTPStatusCodeError{StatusCode: http.StatusBadRequest, ErrorMessage: strPtr(fmt.Sprintf("while checking if service binding for service instance already exists: %v", err))}
	case state:
		opID := bindOp.OperationID
		bindInput, err := svc.prepareBindInput(osbCtx, iID, bID, svcID, svcPlanID, opID, paramHash)
		if err != nil {
			return nil, err
		}

		svc.doAsync(ctx, bindInput)
		out, getIbdErr := svc.getInstanceBindData(iID, bID)
		if getIbdErr != nil {
			return nil, &osb.HTTPStatusCodeError{StatusCode: http.StatusBadRequest, ErrorMessage: strPtr(fmt.Sprintf("while getting bind data from memory for instance id: %q and service binding id: %q with error: %v", iID, bID, err))}
		}

		credsOut := svc.dtoFromModel(out.Credentials)

		if removerErr := svc.instanceBindDataRemover.Remove(iID); removerErr != nil {
			svc.log.Errorf("while removing instance bind data after getting it from memory on bind, got error: %v", removerErr)
		}

		return &osb.BindResponse{
			Async:       false,
			Credentials: credsOut,
		}, nil
	}

	op, err := svc.prepareBindOperation(iID, bID, paramHash)
	if err != nil {
		return nil, err
	}

	if err := svc.bindOperationInserter.Insert(&op); err != nil {
		return nil, &osb.HTTPStatusCodeError{StatusCode: http.StatusBadRequest, ErrorMessage: strPtr(fmt.Sprintf("while inserting instance operation to storage: %v", err))}
	}

	opID := op.OperationID

	bindInput, err := svc.prepareBindInput(osbCtx, iID, bID, svcID, svcPlanID, opID, paramHash)
	if err != nil {
		return nil, err
	}

	svc.doAsync(ctx, bindInput)

	opKey := osb.OperationKey(opID)

	resp := &osb.BindResponse{
		OperationKey: &opKey,
		Async:        true,
	}

	return resp, nil
}

func (svc *bindService) GetLastBindOperation(ctx context.Context, osbCtx OsbContext, req *osb.BindingLastOperationRequest) (*osb.LastOperationResponse, error) {
	iID := internal.InstanceID(req.InstanceID)
	bID := internal.BindingID(req.BindingID)

	var opID internal.OperationID
	if req.OperationKey != nil {
		opID = internal.OperationID(*req.OperationKey)
	}

	op, err := svc.bindOperationGetter.Get(iID, bID, opID)
	switch {
	case IsNotFoundError(err):
		return nil, err
	case err != nil:
		return nil, errors.Wrap(err, "while getting bind operation")
	}

	var descPtr *string
	if op.StateDescription != nil {
		desc := *op.StateDescription
		descPtr = &desc
	}

	resp := osb.LastOperationResponse{
		State:       osb.LastOperationState(op.State),
		Description: descPtr,
	}

	return &resp, nil
}

func (svc *bindService) GetServiceBinding(ctx context.Context, osbCtx OsbContext, req *osb.GetBindingRequest) (*osb.GetBindingResponse, *osb.HTTPStatusCodeError) {
	iID := internal.InstanceID(req.InstanceID)
	bID := internal.BindingID(req.BindingID)

	switch opIDInProgress, inProgress, err := svc.bindStateGetter.IsBindingInProgress(iID, bID); true {
	case err != nil:
		return nil, &osb.HTTPStatusCodeError{StatusCode: http.StatusBadRequest, ErrorMessage: strPtr(fmt.Sprintf("while checking if service binding is being created: %v", err))}
	case inProgress:
		return nil, &osb.HTTPStatusCodeError{StatusCode: http.StatusNotFound, ErrorMessage: strPtr(fmt.Sprintf("service binding id: %q is in progress", opIDInProgress))}
	}

	out, err := svc.instanceBindDataGetter.Get(iID)
	if err != nil {
		return nil, &osb.HTTPStatusCodeError{StatusCode: http.StatusNotFound, ErrorMessage: strPtr(fmt.Sprintf("while getting bind data from memory for instance id: %q and service binding id: %q with error: %v", iID, bID, err))}
	}

	credsOut := svc.dtoFromModel(out.Credentials)

	if removerErr := svc.instanceBindDataRemover.Remove(iID); removerErr != nil {
		svc.log.Errorf("while removing instance bind data after getting it from memory on get service binding, got error: %v", removerErr)
	}

	return &osb.GetBindingResponse{
		Credentials: credsOut,
	}, nil
}

type bindingInput struct {
	brokerNamespace internal.Namespace
	instance        *internal.Instance
	bindingID       internal.BindingID
	operationID     internal.OperationID
	addonPlan       internal.AddonPlan
	isAddonBindable bool
}

func (svc *bindService) prepareBindInput(osbCtx OsbContext, iID internal.InstanceID, bID internal.BindingID, svcID internal.ServiceID, svcPlanID internal.ServicePlanID, opID internal.OperationID, paramHash string) (bindingInput, *osb.HTTPStatusCodeError) {
	instance, err := svc.instanceGetter.Get(iID)
	if err != nil {
		return bindingInput{}, &osb.HTTPStatusCodeError{StatusCode: http.StatusBadRequest, ErrorMessage: strPtr(fmt.Sprintf("while getting instance from storage for id: %q with error: %v", iID, err))}
	}

	addonID := internal.AddonID(svcID)
	addon, err := svc.addonIDGetter.GetByID(osbCtx.BrokerNamespace, addonID)
	if err != nil {
		return bindingInput{}, &osb.HTTPStatusCodeError{StatusCode: http.StatusBadRequest, ErrorMessage: strPtr(fmt.Sprintf("while getting addon from storage in namespace %q for id: %q with error: %v", osbCtx.BrokerNamespace, addonID, err))}
	}

	// addonPlanID is in 1:1 match with servicePlanID (from service catalog)
	addonPlanID := internal.AddonPlanID(svcPlanID)
	addonPlan, found := addon.Plans[addonPlanID]
	if !found {
		return bindingInput{}, &osb.HTTPStatusCodeError{StatusCode: http.StatusBadRequest, ErrorMessage: strPtr(fmt.Sprintf("addon does not contain requested plan (planID: %s): %v", err, addonPlanID))}
	}

	bindInput := bindingInput{
		brokerNamespace: osbCtx.BrokerNamespace,
		instance:        instance,
		bindingID:       bID,
		operationID:     opID,
		addonPlan:       addonPlan,
		isAddonBindable: addon.Bindable,
	}

	return bindInput, nil
}

func (svc *bindService) prepareBindOperation(iID internal.InstanceID, bID internal.BindingID, paramHash string) (internal.BindOperation, *osb.HTTPStatusCodeError) {
	opID, err := svc.operationIDProvider()
	if err != nil {
		return internal.BindOperation{}, &osb.HTTPStatusCodeError{StatusCode: http.StatusBadRequest, ErrorMessage: strPtr(fmt.Sprintf("while generating ID for operation: %v", err))}
	}

	op := internal.BindOperation{
		InstanceID:  iID,
		BindingID:   bID,
		OperationID: opID,
		Type:        internal.OperationTypeCreate,
		State:       internal.OperationStateInProgress,
		ParamsHash:  paramHash,
	}

	return op, nil
}

func (svc *bindService) doAsync(ctx context.Context, input bindingInput) {
	if svc.testHookAsyncCalled != nil {
		svc.testHookAsyncCalled(input.operationID)
	}
	go svc.do(ctx, input)
}

func (svc *bindService) do(ctx context.Context, input bindingInput) {

	fDo := func() error {
		if svc.isBindable(input.addonPlan, input.isAddonBindable) {
			c, err := svc.chartGetter.Get(input.brokerNamespace, input.addonPlan.ChartRef.Name, input.addonPlan.ChartRef.Version)
			if err != nil {
				return errors.Wrap(err, "while getting chart from storage")
			}

			resolveErr := svc.renderAndResolveBindData(input.addonPlan, input.instance, input.bindingID, c)
			if resolveErr != nil {
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

	if err := svc.bindOperationUpdater.UpdateStateDesc(input.instance.ID, input.bindingID, input.operationID, opState, &opDesc); err != nil {
		svc.log.Errorf("State description was not updated, got error: %v", err)
	}

}

func (svc *bindService) isBindable(plan internal.AddonPlan, isAddonBindable bool) bool {
	return (plan.Bindable != nil && *plan.Bindable) || // if bindable field is set on plan it's override bindable field on addon
		(plan.Bindable == nil && isAddonBindable) // if bindable field is NOT set on plan that bindable field on addon is important
}

func (svc *bindService) renderAndResolveBindData(addonPlan internal.AddonPlan, instance *internal.Instance, bID internal.BindingID, ch *chart.Chart) error {
	rendered, err := svc.bindTemplateRenderer.Render(addonPlan.BindTemplate, instance, ch)
	if err != nil {
		return errors.Wrap(err, "while rendering bind yaml template")
	}

	out, err := svc.bindTemplateResolver.Resolve(rendered, instance.Namespace)
	if err != nil {
		return errors.Wrap(err, "while resolving bind yaml values")
	}

	in := internal.InstanceBindData{
		InstanceID:  instance.ID,
		Credentials: out.Credentials,
	}

	err = svc.instanceBindDataInserter.Insert(&in)
	if err != nil {
		return errors.Wrap(err, "while inserting instance bind data to memory")
	}

	return nil
}

func (svc *bindService) getInstanceBindData(iID internal.InstanceID, bID internal.BindingID) (*internal.InstanceBindData, error) {
	if iID.IsZero() || bID.IsZero() {
		return nil, errors.New("both instance and binding id must be set")
	}

	ibd, err := svc.instanceBindDataGetter.Get(iID)
	if err != nil {
		return nil, errors.Wrap(err, "while getting instance bind data from memory")
	}

	return ibd, nil
}

func (*bindService) dtoFromModel(in internal.InstanceCredentials) map[string]interface{} {
	dto := map[string]interface{}{}
	for k, v := range in {
		dto[k] = v
	}
	return dto
}

type notFoundError struct{}

func (notFoundError) Error() string  { return "element not found" }
func (notFoundError) NotFound() bool { return true }
