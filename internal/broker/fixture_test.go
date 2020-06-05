package broker_test

import (
	"fmt"
	"strings"

	"github.com/Masterminds/semver"
	"helm.sh/helm/v3/pkg/chart"

	"github.com/kyma-project/helm-broker/internal"
)

type expAll struct {
	BindingID   internal.BindingID
	InstanceID  internal.InstanceID
	OperationID internal.OperationID
	Addon       struct {
		ID            internal.AddonID
		Version       semver.Version
		Name          internal.AddonName
		Bindable      bool
		RepositoryURL string
	}
	AddonPlan struct {
		ID           internal.AddonPlanID
		Name         internal.AddonPlanName
		DisplayName  string
		BindTemplate internal.AddonPlanBindTemplate
	}
	Chart struct {
		Name    internal.ChartName
		Version semver.Version
	}
	Service struct {
		ID internal.ServiceID
	}
	ServicePlan struct {
		ID internal.ServicePlanID
	}
	Namespace                     internal.Namespace
	ReleaseName                   internal.ReleaseName
	ReleaseInfo                   internal.ReleaseInfo
	ProvisioningParameters        *internal.RequestParameters
	ParamsHash                    string
	RequestProvisioningParameters map[string]interface{}
}

func (exp *expAll) Populate() {
	exp.BindingID = internal.BindingID("fix-unique-bind-ID")
	exp.InstanceID = internal.InstanceID("fix-I-ID")
	exp.OperationID = internal.OperationID("fix-OP-ID")

	exp.Addon.ID = internal.AddonID("fix-B-ID")
	exp.Addon.Version = *semver.MustParse("0.1.2")
	exp.Addon.Name = internal.AddonName("fix-B-Name")
	exp.Addon.Bindable = true
	exp.Addon.RepositoryURL = "fix-url"

	exp.AddonPlan.ID = internal.AddonPlanID("fix-P-ID")
	exp.AddonPlan.Name = internal.AddonPlanName("fix-P-Name")
	exp.AddonPlan.DisplayName = "fix-P-DisplayName"
	exp.AddonPlan.BindTemplate = internal.AddonPlanBindTemplate("template")

	exp.Chart.Name = internal.ChartName("fix-C-Name")
	exp.Chart.Version = *semver.MustParse("1.2.3")

	exp.Service.ID = internal.ServiceID(exp.Addon.ID)
	exp.ServicePlan.ID = internal.ServicePlanID(exp.AddonPlan.ID)

	exp.Namespace = internal.Namespace("fix-namespace")
	exp.ReleaseName = internal.ReleaseName(fmt.Sprintf(
		"hb-%s-%s-%s",
		strings.Trim(string(exp.Addon.Name[:6]), "-"),
		strings.Trim(string(exp.AddonPlan.Name[:6]), "-"),
		exp.InstanceID))
	exp.ReleaseInfo.Revision = 123
	exp.ReleaseInfo.ConfigValues = map[string]interface{}{}
	exp.ProvisioningParameters = &internal.RequestParameters{
		Data: map[string]interface{}{
			"addonsRepositoryURL": exp.Addon.RepositoryURL,
		},
	}
	exp.ParamsHash = "just-regular-hashed-string-b64"
	exp.RequestProvisioningParameters = map[string]interface{}{
		"addonsRepositoryURL": "different-fix-url",
	}
}

func (exp *expAll) NewInstanceCollection() []*internal.Instance {
	return []*internal.Instance{
		&internal.Instance{
			ServiceID: "new-id-not-exist-0",
			Namespace: "fix-namespace",
		},
		&internal.Instance{
			ServiceID: "new-id-not-exist-1",
			Namespace: "fix-namespace",
		},
		&internal.Instance{
			ServiceID: "new-id-not-exist-2",
			Namespace: "fix-namespace",
		},
	}
}

func (exp *expAll) NewChart() *chart.Chart {
	return &chart.Chart{
		Metadata: &chart.Metadata{
			Name:    string(exp.Chart.Name),
			Version: exp.Chart.Version.String(),
		},
	}
}

