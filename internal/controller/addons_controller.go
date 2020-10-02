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

var _ reconcile.Reconciler = &ReconcileAddonsConfiguration{}

// ReconcileAddonsConfiguration reconciles a AddonsConfiguration object
type ReconcileAddonsConfiguration struct {
	log logrus.FieldLogger
	client.Client

	common commonReconciler
}

// NewReconcileAddonsConfiguration returns a new reconcile.Reconciler
func NewReconcileAddonsConfiguration(mgr manager.Manager, addonGetterFactory addonGetterFactory,
	chartStorage chartStorage, addonStorage addonStorage, brokerFacade brokerFacade, docsProvider docsProvider,
	brokerSyncer brokerSyncer, templateService templateService, tmpDir string, reprocessOnErrorDuration time.Duration, log logrus.FieldLogger) reconcile.Reconciler {
	return &ReconcileAddonsConfiguration{
		log:    log.WithField("controller", "addons"),
		Client: mgr.GetClient(),

		common: newControllerCommon(mgr.GetClient(), addonGetterFactory, addonStorage, chartStorage,
			docsProvider, brokerSyncer, brokerFacade, templateService, path.Join(tmpDir, "addon-loader-dst"), reprocessOnErrorDuration, log),
	}
}

// Reconcile reads that state of the cluster for a AddonsConfiguration object and makes changes based on the state read
func (r *ReconcileAddonsConfiguration) Reconcile(request reconcile.Request) (reconcile.Result, error) {
	addon := &v1alpha1.AddonsConfiguration{}
	err := r.Get(context.TODO(), request.NamespacedName, addon)
	if err != nil {
		return reconcile.Result{}, err
	}
	r.common.SetWorkingNamespace(addon.Namespace)
	commonAddon := &internal.CommonAddon{
		Meta:   addon.ObjectMeta,
		Spec:   addon.Spec.CommonAddonsConfigurationSpec,
		Status: addon.Status.CommonAddonsConfigurationStatus,
	}

	return r.common.Reconcile(commonAddon, fmt.Sprintf("AddonsConfiguration `%s/%s`", commonAddon.Meta.Name, commonAddon.Meta.Namespace))
}

// AddonsConfigurationController holds a controller logic
type AddonsConfigurationController struct {
	reconciler reconcile.Reconciler
}

// NewAddonsConfigurationController creates a controller with a given reconciler
func NewAddonsConfigurationController(reconciler reconcile.Reconciler) *AddonsConfigurationController {
	return &AddonsConfigurationController{reconciler: reconciler}
}

// Start starts a controller
func (acc *AddonsConfigurationController) Start(mgr manager.Manager) error {
	// Create a new controller
	c, err := controller.New("addons-controller", mgr, controller.Options{Reconciler: acc.reconciler})
	if err != nil {
		return err
	}

	// Watch for changes to AddonsConfiguration
	err = c.Watch(&source.Kind{Type: &v1alpha1.AddonsConfiguration{}}, &handler.EnqueueRequestForObject{})
	if err != nil {
		return err
	}

	return nil
}
