// +build integration

package integration_test

import (
	"testing"

	"github.com/kyma-project/helm-broker/pkg/apis/addons/v1alpha1"
)

func TestHttpBasicAuth(t *testing.T) {
	// given
	suite := newTestSuite(t, true, true)
	defer suite.tearDown()

	t.Run("namespaced", func(t *testing.T) {
		suite.createSecret("stage", "data-ns", map[string]string{"username": basicUsername, "password": basicPassword})

		// when
		suite.createAddonsConfiguration("stage", addonsConfigName, []string{redisAndAccTestRepo}, sourceHTTP,
			WithSecretReference("stage", "data-ns"),
			WithHTTPBasicAuth("{username}", "{password}"))

		// then
		suite.waitForAddonsConfigurationPhase("stage", addonsConfigName, v1alpha1.AddonsConfigurationReady)
		suite.waitForServicesInCatalogEndpoint("ns/stage", []string{redisAddonID, accTestAddonID})
	})

	t.Run("cluster", func(t *testing.T) {
		suite.createSecret("stage", "data-cluster", map[string]string{"username": basicUsername, "password": basicPassword})

		// when
		suite.createClusterAddonsConfiguration(addonsConfigName, []string{redisAndAccTestRepo}, sourceHTTP,
			WithSecretReference("stage", "data-cluster"),
			WithHTTPBasicAuth("{username}", "{password}"))

		// then
		suite.waitForClusterAddonsConfigurationPhase(addonsConfigName, v1alpha1.AddonsConfigurationReady)
		suite.waitForServicesInCatalogEndpoint("cluster", []string{redisAddonID, accTestAddonID})
	})
}

