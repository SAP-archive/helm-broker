package controller

import (
	"context"
	"path"
	"time"

	"github.com/kyma-project/helm-broker/internal"
	"github.com/kyma-project/helm-broker/internal/storage"
	addonsv1alpha1 "github.com/kyma-project/helm-broker/pkg/apis/addons/v1alpha1"
	exerr "github.com/pkg/errors"
	"github.com/sirupsen/logrus"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller"
	"sigs.k8s.io/controller-runtime/pkg/handler"
	"sigs.k8s.io/controller-runtime/pkg/manager"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
	"sigs.k8s.io/controller-runtime/pkg/source"
)

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
	c, err := controller.New("clusteraddonsconfiguration-controller", mgr, controller.Options{Reconciler: cacc.reconciler})
	if err != nil {
		return err
	}

	// Watch for changes to ClusterAddonsConfiguration
	err = c.Watch(&source.Kind{Type: &addonsv1alpha1.ClusterAddonsConfiguration{}}, &handler.EnqueueRequestForObject{})
	if err != nil {
		return err
	}

	return nil
}

var _ reconcile.Reconciler = &ReconcileClusterAddonsConfiguration{}

// ReconcileClusterAddonsConfiguration reconciles a ClusterAddonsConfiguration object
type ReconcileClusterAddonsConfiguration struct {
	log logrus.FieldLogger
	client.Client
	scheme *runtime.Scheme

	brokerFacade clusterBrokerFacade
	brokerSyncer clusterBrokerSyncer

	*addonManager
}

// NewReconcileClusterAddonsConfiguration returns a new reconcile.Reconciler
func NewReconcileClusterAddonsConfiguration(mgr manager.Manager, addonGetterFactory addonGetterFactory, chartStorage chartStorage, addonStorage addonStorage, brokerFacade clusterBrokerFacade, docsProvider docsProvider, brokerSyncer clusterBrokerSyncer, tmpDir string, log logrus.FieldLogger) reconcile.Reconciler {
	return &ReconcileClusterAddonsConfiguration{
		log:    log.WithField("controller", "cluster-addons-configuration"),
		Client: mgr.GetClient(),
		scheme: mgr.GetScheme(),

		brokerFacade: brokerFacade,
		brokerSyncer: brokerSyncer,
		addonManager: newAddonManager(addonGetterFactory, addonStorage, chartStorage, docsProvider, path.Join(tmpDir, "cluster-addon-loader-dst"), log),
	}
}

// Reconcile reads that state of the cluster for a ClusterAddonsConfiguration object and makes changes based on the state read
// and what is in the ClusterAddonsConfiguration.Spec
func (r *ReconcileClusterAddonsConfiguration) Reconcile(request reconcile.Request) (reconcile.Result, error) {
	addon := &addonsv1alpha1.ClusterAddonsConfiguration{}
	err := r.Get(context.TODO(), request.NamespacedName, addon)
	if err != nil {
		return reconcile.Result{}, err
	}

	if addon.DeletionTimestamp != nil {
		if err := r.deleteAddonsProcess(addon); err != nil {
			r.log.Errorf("while deleting ClusterAddonsConfiguration process: %v", err)
			return reconcile.Result{RequeueAfter: time.Second * 15}, exerr.Wrapf(err, "while deleting ClusterAddonConfiguration %q", request.NamespacedName)
		}
		return reconcile.Result{}, nil
	}

	if addon.Status.ObservedGeneration == 0 {
		r.log.Infof("Start add ClusterAddonsConfiguration %s process", addon.Name)

		preAddon, err := r.prepareForProcessing(addon)
		if err != nil {
			r.log.Errorf("while preparing for processing: %v", err)
			return reconcile.Result{Requeue: true}, exerr.Wrapf(err, "while adding a finalizer to AddonsConfiguration %q", request.NamespacedName)
		}
		err = r.addAddonsProcess(preAddon, preAddon.Status)
		if err != nil {
			r.log.Errorf("while adding ClusterAddonsConfiguration process: %v", err)
			return reconcile.Result{}, exerr.Wrapf(err, "while creating ClusterAddonsConfiguration %q", request.NamespacedName)
		}
		r.log.Infof("Add ClusterAddonsConfiguration process completed")

	} else if addon.Generation > addon.Status.ObservedGeneration {
		r.log.Infof("Start update ClusterAddonsConfiguration %s process", addon.Name)

		lastAddon := addon.DeepCopy()
		addon.Status = addonsv1alpha1.ClusterAddonsConfigurationStatus{}
		err = r.addAddonsProcess(addon, lastAddon.Status)
		if err != nil {
			r.log.Errorf("while updating ClusterAddonsConfiguration process: %v", err)
			return reconcile.Result{}, exerr.Wrapf(err, "while updating ClusterAddonsConfiguration %q", request.NamespacedName)
		}
		r.log.Infof("Update ClusterAddonsConfiguration %s process completed", addon.Name)
	}

	return reconcile.Result{}, nil
}

