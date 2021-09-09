//go:build integration
// +build integration

package integration_test

import (
	"fmt"
	"testing"

	"github.com/kyma-project/helm-broker/pkg/apis/addons/v1alpha1"
)

const (
	stageNS       = "stage"
	prodNS        = "prod"
	clusterPrefix = "cluster"
)

func TestHttpBasicAuth(t *testing.T) {
	// given
	suite := newTestSuiteAndStartControllers(t, true, true)
	defer suite.tearDown()

	t.Run("namespaced", func(t *testing.T) {
		suite.createSecret(stageNS, "data-namespace", map[string]string{"username": basicUsername, "password": basicPassword})

		// when
		suite.createAddonsConfiguration(stageNS, addonsConfigName, []string{redisAndAccTestRepo}, sourceHTTP,
			WithSecretReference(stageNS, "data-namespace"),
			WithHTTPBasicAuth("{username}", "{password}"))

		// then
		suite.waitForAddonsConfigurationPhase(stageNS, addonsConfigName, v1alpha1.AddonsConfigurationReady)
		suite.waitForServicesInCatalogEndpoint(fmt.Sprintf("ns/%s", stageNS), []string{redisAddonID, accTestAddonID})
	})

	t.Run("cluster", func(t *testing.T) {
		suite.createSecret(stageNS, "data-cluster", map[string]string{"username": basicUsername, "password": basicPassword})

		// when
		suite.createClusterAddonsConfiguration(addonsConfigName, []string{redisAndAccTestRepo}, sourceHTTP,
			WithSecretReference(stageNS, "data-cluster"),
			WithHTTPBasicAuth("{username}", "{password}"))

		// then
		suite.waitForClusterAddonsConfigurationPhase(addonsConfigName, v1alpha1.AddonsConfigurationReady)
		suite.waitForServicesInCatalogEndpoint("cluster", []string{redisAddonID, accTestAddonID})
	})
}

func TestPendingClusterAddonsConfiguration(t *testing.T) {
	// given
	suite := newTestSuite(t, false, false)
	defer suite.tearDown()

	// create a ClusterAddonsConfiguration with status Pending
	suite.createClusterAddonsConfiguration(addonsConfigName, []string{redisRepo}, sourceHTTP)
	suite.updateClusterAddonsConfigurationStatusPhase(addonsConfigName, v1alpha1.AddonsConfigurationPending)

	// when
	suite.StartControllers(false)

	// then
	suite.waitForClusterAddonsConfigurationPhase(addonsConfigName, v1alpha1.AddonsConfigurationReady)
}

func TestPendingAddonsConfiguration(t *testing.T) {
	// given
	suite := newTestSuite(t, false, false)
	defer suite.tearDown()

	// create an AddonsConfiguration with status Pending
	suite.createAddonsConfiguration(stageNS, addonsConfigName, []string{redisRepo}, sourceHTTP)
	suite.updateAddonsConfigurationStatusPhase(stageNS, addonsConfigName, v1alpha1.AddonsConfigurationPending)

	// when
	suite.StartControllers(false)

	// then
	suite.waitForAddonsConfigurationPhase(stageNS, addonsConfigName, v1alpha1.AddonsConfigurationReady)
}

