package helm

import (
	"fmt"
	"time"

	"github.com/kyma-project/helm-broker/internal"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
	"helm.sh/helm/v3/pkg/action"
	"helm.sh/helm/v3/pkg/chart"
	"helm.sh/helm/v3/pkg/kube"
	"helm.sh/helm/v3/pkg/release"
	"helm.sh/helm/v3/pkg/storage"
	"helm.sh/helm/v3/pkg/storage/driver"
	"k8s.io/cli-runtime/pkg/genericclioptions"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"k8s.io/kubectl/pkg/cmd/util"
)

const (
	// MaxHistory specifies the maximum number of historical releases that will
	// be retained, including the most recent release. Values of 0 or less are
	// ignored (meaning no limits are imposed).
	maxHistory = 1
)

// Client implements a Helm client compatible with Helm3
type Client struct {
	log        logrus.FieldLogger
	helmDriver string
	restConfig *rest.Config

	installingTimeout time.Duration
}

func NewClient(restConfig *rest.Config, helmDriver string, log logrus.FieldLogger) (*Client, error) {
	if helmDriver == "" {
		helmDriver = "secrets"
	}
	return &Client{
		log:               log,
		helmDriver:        helmDriver,
		restConfig:        restConfig,
		installingTimeout: time.Hour,
	}, nil
}

func (c *Client) Install(chrt *chart.Chart, values internal.ChartValues, releaseName internal.ReleaseName, namespace internal.Namespace) (*release.Release, error) {
	c.log.Infof("Installing chart with release name [%s], namespace: [%s]", releaseName, namespace)

	aCfg, err := c.provideActionConfig(namespace)
	if err != nil {
		return nil, err
	}
	installAction := action.NewInstall(aCfg)
	installAction.ReleaseName = string(releaseName)
	installAction.Namespace = string(namespace)
	installAction.Wait = true
	installAction.Timeout = c.installingTimeout
	installAction.CreateNamespace = true // https://v3.helm.sh/docs/faq/#automatically-creating-namespaces

	release, err := installAction.Run(chrt, values)
	if err != nil {
		return nil, errors.Wrapf(err, "while installing release from chart with name [%s] in namespace [%s]", releaseName, namespace)
	}

	return release, nil
}

// Delete is deleting release of the chart
func (c *Client) Delete(releaseName internal.ReleaseName, namespace internal.Namespace) error {
	c.log.Infof("Deleting chart with release name [%s], namespace: [%s]", releaseName, namespace)
	aCfg, err := c.provideActionConfig(namespace)
	if err != nil {
		return err
	}

	uninstallAction := action.NewUninstall(aCfg)
	_, err = uninstallAction.Run(string(releaseName))
	if err != nil {
		return errors.Wrap(err, "while executing uninstall action")
	}

	return err
}

// ListReleases returns a list of helm releases in the given namespace
func (c *Client) ListReleases(namespace internal.Namespace) ([]*release.Release, error) {
	aCfg, err := c.provideActionConfig(namespace)
	if err != nil {
		return nil, err
	}
	listAction := action.NewList(aCfg)
	return listAction.Run()
}

func (c *Client) provideActionConfig(namespace internal.Namespace) (*action.Configuration, error) {
	restClientGetter := c.newConfigFlags(string(namespace))
	kubeClient := &kube.Client{
		Factory: util.NewFactory(restClientGetter),
		Log:     c.log.Debugf,
	}
	client, err := kubeClient.Factory.KubernetesClientSet()
	if err != nil {
		return nil, errors.Wrap(err, "while getting kube client")
	}

	store, err := c.provideStorage(client, string(namespace))
	if err != nil {
		return nil, errors.Wrap(err, "while getting helm storage")
	}

	return &action.Configuration{
		RESTClientGetter: restClientGetter,
		Releases:         store,
		KubeClient:       kubeClient,
		Log:              c.log.Debugf,
	}, nil
}

func (c *Client) provideStorage(client *kubernetes.Clientset, namespace string) (*storage.Storage, error) {
	switch c.helmDriver {
	case "secret", "secrets", "":
		sec := driver.NewSecrets(client.CoreV1().Secrets(namespace))
		sec.Log = c.log.Debugf
		s := storage.Init(sec)
		s.MaxHistory = maxHistory
		return s, nil
	case "configmap", "configmaps":
		cm := driver.NewConfigMaps(client.CoreV1().ConfigMaps(namespace))
		cm.Log = c.log.Debugf
		s := storage.Init(cm)
		s.MaxHistory = maxHistory
		return s, nil
	case "memory":
		m := driver.NewMemory()
		s := storage.Init(m)
		s.MaxHistory = maxHistory
		return s, nil
	default:
		return nil, fmt.Errorf("unsupported helm driver '%s'", c.helmDriver)
	}
}

func (c *Client) newConfigFlags(namespace string) *genericclioptions.ConfigFlags {
	return &genericclioptions.ConfigFlags{
		Namespace:   &namespace,
		APIServer:   &c.restConfig.Host,
		CAFile:      &c.restConfig.CAFile,
		BearerToken: &c.restConfig.BearerToken,
	}
}

// Sets installing timeout, used in the integration tests
func (c *Client) SetInstallingTimeout(timeout time.Duration) {
	c.installingTimeout = timeout
}
