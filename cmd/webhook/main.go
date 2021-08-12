package main

import (
	"github.com/kyma-project/helm-broker/internal/webhook"
	"github.com/sirupsen/logrus"
	clientgoscheme "k8s.io/client-go/kubernetes/scheme"
	"sigs.k8s.io/controller-runtime/pkg/client/config"
	"sigs.k8s.io/controller-runtime/pkg/manager"
	"sigs.k8s.io/controller-runtime/pkg/runtime/signals"
	k8sWebhook "sigs.k8s.io/controller-runtime/pkg/webhook"
)

func main() {
	lg := logrus.New()
	// Get a config to talk to the apiserver
	lg.Info("Setting up client for manager")
	cfg, err := config.GetConfig()
	fatalOnError(err, "while setting up a client")

	mgr, err := manager.New(cfg, manager.Options{Port: 8443,
		CertDir: "/var/run/webhook"})
	if err != nil {
		fatalOnError(err, "while set up overall controller manager for webhook server")
	}

	mgr.GetWebhookServer().Register(
		"/hb-pod-mutating",
		&k8sWebhook.Admission{Handler: webhook.NewWebhookHandler(mgr.GetClient(), lg.WithField("webhook", "pod-mutating"))})

	fatalOnError(clientgoscheme.AddToScheme(mgr.GetScheme()), "while adding clientgo scheme")

	lg.Info("Starting the Controller.")
	err = mgr.Start(signals.SetupSignalHandler())
	fatalOnError(err, "unable to run the manager")
}

func fatalOnError(err error, msg string) {
	if err != nil {
		logrus.Fatalf("%s: %s", msg, err.Error())
	}
}
