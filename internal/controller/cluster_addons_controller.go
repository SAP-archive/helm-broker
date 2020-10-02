package controller

import (
	"context"
	"fmt"
	"path"
	"time"

	"github.com/kyma-project/helm-broker/internal"
	"github.com/kyma-project/helm-broker/pkg/apis/addons/v1alpha1"
	"github.com/sirupsen/logrus"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller"
	"sigs.k8s.io/controller-runtime/pkg/handler"
	"sigs.k8s.io/controller-runtime/pkg/manager"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
	"sigs.k8s.io/controller-runtime/pkg/source"
)

var _ reconcile.Reconciler = &ReconcileClusterAddonsConfiguration{}

// ReconcileClusterAddonsConfiguration reconciles a ClusterAddonsConfiguration object
type ReconcileClusterAddonsConfiguration struct {
	log logrus.FieldLogger
	client.Client

	common commonReconciler
}

// NewReconcileClusterAddonsConfiguration returns a new reconcile.Reconciler
func NewReconcileClusterAddonsConfiguration(mgr manager.Manager, addonGetterFactory addonGetterFactory, chartStorage chartStorage,
	addonStorage addonStorage, brokerFacade brokerFacade, docsProvider docsProvider, brokerSyncer brokerSyncer,
	templateService templateService, tmpDir string, reprocessOnErrorDuration time.Duration, log logrus.FieldLogger) reconcile.Reconciler {
	return &ReconcileClusterAddonsConfiguration{
		log:    log.WithField("controller", "cluster-addons"),
		Client: mgr.GetClient(),

		common: newControllerCommon(mgr.GetClient(), addonGetterFactory, addonStorage, chartStorage, docsProvider, brokerSyncer, brokerFacade, templateService, path.Join(tmpDir, "cluster-addon-loader-dst"), reprocessOnErrorDuration, log),
	}
}

// Reconcile reads that state of the cluster for a ClusterAddonsConfiguration object and makes changes based on the state read
func (r *ReconcileClusterAddonsConfiguration) Reconcile(request reconcile.Request) (reconcile.Result, error) {
	addon := &v1alpha1.ClusterAddonsConfiguration{}
	err := r.Get(context.TODO(), request.NamespacedName, addon)
	if err != nil {
		return reconcile.Result{}, err
	}
	commonAddon := &internal.CommonAddon{
		Meta:   addon.ObjectMeta,
		Spec:   addon.Spec.CommonAddonsConfigurationSpec,
		Status: addon.Status.CommonAddonsConfigurationStatus,
	}

	return r.common.Reconcile(commonAddon, fmt.Sprintf("ClusterAddonsConfiguration `%s`", commonAddon.Meta.Name))
}

// ClusterAddonsConfigurationController holds controller logic
type ClusterAddonsConfigurationController struct {
	reconciler reconcile.Reconciler
}

// NewClusterAddonsConfigurationController creates new controller with a given reconciler
func NewClusterAddonsConfigurationController(reconciler reconcile.Reconciler) *ClusterAddonsConfigurationController {
	return &ClusterAddonsConfigurationController{reconciler: reconciler}
}

// Start starts a controller
func (cacc *ClusterAddonsConfigurationController) Start(mgr manager.Manager) error {
	// Create a new controller
	c, err := controller.New("cluster-addons-controller", mgr, controller.Options{Reconciler: cacc.reconciler})
	if err != nil {
		return err
	}

	// Watch for changes to ClusterAddonsConfiguration
	err = c.Watch(&source.Kind{Type: &v1alpha1.ClusterAddonsConfiguration{}}, &handler.EnqueueRequestForObject{})
	if err != nil {
		return err
	}

	return nil
}
