package broker

import (
	"fmt"

	"context"

	"github.com/kubernetes-sigs/service-catalog/pkg/apis/servicecatalog/v1beta1"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
	apiErrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

// ClusterSyncer provide services to sync the ClusterServiceBroker
type ClusterSyncer struct {
	clusterBrokerName string
	client            client.Client
	log               logrus.FieldLogger
}

// NewClusterBrokerSyncer allows to sync the ServiceBroker.
func NewClusterBrokerSyncer(cli client.Client, clusterBrokerName string, log logrus.FieldLogger) *ClusterSyncer {
	return &ClusterSyncer{
		client:            cli,
		clusterBrokerName: clusterBrokerName,
		log:               log.WithField("service", "cluster-broker-syncer"),
	}
}

// Sync syncs the ServiceBrokers, does not fail if the broker does not exists
func (r *ClusterSyncer) Sync() error {
	r.log.Infof("- syncing ClusterServiceBroker %s", r.clusterBrokerName)
	for i := 0; i < maxSyncRetries; i++ {
		broker := &v1beta1.ClusterServiceBroker{}
		err := r.client.Get(context.Background(), types.NamespacedName{Name: r.clusterBrokerName}, broker)
		switch {
		case apiErrors.IsNotFound(err):
			return nil
		case err != nil:
			return errors.Wrapf(err, "while getting ClusterServiceBrokers %s", r.clusterBrokerName)
		}

		// update RelistRequests to trigger the relist
		broker.Spec.RelistRequests = broker.Spec.RelistRequests + 1

		err = r.client.Update(context.Background(), broker)
		switch {
		case err == nil:
			return nil
		case apiErrors.IsConflict(err):
			r.log.Infof("(%d/%d) ClusterServiceBroker %s update conflict occurred.", i, maxSyncRetries, broker.Name)
		case err != nil:
			return errors.Wrapf(err, "while updating ClusterServiceBroker %s", broker.Name)
		}
	}

	return fmt.Errorf("could not sync cluster service broker (%s) after %d retries", r.clusterBrokerName, maxSyncRetries)
}

// SetNamespace sets service's working namespace
func (r *ClusterSyncer) SetNamespace(namespace string) {
	return
}
