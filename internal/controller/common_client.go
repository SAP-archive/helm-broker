package controller

import (
	"context"

	"github.com/kyma-project/helm-broker/internal"
	"github.com/kyma-project/helm-broker/pkg/apis/addons/v1alpha1"
	"github.com/pkg/errors"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

// AddonsClient holds shared client for controllers
type AddonsClient struct {
	client.Client
	namespace string
}

// NewAddonsClient creates a new AddonsClient
func NewAddonsClient(cli client.Client) *AddonsClient {
	return &AddonsClient{
		Client:    cli,
		namespace: string(internal.ClusterWide),
	}
}

// IsNamespaceScoped return true if service is namespace-scoped
func (a *AddonsClient) IsNamespaceScoped() bool {
	return a.namespace != string(internal.ClusterWide)
}

// SetNamespace sets service's working namespace
func (a *AddonsClient) SetNamespace(namespace string) {
	a.namespace = namespace
}

// UpdateConfiguration updates ClusterAddonsConfiguration or AddonsConfiguration if namespace is set
func (a *AddonsClient) UpdateConfiguration(addon *internal.CommonAddon) (*internal.CommonAddon, error) {
	if a.IsNamespaceScoped() {
		if err := a.Update(context.Background(), &v1alpha1.AddonsConfiguration{
			ObjectMeta: addon.Meta,
			Spec:       v1alpha1.AddonsConfigurationSpec{CommonAddonsConfigurationSpec: addon.Spec},
			Status:     v1alpha1.AddonsConfigurationStatus{CommonAddonsConfigurationStatus: addon.Status},
		}); err != nil {
			return nil, errors.Wrapf(err, "while updating AddonsConfiguration %s/%s", addon.Meta.Name, addon.Meta.Namespace)
		}
	} else {
		if err := a.Update(context.Background(), &v1alpha1.ClusterAddonsConfiguration{
			ObjectMeta: addon.Meta,
			Spec:       v1alpha1.ClusterAddonsConfigurationSpec{CommonAddonsConfigurationSpec: addon.Spec},
			Status:     v1alpha1.ClusterAddonsConfigurationStatus{CommonAddonsConfigurationStatus: addon.Status},
		}); err != nil {
			return nil, errors.Wrapf(err, "while updating ClusterAddonsConfiguration %s", addon.Meta.Name)
		}
	}
	return addon, nil
}

// UpdateConfigurationStatus updates ClusterAddonsConfiguration or AddonsConfiguration status if namespace is set
func (a *AddonsClient) UpdateConfigurationStatus(addon *internal.CommonAddon) (*internal.CommonAddon, error) {
	if a.IsNamespaceScoped() {
		if err := a.Status().Update(context.Background(), &v1alpha1.AddonsConfiguration{
			ObjectMeta: addon.Meta,
			Status:     v1alpha1.AddonsConfigurationStatus{CommonAddonsConfigurationStatus: addon.Status},
			Spec:       v1alpha1.AddonsConfigurationSpec{CommonAddonsConfigurationSpec: addon.Spec},
		}); err != nil {
			return nil, errors.Wrapf(err, "while updating AddonsConfiguration %s/%s status", addon.Meta.Name, addon.Meta.Namespace)
		}
	} else {
		if err := a.Status().Update(context.Background(), &v1alpha1.ClusterAddonsConfiguration{
			ObjectMeta: addon.Meta,
			Status:     v1alpha1.ClusterAddonsConfigurationStatus{CommonAddonsConfigurationStatus: addon.Status},
			Spec:       v1alpha1.ClusterAddonsConfigurationSpec{CommonAddonsConfigurationSpec: addon.Spec},
		}); err != nil {
			return nil, errors.Wrapf(err, "while updating ClusterAddonsConfiguration %s status", addon.Meta.Name)
		}
	}
	return addon, nil
}

// ListConfigurations lists ClusterAddonsConfiguration or AddonsConfiguration if namespace is set
func (a *AddonsClient) ListConfigurations() ([]internal.CommonAddon, error) {
	var commonAddons []internal.CommonAddon
	if a.IsNamespaceScoped() {
		addonsConfigurationList := &v1alpha1.AddonsConfigurationList{}

		err := a.Client.List(context.TODO(), &client.ListOptions{Namespace: a.namespace}, addonsConfigurationList)
		if err != nil {
			return nil, errors.Wrapf(err, "while fetching AddonConfiguration list from namespace %s", a.namespace)
		}

		for _, addon := range addonsConfigurationList.Items {
			commonAddons = append(commonAddons, internal.CommonAddon{
				Meta:   addon.ObjectMeta,
				Spec:   addon.Spec.CommonAddonsConfigurationSpec,
				Status: addon.Status.CommonAddonsConfigurationStatus,
			})
		}
	} else {
		addonsConfigurationList := &v1alpha1.ClusterAddonsConfigurationList{}

		err := a.Client.List(context.TODO(), &client.ListOptions{}, addonsConfigurationList)
		if err != nil {
			return nil, errors.Wrap(err, "while fetching ClusterAddonConfiguration list")
		}

		for _, addon := range addonsConfigurationList.Items {
			commonAddons = append(commonAddons, internal.CommonAddon{
				Meta:   addon.ObjectMeta,
				Spec:   addon.Spec.CommonAddonsConfigurationSpec,
				Status: addon.Status.CommonAddonsConfigurationStatus,
			})
		}
	}
	return commonAddons, nil
}

// ReprocessRequest bumps reprocessRequest for ClusterAddonsConfiguration or AddonsConfiguration if namespace is set
func (a *AddonsClient) ReprocessRequest(addonName string) error {
	if a.IsNamespaceScoped() {
		ad := &v1alpha1.AddonsConfiguration{}
		if err := a.Client.Get(context.Background(), types.NamespacedName{Name: addonName, Namespace: a.namespace}, ad); err != nil {
			return errors.Wrapf(err, "while getting AddonsConfiguration %s", addonName)
		}
		ad.Spec.ReprocessRequest++
		if err := a.Client.Update(context.Background(), ad); err != nil {
			return errors.Wrapf(err, "while updating AddonsConfiguration %s", addonName)
		}
	} else {
		ad := &v1alpha1.ClusterAddonsConfiguration{}
		if err := a.Client.Get(context.Background(), types.NamespacedName{Name: addonName}, ad); err != nil {
			return errors.Wrapf(err, "while getting ClusterAddonsConfiguration %s", addonName)
		}
		ad.Spec.ReprocessRequest++
		if err := a.Client.Update(context.Background(), ad); err != nil {
			return errors.Wrapf(err, "while updating ClusterAddonsConfiguration %s", addonName)
		}
	}

	return nil
}
