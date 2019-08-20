package provider

import (
	"io"

	"github.com/kyma-project/helm-broker/internal"
)

// AddonClient defines abstraction to get and unmarshal raw index and addon into Models
type AddonClient interface {
	Cleanup() error
	GetCompleteAddon(entry internal.IndexEntry) (internal.AddonWithCharts, error)
	GetIndex() (*internal.Index, error)
}

// RepositoryGetter defines functionality for downloading addons from repository such as git, http, etc.
type RepositoryGetter interface {
	Cleanup() error
	IndexReader() (io.ReadCloser, error)
	AddonLoadInfo(name internal.AddonName, version internal.AddonVersion) (LoadType, string, error)
	AddonDocURL(name internal.AddonName, version internal.AddonVersion) (string, error)
}
