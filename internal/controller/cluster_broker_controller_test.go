package controller_test

import (
	"context"
	"testing"

	"github.com/kubernetes-sigs/service-catalog/pkg/apis/servicecatalog/v1beta1"
	"github.com/kyma-project/helm-broker/internal/controller"
	"github.com/kyma-project/helm-broker/internal/controller/broker"
	"github.com/kyma-project/helm-broker/internal/controller/instance"
	"github.com/kyma-project/helm-broker/internal/platform/logger/spy"
	"github.com/kyma-project/helm-broker/pkg/apis/addons/v1alpha1"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/kubernetes/scheme"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"

	"k8s.io/apimachinery/pkg/api/errors"
)

func TestClusterBrokerControllerReconcile_CreateCSB(t *testing.T) {
	// given
	svc, cli := prepareClusterBrokerController(t, fixReadyClusterAddonsConfiguration())

	// when
	res, err := svc.Reconcile(fixRequest())

	// then
	require.NoError(t, err)
	assert.False(t, res.Requeue)
	_, err = getClusterServiceBroker(cli)
	assert.NoError(t, err)
}

func TestClusterBrokerControllerReconcile_DeleteSB(t *testing.T) {
	// given
	svc, cli := prepareClusterBrokerController(t, fixClusterServiceBroker())

	// when
	res, err := svc.Reconcile(fixRequest())

	// then
	require.NoError(t, err)
	assert.False(t, res.Requeue)
	_, err = getClusterServiceBroker(cli)
	assert.True(t, errors.IsNotFound(err))
}

func TestClusterBrokerControllerReconcile_BlockDeletionByExistingInstances(t *testing.T) {
	// given
	svc, cli := prepareClusterBrokerController(t, fixClusterServiceBroker(), fixServiceInstanceForClusterServiceClass(), fixClusterServiceClass())

	// when
	res, err := svc.Reconcile(fixRequest())

	// then
	require.NoError(t, err)
	assert.False(t, res.Requeue)
	_, err = getClusterServiceBroker(cli)
	assert.NoError(t, err)
}

func fixRequest() reconcile.Request {
	return reconcile.Request{
		NamespacedName: types.NamespacedName{
			Name:      readyAddonsConfigName,
			Namespace: "default",
		},
	}
}

func getClusterServiceBroker(cli client.Client) (*v1beta1.ClusterServiceBroker, error) {
	var obj v1beta1.ClusterServiceBroker
	err := cli.Get(context.TODO(), client.ObjectKey{Name: broker.NamespacedBrokerName}, &obj)
	return &obj, err
}

func fixReadyClusterAddonsConfiguration() *v1alpha1.ClusterAddonsConfiguration {
	return &v1alpha1.ClusterAddonsConfiguration{
		ObjectMeta: metav1.ObjectMeta{
			Name:      readyAddonsConfigName,
			Namespace: "default",
		},
		Spec: v1alpha1.ClusterAddonsConfigurationSpec{
			CommonAddonsConfigurationSpec: v1alpha1.CommonAddonsConfigurationSpec{},
		},
		Status: v1alpha1.ClusterAddonsConfigurationStatus{
			CommonAddonsConfigurationStatus: v1alpha1.CommonAddonsConfigurationStatus{
				Phase: v1alpha1.AddonsConfigurationReady,
			},
		},
	}
}

func fixClusterServiceBroker() *v1beta1.ClusterServiceBroker {
	return &v1beta1.ClusterServiceBroker{
		ObjectMeta: metav1.ObjectMeta{
			Name: broker.NamespacedBrokerName,
		},
		Spec: v1beta1.ClusterServiceBrokerSpec{},
	}
}

func fixServiceInstanceForClusterServiceClass() *v1beta1.ServiceInstance {
	return &v1beta1.ServiceInstance{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "si",
			Namespace: "default",
		},
		Spec: v1beta1.ServiceInstanceSpec{
			ClusterServiceClassRef: &v1beta1.ClusterObjectReference{
				Name: "sc-service",
			},
		}}
}

func fixClusterServiceClass() *v1beta1.ClusterServiceClass {
	return &v1beta1.ClusterServiceClass{
		ObjectMeta: metav1.ObjectMeta{
			Name: "sc-service",
		},
		Spec: v1beta1.ClusterServiceClassSpec{
			ClusterServiceBrokerName: broker.NamespacedBrokerName,
		}}
}

func prepareClusterBrokerController(t *testing.T, initObjs ...runtime.Object) (*controller.ClusterBrokerController, client.Client) {
	require.NoError(t, v1beta1.AddToScheme(scheme.Scheme))
	require.NoError(t, v1alpha1.AddToScheme(scheme.Scheme))
	cli := fake.NewFakeClientWithScheme(scheme.Scheme, initObjs...)
	iChecker := instance.New(cli, broker.NamespacedBrokerName)
	bFacade := broker.NewClusterBrokersFacade(cli, "default", "helm-broker", broker.NamespacedBrokerName, spy.NewLogDummy())
	svc := controller.NewClusterBrokerController(iChecker, cli, bFacade, broker.NamespacedBrokerName)

	return svc, cli
}
