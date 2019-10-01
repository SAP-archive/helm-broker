package testdata

import (
	"github.com/kyma-project/helm-broker/internal/storage"
	"github.com/kyma-project/helm-broker/internal/storage/driver/etcd"
)

func GoldenConfigMemorySingleAll() storage.ConfigList {
	return storage.ConfigList{
		{
			Driver: storage.DriverMemory,
			Provide: storage.ProviderConfigMap{
				storage.EntityAll: storage.ProviderConfig{},
			},
		},
	}
}

func GoldenConfigMemorySingleSeparate() storage.ConfigList {
	return storage.ConfigList{
		{
			Driver: storage.DriverMemory,
			Provide: storage.ProviderConfigMap{
				storage.EntityAddon:             storage.ProviderConfig{},
				storage.EntityChart:             storage.ProviderConfig{},
				storage.EntityInstance:          storage.ProviderConfig{},
				storage.EntityInstanceOperation: storage.ProviderConfig{},
			},
		},
	}
}

func GoldenConfigMemoryMultipleSeparate() storage.ConfigList {
	return storage.ConfigList{
		{Driver: storage.DriverMemory, Provide: storage.ProviderConfigMap{storage.EntityAddon: storage.ProviderConfig{}}},
		{Driver: storage.DriverMemory, Provide: storage.ProviderConfigMap{storage.EntityChart: storage.ProviderConfig{}}},
		{Driver: storage.DriverMemory, Provide: storage.ProviderConfigMap{storage.EntityInstance: storage.ProviderConfig{}}},
		{Driver: storage.DriverMemory, Provide: storage.ProviderConfigMap{storage.EntityInstanceOperation: storage.ProviderConfig{}}},
	}
}

func GoldenConfigEtcdSingleAll() storage.ConfigList {
	return storage.ConfigList{
		{
			Driver: storage.DriverEtcd,
			Provide: storage.ProviderConfigMap{
				storage.EntityAll: storage.ProviderConfig{},
			},
			Etcd: etcd.Config{
				DialTimeout:          "5ms",
				DialKeepAliveTime:    "2s",
				DialKeepAliveTimeout: "5ms",
				Endpoints:            []string{"fix:404"},
			},
		},
	}
}

func GoldenConfigEtcdSingleSeparate() storage.ConfigList {
	return storage.ConfigList{
		{
			Driver: storage.DriverEtcd,
			Provide: storage.ProviderConfigMap{
				storage.EntityAddon:             storage.ProviderConfig{},
				storage.EntityChart:             storage.ProviderConfig{},
				storage.EntityInstance:          storage.ProviderConfig{},
				storage.EntityInstanceOperation: storage.ProviderConfig{},
			},
			Etcd: etcd.Config{
				DialTimeout:          "5ms",
				DialKeepAliveTime:    "2s",
				DialKeepAliveTimeout: "5ms",
				Endpoints:            []string{"fix:404"},
			},
		},
	}
}

func GoldenConfigEtcdMultipleSeparate() storage.ConfigList {
	return storage.ConfigList{
		{Driver: storage.DriverEtcd, Provide: storage.ProviderConfigMap{storage.EntityAddon: storage.ProviderConfig{}},
			Etcd: etcd.Config{
				DialTimeout:          "5ms",
				DialKeepAliveTime:    "2s",
				DialKeepAliveTimeout: "5ms",
				Endpoints:            []string{"fix:404"},
			}},
		{Driver: storage.DriverEtcd, Provide: storage.ProviderConfigMap{storage.EntityChart: storage.ProviderConfig{}},
			Etcd: etcd.Config{
				DialTimeout:          "5ms",
				DialKeepAliveTime:    "2s",
				DialKeepAliveTimeout: "5ms",
				Endpoints:            []string{"fix:404"},
			}},
		{Driver: storage.DriverEtcd, Provide: storage.ProviderConfigMap{storage.EntityInstance: storage.ProviderConfig{}},
			Etcd: etcd.Config{
				DialTimeout:          "5ms",
				DialKeepAliveTime:    "2s",
				DialKeepAliveTimeout: "5ms",
				Endpoints:            []string{"fix:404"},
			}},
		{Driver: storage.DriverEtcd, Provide: storage.ProviderConfigMap{storage.EntityInstanceOperation: storage.ProviderConfig{}},
			Etcd: etcd.Config{
				DialTimeout:          "5ms",
				DialKeepAliveTime:    "2s",
				DialKeepAliveTimeout: "5ms",
				Endpoints:            []string{"fix:404"},
			}},
	}
}

func GoldenConfigMixEMMESeparate() storage.ConfigList {
	return storage.ConfigList{
		{Driver: storage.DriverEtcd, Provide: storage.ProviderConfigMap{storage.EntityAddon: storage.ProviderConfig{}},
			Etcd: etcd.Config{
				DialTimeout:          "5ms",
				DialKeepAliveTime:    "2s",
				DialKeepAliveTimeout: "5ms",
				Endpoints:            []string{"fix:404"},
			}},
		{Driver: storage.DriverMemory, Provide: storage.ProviderConfigMap{storage.EntityChart: storage.ProviderConfig{}}},
		{Driver: storage.DriverMemory, Provide: storage.ProviderConfigMap{storage.EntityInstance: storage.ProviderConfig{}}},
		{Driver: storage.DriverEtcd, Provide: storage.ProviderConfigMap{storage.EntityInstanceOperation: storage.ProviderConfig{}},
			Etcd: etcd.Config{
				DialTimeout:          "5ms",
				DialKeepAliveTime:    "2s",
				DialKeepAliveTimeout: "5ms",
				Endpoints:            []string{"fix:404"},
			}},
	}
}

func GoldenConfigMixMMEEGrouped() storage.ConfigList {
	return storage.ConfigList{
		{Driver: storage.DriverMemory, Provide: storage.ProviderConfigMap{
			storage.EntityAddon: storage.ProviderConfig{},
			storage.EntityChart: storage.ProviderConfig{},
		},
			Etcd: etcd.Config{
				DialTimeout:          "5ms",
				DialKeepAliveTime:    "2s",
				DialKeepAliveTimeout: "5ms",
				Endpoints:            []string{"fix:404"},
			}},
		{Driver: storage.DriverEtcd, Provide: storage.ProviderConfigMap{
			storage.EntityInstance:          storage.ProviderConfig{},
			storage.EntityInstanceOperation: storage.ProviderConfig{},
		},
			Etcd: etcd.Config{
				DialTimeout:          "5ms",
				DialKeepAliveTime:    "2s",
				DialKeepAliveTimeout: "5ms",
				Endpoints:            []string{"fix:404"},
			}},
	}
}
