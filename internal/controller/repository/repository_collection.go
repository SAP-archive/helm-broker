package repository

import (
	"fmt"

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
	for _, repository := range rc.Repositories {
		if repository.IsFailed() {
			return true
		}
		if repository.HasFailedAddons() {
			repository.Failed()
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
				addonsName:    fmt.Sprintf("%s:%s", addon.Entry.Name, addon.Entry.Version),
			}
		}
	}
}

// ReviseAddonDuplicationInStorage checks all completed addons (addons without fetch/load error)
// they have no name:version conflict with other AddonConfiguration
func (rc *Collection) ReviseAddonDuplicationInStorage(acList *addonsv1alpha1.AddonsConfigurationList) {
	for _, addon := range rc.completeAddons() {
		rc.findExistingAddon(addon, acList)
	}
}

// ReviseAddonDuplicationInClusterStorage checks all completed addons (addons without fetch/load error)
// they have no name:version conflict with other AddonConfiguration
func (rc *Collection) ReviseAddonDuplicationInClusterStorage(acList *addonsv1alpha1.ClusterAddonsConfigurationList) {
	for _, addon := range rc.completeAddons() {
		rc.findExistingClusterAddon(addon, acList)
	}
}

func (rc *Collection) findExistingAddon(addon *Entry, list *addonsv1alpha1.AddonsConfigurationList) {
	for _, existAddonConfiguration := range list.Items {
		for _, repo := range existAddonConfiguration.Status.Repositories {
			if rc.addonAlreadyRegistered(*addon, rc.filterReadyAddons(repo)) {
				addon.ConflictWithAlreadyRegisteredAddons(fmt.Errorf("[ConfigurationName: %s, url: %s, addons: %s:%s]", existAddonConfiguration.Name, repo.URL, addon.Entry.Name, addon.Entry.Version))
			}
		}
	}
}

func (rc *Collection) findExistingClusterAddon(addon *Entry, list *addonsv1alpha1.ClusterAddonsConfigurationList) {
	for _, existAddonConfiguration := range list.Items {
		for _, repo := range existAddonConfiguration.Status.Repositories {
			if rc.addonAlreadyRegistered(*addon, rc.filterReadyAddons(repo)) {
				addon.ConflictWithAlreadyRegisteredAddons(fmt.Errorf("[ConfigurationName: %s, url: %s, addons: %s:%s]", existAddonConfiguration.Name, repo.URL, addon.Entry.Name, addon.Entry.Version))
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
		if addon.Entry.Name == existAddon.Name && addon.Entry.Version == existAddon.Version {
			return true
		}
	}

	return false
}
