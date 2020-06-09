package broker

import (
	"testing"

	"github.com/kubernetes-sigs/service-catalog/pkg/apis/servicecatalog/v1beta1"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"context"

	"github.com/kyma-project/helm-broker/internal/platform/logger/spy"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/kubernetes/scheme"
	k8sigs "sigs.k8s.io/controller-runtime/pkg/client/fake"
)

func TestServiceBrokerSync_Success(t *testing.T) {
	// given
	destNs := fixDestNs()
	serviceBroker := fixServiceBroker()
	require.NoError(t, v1beta1.AddToScheme(scheme.Scheme))
	cli := k8sigs.NewFakeClientWithScheme(scheme.Scheme, serviceBroker)
	sbSyncer := NewBrokerSyncer(cli, spy.NewLogDummy())
	sbSyncer.SetNamespace(destNs)

	// when
	err := sbSyncer.Sync()
	require.NoError(t, err)

	// then
	sb := &v1beta1.ServiceBroker{}
	err = cli.Get(context.Background(), types.NamespacedName{Namespace: destNs, Name: serviceBroker.Name}, sb)
	require.NoError(t, err)

	assert.Equal(t, int64(1), sb.Spec.RelistRequests)
	assert.Nil(t, err)
}

func TestServiceBrokerSync_NotExistingBroker(t *testing.T) {
	// given
	require.NoError(t, v1beta1.AddToScheme(scheme.Scheme))
	cli := k8sigs.NewFakeClientWithScheme(scheme.Scheme)
	sbSyncer := NewBrokerSyncer(cli, spy.NewLogDummy())

	// when
	err := sbSyncer.Sync()

	// then
	assert.NoError(t, err)
}

func fixServiceBroker() *v1beta1.ServiceBroker {
	return &v1beta1.ServiceBroker{
		ObjectMeta: v1.ObjectMeta{
			Name:      fixBrokerName(),
			Namespace: fixDestNs(),
			Labels: map[string]string{
				"app": "label",
			},
		},
	}
}
