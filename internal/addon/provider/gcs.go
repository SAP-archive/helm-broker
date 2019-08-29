package provider

import (
	getter "github.com/hashicorp/go-getter"
	"github.com/kyma-project/helm-broker/internal/assetstore"
)

// GCSGetterCreator provides functionality for loading addon from any GCS repository.
type GCSGetterCreator struct {
	Cli    assetstore.Client
	TmpDir string
}

// NewGCS returns new instance of RepositoryGetter with
// functionality for loading addon from any GCS repository.
func (g GCSGetterCreator) NewGCS(addr, src string) (RepositoryGetter, error) {
	return NewClientModeDirGetter(ClientModeDirGetterCfg{
		Underlying: &getter.GCSGetter{},
		TmpDir:     g.TmpDir,
		Cli:        g.Cli,
		Addr:       addr,
		Src:        src,
	})
}
