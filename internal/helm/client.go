package helm

import (
	"time"

	"github.com/kyma-project/helm-broker/internal"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
	"helm.sh/helm/v3/pkg/action"
	"helm.sh/helm/v3/pkg/chart"
	"helm.sh/helm/v3/pkg/release"
	"k8s.io/cli-runtime/pkg/genericclioptions"
	"k8s.io/client-go/rest"
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

	ns := string(namespace)
	cfg, err := c.getConfig(ns)
	if err != nil {
		return nil, errors.Wrap(err, "while getting config")
	}

	installAction := action.NewInstall(cfg)
	installAction.ReleaseName = string(releaseName)
	installAction.Namespace = ns
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
	cfg, err := c.getConfig(string(namespace))
	if err != nil {
		return errors.Wrap(err, "while getting config")
	}

	uninstallAction := action.NewUninstall(cfg)
	_, err = uninstallAction.Run(string(releaseName))
	if err != nil {
		return errors.Wrap(err, "while executing uninstall action")
	}

	return err
}

// ListReleases returns a list of helm releases in the given namespace
func (c *Client) ListReleases(namespace internal.Namespace) ([]*release.Release, error) {
	cfg, err := c.getConfig(string(namespace))
	if err != nil {
		return nil, errors.Wrap(err, "while getting config")
	}
	listAction := action.NewList(cfg)
	return listAction.Run()
}

func (c *Client) getConfig(namespace string) (*action.Configuration, error) {
	actionConfig := new(action.Configuration)
	// You can pass an empty string to all namespaces
	err := actionConfig.Init(c.newConfigFlags(namespace), namespace, c.helmDriver, c.log.Debugf)
	if err != nil {
		return nil, err
	}
	return actionConfig, nil
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
