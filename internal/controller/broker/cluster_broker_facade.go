package broker

import (
	"fmt"
	"time"

	"k8s.io/apimachinery/pkg/util/wait"

	"context"

	"github.com/kubernetes-sigs/service-catalog/pkg/apis/servicecatalog/v1beta1"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
	k8serrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

type clusterBrokerSyncer interface {
	Sync() error
}

// ClusterFacade is responsible for creation k8s objects for namespaced broker
type ClusterFacade struct {
	client            client.Client
	workingNamespace  string
	serviceName       string
	log               logrus.FieldLogger
	clusterBrokerName string

	clusterBrokerSyncer clusterBrokerSyncer
}

// NewClusterBrokersFacade returns facade
func NewClusterBrokersFacade(client client.Client, workingNamespace, serviceName, clusterBrokerName string, log logrus.FieldLogger) *ClusterFacade {
	return &ClusterFacade{
		client:            client,
		workingNamespace:  workingNamespace,
		clusterBrokerName: clusterBrokerName,
		serviceName:       serviceName,
		log:               log.WithField("service", "cluster-broker-facade"),
	}
}

// Create creates ClusterServiceBroker
func (f *ClusterFacade) Create() error {
	f.log.Infof("- creating ClusterServiceBroker %s", f.clusterBrokerName)
	svcURL := fmt.Sprintf("http://%s.%s.svc.cluster.local", f.serviceName, f.workingNamespace)

	err := wait.PollImmediate(time.Second, time.Second*30, func() (bool, error) {
		_, err := f.createClusterServiceBroker(svcURL)
		if err != nil && !k8serrors.IsAlreadyExists(err) {
			f.log.Errorf("creation of ClusterServiceBroker %s results in error: [%s]", f.clusterBrokerName, err)
			return false, nil
		}
		return true, nil
	})
	if err != nil {
		return errors.Wrap(err, "while waiting for cluster service broker creation")
	}
	return nil
}

// Delete removes ClusterServiceBroker. Errors don't stop execution of method. NotFound errors are ignored.
func (f *ClusterFacade) Delete() error {
	csb := &v1beta1.ClusterServiceBroker{
		ObjectMeta: metav1.ObjectMeta{
			Name: f.clusterBrokerName,
		},
	}
	f.log.Infof("- deleting ClusterServiceBroker %s", f.clusterBrokerName)
	err := f.client.Delete(context.Background(), csb)
	switch {
	case k8serrors.IsNotFound(err):
		return nil
	case err != nil:
		f.log.Warnf("Deletion of ClusterServiceBroker %s results in error: [%s].", f.clusterBrokerName, err)
	}
	return err

}

// Exist check if ClusterServiceBroker exists.
func (f *ClusterFacade) Exist() (bool, error) {
	err := f.client.Get(context.Background(), types.NamespacedName{Name: f.clusterBrokerName}, &v1beta1.ClusterServiceBroker{})
	switch {
	case k8serrors.IsNotFound(err):
		return false, nil
	case err != nil:
		return false, errors.Wrapf(err, "while checking if ClusterServiceBroker [%s] exists", f.clusterBrokerName)
	}

	return true, nil
}

// createServiceBroker returns just created or existing ClusterServiceBroker
func (f *ClusterFacade) createClusterServiceBroker(svcURL string) (*v1beta1.ClusterServiceBroker, error) {
	url := fmt.Sprintf("%s/cluster", svcURL)
	broker := &v1beta1.ClusterServiceBroker{
		ObjectMeta: metav1.ObjectMeta{
			Name: f.clusterBrokerName,
		},
		Spec: v1beta1.ClusterServiceBrokerSpec{
			CommonServiceBrokerSpec: v1beta1.CommonServiceBrokerSpec{
				URL:            url,
				RelistRequests: 1,
			},
		},
	}

	err := f.client.Create(context.Background(), broker)
	if k8serrors.IsAlreadyExists(err) {
		f.log.Infof("ClusterServiceBroker [%s] already exist. Attempt to get resource.", broker.Name)
		createdBroker := &v1beta1.ClusterServiceBroker{}
		err = f.client.Get(context.Background(), types.NamespacedName{Name: f.clusterBrokerName}, createdBroker)
		return createdBroker, err
	}

	return broker, err
}

// SetNamespace sets service's working namespace
func (f *ClusterFacade) SetNamespace(namespace string) {
	return
}
