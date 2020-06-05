package controller

import (
	"github.com/Masterminds/semver"
	"github.com/kyma-project/helm-broker/internal"
	"github.com/kyma-project/helm-broker/internal/addon/provider"
	"github.com/kyma-project/helm-broker/pkg/apis/addons/v1alpha1"
	"helm.sh/helm/v3/pkg/chart"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
)

// NamespacedService represents service layer which can be applied to both namespace scoped and cluster-wide resources
type NamespacedService interface {
	SetNamespace(namespace string)
}

//go:generate mockery -name=addonStorage -output=automock -outpkg=automock -case=underscore
type addonStorage interface {
	Get(internal.Namespace, internal.AddonName, semver.Version) (*internal.Addon, error)
	Upsert(internal.Namespace, *internal.Addon) (replace bool, err error)
	Remove(internal.Namespace, internal.AddonName, semver.Version) error
	FindAll(internal.Namespace) ([]*internal.Addon, error)
}

//go:generate mockery -name=chartStorage -output=automock -outpkg=automock -case=underscore
type chartStorage interface {
	Upsert(internal.Namespace, *chart.Chart) (replace bool, err error)
	Remove(internal.Namespace, internal.ChartName, semver.Version) error
}

//go:generate mockery -name=addonGetterFactory -output=automock -outpkg=automock -case=underscore
type addonGetterFactory interface {
	NewGetter(rawURL, instPath string) (provider.AddonClient, error)
}

//go:generate mockery -name=addonGetter -output=automock -outpkg=automock -case=underscore
type addonGetter interface {
	Cleanup() error
	GetCompleteAddon(entry internal.IndexEntry) (internal.AddonWithCharts, error)
	GetIndex() (*internal.Index, error)
}

//go:generate mockery -name=docsProvider -output=automock -outpkg=automock -case=underscore
type docsProvider interface {
	NamespacedService
	EnsureAssetGroup(addon *internal.Addon) error
	EnsureAssetGroupRemoved(id string) error
}

//go:generate mockery -name=brokerFacade -output=automock -outpkg=automock -case=underscore
type brokerFacade interface {
	NamespacedService
	Create() error
	Exist() (bool, error)
	Delete() error
}

//go:generate mockery -name=brokerSyncer -output=automock -outpkg=automock -case=underscore
type brokerSyncer interface {
	NamespacedService
	Sync() error
}

//go:generate mockery -name=commonClient -output=automock -outpkg=automock -case=underscore
type commonClient interface {
	NamespacedService
	UpdateConfiguration(*internal.CommonAddon) error
	UpdateConfigurationStatus(*internal.CommonAddon) error
	ListConfigurations() ([]internal.CommonAddon, error)
	ReprocessRequest(addonName string) error
	IsNamespaceScoped() bool
}

//go:generate mockery -name=templateService -output=automock -outpkg=automock -case=underscore
type templateService interface {
	NamespacedService
	TemplateURL(repository v1alpha1.SpecRepository) (string, error)
}

type commonReconciler interface {
	Reconcile(addon *internal.CommonAddon, trace string) (reconcile.Result, error)
	SetWorkingNamespace(namespace string)
}

type instanceChecker interface {
	AnyServiceInstanceExistsForNamespacedServiceBroker(namespace string) (bool, error)
	AnyServiceInstanceExistsForClusterServiceBroker() (bool, error)
}
