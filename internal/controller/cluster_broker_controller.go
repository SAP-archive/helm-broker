package controller

import (
	"context"

	"github.com/kubernetes-sigs/service-catalog/pkg/apis/servicecatalog/v1beta1"
	"github.com/kyma-project/helm-broker/pkg/apis/addons/v1alpha1"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller"
	"sigs.k8s.io/controller-runtime/pkg/event"
	"sigs.k8s.io/controller-runtime/pkg/manager"
	"sigs.k8s.io/controller-runtime/pkg/predicate"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
	"sigs.k8s.io/controller-runtime/pkg/source"
)

// ClusterBrokerController is a controller which reacts on changes for ServiceInstance, ClusterServiceBroker and ClusterAddonsConfiguration.
// Only this controller should create/delete ClusterServiceBroker.
type ClusterBrokerController struct {
	instanceChecker instanceChecker
	cli             client.Client

	clusterBrokerFacade brokerFacade
	clusterBrokerName   string
}

// NewClusterBrokerController creates ClusterBrokerController instance.
func NewClusterBrokerController(checker instanceChecker, cli client.Client, bFacade brokerFacade, clusterBrokerName string) *ClusterBrokerController {
	return &ClusterBrokerController{
		instanceChecker:     checker,
		cli:                 cli,
		clusterBrokerFacade: bFacade,
		clusterBrokerName:   clusterBrokerName,
	}
}

// Start starts the controller
func (sbc *ClusterBrokerController) Start(mgr manager.Manager) error {
	// Create a new controller
	c, err := controller.New("cluster-broker-controller", mgr, controller.Options{Reconciler: sbc})
	if err != nil {
		return err
	}

	// Watch for changes to ServiceInstance, ClusterAddonsConfiguration, ClusterServiceBroker
	err = c.Watch(&source.Kind{Type: &v1beta1.ServiceInstance{}}, eventHandler, createDeletePredicate)
	if err != nil {
		return err
	}

	err = c.Watch(&source.Kind{Type: &v1alpha1.ClusterAddonsConfiguration{}}, eventHandler, createDeletePredicate)
	if err != nil {
		return err
	}

	err = c.Watch(&source.Kind{Type: &v1beta1.ClusterServiceBroker{}}, eventHandler, predicate.Funcs{
		// filter out all other ClusterServiceBroker, only "helm-broker" is interesting for us
		CreateFunc: func(e event.CreateEvent) bool { return e.Meta.GetName() == sbc.clusterBrokerName },
		DeleteFunc: func(e event.DeleteEvent) bool { return e.Meta.GetName() == sbc.clusterBrokerName },
		UpdateFunc: func(_ event.UpdateEvent) bool { return false },
	})
	if err != nil {
		return err
	}

	return nil
}

// Reconcile checks if the cluster service broker must be removed
func (sbc *ClusterBrokerController) Reconcile(request reconcile.Request) (reconcile.Result, error) {
	csbExists, err := sbc.clusterBrokerFacade.Exist()
	if err != nil {
		return reconcile.Result{}, err
	}
	cacList := v1alpha1.ClusterAddonsConfigurationList{}
	err = sbc.cli.List(context.TODO(), &cacList)
	if err != nil {
		return reconcile.Result{}, err
	}

	configurationsExist := len(cacList.Items) > 0
	instancesExist, err := sbc.instanceChecker.AnyServiceInstanceExistsForClusterServiceBroker()
	if err != nil {
		return reconcile.Result{}, err
	}

	if csbExists && (!configurationsExist && !instancesExist) {
		if err = sbc.clusterBrokerFacade.Delete(); err != nil {
			return reconcile.Result{}, err
		}
	}
	if !csbExists && (configurationsExist || instancesExist) {
		if err = sbc.clusterBrokerFacade.Create(); err != nil {
			return reconcile.Result{}, err
		}
	}
	return reconcile.Result{}, nil
}
