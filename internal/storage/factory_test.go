package storage_test

import (
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/coreos/etcd/pkg/mock/mockserver"
	"github.com/kyma-project/helm-broker/internal/storage"
	"github.com/kyma-project/helm-broker/internal/storage/driver/etcd"
	"github.com/kyma-project/helm-broker/internal/storage/driver/memory"
	"github.com/kyma-project/helm-broker/internal/storage/testdata"
	"github.com/stretchr/testify/require"
)

func TestNewFactory(t *testing.T) {
	for s, tc := range map[string]struct {
		cfgGen               func() storage.ConfigList
		expAddon             interface{}
		expChart             interface{}
		expInstance          interface{}
		expInstanceOperation interface{}
	}{
		"MemorySingleAll":        {testdata.GoldenConfigMemorySingleAll, &memory.Addon{}, &memory.Chart{}, &memory.Instance{}, &memory.InstanceOperation{}},
		"MemorySingleSeparate":   {testdata.GoldenConfigMemorySingleSeparate, &memory.Addon{}, &memory.Chart{}, &memory.Instance{}, &memory.InstanceOperation{}},
		"MemoryMultipleSeparate": {testdata.GoldenConfigMemoryMultipleSeparate, &memory.Addon{}, &memory.Chart{}, &memory.Instance{}, &memory.InstanceOperation{}},
	} {
		t.Run(s, func(t *testing.T) {
			// GIVEN:
			cfg := tc.cfgGen()

			// WHEN:
			got, err := storage.NewFactory(&cfg)

			// THEN:
			assert.NoError(t, err)

			assert.IsType(t, tc.expAddon, got.Addon())
			assert.IsType(t, tc.expChart, got.Chart())
			assert.IsType(t, tc.expInstance, got.Instance())
			assert.IsType(t, tc.expInstanceOperation, got.InstanceOperation())
		})
	}
}

func TestNewFactory_WithEtcd(t *testing.T) {
	srv, err := mockserver.StartMockServers(1)
	require.NoError(t, err)
	defer srv.Stop()

	for s, tc := range map[string]struct {
		cfgGen               func(address string) storage.ConfigList
		expAddon             interface{}
		expChart             interface{}
		expInstance          interface{}
		expInstanceOperation interface{}
	}{
		"EtcdSingleAll":        {testdata.GoldenConfigEtcdSingleAll, &etcd.Addon{}, &etcd.Chart{}, &etcd.Instance{}, &etcd.InstanceOperation{}},
		"EtcdSingleSeparate":   {testdata.GoldenConfigEtcdSingleSeparate, &etcd.Addon{}, &etcd.Chart{}, &etcd.Instance{}, &etcd.InstanceOperation{}},
		"EtcdMultipleSeparate": {testdata.GoldenConfigEtcdMultipleSeparate, &etcd.Addon{}, &etcd.Chart{}, &etcd.Instance{}, &etcd.InstanceOperation{}},
		"MixEMMESeparate":      {testdata.GoldenConfigMixEMMESeparate, &etcd.Addon{}, &memory.Chart{}, &memory.Instance{}, &etcd.InstanceOperation{}},
		"MixMMEEGrouped":       {testdata.GoldenConfigMixMMEEGrouped, &memory.Addon{}, &memory.Chart{}, &etcd.Instance{}, &etcd.InstanceOperation{}},
	} {
		t.Run(s, func(t *testing.T) {
			// GIVEN:
			cfg := tc.cfgGen(srv.Servers[0].Address)

			got, err := storage.NewFactory(&cfg)

			// THEN:
			assert.NoError(t, err)

			assert.IsType(t, tc.expAddon, got.Addon())
			assert.IsType(t, tc.expChart, got.Chart())
			assert.IsType(t, tc.expInstance, got.Instance())
			assert.IsType(t, tc.expInstanceOperation, got.InstanceOperation())
		})
	}
}
