package docs

import (
	"context"

	"reflect"

	"time"

	"github.com/kyma-project/helm-broker/internal"
	"github.com/kyma-project/rafter/pkg/apis/rafter/v1beta1"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
	apiErrors "k8s.io/apimachinery/pkg/api/errors"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/apimachinery/pkg/util/wait"
	"k8s.io/client-go/util/retry"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

// Provider allows to maintain the addons namespace-scoped documentation
type Provider struct {
	dynamicClient client.Client
	namespace     string

	log logrus.FieldLogger
}

// NewProvider creates a new Provider
func NewProvider(dynamicClient client.Client, log logrus.FieldLogger) *Provider {
	return &Provider{
		dynamicClient: dynamicClient,
		log:           log,
	}
}

const (
	rafterLabelKey = "rafter.kyma-project.io/view-context"
	hbLabelKey     = "helm-broker.kyma-project.io/addon-docs"
)

// SetNamespace sets service's working namespace
func (d *Provider) SetNamespace(namespace string) {
	d.namespace = namespace
}

// EnsureAssetGroup creates AssetGroup for a given addon or updates it in case it already exists
func (d *Provider) EnsureAssetGroup(addon *internal.Addon) error {
	addon.Docs[0].Template.Sources = defaultDocsSourcesURLs(addon)
	dt := &v1beta1.AssetGroup{
		ObjectMeta: v1.ObjectMeta{
			Name:      string(addon.ID),
			Namespace: d.namespace,
			Labels: map[string]string{
				rafterLabelKey: "service-catalog",
				hbLabelKey:     "true",
			},
		},
		Spec: v1beta1.AssetGroupSpec{CommonAssetGroupSpec: addon.Docs[0].Template},
	}
	d.log.Infof("- ensuring AssetGroup %s/%s", addon.ID, d.namespace)

	return wait.PollImmediate(time.Millisecond*500, time.Second*3, func() (bool, error) {
		err := d.dynamicClient.Create(context.Background(), dt)
		switch {
		case err == nil:
		case apiErrors.IsAlreadyExists(err):
			if err := d.updateAssetGroup(addon, d.namespace); err != nil {
				d.log.Errorf("while AssetGroup %s already exists", addon.ID)
				return false, nil
			}
		default:
			d.log.Errorf("while creating AssetGroup %s", addon.ID)
			return false, nil
		}
		return true, nil
	})
}

// EnsureAssetGroupRemoved removes AssetGroup for a given addon
func (d *Provider) EnsureAssetGroupRemoved(id string) error {
	dt := &v1beta1.AssetGroup{
		ObjectMeta: v1.ObjectMeta{
			Name:      id,
			Namespace: d.namespace,
		},
	}
	d.log.Infof("- removing AssetGroup %s/%s", id, d.namespace)

	return wait.PollImmediate(time.Millisecond*500, time.Second*3, func() (bool, error) {
		err := d.dynamicClient.Delete(context.Background(), dt)
		if err != nil && !apiErrors.IsNotFound(err) {
			d.log.Errorf("while deleting AssetGroup %s", id)
			return false, nil
		}
		return true, nil
	})
}

func defaultDocsSourcesURLs(addon *internal.Addon) []v1beta1.Source {
	// we use repositoryURL as the default sourceURL if its not provided
	var sources []v1beta1.Source
	for _, source := range addon.Docs[0].Template.Sources {
		if source.URL == "" {
			source.URL = addon.RepositoryURL
		}
		sources = append(sources, source)
	}
	return sources
}

func (d *Provider) updateAssetGroup(addon *internal.Addon, namespace string) error {
	dt := &v1beta1.AssetGroup{}

	return retry.RetryOnConflict(retry.DefaultBackoff, func() error {
		if err := d.dynamicClient.Get(context.Background(), types.NamespacedName{Name: string(addon.ID), Namespace: namespace}, dt); err != nil {
			return errors.Wrapf(err, "while getting AssetGroup %s", addon.ID)
		}
		if reflect.DeepEqual(dt.Spec.CommonAssetGroupSpec, addon.Docs[0].Template) {
			return nil
		}
		dt.Spec = v1beta1.AssetGroupSpec{CommonAssetGroupSpec: addon.Docs[0].Template}

		if err := d.dynamicClient.Update(context.Background(), dt); err != nil {
			return err
		}
		return nil
	})
}
