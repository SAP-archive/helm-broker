package repository

import (
	"fmt"
	"strings"

	"github.com/Masterminds/semver"
	"github.com/kyma-project/helm-broker/internal"
	"github.com/kyma-project/helm-broker/pkg/apis/addons/v1alpha1"
)

// Entry is a wraper for AddonWithCharts element with extra fields like URL or ID
type Entry struct {
	ID  string
	URL string

	AddonWithCharts *internal.AddonWithCharts
}

// NewRepositoryEntry returns pointer to new Entry based on name, version and url
func NewRepositoryEntry(n, v, u string) *Entry {
	return &Entry{
		URL: u,
		AddonWithCharts: &internal.AddonWithCharts{
			Addon: &internal.Addon{
				Name:    internal.AddonName(n),
				Version: *semver.MustParse(v),
				Status:  v1alpha1.AddonStatusReady,
			},
		},
	}
}

// IsReady informs addon is in ready status
func (a *Entry) IsReady() bool {
	return a.AddonWithCharts.Addon.Status == v1alpha1.AddonStatusReady
}

// IsComplete informs RepositoryEntry has no fetching/loading error, what means own ID (from addon)
func (a *Entry) IsComplete() bool {
	return a.ID != ""
}

// FetchingError sets addons as failed, sets addon reason as FetchingError
func (a *Entry) FetchingError(err error) {
	a.failed()
	a.setEntryStatus(v1alpha1.AddonFetchingError, a.limitMessage(err.Error()))
}

// LoadingError sets addons as failed, sets addon reason as LoadingError
func (a *Entry) LoadingError(err error) {
	a.failed()
	a.setEntryStatus(v1alpha1.AddonLoadingError, err.Error())
}

// ConflictInSpecifiedRepositories sets addons as failed, sets addon reason as ConflictInSpecifiedRepositories
func (a *Entry) ConflictInSpecifiedRepositories(err error) {
	a.failed()
	a.setEntryStatus(v1alpha1.AddonConflictInSpecifiedRepositories, err.Error())
}

// ConflictWithAlreadyRegisteredAddons sets addons as failed, sets addon reason as ConflictWithAlreadyRegisteredAddons
func (a *Entry) ConflictWithAlreadyRegisteredAddons(err error) {
	a.failed()
	a.setEntryStatus(v1alpha1.AddonConflictWithAlreadyRegisteredAddons, err.Error())
}

// RegisteringError sets addons as failed, sets addon reason as RegisteringError
func (a *Entry) RegisteringError(err error) {
	a.failed()
	a.setEntryStatus(v1alpha1.AddonRegisteringError, err.Error())
}

func (a *Entry) failed() {
	a.AddonWithCharts.Addon.Status = v1alpha1.AddonStatusFailed
}

func (a *Entry) setEntryStatus(reason v1alpha1.AddonStatusReason, message string) {
	a.AddonWithCharts.Addon.Reason = reason
	a.AddonWithCharts.Addon.Message = fmt.Sprintf(reason.Message(), message)
}

// limitMessage limits content of message field for AddonConfiguration which e.g. for fetching error
// could be very long. Full message occurs in controller log
func (a *Entry) limitMessage(content string) string {
	parts := strings.Split(content, ":")
	if len(parts) <= 4 {
		return content
	}

	return strings.Join(parts[:4], ":")
}
