package controller

import (
	"context"

	"github.com/kubernetes-sigs/service-catalog/pkg/apis/servicecatalog/v1beta1"
	"github.com/kyma-project/helm-broker/internal/controller/broker"
	"github.com/kyma-project/helm-broker/pkg/apis/addons/v1alpha1"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller"
	"sigs.k8s.io/controller-runtime/pkg/event"
	"sigs.k8s.io/controller-runtime/pkg/handler"
	"sigs.k8s.io/controller-runtime/pkg/manager"
	"sigs.k8s.io/controller-runtime/pkg/predicate"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
	"sigs.k8s.io/controller-runtime/pkg/source"
)

// BrokerController is a controller which reacts on changes for ServiceInstance, ServiceBroker and AddonsConfiguration.
// Only this controller should create/delete ServiceBroker.
type BrokerController struct {
	instanceChecker instanceChecker
	cli             client.Client

	namespacedBrokerFacade brokerFacade
}

var createDeletePredicate = predicate.Funcs{
	CreateFunc: func(_ event.CreateEvent) bool { return true },
	DeleteFunc: func(_ event.DeleteEvent) bool { return true },
	UpdateFunc: func(_ event.UpdateEvent) bool { return false },
}

var eventHandler = &handler.EnqueueRequestsFromMapFunc{
	ToRequests: handler.ToRequestsFunc(
		func(mp handler.MapObject) []reconcile.Request {
			return []reconcile.Request{{NamespacedName: types.NamespacedName{Namespace: mp.Meta.GetNamespace()}}}
		},
	)}

// NewBrokerController creates BrokerController instance.
func NewBrokerController(checker instanceChecker, cli client.Client, bFacade brokerFacade) *BrokerController {
	return &BrokerController{
		instanceChecker:        checker,
		cli:                    cli,
		namespacedBrokerFacade: bFacade,
	}
}

// Start starts the controller
func (sbc *BrokerController) Start(mgr manager.Manager) error {
	// Create a new controller
	c, err := controller.New("broker-controller", mgr, controller.Options{Reconciler: sbc})
	if err != nil {
		return err
	}

	// Watch for changes to ServiceInstance
	err = c.Watch(&source.Kind{Type: &v1beta1.ServiceInstance{}}, eventHandler, createDeletePredicate)
	if err != nil {
		return err
	}

	err = c.Watch(&source.Kind{Type: &v1alpha1.AddonsConfiguration{}}, eventHandler, createDeletePredicate)
	if err != nil {
		return err
	}

	err = c.Watch(&source.Kind{Type: &v1beta1.ServiceBroker{}}, eventHandler, predicate.Funcs{
		CreateFunc: func(e event.CreateEvent) bool { return e.Meta.GetName() == broker.NamespacedBrokerName },
		DeleteFunc: func(e event.DeleteEvent) bool { return e.Meta.GetName() == broker.NamespacedBrokerName },
		UpdateFunc: func(_ event.UpdateEvent) bool { return false },
	})
	if err != nil {
		return err
	}

	return nil
}

// Reconcile checks if the (cluster) service broker must be removed
func (sbc *BrokerController) Reconcile(request reconcile.Request) (reconcile.Result, error) {
	currentNamespace := request.Namespace
	sbc.namespacedBrokerFacade.SetNamespace(request.Namespace)

	sbExists, err := sbc.namespacedBrokerFacade.Exist()
	if err != nil {
		return reconcile.Result{}, err
	}
	acList := v1alpha1.AddonsConfigurationList{}
	err = sbc.cli.List(context.TODO(), &acList, client.InNamespace(currentNamespace))
	if err != nil {
		return reconcile.Result{}, err
	}

	configurationsExist := len(acList.Items) > 0
	instancesExist, err := sbc.instanceChecker.AnyServiceInstanceExistsForNamespacedServiceBroker(currentNamespace)
	if err != nil {
		return reconcile.Result{}, err
	}

	if sbExists && !configurationsExist && !instancesExist {
		if err := sbc.namespacedBrokerFacade.Delete(); err != nil {
			return reconcile.Result{}, err
		}
	}
	if !sbExists && (configurationsExist || instancesExist) {
		if err := sbc.namespacedBrokerFacade.Create(); err != nil {
			return reconcile.Result{}, err
		}
	}
	return reconcile.Result{}, nil
}