func (r *ReconcileClusterAddonsConfiguration) addAddonsProcess(addon *addonsv1alpha1.ClusterAddonsConfiguration, lastStatus addonsv1alpha1.ClusterAddonsConfigurationStatus) error {
	r.log.Infof("- load addons and charts for each addon")
	repositories := r.Load(addon.Spec.Repositories)

	r.log.Info("- check duplicate ID addons alongside repositories")
	repositories.ReviseAddonDuplicationInRepository()

	r.log.Info("- check duplicates ID addons in existing ClusterAddonsConfigurations")
	list, err := r.existingAddonsConfigurations(addon.Name)
	if err != nil {
		return exerr.Wrap(err, "while fetching ClusterAddonsConfigurations list")
	}
	repositories.ReviseAddonDuplicationInClusterStorage(list)

	if repositories.IsRepositoriesFailed() {
		addon.Status.Phase = addonsv1alpha1.AddonsConfigurationFailed
	} else {
		addon.Status.Phase = addonsv1alpha1.AddonsConfigurationReady
	}
	r.log.Infof("- status: %s", addon.Status.Phase)

	var deletedAddons []string
	saved := false

	switch addon.Status.Phase {
	case addonsv1alpha1.AddonsConfigurationFailed:
		r.statusSnapshot(&addon.Status.CommonAddonsConfigurationStatus, repositories)
		if _, err = r.updateAddonStatus(addon); err != nil {
			return exerr.Wrap(err, "while update ClusterAddonsConfiguration status")
		}
		if lastStatus.Phase == addonsv1alpha1.AddonsConfigurationReady {
			deletedAddons, err = r.deletePreviousAddons(string(internal.ClusterWide), lastStatus.Repositories)
			if err != nil {
				return exerr.Wrap(err, "while deleting addons from repository")
			}

		}
	case addonsv1alpha1.AddonsConfigurationReady:
		r.log.Info("- save ready addons and charts in storage")
		saved = r.saveAddons(string(internal.ClusterWide), repositories)

		r.statusSnapshot(&addon.Status.CommonAddonsConfigurationStatus, repositories)
		if _, err = r.updateAddonStatus(addon); err != nil {
			return exerr.Wrap(err, "while update ClusterAddonsConfiguration status")
		}
		if lastStatus.Phase == addonsv1alpha1.AddonsConfigurationReady {
			deletedAddons, err = r.deleteOrphanAddons(string(internal.ClusterWide), addon.Status.Repositories, lastStatus.Repositories)
			if err != nil {
				return exerr.Wrap(err, "while deleting orphan addons from storage")
			}
		}
	}
	if saved || len(deletedAddons) > 0 {
		r.log.Info("- ensure ClusterServiceBroker")
		if err = r.ensureBroker(addon); err != nil {
			return exerr.Wrap(err, "while ensuring ClusterServiceBroker")
		}
	}

	if len(deletedAddons) > 0 {
		r.log.Info("- reprocessing conflicting addons configurations")
		for _, key := range deletedAddons {
			for _, existingAddon := range list.Items {
				if hasConflict := r.isConfigurationConflicting(key, existingAddon.Status.CommonAddonsConfigurationStatus); hasConflict {
					if err := r.reprocessAddonsConfiguration(&existingAddon); err != nil {
						return exerr.Wrapf(err, "while reprocessing addon %s", existingAddon.Name)
					}
				}
			}
		}
	}

	return nil
}

func (r *ReconcileClusterAddonsConfiguration) deleteAddonsProcess(addon *addonsv1alpha1.ClusterAddonsConfiguration) error {
	r.log.Infof("Start delete ClusterAddonsConfiguration %s", addon.Name)

	if addon.Status.Phase == addonsv1alpha1.AddonsConfigurationReady {
		adds, err := r.existingAddonsConfigurations(addon.Name)
		if err != nil {
			return exerr.Wrap(err, "while listing ClusterAddonsConfigurations")
		}

		deleteBroker := true
		for _, addon := range adds.Items {
			if addon.Status.Phase != addonsv1alpha1.AddonsConfigurationReady {
				// reprocess ClusterAddonsConfiguration again if was failed
				if err := r.reprocessAddonsConfiguration(&addon); err != nil {
					return exerr.Wrapf(err, "while requesting reprocess for ClusterAddonsConfiguration %s", addon.Name)
				}
			} else {
				deleteBroker = false
			}
		}
		if deleteBroker {
			r.log.Info("- delete ClusterServiceBroker")
			if err := r.brokerFacade.Delete(); err != nil {
				return exerr.Wrap(err, "while deleting ClusterServiceBroker")
			}
		}

		addonRemoved := false
		for _, repo := range addon.Status.Repositories {
			for _, a := range repo.Addons {
				addonRemoved, err = r.removeAddon(a, internal.ClusterWide)
				if err != nil && !storage.IsNotFoundError(err) {
					return exerr.Wrapf(err, "while deleting addon with charts for addon %s", a.Name)
				}
			}
		}
		if !deleteBroker && addonRemoved {
			if err := r.brokerSyncer.Sync(); err != nil {
				return exerr.Wrapf(err, "while syncing ClusterServiceBroker for addon %s", addon.Name)
			}
		}
	}
	if err := r.deleteFinalizer(addon); err != nil {
		return exerr.Wrapf(err, "while deleting finalizer from ClusterAddonsConfiguration %s", addon.Name)
	}

	r.log.Info("Delete ClusterAddonsConfiguration process completed")
	return nil
}

