package provider

import (
	"io"

	"github.com/kyma-project/helm-broker/internal/addons"
)

// AddonClient defines abstraction to get and unmarshal raw index and addon into Models
type AddonClient interface {
	Cleanup() error
	GetCompleteAddon(entry addons.EntryDTO) (addons.AddonDTO, error)
	GetIndex() (*addons.IndexDTO, error)
}

// RepositoryGetter defines functionality for downloading addons from repository such as git, http, etc.
type RepositoryGetter interface {
	Cleanup() error
	IndexReader() (io.ReadCloser, error)
	AddonLoadInfo(name addons.Name, version addons.Version) (LoadType, string, error)
	AddonDocURL(name addons.Name, version addons.Version) (string, error)
}
