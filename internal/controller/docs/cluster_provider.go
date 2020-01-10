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

// EnsureAssetGroup creates ClusterAssetGroup for a given addon or updates it in case it already exists
func (d *ClusterProvider) EnsureAssetGroup(addon *internal.Addon) error {
	addon.Docs[0].Template.Sources = defaultDocsSourcesURLs(addon)
	cdt := &v1beta1.ClusterAssetGroup{
		ObjectMeta: v1.ObjectMeta{
			Name: string(addon.ID),
			Labels: map[string]string{
				rafterLabelKey: "service-catalog",
				hbLabelKey:     "true",
			},
		},
		Spec: v1beta1.ClusterAssetGroupSpec{CommonAssetGroupSpec: addon.Docs[0].Template},
	}
	d.log.Infof("- ensuring ClusterAssetGroup %s", addon.ID)

	return wait.PollImmediate(time.Millisecond*500, time.Second*3, func() (bool, error) {
		err := d.dynamicClient.Create(context.Background(), cdt)
		switch {
		case err == nil:
		case apiErrors.IsAlreadyExists(err):
			if err := d.updateClusterAssetGroup(addon); err != nil {
				d.log.Errorf("while ClusterAssetGroup %s already exists", addon.ID)
				return false, nil
			}
		default:
			d.log.Errorf("while creating ClusterAssetGroup %s", addon.ID)
			return false, nil
		}
		return true, nil
	})
}

// EnsureAssetGroupRemoved removes ClusterAssetGroup for a given addon
func (d *ClusterProvider) EnsureAssetGroupRemoved(id string) error {
	cdt := &v1beta1.ClusterAssetGroup{
		ObjectMeta: v1.ObjectMeta{
			Name: id,
		},
	}
	d.log.Infof("- removing ClusterAssetGroup %s", id)

	return wait.PollImmediate(time.Millisecond*500, time.Second*3, func() (bool, error) {
		err := d.dynamicClient.Delete(context.Background(), cdt)
		if err != nil && !apiErrors.IsNotFound(err) {
			d.log.Errorf("while deleting ClusterAssetGroup %s", id)
			return false, nil
		}
		return true, nil
	})
}

func (d *ClusterProvider) updateClusterAssetGroup(addon *internal.Addon) error {
	cdt := &v1beta1.ClusterAssetGroup{}
	return retry.RetryOnConflict(retry.DefaultBackoff, func() error {
		if err := d.dynamicClient.Get(context.Background(), types.NamespacedName{Name: string(addon.ID)}, cdt); err != nil {
			return errors.Wrapf(err, "while getting ClusterAssetGroup %s", addon.ID)
		}
		if reflect.DeepEqual(cdt.Spec.CommonAssetGroupSpec, addon.Docs[0].Template) {
			return nil
		}
		cdt.Spec = v1beta1.ClusterAssetGroupSpec{CommonAssetGroupSpec: addon.Docs[0].Template}

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
