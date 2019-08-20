package controller

import (
	"context"
	"testing"

	"github.com/kyma-project/helm-broker/internal"
	"github.com/kyma-project/helm-broker/pkg/apis/addons/v1alpha1"
	"github.com/sirupsen/logrus"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"k8s.io/apimachinery/pkg/types"
)

func TestCommonAddonsClient_IsNamespaceScoped(t *testing.T) {
	client := NewCommonClient(nil, logrus.New())

	assert.False(t, client.IsNamespaceScoped())

	client.SetNamespace("test")

	assert.True(t, client.IsNamespaceScoped())

}

func TestCommonAddonsClient_ReprocessRequest(t *testing.T) {
	clusterConfiguration := fixClusterAddonsConfiguration()
	configuration := fixAddonsConfiguration()
	ts := getTestSuite(t, clusterConfiguration, configuration)

	client := NewCommonClient(ts.mgr.GetClient(), logrus.New())

	assert.NoError(t, client.ReprocessRequest(clusterConfiguration.Name))

	client.SetNamespace(configuration.Namespace)
	assert.NoError(t, client.ReprocessRequest(configuration.Name))
}

func TestCommonAddonsClient_ListConfigurations(t *testing.T) {
	clusterConfiguration := fixClusterAddonsConfiguration()
	configuration := fixAddonsConfiguration()
	ts := getTestSuite(t, clusterConfiguration, configuration)

	client := NewCommonClient(ts.mgr.GetClient(), logrus.New())

	add, err := client.ListConfigurations()
	require.NoError(t, err)
	assert.Equal(t, clusterConfiguration.Name, add[0].Meta.Name)

	client.SetNamespace(configuration.Namespace)
	add, err = client.ListConfigurations()
	require.NoError(t, err)
	assert.Equal(t, configuration.Name, add[0].Meta.Name)
}

func TestCommonAddonsClient_UpdateConfiguration(t *testing.T) {
	clusterConfiguration := fixClusterAddonsConfiguration()
	configuration := fixAddonsConfiguration()
	ts := getTestSuite(t, clusterConfiguration, configuration)

	client := NewCommonClient(ts.mgr.GetClient(), logrus.New())

	fix := "fix"
	expFinalizers := []string{fix}
	expURLs := []v1alpha1.SpecRepository{
		{
			URL: fix,
		},
	}

	clusterConfiguration.ObjectMeta.Finalizers = expFinalizers
	clusterConfiguration.Spec.Repositories = expURLs
	err := client.UpdateConfiguration(&internal.CommonAddon{
		Meta:   clusterConfiguration.ObjectMeta,
		Status: clusterConfiguration.Status.CommonAddonsConfigurationStatus,
		Spec:   clusterConfiguration.Spec.CommonAddonsConfigurationSpec,
	})
	require.NoError(t, err)

	result := &v1alpha1.ClusterAddonsConfiguration{}
	err = client.Get(context.Background(), types.NamespacedName{Name: clusterConfiguration.Name}, result)
	require.NoError(t, err)
	assert.Equal(t, expURLs, result.Spec.Repositories)
	assert.Equal(t, expFinalizers, result.Finalizers)

	configuration.ObjectMeta.Finalizers = expFinalizers
	configuration.Spec.Repositories = expURLs
	client.SetNamespace(configuration.Namespace)
	err = client.UpdateConfiguration(&internal.CommonAddon{
		Meta:   configuration.ObjectMeta,
		Status: configuration.Status.CommonAddonsConfigurationStatus,
		Spec:   configuration.Spec.CommonAddonsConfigurationSpec,
	})
	require.NoError(t, err)

	nsResult := &v1alpha1.AddonsConfiguration{}
	client.Get(context.Background(), types.NamespacedName{Name: configuration.Name, Namespace: configuration.Namespace}, nsResult)
	assert.Equal(t, expURLs, nsResult.Spec.Repositories)
	assert.Equal(t, expFinalizers, nsResult.Finalizers)

}

func TestCommonAddonsClient_UpdateConfigurationStatus(t *testing.T) {
	clusterConfiguration := fixClusterAddonsConfiguration()
	configuration := fixAddonsConfiguration()
	ts := getTestSuite(t, clusterConfiguration, configuration)

	client := NewCommonClient(ts.mgr.GetClient(), logrus.New())
	expStatus := v1alpha1.AddonsConfigurationReady

	clusterConfiguration.Status.CommonAddonsConfigurationStatus.Phase = expStatus
	err := client.UpdateConfigurationStatus(&internal.CommonAddon{
		Meta:   clusterConfiguration.ObjectMeta,
		Status: clusterConfiguration.Status.CommonAddonsConfigurationStatus,
		Spec:   clusterConfiguration.Spec.CommonAddonsConfigurationSpec,
	})
	require.NoError(t, err)

	result := &v1alpha1.ClusterAddonsConfiguration{}
	err = client.Get(context.Background(), types.NamespacedName{Name: clusterConfiguration.Name}, result)
	require.NoError(t, err)
	assert.Equal(t, expStatus, result.Status.Phase)

	configuration.Status.CommonAddonsConfigurationStatus.Phase = expStatus
	client.SetNamespace(configuration.Namespace)
	err = client.UpdateConfigurationStatus(&internal.CommonAddon{
		Meta:   configuration.ObjectMeta,
		Status: configuration.Status.CommonAddonsConfigurationStatus,
		Spec:   configuration.Spec.CommonAddonsConfigurationSpec,
	})
	require.NoError(t, err)

	nsResult := &v1alpha1.AddonsConfiguration{}
	err = client.Get(context.Background(), types.NamespacedName{Name: configuration.Name, Namespace: configuration.Namespace}, nsResult)
	require.NoError(t, err)
	assert.Equal(t, expStatus, nsResult.Status.Phase)

}
