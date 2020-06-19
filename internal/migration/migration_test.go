package migration

import (
	"context"
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/kyma-project/helm-broker/internal"
	"github.com/kyma-project/helm-broker/internal/storage"
	"github.com/kyma-project/helm-broker/pkg/apis"
	"github.com/kyma-project/helm-broker/pkg/apis/addons/v1alpha1"
	"github.com/sirupsen/logrus"
	"github.com/stretchr/testify/require"
	v1 "k8s.io/api/apps/v1"
	k8sMeta "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/kubernetes/scheme"
	"k8s.io/helm/pkg/proto/hapi/chart"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
)

const (
	deployName      = "app"
	deployNamespace = "test"
)

func TestExecutor_Execute(t *testing.T) {
	require.NoError(t, apis.AddToScheme(scheme.Scheme))

	for tn, tc := range map[string]struct {
		instances []*internal.Instance
		addons    []runtime.Object
	}{
		"success": {
			instances: []*internal.Instance{
				fixInstance("test", internal.ReleaseInfo{
					Config: &chart.Config{
						Raw: "{\"test\": \"test\"}",
					},
				}),
				fixInstance("test1", internal.ReleaseInfo{}),
			},
			addons: []runtime.Object{
				fixAddon("test"),
				fixAddon("test1"),
				fixClusterAddon("test"),
			},
		},
	} {
		t.Run(tn, func(t *testing.T) {
			// given
			sFact, err := storage.NewFactory(storage.NewConfigListAllMemory())
			require.NoError(t, err)
			instances := sFact.Instance()

			for _, instance := range tc.instances {
				_, err := instances.Upsert(instance)
				require.NoError(t, err)
			}
			cli := fake.NewFakeClientWithScheme(scheme.Scheme, tc.addons...)
			err = cli.Create(context.Background(), &v1.Deployment{
				ObjectMeta: k8sMeta.ObjectMeta{
					Name:      deployName,
					Namespace: deployNamespace,
				},
			})
			require.NoError(t, err)

			// when
			err = New(cli, instances, deployName, deployNamespace, logrus.New()).Execute()

			// then
			require.NoError(t, err)
			checkAddons(t, cli, tc.addons)
			checkInstances(t, instances, tc.instances)
			checkLabel(t, cli)
		})
	}
}

func TestExecutor_WhenLabelled(t *testing.T) {
	cli := fake.NewFakeClientWithScheme(scheme.Scheme)
	err := cli.Create(context.Background(), &v1.Deployment{
		ObjectMeta: k8sMeta.ObjectMeta{
			Name:      deployName,
			Namespace: deployNamespace,
			Labels: map[string]string{
				markLabel: markLabelValue,
			},
		},
	})
	require.NoError(t, err)

	// when
	err = New(cli, nil, deployName, deployNamespace, logrus.New()).Execute()

	// then
	require.NoError(t, err)

}

func checkLabel(t *testing.T, cli client.Client) {
	obj := &v1.Deployment{}
	err := cli.Get(context.Background(), client.ObjectKey{Name: deployName, Namespace: deployNamespace}, obj)
	require.NoError(t, err)
	require.Equal(t, markLabelValue, obj.Labels[markLabel])
}

func checkInstances(t *testing.T, store storage.Instance, instances []*internal.Instance) {
	for _, instance := range instances {
		i, err := store.Get(instance.ID)
		require.NoError(t, err)

		if instance.ReleaseInfo.Config != nil {
			res, err := json.Marshal(instance.ReleaseInfo.Config.Raw)
			require.NoError(t, err)
			assert.Equal(t, instance.ReleaseInfo.Config.Raw, string(res))
		}

		assert.True(t, i.ReleaseInfo.Config == nil)
	}
}

func checkAddons(t *testing.T, cli client.Client, addons []runtime.Object) {
	for _, addon := range addons {
		ad, ok := addon.(*v1alpha1.AddonsConfiguration)
		if !ok {
			clusterAdd, ok := addon.(*v1alpha1.ClusterAddonsConfiguration)
			require.True(t, ok)

			addToCheck := &v1alpha1.ClusterAddonsConfiguration{}
			err := cli.Get(context.Background(), client.ObjectKey{
				Name: clusterAdd.Name,
			}, addToCheck)
			require.NoError(t, err)
			require.Equal(t, int64(1), addToCheck.Spec.ReprocessRequest)
		} else {
			addToCheck := &v1alpha1.AddonsConfiguration{}
			err := cli.Get(context.Background(), client.ObjectKey{
				Name:      ad.Name,
				Namespace: ad.Namespace,
			}, addToCheck)
			require.NoError(t, err)
			require.Equal(t, int64(1), addToCheck.Spec.ReprocessRequest)
		}
	}
}

func fixAddon(name string) *v1alpha1.AddonsConfiguration {
	return &v1alpha1.AddonsConfiguration{
		ObjectMeta: k8sMeta.ObjectMeta{
			Name:      name,
			Namespace: name,
		},
	}
}

func fixClusterAddon(name string) *v1alpha1.ClusterAddonsConfiguration {
	return &v1alpha1.ClusterAddonsConfiguration{
		ObjectMeta: k8sMeta.ObjectMeta{
			Name: name,
		},
	}
}

func fixInstance(id string, info internal.ReleaseInfo) *internal.Instance {
	return &internal.Instance{
		ID:          internal.InstanceID(id),
		ReleaseInfo: info,
	}
}
