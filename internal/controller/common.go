package controller

import (
	"fmt"

	"github.com/Masterminds/semver"
	"github.com/kyma-project/helm-broker/internal"
	"github.com/kyma-project/helm-broker/internal/addon"
	"github.com/kyma-project/helm-broker/internal/controller/repository"
	"github.com/kyma-project/helm-broker/internal/storage"
	"github.com/kyma-project/helm-broker/pkg/apis/addons/v1alpha1"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
	"k8s.io/helm/pkg/proto/hapi/chart"
)

type addonManager struct {
	addonGetterFactory addonGetterFactory

	addonStorage addonStorage
	chartStorage chartStorage

	docsProvider docsProvider
	protection   protection
	dstPath      string

	log logrus.FieldLogger
}

func newAddonManager(addonGetterFactory addonGetterFactory, addonStorage addonStorage, chartStorage chartStorage, docsProvider docsProvider, dstPath string, log logrus.FieldLogger) *addonManager {
	return &addonManager{
		addonGetterFactory: addonGetterFactory,

		addonStorage: addonStorage,
		chartStorage: chartStorage,

		docsProvider: docsProvider,
		protection:   protection{},
		dstPath:      dstPath,

		log: log,
	}
}

// Load loads repositories from given addon
func (a *addonManager) Load(repos []v1alpha1.SpecRepository) *repository.Collection {
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
				ad.Addon = completeAddon.Addon
				ad.Charts = completeAddon.Charts
			case addon.IsFetchingError(err):
				ad.FetchingError(err)
				a.log.WithField("ad", fmt.Sprintf("%s-%s", entry.Name, entry.Version)).Errorf("while fetching ad: %s", err)
			default:
				ad.LoadingError(err)
				a.log.WithField("ad", fmt.Sprintf("%s-%s", entry.Name, entry.Version)).Errorf("while loading ad: %s", err)
			}
		}
	}

	return adds, nil
}

func (a *addonManager) saveAddons(namespace string, repositories *repository.Collection) bool {
	saved := false
	for _, ad := range repositories.ReadyAddons() {
		if len(ad.Addon.Docs) == 1 {
			a.log.Infof("- ensure ClusterDocsTopic for ad %s", ad.Addon.ID)
			if err := a.docsProvider.EnsureDocsTopic(ad.Addon, namespace); err != nil {
				a.log.Errorf("while ensuring ClusterDocsTopic for ad %s: %v", ad.Addon.ID, err)
			}
		}
		exist, err := a.addonStorage.Upsert(internal.Namespace(namespace), ad.Addon)
		if err != nil {
			ad.RegisteringError(err)
			a.log.Errorf("cannot upsert ad %v:%v into storage", ad.Addon.Name, ad.Addon.Version)
			continue
		}
		saved = true
		err = a.saveCharts(namespace, ad.Charts)
		if err != nil {
			ad.RegisteringError(err)
			a.log.Errorf("cannot upsert charts of %v:%v ad", ad.Addon.Name, ad.Addon.Version)
			continue
		}
		if exist {
			a.log.Infof("ad %v:%v already existed in storage, ad was replaced", ad.Addon.Name, ad.Addon.Version)
		}
	}
	return saved
}

func (a *addonManager) saveCharts(namespace string, charts []*chart.Chart) error {
	for _, addonChart := range charts {
		exist, err := a.chartStorage.Upsert(internal.Namespace(namespace), addonChart)
		if err != nil {
			return err
		}
		if exist {
			a.log.Infof("chart %s already existed in storage, chart was replaced", addonChart.Metadata.Name)
		}
	}
	return nil
}

func (a *addonManager) removeAddon(ad v1alpha1.Addon, namespace internal.Namespace) (bool, error) {
	removed := false
	a.log.Infof("- delete addon %s from storage", ad.Name)
	add, err := a.addonStorage.Get(namespace, internal.AddonName(ad.Name), *semver.MustParse(ad.Version))
	if err != nil {
		return false, err
	}

	err = a.addonStorage.Remove(namespace, internal.AddonName(ad.Name), add.Version)
	if err != nil {
		return false, err
	}
	removed = true
	a.log.Infof("- delete DocsTopic for addon %s", add)
	if err := a.docsProvider.EnsureDocsTopicRemoved(string(add.ID), string(namespace)); err != nil {
		return removed, errors.Wrapf(err, "while ensuring DocsTopic for addon %s is removed", add.ID)
	}

	for _, plan := range add.Plans {
		err = a.chartStorage.Remove(namespace, plan.ChartRef.Name, plan.ChartRef.Version)
		if err != nil {
			return removed, err
		}
	}
	return removed, nil
}

// deletePreviousAddons delete addons if configuration was ready and then failed
func (a *addonManager) deletePreviousAddons(namespace internal.Namespace, repos []v1alpha1.StatusRepository) ([]string, error) {
	var deletedAddonsIDs []string
	for _, repo := range repos {
		for _, ad := range repo.Addons {
			if _, err := a.removeAddon(ad, namespace); err != nil && !storage.IsNotFoundError(err) {
				return nil, errors.Wrapf(err, "while deleting addons and charts for addon %s", ad.Name)
			}
			deletedAddonsIDs = append(deletedAddonsIDs, ad.Key())
		}
	}
	return deletedAddonsIDs, nil
}

// deleteOrphanAddons deletes addons if configuration was modified and some addons have ceased to be provided
func (a *addonManager) deleteOrphanAddons(namespace internal.Namespace, repos []v1alpha1.StatusRepository, lastRepos []v1alpha1.StatusRepository) ([]string, error) {
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
				if _, err := a.removeAddon(ad, namespace); err != nil && !storage.IsNotFoundError(err) {
					return nil, errors.Wrapf(err, "while deleting addons and charts for addon %s", ad.Name)
				}
				deletedAddonsIDs = append(deletedAddonsIDs, ad.Key())
			}
		}
	}
	return deletedAddonsIDs, nil
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

func (a *addonManager) statusSnapshot(status *v1alpha1.CommonAddonsConfigurationStatus, repositories *repository.Collection) {
	status.Repositories = nil

	for _, repo := range repositories.Repositories {
		addonsRepository := repo.Repository
		addonsRepository.Addons = []v1alpha1.Addon{}
		for _, ad := range repo.Addons {
			addonsRepository.Addons = append(addonsRepository.Addons, ad.Entry)
		}
		status.Repositories = append(status.Repositories, addonsRepository)
	}
}