func TestGetCatalogHappyPath(t *testing.T) {
	// given
	suite := newTestSuiteAndStartControllers(t, true, false)
	// run minio server only if S3 is tested, running the "minio server" each time initiate test in "TestSuite" takes too much time
	suite.initMinioServer()
	defer suite.tearDown()

	for name, c := range map[string]struct {
		kind      string
		addonName string
		redisID   string
		testID    string
	}{
		"namespaced-http": {
			kind:      sourceHTTP,
			addonName: addonsConfigName,
			redisID:   redisAddonID,
			testID:    accTestAddonID,
		},
		"namespaced-git": {
			kind:      sourceGit,
			addonName: addonsConfigNameGit,
			redisID:   redisAddonIDGit,
			testID:    accTestAddonIDGit,
		},
		"namespaced-hg": {
			kind:      sourceHg,
			addonName: addonsConfigNameHg,
			redisID:   redisAddonIDHg,
			testID:    accTestAddonIDHg,
		},
		"namespaced-s3": {
			kind:      sourceS3,
			addonName: addonsConfigNameS3,
			redisID:   redisAddonIDS3,
			testID:    accTestAddonIDS3,
		},
	} {
		t.Run(name, func(t *testing.T) {
			suite.assertNoServicesInCatalogEndpoint(fmt.Sprintf("ns/%s", stageNS))

			// when
			suite.createAddonsConfiguration(stageNS, c.addonName, []string{redisAndAccTestRepo}, c.kind)

			// then
			suite.waitForAddonsConfigurationPhase(stageNS, c.addonName, v1alpha1.AddonsConfigurationReady)
			suite.waitForServicesInCatalogEndpoint(fmt.Sprintf("ns/%s", stageNS), []string{c.redisID, c.testID})
			suite.assertNoServicesInCatalogEndpoint(fmt.Sprintf("ns/%s", prodNS))
			suite.assertNoServicesInCatalogEndpoint("cluster")

			// when
			suite.createAddonsConfiguration(prodNS, c.addonName, []string{redisAndAccTestRepo}, c.kind)
			suite.waitForAddonsConfigurationPhase(prodNS, c.addonName, v1alpha1.AddonsConfigurationReady)
			suite.waitForServicesInCatalogEndpoint(fmt.Sprintf("ns/%s", prodNS), []string{c.redisID, c.testID})
			suite.waitForServiceBrokerRegistered(prodNS)

			// when
			suite.updateAddonsConfigurationRepositories(stageNS, c.addonName, []string{}, c.kind)
			suite.updateAddonsConfigurationRepositories(prodNS, c.addonName, []string{}, c.kind)

			// then
			suite.waitForEmptyCatalogResponse(fmt.Sprintf("ns/%s", stageNS))
			suite.waitForEmptyCatalogResponse(fmt.Sprintf("ns/%s", prodNS))
		})
	}

	for name, c := range map[string]struct {
		kind      string
		addonName string
		redisID   string
		testID    string
	}{
		"cluster-http": {
			kind:      sourceHTTP,
			addonName: addonsConfigName,
			redisID:   redisAddonID,
		},
		"cluster-git": {
			kind:      sourceGit,
			addonName: addonsConfigNameGit,
			redisID:   redisAddonIDGit,
		},
		"cluster-hg": {
			kind:      sourceHg,
			addonName: addonsConfigNameHg,
			redisID:   redisAddonIDHg,
		},
		"cluster-s3": {
			kind:      sourceS3,
			addonName: addonsConfigNameS3,
			redisID:   redisAddonIDS3,
		},
	} {
		t.Run(name, func(t *testing.T) {
			suite.assertNoServicesInCatalogEndpoint("cluster")

			// when
			suite.createClusterAddonsConfiguration(c.addonName, []string{redisRepo}, c.kind)

			// then
			suite.waitForClusterAddonsConfigurationPhase(c.addonName, v1alpha1.AddonsConfigurationReady)
			suite.waitForServicesInCatalogEndpoint("cluster", []string{c.redisID})

			// when
			suite.updateClusterAddonsConfigurationRepositories(c.addonName, []string{}, c.kind)

			// then
			suite.waitForEmptyCatalogResponse("cluster")
		})
	}
}

func TestProvisioning(t *testing.T) {
	// given
	suite := newTestSuiteAndStartControllers(t, false, false)
	defer suite.tearDown()

	suite.createAddonsConfiguration(stageNS, addonsConfigName, []string{redisRepo}, sourceHTTP)
	suite.waitForServicesInCatalogEndpoint(fmt.Sprintf("ns/%s", stageNS), []string{redisAddonID})
	suite.waitForNumberOfReleases(0, stageNS)

	// when
	suite.provisionInstanceFromServiceClass(fmt.Sprintf("ns/%s", stageNS), stageNS)

	// then
	suite.waitForNumberOfReleases(1, stageNS)
}

func TestUnregisteringServiceBroker(t *testing.T) {
	// given
	suite := newTestSuiteAndStartControllers(t, false, false)
	defer suite.tearDown()

	suite.createAddonsConfiguration(stageNS, addonsConfigName, []string{redisRepo}, sourceHTTP)
	suite.waitForServiceBrokerRegistered(stageNS)
	suite.waitForServicesInCatalogEndpoint(fmt.Sprintf("ns/%s", stageNS), []string{redisAddonID})

	// when
	suite.provisionInstanceFromServiceClass(fmt.Sprintf("ns/%s", stageNS), stageNS)

	// then
	suite.waitForNumberOfReleases(1, stageNS)

	// when
	suite.deleteAddonsConfiguration(stageNS, addonsConfigName)

	// then
	suite.waitForServiceBrokerRegistered(stageNS)

	// when
	suite.deprovisionInstance(fmt.Sprintf("ns/%s", stageNS), stageNS)

	// then
	suite.waitForServiceBrokerNotRegistered(stageNS)
}

