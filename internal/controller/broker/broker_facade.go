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

const (
	// NamespacedBrokerName name of the namespaced Service Broker
	NamespacedBrokerName = "helm-broker"
	// BrokerLabelKey key of the namespaced Service Broker label
	BrokerLabelKey = "namespaced-helm-broker"
	// BrokerLabelValue value of the namespaced Service Broker label
	BrokerLabelValue = "true"
)

// Facade is responsible for creation k8s objects for namespaced broker. The Facade is not thread-safe.
type Facade struct {
	client          client.Client
	systemNamespace string
	serviceName     string
	namespace       string

	log logrus.FieldLogger
}

// NewBrokersFacade returns facade
func NewBrokersFacade(cli client.Client, systemNamespace, serviceName string, log logrus.FieldLogger) *Facade {
	return &Facade{
		client:          cli,
		systemNamespace: systemNamespace,
		serviceName:     serviceName,
		log:             log.WithField("service", "broker-facade"),
	}
}

// SetNamespace sets service's working namespace
func (f *Facade) SetNamespace(namespace string) {
	f.namespace = namespace
}

// Create creates ServiceBroker
func (f *Facade) Create() error {
	f.log.Infof("- creating ServiceBroker %s/%s", NamespacedBrokerName, f.namespace)
	svcURL := fmt.Sprintf("http://%s.%s.svc.cluster.local", f.serviceName, f.systemNamespace)

	err := wait.PollImmediate(time.Second, time.Second*30, func() (bool, error) {
		_, err := f.createServiceBroker(svcURL)
		if err != nil && !k8serrors.IsAlreadyExists(err) {
			f.log.Errorf("creation of ServiceBroker %s results in error: [%s]", NamespacedBrokerName, err)
			return false, nil
		}
		return true, nil
	})
	if err != nil {
		return errors.Wrap(err, "while waiting for service broker creation")
	}
	return nil
}

// createServiceBroker returns just created or existing ServiceBroker
func (f *Facade) createServiceBroker(svcURL string) (*v1beta1.ServiceBroker, error) {
	url := fmt.Sprintf("%s/ns/%s", svcURL, f.namespace)
	broker := &v1beta1.ServiceBroker{
		ObjectMeta: metav1.ObjectMeta{
			Name:      NamespacedBrokerName,
			Namespace: f.namespace,
			Labels: map[string]string{
				BrokerLabelKey: BrokerLabelValue,
			},
		},
		Spec: v1beta1.ServiceBrokerSpec{
			CommonServiceBrokerSpec: v1beta1.CommonServiceBrokerSpec{
				URL:            url,
				RelistRequests: 1,
			},
		},
	}

	err := f.client.Create(context.Background(), broker)
	if k8serrors.IsAlreadyExists(err) {
		f.log.Infof("ServiceBroker for namespace [%s] already exist. Attempt to get resource.", f.namespace)
		result := &v1beta1.ServiceBroker{}
		err := f.client.Get(context.Background(), types.NamespacedName{Namespace: f.namespace, Name: NamespacedBrokerName}, result)
		return result, err
	}

	return broker, err
}

// Delete removes ServiceBroker. Errors don't stop execution of method. NotFound errors are ignored.
func (f *Facade) Delete() error {
	sb := &v1beta1.ServiceBroker{
		ObjectMeta: metav1.ObjectMeta{
			Name:      NamespacedBrokerName,
			Namespace: f.namespace,
		},
	}
	f.log.Infof("- deleting ServiceBroker %s/%s", NamespacedBrokerName, f.namespace)
	err := f.client.Delete(context.Background(), sb)
	switch {
	case k8serrors.IsNotFound(err):
		return nil
	case err != nil:
		f.log.Warnf("Deletion of namespaced-broker for namespace [%s] results in error: [%s].", f.namespace, err)
	}
	return err

}

// Exist check if ServiceBroker exists.
func (f *Facade) Exist() (bool, error) {
	err := f.client.Get(context.Background(), types.NamespacedName{Namespace: f.namespace, Name: NamespacedBrokerName}, &v1beta1.ServiceBroker{})
	switch {
	case k8serrors.IsNotFound(err):
		return false, nil
	case err != nil:
		return false, errors.Wrapf(err, "while checking if ServiceBroker [%s] exists in the namespace [%s]", NamespacedBrokerName, f.namespace)
	}

	return true, nil
}
