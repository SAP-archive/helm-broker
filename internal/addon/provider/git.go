package provider

import (
	getter "github.com/hashicorp/go-getter"
	"github.com/kyma-project/helm-broker/internal/rafter"
)

// GitGetterCreator provides functionality for loading addon from any Git repository.
type GitGetterCreator struct {
	Cli    rafter.Client
	TmpDir string
}

// NewGit returns new instance of RepositoryGetter with
// functionality for loading addon from any Git repository.
func (g GitGetterCreator) NewGit(addr, src string) (RepositoryGetter, error) {
	return NewClientModeDirGetter(ClientModeDirGetterCfg{
		Protocol:   "git",
		Underlying: &getter.GitGetter{},
		TmpDir:     g.TmpDir,
		Cli:        g.Cli,
		Addr:       addr,
		Src:        src,
	})
}
