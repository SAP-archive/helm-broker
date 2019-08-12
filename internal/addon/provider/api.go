package provider

import (
	"io"

	"github.com/kyma-project/helm-broker/internal"
)

// AddonClient defines abstraction to get and unmarshal raw index and addon into Models
type AddonClient interface {
	Cleanup() error
	GetCompleteAddon(entry internal.EntryDTO) (internal.AddonDTO, error)
	GetIndex() (*internal.IndexDTO, error)
}

// RepositoryGetter defines functionality for downloading addons from repository such as git, http, etc.
type RepositoryGetter interface {
	Cleanup() error
	IndexReader() (io.ReadCloser, error)
	AddonLoadInfo(name internal.Name, version internal.Version) (LoadType, string, error)
	AddonDocURL(name internal.Name, version internal.Version) (string, error)
}
