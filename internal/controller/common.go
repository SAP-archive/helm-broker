package controller

import (
	"fmt"
	"strings"
	"time"

	"github.com/Masterminds/semver"
	"github.com/kyma-project/helm-broker/internal"
	"github.com/kyma-project/helm-broker/internal/addon"
	"github.com/kyma-project/helm-broker/internal/controller/repository"
	"github.com/kyma-project/helm-broker/internal/storage"
	"github.com/kyma-project/helm-broker/pkg/apis/addons/v1alpha1"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
	"helm.sh/helm/v3/pkg/chart"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
)

type common struct {
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

	commonClient    commonClient
	templateService templateService
	log             logrus.FieldLogger

	trace                    string
	reprocessOnErrorDuration time.Duration
}

var TemporaryError = errors.New("Temporary error")

func newControllerCommon(client client.Client, addonGetterFactory addonGetterFactory, addonStorage addonStorage,
	chartStorage chartStorage, docsProvider docsProvider, brokerSyncer brokerSyncer, brokerFacade brokerFacade,
	templateService templateService, dstPath string, reprocessOnErrorDuration time.Duration, log logrus.FieldLogger) *common {
	return &common{
		addonGetterFactory: addonGetterFactory,

		addonStorage: addonStorage,
		chartStorage: chartStorage,

		brokerFacade: brokerFacade,
		brokerSyncer: brokerSyncer,

		docsProvider: docsProvider,
		protection:   protection{},

		namespace:       internal.ClusterWide,
		templateService: templateService,
		commonClient:    NewCommonClient(client, log),

		dstPath: dstPath,
		log:     log,

		reprocessOnErrorDuration: reprocessOnErrorDuration,
	}
}

// SetWorkingNamespace sets services's working namespace. It should only be used by namespace-scoped controller.
func (c *common) SetWorkingNamespace(namespace string) {
	c.namespace = internal.Namespace(namespace)
	for _, svc := range []NamespacedService{
		c.brokerFacade, c.brokerSyncer, c.docsProvider, c.commonClient, c.templateService,
	} {
		svc.SetNamespace(namespace)
	}
}

// Reconcile handles configuration state change
func (c *common) Reconcile(addon *internal.CommonAddon, trace string) (reconcile.Result, error) {
	if addon.Meta.DeletionTimestamp != nil {
		c.log.Infof("Start delete %s", trace)

		if err := c.OnDelete(addon); err != nil {
			c.log.Errorf("while deleting %s process: %v", trace, err)
			return reconcile.Result{RequeueAfter: time.Second * 15}, errors.Wrapf(err, "while deleting %s", trace)
		}
		c.log.Infof("Delete %s process completed", trace)
		return reconcile.Result{}, nil
	}

	if addon.IsReadyForInitialProcessing() {
		c.log.Infof("Start add %s process", trace)

		if err := c.PrepareForProcessing(addon); err != nil {
			c.log.Errorf("while preparing %s for processing: %v", trace, err)
			return reconcile.Result{}, errors.Wrapf(err, "while preparing %s for processing", trace)
		}
		if err := c.OnAdd(addon, addon.Status); err != nil {
			c.log.Errorf("while adding %s process: %v", trace, err)
			if errors.Is(err, TemporaryError) {
				c.log.Infof("The error is temporary, marking for a reprocess")
				return reconcile.Result{RequeueAfter: c.reprocessOnErrorDuration}, errors.Wrapf(err, "while updating %s", trace)
			}
			return reconcile.Result{}, errors.Wrapf(err, "while creating %s", trace)
		}
		c.log.Infof("Add %s process completed", trace)

	} else if addon.IsReadyForReprocessing() {
		c.log.Infof("Start update %s process", trace)

		lastStatus := addon.Status
		addon.Status = v1alpha1.CommonAddonsConfigurationStatus{}

		if err := c.OnAdd(addon, lastStatus); err != nil {
			c.log.Errorf("while updating %s process: %v", trace, err)
			if errors.Is(err, TemporaryError) {
				c.log.Infof("The error is temporary, reprocessing")
				return reconcile.Result{RequeueAfter: c.reprocessOnErrorDuration}, errors.Wrapf(err, "while updating %s", trace)
			}
			return reconcile.Result{}, errors.Wrapf(err, "while updating %s", trace)
		}
		c.log.Infof("Update %s process completed", trace)
	}

	return reconcile.Result{}, nil
}