func TestUnregisteringClusterServiceBroker(t *testing.T) {
	// given
	suite := newTestSuiteAndStartControllers(t, false, false)
	defer suite.tearDown()

	suite.createClusterAddonsConfiguration(addonsConfigName, []string{redisRepo}, sourceHTTP)
	suite.waitForClusterServiceBrokerRegistered()
	suite.waitForServicesInCatalogEndpoint("cluster", []string{redisAddonID})

	// when
	suite.provisionInstanceFromClusterServiceClass("cluster", stageNS)

	// then
	suite.waitForNumberOfReleases(1, stageNS)

	// when
	suite.deleteClusterAddonsConfiguration(addonsConfigName)

	// then
	suite.waitForServicesInCatalogEndpoint("cluster", []string{})
	// CSB still is registered because of the existing instance
	suite.waitForClusterServiceBrokerRegistered()

	// when
	suite.deprovisionInstance("cluster", stageNS)

	// then
	suite.waitForClusterServiceBrokerNotRegistered()
}

func TestRetryOnNetworkError(t *testing.T) {
	// given
	suite := newTestSuiteAndStartControllers(t, false, false)
	defer suite.tearDown()
	suite.brokeRepoServer()

	acName := "sample-ac"

	// prepare failed addons configuration
	suite.createAddonsConfiguration(stageNS, acName, []string{redisRepo}, sourceHTTP)
	suite.waitForAddonsConfigurationPhase(stageNS, acName, v1alpha1.AddonsConfigurationFailed)

	// when
	suite.repairRepoServer()

	// then
	suite.waitForAddonsConfigurationPhase(stageNS, acName, v1alpha1.AddonsConfigurationReady)
}

