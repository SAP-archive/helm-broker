package main

import (
	"context"
	"flag"
	"os"
	"os/signal"
	"syscall"

	"github.com/kyma-project/helm-broker/internal/config"
	"github.com/kyma-project/helm-broker/internal/platform/logger"
	"github.com/kyma-project/helm-broker/internal/storage"

	"github.com/sirupsen/logrus"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
)

func main() {
	verbose := flag.Bool("verbose", false, "specify if log verbosely loading configuration")
	flag.Parse()
	cfg, err := config.Load(*verbose)
	fatalOnError(err)

	// creates the in-cluster k8sConfig
	//k8sConfig, err := newRestClientConfig(cfg.KubeconfigPath)
	//fatalOnError(err)

	// creates the clientset
	//clientset, err := kubernetes.NewForConfig(k8sConfig)
	//fatalOnError(err)
	//
	log := logger.New(&cfg.Logger)

	//helmClient := helm.NewClient(cfg.Helm, log)

	storageConfig := storage.ConfigList(cfg.Storage)
	sFact, err := storage.NewFactory(&storageConfig)
	fatalOnError(err)

	instances, err := sFact.Instance().GetAll()
	fatalOnError(err)

	for _, inst := range instances {
		log.Infof("ID: %s, ns: %s, name: %s, rev: %d", inst.ID, inst.Namespace, inst.ReleaseName, inst.ReleaseInfo.Revision)
	}

}

func fatalOnError(err error) {
	if err != nil {
		logrus.Fatal(err.Error())
	}
}

// cancelOnInterrupt calls cancel func when os.Interrupt or SIGTERM is received
func cancelOnInterrupt(ctx context.Context, cancel context.CancelFunc) {
	c := make(chan os.Signal, 1)
	signal.Notify(c, os.Interrupt, syscall.SIGTERM)
	go func() {
		select {
		case <-ctx.Done():
		case <-c:
			cancel()
		}
	}()
}

func newRestClientConfig(kubeConfigPath string) (*rest.Config, error) {
	if kubeConfigPath != "" {
		return clientcmd.BuildConfigFromFlags("", kubeConfigPath)
	}

	return rest.InClusterConfig()
}
