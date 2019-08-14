package controller

import (
	"fmt"

	"time"

	"github.com/Masterminds/semver"
	"github.com/kyma-project/helm-broker/internal"
	"github.com/kyma-project/helm-broker/internal/addon"
	"github.com/kyma-project/helm-broker/internal/controller/repository"
	"github.com/kyma-project/helm-broker/internal/storage"
	"github.com/kyma-project/helm-broker/pkg/apis/addons/v1alpha1"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/helm/pkg/proto/hapi/chart"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

type addonManager struct {
	addonGetterFactory addonGetterFactory

	addonStorage addonStorage
	chartStorage chartStorage

	brokerSyncer brokerSyncer
	brokerFacade brokerFacade

	docsProvider docsProvider
	protection   protection
	dstPath      string

	// used to distinguish namespace-scoped and cluster-wide addons configurations
	namespace internal.Namespace

	addonsClient
	log logrus.FieldLogger
}

func newAddonManager(client client.Client, addonGetterFactory addonGetterFactory, addonStorage addonStorage, chartStorage chartStorage, docsProvider docsProvider, brokerSyncer brokerSyncer, brokerFacade brokerFacade, dstPath string, log logrus.FieldLogger) *addonManager {
	return &addonManager{
		addonGetterFactory: addonGetterFactory,

		addonStorage: addonStorage,
		chartStorage: chartStorage,

		brokerFacade: brokerFacade,
		brokerSyncer: brokerSyncer,

		docsProvider: docsProvider,
		protection:   protection{},
		dstPath:      dstPath,

		namespace: internal.ClusterWide,

		addonsClient: NewAddonsClient(client),

		log: log,
	}
}

// SetWorkingNamespace sets services's working namespace. It should only be used by the namespace-scoped controller.
func (a *addonManager) SetWorkingNamespace(namespace string) {
	a.namespace = internal.Namespace(namespace)
	for _, svc := range []NamespacedService{
		a.brokerSyncer, a.brokerFacade, a.docsProvider, a.addonsClient,
	} {
		svc.SetNamespace(namespace)
	}
}

func (a *addonManager) ReconcileOnAdd(addon *internal.CommonAddon, lastStatus v1alpha1.CommonAddonsConfigurationStatus) error {
	a.log.Infof("- load addons and charts for each addon")
	repositories := a.load(addon.Spec.Repositories)

	a.log.Info("- check duplicate ID addons alongside repositories")
	repositories.ReviseAddonDuplicationInRepository()

	a.log.Info("- check duplicates ID addons in existing addons configurations")
	list, err := a.existingAddonsConfigurations(addon.Meta.Name)
	if err != nil {
		return errors.Wrap(err, "while fetching addons configurations list")
	}
	repositories.ReviseAddonDuplicationInStorage(list)

	if repositories.IsRepositoriesFailed() {
		addon.Status.Phase = v1alpha1.AddonsConfigurationFailed
	} else {
		addon.Status.Phase = v1alpha1.AddonsConfigurationReady
	}
	a.log.Infof("- status: %s", addon.Status.Phase)

	var deletedAddonsIDs []string
	saved := false

	switch addon.Status.Phase {
	case v1alpha1.AddonsConfigurationFailed:
		addon.Status = a.statusSnapshot(addon.Status, repositories)
		if _, err = a.updateAddonStatus(addon); err != nil {
			return errors.Wrap(err, "while update addons configuration status")
		}
		if lastStatus.Phase == v1alpha1.AddonsConfigurationReady {
			deletedAddonsIDs, err = a.deletePreviousAddons(lastStatus.Repositories)
			if err != nil {
				return errors.Wrap(err, "while deleting previous addons from storage")
			}
		}
	case v1alpha1.AddonsConfigurationReady:
		saved = a.saveAddons(repositories)

		addon.Status = a.statusSnapshot(addon.Status, repositories)
		if _, err = a.updateAddonStatus(addon); err != nil {
			return errors.Wrap(err, "while update addons configuration status")
		}
		if lastStatus.Phase == v1alpha1.AddonsConfigurationReady {
			deletedAddonsIDs, err = a.deleteOrphanAddons(addon.Status.Repositories, lastStatus.Repositories)
			if err != nil {
				return errors.Wrap(err, "while deleting orphan addons from storage")
			}
		}
	}
	if saved || len(deletedAddonsIDs) > 0 {
		if err = a.ensureBroker(); err != nil {
			return errors.Wrap(err, "while ensuring broker")
		}
	}

	if len(deletedAddonsIDs) > 0 {
		a.log.Info("- reprocessing conflicting addons configurations")
		if err := a.reprocessConfigurationsInConflict(deletedAddonsIDs, list); err != nil {
			return errors.Wrap(err, "while reprocessing configurations in conflict")
		}
	}

	return nil
}

func (a *addonManager) ReconcileOnDelete(addon *internal.CommonAddon) error {
	if addon.Status.Phase == v1alpha1.AddonsConfigurationReady {
		adds, err := a.existingAddonsConfigurations(addon.Meta.Name)
		if err != nil {
			return errors.Wrap(err, "while listing addons configurations")
		}

		deleteBroker := true
		for _, ad := range adds {
			if ad.Status.Phase != v1alpha1.AddonsConfigurationReady {
				// reprocess addons configuration again if was failed
				if err := a.ReprocessRequest(ad.Meta.Name); err != nil {
					return errors.Wrapf(err, "while requesting reprocess addons configuration %s", ad.Meta.Name)
				}
			} else {
				deleteBroker = false
			}
		}
		if deleteBroker {
			if err := a.brokerFacade.Delete(); err != nil {
				return errors.Wrap(err, "while deleting broker")
			}
		}

		addonRemoved := false
		for _, repo := range addon.Status.Repositories {
			for _, ad := range repo.Addons {
				addonRemoved, err = a.removeAddon(ad)
				if err != nil && !storage.IsNotFoundError(err) {
					return errors.Wrapf(err, "while deleting addon with charts for addon %s", ad.Name)
				}
			}
		}
		if !deleteBroker && addonRemoved {
			if err := a.brokerSyncer.Sync(); err != nil {
				return errors.Wrapf(err, "while syncing broker for addon %s", addon.Meta.Name)
			}
		}
	}
	if _, err := a.deleteFinalizer(addon); err != nil {
		return errors.Wrapf(err, "while deleting finalizer from addons configuration %s", addon.Meta.Name)
	}

	return nil
}

// load loads repositories from given addon
func (a *addonManager) load(repos []v1alpha1.SpecRepository) *repository.Collection {
	repositories := repository.NewRepositoryCollection()
	for _, specRepository := range repos {
		a.log.Infof("- create addons for %q repository", specRepository.URL)
		repo := repository.NewAddonsRepository(specRepository.URL)

		adds, err := a.createAddons(specRepository.URL)
		if err != nil {
			repo.FetchingError(err)
			repositories.AddRepository(repo)

			a.log.Errorf("while creating addons for repository from %q: %s", specRepository.URL, err)
			continue
		}

		repo.Addons = adds
		repositories.AddRepository(repo)
	}
	return repositories
}

func (a *addonManager) createAddons(URL string) ([]*repository.Entry, error) {
	concreteGetter, err := a.addonGetterFactory.NewGetter(URL, a.dstPath)
	if err != nil {
		return nil, err
	}
	defer concreteGetter.Cleanup()

	// fetch repository index
	index, err := concreteGetter.GetIndex()
	if err != nil {
		return nil, errors.Wrap(err, "while reading repository index")
	}

	// for each repository entry create addon
	var adds []*repository.Entry
	for _, entries := range index.Entries {
		for _, entry := range entries {
			ad := repository.NewRepositoryEntry(string(entry.Name), string(entry.Version), URL)
			adds = append(adds, ad)

			completeAddon, err := concreteGetter.GetCompleteAddon(entry)
			switch {
			case err == nil:
				ad.ID = string(completeAddon.Addon.ID)
				ad.AddonWithCharts.Addon = completeAddon.Addon
				ad.AddonWithCharts.Charts = completeAddon.Charts
				ad.AddonWithCharts.Addon.Status = v1alpha1.AddonStatusReady
			case addon.IsFetchingError(err):
				ad.FetchingError(err)
				a.log.WithField("addon", fmt.Sprintf("%s-%s", entry.Name, entry.Version)).Errorf("while fetching addon: %s", err)
			default:
				ad.LoadingError(err)
				a.log.WithField("addon", fmt.Sprintf("%s-%s", entry.Name, entry.Version)).Errorf("while loading addon: %s", err)
			}
		}
	}

	return adds, nil
}

func (a *addonManager) saveAddons(repositories *repository.Collection) bool {
	a.log.Info("- save ready addons and charts in storage")
	saved := false

	for _, ad := range repositories.ReadyAddons() {
		if len(ad.AddonWithCharts.Addon.Docs) == 1 {
			if err := a.docsProvider.EnsureDocsTopic(ad.AddonWithCharts.Addon); err != nil {
				a.log.Errorf("while ensuring DocsTopic for addon %s: %v", ad.ID, err)
			}
		}
		exist, err := a.addonStorage.Upsert(a.namespace, ad.AddonWithCharts.Addon)
		if err != nil {
			ad.RegisteringError(err)
			a.log.Errorf("cannot upsert addon %v:%v into storage", ad.AddonWithCharts.Addon.Name, ad.AddonWithCharts.Addon.Version.Original())
			continue
		}
		saved = true
		err = a.saveCharts(ad.AddonWithCharts.Charts)
		if err != nil {
			ad.RegisteringError(err)
			a.log.Errorf("cannot upsert charts of %v:%v addon", ad.AddonWithCharts.Addon.Name, ad.AddonWithCharts.Addon.Version.Original())
			continue
		}
		if exist {
			a.log.Infof("addon %v:%v already existed in storage, addon was replaced", ad.AddonWithCharts.Addon.Name, ad.AddonWithCharts.Addon.Version.Original())
		}
	}
	return saved
}

func (a *addonManager) saveCharts(charts []*chart.Chart) error {
	for _, addonChart := range charts {
		exist, err := a.chartStorage.Upsert(a.namespace, addonChart)
		if err != nil {
			return err
		}
		if exist {
			a.log.Infof("chart %s already existed in storage, chart was replaced", addonChart.Metadata.Name)
		}
	}
	return nil
}

func (a *addonManager) removeAddon(ad v1alpha1.Addon) (bool, error) {
	removed := false
	a.log.Infof("- delete addon %s from storage", ad.Name)
	add, err := a.addonStorage.Get(a.namespace, internal.AddonName(ad.Name), *semver.MustParse(ad.Version))
	if err != nil {
		return false, err
	}

	err = a.addonStorage.Remove(a.namespace, internal.AddonName(ad.Name), add.Version)
	if err != nil {
		return false, err
	}
	removed = true
	if err := a.docsProvider.EnsureDocsTopicRemoved(string(add.ID)); err != nil {
		return removed, errors.Wrapf(err, "while ensuring DocsTopic for addon %s is removed", add.ID)
	}

	for _, plan := range add.Plans {
		err = a.chartStorage.Remove(a.namespace, plan.ChartRef.Name, plan.ChartRef.Version)
		if err != nil {
			return removed, err
		}
	}
	return removed, nil
}

// deletePreviousAddons delete addons if configuration was ready and then failed
func (a *addonManager) deletePreviousAddons(repos []v1alpha1.StatusRepository) ([]string, error) {
	var deletedAddonsIDs []string
	for _, repo := range repos {
		for _, ad := range repo.Addons {
			if _, err := a.removeAddon(ad); err != nil && !storage.IsNotFoundError(err) {
				return nil, errors.Wrapf(err, "while deleting addons and charts for addon %s", ad.Name)
			}
			deletedAddonsIDs = append(deletedAddonsIDs, ad.Key())
		}
	}
	return deletedAddonsIDs, nil
}

// deleteOrphanAddons deletes addons if configuration was modified and some addons have ceased to be provided
func (a *addonManager) deleteOrphanAddons(repos []v1alpha1.StatusRepository, lastRepos []v1alpha1.StatusRepository) ([]string, error) {
	addonsToStay := map[string]v1alpha1.Addon{}
	for _, repo := range repos {
		for _, ad := range repo.Addons {
			addonsToStay[ad.Key()] = ad
		}
	}
	var deletedAddonsIDs []string
	for _, repo := range lastRepos {
		for _, ad := range repo.Addons {
			if _, exist := addonsToStay[ad.Key()]; !exist {
				if _, err := a.removeAddon(ad); err != nil && !storage.IsNotFoundError(err) {
					return nil, errors.Wrapf(err, "while deleting addons and charts for addon %s", ad.Name)
				}
				deletedAddonsIDs = append(deletedAddonsIDs, ad.Key())
			}
		}
	}
	return deletedAddonsIDs, nil
}

func (a *addonManager) ensureBroker() error {
	exist, err := a.brokerFacade.Exist()
	if err != nil {
		return errors.Wrap(err, "while checking if Broker exist")
	}
	if !exist {
		if err := a.brokerFacade.Create(); err != nil {
			return errors.Wrap(err, "while creating Broker")
		}
	} else {
		if err := a.brokerSyncer.Sync(); err != nil {
			return errors.Wrap(err, "while syncing Broker")
		}
	}
	return nil
}

func (a *addonManager) reprocessConfigurationsInConflict(deletedAddonsIDs []string, list []internal.CommonAddon) error {
	for _, id := range deletedAddonsIDs {
		for _, configuration := range list {
			if hasConflict := a.isConfigurationInConflict(id, configuration.Status); hasConflict {
				if err := a.ReprocessRequest(configuration.Meta.Name); err != nil {
					return errors.Wrapf(err, "while reprocessing addons configuration %s", configuration.Meta.Name)
				}
			}
		}
	}
	return nil
}

func (a *addonManager) isConfigurationInConflict(key string, status v1alpha1.CommonAddonsConfigurationStatus) bool {
	if status.Phase != v1alpha1.AddonsConfigurationReady {
		for _, repo := range status.Repositories {
			if repo.Status != v1alpha1.RepositoryStatusReady {
				for _, a := range repo.Addons {
					if a.Key() == key {
						return true
					}
				}
			}
		}
	}
	return false
}

func (a *addonManager) statusSnapshot(status v1alpha1.CommonAddonsConfigurationStatus, repositories *repository.Collection) v1alpha1.CommonAddonsConfigurationStatus {
	status.Repositories = nil

	for _, repo := range repositories.Repositories {
		addonsRepository := repo.Repository
		addonsRepository.Addons = []v1alpha1.Addon{}
		for _, ad := range repo.Addons {
			addonsRepository.Addons = append(addonsRepository.Addons, v1alpha1.Addon{
				Name:    string(ad.AddonWithCharts.Addon.Name),
				Status:  ad.AddonWithCharts.Addon.Status,
				Reason:  ad.AddonWithCharts.Addon.Reason,
				Message: ad.AddonWithCharts.Addon.Message,
				Version: ad.AddonWithCharts.Addon.Version.Original(),
			})
		}
		status.Repositories = append(status.Repositories, addonsRepository)
	}
	return status
}

func (a *addonManager) PrepareForProcessing(addon *internal.CommonAddon) (*internal.CommonAddon, error) {
	addon.Status.Phase = v1alpha1.AddonsConfigurationPending
	updatedAddon, err := a.updateAddonStatus(addon)
	if err != nil {
		return nil, errors.Wrap(err, "while updating status")
	}
	updatedAddon, err = a.addFinalizer(updatedAddon)
	if err != nil {
		return nil, errors.Wrap(err, "while adding finalizer")
	}

	return updatedAddon, nil
}

func (a *addonManager) existingAddonsConfigurations(addonName string) ([]internal.CommonAddon, error) {
	var result []internal.CommonAddon

	cfgs, err := a.ListConfigurations()
	if err != nil {
		return nil, errors.Wrap(err, "while listing addons configurations")
	}
	for _, cfg := range cfgs {
		if cfg.Meta.Name != addonName {
			result = append(result, cfg)
		}
	}

	return result, nil
}

func (a *addonManager) deleteFinalizer(addon *internal.CommonAddon) (*internal.CommonAddon, error) {
	a.log.Info("- delete a finalizer")
	addon.Meta.Finalizers = a.protection.removeFinalizer(addon.Meta.Finalizers)

	return a.UpdateConfiguration(addon)
}

func (a *addonManager) addFinalizer(addon *internal.CommonAddon) (*internal.CommonAddon, error) {
	a.log.Info("- add a finalizer")
	addon.Meta.Finalizers = a.protection.addFinalizer(addon.Meta.Finalizers)

	return a.UpdateConfiguration(addon)
}

func (a *addonManager) updateAddonStatus(addon *internal.CommonAddon) (*internal.CommonAddon, error) {
	addon.Status.ObservedGeneration = addon.Meta.Generation
	addon.Status.LastProcessedTime = &v1.Time{Time: time.Now()}

	return a.UpdateConfigurationStatus(addon)
}
