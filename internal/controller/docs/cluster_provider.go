package docs

import (
	"context"

	"reflect"

	"time"

	"github.com/kyma-project/helm-broker/internal"
	"github.com/kyma-project/kyma/components/cms-controller-manager/pkg/apis/cms/v1alpha1"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
	apiErrors "k8s.io/apimachinery/pkg/api/errors"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/apimachinery/pkg/util/wait"
	"k8s.io/client-go/util/retry"
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
		log:           log,
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

	return wait.PollImmediate(time.Millisecond*500, time.Second*3, func() (bool, error) {
		err := d.dynamicClient.Create(context.Background(), cdt)
		switch {
		case err == nil:
		case apiErrors.IsAlreadyExists(err):
			if err := d.updateClusterDocsTopic(addon); err != nil {
				d.log.Errorf("while ClusterDocsTopic %s already exists", addon.ID)
				return false, nil
			}
		default:
			d.log.Errorf("while creating ClusterDocsTopic %s", addon.ID)
			return false, nil
		}
		return true, nil
	})
}

// EnsureDocsTopicRemoved removes ClusterDocsTopic for a given addon
func (d *ClusterProvider) EnsureDocsTopicRemoved(id string) error {
	cdt := &v1alpha1.ClusterDocsTopic{
		ObjectMeta: v1.ObjectMeta{
			Name: id,
		},
	}
	d.log.Infof("- removing ClusterDocsTopic %s", id)

	return wait.PollImmediate(time.Millisecond*500, time.Second*3, func() (bool, error) {
		err := d.dynamicClient.Delete(context.Background(), cdt)
		if err != nil && !apiErrors.IsNotFound(err) {
			d.log.Errorf("while deleting ClusterDocsTopic %s", id)
			return false, nil
		}
		return true, nil
	})
}

func (d *ClusterProvider) updateClusterDocsTopic(addon *internal.Addon) error {
	cdt := &v1alpha1.ClusterDocsTopic{}
	return retry.RetryOnConflict(retry.DefaultBackoff, func() error {
		if err := d.dynamicClient.Get(context.Background(), types.NamespacedName{Name: string(addon.ID)}, cdt); err != nil {
			return errors.Wrapf(err, "while getting ClusterDocsTopic %s", addon.ID)
		}
		if reflect.DeepEqual(cdt.Spec.CommonDocsTopicSpec, addon.Docs[0].Template) {
			return nil
		}
		cdt.Spec = v1alpha1.ClusterDocsTopicSpec{CommonDocsTopicSpec: addon.Docs[0].Template}

		if err := d.dynamicClient.Update(context.Background(), cdt); err != nil {
			return err
		}
		return nil
	})
}

// SetNamespace sets service's working namespace
func (d *ClusterProvider) SetNamespace(namespace string) {
	return
}
