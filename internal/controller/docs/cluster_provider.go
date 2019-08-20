package docs

import (
	"context"

	"reflect"

	"github.com/kyma-project/helm-broker/internal"
	"github.com/kyma-project/kyma/components/cms-controller-manager/pkg/apis/cms/v1alpha1"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
	apiErrors "k8s.io/apimachinery/pkg/api/errors"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

// ClusterProvider allows to maintain the addons cluster-wide documentation
type ClusterProvider struct {
	dynamicClient client.Client
	log           logrus.FieldLogger
}

// NewClusterProvider creates a new Provider
func NewClusterProvider(dynamicClient client.Client, log logrus.FieldLogger) *ClusterProvider {
	return &ClusterProvider{
		dynamicClient: dynamicClient,
		log:           log.WithField("cluster-docs", "provider"),
	}
}

// EnsureDocsTopic creates ClusterDocsTopic for a given addon or updates it in case it already exists
func (d *ClusterProvider) EnsureDocsTopic(addon *internal.Addon) error {
	addon.Docs[0].Template.Sources = defaultDocsSourcesURLs(addon)
	cdt := &v1alpha1.ClusterDocsTopic{
		ObjectMeta: v1.ObjectMeta{
			Name: string(addon.ID),
			Labels: map[string]string{
				cmsLabelKey: "service-catalog",
				hbLabelKey:  "true",
			},
		},
		Spec: v1alpha1.ClusterDocsTopicSpec{CommonDocsTopicSpec: addon.Docs[0].Template},
	}

	d.log.Infof("- ensuring ClusterDocsTopic %s", addon.ID)
	err := d.dynamicClient.Create(context.Background(), cdt)
	switch {
	case err == nil:
	case apiErrors.IsAlreadyExists(err):
		if err := d.updateClusterDocsTopic(addon); err != nil {
			return errors.Wrapf(err, "while ClusterDocsTopic %s already exists", addon.ID)
		}
	default:
		return errors.Wrapf(err, "while creating ClusterDocsTopic %s", addon.ID)
	}

	return nil
}

// EnsureDocsTopicRemoved removes ClusterDocsTopic for a given addon
func (d *ClusterProvider) EnsureDocsTopicRemoved(id string) error {
	cdt := &v1alpha1.ClusterDocsTopic{
		ObjectMeta: v1.ObjectMeta{
			Name: id,
		},
	}
	d.log.Infof("- removing ClusterDocsTopic %s", id)
	err := d.dynamicClient.Delete(context.Background(), cdt)
	if err != nil && !apiErrors.IsNotFound(err) {
		return errors.Wrapf(err, "while deleting ClusterDocsTopic %s", id)
	}
	return nil
}

func (d *ClusterProvider) updateClusterDocsTopic(addon *internal.Addon) error {
	cdt := &v1alpha1.ClusterDocsTopic{}
	if err := d.dynamicClient.Get(context.Background(), types.NamespacedName{Name: string(addon.ID)}, cdt); err != nil {
		return errors.Wrapf(err, "while getting ClusterDocsTopic %s", addon.ID)
	}
	if reflect.DeepEqual(cdt.Spec.CommonDocsTopicSpec, addon.Docs[0].Template) {
		return nil
	}
	cdt.Spec = v1alpha1.ClusterDocsTopicSpec{CommonDocsTopicSpec: addon.Docs[0].Template}

	if err := d.dynamicClient.Update(context.Background(), cdt); err != nil {
		return errors.Wrapf(err, "while updating ClusterDocsTopic %s", addon.ID)
	}

	return nil
}

// SetNamespace sets service's working namespace
func (d *ClusterProvider) SetNamespace(namespace string) {
	return
}
