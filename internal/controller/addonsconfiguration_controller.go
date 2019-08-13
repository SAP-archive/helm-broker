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
	c, err := controller.New("addonsconfiguration-controller", mgr, controller.Options{Reconciler: acc.reconciler})
	if err != nil {
		return err
	}

	// Watch for changes to AddonsConfiguration
	err = c.Watch(&source.Kind{Type: &addonsv1alpha1.AddonsConfiguration{}}, &handler.EnqueueRequestForObject{})
	if err != nil {
		return err
	}

	return nil
}

var _ reconcile.Reconciler = &ReconcileAddonsConfiguration{}

// ReconcileAddonsConfiguration reconciles a AddonsConfiguration object
type ReconcileAddonsConfiguration struct {
	log logrus.FieldLogger
	client.Client
	scheme *runtime.Scheme

	brokerFacade brokerFacade
	brokerSyncer brokerSyncer

	*addonManager
}

// NewReconcileAddonsConfiguration returns a new reconcile.Reconciler
func NewReconcileAddonsConfiguration(mgr manager.Manager, addonGetterFactory addonGetterFactory, chartStorage chartStorage, addonStorage addonStorage, brokerFacade brokerFacade, docsProvider docsProvider, brokerSyncer brokerSyncer, tmpDir string, log logrus.FieldLogger) reconcile.Reconciler {
	return &ReconcileAddonsConfiguration{
		log:    log.WithField("controller", "addons-configuration"),
		Client: mgr.GetClient(),
		scheme: mgr.GetScheme(),

		brokerSyncer: brokerSyncer,
		brokerFacade: brokerFacade,
		addonManager: newAddonManager(addonGetterFactory, addonStorage, chartStorage, docsProvider, path.Join(tmpDir, "addon-loader-dst"), log),
	}
}

// Reconcile reads that state of the cluster for a AddonsConfiguration object and makes changes based on the state read
// and what is in the AddonsConfiguration.Spec
func (r *ReconcileAddonsConfiguration) Reconcile(request reconcile.Request) (reconcile.Result, error) {
	addon := &addonsv1alpha1.AddonsConfiguration{}
	err := r.Get(context.TODO(), request.NamespacedName, addon)
	if err != nil {
		return reconcile.Result{}, err
	}

	if addon.DeletionTimestamp != nil {
		if err := r.deleteAddonsProcess(addon); err != nil {
			r.log.Errorf("while deleting AddonsConfiguration process: %v", err)
			return reconcile.Result{RequeueAfter: time.Second * 15}, exerr.Wrapf(err, "while deleting AddonConfiguration %q", request.NamespacedName)
		}
		return reconcile.Result{}, nil
	}

	if addon.Status.ObservedGeneration == 0 {
		r.log.Infof("Start add AddonsConfiguration %s/%s process", addon.Name, addon.Namespace)

		preAddon, err := r.prepareForProcessing(addon)
		if err != nil {
			r.log.Errorf("while preparing for processing: %v", err)
			return reconcile.Result{Requeue: true}, exerr.Wrapf(err, "while adding a finalizer to AddonsConfiguration %q", request.NamespacedName)
		}
		err = r.addAddonsProcess(preAddon, preAddon.Status)
		if err != nil {
			r.log.Errorf("while adding AddonsConfiguration process: %v", err)
			return reconcile.Result{}, exerr.Wrapf(err, "while creating AddonsConfiguration %q", request.NamespacedName)
		}
		r.log.Info("Add AddonsConfiguration process completed")

	} else if addon.Generation > addon.Status.ObservedGeneration {
		r.log.Infof("Start update AddonsConfiguration %s/%s process", addon.Name, addon.Namespace)

		lastAddon := addon.DeepCopy()
		addon.Status = addonsv1alpha1.AddonsConfigurationStatus{}
		err = r.addAddonsProcess(addon, lastAddon.Status)
		if err != nil {
			r.log.Errorf("while updating AddonsConfiguration process: %v", err)
			return reconcile.Result{}, exerr.Wrapf(err, "while updating AddonsConfiguration %q", request.NamespacedName)
		}
		r.log.Info("Update AddonsConfiguration process completed")
	}

	return reconcile.Result{}, nil
}

