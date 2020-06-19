package migration

import (
	"context"
	"time"

	"sigs.k8s.io/yaml"

	v1 "k8s.io/api/apps/v1"

	"github.com/kyma-project/helm-broker/internal/controller"
	"github.com/kyma-project/helm-broker/internal/storage"
	"github.com/kyma-project/helm-broker/pkg/apis/addons/v1alpha1"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

const (
	markLabel      = "helm-broker.kyma-project.io/migration"
	markLabelValue = "true"
)

type Executor struct {
	cli             *controller.CommonClient
	instStorage     storage.Instance
	deployName      string
	deployNamespace string
	log             logrus.FieldLogger
}

func New(cli client.Client, instStorage storage.Instance, deployName, deployNamespace string, log logrus.FieldLogger) *Executor {
	return &Executor{
		instStorage:     instStorage,
		cli:             controller.NewCommonClient(cli, log),
		deployName:      deployName,
		deployNamespace: deployNamespace,
		log:             log,
	}
}

func (e *Executor) Execute() error {
	deploy := &v1.Deployment{}
	err := e.cli.Get(context.Background(), client.ObjectKey{Name: e.deployName, Namespace: e.deployNamespace}, deploy)
	if err != nil {
		return errors.Wrap(err, "while getting deployment")
	}
	if e.isLabelSet(deploy) {
		e.log.Infof("Omitting migration due to label %s set", markLabel)
		return nil
	}

	errChan := make(chan error)
	migrations := []func() error{
		e.migrateCharts,
		e.migrateInstances,
	}

	for _, migration := range migrations {
		e.run(migration, errChan)
	}
	for range migrations {
		err := <-errChan
		if err != nil {
			return errors.Wrap(err, "while waiting for migration result")
		}
	}

	err = e.addLabel(deploy)
	if err != nil {
		return errors.Wrap(err, "while adding label")
	}

	return nil
}

func (e *Executor) run(migration func() error, errChan chan error) {
	go func() {
		err := migration()
		if err != nil {
			e.log.Errorf("while migration: %v", err)
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
		e.log.Infof("Reprocessing ClusterAddonConfiguration %s", cfg.Name)
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
		e.log.Infof("Reprocessing AddonConfiguration %s/%s", cfg.Namespace, cfg.Name)
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
			err := yaml.Unmarshal([]byte(instance.ReleaseInfo.Config.Raw), &instance.ReleaseInfo.ConfigValues)
			if err != nil {
				return errors.Wrap(err, "during config unmarshal")
			}
			instance.ReleaseInfo.Config = nil
			if instance.ReleaseInfo.Time != nil {
				instance.ReleaseInfo.ReleaseTime = time.Unix(instance.ReleaseInfo.Time.Seconds, int64(instance.ReleaseInfo.Time.Nanos))
			}
			_, err = e.instStorage.Upsert(instance)
			if err != nil {
				return errors.Wrap(err, "while upserting instance")
			}
		}
	}

	return nil
}

func (e *Executor) isLabelSet(s *v1.Deployment) bool {
	val, ok := s.Labels[markLabel]
	if !ok || val != markLabelValue {
		return false
	}
	return true
}

func (e *Executor) addLabel(s *v1.Deployment) error {
	if s.Labels == nil {
		s.Labels = make(map[string]string, 0)
	}
	s.Labels[markLabel] = markLabelValue
	err := e.cli.Update(context.Background(), s)
	if err != nil {
		return errors.Wrap(err, "while updating deployment")
	}
	return nil
}
