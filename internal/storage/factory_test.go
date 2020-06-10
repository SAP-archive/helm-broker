package storage_test

import (
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/kyma-project/helm-broker/internal/storage"
	"github.com/kyma-project/helm-broker/internal/storage/driver/etcd"
	"github.com/kyma-project/helm-broker/internal/storage/driver/memory"
	"github.com/kyma-project/helm-broker/internal/storage/testdata"
	"github.com/stretchr/testify/require"
	"go.etcd.io/etcd/pkg/mock/mockserver"
)

func TestNewFactory(t *testing.T) {
	for s, tc := range map[string]struct {
		cfgGen               func() storage.ConfigList
		expAddon             interface{}
		expChart             interface{}
		expInstance          interface{}
		expInstanceOperation interface{}
		expInstanceBindData  interface{}
		expBindOperation     interface{}
	}{
		"MemorySingleAll":        {testdata.GoldenConfigMemorySingleAll, &memory.Addon{}, &memory.Chart{}, &memory.Instance{}, &memory.InstanceOperation{}, &memory.InstanceBindData{}, &memory.BindOperation{}},
		"MemorySingleSeparate":   {testdata.GoldenConfigMemorySingleSeparate, &memory.Addon{}, &memory.Chart{}, &memory.Instance{}, &memory.InstanceOperation{}, &memory.InstanceBindData{}, &memory.BindOperation{}},
		"MemoryMultipleSeparate": {testdata.GoldenConfigMemoryMultipleSeparate, &memory.Addon{}, &memory.Chart{}, &memory.Instance{}, &memory.InstanceOperation{}, &memory.InstanceBindData{}, &memory.BindOperation{}},
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
			assert.IsType(t, tc.expInstanceBindData, got.InstanceBindData())
			assert.IsType(t, tc.expBindOperation, got.BindOperation())
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
		expInstanceBindData  interface{}
		expBindOperation     interface{}
	}{
		"EtcdSingleAll":        {testdata.GoldenConfigEtcdSingleAll, &etcd.Addon{}, &etcd.Chart{}, &etcd.Instance{}, &etcd.InstanceOperation{}, &memory.InstanceBindData{}, &etcd.BindOperation{}},
		"EtcdSingleSeparate":   {testdata.GoldenConfigEtcdSingleSeparate, &etcd.Addon{}, &etcd.Chart{}, &etcd.Instance{}, &etcd.InstanceOperation{}, &memory.InstanceBindData{}, &etcd.BindOperation{}},
		"EtcdMultipleSeparate": {testdata.GoldenConfigEtcdMultipleSeparate, &etcd.Addon{}, &etcd.Chart{}, &etcd.Instance{}, &etcd.InstanceOperation{}, &memory.InstanceBindData{}, &etcd.BindOperation{}},
		"MixEMMESeparate":      {testdata.GoldenConfigMixEMMEMESeparate, &etcd.Addon{}, &memory.Chart{}, &memory.Instance{}, &etcd.InstanceOperation{}, &memory.InstanceBindData{}, &etcd.BindOperation{}},
		"MixMMEEGrouped":       {testdata.GoldenConfigMixMMMEEEGrouped, &memory.Addon{}, &memory.Chart{}, &etcd.Instance{}, &etcd.InstanceOperation{}, &memory.InstanceBindData{}, &etcd.BindOperation{}},
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
			assert.IsType(t, tc.expInstanceBindData, got.InstanceBindData())
			assert.IsType(t, tc.expBindOperation, got.BindOperation())
		})
	}
}