// TestAddonsConflicts check Helm Broker contract with conflicts on Addons.
// It's tested only with HTTP, testing other protocols do not make sense, cause
// conflicts resolving is in higher layer, so it's protocol agnostic.
func TestAddonsConflicts(t *testing.T) {
	// given
	suite := newTestSuiteAndStartControllers(t, true, false)
	defer suite.tearDown()

	for name, c := range map[string]struct {
		kind    string
		redisID string
		testID  string
	}{
		"namespaced-http": {
			kind:    sourceHTTP,
			redisID: redisAddonID,
			testID:  accTestAddonID,
		},
	} {
		t.Run(name, func(t *testing.T) {
			first := "first-" + c.kind
			second := "second-" + c.kind
			third := "third-" + c.kind

			// when
			//  - create an addons configuration with repo with redis addon
			suite.createAddonsConfiguration(stageNS, first, []string{redisRepo}, c.kind)

			// then
			//  - wait for readiness and wait for service redis at the catalog endpoint
			suite.waitForAddonsConfigurationPhase(stageNS, first, v1alpha1.AddonsConfigurationReady)
			suite.waitForServicesInCatalogEndpoint(fmt.Sprintf("ns/%s", stageNS), []string{c.redisID})

			// when
			// - create second addons configuration with a repo with redis and acc-test addons
			suite.createAddonsConfiguration(stageNS, second, []string{redisAndAccTestRepo}, c.kind)

			// then
			// - expect phase "failed", still redis service at the catalog endpoint
			suite.waitForAddonsConfigurationPhase(stageNS, second, v1alpha1.AddonsConfigurationFailed)
			suite.waitForServicesInCatalogEndpoint(fmt.Sprintf("ns/%s", stageNS), []string{c.redisID})

			// when
			// - remove repo with redis from the first (cluster) addon
			suite.updateAddonsConfigurationRepositories(stageNS, first, []string{}, c.kind)

			// then
			// - expect for readiness and 2 services at the catalog endpoint
			suite.waitForAddonsConfigurationPhase(stageNS, second, v1alpha1.AddonsConfigurationReady)
			suite.waitForServicesInCatalogEndpoint(fmt.Sprintf("ns/%s", stageNS), []string{c.redisID, c.testID})

			// when
			// - create third addons configuration with a repo with acc-test addons
			suite.createAddonsConfiguration(stageNS, third, []string{accTestRepo}, c.kind)

			// then
			// - expect failed (because of the conflict)
			suite.waitForAddonsConfigurationPhase(stageNS, third, v1alpha1.AddonsConfigurationFailed)

			// when
			// - delete second (cluster) addons configuration, so the third will be reprocessed
			suite.deleteAddonsConfiguration(stageNS, second)

			// then
			// - expect readiness
			suite.waitForAddonsConfigurationPhase(stageNS, third, v1alpha1.AddonsConfigurationReady)
			suite.waitForServicesInCatalogEndpoint(fmt.Sprintf("ns/%s", stageNS), []string{c.testID})
		})
	}

	for name, c := range map[string]struct {
		kind    string
		redisID string
		testID  string
	}{
		"cluster-http": {
			kind:    sourceHTTP,
			redisID: redisAddonID,
			testID:  accTestAddonID,
		},
		"cluster-git": {
			kind:    sourceGit,
			redisID: redisAddonIDGit,
			testID:  accTestAddonIDGit,
		},
		"cluster-hg": {
			kind:    sourceHg,
			redisID: redisAddonIDHg,
			testID:  accTestAddonIDHg,
		},
	} {
		t.Run(name, func(t *testing.T) {
			first := "first-" + c.kind
			second := "second-" + c.kind
			third := "third-" + c.kind

			// when
			//  - create an cluster addons configuration with repo with redis addon
			suite.createClusterAddonsConfiguration(first, []string{redisRepo}, c.kind)

			// then
			//  - wait for readiness and wait for service redis at the catalog endpoint
			suite.waitForClusterAddonsConfigurationPhase(first, v1alpha1.AddonsConfigurationReady)
			suite.waitForServicesInCatalogEndpoint("cluster", []string{c.redisID})

			// when
			// - create second cluster addons configuration with a repo with redis and acc-test addons
			suite.createClusterAddonsConfiguration(second, []string{redisAndAccTestRepo}, c.kind)

			// then
			// - expect phase "failed", still redis service at the catalog endpoint
			suite.waitForClusterAddonsConfigurationPhase(second, v1alpha1.AddonsConfigurationFailed)
			suite.waitForServicesInCatalogEndpoint("cluster", []string{c.redisID})

			// when
			// - remove repo with redis from the first (cluster) addon
			suite.updateClusterAddonsConfigurationRepositories(first, []string{}, c.kind)

			// then
			// - expect for readiness and 2 services at the catalog endpoint
			suite.waitForClusterAddonsConfigurationPhase(second, v1alpha1.AddonsConfigurationReady)
			suite.waitForServicesInCatalogEndpoint("cluster", []string{c.redisID, c.testID})

			// when
			// - create third cluster addons configuration with a repo with acc-test addons
			suite.createClusterAddonsConfiguration(third, []string{accTestRepo}, c.kind)

			// then
			// - expect failed (because of the conflict)
			suite.waitForClusterAddonsConfigurationPhase(third, v1alpha1.AddonsConfigurationFailed)

			// when
			// - delete second cluster addons configuration, so the third will be reprocessed
			suite.deleteClusterAddonsConfiguration(second)

			// then
			// - expect readiness
			suite.waitForClusterAddonsConfigurationPhase(third, v1alpha1.AddonsConfigurationReady)
			suite.waitForServicesInCatalogEndpoint("cluster", []string{c.testID})
		})
	}
}