// PrepareForProcessing prepares ClusterAddonsConfiguration or AddonsConfiguration if namespace is set
func (c *common) PrepareForProcessing(addon *internal.CommonAddon) error {
	err := c.addFinalizer(addon)
	if err != nil {
		return errors.Wrap(err, "while adding finalizer")
	}
	addon.Status.Phase = v1alpha1.AddonsConfigurationPending
	err = c.updateAddonStatus(addon)
	if err != nil {
		return errors.Wrap(err, "while updating status")
	}

	return nil
}

// OnAdd executes logic on adding ClusterAddonsConfiguration or AddonsConfiguration if namespace is set
func (c *common) OnAdd(addon *internal.CommonAddon, lastStatus v1alpha1.CommonAddonsConfigurationStatus) error {
	c.log.Infof("- load addons and charts for each addon")
	repositories := c.loadRepositories(addon.Spec.Repositories)
	if repositories.IsRepositoriesFetchingError() {
		c.log.Warnf("Unable to read repositories, scheduling for a retry")
		addon.Status.Phase = v1alpha1.AddonsConfigurationFailed

		c.statusSnapshot(&addon.Status, repositories)
		if !addon.Status.Equals(&lastStatus) {
			// update status only if Status.Repositories has been changed
			if err := c.updateAddonStatus(addon); err != nil {
				return errors.Wrap(err, "while update addons configuration status")
			}
		}
		return TemporaryError
	}

	c.log.Info("- check duplicate ID addons alongside repositories")
	repositories.ReviseAddonDuplicationInRepository()

	c.log.Info("- check duplicates ID addons in existing addons configurations")
	list, err := c.listExistingConfigurations(addon.Meta.Name)
	if err != nil {
		return errors.Wrap(err, "while fetching addons configurations list")
	}
	repositories.ReviseAddonDuplicationInStorage(list)

	if repositories.IsRepositoriesFailed() {
		addon.Status.Phase = v1alpha1.AddonsConfigurationFailed
	} else {
		addon.Status.Phase = v1alpha1.AddonsConfigurationReady
	}
	c.log.Infof("- status: %s", addon.Status.Phase)

	var deletedAddonsIDs []string
	saved := false

	switch addon.Status.Phase {
	case v1alpha1.AddonsConfigurationFailed:
		c.statusSnapshot(&addon.Status, repositories)
		if err = c.updateAddonStatus(addon); err != nil {
			return errors.Wrap(err, "while update addons configuration status")
		}
		if lastStatus.Phase == v1alpha1.AddonsConfigurationReady {
			deletedAddonsIDs, err = c.deletePreviousAddons(lastStatus.Repositories)
			if err != nil {
				return errors.Wrap(err, "while deleting previous addons from storage")
			}
		}
	case v1alpha1.AddonsConfigurationReady:
		saved = c.saveAddons(repositories)
		c.statusSnapshot(&addon.Status, repositories)
		if err = c.updateAddonStatus(addon); err != nil {
			return errors.Wrap(err, "while update addons configuration status")
		}
		if lastStatus.Phase == v1alpha1.AddonsConfigurationReady {
			deletedAddonsIDs, err = c.deleteOrphanAddons(addon.Status.Repositories, lastStatus.Repositories)
			if err != nil {
				return errors.Wrap(err, "while deleting orphan addons from storage")
			}
		}
	}
	removed := len(deletedAddonsIDs) > 0

	if saved || removed {
		if err = c.ensureBroker(); err != nil {
			return errors.Wrap(err, "while ensuring broker")
		}
	}

	if removed {
		if err := c.reprocessConfigurationsInConflict(deletedAddonsIDs, list); err != nil {
			return errors.Wrap(err, "while reprocessing configurations in conflict")
		}
	}

	return nil
}

// OnDelete executes logic on deleting ClusterAddonsConfiguration or AddonsConfiguration if namespace is set
func (c *common) OnDelete(addon *internal.CommonAddon) error {
	if addon.Status.Phase == v1alpha1.AddonsConfigurationReady {
		adds, err := c.listExistingConfigurations(addon.Meta.Name)
		if err != nil {
			return errors.Wrap(err, "while listing addons configurations")
		}

		for _, ad := range adds {
			if ad.Status.Phase != v1alpha1.AddonsConfigurationReady {
				c.log.Infof("- reprocessing unblocked conflicting configuration `%s`", ad.Meta.Name)
				if err := c.commonClient.ReprocessRequest(ad.Meta.Name); err != nil {
					return errors.Wrapf(err, "while requesting reprocess addons configuration %s", ad.Meta.Name)
				}
			}
		}

		anyAddonRemoved := false
		for _, repo := range addon.Status.Repositories {
			for _, ad := range repo.Addons {
				removed, err := c.removeAddon(ad)
				anyAddonRemoved = anyAddonRemoved || removed
				if err != nil && !storage.IsNotFoundError(err) {
					return errors.Wrapf(err, "while deleting addon with charts for addon %s", ad.Name)
				}
			}
		}
		// if there was any removal processed - resync broker
		if anyAddonRemoved {
			if err := c.brokerSyncer.Sync(); err != nil {
				return errors.Wrapf(err, "while syncing broker for addon %s", addon.Meta.Name)
			}
		}
	}
	if err := c.deleteFinalizer(addon); err != nil {
		return errors.Wrapf(err, "while deleting finalizer from addons configuration %s", addon.Meta.Name)
	}

	return nil
}

