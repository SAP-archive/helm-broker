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

// Syncer provide services to sync the ServiceBroker
type Syncer struct {
	namespace string
	client    client.Client
	log       logrus.FieldLogger
}

// NewBrokerSyncer allows to sync the ServiceBroker.
func NewBrokerSyncer(cli client.Client, log logrus.FieldLogger) *Syncer {
	return &Syncer{
		client: cli,
		log:    log.WithField("service", "broker-syncer"),
	}
}

const maxSyncRetries = 5

// Sync syncs the ServiceBrokers, does not fail if the broker does not exists
func (r *Syncer) Sync() error {
	r.log.Infof("- syncing ServiceBroker %s/%s", NamespacedBrokerName, r.namespace)
	for i := 0; i < maxSyncRetries; i++ {
		broker := &v1beta1.ServiceBroker{}
		err := r.client.Get(context.Background(), types.NamespacedName{Name: NamespacedBrokerName, Namespace: r.namespace}, broker)
		switch {
		case apiErrors.IsNotFound(err):
			return nil
		case err != nil:
			return errors.Wrapf(err, "while getting ServiceBrokers %s", NamespacedBrokerName)
		}

		// update RelistRequests to trigger the relist
		broker.Spec.RelistRequests = broker.Spec.RelistRequests + 1

		err = r.client.Update(context.Background(), broker)
		switch {
		case err == nil:
			return nil
		case apiErrors.IsConflict(err):
			r.log.Infof("(%d/%d) ServiceBroker %s update conflict occurred.", i, maxSyncRetries, broker.Name)
		case err != nil:
			return errors.Wrapf(err, "while updating ServiceBroker %s", broker.Name)
		}
	}

	return fmt.Errorf("could not sync cluster service broker (%s) after %d retries", NamespacedBrokerName, maxSyncRetries)
}

// SetNamespace sets service's working namespace
func (r *Syncer) SetNamespace(namespace string) {
	r.namespace = namespace
}
