package main

import (
	"flag"
	"fmt"

	envs "github.com/kyma-project/helm-broker/internal/config"
	"github.com/kyma-project/helm-broker/internal/controller"
	"github.com/kyma-project/helm-broker/internal/health"
	"github.com/kyma-project/helm-broker/internal/platform/logger"
	"github.com/kyma-project/helm-broker/internal/rafter"
	"github.com/kyma-project/helm-broker/internal/storage"
	"github.com/kyma-project/helm-broker/internal/storage/migration"

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

	// TODO: switch to native implementation after merge: https://github.com/kubernetes-sigs/controller-runtime/pull/419
	go health.NewControllerProbes(fmt.Sprintf(":%d", ctrCfg.StatusPort), storageConfig.ExtractEtcdURL(), mgr.GetClient(), ctrCfg.Namespace).Handle()

	fatalOnError(storageConfig.WaitForEtcdReadiness(), "while waiting for etcd to be ready")

	migrate, err := migration.New(cfg, sFact.Instance(), lg)
	fatalOnError(err, "while creating migration")

	err = migrate.Execute()
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
