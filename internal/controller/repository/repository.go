package repository

import (
	"fmt"

	addonsv1alpha1 "github.com/kyma-project/helm-broker/pkg/apis/addons/v1alpha1"
)

// Repository is a wraper for StatusRepository
type Repository struct {
	Repository addonsv1alpha1.StatusRepository
	Addons     []*Entry
}

// NewAddonsRepository returns pointer to new Repository with url and ready status
func NewAddonsRepository(url string) *Repository {
	return &Repository{
		Repository: addonsv1alpha1.StatusRepository{
			URL:    url,
			Status: addonsv1alpha1.RepositoryStatusReady,
		},
	}
}

// Failed sets StatusRepository as failed
func (ar *Repository) Failed() {
	ar.Repository.Status = addonsv1alpha1.RepositoryStatusFailed
}

// IsFailed checks is StatusRepository is in failed state
func (ar *Repository) IsFailed() bool {
	return ar.Repository.Status == addonsv1alpha1.RepositoryStatusFailed
}

// HasFailedAddons returns true if any addon in the repository has status Failed
func (ar *Repository) HasFailedAddons() bool {
	for _, addon := range ar.Addons {
		if !addon.IsReady() {
			return true
		}
	}
	return false
}

// FetchingError sets StatusRepository as failed with URLFetchingError as a reason
func (ar *Repository) FetchingError(err error) {
	reason := addonsv1alpha1.RepositoryURLFetchingError
	ar.Failed()
	ar.Repository.Reason = reason
	ar.Repository.Message = fmt.Sprintf(reason.Message(), err.Error())
}

// IsFetchingError checks if the repository failed because of Fetching error
func (ar *Repository) IsFetchingError() bool {
	return ar.Repository.Status == addonsv1alpha1.RepositoryStatusFailed &&
		ar.Repository.Reason == addonsv1alpha1.RepositoryURLFetchingError
}

// TemplatingError sets StatusRepository as failed with URLTemplatingError as a reason
func (ar *Repository) TemplatingError(err error) {
	reason := addonsv1alpha1.RepositoryURLTemplatingError
	ar.Failed()
	ar.Repository.Reason = reason
	ar.Repository.Message = fmt.Sprintf(reason.Message(), err.Error())
}

// EmptyURLError sets StatusRepository as failed with EmptyURLError as a reason
func (ar *Repository) EmptyURLError(err error) {
	reason := addonsv1alpha1.RepositoryEmptyURLError
	ar.Failed()
	ar.Repository.Reason = reason
	ar.Repository.Message = fmt.Sprintf(reason.Message())
}
