package controller

import (
	"context"

	"github.com/kyma-project/helm-broker/internal"
	"github.com/kyma-project/helm-broker/pkg/apis/addons/v1alpha1"
	"github.com/pkg/errors"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"github.com/sirupsen/logrus"
	"k8s.io/client-go/util/retry"
)

// CommonAddonsClient holds shared client for controllers
type CommonAddonsClient struct {
	client.Client
	namespace string

	log logrus.FieldLogger
}

// NewCommonAddonsClient creates a new CommonAddonsClient
func NewCommonAddonsClient(cli client.Client, log logrus.FieldLogger) *CommonAddonsClient {
	return &CommonAddonsClient{
		Client:    cli,
		namespace: string(internal.ClusterWide),

		log: log.WithField("common", "client"),
	}
}

// IsNamespaceScoped return true if service is namespace-scoped
func (a *CommonAddonsClient) IsNamespaceScoped() bool {
	return a.namespace != string(internal.ClusterWide)
}

// SetNamespace sets service's working namespace
func (a *CommonAddonsClient) SetNamespace(namespace string) {
	a.namespace = namespace
}

// UpdateConfiguration updates ClusterAddonsConfiguration or AddonsConfiguration if namespace is set
func (a *CommonAddonsClient) UpdateConfiguration(addon *internal.CommonAddon) error {
	err := retry.RetryOnConflict(retry.DefaultRetry, func() error {
		if a.IsNamespaceScoped() {
			ad := &v1alpha1.AddonsConfiguration{}
			if err := a.Get(context.Background(), types.NamespacedName{Name: addon.Meta.Name, Namespace: a.namespace}, ad); err != nil {
				return err
			}
			ad.Finalizers = addon.Meta.Finalizers
			ad.Spec.CommonAddonsConfigurationSpec = addon.Spec

			if err := a.Update(context.Background(), ad); err != nil {
				a.log.Infof("%v ERROR: %v", ad, err)
				return err
			}
		} else {
			ad := &v1alpha1.ClusterAddonsConfiguration{}
			if err := a.Get(context.Background(), types.NamespacedName{Name: addon.Meta.Name}, ad); err != nil {
				return err
			}
			ad.ObjectMeta.Finalizers = addon.Meta.Finalizers
			ad.Spec.CommonAddonsConfigurationSpec = addon.Spec

			if err := a.Update(context.Background(), ad); err != nil {
				return err
			}
			return nil
		}
		return nil
	})
	if err != nil {
		return errors.Wrapf(err, "while updating addons configuration %s/%s", a.namespace, addon.Meta.Name)
	}
	return nil
}

// UpdateConfigurationStatus updates ClusterAddonsConfiguration or AddonsConfiguration status if namespace is set
func (a *CommonAddonsClient) UpdateConfigurationStatus(addon *internal.CommonAddon) error {
	err := retry.RetryOnConflict(retry.DefaultRetry, func() error {
		if a.IsNamespaceScoped() {
			ad := &v1alpha1.AddonsConfiguration{}
			if err := a.Get(context.Background(), types.NamespacedName{Name: addon.Meta.Name, Namespace: a.namespace}, ad); err != nil {
				return err
			}
			ad.Status.CommonAddonsConfigurationStatus = addon.Status

			if err := a.Status().Update(context.Background(), ad); err != nil {
				return err
			}
		} else {
			ad := &v1alpha1.ClusterAddonsConfiguration{}
			if err := a.Get(context.Background(), types.NamespacedName{Name: addon.Meta.Name}, ad); err != nil {
				return err
			}
			ad.Status.CommonAddonsConfigurationStatus = addon.Status

			if err := a.Status().Update(context.Background(), ad); err != nil {
				return err
			}
		}
		return nil
	})
	if err != nil {
		return errors.Wrapf(err, "while updating addons configuration %s/%s status", a.namespace, addon.Meta.Name)
	}
	return nil
}

// ListConfigurations lists ClusterAddonsConfiguration or AddonsConfiguration if namespace is set
func (a *CommonAddonsClient) ListConfigurations() ([]internal.CommonAddon, error) {
	var commonAddons []internal.CommonAddon
	if a.IsNamespaceScoped() {
		addonsConfigurationList := &v1alpha1.AddonsConfigurationList{}

		err := a.List(context.TODO(), &client.ListOptions{Namespace: a.namespace}, addonsConfigurationList)
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

		err := a.List(context.TODO(), &client.ListOptions{}, addonsConfigurationList)
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
func (a *CommonAddonsClient) ReprocessRequest(addonName string) error {
	err := retry.RetryOnConflict(retry.DefaultRetry, func() error {
		if a.IsNamespaceScoped() {
			ad := &v1alpha1.AddonsConfiguration{}
			if err := a.Get(context.Background(), types.NamespacedName{Name: addonName, Namespace: a.namespace}, ad); err != nil {
				return errors.Wrapf(err, "while getting AddonsConfiguration %s", addonName)
			}
			ad.Spec.ReprocessRequest++
			if err := a.Update(context.Background(), ad); err != nil {
				return errors.Wrapf(err, "while updating AddonsConfiguration %s", addonName)
			}
		} else {
			ad := &v1alpha1.ClusterAddonsConfiguration{}
			if err := a.Get(context.Background(), types.NamespacedName{Name: addonName}, ad); err != nil {
				return errors.Wrapf(err, "while getting ClusterAddonsConfiguration %s", addonName)
			}
			ad.Spec.ReprocessRequest++
			if err := a.Update(context.Background(), ad); err != nil {
				return errors.Wrapf(err, "while updating ClusterAddonsConfiguration %s", addonName)
			}
		}
		return nil
	})
	if err != nil {
		return errors.Wrapf(err, "while updating addons configuration %s/%s status", a.namespace, addonName)
	}
	return nil
}