func (r *ReconcileAddonsConfiguration) addAddonsProcess(addon *addonsv1alpha1.AddonsConfiguration, lastStatus addonsv1alpha1.AddonsConfigurationStatus) error {
	r.log.Infof("- load addons and charts for each addon")
	repositories := r.Load(addon.Spec.Repositories)

	r.log.Info("- check duplicate ID addons alongside repositories")
	repositories.ReviseAddonDuplicationInRepository()

	list, err := r.existingAddonsConfigurations(addon)
	if err != nil {
		return exerr.Wrap(err, "while fetching AddonsConfiguration list")
	}
	r.log.Info("- check duplicates ID addons in existing AddonsConfiguration")
	repositories.ReviseAddonDuplicationInStorage(list)

	if repositories.IsRepositoriesFailed() {
		addon.Status.Phase = addonsv1alpha1.AddonsConfigurationFailed
	} else {
		addon.Status.Phase = addonsv1alpha1.AddonsConfigurationReady
	}
	r.log.Infof("- status: %s", addon.Status.Phase)

	var deletedAddonsIDs []string
	saved := false

	switch addon.Status.Phase {
	case addonsv1alpha1.AddonsConfigurationFailed:
		r.statusSnapshot(&addon.Status.CommonAddonsConfigurationStatus, repositories)
		if _, err = r.updateAddonStatus(addon); err != nil {
			return exerr.Wrap(err, "while updating AddonsConfiguration status")
		}
		if lastStatus.Phase == addonsv1alpha1.AddonsConfigurationReady {
			deletedAddonsIDs, err = r.deletePreviousAddons(internal.Namespace(addon.Namespace), lastStatus.Repositories)
			if err != nil {
				return exerr.Wrap(err, "while deleting addons from repository")
			}
		}
	case addonsv1alpha1.AddonsConfigurationReady:
		r.log.Info("- save ready addons and charts in storage")
		saved = r.saveAddons(addon.Namespace, repositories)

		r.statusSnapshot(&addon.Status.CommonAddonsConfigurationStatus, repositories)
		if _, err := r.updateAddonStatus(addon); err != nil {
			return exerr.Wrap(err, "while updating AddonsConfiguration status")
		}
		if lastStatus.Phase == addonsv1alpha1.AddonsConfigurationReady {
			deletedAddonsIDs, err = r.deleteOrphanAddons(internal.Namespace(addon.Namespace), addon.Status.Repositories, lastStatus.Repositories)
			if err != nil {
				return exerr.Wrap(err, "while deleting orphan addons from storage")
			}
		}
	}
	if saved || len(deletedAddonsIDs) > 0 {
		r.log.Info("- ensure ServiceBroker")
		if err = r.ensureBroker(addon); err != nil {
			return exerr.Wrap(err, "while ensuring ServiceBroker")
		}
	}

	if len(deletedAddonsIDs) > 0 {
		r.log.Info("- reprocessing conflicting addons configurations")
		if err := r.reprocessConfigurationsInConflict(deletedAddonsIDs, list); err != nil {
			return exerr.Wrap(err, "while reprocessing configurations in conflict")
		}
	}

	return nil
}

func (r *ReconcileAddonsConfiguration) reprocessConfigurationsInConflict(deletedAddonsIDs []string, list *addonsv1alpha1.AddonsConfigurationList) error {
	for _, id := range deletedAddonsIDs {
		for _, configuration := range list.Items {
			if hasConflict := r.isConfigurationInConflict(id, configuration.Status.CommonAddonsConfigurationStatus); hasConflict {
				if err := r.reprocessRequest(&configuration); err != nil {
					return exerr.Wrapf(err, "while reprocessing AddonsConfiguration %s", configuration.Name)
				}
			}
		}
	}
	return nil
}

func (r *ReconcileAddonsConfiguration) deleteAddonsProcess(addon *addonsv1alpha1.AddonsConfiguration) error {
	r.log.Infof("Start delete AddonsConfiguration %s/%s process", addon.Name, addon.Namespace)

	if addon.Status.Phase == addonsv1alpha1.AddonsConfigurationReady {
		adds, err := r.existingAddonsConfigurations(addon)
		if err != nil {
			return exerr.Wrapf(err, "while listing AddonsConfigurations in namespace %s", addon.Namespace)
		}

		deleteBroker := true
		for _, addon := range adds.Items {
			if addon.Status.Phase != addonsv1alpha1.AddonsConfigurationReady {
				// reprocess AddonConfig again if it was failed
				if err := r.reprocessRequest(&addon); err != nil {
					return exerr.Wrapf(err, "while requesting reprocess for AddonsConfiguration %s", addon.Name)

				}
			} else {
				deleteBroker = false
			}
		}
		if deleteBroker {
			r.log.Info("- delete ServiceBroker from namespace %s", addon.Namespace)
			if err := r.brokerFacade.Delete(addon.Namespace); err != nil {
				return exerr.Wrapf(err, "while deleting ServiceBroker from namespace %s", addon.Namespace)
			}
		}

		addonRemoved := false
		for _, repo := range addon.Status.Repositories {
			for _, a := range repo.Addons {
				addonRemoved, err = r.removeAddon(a, internal.Namespace(addon.Namespace))
				if err != nil && !storage.IsNotFoundError(err) {
					return exerr.Wrapf(err, "while deleting addon with charts for addon %s", a.Name)
				}
			}
		}
		if !deleteBroker && addonRemoved {
			if err := r.brokerSyncer.SyncServiceBroker(addon.Namespace); err != nil {
				return exerr.Wrapf(err, "while syncing ServiceBroker for addon %s", addon.Name)
			}
		}
	}
	if err := r.deleteFinalizer(addon); err != nil {
		return exerr.Wrapf(err, "while deleting finalizer for AddonConfiguration %s/%s", addon.Name, addon.Namespace)
	}

	r.log.Info("Delete AddonsConfiguration process completed")
	return nil
}

