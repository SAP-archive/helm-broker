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

var _ reconcile.Reconciler = &ReconcileClusterAddonsConfiguration{}

// ReconcileClusterAddonsConfiguration reconciles a ClusterAddonsConfiguration object
type ReconcileClusterAddonsConfiguration struct {
	log logrus.FieldLogger
	client.Client

	*addonManager
}

// NewReconcileClusterAddonsConfiguration returns a new reconcile.Reconciler
func NewReconcileClusterAddonsConfiguration(mgr manager.Manager, addonGetterFactory addonGetterFactory, chartStorage chartStorage, addonStorage addonStorage, brokerFacade brokerFacade, docsProvider docsProvider, brokerSyncer brokerSyncer, tmpDir string, log logrus.FieldLogger) reconcile.Reconciler {
	return &ReconcileClusterAddonsConfiguration{
		log:    log.WithField("controller", "cluster-addons"),
		Client: mgr.GetClient(),

		addonManager: newAddonManager(mgr.GetClient(), addonGetterFactory, addonStorage, chartStorage, docsProvider, brokerSyncer, brokerFacade, path.Join(tmpDir, "cluster-addon-loader-dst"), log),
	}
}

// Reconcile reads that state of the cluster for a ClusterAddonsConfiguration object and makes changes based on the state read
// and what is in the ClusterAddonsConfiguration.Spec
func (r *ReconcileClusterAddonsConfiguration) Reconcile(request reconcile.Request) (reconcile.Result, error) {
	addon := &v1alpha1.ClusterAddonsConfiguration{}
	err := r.Get(context.TODO(), request.NamespacedName, addon)
	if err != nil {
		return reconcile.Result{}, err
	}
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
		r.log.Infof("Start delete ClusterAddonsConfiguration %s", addon.Name)

		if err := r.ReconcileOnDelete(commonAddon); err != nil {
			r.log.Errorf("while deleting ClusterAddonsConfiguration process: %v", err)
			return reconcile.Result{RequeueAfter: time.Second * 15}, errors.Wrapf(err, "while deleting ClusterAddonConfiguration %q", request.NamespacedName)
		}
		r.log.Info("Delete ClusterAddonsConfiguration process completed")
		return reconcile.Result{}, nil
	}

	if addon.Status.ObservedGeneration == 0 {
		r.log.Infof("Start add ClusterAddonsConfiguration %s process", addon.Name)

		preAddon, err := r.PrepareForProcessing(commonAddon)
		if err != nil {
			return reconcile.Result{}, errors.Wrapf(err, "while preparing AddonsConfiguration %q for processing", request.NamespacedName)
		}
		r.log.Infof("%v VS %s", commonAddon, preAddon)
		if err = r.ReconcileOnAdd(preAddon, preAddon.Status); err != nil {
			r.log.Errorf("while adding ClusterAddonsConfiguration process: %v", err)
			return reconcile.Result{}, errors.Wrapf(err, "while creating ClusterAddonsConfiguration %q", request.NamespacedName)
		}
		r.log.Infof("Add ClusterAddonsConfiguration process completed")

	} else if addon.Generation > addon.Status.ObservedGeneration {
		r.log.Infof("Start update ClusterAddonsConfiguration %s process", addon.Name)

		lastStatus := commonAddon.Status
		commonAddon.Status = v1alpha1.CommonAddonsConfigurationStatus{}

		if err = r.ReconcileOnAdd(commonAddon, lastStatus); err != nil {
			r.log.Errorf("while updating ClusterAddonsConfiguration process: %v", err)
			return reconcile.Result{}, errors.Wrapf(err, "while updating ClusterAddonsConfiguration %q", request.NamespacedName)
		}
		r.log.Infof("Update ClusterAddonsConfiguration %s process completed", addon.Name)
	}

	return reconcile.Result{}, nil
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
