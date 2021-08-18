package main

import (
	"context"
	"flag"
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"syscall"

	"github.com/gorilla/mux"
	"github.com/kyma-project/helm-broker/internal/bind"
	"github.com/kyma-project/helm-broker/internal/broker"
	"github.com/kyma-project/helm-broker/internal/config"
	"github.com/kyma-project/helm-broker/internal/health"
	"github.com/kyma-project/helm-broker/internal/helm"
	"github.com/kyma-project/helm-broker/internal/platform/logger"
	"github.com/kyma-project/helm-broker/internal/storage"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"github.com/sirupsen/logrus"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
)

func main() {
	verbose := flag.Bool("verbose", false, "specify if log verbosely loading configuration")
	flag.Parse()
	cfg, err := config.Load(*verbose)
	fatalOnError(err)

	// creates the in-cluster k8sConfig
	k8sConfig, err := newRestClientConfig(cfg.KubeconfigPath)
	fatalOnError(err)

	// creates the clientset
	clientset, err := kubernetes.NewForConfig(k8sConfig)
	fatalOnError(err)

	log := logger.New(&cfg.Logger)

	helmClient, err := helm.NewClient(k8sConfig, "secrets", log)
	fatalOnError(err)

	storageConfig := storage.ConfigList(cfg.Storage)
	sFact, err := storage.NewFactory(&storageConfig)
	fatalOnError(err)

	srv := broker.New(sFact.Addon(), sFact.Chart(), sFact.InstanceOperation(), sFact.BindOperation(), sFact.Instance(), sFact.InstanceBindData(),
		bind.NewRenderer(), bind.NewResolver(clientset.CoreV1()), helmClient, log)

	go health.NewBrokerProbes(fmt.Sprintf(":%d", cfg.StatusPort), storageConfig.ExtractEtcdURL()).Handle()
	go runMetricsServer(fmt.Sprintf(":%d", cfg.MetricsPort))

	startedCh := make(chan struct{})

	ctx, cancelFunc := context.WithCancel(context.Background())
	defer cancelFunc()
	cancelOnInterrupt(ctx, cancelFunc)

	fatalOnError(storageConfig.WaitForEtcdReadiness(log))

	err = srv.Run(ctx, fmt.Sprintf(":%d", cfg.Port), startedCh)
	fatalOnError(err)
}

func fatalOnError(err error) {
	if err != nil {
		logrus.Fatal(err.Error())
	}
}

// runMetricsServer lunches a separate server for metrics
func runMetricsServer(port string) {
	logrus.Infof("Start metrics server on %s port", port)

	var m = mux.NewRouter()
	m.Handle("/metrics", promhttp.Handler())

	srv := &http.Server{Addr: port, Handler: m}
	if err := srv.ListenAndServe(); err != http.ErrServerClosed {
		logrus.Errorf("Cannot run HTTP metrics server: %v", err)
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