func TestGetCatalogHappyPath(t *testing.T) {
	// given
	suite := newTestSuite(t, true, false)
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
	} {
		t.Run(name, func(t *testing.T) {
			suite.assertNoServicesInCatalogEndpoint("ns/stage")

			// when
			suite.createAddonsConfiguration("stage", c.addonName, []string{redisAndAccTestRepo}, c.kind)

			// then
			suite.waitForAddonsConfigurationPhase("stage", c.addonName, v1alpha1.AddonsConfigurationReady)
			suite.waitForServicesInCatalogEndpoint("ns/stage", []string{c.redisID, c.testID})
			suite.assertNoServicesInCatalogEndpoint("ns/prod")
			suite.assertNoServicesInCatalogEndpoint("cluster")

			// when
			suite.createAddonsConfiguration("prod", c.addonName, []string{redisAndAccTestRepo}, c.kind)
			suite.waitForAddonsConfigurationPhase("prod", c.addonName, v1alpha1.AddonsConfigurationReady)
			suite.waitForServicesInCatalogEndpoint("ns/prod", []string{c.redisID, c.testID})

			// when
			suite.updateAddonsConfigurationRepositories("stage", c.addonName, []string{}, c.kind)
			suite.updateAddonsConfigurationRepositories("prod", c.addonName, []string{}, c.kind)

			// then
			suite.waitForEmptyCatalogResponse("ns/stage")
			suite.waitForEmptyCatalogResponse("ns/prod")
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

func TestAddonsConflicts(t *testing.T) {
	// given
	suite := newTestSuite(t, true, false)
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
		"namespaced-git": {
			kind:    sourceGit,
			redisID: redisAddonIDGit,
			testID:  accTestAddonIDGit,
		},
	} {
		t.Run(name, func(t *testing.T) {
			first := "first-" + c.kind
			second := "second-" + c.kind
			third := "third-" + c.kind

			// when
			//  - create an addons configuration with repo with redis addon
			suite.createAddonsConfiguration("stage", first, []string{redisRepo}, c.kind)

			// then
			//  - wait for readiness and wait for service redis at the catalog endpoint
			suite.waitForAddonsConfigurationPhase("stage", first, v1alpha1.AddonsConfigurationReady)
			suite.waitForServicesInCatalogEndpoint("ns/stage", []string{c.redisID})

			// when
			// - create second addons configuration with a repo with redis and acc-test addons
			suite.createAddonsConfiguration("stage", second, []string{redisAndAccTestRepo}, c.kind)

			// then
			// - expect phase "failed", still redis service at the catalog endpoint
			suite.waitForAddonsConfigurationPhase("stage", second, v1alpha1.AddonsConfigurationFailed)
			suite.waitForServicesInCatalogEndpoint("ns/stage", []string{c.redisID})

			// when
			// - remove repo with redis from the first (cluster) addon
			suite.updateAddonsConfigurationRepositories("stage", first, []string{}, c.kind)

			// then
			// - expect for readiness and 2 services at the catalog endpoint
			suite.waitForAddonsConfigurationPhase("stage", second, v1alpha1.AddonsConfigurationReady)
			suite.waitForServicesInCatalogEndpoint("ns/stage", []string{c.redisID, c.testID})

			// when
			// - create third addons configuration with a repo with acc-test addons
			suite.createAddonsConfiguration("stage", third, []string{accTestRepo}, c.kind)

			// then
			// - expect failed (because of the conflict)
			suite.waitForAddonsConfigurationPhase("stage", third, v1alpha1.AddonsConfigurationFailed)

			// when
			// - delete second (cluster) addons configuration, so the third will be reprocessed
			suite.deleteAddonsConfiguration("stage", second)

			// then
			// - expect readiness
			suite.waitForAddonsConfigurationPhase("stage", third, v1alpha1.AddonsConfigurationReady)
			suite.waitForServicesInCatalogEndpoint("ns/stage", []string{c.testID})
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

func TestDocsTopic(t *testing.T) {
	// given
	suite := newTestSuite(t, true, false)
	defer suite.tearDown()

	for name, c := range map[string]struct {
		kind        string
		addonName   string
		docsTopicID string
	}{
		"namespaced-http": {
			kind:        sourceHTTP,
			addonName:   addonsConfigName,
			docsTopicID: accTestAddonID,
		},
		"namespaced-git": {
			kind:        sourceGit,
			addonName:   addonsConfigNameGit,
			docsTopicID: accTestAddonIDGit,
		},
	} {
		t.Run(name, func(t *testing.T) {
			// when
			suite.createAddonsConfiguration("stage", c.addonName, []string{redisAndAccTestRepo}, c.kind)

			// then
			suite.waitForAddonsConfigurationPhase("stage", c.addonName, v1alpha1.AddonsConfigurationReady)
			suite.assertDocsTopicExist("stage", c.docsTopicID)

			// when
			suite.updateAddonsConfigurationRepositories("stage", c.addonName, []string{redisRepo}, c.kind)

			// then
			suite.assertDocsTopicListIsEmpty()
		})
	}

	for name, c := range map[string]struct {
		kind        string
		addonName   string
		docsTopicID string
	}{
		"cluster-http": {
			kind:        sourceHTTP,
			addonName:   addonsConfigName,
			docsTopicID: accTestAddonID,
		},
		"cluster-git": {
			kind:        sourceGit,
			addonName:   addonsConfigNameGit,
			docsTopicID: accTestAddonIDGit,
		},
	} {
		t.Run(name, func(t *testing.T) {
			suite.createClusterAddonsConfiguration(c.addonName, []string{redisAndAccTestRepo}, c.kind)

			// then
			suite.waitForClusterAddonsConfigurationPhase(c.addonName, v1alpha1.AddonsConfigurationReady)
			suite.assertClusterDocsTopicExist(c.docsTopicID)

			// when
			suite.updateClusterAddonsConfigurationRepositories(c.addonName, []string{redisRepo}, c.kind)

			// then
			suite.assertClusterDocsTopicListIsEmpty()
		})
	}
}

func TestDisabledDocs(t *testing.T) {
	suite := newTestSuite(t, false, false)
	defer suite.tearDown()

	t.Run("namespaced", func(t *testing.T) {
		suite.assertNoServicesInCatalogEndpoint("ns/stage")

		// when
		suite.createAddonsConfiguration("stage", "addon1", []string{accTestRepo}, sourceHTTP)

		// then
		suite.waitForAddonsConfigurationPhase("stage", "addon1", v1alpha1.AddonsConfigurationReady)
	})

	t.Run("cluster", func(t *testing.T) {
		suite.assertNoServicesInCatalogEndpoint("cluster")

		// when
		suite.createClusterAddonsConfiguration("addon1", []string{accTestRepo}, sourceHTTP)

		// then
		suite.waitForClusterAddonsConfigurationPhase("addon1", v1alpha1.AddonsConfigurationReady)
	})
}
