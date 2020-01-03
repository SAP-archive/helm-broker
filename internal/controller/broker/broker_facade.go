package broker

import (
	"fmt"

	"context"

	multierror "github.com/hashicorp/go-multierror"
	"github.com/kubernetes-incubator/service-catalog/pkg/apis/servicecatalog/v1beta1"
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

// Create creates ServiceBroker. Errors don't stop execution of method. AlreadyExist errors are ignored.
func (f *Facade) Create() error {
	f.log.Infof("- creating ServiceBroker %s/%s", NamespacedBrokerName, f.namespace)

	var resultErr *multierror.Error
	svcURL := fmt.Sprintf("http://%s.%s.svc.cluster.local", f.serviceName, f.systemNamespace)
	_, err := f.createServiceBroker(svcURL)
	if err != nil {
		resultErr = multierror.Append(resultErr, err)
		f.log.Warnf("Creation of namespaced-broker for namespace [%s] results in error: [%s]. AlreadyExist errors will be ignored.", f.namespace, err)
	}
	resultErr = filterOutMultiError(resultErr, ignoreAlreadyExist)

	if resultErr == nil {
		return nil
	}
	return resultErr
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

func filterOutMultiError(merr *multierror.Error, predicate func(err error) bool) *multierror.Error {
	if merr == nil {
		return nil
	}
	var out *multierror.Error
	for _, wrapped := range merr.Errors {
		if predicate(wrapped) {
			out = multierror.Append(out, wrapped)
		}
	}
	return out

}

func ignoreAlreadyExist(err error) bool {
	return !k8serrors.IsAlreadyExists(err)
}
