package controller

import (
	"fmt"
	"time"

	"github.com/kubernetes-sigs/service-catalog/pkg/apis/servicecatalog/v1beta1"
	"github.com/kyma-project/helm-broker/internal/addon"
	"github.com/kyma-project/helm-broker/internal/addon/provider"
	"github.com/kyma-project/helm-broker/internal/config"
	"github.com/kyma-project/helm-broker/internal/controller/broker"
	"github.com/kyma-project/helm-broker/internal/controller/docs"
	"github.com/kyma-project/helm-broker/internal/controller/instance"
	"github.com/kyma-project/helm-broker/internal/controller/repository"
	"github.com/kyma-project/helm-broker/internal/rafter"
	"github.com/kyma-project/helm-broker/internal/storage"
	"github.com/kyma-project/helm-broker/pkg/apis"
	rafterv1beta1 "github.com/kyma-project/rafter/pkg/apis/rafter/v1beta1"
	"github.com/sirupsen/logrus"
	"k8s.io/client-go/rest"
	"sigs.k8s.io/controller-runtime/pkg/manager"
)

// SetupAndStartController creates and starts the controller
func SetupAndStartController(cfg *rest.Config, ctrCfg *config.ControllerConfig, metricsAddr string, sFact storage.Factory, uploadClient rafter.Client, lg *logrus.Entry) manager.Manager {
	// Create a new Cmd to provide shared dependencies and start components
	lg.Info("Setting up manager")
	var mgr manager.Manager
	fatalOnError(waitAtMost(func() (bool, error) {
		newMgr, err := manager.New(cfg, manager.Options{MetricsBindAddress: metricsAddr})
		if err != nil {
			return false, err
		}
		mgr = newMgr
		return true, nil
	}, time.Minute*3), "while setting up a manager")

	lg.Info("Registering Components.")

	// Setup Scheme for all resources
	lg.Info("Setting up schemes")
	fatalOnError(apis.AddToScheme(mgr.GetScheme()), "while adding AC scheme")
	fatalOnError(v1beta1.AddToScheme(mgr.GetScheme()), "while adding SC scheme")
	fatalOnError(rafterv1beta1.AddToScheme(mgr.GetScheme()), "while adding RAFTER scheme")

	// Setup dependencies

	var dtProvider docsProvider
	var cdtProvider docsProvider

	dtProvider = docs.NewProvider(mgr.GetClient(), lg)
	cdtProvider = docs.NewClusterProvider(mgr.GetClient(), lg)
	if !ctrCfg.DocumentationEnabled {
		dtProvider = &docs.DummyProvider{}
		cdtProvider = &docs.DummyProvider{}
	}
	sbSyncer := broker.NewBrokerSyncer(mgr.GetClient(), lg)
	csbSyncer := broker.NewClusterBrokerSyncer(mgr.GetClient(), ctrCfg.ClusterServiceBrokerName, lg)
	sbFacade := broker.NewBrokersFacade(mgr.GetClient(), ctrCfg.Namespace, ctrCfg.ServiceName, lg)
	csbFacade := broker.NewClusterBrokersFacade(mgr.GetClient(), ctrCfg.Namespace, ctrCfg.ServiceName, ctrCfg.ClusterServiceBrokerName, lg)

	templateService := repository.NewTemplate(mgr.GetClient())

	var (
		gitGetterFactory = provider.GitGetterCreator{Cli: uploadClient, TmpDir: ctrCfg.TmpDir}
		hgGetterFactory  = provider.HgGetterCreator{Cli: uploadClient, TmpDir: ctrCfg.TmpDir}
		s3GetterFactory  = provider.S3GetterCreator{Cli: uploadClient, TmpDir: ctrCfg.TmpDir}
	)

	allowedGetters := map[string]provider.Provider{
		"git":   gitGetterFactory.NewGit,
		"hg":    hgGetterFactory.NewHg,
		"s3":    s3GetterFactory.NewS3,
		"https": provider.NewHTTP,
	}
	if ctrCfg.DevelopMode {
		lg.Infof("Enabling support for HTTP protocol because DevelopMode is set to true.")
		allowedGetters["http"] = provider.NewHTTP
	} else {
		lg.Infof("Disabling support for HTTP protocol because DevelopMode is set to false.")
	}

	addonGetterFactory, err := provider.NewClientFactory(allowedGetters, addon.NewLoader(ctrCfg.TmpDir, lg), ctrCfg.DocumentationEnabled, lg)
	fatalOnError(err, "cannot setup addon getter")

	instChecker := instance.New(mgr.GetClient(), ctrCfg.ClusterServiceBrokerName)

	// Creating controllers
	lg.Info("Setting up controller")
	acReconcile := NewReconcileAddonsConfiguration(mgr, addonGetterFactory, sFact.Chart(), sFact.Addon(), sbFacade, dtProvider, sbSyncer, templateService, ctrCfg.TmpDir, ctrCfg.ReprocessOnErrorDuration, lg)
	acController := NewAddonsConfigurationController(acReconcile)
	err = acController.Start(mgr)
	fatalOnError(err, "unable to start AddonsConfigurationController")

	cacReconcile := NewReconcileClusterAddonsConfiguration(mgr, addonGetterFactory, sFact.Chart(), sFact.Addon(), csbFacade, cdtProvider, csbSyncer, templateService, ctrCfg.TmpDir, ctrCfg.ReprocessOnErrorDuration, lg)
	cacController := NewClusterAddonsConfigurationController(cacReconcile)
	err = cacController.Start(mgr)
	fatalOnError(err, "unable to start ClusterAddonsConfigurationController")

	bController := NewBrokerController(instChecker, mgr.GetClient(), broker.NewBrokersFacade(mgr.GetClient(), ctrCfg.Namespace, ctrCfg.ServiceName, lg))
	err = bController.Start(mgr)
	fatalOnError(err, "unable to start BrokerController")

	cbController := NewClusterBrokerController(instChecker, mgr.GetClient(), csbFacade, ctrCfg.ClusterServiceBrokerName)
	err = cbController.Start(mgr)
	fatalOnError(err, "unable to start ClusterBrokerController")

	return mgr
}

func fatalOnError(err error, msg string) {
	if err != nil {
		logrus.Fatalf("%s: %s", msg, err.Error())
	}
}

func waitAtMost(fn func() (bool, error), duration time.Duration) error {
	timeout := time.After(duration)
	tick := time.Tick(time.Second * 5)

	for {
		ok, err := fn()
		select {
		case <-timeout:
			return fmt.Errorf("waiting for resource failed in given timeout %f second(s)", duration.Seconds())
		case <-tick:
			if err != nil {
				logrus.Println(err)
			} else if ok {
				return nil
			}
		}
	}
}
