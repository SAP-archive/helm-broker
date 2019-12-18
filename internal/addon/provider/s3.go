package provider

import (
	getter "github.com/hashicorp/go-getter"
	"github.com/kyma-project/helm-broker/internal/rafter"
)

// S3GetterCreator provides functionality for loading addon from any S3 repository.
type S3GetterCreator struct {
	Cli    rafter.Client
	TmpDir string
}

// NewS3 returns new instance of RepositoryGetter with
// functionality for loading addon from any S3 repository.
func (g S3GetterCreator) NewS3(addr, src string) (RepositoryGetter, error) {
	return NewClientModeDirGetter(ClientModeDirGetterCfg{
		Underlying: &getter.S3Getter{},
		TmpDir:     g.TmpDir,
		Cli:        g.Cli,
		Addr:       addr,
		Src:        src,
	})
}
