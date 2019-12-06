package broker

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"sync"

	jsonhash "github.com/komkom/go-jsonhash"
	"github.com/kyma-project/helm-broker/internal"
	"github.com/pkg/errors"
	osb "github.com/pmorie/go-open-service-broker-client/v2"
	"github.com/sirupsen/logrus"
	rls "k8s.io/helm/pkg/proto/hapi/services"
)

const (
	goTplEngine             = "gotpl"
	addonsRepositoryURLName = "addonsRepositoryURL"
)

type provisionService struct {
	addonIDGetter       addonIDGetter
	chartGetter         chartGetter
	instanceInserter    instanceInserter
	instanceGetter      instanceGetter
	instanceStateGetter instanceStateProvisionGetter
	operationInserter   operationInserter
	operationUpdater    operationUpdater
	operationIDProvider func() (internal.OperationID, error)
	helmInstaller       helmInstaller
	mu                  sync.Mutex

	log *logrus.Entry

	testHookAsyncCalled func(internal.OperationID)
}

func (svc *provisionService) Provision(ctx context.Context, osbCtx OsbContext, req *osb.ProvisionRequest) (*osb.ProvisionResponse, *osb.HTTPStatusCodeError) {
	if !req.AcceptsIncomplete {
		return nil, &osb.HTTPStatusCodeError{StatusCode: http.StatusBadRequest, ErrorMessage: strPtr("asynchronous operation mode required")}
	}

	// Single provisioning is supported concurrently.
	// TODO: switch to lock per instanceID
	svc.mu.Lock()
	defer svc.mu.Unlock()

	iID := internal.InstanceID(req.InstanceID)
	paramHash := jsonhash.HashS(req.Parameters)

	switch state, err := svc.instanceStateGetter.IsProvisioned(iID); true {
	case err != nil:
		return nil, &osb.HTTPStatusCodeError{StatusCode: http.StatusInternalServerError, ErrorMessage: strPtr(fmt.Sprintf("while checking if instance is already provisioned: %v", err))}
	case state:
		if err := svc.compareProvisioningParameters(iID, paramHash); err != nil {
			return nil, &osb.HTTPStatusCodeError{StatusCode: http.StatusConflict, ErrorMessage: strPtr(fmt.Sprintf("while comparing provisioning parameters %v: %v", req.Parameters, err))}
		}
		return &osb.ProvisionResponse{Async: false}, nil
	}

	switch opIDInProgress, inProgress, err := svc.instanceStateGetter.IsProvisioningInProgress(iID); true {
	case err != nil:
		return nil, &osb.HTTPStatusCodeError{StatusCode: http.StatusInternalServerError, ErrorMessage: strPtr(fmt.Sprintf("while checking if instance is being provisioned: %v", err))}
	case inProgress:
		if err := svc.compareProvisioningParameters(iID, paramHash); err != nil {
			return nil, &osb.HTTPStatusCodeError{StatusCode: http.StatusConflict, ErrorMessage: strPtr(fmt.Sprintf("while comparing provisioning parameters %v: %v", req.Parameters, err))}
		}
		opKeyInProgress := osb.OperationKey(opIDInProgress)
		return &osb.ProvisionResponse{Async: true, OperationKey: &opKeyInProgress}, nil
	}

	namespace, err := getNamespaceFromContext(req.Context)
	if err != nil {
		return nil, &osb.HTTPStatusCodeError{StatusCode: http.StatusBadRequest, ErrorMessage: strPtr(fmt.Sprintf("while getting namespace from context: %v", err))}
	}

	// addonID is in 1:1 match with serviceID (from service catalog)
	svcID := internal.ServiceID(req.ServiceID)
	addonID := internal.AddonID(svcID)
	addon, err := svc.addonIDGetter.GetByID(osbCtx.BrokerNamespace, addonID)
	switch {
	case IsNotFoundError(err):
		return nil, &osb.HTTPStatusCodeError{StatusCode: http.StatusBadRequest, ErrorMessage: strPtr(fmt.Sprintf("while getting addon: %v", err))}
	case err != nil:
		return nil, &osb.HTTPStatusCodeError{StatusCode: http.StatusInternalServerError, ErrorMessage: strPtr(fmt.Sprintf("while getting addon: %v", err))}
	}

	instances, err := svc.instanceGetter.GetAll()
	switch {
	case IsNotFoundError(err):
		return nil, &osb.HTTPStatusCodeError{StatusCode: http.StatusBadRequest, ErrorMessage: strPtr(fmt.Sprintf("while getting instance collection: %v", err))}
	case err != nil:
		return nil, &osb.HTTPStatusCodeError{StatusCode: http.StatusInternalServerError, ErrorMessage: strPtr(fmt.Sprintf("while getting instance collection: %v", err))}
	}
	if !addon.IsProvisioningAllowed(namespace, instances) {
		svc.log.Infof("addon with name: %q (id: %s) and flag 'provisionOnlyOnce' in namespace %q will be not provisioned because his instance already exist", addon.Name, addon.ID, namespace)
		return nil, &osb.HTTPStatusCodeError{StatusCode: http.StatusBadRequest, ErrorMessage: strPtr(fmt.Sprintf("addon with name: %q (id: %s) and flag 'provisionOnlyOnce' in namespace %q will be not provisioned because his instance already exist", addon.Name, addon.ID, namespace))}
	}

	opID, err := svc.operationIDProvider()
	if err != nil {
		return nil, &osb.HTTPStatusCodeError{StatusCode: http.StatusInternalServerError, ErrorMessage: strPtr(fmt.Sprintf("while generating operation ID: %v", err))}
	}

	op := internal.InstanceOperation{
		InstanceID:  iID,
		OperationID: opID,
		Type:        internal.OperationTypeCreate,
		State:       internal.OperationStateInProgress,
		ParamsHash:  paramHash,
	}

	if err := svc.operationInserter.Insert(&op); err != nil {
		return nil, &osb.HTTPStatusCodeError{StatusCode: http.StatusBadRequest, ErrorMessage: strPtr(fmt.Sprintf("while inserting instance operation to storage: %v", err))}
	}

	svcPlanID := internal.ServicePlanID(req.PlanID)

	// addonPlanID is in 1:1 match with servicePlanID (from service catalog)
	addonPlanID := internal.AddonPlanID(svcPlanID)
	addonPlan, found := addon.Plans[addonPlanID]
	if !found {
		return nil, &osb.HTTPStatusCodeError{StatusCode: http.StatusBadRequest, ErrorMessage: strPtr(fmt.Sprintf("addon does not contain requested plan (planID: %s): %v", err, addonPlanID))}
	}
	releaseName := createReleaseName(addon.Name, addonPlan.Name, iID)

	i := internal.Instance{
		ID:            iID,
		Namespace:     namespace,
		ServiceID:     svcID,
		ServicePlanID: svcPlanID,
		ReleaseName:   releaseName,
		ParamsHash:    paramHash,
		ReleaseInfo:   internal.ReleaseInfo{},
	}

	exist, err := svc.instanceInserter.Upsert(&i)
	if err != nil {
		return nil, &osb.HTTPStatusCodeError{StatusCode: http.StatusBadRequest, ErrorMessage: strPtr(fmt.Sprintf("while inserting instance to storage: %v", err))}
	}
	if exist {
		svc.log.Infof("Instance %s already existed in storage, instance was replaced", i.ID)
	}

	chartOverrides := internal.ChartValues(req.Parameters)

	provisionInput := provisioningInput{
		instanceID:          iID,
		operationID:         opID,
		namespace:           namespace,
		brokerNamespace:     osbCtx.BrokerNamespace,
		releaseName:         releaseName,
		addonPlan:           addonPlan,
		isAddonBindable:     addon.Bindable,
		addonsRepositoryURL: addon.RepositoryURL,
		chartOverrides:      chartOverrides,
		instanceToUpdate:    &i,
	}

	svc.doAsync(ctx, provisionInput)

	opKey := osb.OperationKey(op.OperationID)
	resp := &osb.ProvisionResponse{
		OperationKey: &opKey,
		Async:        true,
	}

	return resp, nil
}

