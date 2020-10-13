package repository

import (
	"fmt"

	"github.com/kyma-project/helm-broker/internal"
	addonsv1alpha1 "github.com/kyma-project/helm-broker/pkg/apis/addons/v1alpha1"
)

// Collection keeps and process collection of Repository
type Collection struct {
	Repositories []*Repository
}

// NewRepositoryCollection returns pointer to RepositoryCollection
func NewRepositoryCollection() *Collection {
	return &Collection{
		Repositories: []*Repository{},
	}
}

// AddRepository adds new Repository to RepositoryCollection
func (rc *Collection) AddRepository(repo *Repository) {
	rc.Repositories = append(rc.Repositories, repo)
}

func (rc *Collection) addons() []*Entry {
	var addons []*Entry

	for _, repo := range rc.Repositories {
		for _, addon := range repo.Addons {
			addons = append(addons, addon)
		}
	}

	return addons
}

func (rc *Collection) completeAddons() []*Entry {
	var addons []*Entry

	for _, addon := range rc.addons() {
		if !addon.IsComplete() {
			continue
		}
		addons = append(addons, addon)
	}

	return addons
}

// ReadyAddons returns all addons from all repositories which ready status
func (rc *Collection) ReadyAddons() []*Entry {
	var addons []*Entry

	for _, addon := range rc.addons() {
		if !addon.IsReady() {
			continue
		}
		addons = append(addons, addon)
	}

	return addons
}

// IsRepositoriesFailed informs if any of repositories in collection is in failed status
func (rc *Collection) IsRepositoriesFailed() bool {
	result := false
	for _, repository := range rc.Repositories {
		if repository.IsFailed() {
			result = true
			continue
		}
		if repository.HasFailedAddons() {
			repository.Failed()
			result = true
		}
	}

	return result
}

func (rc *Collection) IsRepositoriesFetchingError() bool {
	for _, repository := range rc.Repositories {
		if repository.IsFetchingError() {
			return true
		}
	}
	return false
}

type idConflictData struct {
	repositoryURL string
	addonsName    string
}

// ReviseAddonDuplicationInRepository checks all completed addons (addons without fetch/load error)
// they have no ID conflict with other addons in other or the same repository
func (rc *Collection) ReviseAddonDuplicationInRepository() {
	ids := make(map[string]idConflictData)

	for _, addon := range rc.completeAddons() {
		if data, ok := ids[addon.ID]; ok {
			addon.ConflictInSpecifiedRepositories(fmt.Errorf("[url: %s, addons: %s]", data.repositoryURL, data.addonsName))
		} else {
			ids[addon.ID] = idConflictData{
				repositoryURL: addon.URL,
				addonsName:    fmt.Sprintf("%s:%v", addon.AddonWithCharts.Addon.Name, addon.AddonWithCharts.Addon.Version.Original()),
			}
		}
	}
}

// ReviseAddonDuplicationInStorage checks all completed addons (addons without fetch/load error)
// they have no name:version conflict with other AddonConfiguration
func (rc *Collection) ReviseAddonDuplicationInStorage(addonsList []internal.CommonAddon) {
	for _, addon := range rc.completeAddons() {
		rc.findExistingAddon(addon, addonsList)
	}
}

func (rc *Collection) findExistingAddon(addon *Entry, list []internal.CommonAddon) {
	for _, existAddonConfiguration := range list {
		for _, repo := range existAddonConfiguration.Status.Repositories {
			if rc.addonAlreadyRegistered(*addon, rc.filterReadyAddons(repo)) {
				addon.ConflictWithAlreadyRegisteredAddons(fmt.Errorf("[ConfigurationName: %s, url: %s, addons: %s:%v]", existAddonConfiguration.Meta.Name, repo.URL, addon.AddonWithCharts.Addon.Name, addon.AddonWithCharts.Addon.Version.Original()))
			}
		}
	}
}

func (rc *Collection) filterReadyAddons(repository addonsv1alpha1.StatusRepository) []addonsv1alpha1.Addon {
	var addons []addonsv1alpha1.Addon

	for _, add := range repository.Addons {
		if add.Status == addonsv1alpha1.AddonStatusReady {
			addons = append(addons, add)
		}
	}

	return addons
}

func (rc *Collection) addonAlreadyRegistered(addon Entry, addons []addonsv1alpha1.Addon) bool {
	for _, existAddon := range addons {
		if string(addon.AddonWithCharts.Addon.Name) == existAddon.Name && addon.AddonWithCharts.Addon.Version.Original() == existAddon.Version {
			return true
		}
	}

	return false
}
