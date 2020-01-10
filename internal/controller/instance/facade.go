package instance

import (
	"context"

	"github.com/kubernetes-sigs/service-catalog/pkg/apis/servicecatalog/v1beta1"
	"github.com/kyma-project/helm-broker/internal/controller/broker"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

// Facade is responsible to provide information about service instances.
type Facade struct {
	client            client.Client
	clusterBrokerName string
}

// New creates new Facade instance
func New(c client.Client, clusterBrokerName string) *Facade {
	return &Facade{
		client:            c,
		clusterBrokerName: clusterBrokerName,
	}
}

// AnyServiceInstanceExistsForNamespacedServiceBroker checks whether there is at least one service instance created with helm broker service class.
func (f *Facade) AnyServiceInstanceExistsForNamespacedServiceBroker(namespace string) (bool, error) {
	instanceList := &v1beta1.ServiceInstanceList{}
	err := f.client.List(context.TODO(), instanceList, client.InNamespace(namespace))
	if err != nil {
		return false, err
	}
	if len(instanceList.Items) == 0 {
		return false, nil
	}

	classList := &v1beta1.ServiceClassList{}
	err = f.client.List(context.TODO(), classList, client.InNamespace(namespace))
	if err != nil {
		return false, err
	}

	appBrokerClassNames := map[string]struct{}{}
	for _, c := range classList.Items {
		if c.Spec.ServiceBrokerName == broker.NamespacedBrokerName {
			appBrokerClassNames[c.Name] = struct{}{}
		}
	}

	// iterate over all instances
	for _, inst := range instanceList.Items {
		// check only created with namespaced service class
		if inst.Spec.ServiceClassRef != nil {
			if _, exists := appBrokerClassNames[inst.Spec.ServiceClassRef.Name]; exists {
				return true, nil
			}
		}
	}
	return false, nil
}

// AnyServiceInstanceExistsForClusterServiceBroker checks whether there is at least one service instance created with helm broker cluster service class.
func (f *Facade) AnyServiceInstanceExistsForClusterServiceBroker() (bool, error) {
	instanceList := &v1beta1.ServiceInstanceList{}
	err := f.client.List(context.TODO(), instanceList)
	if err != nil {
		return false, err
	}
	if len(instanceList.Items) == 0 {
		return false, nil
	}

	classList := &v1beta1.ClusterServiceClassList{}
	err = f.client.List(context.TODO(), classList)
	if err != nil {
		return false, err
	}

	brokerClassNames := map[string]struct{}{}
	for _, cl := range classList.Items {
		if cl.Spec.ClusterServiceBrokerName == f.clusterBrokerName {
			brokerClassNames[cl.Name] = struct{}{}
		}
	}

	// iterate over all instances
	for _, inst := range instanceList.Items {
		// check only created with cluster wide service class
		if inst.Spec.ClusterServiceClassRef != nil {
			if _, exists := brokerClassNames[inst.Spec.ClusterServiceClassRef.Name]; exists {
				return true, nil
			}
		}
	}
	return false, nil
}
