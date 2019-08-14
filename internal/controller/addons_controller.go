package controller

import (
	"context"
	"path"
	"time"

	"github.com/kyma-project/helm-broker/internal"
	"github.com/kyma-project/helm-broker/pkg/apis/addons/v1alpha1"
	"github.com/pkg/errors"
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

	*addonManager
}

// NewReconcileAddonsConfiguration returns a new reconcile.Reconciler
func NewReconcileAddonsConfiguration(mgr manager.Manager, addonGetterFactory addonGetterFactory, chartStorage chartStorage, addonStorage addonStorage, brokerFacade brokerFacade, docsProvider docsProvider, brokerSyncer brokerSyncer, tmpDir string, log logrus.FieldLogger) reconcile.Reconciler {
	return &ReconcileAddonsConfiguration{
		log:    log.WithField("controller", "addons"),
		Client: mgr.GetClient(),

		addonManager: newAddonManager(mgr.GetClient(), addonGetterFactory, addonStorage, chartStorage, docsProvider, brokerSyncer, brokerFacade, path.Join(tmpDir, "addon-loader-dst"), log),
	}
}

// Reconcile reads that state of the cluster for a AddonsConfiguration object and makes changes based on the state read
// and what is in the AddonsConfiguration.Spec
func (r *ReconcileAddonsConfiguration) Reconcile(request reconcile.Request) (reconcile.Result, error) {
	addon := &v1alpha1.AddonsConfiguration{}
	err := r.Get(context.TODO(), request.NamespacedName, addon)
	if err != nil {
		return reconcile.Result{}, err
	}
	r.SetWorkingNamespace(addon.Namespace)
	commonAddon := &internal.CommonAddon{
		Meta: addon.ObjectMeta,
		Spec: v1alpha1.CommonAddonsConfigurationSpec{
			ReprocessRequest: addon.Spec.ReprocessRequest,
			Repositories:     addon.Spec.Repositories,
		},
		Status: v1alpha1.CommonAddonsConfigurationStatus{
			Repositories:       addon.Status.Repositories,
			Phase:              addon.Status.Phase,
			ObservedGeneration: addon.Status.ObservedGeneration,
			LastProcessedTime:  addon.Status.LastProcessedTime,
		},
	}

	if addon.DeletionTimestamp != nil {
		r.log.Infof("Start delete AddonsConfiguration %s/%s process", addon.Name, addon.Namespace)

		if err := r.ReconcileOnDelete(commonAddon); err != nil {
			r.log.Errorf("while deleting AddonsConfiguration process: %v", err)
			return reconcile.Result{RequeueAfter: time.Second * 15}, errors.Wrapf(err, "while deleting AddonConfiguration %q", request.NamespacedName)
		}
		r.log.Info("Delete AddonsConfiguration process completed")
		return reconcile.Result{}, nil
	}

	if addon.Status.ObservedGeneration == 0 {
		r.log.Infof("Start add AddonsConfiguration %s/%s process", addon.Name, addon.Namespace)

		preAddon, err := r.PrepareForProcessing(commonAddon)
		if err != nil {
			return reconcile.Result{}, errors.Wrapf(err, "while preparing AddonsConfiguration %q for processing", request.NamespacedName)
		}
		if err = r.ReconcileOnAdd(preAddon, preAddon.Status); err != nil {
			r.log.Errorf("while adding AddonsConfiguration process: %v", err)
			return reconcile.Result{}, errors.Wrapf(err, "while creating AddonsConfiguration %q", request.NamespacedName)
		}
		r.log.Info("Add AddonsConfiguration process completed")

	} else if addon.Generation > addon.Status.ObservedGeneration {
		r.log.Infof("Start update AddonsConfiguration %s/%s process", addon.Name, addon.Namespace)

		lastStatus := commonAddon.Status
		commonAddon.Status = v1alpha1.CommonAddonsConfigurationStatus{}

		if err = r.ReconcileOnAdd(commonAddon, lastStatus); err != nil {
			r.log.Errorf("while updating AddonsConfiguration process: %v", err)
			return reconcile.Result{}, errors.Wrapf(err, "while updating AddonsConfiguration %q", request.NamespacedName)
		}
		r.log.Info("Update AddonsConfiguration process completed")
	}

	return reconcile.Result{}, nil
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