func (r *ReconcileAddonsConfiguration) ensureBroker(addon *addonsv1alpha1.AddonsConfiguration) error {
	exist, err := r.brokerFacade.Exist(addon.Namespace)
	if err != nil {
		return exerr.Wrapf(err, "while checking if ServiceBroker exist in namespace %s", addon.Namespace)
	}
	if !exist {
		r.log.Infof("- creating ServiceBroker in namespace %s", addon.Namespace)
		if err := r.brokerFacade.Create(addon.Namespace); err != nil {
			return exerr.Wrapf(err, "while creating ServiceBroker for AddonConfiguration %s/%s", addon.Name, addon.Namespace)
		}
	} else {
		r.log.Infof("- syncing ServiceBroker in namespace %s", addon.Namespace)
		if err := r.brokerSyncer.SyncServiceBroker(addon.Namespace); err != nil {
			return exerr.Wrapf(err, "while syncing ServiceBroker for AddonConfiguration %s/%s", addon.Name, addon.Namespace)
		}
	}
	return nil
}

func (r *ReconcileAddonsConfiguration) prepareForProcessing(addon *addonsv1alpha1.AddonsConfiguration) (*addonsv1alpha1.AddonsConfiguration, error) {
	obj := addon.DeepCopy()
	obj.Status.Phase = addonsv1alpha1.AddonsConfigurationPending

	pendingInstance, err := r.updateAddonStatus(obj)
	if err != nil {
		return nil, err
	}
	if r.protection.hasFinalizer(pendingInstance.Finalizers) {
		return pendingInstance, nil
	}
	r.log.Info("- add a finalizer")
	pendingInstance.Finalizers = r.protection.addFinalizer(pendingInstance.Finalizers)

	err = r.Client.Update(context.Background(), pendingInstance)
	if err != nil {
		return nil, err
	}
	return pendingInstance, nil
}

func (r *ReconcileAddonsConfiguration) deleteFinalizer(addon *addonsv1alpha1.AddonsConfiguration) error {
	obj := addon.DeepCopy()
	if !r.protection.hasFinalizer(obj.Finalizers) {
		return nil
	}
	r.log.Info("- delete a finalizer")
	obj.Finalizers = r.protection.removeFinalizer(obj.Finalizers)

	return r.Client.Update(context.Background(), obj)
}

func (r *ReconcileAddonsConfiguration) existingAddonsConfigurations(addon *addonsv1alpha1.AddonsConfiguration) (*addonsv1alpha1.AddonsConfigurationList, error) {
	addonsList := &addonsv1alpha1.AddonsConfigurationList{}
	addonsConfigurationList := &addonsv1alpha1.AddonsConfigurationList{}

	err := r.Client.List(context.TODO(), &client.ListOptions{Namespace: addon.Namespace}, addonsConfigurationList)
	if err != nil {
		return addonsConfigurationList, exerr.Wrap(err, "during fetching AddonConfiguration list by client")
	}

	for _, existAddon := range addonsConfigurationList.Items {
		if existAddon.Name != addon.Name {
			addonsList.Items = append(addonsList.Items, existAddon)
		}
	}

	return addonsList, nil
}

func (r *ReconcileAddonsConfiguration) reprocessRequest(addon *addonsv1alpha1.AddonsConfiguration) error {
	ad := &addonsv1alpha1.AddonsConfiguration{}
	if err := r.Client.Get(context.Background(), types.NamespacedName{Name: addon.Name, Namespace: addon.Namespace}, ad); err != nil {
		return exerr.Wrapf(err, "while getting AddonsConfiguration %s", addon.Name)
	}
	ad.Spec.ReprocessRequest++
	if err := r.Client.Update(context.Background(), ad); err != nil {
		return exerr.Wrapf(err, "while incrementing a reprocess requests for AddonsConfiguration %s", addon.Name)
	}
	return nil
}

func (r *ReconcileAddonsConfiguration) updateAddonStatus(addon *addonsv1alpha1.AddonsConfiguration) (*addonsv1alpha1.AddonsConfiguration, error) {
	addon.Status.ObservedGeneration = addon.Generation
	addon.Status.LastProcessedTime = &v1.Time{Time: time.Now()}

	r.log.Infof("- update AddonsConfiguration %s/%s status", addon.Name, addon.Namespace)
	err := r.Status().Update(context.TODO(), addon)
	if err != nil {
		return nil, exerr.Wrap(err, "while update AddonsConfiguration status")
	}
	return addon, nil
}
