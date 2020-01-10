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
	"k8s.io/client-go/kubernetes/scheme"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"

	"k8s.io/apimachinery/pkg/api/errors"
)

const (
	readyAddonsConfigName = "readyAC"
)

func TestBrokerControllerReconcile_CreateSB(t *testing.T) {
	// given
	svc, cli := prepareBrokerController(t, fixReadyAddonsConfiguration())

	// when
	res, err := svc.Reconcile(fixRequest())

	// then
	require.NoError(t, err)
	assert.False(t, res.Requeue)
	_, err = getServiceBroker(cli, "default")
	assert.NoError(t, err)
}

func TestBrokerControllerReconcile_DeleteSB(t *testing.T) {
	// given
	svc, cli := prepareBrokerController(t, fixServiceBroker())

	// when
	res, err := svc.Reconcile(fixRequest())

	// then
	require.NoError(t, err)
	assert.False(t, res.Requeue)
	_, err = getServiceBroker(cli, "default")
	assert.True(t, errors.IsNotFound(err))
}

func TestBrokerControllerReconcile_BlockDeletionByExistingInstances(t *testing.T) {
	// given
	svc, cli := prepareBrokerController(t, fixServiceBroker(), fixServiceInstance(), fixServiceClass())

	// when
	res, err := svc.Reconcile(fixRequest())

	// then
	require.NoError(t, err)
	assert.False(t, res.Requeue)
	_, err = getServiceBroker(cli, "default")
	assert.NoError(t, err)
}

func getServiceBroker(cli client.Client, ns string) (*v1beta1.ServiceBroker, error) {
	var obj v1beta1.ServiceBroker
	err := cli.Get(context.TODO(), client.ObjectKey{Namespace: "default", Name: broker.NamespacedBrokerName}, &obj)
	return &obj, err
}

func fixReadyAddonsConfiguration() *v1alpha1.AddonsConfiguration {
	return &v1alpha1.AddonsConfiguration{
		ObjectMeta: metav1.ObjectMeta{
			Name:      readyAddonsConfigName,
			Namespace: "default",
		},
		Spec: v1alpha1.AddonsConfigurationSpec{
			CommonAddonsConfigurationSpec: v1alpha1.CommonAddonsConfigurationSpec{},
		},
		Status: v1alpha1.AddonsConfigurationStatus{
			CommonAddonsConfigurationStatus: v1alpha1.CommonAddonsConfigurationStatus{
				Phase: v1alpha1.AddonsConfigurationReady,
			},
		},
	}
}

func fixServiceBroker() *v1beta1.ServiceBroker {
	return &v1beta1.ServiceBroker{
		ObjectMeta: metav1.ObjectMeta{
			Name:      broker.NamespacedBrokerName,
			Namespace: "default",
		},
		Spec: v1beta1.ServiceBrokerSpec{},
	}
}

func fixServiceInstance() *v1beta1.ServiceInstance {
	return &v1beta1.ServiceInstance{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "si",
			Namespace: "default",
		},
		Spec: v1beta1.ServiceInstanceSpec{
			ServiceClassRef: &v1beta1.LocalObjectReference{
				Name: "sc-service",
			},
		}}
}

func fixServiceClass() *v1beta1.ServiceClass {
	return &v1beta1.ServiceClass{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "sc-service",
			Namespace: "default",
		},
		Spec: v1beta1.ServiceClassSpec{
			ServiceBrokerName: broker.NamespacedBrokerName,
		}}
}

func prepareBrokerController(t *testing.T, initObjs ...runtime.Object) (*controller.BrokerController, client.Client) {
	require.NoError(t, v1beta1.AddToScheme(scheme.Scheme))
	require.NoError(t, v1alpha1.AddToScheme(scheme.Scheme))
	cli := fake.NewFakeClientWithScheme(scheme.Scheme, initObjs...)
	iChecker := instance.New(cli, broker.NamespacedBrokerName)
	bFacade := broker.NewBrokersFacade(cli, "default", broker.NamespacedBrokerName, spy.NewLogDummy())
	svc := controller.NewBrokerController(iChecker, cli, bFacade)

	return svc, cli
}