// loadRepositories loads repositories from given addon
func (c *common) loadRepositories(repos []v1alpha1.SpecRepository) *repository.Collection {
	repositories := repository.NewRepositoryCollection()
	for _, specRepository := range repos {
		repositoryURL := strings.Split(specRepository.URL, "?")[0]
		c.log.Infof("- create addons for %q repository", repositoryURL)

		repo := repository.NewAddonsRepository(specRepository.URL)
		if specRepository.URL == "" {
			repo.EmptyURLError(errors.New("URL must not be empty"))
			repositories.AddRepository(repo)
			c.log.Error("URL must not be empty")
			continue
		}

		addonsURL, err := c.templateService.TemplateURL(specRepository)
		if err != nil {
			repo.TemplatingError(err)
			repositories.AddRepository(repo)
			c.log.Errorf("while templating repository URL `%s`: %v", repositoryURL, err)
			continue
		}

		adds, err := c.createAddons(addonsURL)
		if err != nil {
			repo.FetchingError(err)
			repositories.AddRepository(repo)
			c.log.Errorf("while creating addons for repository from %s: %v", repositoryURL, err)
			continue
		}

		repo.Addons = adds
		repositories.AddRepository(repo)
	}
	return repositories
}

func (c *common) createAddons(URL string) ([]*repository.Entry, error) {
	concreteGetter, err := c.addonGetterFactory.NewGetter(URL, c.dstPath)
	if err != nil {
		return nil, errors.Wrap(err, "while getting getter from factory")
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
				c.log.WithField("addon", fmt.Sprintf("%s-%s", entry.Name, entry.Version)).Errorf("while fetching addon: %s", err)
			default:
				ad.LoadingError(err)
				c.log.WithField("addon", fmt.Sprintf("%s-%s", entry.Name, entry.Version)).Errorf("while loading addon: %s", err)
			}
		}
	}

	return adds, nil
}

func (c *common) saveAddons(repositories *repository.Collection) bool {
	c.log.Info("- save ready addons and charts in storage")
	saved := false

	for _, ad := range repositories.ReadyAddons() {
		if len(ad.AddonWithCharts.Addon.Docs) == 1 {
			if err := c.docsProvider.EnsureAssetGroup(ad.AddonWithCharts.Addon); err != nil {
				c.log.Errorf("while ensuring documentation for addon %s: %v", ad.ID, err)
			}
		}
		exist, err := c.addonStorage.Upsert(c.namespace, ad.AddonWithCharts.Addon)
		if err != nil {
			ad.RegisteringError(err)
			c.log.Errorf("cannot upsert addon %v:%v into storage", ad.AddonWithCharts.Addon.Name, ad.AddonWithCharts.Addon.Version.Original())
			continue
		}
		saved = true
		err = c.saveCharts(ad.AddonWithCharts.Charts)
		if err != nil {
			ad.RegisteringError(err)
			c.log.Errorf("cannot upsert charts of %v:%v addon", ad.AddonWithCharts.Addon.Name, ad.AddonWithCharts.Addon.Version.Original())
			continue
		}
		if exist {
			c.log.Infof("- addon `%v:%v` already existed in storage, addon was replaced", ad.AddonWithCharts.Addon.Name, ad.AddonWithCharts.Addon.Version.Original())
		} else {
			c.log.Infof("- addon `%v:%v` saved to storage %s", ad.AddonWithCharts.Addon.Name, ad.AddonWithCharts.Addon.Version.Original(), c.namespace)
		}
	}
	return saved
}

func (c *common) saveCharts(charts []*chart.Chart) error {
	for _, addonChart := range charts {
		exist, err := c.chartStorage.Upsert(c.namespace, addonChart)
		if err != nil {
			return err
		}
		if exist {
			c.log.Infof("chart %s already existed in storage, chart was replaced", addonChart.Metadata.Name)
		}
	}
	return nil
}

