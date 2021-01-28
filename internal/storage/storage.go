package storage

import (
	"net/http"
	"time"

	"github.com/sirupsen/logrus"

	"github.com/kyma-project/helm-broker/internal/storage/driver/etcd"
	"github.com/kyma-project/helm-broker/internal/storage/driver/memory"

	"github.com/pkg/errors"
	yaml "gopkg.in/yaml.v2"
	"k8s.io/apimachinery/pkg/util/wait"
)

// Factory provides access to concrete storage.
// Multiple calls should to specific storage return the same storage instance.
type Factory interface {
	Addon() Addon
	Chart() Chart
	Instance() Instance
	InstanceOperation() InstanceOperation
	InstanceBindData() InstanceBindData
	BindOperation() BindOperation
}

// DriverType defines type of data storage
type DriverType string

const (
	// DriverEtcd is a driver for key-value store - Etcd
	DriverEtcd DriverType = "etcd"
	// DriverMemory is a driver to local in-memory store
	DriverMemory DriverType = "memory"
)

// EntityName defines name of the entity in database
type EntityName string

const (
	// EntityAll represents name of all entities
	EntityAll EntityName = "all"
	// EntityChart represents name of chart entities
	EntityChart EntityName = "chart"
	// EntityAddon represents name of addon entities
	EntityAddon EntityName = "addon"
	// EntityInstance represents name of services instances entities
	EntityInstance EntityName = "instance"
	// EntityInstanceOperation represents name of instances operations entities
	EntityInstanceOperation EntityName = "instanceOperation"
	// EntityInstanceBindData represents name of bind data entities
	EntityInstanceBindData EntityName = "entityInstanceBindData"
	// EntityBindOperation represents name of bind operations entities
	EntityBindOperation EntityName = "bindOperation"
)

// ProviderConfig provides configuration to the database provider
type ProviderConfig struct{}

// ProviderConfigMap contains map of provided configurations for given entities
type ProviderConfigMap map[EntityName]ProviderConfig

// Config contains database configuration.
type Config struct {
	Driver  DriverType        `json:"driver" valid:"required"`
	Provide ProviderConfigMap `json:"provide" valid:"required"`
	Etcd    etcd.Config       `json:"etcd"`
	Memory  memory.Config     `json:"memory"`
}

// ConfigList is a list of configurations
type ConfigList []Config

// ConfigParse is parsing yaml file to the ConfigList
func ConfigParse(inByte []byte) (*ConfigList, error) {
	var cl ConfigList

	if err := yaml.Unmarshal(inByte, &cl); err != nil {
		return nil, errors.Wrap(err, "while unmarshalling yaml")
	}

	return &cl, nil
}

// NewConfigListAllMemory returns configured configList with the memory driver for all entities.
func NewConfigListAllMemory() *ConfigList {
	return &ConfigList{{Driver: DriverMemory, Provide: ProviderConfigMap{EntityAll: ProviderConfig{}}}}
}

// ExtractEtcdURL extracts URL to the ETCD from config
func (cl *ConfigList) ExtractEtcdURL() string {
	etcdURL := ""
	for _, cfg := range *cl {
		if cfg.Driver == DriverEtcd {
			etcdURL = cfg.Etcd.Endpoints[0]
		}
	}
	return etcdURL
}

// WaitForEtcdReadiness waits for ETCD to be ready
func (cl *ConfigList) WaitForEtcdReadiness(log logrus.FieldLogger) error {
	var (
		resp       *http.Response
		lastErr    error
		lastStatus int
	)

	if err := wait.Poll(time.Second*5, time.Minute*5, func() (bool, error) {
		resp, lastErr = http.Get(cl.ExtractEtcdURL() + "/health")
		if lastErr != nil {
			log.Errorf("while getting etcd server status: %v", lastErr)
			return false, nil
		}
		if resp.StatusCode == http.StatusOK {
			return true, nil
		}
		lastStatus = resp.StatusCode
		log.Warnf("expected status code %d, got %d", http.StatusOK, lastStatus)
		return false, nil
	}); err != nil {
		return errors.Errorf("while waiting for etcd: %v: status code %d", lastErr, lastStatus)
	}
	return nil
}

