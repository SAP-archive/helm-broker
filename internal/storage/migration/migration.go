package migration

import (
	"context"

	"github.com/kyma-project/helm-broker/internal/controller"
	"github.com/kyma-project/helm-broker/internal/storage"
	"github.com/kyma-project/helm-broker/pkg/apis/addons/v1alpha1"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
	"k8s.io/client-go/kubernetes/scheme"
	"k8s.io/client-go/rest"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

type Executor struct {
	cli         *controller.CommonClient
	instStorage storage.Instance
	log         logrus.FieldLogger
}

func New(cfg *rest.Config, instStorage storage.Instance, log logrus.FieldLogger) (*Executor, error) {
	cli, err := client.New(cfg, client.Options{
		Scheme: scheme.Scheme,
	})
	if err != nil {
		return nil, errors.Wrap(err, "while creating a client")
	}

	return &Executor{
		instStorage: instStorage,
		cli:         controller.NewCommonClient(cli, log),
		log:         log,
	}, nil
}

func (e *Executor) Execute() error {
	errChan := make(chan error)
	migrations := []func() error{
		e.migrateCharts,
		e.migrateInstances,
	}

	for _, migration := range migrations {
		e.async(migration, errChan)
	}
	for range migrations {
		err := <-errChan
		if err != nil {
			return errors.Wrap(err, "while waiting for migration result")
		}
	}

	return nil
}

func (e *Executor) async(migration func() error, errChan chan error) {
	go func() {
		err := migration()
		if err != nil {
			e.log.Error("while migration: %v", err)
			errChan <- err
		}
		errChan <- nil
	}()
}

func (e *Executor) migrateCharts() error {
	clusterAddonsList := v1alpha1.ClusterAddonsConfigurationList{}
	err := e.cli.List(context.TODO(), &clusterAddonsList)
	if err != nil {
		return errors.Wrap(err, "while listing cluster addon configurations")
	}
	for _, cfg := range clusterAddonsList.Items {
		e.log.Info("Reprocessing ClusterAddonConfiguration %s", cfg.Name)
		err := e.cli.ReprocessRequest(cfg.Name)
		if err != nil {
			return errors.Wrap(err, "error on cluster addon reprocess")
		}
	}

	addonsList := v1alpha1.AddonsConfigurationList{}
	err = e.cli.List(context.TODO(), &addonsList)
	if err != nil {
		return errors.Wrap(err, "while listing addon configurations")
	}
	for _, cfg := range addonsList.Items {
		e.log.Info("Reprocessing AddonConfiguration %s/%s", cfg.Namespace, cfg.Name)
		e.cli.SetNamespace(cfg.Namespace)
		err := e.cli.ReprocessRequest(cfg.Name)
		if err != nil {
			return errors.Wrap(err, "error on addon reprocess")
		}
	}

	return nil
}

func (e *Executor) migrateInstances() error {
	instances, err := e.instStorage.GetAll()
	if err != nil {
		return errors.Wrap(err, "while listing instances")
	}
	for _, instance := range instances {
		if instance.ReleaseInfo.Config != nil {
			e.log.Infof("Migrating instance %s in storage", instance.ID)
			// convert to V3
			instance.ReleaseInfo.Config = nil
		}
	}

	return nil
}
