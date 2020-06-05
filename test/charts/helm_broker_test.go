package charts

import (
	"context"
	"fmt"
	"testing"
	"time"

	"k8s.io/apimachinery/pkg/util/wait"

	osb "github.com/kubernetes-sigs/go-open-service-broker-client/v2"
	"github.com/kyma-project/helm-broker/pkg/apis/addons/v1alpha1"
	"github.com/kyma-project/helm-broker/pkg/client/clientset/versioned"
	"github.com/stretchr/testify/require"
	"github.com/vrischmann/envconfig"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/rand"

	//_ "k8s.io/client-go/plugin/pkg/client/auth"
	restclient "k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
)

type Config struct {
	HelmBrokerURL                 string
	Kubeconfig                    string `envconfig:"optional"`
	ClusterAddonsConfigurationURL string
	ExpectedAddonID               string
	TimeoutPerAssertion           time.Duration `envconfig:"default=2m"`
}

// TestHelmBrokerChartHappyPath verifies if newly registered Addons
// will be served on /catalog endpoint
//
// This test requires such envs:
// HELM_BROKER_URL
// CLUSTER_ADDONS_CONFIGURATION_URL
// EXPECTED_ADDON_ID

// Test can be execute also locally against k8s cluster:
// 1. Expose helm broker service locally
//     kubectl port-forward <helm_broker_pod_name> -n kyma-system 8081:8081
// 2. Execute test
//    env HELM_BROKER_URL=http://localhost:8081/cluster \
//    KUBECONFIG=/Users/$USER/.kube/config \
//    CLUSTER_ADDONS_CONFIGURATION_URL="github.com/kyma-project/addons//addons/index-acc-testing.yaml?ref=2d671e3a9d840b877dd8cd5fd9e6e0147ad4caf0" \
//    EXPECTED_ADDON_ID="a54abe18-0a84-22e9-ab34-d663bbce3d88" \
//    go test test/charts/helm_broker_test.go -v

func TestHelmBrokerChartHappyPath(t *testing.T) {
	// given
	suite := NewTestSuite(t)

	// when
	suite.createSampleClusterAddonsConfiguration()
	defer suite.deleteSampleClusterAddonsConfiguration()

	suite.waitForSampleClusterAddonsConfiguration(suite.cfg.TimeoutPerAssertion)

	// then
	suite.assertSampleClusterAddonsAreExposedOnCatalogEndpoint(suite.cfg.TimeoutPerAssertion)
}

type TestSuite struct {
	t *testing.T

	k8sClientCfg *restclient.Config
	addonsCli    *versioned.Clientset

	sampleClusterAddonsCfgName string
	cfg                        Config
}

func NewTestSuite(t *testing.T) *TestSuite {
	var cfg Config
	err := envconfig.Init(&cfg)
	require.NoError(t, err)

	k8sCfg, err := clientcmd.BuildConfigFromFlags("", cfg.Kubeconfig)
	require.NoError(t, err)

	cli, err := versioned.NewForConfig(k8sCfg)
	require.NoError(t, err)

	randID := rand.String(5)

	return &TestSuite{
		t:   t,
		cfg: cfg,

		k8sClientCfg: k8sCfg,
		addonsCli:    cli,

		sampleClusterAddonsCfgName: fmt.Sprintf("test-hb-chart-%s", randID),
	}
}

func (s *TestSuite) createSampleClusterAddonsConfiguration() {
	_, err := s.addonsCli.AddonsV1alpha1().ClusterAddonsConfigurations().Create(context.TODO(), &v1alpha1.ClusterAddonsConfiguration{
		ObjectMeta: v1.ObjectMeta{
			Name: s.sampleClusterAddonsCfgName,
		},
		Spec: v1alpha1.ClusterAddonsConfigurationSpec{
			CommonAddonsConfigurationSpec: v1alpha1.CommonAddonsConfigurationSpec{
				Repositories: []v1alpha1.SpecRepository{
					{URL: s.cfg.ClusterAddonsConfigurationURL},
				},
			},
		},
	}, v1.CreateOptions{})

	require.NoErrorf(s.t, err, "while creating cluster addons configuration")
	s.t.Logf("ClusterAddonsConfigurations %q is created", s.sampleClusterAddonsCfgName)
}

func (s *TestSuite) deleteSampleClusterAddonsConfiguration() {
	err := s.addonsCli.AddonsV1alpha1().ClusterAddonsConfigurations().Delete(context.TODO(), s.sampleClusterAddonsCfgName, v1.DeleteOptions{})
	require.NoError(s.t, err, "while creating cluster addons configuration")
	s.t.Logf("ClusterAddonsConfigurations %q is deleted", s.sampleClusterAddonsCfgName)
}

func (s *TestSuite) waitForSampleClusterAddonsConfiguration(timeout time.Duration) {
	sampleClusterAddonsAvailable := func() (done bool, err error) {
		cac, err := s.addonsCli.AddonsV1alpha1().ClusterAddonsConfigurations().Get(context.TODO(), s.sampleClusterAddonsCfgName, v1.GetOptions{})
		if err != nil {
			return false, err
		}

		if cac.Status.Phase != v1alpha1.AddonsConfigurationReady {
			s.t.Logf("ClusterAddonsConfiguration %q is not in %s state. Current status: %v",
				s.sampleClusterAddonsCfgName,
				v1alpha1.AddonsConfigurationReady,
				cac.Status,
			)
			return false, nil
		}

		return true, nil
	}

	err := wait.PollImmediate(time.Second, timeout, sampleClusterAddonsAvailable)
	require.NoError(s.t, err)

	s.t.Logf("ClusterAddonsConfigurations %q has %s state", s.sampleClusterAddonsCfgName, v1alpha1.AddonsConfigurationReady)
}

func (s *TestSuite) assertSampleClusterAddonsAreExposedOnCatalogEndpoint(timeout time.Duration) {
	config := osb.DefaultClientConfiguration()
	config.URL = s.cfg.HelmBrokerURL

	client, err := osb.NewClient(config)
	require.NoError(s.t, err, "while creating osb client for broker with URL: %s", s.cfg.HelmBrokerURL)

	err = wait.PollImmediate(time.Second, timeout, func() (bool, error) {
		response, err := client.GetCatalog()
		if err != nil {
			s.t.Logf("error when calling for catalog %v", err)
		}
		containService := func(id string) bool {
			for _, svc := range response.Services {
				if svc.ID == id {
					return true
				}
			}
			return false
		}
		if !containService(s.cfg.ExpectedAddonID) {
			s.t.Logf("expected addon %s was not found", s.cfg.ExpectedAddonID)
			return false, nil
		}

		s.t.Logf("Helm Broker exposes Service [id: %q] from ClusterAddonsConfiguration %s properly", s.cfg.ExpectedAddonID, s.sampleClusterAddonsCfgName)
		return true, nil
	})
	require.NoError(s.t, err, "while getting catalog from broker with URL: %s", s.cfg.HelmBrokerURL)
}
