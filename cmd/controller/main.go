package main

import (
	"flag"
	"fmt"

	envs "github.com/kyma-project/helm-broker/internal/config"
	"github.com/kyma-project/helm-broker/internal/controller"
	"github.com/kyma-project/helm-broker/internal/health"
	"github.com/kyma-project/helm-broker/internal/migration"
	"github.com/kyma-project/helm-broker/internal/platform/logger"
	"github.com/kyma-project/helm-broker/internal/rafter"
	"github.com/kyma-project/helm-broker/internal/storage"
	"k8s.io/client-go/kubernetes/scheme"
	"sigs.k8s.io/controller-runtime/pkg/client"

	"github.com/sirupsen/logrus"
	"sigs.k8s.io/controller-runtime/pkg/client/config"
	"sigs.k8s.io/controller-runtime/pkg/runtime/signals"
)

func main() {
	var metricsAddr string
	flag.StringVar(&metricsAddr, "metrics-addr", ":8080", "The address the metric endpoint binds to.")
	verbose := flag.Bool("verbose", false, "specify if lg verbosely loading configuration")
	flag.Parse()

	ctrCfg, err := envs.LoadControllerConfig(*verbose)
	fatalOnError(err, "while loading config")

	storageConfig := storage.ConfigList(ctrCfg.Storage)
	sFact, err := storage.NewFactory(&storageConfig)
	fatalOnError(err, "while setting up a storage")

	lg := logger.New(&ctrCfg.Logger)

	// Get a config to talk to the apiserver
	lg.Info("Setting up client for manager")
	cfg, err := config.GetConfig()
	fatalOnError(err, "while setting up a client")

	uploadClient := rafter.NewClient(ctrCfg.UploadServiceURL, lg)
	mgr := controller.SetupAndStartController(cfg, ctrCfg, metricsAddr, sFact, uploadClient, lg)

	fatalOnError(storageConfig.WaitForEtcdReadiness(lg), "while waiting for etcd to be ready")

	// TODO: switch to native implementation after merge: https://github.com/kubernetes-sigs/controller-runtime/pull/419
	go health.NewControllerProbes(fmt.Sprintf(":%d", ctrCfg.StatusPort), storageConfig.ExtractEtcdURL(), mgr.GetClient(), ctrCfg.Namespace).Handle()

	cli, err := client.New(cfg, client.Options{
		Scheme: scheme.Scheme,
	})
	fatalOnError(err, "while creating client")

	err = migration.New(cli, sFact.Instance(), ctrCfg.ServiceName, ctrCfg.Namespace, lg.WithField("service", "migration")).Execute()
	fatalOnError(err, "while executing migration")

	lg.Info("Starting the Controller.")
	err = mgr.Start(signals.SetupSignalHandler())
	fatalOnError(err, "unable to run the manager")
}

func fatalOnError(err error, msg string) {
	if err != nil {
		logrus.Fatalf("%s: %s", msg, err.Error())
	}
}