// provisioningInput holds all information required to provision a given instance
type provisioningInput struct {
	instanceID          internal.InstanceID
	operationID         internal.OperationID
	namespace           internal.Namespace
	brokerNamespace     internal.Namespace
	releaseName         internal.ReleaseName
	addonPlan           internal.AddonPlan
	isAddonBindable     bool
	chartOverrides      internal.ChartValues
	addonsRepositoryURL string
	instanceToUpdate    *internal.Instance
}

func (svc *provisionService) doAsync(ctx context.Context, input provisioningInput) {
	if svc.testHookAsyncCalled != nil {
		svc.testHookAsyncCalled(input.operationID)
	}
	go svc.do(ctx, input)
}

// do is called asynchronously
func (svc *provisionService) do(ctx context.Context, input provisioningInput) {

	fDo := func() error {

		c, err := svc.chartGetter.Get(input.brokerNamespace, input.addonPlan.ChartRef.Name, input.addonPlan.ChartRef.Version)
		if err != nil {
			return errors.Wrap(err, "while getting chart from storage")
		}

		out, err := deepCopy(input.addonPlan.ChartValues)
		if err != nil {
			return errors.Wrap(err, "while coping plan values")
		}

		out = mergeValues(out, input.chartOverrides)

		out[addonsRepositoryURLName] = input.addonsRepositoryURL

		svc.log.Infof("Merging values for operation [%s], releaseName [%s], namespace [%s], addonPlan [%s]. Plan values are: [%v], overrides: [%v], merged: [%v] ",
			input.operationID, input.releaseName, input.namespace, input.addonPlan.Name, input.addonPlan.ChartValues, input.chartOverrides, out)

		resp, err := svc.helmInstaller.Install(c, internal.ChartValues(out), input.releaseName, input.namespace)
		if err != nil {
			return errors.Wrap(err, "while installing helm release")
		}

		relInfo := internal.ReleaseInfo{
			Time:     resp.GetRelease().GetInfo().GetLastDeployed(),
			Revision: int(resp.GetRelease().GetVersion()),
			Config:   resp.GetRelease().GetConfig(),
		}

		updatedInstance := input.instanceToUpdate
		updatedInstance.ReleaseInfo = relInfo

		exist, err := svc.instanceInserter.Upsert(updatedInstance)
		if err != nil {
			return &osb.HTTPStatusCodeError{StatusCode: http.StatusConflict, ErrorMessage: strPtr(fmt.Sprintf("while updating instance in storage: %v", err))}
		}
		if exist {
			svc.log.Infof("Instance %s already existed in storage, instance was replaced on update", updatedInstance.ID)
		}
		isRespValid := validateInstallReleaseResponse(resp)
		if isRespValid != nil {
			return errors.Wrap(err, "while validating input")
		}

		return nil
	}

	opState := internal.OperationStateSucceeded
	opDesc := "provisioning succeeded"

	err := fDo()

	if err != nil {
		opState = internal.OperationStateFailed
		opDesc = fmt.Sprintf("provisioning failed on error: %s", err.Error())
	}

	if err := svc.operationUpdater.UpdateStateDesc(input.instanceID, input.operationID, opState, &opDesc); err != nil {
		svc.log.Errorf("State description was not updated, got error: %v", err)
	}
}

