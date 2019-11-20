package provider

import (
	getter "github.com/hashicorp/go-getter"
	"github.com/kyma-project/helm-broker/internal/rafter"
)

// HgGetterCreator provides functionality for loading addon from any Mercurial repository.
type HgGetterCreator struct {
	Cli    rafter.Client
	TmpDir string
}

// NewHg returns new instance of RepositoryGetter with
// functionality for loading addon from any  Mercurial repository.
func (g HgGetterCreator) NewHg(addr, src string) (RepositoryGetter, error) {
	return NewClientModeDirGetter(ClientModeDirGetterCfg{
		Protocol:   "hg",
		Underlying: &getter.HgGetter{},
		TmpDir:     g.TmpDir,
		Cli:        g.Cli,
		Addr:       addr,
		Src:        src,
	})
}