// NewFactory is a factory for entities based on given ConfigList
// TODO: add error handling
func NewFactory(cl *ConfigList) (Factory, error) {
	fact := concreteFactory{}

	for _, cfg := range *cl {

		var (
			addonFact             func() (Addon, error)
			chartFact             func() (Chart, error)
			instanceFact          func() (Instance, error)
			instanceOperationFact func() (InstanceOperation, error)
			instanceBindDataFact  func() (InstanceBindData, error)
			bindOperationFact     func() (BindOperation, error)
		)

		switch cfg.Driver {
		case DriverMemory:
			addonFact = func() (Addon, error) {
				return memory.NewAddon(), nil
			}
			chartFact = func() (Chart, error) {
				return memory.NewChart(), nil
			}
			instanceFact = func() (Instance, error) {
				return memory.NewInstance(), nil
			}
			instanceOperationFact = func() (InstanceOperation, error) {
				return memory.NewInstanceOperation(), nil
			}
			instanceBindDataFact = func() (InstanceBindData, error) {
				return memory.NewInstanceBindData(), nil
			}
			bindOperationFact = func() (BindOperation, error) {
				return memory.NewBindOperation(), nil
			}
		case DriverEtcd:
			var err error
			var cli etcd.Client
			if cfg.Etcd.ForceClient != nil {
				cli = cfg.Etcd.ForceClient
			} else {
				cli, err = etcd.NewClient(cfg.Etcd)
				if err != nil {
					return nil, errors.Wrap(err, "while creating etcd client")
				}
			}

			addonFact = func() (Addon, error) {
				return etcd.NewAddon(cli)
			}
			chartFact = func() (Chart, error) {
				return etcd.NewChart(cli)
			}
			instanceFact = func() (Instance, error) {
				return etcd.NewInstance(cli)
			}
			instanceOperationFact = func() (InstanceOperation, error) {
				return etcd.NewInstanceOperation(cli)
			}
			instanceBindDataFact = func() (InstanceBindData, error) {
				return memory.NewInstanceBindData(), errors.New("warning: instance bind data storage was set to memory for security reasons")
			}
			bindOperationFact = func() (BindOperation, error) {
				return etcd.NewBindOperation(cli)
			}
		default:
			return nil, errors.New("unknown driver type")
		}

		for em := range cfg.Provide {
			switch em {
			case EntityChart:
				fact.chart, _ = chartFact()
			case EntityAddon:
				fact.addon, _ = addonFact()
			case EntityInstance:
				fact.instance, _ = instanceFact()
			case EntityInstanceOperation:
				fact.instanceOperation, _ = instanceOperationFact()
			case EntityInstanceBindData:
				fact.instanceBindData, _ = instanceBindDataFact()
			case EntityBindOperation:
				fact.bindOperation, _ = bindOperationFact()
			case EntityAll:
				fact.chart, _ = chartFact()
				fact.addon, _ = addonFact()
				fact.instance, _ = instanceFact()
				fact.instanceOperation, _ = instanceOperationFact()
				fact.instanceBindData, _ = instanceBindDataFact()
				fact.bindOperation, _ = bindOperationFact()
			default:
			}
		}
	}

	return &fact, nil
}

type concreteFactory struct {
	addon             Addon
	chart             Chart
	instance          Instance
	instanceOperation InstanceOperation
	instanceBindData  InstanceBindData
	bindOperation     BindOperation
}

func (f *concreteFactory) Addon() Addon {
	return f.addon
}
func (f *concreteFactory) Chart() Chart {
	return f.chart
}
func (f *concreteFactory) Instance() Instance {
	return f.instance
}
func (f *concreteFactory) InstanceOperation() InstanceOperation {
	return f.instanceOperation
}
func (f *concreteFactory) InstanceBindData() InstanceBindData {
	return f.instanceBindData
}
func (f *concreteFactory) BindOperation() BindOperation {
	return f.bindOperation
}