// TestAssetGroup check Helm Broker  with conflicts on Addons.
// It's tested only with HTTP and GIT protocols.
// Test case with GIT protocol covers also implementation
// for HG and S3 because they are using the same abstraction factory.
func TestAssetGroup(t *testing.T) {
	// given
	suite := newTestSuiteAndStartControllers(t, true, false)
	defer suite.tearDown()

	for name, c := range map[string]struct {
		kind         string
		addonName    string
		assetGroupID string
	}{
		"namespaced-http": {
			kind:         sourceHTTP,
			addonName:    addonsConfigName,
			assetGroupID: accTestAddonID,
		},
		"namespaced-git": {
			kind:         sourceGit,
			addonName:    addonsConfigNameGit,
			assetGroupID: accTestAddonIDGit,
		},
	} {
		t.Run(name, func(t *testing.T) {
			// when
			suite.createAddonsConfiguration(stageNS, c.addonName, []string{redisAndAccTestRepo}, c.kind)

			// then
			suite.waitForAddonsConfigurationPhase(stageNS, c.addonName, v1alpha1.AddonsConfigurationReady)
			suite.assertAssetGroupExist(stageNS, c.assetGroupID)

			// when
			suite.updateAddonsConfigurationRepositories(stageNS, c.addonName, []string{redisRepo}, c.kind)

			// then
			suite.assertAssetGroupListIsEmpty()
		})
	}

	for name, c := range map[string]struct {
		kind         string
		addonName    string
		assetGroupID string
	}{
		"cluster-http": {
			kind:         sourceHTTP,
			addonName:    addonsConfigName,
			assetGroupID: accTestAddonID,
		},
		"cluster-git": {
			kind:         sourceGit,
			addonName:    addonsConfigNameGit,
			assetGroupID: accTestAddonIDGit,
		},
	} {
		t.Run(name, func(t *testing.T) {
			suite.createClusterAddonsConfiguration(c.addonName, []string{redisAndAccTestRepo}, c.kind)

			// then
			suite.waitForClusterAddonsConfigurationPhase(c.addonName, v1alpha1.AddonsConfigurationReady)
			suite.assertClusterAssetGroupExist(c.assetGroupID)

			// when
			suite.updateClusterAddonsConfigurationRepositories(c.addonName, []string{redisRepo}, c.kind)

			// then
			suite.assertClusterAssetGroupListIsEmpty()
		})
	}
}

// TestDisabledDocs check Helm Broker  with conflicts on Addons.
// It's tested only with HTTP and GIT protocols.
// Test case with GIT protocol covers also implementation
// for HG and S3 because they are using the same abstraction factory.
func TestDisabledDocs(t *testing.T) {
	suite := newTestSuiteAndStartControllers(t, false, false)
	defer suite.tearDown()

	for tn, tc := range map[string]struct {
		kind      string
		addonName string
		redisID   string
	}{
		"namespaced-http": {
			kind:      sourceHTTP,
			addonName: addonsConfigName,
			redisID:   redisAddonID,
		},
		"namespaced-git": {
			kind:      sourceGit,
			addonName: addonsConfigNameGit,
			redisID:   redisAddonIDGit,
		},
	} {
		t.Run(tn, func(t *testing.T) {
			// given
			suite.assertNoServicesInCatalogEndpoint(fmt.Sprintf("ns/%s", stageNS))

			// when
			suite.createAddonsConfiguration(stageNS, tc.addonName, []string{redisRepo}, tc.kind)

			// then
			suite.waitForAddonsConfigurationPhase(stageNS, tc.addonName, v1alpha1.AddonsConfigurationReady)
			suite.waitForServicesInCatalogEndpoint(fmt.Sprintf("ns/%s", stageNS), []string{tc.redisID})

			suite.deleteAddonsConfiguration(stageNS, tc.addonName)
			suite.waitForEmptyCatalogResponse(fmt.Sprintf("ns/%s", stageNS))
		})
	}

	for tn, tc := range map[string]struct {
		kind      string
		addonName string
		redisID   string
	}{
		"cluster-http": {
			kind:      sourceHTTP,
			addonName: addonsConfigName,
			redisID:   redisAddonID,
		},
		"cluster-git": {
			kind:      sourceGit,
			addonName: addonsConfigNameGit,
			redisID:   redisAddonIDGit,
		},
	} {
		t.Run(tn, func(t *testing.T) {
			// given
			suite.assertNoServicesInCatalogEndpoint("cluster")

			// when
			suite.createClusterAddonsConfiguration(tc.addonName, []string{redisRepo}, tc.kind)

			// then
			suite.waitForClusterAddonsConfigurationPhase(tc.addonName, v1alpha1.AddonsConfigurationReady)
			suite.waitForServicesInCatalogEndpoint("cluster", []string{tc.redisID})

			suite.deleteClusterAddonsConfiguration(tc.addonName)
			suite.waitForEmptyCatalogResponse("cluster")
		})
	}
}