func (exp *expAll) NewAddon() *internal.Addon {
	return &internal.Addon{
		ID:            exp.Addon.ID,
		Version:       exp.Addon.Version,
		Name:          exp.Addon.Name,
		Bindable:      exp.Addon.Bindable,
		RepositoryURL: exp.Addon.RepositoryURL,
		Plans: map[internal.AddonPlanID]internal.AddonPlan{
			exp.AddonPlan.ID: {
				ID:   exp.AddonPlan.ID,
				Name: exp.AddonPlan.Name,
				ChartRef: internal.ChartRef{
					Name:    exp.Chart.Name,
					Version: exp.Chart.Version,
				},
				Metadata: internal.AddonPlanMetadata{
					DisplayName: exp.AddonPlan.DisplayName,
				},
				BindTemplate: exp.AddonPlan.BindTemplate,
			},
		},
	}
}

func (exp *expAll) NewInstance() *internal.Instance {
	return &internal.Instance{
		ID:                     exp.InstanceID,
		ServiceID:              exp.Service.ID,
		ServicePlanID:          exp.ServicePlan.ID,
		ReleaseName:            exp.ReleaseName,
		Namespace:              exp.Namespace,
		ProvisioningParameters: exp.ProvisioningParameters,
		ParamsHash:             exp.ParamsHash,
	}
}

func (exp *expAll) NewReleaseInfo() internal.ReleaseInfo {
	return internal.ReleaseInfo{
		ReleaseTime: exp.ReleaseInfo.ReleaseTime,
		Revision:    exp.ReleaseInfo.Revision,
		Config:      exp.ReleaseInfo.Config,
	}
}

func (exp *expAll) NewInstanceWithInfo() *internal.Instance {
	r := exp.NewReleaseInfo()
	return &internal.Instance{
		ID:                     exp.InstanceID,
		ServiceID:              exp.Service.ID,
		ServicePlanID:          exp.ServicePlan.ID,
		ReleaseName:            exp.ReleaseName,
		Namespace:              exp.Namespace,
		ProvisioningParameters: exp.ProvisioningParameters,
		ReleaseInfo:            r,
	}
}

func (exp *expAll) NewInstanceBindData(cr internal.InstanceCredentials) *internal.InstanceBindData {
	return &internal.InstanceBindData{
		InstanceID:  exp.InstanceID,
		Credentials: cr,
	}
}

func (exp *expAll) NewInstanceCredentials() *internal.InstanceCredentials {
	return &internal.InstanceCredentials{
		"password": "secret",
	}
}

func (exp *expAll) NewInstanceOperation(tpe internal.OperationType, state internal.OperationState) *internal.InstanceOperation {
	return &internal.InstanceOperation{
		InstanceID:             exp.InstanceID,
		OperationID:            exp.OperationID,
		Type:                   tpe,
		State:                  state,
		ProvisioningParameters: exp.ProvisioningParameters,
	}
}

func (exp *expAll) NewInstanceOperationWithEmptyParams(tpe internal.OperationType, state internal.OperationState) *internal.InstanceOperation {
	return &internal.InstanceOperation{
		InstanceID:             exp.InstanceID,
		OperationID:            exp.OperationID,
		Type:                   tpe,
		State:                  state,
		ProvisioningParameters: &internal.RequestParameters{Data: make(map[string]interface{})},
	}
}

func (exp *expAll) NewBindOperation(tpe internal.OperationType, state internal.OperationState) *internal.BindOperation {
	return &internal.BindOperation{
		InstanceID:  exp.InstanceID,
		BindingID:   exp.BindingID,
		OperationID: exp.OperationID,
		Type:        tpe,
		State:       state,
	}
}

func (exp *expAll) NewBindOperationCollection() []*internal.BindOperation {
	return []*internal.BindOperation{
		&internal.BindOperation{
			InstanceID:  exp.InstanceID,
			BindingID:   exp.BindingID,
			OperationID: exp.OperationID,
			Type:        internal.OperationTypeCreate,
			State:       internal.OperationStateSucceeded,
		},
		&internal.BindOperation{
			InstanceID:  "new-id-not-exist-1",
			BindingID:   "new-bid-not-exists-1",
			OperationID: "new-opid-not-exists-1",
			Type:        internal.OperationTypeCreate,
			State:       internal.OperationStateSucceeded,
		},
		&internal.BindOperation{
			InstanceID:  "new-id-not-exist-1",
			BindingID:   "new-bid-not-exists-1",
			OperationID: "new-opid-not-exists-1",
			Type:        internal.OperationTypeCreate,
			State:       internal.OperationStateSucceeded,
		},
	}
}
