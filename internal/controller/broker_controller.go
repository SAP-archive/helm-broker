package controller

import (
	"context"

	"github.com/kubernetes-incubator/service-catalog/pkg/apis/servicecatalog/v1beta1"
	"github.com/kyma-project/helm-broker/pkg/apis/addons/v1alpha1"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller"
	"sigs.k8s.io/controller-runtime/pkg/event"
	"sigs.k8s.io/controller-runtime/pkg/handler"
	"sigs.k8s.io/controller-runtime/pkg/manager"
	"sigs.k8s.io/controller-runtime/pkg/predicate"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
	"sigs.k8s.io/controller-runtime/pkg/source"
)

// BrokerController is a controller which reacts on service instance deletion and delete (cluster)service broker if necessary
type BrokerController struct {
	instanceChecker instanceChecker
	cli             client.Client

	clusterBrokerFacade    brokerFacade
	namespacedBrokerFacade brokerFacade
}

// Start starts the controller
func (sbc *BrokerController) Start(mgr manager.Manager) error {
	// Create a new controller
	c, err := controller.New("broker-controller", mgr, controller.Options{Reconciler: sbc})
	if err != nil {
		return err
	}

	// Watch for changes to ServiceInstance
	err = c.Watch(&source.Kind{Type: &v1beta1.ServiceInstance{}}, &handler.EnqueueRequestForObject{})
	if err != nil {
		return err
	}

	err = c.Watch(&source.Kind{Type: &v1alpha1.AddonsConfiguration{}}, &handler.EnqueueRequestForObject{})
	if err != nil {
		return err
	}

	err = c.Watch(&source.Kind{Type: &v1beta1.ServiceBroker{}}, &handler.EnqueueRequestForObject{}, predicate.Funcs{
		CreateFunc: func(_ event.CreateEvent) bool { return true },
		DeleteFunc: func(_ event.DeleteEvent) bool { return true },
		UpdateFunc: func(_ event.UpdateEvent) bool { return false },
	})
	if err != nil {
		return err
	}

	return nil
}

// Reconcile checks if the (clsuter) service broker must be removed
func (sbc *BrokerController) Reconcile(request reconcile.Request) (reconcile.Result, error) {
	currentNamespace := request.Namespace

	// namespaced check
	sbc.namespacedBrokerFacade.SetNamespace(request.Namespace)
	sbExists, err := sbc.namespacedBrokerFacade.Exist()
	if err != nil {
		return reconcile.Result{}, err
	}
	// list addons configurations
	acList := v1alpha1.AddonsConfigurationList{}
	err = sbc.cli.List(context.TODO(), &client.ListOptions{Namespace: currentNamespace}, &acList)
	if err != nil {
		return reconcile.Result{}, err
	}
	anyNamespacedConfigExists := len(acList.Items) > 0

	instanceByNamespacedExists, err := sbc.instanceChecker.AnyServiceInstanceExistsForNamespacedServiceBroker(currentNamespace)
	if err != nil {
		return reconcile.Result{}, err
	}
	if sbExists && !anyNamespacedConfigExists && !instanceByNamespacedExists {
		sbc.namespacedBrokerFacade.Delete()
	}
	if !sbExists && (anyNamespacedConfigExists || instanceByNamespacedExists) {
		sbc.namespacedBrokerFacade.Create()
	}

	return reconcile.Result{}, nil
}
