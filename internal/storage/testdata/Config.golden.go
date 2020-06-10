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
				storage.EntityInstanceBindData:  storage.ProviderConfig{},
				storage.EntityBindOperation:     storage.ProviderConfig{},
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
		{Driver: storage.DriverMemory, Provide: storage.ProviderConfigMap{storage.EntityInstanceBindData: storage.ProviderConfig{}}},
		{Driver: storage.DriverMemory, Provide: storage.ProviderConfigMap{storage.EntityBindOperation: storage.ProviderConfig{}}},
	}
}

func GoldenConfigEtcdSingleAll(address string) storage.ConfigList {
	return storage.ConfigList{
		{
			Driver: storage.DriverEtcd,
			Provide: storage.ProviderConfigMap{
				storage.EntityAll: storage.ProviderConfig{},
			},
			Etcd: etcd.Config{
				DialTimeout:          "5s",
				DialKeepAliveTime:    "2s",
				DialKeepAliveTimeout: "5s",
				Endpoints:            []string{address},
			},
		},
	}
}

func GoldenConfigEtcdSingleSeparate(address string) storage.ConfigList {
	return storage.ConfigList{
		{
			Driver: storage.DriverEtcd,
			Provide: storage.ProviderConfigMap{
				storage.EntityAddon:             storage.ProviderConfig{},
				storage.EntityChart:             storage.ProviderConfig{},
				storage.EntityInstance:          storage.ProviderConfig{},
				storage.EntityInstanceOperation: storage.ProviderConfig{},
				storage.EntityInstanceBindData:  storage.ProviderConfig{},
				storage.EntityBindOperation:     storage.ProviderConfig{},
			},
			Etcd: etcd.Config{
				DialTimeout:          "5s",
				DialKeepAliveTime:    "2s",
				DialKeepAliveTimeout: "5s",
				Endpoints:            []string{address},
			},
		},
	}
}

func GoldenConfigEtcdMultipleSeparate(address string) storage.ConfigList {
	return storage.ConfigList{
		{Driver: storage.DriverEtcd, Provide: storage.ProviderConfigMap{storage.EntityAddon: storage.ProviderConfig{}},
			Etcd: etcd.Config{
				DialTimeout:          "5s",
				DialKeepAliveTime:    "2s",
				DialKeepAliveTimeout: "5s",
				Endpoints:            []string{address},
			}},
		{Driver: storage.DriverEtcd, Provide: storage.ProviderConfigMap{storage.EntityChart: storage.ProviderConfig{}},
			Etcd: etcd.Config{
				DialTimeout:          "5s",
				DialKeepAliveTime:    "2s",
				DialKeepAliveTimeout: "5s",
				Endpoints:            []string{address},
			}},
		{Driver: storage.DriverEtcd, Provide: storage.ProviderConfigMap{storage.EntityInstance: storage.ProviderConfig{}},
			Etcd: etcd.Config{
				DialTimeout:          "5s",
				DialKeepAliveTime:    "2s",
				DialKeepAliveTimeout: "5s",
				Endpoints:            []string{address},
			}},
		{Driver: storage.DriverEtcd, Provide: storage.ProviderConfigMap{storage.EntityInstanceOperation: storage.ProviderConfig{}},
			Etcd: etcd.Config{
				DialTimeout:          "5s",
				DialKeepAliveTime:    "2s",
				DialKeepAliveTimeout: "5s",
				Endpoints:            []string{address},
			}},
		{Driver: storage.DriverEtcd, Provide: storage.ProviderConfigMap{storage.EntityInstanceBindData: storage.ProviderConfig{}},
			Etcd: etcd.Config{
				DialTimeout:          "5s",
				DialKeepAliveTime:    "2s",
				DialKeepAliveTimeout: "5s",
				Endpoints:            []string{address},
			}},
		{Driver: storage.DriverEtcd, Provide: storage.ProviderConfigMap{storage.EntityBindOperation: storage.ProviderConfig{}},
			Etcd: etcd.Config{
				DialTimeout:          "5s",
				DialKeepAliveTime:    "2s",
				DialKeepAliveTimeout: "5s",
				Endpoints:            []string{address},
			}},
	}
}

func GoldenConfigMixEMMEMESeparate(address string) storage.ConfigList {
	return storage.ConfigList{
		{Driver: storage.DriverEtcd, Provide: storage.ProviderConfigMap{storage.EntityAddon: storage.ProviderConfig{}},
			Etcd: etcd.Config{
				DialTimeout:          "5s",
				DialKeepAliveTime:    "2s",
				DialKeepAliveTimeout: "5s",
				Endpoints:            []string{address},
			}},
		{Driver: storage.DriverMemory, Provide: storage.ProviderConfigMap{storage.EntityChart: storage.ProviderConfig{}}},
		{Driver: storage.DriverMemory, Provide: storage.ProviderConfigMap{storage.EntityInstance: storage.ProviderConfig{}}},
		{Driver: storage.DriverEtcd, Provide: storage.ProviderConfigMap{storage.EntityInstanceOperation: storage.ProviderConfig{}},
			Etcd: etcd.Config{
				DialTimeout:          "5s",
				DialKeepAliveTime:    "2s",
				DialKeepAliveTimeout: "5s",
				Endpoints:            []string{address},
			}},
		{Driver: storage.DriverMemory, Provide: storage.ProviderConfigMap{storage.EntityInstanceBindData: storage.ProviderConfig{}}},
		{Driver: storage.DriverEtcd, Provide: storage.ProviderConfigMap{storage.EntityBindOperation: storage.ProviderConfig{}},
			Etcd: etcd.Config{
				DialTimeout:          "5s",
				DialKeepAliveTime:    "2s",
				DialKeepAliveTimeout: "5s",
				Endpoints:            []string{address},
			}},
	}
}

func GoldenConfigMixMMMEEEGrouped(address string) storage.ConfigList {
	return storage.ConfigList{
		{Driver: storage.DriverMemory, Provide: storage.ProviderConfigMap{
			storage.EntityAddon:            storage.ProviderConfig{},
			storage.EntityChart:            storage.ProviderConfig{},
			storage.EntityInstanceBindData: storage.ProviderConfig{},
		},
			Etcd: etcd.Config{
				DialTimeout:          "5s",
				DialKeepAliveTime:    "2s",
				DialKeepAliveTimeout: "5s",
				Endpoints:            []string{address},
			}},
		{Driver: storage.DriverEtcd, Provide: storage.ProviderConfigMap{
			storage.EntityInstance:          storage.ProviderConfig{},
			storage.EntityInstanceOperation: storage.ProviderConfig{},
			storage.EntityBindOperation:     storage.ProviderConfig{},
		},
			Etcd: etcd.Config{
				DialTimeout:          "5s",
				DialKeepAliveTime:    "2s",
				DialKeepAliveTimeout: "5s",
				Endpoints:            []string{address},
			}},
	}
}
