package controller

import (
	"context"

	"github.com/kyma-project/helm-broker/internal"
	"github.com/kyma-project/helm-broker/pkg/apis/addons/v1alpha1"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/util/retry"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

// CommonClient holds shared client for controllers
type CommonClient struct {
	client.Client
	namespace string

	log logrus.FieldLogger
}

// NewCommonClient creates a new CommonClient
func NewCommonClient(cli client.Client, log logrus.FieldLogger) *CommonClient {
	return &CommonClient{
		Client:    cli,
		namespace: string(internal.ClusterWide),

		log: log.WithField("common", "client"),
	}
}

// IsNamespaceScoped return true if service is namespace-scoped
func (a *CommonClient) IsNamespaceScoped() bool {
	return a.namespace != string(internal.ClusterWide)
}

// SetNamespace sets service's working namespace
func (a *CommonClient) SetNamespace(namespace string) {
	a.namespace = namespace
}

// UpdateConfiguration updates ClusterAddonsConfiguration or AddonsConfiguration if namespace is set
func (a *CommonClient) UpdateConfiguration(addon *internal.CommonAddon) error {
	err := retry.RetryOnConflict(retry.DefaultRetry, func() error {
		if a.IsNamespaceScoped() {
			ad := &v1alpha1.AddonsConfiguration{}
			if err := a.Get(context.Background(), types.NamespacedName{Name: addon.Meta.Name, Namespace: a.namespace}, ad); err != nil {
				return err
			}
			ad.Finalizers = addon.Meta.Finalizers
			ad.Spec.CommonAddonsConfigurationSpec = addon.Spec

			if err := a.Update(context.Background(), ad); err != nil {
				return err
			}
			addon.Meta.Generation = ad.Generation
			return nil
		}
		ad := &v1alpha1.ClusterAddonsConfiguration{}
		if err := a.Get(context.Background(), types.NamespacedName{Name: addon.Meta.Name}, ad); err != nil {
			return err
		}
		ad.ObjectMeta.Finalizers = addon.Meta.Finalizers
		ad.Spec.CommonAddonsConfigurationSpec = addon.Spec

		if err := a.Update(context.Background(), ad); err != nil {
			return err
		}
		addon.Meta.Generation = ad.Generation
		return nil
	})
	if err != nil {
		return errors.Wrapf(err, "while updating addons configuration %s/%s", a.namespace, addon.Meta.Name)
	}
	return nil
}

// UpdateConfigurationStatus updates ClusterAddonsConfiguration or AddonsConfiguration status if namespace is set
func (a *CommonClient) UpdateConfigurationStatus(addon *internal.CommonAddon) error {
	err := retry.RetryOnConflict(retry.DefaultRetry, func() error {
		if a.IsNamespaceScoped() {
			ad := &v1alpha1.AddonsConfiguration{}
			if err := a.Get(context.Background(), types.NamespacedName{Name: addon.Meta.Name, Namespace: a.namespace}, ad); err != nil {
				return err
			}
			ad.Status.CommonAddonsConfigurationStatus = addon.Status

			return a.Status().Update(context.Background(), ad)
		}
		ad := &v1alpha1.ClusterAddonsConfiguration{}
		if err := a.Get(context.Background(), types.NamespacedName{Name: addon.Meta.Name}, ad); err != nil {
			return err
		}
		ad.Status.CommonAddonsConfigurationStatus = addon.Status

		return a.Status().Update(context.Background(), ad)
	})
	if err != nil {
		return errors.Wrapf(err, "while updating addons configuration %s/%s status", a.namespace, addon.Meta.Name)
	}
	return nil
}

// ListConfigurations lists ClusterAddonsConfiguration or AddonsConfiguration if namespace is set
func (a *CommonClient) ListConfigurations() ([]internal.CommonAddon, error) {
	var commonAddons []internal.CommonAddon
	if a.IsNamespaceScoped() {
		addonsConfigurationList := &v1alpha1.AddonsConfigurationList{}

		err := a.List(context.TODO(), addonsConfigurationList, client.InNamespace(a.namespace))
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

		err := a.List(context.TODO(), addonsConfigurationList)
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
func (a *CommonClient) ReprocessRequest(addonName string) error {
	err := retry.RetryOnConflict(retry.DefaultRetry, func() error {
		if a.IsNamespaceScoped() {
			ad := &v1alpha1.AddonsConfiguration{}
			if err := a.Get(context.Background(), types.NamespacedName{Name: addonName, Namespace: a.namespace}, ad); err != nil {
				return err
			}
			ad.Spec.ReprocessRequest++
			return a.Update(context.Background(), ad)
		}
		ad := &v1alpha1.ClusterAddonsConfiguration{}
		if err := a.Get(context.Background(), types.NamespacedName{Name: addonName}, ad); err != nil {
			return err
		}
		ad.Spec.ReprocessRequest++
		return a.Update(context.Background(), ad)
	})
	if err != nil {
		return errors.Wrapf(err, "while updating addons configuration %s/%s", a.namespace, addonName)
	}
	return nil
}
