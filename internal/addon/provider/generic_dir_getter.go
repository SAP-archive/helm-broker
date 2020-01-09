package provider

import (
	"fmt"
	"io"
	"io/ioutil"
	"net/url"
	"os"
	"path"
	"strings"

	getter "github.com/hashicorp/go-getter"
	"github.com/kyma-project/helm-broker/internal"
	"github.com/kyma-project/helm-broker/internal/rafter"
	"github.com/mholt/archiver"
	exerr "github.com/pkg/errors"
	"k8s.io/apimachinery/pkg/util/rand"
)

// ClientModeDirGetter downloads a directory. In this mode, dst must be
// a directory path (doesn't have to exist). Src must point to an
// archive or directory (such as in s3).
type ClientModeDirGetter struct {
	underlying getter.Getter

	dst          string
	idxPath      string
	addonDirPath string
	docsURL      string

	cli      rafter.Client
	tmpDir   string
	protocol string
}

// ClientModeDirGetterCfg holds input parameters for ClientModeDirGetter constructor
type ClientModeDirGetterCfg struct {
	Cli        rafter.Client
	TmpDir     string
	Underlying getter.Getter
	Addr       string
	Src        string
	Protocol   string
}

// NewClientModeDirGetter returns new instance of ClientModeDirGetter
func NewClientModeDirGetter(in ClientModeDirGetterCfg) (RepositoryGetter, error) {
	finalDst := path.Join(in.Src, rand.String(10))
	upstreamAddr, indexPath := getter.SourceDirSubdir(in.Addr)
	if indexPath == "" {
		return nil, fmt.Errorf("index path needs to be provided. Check documentation about using %s protocol", in.Protocol)
	}

	ru, err := url.Parse(upstreamAddr)
	if err != nil {
		return nil, err
	}

	if err = in.Underlying.Get(finalDst, ru); err != nil {
		return nil, err
	}

	return &ClientModeDirGetter{
		protocol:   in.Protocol,
		underlying: in.Underlying,
		tmpDir:     in.TmpDir,
		cli:        in.Cli,

		idxPath:      indexPath,
		dst:          finalDst,
		addonDirPath: strings.TrimRight(indexPath, path.Base(indexPath)),
	}, nil
}

// Cleanup  removes folder where repository was cloned.
func (g *ClientModeDirGetter) Cleanup() error {
	return os.RemoveAll(g.dst)
}

// IndexReader returns index reader
func (g *ClientModeDirGetter) IndexReader() (io.ReadCloser, error) {
	return os.Open(path.Join(g.dst, g.idxPath))
}

// AddonLoadInfo returns information how to load addon
func (g *ClientModeDirGetter) AddonLoadInfo(name internal.AddonName, version internal.AddonVersion) (LoadType, string, error) {
	var (
		addonDirName = fmt.Sprintf("%s-%s", name, version)
		pathToAddon  = path.Join(g.dst, g.addonDirPath, addonDirName)
	)

	return DirectoryLoadType, pathToAddon, nil
}

// AddonDocURL returns url for addon documentation
func (g *ClientModeDirGetter) AddonDocURL(name internal.AddonName, version internal.AddonVersion) (string, error) {
	var (
		addonDirName = fmt.Sprintf("%s-%s", name, version)
		pathToAddon  = path.Join(g.dst, g.addonDirPath, addonDirName)
		pathToDocs   = path.Join(pathToAddon, "/docs")
		pathToTgz    = fmt.Sprintf("%s/docs-%s.tgz", g.tmpDir, addonDirName)
	)

	_, err := os.Stat(pathToDocs)
	switch {
	case err == nil:
	case os.IsNotExist(err):
		return "", nil
	default:
		return "", exerr.Wrap(err, "while checking if doc exists")
	}

	tar := archiver.NewTarGz()
	tar.OverwriteExisting = true
	err = tar.Archive([]string{pathToDocs}, pathToTgz)
	if err != nil {
		return "", exerr.Wrapf(err, "while creating archive '%s'", pathToTgz)
	}
	defer func() {
		os.Remove(pathToTgz)
	}()

	file, err := os.Open(pathToTgz)
	if err != nil {
		return "", exerr.Wrapf(err, "while opening file '%s'", pathToTgz)
	}
	defer func() {
		file.Close()
	}()

	docs, err := ioutil.ReadAll(file)
	if err != nil {
		return "", exerr.Wrapf(err, "while reading file '%s'", file.Name())
	}

	uploaded, err := g.cli.Upload(pathToTgz, docs)
	if err != nil {
		return "", exerr.Wrapf(err, "while uploading Tgz '%s' to uploadService", pathToTgz)
	}

	return uploaded.RemotePath, nil
}