func (c *common) removeAddon(ad v1alpha1.Addon) (bool, error) {
	removed := false
	c.log.Infof("- addon `%s:%s` removed from storage %s", ad.Name, ad.Version, c.namespace)
	add, err := c.addonStorage.Get(c.namespace, internal.AddonName(ad.Name), *semver.MustParse(ad.Version))
	if err != nil {
		return false, err
	}

	err = c.addonStorage.Remove(c.namespace, internal.AddonName(ad.Name), add.Version)
	if err != nil {
		return false, err
	}
	removed = true
	if err := c.docsProvider.EnsureAssetGroupRemoved(string(add.ID)); err != nil {
		return removed, errors.Wrapf(err, "while deleting documentation for addon %s", add.ID)
	}

	for _, plan := range add.Plans {
		err = c.chartStorage.Remove(c.namespace, plan.ChartRef.Name, plan.ChartRef.Version)
		if err != nil {
			return removed, err
		}
	}
	return removed, nil
}

// deletePreviousAddons delete addons if configuration was ready and then failed
func (c *common) deletePreviousAddons(repos []v1alpha1.StatusRepository) ([]string, error) {
	var deletedAddonsIDs []string
	for _, repo := range repos {
		for _, ad := range repo.Addons {
			if _, err := c.removeAddon(ad); err != nil && !storage.IsNotFoundError(err) {
				return nil, errors.Wrapf(err, "while deleting previous addons and charts for addon `%s`", ad.Name)
			}
			deletedAddonsIDs = append(deletedAddonsIDs, ad.Key())
		}
	}
	return deletedAddonsIDs, nil
}

// deleteOrphanAddons deletes addons if configuration was modified and some addons have ceased to be provided
func (c *common) deleteOrphanAddons(repos []v1alpha1.StatusRepository, lastRepos []v1alpha1.StatusRepository) ([]string, error) {
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
				if _, err := c.removeAddon(ad); err != nil && !storage.IsNotFoundError(err) {
					return nil, errors.Wrapf(err, "while deleting orphan addons and charts for addon `%s`", ad.Name)
				}
				deletedAddonsIDs = append(deletedAddonsIDs, ad.Key())
			}
		}
	}
	return deletedAddonsIDs, nil
}

func (c *common) ensureBroker() error {
	exist, err := c.brokerFacade.Exist()
	if err != nil {
		return errors.Wrap(err, "while checking if Broker exist")
	}
	if exist {
		if err := c.brokerSyncer.Sync(); err != nil {
			return errors.Wrap(err, "while syncing Broker")
		}
	}
	return nil
}

func (c *common) reprocessConfigurationsInConflict(deletedAddonsIDs []string, list []internal.CommonAddon) error {
	for _, id := range deletedAddonsIDs {
		for _, configuration := range list {
			if hasConflict := c.isConfigurationInConflict(id, configuration.Status); hasConflict {
				c.log.Infof("- reprocessing conflicting addons configuration `%s/%s`", configuration.Meta.Namespace, configuration.Meta.Name)
				if err := c.commonClient.ReprocessRequest(configuration.Meta.Name); err != nil {
					return errors.Wrapf(err, "while reprocessing conflicting addons configuration `%s`", configuration.Meta.Name)
				}
			}
		}
	}
	return nil
}

func (c *common) isConfigurationInConflict(key string, status v1alpha1.CommonAddonsConfigurationStatus) bool {
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

func (c *common) statusSnapshot(status *v1alpha1.CommonAddonsConfigurationStatus, repositories *repository.Collection) {
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
}

func (c *common) listExistingConfigurations(addonName string) ([]internal.CommonAddon, error) {
	var result []internal.CommonAddon

	cfgs, err := c.commonClient.ListConfigurations()
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

func (c *common) deleteFinalizer(addon *internal.CommonAddon) error {
	c.log.Info("- delete a finalizer")
	addon.Meta.Finalizers = c.protection.removeFinalizer(addon.Meta.Finalizers)

	return c.commonClient.UpdateConfiguration(addon)
}

func (c *common) addFinalizer(addon *internal.CommonAddon) error {
	c.log.Info("- add a finalizer")
	addon.Meta.Finalizers = c.protection.addFinalizer(addon.Meta.Finalizers)

	return c.commonClient.UpdateConfiguration(addon)
}

func (c *common) updateAddonStatus(addon *internal.CommonAddon) error {
	addon.Status.ObservedGeneration = addon.Meta.Generation
	addon.Status.LastProcessedTime = &v1.Time{Time: time.Now()}

	return c.commonClient.UpdateConfigurationStatus(addon)
}