func (r *ReconcileClusterAddonsConfiguration) ensureBroker(addon *addonsv1alpha1.ClusterAddonsConfiguration) error {
	exist, err := r.brokerFacade.Exist()
	if err != nil {
		return exerr.Wrap(err, "while checking if ClusterServiceBroker exists")
	}
	if !exist {
		r.log.Info("- creating ClusterServiceBroker")
		if err := r.brokerFacade.Create(); err != nil {
			return exerr.Wrapf(err, "while creating ClusterServiceBroker for addon %s", addon.Name)
		}
	} else {
		if err := r.brokerSyncer.Sync(); err != nil {
			return exerr.Wrapf(err, "while syncing ClusterServiceBroker for addon %s", addon.Name)
		}
	}
	return nil
}

func (r *ReconcileClusterAddonsConfiguration) existingAddonsConfigurations(addonName string) (*addonsv1alpha1.ClusterAddonsConfigurationList, error) {
	addonsList := &addonsv1alpha1.ClusterAddonsConfigurationList{}
	addonsConfigurationList, err := r.addonsConfigurationList()
	if err != nil {
		return nil, exerr.Wrap(err, "while listing ClusterAddonsConfigurations")
	}

	for _, existAddon := range addonsConfigurationList.Items {
		if existAddon.Name != addonName {
			addonsList.Items = append(addonsList.Items, existAddon)
		}
	}

	return addonsList, nil
}

func (r *ReconcileClusterAddonsConfiguration) reprocessAddonsConfiguration(addon *addonsv1alpha1.ClusterAddonsConfiguration) error {
	ad := &addonsv1alpha1.ClusterAddonsConfiguration{}
	if err := r.Client.Get(context.Background(), types.NamespacedName{Name: addon.Name}, ad); err != nil {
		return exerr.Wrapf(err, "while getting ClusterAddonsConfiguration %s", addon.Name)
	}
	ad.Spec.ReprocessRequest++
	if err := r.Client.Update(context.Background(), ad); err != nil {
		return exerr.Wrapf(err, "while incrementing a reprocess requests for ClusterAddonsConfiguration %s", addon.Name)
	}
	return nil
}

func (r *ReconcileClusterAddonsConfiguration) updateAddonStatus(addon *addonsv1alpha1.ClusterAddonsConfiguration) (*addonsv1alpha1.ClusterAddonsConfiguration, error) {
	addon.Status.ObservedGeneration = addon.Generation
	addon.Status.LastProcessedTime = &v1.Time{Time: time.Now()}

	r.log.Infof("- update ClusterAddonsConfiguration %s status", addon.Name)
	err := r.Status().Update(context.TODO(), addon)
	if err != nil {
		return nil, exerr.Wrap(err, "while update ClusterAddonsConfiguration")
	}
	return addon, nil
}

func (r *ReconcileClusterAddonsConfiguration) prepareForProcessing(addon *addonsv1alpha1.ClusterAddonsConfiguration) (*addonsv1alpha1.ClusterAddonsConfiguration, error) {
	obj := addon.DeepCopy()
	obj.Status.Phase = addonsv1alpha1.AddonsConfigurationPending

	pendingInstance, err := r.updateAddonStatus(obj)
	if err != nil {
		return nil, exerr.Wrap(err, "while updating addons status")
	}

	if r.protection.hasFinalizer(pendingInstance.Finalizers) {
		return pendingInstance, nil
	}
	r.log.Info("- add a finalizer")
	pendingInstance.Finalizers = r.protection.addFinalizer(pendingInstance.Finalizers)

	err = r.Client.Update(context.Background(), pendingInstance)
	if err != nil {
		return nil, exerr.Wrap(err, "while updating addons status")
	}
	return pendingInstance, nil
}

func (r *ReconcileClusterAddonsConfiguration) deleteFinalizer(addon *addonsv1alpha1.ClusterAddonsConfiguration) error {
	obj := addon.DeepCopy()
	if !r.protection.hasFinalizer(obj.Finalizers) {
		return nil
	}
	r.log.Info("- delete a finalizer")
	obj.Finalizers = r.protection.removeFinalizer(obj.Finalizers)

	return r.Client.Update(context.Background(), obj)
}

func (r *ReconcileClusterAddonsConfiguration) addonsConfigurationList() (*addonsv1alpha1.ClusterAddonsConfigurationList, error) {
	addonsConfigurationList := &addonsv1alpha1.ClusterAddonsConfigurationList{}

	err := r.Client.List(context.TODO(), &client.ListOptions{}, addonsConfigurationList)
	if err != nil {
		return addonsConfigurationList, exerr.Wrap(err, "during fetching ClusterAddonConfiguration list by client")
	}

	return addonsConfigurationList, nil
}