func (svc *provisionService) compareProvisioningParameters(iID internal.InstanceID, newHash string) error {
	instance, err := svc.instanceGetter.Get(iID)
	switch {
	case err == nil:
	case IsNotFoundError(err):
		return nil
	default:
		return errors.Wrapf(err, "while getting instance %s from storage", iID)
	}

	if instance.ParamsHash != newHash {
		return errors.Errorf("provisioning parameters hash differs - new %s, old %s, for instance %s", newHash, instance.ParamsHash, iID)
	}

	return nil
}

func getNamespaceFromContext(contextProfile map[string]interface{}) (internal.Namespace, error) {
	ns, ok := contextProfile["namespace"]
	if !ok {
		return internal.Namespace(""), errors.New("namespace does not exists in given context")
	}
	return internal.Namespace(ns.(string)), nil
}

func createReleaseName(name internal.AddonName, planName internal.AddonPlanName, iID internal.InstanceID) internal.ReleaseName {
	maxLen := 53
	relName := fmt.Sprintf("hb-%s-%s-%s", name, planName, iID)
	if len(relName) <= maxLen {
		return internal.ReleaseName(relName)
	}
	return internal.ReleaseName(relName[:maxLen])
}

// to work correctly, https://github.com/ghodss/yaml has to be used
func mergeValues(dest map[string]interface{}, src map[string]interface{}) map[string]interface{} {
	for k, v := range src {
		// If the key doesn't exist already, then just set the key to that value
		if _, exists := dest[k]; !exists {
			dest[k] = v
			continue
		}

		nextMap, ok := v.(map[string]interface{})
		// If it isn't another map, overwrite the value
		if !ok {
			dest[k] = v
			continue
		}
		// If the key doesn't exist already, then just set the key to that value
		if _, exists := dest[k]; !exists {
			dest[k] = nextMap
			continue
		}
		// Edge case: If the key exists in the destination, but isn't a map
		destMap, isMap := dest[k].(map[string]interface{})
		// If the source map has a map for this key, prefer it
		if !isMap {
			dest[k] = v
			continue
		}
		// If we got to this point, it is a map in both, so merge them
		dest[k] = mergeValues(destMap, nextMap)
	}
	return dest
}

func validateInstallReleaseResponse(resp *rls.InstallReleaseResponse) error {
	if resp == nil {
		return fmt.Errorf("input parameter 'InstallReleaseResponse' cannot be nil")
	}

	if resp.Release == nil {
		return fmt.Errorf("'Release' filed from 'InstallReleaseResponse' is missing")
	}

	if resp.Release.Info == nil {
		return fmt.Errorf("'Info' filed from 'InstallReleaseResponse' is missing")
	}

	ch := resp.Release.Chart
	if ch.Metadata.Engine != "" && ch.Metadata.Engine != goTplEngine {
		return fmt.Errorf("chart %q requested non-existent template engine %q", ch.Metadata.Name, ch.Metadata.Engine)
	}

	return nil
}

func deepCopy(in map[string]interface{}) (map[string]interface{}, error) {
	out := map[string]interface{}{}
	if in != nil {
		b, err := json.Marshal(in)
		if err != nil {
			return nil, errors.Wrap(err, "while performing deep copy (marshal)")
		}

		if err = json.Unmarshal(b, &out); err != nil {
			return nil, errors.Wrap(err, "while performing deep copy (unmarshal)")
		}
	}
	return out, nil
}

func strPtr(str string) *string {
	return &str
}
