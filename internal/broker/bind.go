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
	addonIDGetter        addonIDGetter
	chartGetter          chartGetter
	instanceGetter       instanceGetter
	bindTemplateRenderer bindTemplateRenderer
	bindTemplateResolver bindTemplateResolver
	resolvedBindData     map[internal.BindingID]*internal.InstanceBindData
	bindOperation        map[internal.InstanceID][]*internal.BindOperation
	operationIDProvider  func() (internal.OperationID, error)
	mu                   sync.Mutex

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

	paramHash := jsonhash.HashS(req.Parameters)

	switch state, err := svc.isBinded(iID, bID); true {
	case err != nil:
		return nil, &osb.HTTPStatusCodeError{StatusCode: http.StatusBadRequest, ErrorMessage: strPtr(fmt.Sprintf("while checking if service binding for service instance already exists: %v", err))}
	case state:
		if err := svc.compareBindingParameters(iID, bID, paramHash); err != nil { // TODO: verify if comparision is needed
			return nil, &osb.HTTPStatusCodeError{StatusCode: http.StatusConflict, ErrorMessage: strPtr(fmt.Sprintf("while comparing binding parameters %v: %v", req.Parameters, err))}
		}
		out, err := svc.getInstanceBindData(iID, bID)
		if err != nil {
			return nil, &osb.HTTPStatusCodeError{StatusCode: http.StatusConflict, ErrorMessage: strPtr(fmt.Sprintf("while getting bind data from memory for instance id: %q and service binding id: %q with error: %v", iID, bID, err))}
		}
		return &osb.BindResponse{
			Async:       false,
			Credentials: svc.dtoFromModel(out.Credentials),
		}, nil
	}

	switch opIDInProgress, inProgress, err := svc.isBindingInProgress(iID, bID); true {
	case err != nil:
		return nil, &osb.HTTPStatusCodeError{StatusCode: http.StatusBadRequest, ErrorMessage: strPtr(fmt.Sprintf("while checking if service binding is being created: %v", err))}
	case inProgress:
		if err := svc.compareBindingParameters(iID, bID, paramHash); err != nil {
			return nil, &osb.HTTPStatusCodeError{StatusCode: http.StatusConflict, ErrorMessage: strPtr(fmt.Sprintf("while comparing binding parameters %v: %v", req.Parameters, err))}
		}
		opKeyInProgress := osb.OperationKey(opIDInProgress)
		return &osb.BindResponse{Async: true, OperationKey: &opKeyInProgress}, nil
	}

	opID, err := svc.operationIDProvider()
	if err != nil {
		return nil, &osb.HTTPStatusCodeError{StatusCode: http.StatusBadRequest, ErrorMessage: strPtr(fmt.Sprintf("while generating ID for operation: %v", err))}
	}

	// TODO: handle operation async
	op := internal.BindOperation{
		InstanceID:  iID,
		BindingID:   bID,
		OperationID: opID,
		Type:        internal.OperationTypeCreate,
		State:       internal.OperationStateInProgress,
		ParamsHash:  paramHash,
	}

	if err := svc.insertBindOperation(&op); err != nil {
		return nil, &osb.HTTPStatusCodeError{StatusCode: http.StatusBadRequest, ErrorMessage: strPtr(fmt.Sprintf("while inserting instance operation to storage: %v", err))}
	}

	// here ends operation
	instance, err := svc.instanceGetter.Get(iID)
	if err != nil {
		return nil, &osb.HTTPStatusCodeError{StatusCode: http.StatusBadRequest, ErrorMessage: strPtr(fmt.Sprintf("while getting instance from storage for id: %q with error: %v", iID, err))}
	}
	svcID := internal.ServiceID(req.ServiceID)
	addonID := internal.AddonID(svcID)
	addon, err := svc.addonIDGetter.GetByID(osbCtx.BrokerNamespace, addonID)
	if err != nil {
		return nil, &osb.HTTPStatusCodeError{StatusCode: http.StatusBadRequest, ErrorMessage: strPtr(fmt.Sprintf("while getting addon from storage in namespace %q for id: %q with error: %v", osbCtx.BrokerNamespace, addonID, err))}
	}

	svcPlanID := internal.ServicePlanID(req.PlanID)
	// addonPlanID is in 1:1 match with servicePlanID (from service catalog)
	addonPlanID := internal.AddonPlanID(svcPlanID)
	addonPlan, found := addon.Plans[addonPlanID]
	if !found {
		return nil, &osb.HTTPStatusCodeError{StatusCode: http.StatusBadRequest, ErrorMessage: strPtr(fmt.Sprintf("addon does not contain requested plan (planID: %s): %v", err, addonPlanID))}
	}

	bindInput := bindingInput{
		brokerNamespace: osbCtx.BrokerNamespace,
		instance:        instance,
		bindingID:       bID,
		operationID:     op.OperationID,
		addonPlan:       addonPlan,
		isAddonBindable: addon.Bindable,
	}
	// TODO: ASYNC HERE
	svc.doAsync(ctx, bindInput)

	opKey := osb.OperationKey(op.OperationID)

	resp := &osb.BindResponse{
		OperationKey: &opKey,
		Async:        true,
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

func (svc *bindService) insertBindOperation(bo *internal.BindOperation) error {
	if bo == nil {
		return errors.New("entity may not be nil")
	}

	if bo.InstanceID.IsZero() || bo.BindingID.IsZero() || bo.OperationID.IsZero() {
		return errors.New("all parameters: instance, binding and operation id must be set")
	}

	svc.bindOperation[bo.InstanceID] = append(svc.bindOperation[bo.InstanceID], bo)

	return nil
}

func (svc *bindService) updateBindOperationStateDesc(iID internal.InstanceID, bID internal.BindingID, opID internal.OperationID, state internal.OperationState, desc *string) error {
	op, err := svc.getBindOperation(iID, bID, opID)
	if err != nil {
		return errors.Wrap(err, "while getting bind operation")
	}

	op.State = state
	op.StateDescription = desc

	return nil
}

func (svc *bindService) getBindOperation(iID internal.InstanceID, bID internal.BindingID, opID internal.OperationID) (*internal.BindOperation, error) {
	if iID.IsZero() || bID.IsZero() || opID.IsZero() {
		return nil, errors.Errorf("all parameters: instance, binding and operation id must be set. InstanceID: %q | BindingID: %q | OperationID: %q", iID, bID, opID)
	}

	out := &internal.BindOperation{}

	for _, singleOperation := range svc.bindOperation[iID] {
		if singleOperation.BindingID == bID && singleOperation.OperationID == opID {
			out = singleOperation
			return out, nil
		}

	}
	return nil, notFoundError{}

}

func (svc *bindService) getAllBindOperation(iID internal.InstanceID) ([]*internal.BindOperation, error) {
	if iID.IsZero() {
		return nil, errors.New("instance id must be set")
	}

	out := svc.bindOperation[iID]

	if len(out) == 0 {
		return nil, notFoundError{}
	}

	return out, nil
}

func (svc *bindService) isBinded(iID internal.InstanceID, bID internal.BindingID) (bool, error) {
	result := false

	ops, err := svc.getAllBindOperation(iID)
	switch {
	case err == nil:
	case IsNotFoundError(err):
		return false, nil
	default:
		return false, errors.Wrap(err, "while getting operations from memory")
	}

	for _, op := range ops {
		if op.Type == internal.OperationTypeCreate && op.State == internal.OperationStateSucceeded && op.BindingID == bID {
			result = true
		}
		if op.Type == internal.OperationTypeRemove && op.State == internal.OperationStateSucceeded && op.BindingID == bID {
			result = false
			break
		}
	}

	return result, nil
}

func (svc *bindService) isBindingInProgress(iID internal.InstanceID, bID internal.BindingID) (internal.OperationID, bool, error) {
	result := false
	var resultOpID internal.OperationID

	ops, err := svc.getAllBindOperation(iID)
	switch {
	case err == nil:
	case IsNotFoundError(err):
		return resultOpID, false, nil
	default:
		return resultOpID, false, errors.Wrap(err, "while getting operations from memory")
	}

	for _, op := range ops {
		if op.Type == internal.OperationTypeCreate && op.State == internal.OperationStateInProgress && op.BindingID == bID {
			result = true
			resultOpID = op.OperationID
			break
		}
	}

	return resultOpID, result, nil
}

func (svc *bindService) GetLastBindOperation(ctx context.Context, osbCtx OsbContext, req *osb.BindingLastOperationRequest) (*osb.LastOperationResponse, error) {
	iID := internal.InstanceID(req.InstanceID)
	bID := internal.BindingID(req.BindingID)

	var opID internal.OperationID
	if req.OperationKey != nil {
		opID = internal.OperationID(*req.OperationKey)
	}

	op, err := svc.getBindOperation(iID, bID, opID)
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

func (svc *bindService) GetServiceBinding(ctx context.Context, osbCtx OsbContext, req *osb.GetBindingRequest) (*osb.GetBindingResponse, error) {
	iID := internal.InstanceID(req.InstanceID)
	bID := internal.BindingID(req.BindingID)

	switch opIDInProgress, inProgress, err := svc.isBindingInProgress(iID, bID); true {
	case err != nil:
		return nil, &osb.HTTPStatusCodeError{StatusCode: http.StatusBadRequest, ErrorMessage: strPtr(fmt.Sprintf("while checking if service binding is being created: %v", err))}
	case inProgress:
		return nil, &osb.HTTPStatusCodeError{StatusCode: http.StatusNotFound, ErrorMessage: strPtr(fmt.Sprintf("service binding id: %q is in progress", opIDInProgress))}
	}

	out, err := svc.getInstanceBindData(iID, bID)
	if err != nil {
		return nil, &osb.HTTPStatusCodeError{StatusCode: http.StatusNotFound, ErrorMessage: strPtr(fmt.Sprintf("while getting bind data from memory for instance id: %q and service binding id: %q with error: %v", iID, bID, err))}
	}
	return &osb.GetBindingResponse{
		Credentials: svc.dtoFromModel(out.Credentials),
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

	if err := svc.updateBindOperationStateDesc(input.instance.ID, input.bindingID, input.operationID, opState, &opDesc); err != nil {
		svc.log.Errorf("State description was not updated, got error: %v", err)
	}
}

func (svc *bindService) isBindable(plan internal.AddonPlan, isAddonBindable bool) bool {
	return (plan.Bindable != nil && *plan.Bindable) || // if bindable field is set on plan it's override bindable field on addon
		(plan.Bindable == nil && isAddonBindable) // if bindable field is NOT set on plan that bindable field on addon is important
}

func (svc *bindService) renderAndResolveBindData(addonPlan internal.AddonPlan, instance *internal.Instance, bID internal.BindingID, ch *chart.Chart) error {
	rendered, err := svc.bindTemplateRenderer.RenderOnBind(addonPlan.BindTemplate, instance, ch)
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
	svc.resolvedBindData[bID] = &in
	return nil
}

func (svc *bindService) getInstanceBindData(iID internal.InstanceID, bID internal.BindingID) (*internal.InstanceBindData, error) {
	if iID.IsZero() || bID.IsZero() {
		return nil, errors.New("both instance and binding id must be set")
	}

	if svc.resolvedBindData[bID] == nil {
		return nil, notFoundError{}
	}

	resultBindData := svc.resolvedBindData[bID]

	return resultBindData, nil
}

func (svc *bindService) compareBindingParameters(iID internal.InstanceID, bID internal.BindingID, newHash string) error {
	ops, err := svc.getAllBindOperation(iID)
	switch {
	case err == nil:
	case IsNotFoundError(err):
		return nil
	default:
		return errors.Wrapf(err, "while getting bind operation for instance id: %q from memory", iID)
	}

	outOp := &internal.BindOperation{}
	for _, singleOperation := range ops {
		if singleOperation.InstanceID == iID && singleOperation.BindingID == bID {
			outOp = singleOperation
			break
		}
	}

	if outOp.ParamsHash != newHash {
		return errors.Errorf("binding parameters hash differs - new %s, old %s, for instance %s", newHash, outOp.ParamsHash, iID)
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

type notFoundError struct{}

func (notFoundError) Error() string  { return "element not found" }
func (notFoundError) NotFound() bool { return true }
