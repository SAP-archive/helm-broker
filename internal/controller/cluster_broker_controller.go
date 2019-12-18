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

// ClusterBrokerController is a controller which reacts on service instance deletion and delete (cluster)service broker if necessary
type ClusterBrokerController struct {
	instanceChecker instanceChecker
	cli             client.Client

	clusterBrokerFacade brokerFacade
}

// Start starts the controller
func (sbc *ClusterBrokerController) Start(mgr manager.Manager) error {
	// Create a new controller
	c, err := controller.New("cluster-broker-controller", mgr, controller.Options{Reconciler: sbc})
	if err != nil {
		return err
	}

	// Watch for changes to ServiceInstance, ClusterAddonsConfiguration, ClusterServiceBroker
	err = c.Watch(&source.Kind{Type: &v1beta1.ServiceInstance{}}, &handler.EnqueueRequestForObject{})
	if err != nil {
		return err
	}

	err = c.Watch(&source.Kind{Type: &v1alpha1.ClusterAddonsConfiguration{}}, &handler.EnqueueRequestForObject{}, predicate.Funcs{
		CreateFunc: func(_ event.CreateEvent) bool { return true },
		DeleteFunc: func(_ event.DeleteEvent) bool { return true },
		UpdateFunc: func(_ event.UpdateEvent) bool { return false },
	})
	if err != nil {
		return err
	}

	err = c.Watch(&source.Kind{Type: &v1beta1.ClusterServiceBroker{}}, &handler.EnqueueRequestForObject{}, predicate.Funcs{
		CreateFunc: func(_ event.CreateEvent) bool { return true },
		DeleteFunc: func(_ event.DeleteEvent) bool { return true },
		UpdateFunc: func(_ event.UpdateEvent) bool { return false },
	})
	if err != nil {
		return err
	}

	return nil
}

// Reconcile checks if the cluster service broker must be removed
func (sbc *ClusterBrokerController) Reconcile(request reconcile.Request) (reconcile.Result, error) {
	currentNamespace := request.Namespace

	csbExists, err := sbc.clusterBrokerFacade.Exist()
	if err != nil {
		return reconcile.Result{}, err
	}
	cacList := v1alpha1.ClusterAddonsConfigurationList{}
	err = sbc.cli.List(context.TODO(), &client.ListOptions{}, &cacList)
	if err != nil {
		return reconcile.Result{}, err
	}
	anyClusterAddonsConfigExists := len(cacList.Items) > 0
	instanceExists, err := sbc.instanceChecker.AnyServiceInstanceExistsForClusterServiceBroker(currentNamespace)
	if err != nil {
		return reconcile.Result{}, err
	}

	if csbExists && (!anyClusterAddonsConfigExists && !instanceExists) {
		sbc.clusterBrokerFacade.Delete()
	}
	if !csbExists && (anyClusterAddonsConfigExists || instanceExists) {
		sbc.clusterBrokerFacade.Create()

		// todo: Sync()
	}

	return reconcile.Result{}, nil
}
