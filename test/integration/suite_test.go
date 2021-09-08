//go:build integration
// +build integration

package integration_test

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"
	"time"

	osb "github.com/kubernetes-sigs/go-open-service-broker-client/v2"
	"github.com/kubernetes-sigs/service-catalog/pkg/apis/servicecatalog/v1beta1"
	"github.com/sirupsen/logrus"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/apimachinery/pkg/util/wait"
	k8s "k8s.io/client-go/kubernetes"
	kubernetes "k8s.io/client-go/kubernetes/fake"
	"k8s.io/client-go/rest"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/envtest"

	"strings"

	"github.com/kyma-project/helm-broker/internal"
	"github.com/kyma-project/helm-broker/internal/bind"
	"github.com/kyma-project/helm-broker/internal/broker"
	"github.com/kyma-project/helm-broker/internal/config"
	"github.com/kyma-project/helm-broker/internal/controller"
	broker2 "github.com/kyma-project/helm-broker/internal/controller/broker"
	"github.com/kyma-project/helm-broker/internal/helm"
	"github.com/kyma-project/helm-broker/internal/rafter"
	"github.com/kyma-project/helm-broker/internal/rafter/automock"
	"github.com/kyma-project/helm-broker/internal/storage"
	"github.com/kyma-project/helm-broker/internal/storage/testdata"
	"github.com/kyma-project/helm-broker/pkg/apis"
	"github.com/kyma-project/helm-broker/pkg/apis/addons/v1alpha1"
	dtv1beta1 "github.com/kyma-project/rafter/pkg/apis/rafter/v1beta1"
)

func init() {
	EnsureHgInstalled()
	EnsureMinioInstalled()
}

const (
	pollingInterval = 100 * time.Millisecond

	// redisAddonID is the ID of the bundle redis in testdata dir
	redisAddonID     = "id-09834-abcd-234"
	accTestAddonID   = "a54abe18-0a84-22e9-ab34-d663bbce3d88"
	addonsConfigName = "addons"

	redisAddonIDGit     = "91c753f0-813b-4bf0-a6b6-f682b1327a21"
	accTestAddonIDGit   = "6308335c-1ace-48ef-a253-47a5c31dd52c"
	addonsConfigNameGit = "git-addons"

	redisAddonIDHg     = "91c753f0-813b-4bf0-a6b6-123-123-123"
	accTestAddonIDHg   = "6308335c-1ace-48ef-a253-123-123-123"
	addonsConfigNameHg = "hg-addons"

	redisAddonIDS3     = "67515fb2-3cc0-4267-8eb6-76c55441dbca"
	accTestAddonIDS3   = "cc2777d7-fe0c-499d-a7da-a4860c30f8d0"
	addonsConfigNameS3 = "s3-addons"

	redisRepo           = "index-redis.yaml"
	accTestRepo         = "index-acc-testing.yaml"
	redisAndAccTestRepo = "index.yaml"

	sourceHTTP = "http"
	sourceGit  = "git"
	sourceHg   = "hg"
	sourceS3   = "s3"

	basicPassword = "pAssword{"
	basicUsername = "user001"

	instanceID = "instance-id-001"

	clusterBrokerName = "helm-broker"
)

func newTestSuiteAndStartControllers(t *testing.T, docsEnabled, httpBasicAuth bool) *testSuite {
	ts := newTestSuite(t, docsEnabled, httpBasicAuth)
	ts.StartControllers(docsEnabled)
	return ts
}

func newTestSuite(t *testing.T, docsEnabled, httpBasicAuth bool) *testSuite {
	sch, err := v1alpha1.SchemeBuilder.Build()
	require.NoError(t, err)
	require.NoError(t, apis.AddToScheme(sch))
	require.NoError(t, v1beta1.AddToScheme(sch))
	require.NoError(t, corev1.AddToScheme(sch))
	require.NoError(t, dtv1beta1.AddToScheme(sch))

	k8sClientset := kubernetes.NewSimpleClientset()

	cfg := &config.Config{
		TmpDir:  os.TempDir(),
		Storage: testdata.GoldenConfigMemorySingleAll(),
	}
	storageConfig := storage.ConfigList(cfg.Storage)
	sFact, err := storage.NewFactory(&storageConfig)
	require.NoError(t, err)
	logger := logrus.New()

	// setup and start kube-apiserver
	environment := &envtest.Environment{}
	restConfig, err := environment.Start()
	require.NoError(t, err)
	_, err = envtest.InstallCRDs(restConfig, envtest.CRDInstallOptions{
		Paths:              []string{"crds/hb/", "crds/sc/"},
		ErrorIfPathMissing: true,
	})
	require.NoError(t, err)
	if docsEnabled {
		_, err = envtest.InstallCRDs(restConfig, envtest.CRDInstallOptions{
			Paths:              []string{"crds/docs/"},
			ErrorIfPathMissing: true,
		})
		require.NoError(t, err)
	}
	helmClient, err := helm.NewClient(restConfig, "secrets", logger.WithField("service", "helmclient"))
	require.NoError(t, err)
	// in the testing environment release won't be provisioned, so we need to set
	// the timeout to a second to see the expected release exists
	helmClient.SetInstallingTimeout(time.Second)

	brokerServer := broker.New(sFact.Addon(), sFact.Chart(), sFact.InstanceOperation(), sFact.BindOperation(), sFact.Instance(), sFact.InstanceBindData(),
		bind.NewRenderer(), bind.NewResolver(k8sClientset.CoreV1()), helmClient, logger.WithField("test", "int"))

	// OSB API Server
	server := httptest.NewServer(brokerServer.CreateHandler())

	// create a client for managing (cluster) addons configurations
	dynamicClient, err := client.New(restConfig, client.Options{Scheme: sch})

	// create namespaces required in the tests
	for _, ns := range []string{stageNS, prodNS} {
		err = dynamicClient.Create(context.TODO(), &corev1.Namespace{
			ObjectMeta: v1.ObjectMeta{Name: ns},
		})
		require.NoError(t, err)
	}

	// initialize git repositoryDirName
	gitRepository, err := newGitRepository(t, addonSource)
	require.NoError(t, err)
	stopCh := make(chan struct{})

	ts := &testSuite{
		t: t,

		dynamicClient: dynamicClient,
		server:        server,
		k8sClient:     k8sClientset,
		helmClient:    helmClient,

		stopCh:         stopCh,
		tmpDir:         cfg.TmpDir,
		gitRepository:  gitRepository,
		restConfig:     restConfig,
		storageFactory: sFact,

		logger: logger,
	}

	// server with addons repository
	staticSvr := http.FileServer(http.Dir("testdata"))
	ts.repoServer = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if httpBasicAuth {
			u, p, ok := r.BasicAuth()
			assert.True(t, ok, "basic auth required")
			assert.Equal(t, basicUsername, u)
			assert.Equal(t, basicPassword, p)
		}

		if ts.IsRepoServerBroker() {
			w.WriteHeader(http.StatusInternalServerError)
		}

		staticSvr.ServeHTTP(w, r)
	}))

	return ts
}

func (ts *testSuite) StartControllers(docsEnabled bool) {
	uploadClient := &automock.Client{}
	if docsEnabled {
		uploadClient.On("Upload", mock.AnythingOfType("string"), mock.Anything).Return(rafter.UploadedFile{}, nil)
	} else {
		uploadClient.On("Upload", mock.AnythingOfType("string"), mock.Anything).Return(rafter.UploadedFile{}, errors.New("Upload must not be called, the service does not exists"))
	}

	mgr := controller.SetupAndStartController(ts.restConfig, &config.ControllerConfig{
		DevelopMode:              true, // DevelopMode allows "http" urls
		ClusterServiceBrokerName: clusterBrokerName,
		TmpDir:                   os.TempDir(),
		DocumentationEnabled:     docsEnabled,
		ReprocessOnErrorDuration: 250 * time.Millisecond,
	}, ":8001", ts.storageFactory, uploadClient, ts.logger.WithField("svc", "broker"))

	go func() {
		if err := mgr.Start(ts.stopCh); err != nil {
			ts.t.Errorf("Controller Manager could not start: %v", err.Error())
		}
	}()
}

func newOSBClient(url string) (osb.Client, error) {
	config := osb.DefaultClientConfiguration()
	config.URL = url

	osbClient, err := osb.NewClient(config)
	if err != nil {
		return nil, err
	}

	return osbClient, nil
}

type testSuite struct {
	t             *testing.T
	server        *httptest.Server
	repoServer    *httptest.Server
	gitRepository *gitRepo
	minio         *minioServer

	osbClient     osb.Client
	dynamicClient client.Client
	k8sClient     k8s.Interface
	restConfig    *rest.Config
	helmClient    *helm.Client

	tmpDir         string
	stopCh         chan struct{}
	storageFactory storage.Factory

	logger logrus.FieldLogger

	planID             string
	serviceID          string
	isRepoServerBroken bool
}

func (ts *testSuite) tearDown() {
	ts.server.Close()
	ts.repoServer.Close()
	close(ts.stopCh)
	ts.gitRepository.removeTmpDir()
	if ts.minio != nil {
		err := ts.minio.clearMinio()
		require.NoError(ts.t, err)
		ts.minio.stopMinioServer()
	}
}

func (ts *testSuite) brokeRepoServer() {
	ts.isRepoServerBroken = true
}

func (ts *testSuite) repairRepoServer() {
	ts.isRepoServerBroken = false
}

func (ts *testSuite) waitForNumberOfReleases(n int, ns string) {
	timeoutCh := time.After(150 * time.Second)
	for {
		releases, err := ts.helmClient.ListReleases(internal.Namespace(ns))
		if err != nil {
			ts.t.Logf("unable to get releases: %s", err.Error())
		}
		ts.logger.Infof("num of releases: %d", len(releases))
		if len(releases) == n {
			return
		}

		select {
		case <-timeoutCh:
			assert.Fail(ts.t, "The timeout exceeded while waiting for the expected number of Helm releases")
			return
		default:
			time.Sleep(pollingInterval)
		}
	}
}

func (ts *testSuite) initMinioServer() {
	minioServer, err := runMinioServer(ts.t, ts.tmpDir)
	require.NoError(ts.t, err)

	ts.minio = minioServer
}

func (ts *testSuite) provisionInstanceFromServiceClass(prefix, namespace string) {
	osbClient, err := newOSBClient(fmt.Sprintf("%s/%s", ts.server.URL, prefix))
	require.NoError(ts.t, err)
	resp, err := osbClient.GetCatalog()
	ts.planID = resp.Services[0].Plans[0].ID
	ts.serviceID = resp.Services[0].ID
	require.NoError(ts.t, err)

	_, err = osbClient.ProvisionInstance(&osb.ProvisionRequest{
		PlanID:            ts.planID,
		ServiceID:         ts.serviceID,
		InstanceID:        instanceID,
		OrganizationGUID:  "org-guid",
		SpaceGUID:         "spaceGUID",
		AcceptsIncomplete: true,
		Context: map[string]interface{}{
			"namespace": namespace,
		},
	})
	require.NoError(ts.t, err)

	// The controller checks if there is any Service Instance (managed by Service Catalog).
	// The following code simulates Service Catalog actions
	err = ts.dynamicClient.Create(context.TODO(), &v1beta1.ServiceClass{
		ObjectMeta: v1.ObjectMeta{
			Name:      "a-class",
			Namespace: namespace,
		},
		Spec: v1beta1.ServiceClassSpec{
			ServiceBrokerName: broker2.NamespacedBrokerName,
		},
	})
	require.NoError(ts.t, err)
	err = ts.dynamicClient.Create(context.TODO(), &v1beta1.ServiceInstance{
		ObjectMeta: v1.ObjectMeta{
			Name:      "instance-001",
			Namespace: namespace,
		},
		Spec: v1beta1.ServiceInstanceSpec{
			ServiceClassRef: &v1beta1.LocalObjectReference{
				Name: "a-class",
			},
		},
	})
	require.NoError(ts.t, err)
}

func (ts *testSuite) provisionInstanceFromClusterServiceClass(prefix, namespace string) {
	osbClient, err := newOSBClient(fmt.Sprintf("%s/%s", ts.server.URL, prefix))
	require.NoError(ts.t, err)
	resp, err := osbClient.GetCatalog()
	ts.planID = resp.Services[0].Plans[0].ID
	ts.serviceID = resp.Services[0].ID
	require.NoError(ts.t, err)

	_, err = osbClient.ProvisionInstance(&osb.ProvisionRequest{
		PlanID:            ts.planID,
		ServiceID:         ts.serviceID,
		InstanceID:        instanceID,
		OrganizationGUID:  "org-guid",
		SpaceGUID:         "spaceGUID",
		AcceptsIncomplete: true,
		Context: map[string]interface{}{
			"namespace": namespace,
		},
	})
	require.NoError(ts.t, err)

	// The controller checks if there is any Service Instance (managed by Service Catalog).
	// The following code simulates Service Catalog actions
	err = ts.dynamicClient.Create(context.TODO(), &v1beta1.ClusterServiceClass{
		ObjectMeta: v1.ObjectMeta{
			Name: "some-class",
		},
		Spec: v1beta1.ClusterServiceClassSpec{
			ClusterServiceBrokerName: clusterBrokerName,
		},
	})
	require.NoError(ts.t, err)
	err = ts.dynamicClient.Create(context.TODO(), &v1beta1.ServiceInstance{
		ObjectMeta: v1.ObjectMeta{
			Name:      "instance-001",
			Namespace: namespace,
		},
		Spec: v1beta1.ServiceInstanceSpec{
			ClusterServiceClassRef: &v1beta1.ClusterObjectReference{
				Name: "some-class",
			},
		},
	})
	require.NoError(ts.t, err)
}

func (ts *testSuite) deprovisionInstance(prefix, namespace string) {
	osbClient, err := newOSBClient(fmt.Sprintf("%s/%s", ts.server.URL, prefix))
	require.NoError(ts.t, err)
	require.NoError(ts.t, err)

	_, err = osbClient.DeprovisionInstance(&osb.DeprovisionRequest{
		PlanID:            ts.planID,
		ServiceID:         ts.serviceID,
		InstanceID:        instanceID,
		AcceptsIncomplete: true,
	})
	require.NoError(ts.t, err)

	// The controller checks if there is any Service Instance (managed by Service Catalog).
	// The following code simulates Service Catalog actions
	err = ts.dynamicClient.Delete(context.TODO(), &v1beta1.ServiceInstance{
		ObjectMeta: v1.ObjectMeta{
			Name:      "instance-001",
			Namespace: namespace,
		},
		Spec: v1beta1.ServiceInstanceSpec{
			ServiceClassRef: &v1beta1.LocalObjectReference{
				Name: "a-class",
			},
		},
	})
	require.NoError(ts.t, err)
}

func (ts *testSuite) assertNoServicesInCatalogEndpoint(prefix string) {
	osbClient, err := newOSBClient(fmt.Sprintf("%s/%s", ts.server.URL, prefix))
	require.NoError(ts.t, err)
	resp, err := osbClient.GetCatalog()
	require.NoError(ts.t, err)

	assert.Empty(ts.t, resp.Services)
}

func (ts *testSuite) waitForEmptyCatalogResponse(prefix string) {
	osbClient, err := newOSBClient(fmt.Sprintf("%s/%s", ts.server.URL, prefix))
	require.NoError(ts.t, err)

	timeoutCh := time.After(3 * time.Second)
	for {
		resp, err := osbClient.GetCatalog()
		require.NoError(ts.t, err)
		if len(resp.Services) == 0 {
			return
		}

		select {
		case <-timeoutCh:
			assert.Fail(ts.t, "The timeout exceeded while waiting for the expected empty OSB catalog response")
			return
		default:
			time.Sleep(pollingInterval)
		}
	}
}

func (ts *testSuite) waitForServicesInCatalogEndpoint(prefix string, ids []string) {
	osbClient, err := newOSBClient(fmt.Sprintf("%s/%s", ts.server.URL, prefix))
	require.NoError(ts.t, err)

	timeoutCh := time.After(3 * time.Second)
	for {
		err := ts.checkServiceIDs(osbClient, ids)
		if err == nil {
			return
		}
		select {
		case <-timeoutCh:
			assert.Failf(ts.t, "The timeout exceeded while waiting for the OSB catalog response, last error: %s", err.Error())
			return
		default:
			time.Sleep(pollingInterval)
		}
	}
}

func (ts *testSuite) checkServiceIDs(osbClient osb.Client, ids []string) error {
	osbResponse, err := osbClient.GetCatalog()
	if err != nil {
		return err
	}

	idsToCheck := make(map[string]struct{})
	for _, id := range ids {
		idsToCheck[id] = struct{}{}
	}

	for _, service := range osbResponse.Services {
		delete(idsToCheck, service.ID)
	}

	if len(idsToCheck) > 0 {
		return fmt.Errorf("unexpected GetCatalogResponse, missing services: %v", idsToCheck)
	}

	return nil
}

func (ts *testSuite) waitForClusterAddonsConfigurationPhase(name string, expectedPhase v1alpha1.AddonsConfigurationPhase) {
	var cac v1alpha1.ClusterAddonsConfiguration
	ts.waitForPhase(&cac, &(cac.Status.CommonAddonsConfigurationStatus), types.NamespacedName{Name: name}, expectedPhase)
}

func (ts *testSuite) waitForAddonsConfigurationPhase(namespace, name string, expectedPhase v1alpha1.AddonsConfigurationPhase) {
	var ac v1alpha1.AddonsConfiguration
	ts.waitForPhase(&ac, &(ac.Status.CommonAddonsConfigurationStatus), types.NamespacedName{Name: name, Namespace: namespace}, expectedPhase)
}

func (ts *testSuite) waitForPhase(obj runtime.Object, status *v1alpha1.CommonAddonsConfigurationStatus, nn types.NamespacedName, expectedPhase v1alpha1.AddonsConfigurationPhase) {
	timeoutCh := time.After(5 * time.Second)
	for {
		err := ts.dynamicClient.Get(context.TODO(), nn, obj)
		require.NoError(ts.t, err)

		if status.Phase == expectedPhase {
			return
		}

		select {
		case <-timeoutCh:
			assert.Fail(ts.t, fmt.Sprintf("The timeout exceeded while waiting for the Phase %s (%q), current phase: %s", expectedPhase, nn.String(), string(status.Phase)))
			return
		default:
			time.Sleep(100 * time.Millisecond)
		}
	}
}

func (ts *testSuite) waitForServiceBrokerRegistered(namespace string) {
	timeoutCh := time.After(3 * time.Second)
	for {
		var obj v1beta1.ServiceBroker
		err := ts.dynamicClient.Get(context.TODO(), types.NamespacedName{Namespace: namespace, Name: broker2.NamespacedBrokerName}, &obj)
		if err == nil {
			return
		}
		select {
		case <-timeoutCh:
			assert.Fail(ts.t, "The timeout exceeded while waiting for the ServiceBroker", err)
			return
		default:
			time.Sleep(100 * time.Millisecond)
		}
	}
}

func (ts *testSuite) waitForClusterServiceBrokerRegistered() {
	timeoutCh := time.After(3 * time.Second)
	for {
		var obj v1beta1.ClusterServiceBroker
		err := ts.dynamicClient.Get(context.TODO(), types.NamespacedName{Name: clusterBrokerName}, &obj)
		if err == nil {
			return
		}
		select {
		case <-timeoutCh:
			assert.Fail(ts.t, "The timeout exceeded while waiting for the ClusterServiceBroker", err)
			return
		default:
			time.Sleep(100 * time.Millisecond)
		}
	}
}

func (ts *testSuite) waitForServiceBrokerNotRegistered(namespace string) {
	timeoutCh := time.After(3 * time.Second)
	for {
		var obj v1beta1.ServiceBroker
		err := ts.dynamicClient.Get(context.TODO(), types.NamespacedName{Namespace: namespace, Name: broker2.NamespacedBrokerName}, &obj)
		if apierrors.IsNotFound(err) {
			return
		}
		select {
		case <-timeoutCh:
			assert.Fail(ts.t, "The timeout exceeded while waiting for the ServiceBroker unregistration")
			return
		default:
			time.Sleep(100 * time.Millisecond)
		}
	}
}

func (ts *testSuite) waitForClusterServiceBrokerNotRegistered() {
	timeoutCh := time.After(3 * time.Second)
	for {
		var obj v1beta1.ClusterServiceBroker
		err := ts.dynamicClient.Get(context.TODO(), types.NamespacedName{Name: clusterBrokerName}, &obj)
		if apierrors.IsNotFound(err) {
			return
		}
		select {
		case <-timeoutCh:
			assert.Fail(ts.t, "The timeout exceeded while waiting for the ClusterServiceBroker", err)
			return
		default:
			time.Sleep(100 * time.Millisecond)
		}
	}
}

func (ts *testSuite) deleteAddonsConfiguration(namespace, name string) {
	require.NoError(ts.t, ts.dynamicClient.Delete(context.TODO(), &v1alpha1.AddonsConfiguration{
		ObjectMeta: v1.ObjectMeta{
			Name:      name,
			Namespace: namespace,
		}}))
}

func (ts *testSuite) deleteClusterAddonsConfiguration(name string) {
	require.NoError(ts.t, ts.dynamicClient.Delete(context.TODO(), &v1alpha1.ClusterAddonsConfiguration{
		ObjectMeta: v1.ObjectMeta{
			Name: name,
		}}))
}

func (ts *testSuite) createAddonsConfiguration(namespace, name string, urls []string, repoKind string, repoModifiers ...func(r *v1alpha1.SpecRepository)) {
	err := ts.dynamicClient.Create(context.TODO(), &v1alpha1.AddonsConfiguration{
		ObjectMeta: v1.ObjectMeta{
			Name:      name,
			Namespace: namespace,
		},
		Spec: v1alpha1.AddonsConfigurationSpec{
			CommonAddonsConfigurationSpec: v1alpha1.CommonAddonsConfigurationSpec{
				Repositories: ts.createSpecRepositories(urls, repoKind, repoModifiers...),
			},
		},
	})

	if err != nil {
		ts.t.Logf("Failed during creating AddonsConfiguration: %s", err)
	}
}

func (ts *testSuite) createClusterAddonsConfiguration(name string, urls []string, repoKind string, repoModifiers ...func(r *v1alpha1.SpecRepository)) {
	ts.dynamicClient.Create(context.TODO(), &v1alpha1.ClusterAddonsConfiguration{
		ObjectMeta: v1.ObjectMeta{
			Name: name,
		},
		Spec: v1alpha1.ClusterAddonsConfigurationSpec{
			CommonAddonsConfigurationSpec: v1alpha1.CommonAddonsConfigurationSpec{
				Repositories: ts.createSpecRepositories(urls, repoKind, repoModifiers...),
			},
		},
	})
}

func (ts *testSuite) updateAddonsConfigurationStatusPhase(namespace, name string, phase v1alpha1.AddonsConfigurationPhase) {
	var addon v1alpha1.AddonsConfiguration
	ts.dynamicClient.Get(context.TODO(), types.NamespacedName{Name: name, Namespace: namespace}, &addon)

	// copy from common.go updateAddonStatus
	addon.Status.ObservedGeneration = addon.Generation
	addon.Status.LastProcessedTime = &v1.Time{Time: time.Now()}
	addon.Status.Phase = phase

	ts.dynamicClient.Status().Update(context.TODO(), &addon)
}

func (ts *testSuite) updateClusterAddonsConfigurationStatusPhase(name string, phase v1alpha1.AddonsConfigurationPhase) {
	var addon v1alpha1.ClusterAddonsConfiguration
	ts.dynamicClient.Get(context.TODO(), types.NamespacedName{Name: name}, &addon)

	// copy from common.go updateAddonStatus
	addon.Status.ObservedGeneration = addon.Generation
	addon.Status.LastProcessedTime = &v1.Time{Time: time.Now()}
	addon.Status.Phase = phase

	ts.dynamicClient.Status().Update(context.TODO(), &addon)
}

func (ts *testSuite) createSpecRepositories(urls []string, repoKind string, repoModifiers ...func(r *v1alpha1.SpecRepository)) []v1alpha1.SpecRepository {
	// v1alpha1.SpecRepository cannot be null, needs to be empty array
	if len(urls) == 0 {
		return []v1alpha1.SpecRepository{}
	}
	var repositories []v1alpha1.SpecRepository
	for _, url := range urls {

		var fullURL string
		switch repoKind {
		case sourceHTTP:
			fullURL = ts.repoServer.URL + "/" + url
		case sourceGit:
			fullURL = "git::" + ts.gitRepository.path(url)
		case sourceHg:
			fullURL = "hg::" + fakeHgRepoURL(url)
		case sourceS3:
			fullURL = "s3::" + ts.minio.minioURL(url)
		default:
			ts.t.Fatalf("Unsupported source kind: %s", repoKind)
		}

		specRepoItem := v1alpha1.SpecRepository{URL: fullURL}
		for _, m := range repoModifiers {
			m(&specRepoItem)
		}
		repositories = append(repositories, specRepoItem)
	}
	return repositories
}

func WithSecretReference(namespace, name string) func(r *v1alpha1.SpecRepository) {
	return func(r *v1alpha1.SpecRepository) {
		r.SecretRef = &corev1.SecretReference{
			Namespace: namespace,
			Name:      name,
		}
	}
}

func WithHTTPBasicAuth(username, password string) func(r *v1alpha1.SpecRepository) {
	return func(r *v1alpha1.SpecRepository) {
		r.URL = strings.Replace(r.URL, "http://", fmt.Sprintf("http://%s:%s@", username, password), 1)
	}
}

func (ts *testSuite) createSecret(namespace, name string, values map[string]string) {
	ts.dynamicClient.Create(context.TODO(), &corev1.Secret{
		ObjectMeta: v1.ObjectMeta{
			Name:      name,
			Namespace: namespace,
		},
		StringData: values,
	})
}

func (ts *testSuite) updateAddonsConfigurationRepositories(namespace, name string, urls []string, repoKind string) {
	var addonsConfiguration v1alpha1.AddonsConfiguration
	require.NoError(ts.t, ts.dynamicClient.Get(context.TODO(), types.NamespacedName{Namespace: namespace, Name: name}, &addonsConfiguration))

	addonsConfiguration.Spec.Repositories = ts.createSpecRepositories(urls, repoKind)
	require.NoError(ts.t, ts.dynamicClient.Update(context.TODO(), &addonsConfiguration))
}

func (ts *testSuite) updateClusterAddonsConfigurationRepositories(name string, urls []string, repoKind string) {
	var clusterAddonsConfiguration v1alpha1.ClusterAddonsConfiguration
	require.NoError(ts.t, ts.dynamicClient.Get(context.TODO(), types.NamespacedName{Name: name}, &clusterAddonsConfiguration))

	clusterAddonsConfiguration.Spec.Repositories = ts.createSpecRepositories(urls, repoKind)
	require.NoError(ts.t, ts.dynamicClient.Update(context.TODO(), &clusterAddonsConfiguration))
}

func (ts *testSuite) assertAssetGroupExist(namespace, name string) {
	var assetGroup dtv1beta1.AssetGroup

	err := wait.Poll(1*time.Second, 30*time.Second, func() (done bool, err error) {
		key := types.NamespacedName{Name: name, Namespace: namespace}
		err = ts.dynamicClient.Get(context.TODO(), key, &assetGroup)
		if apierrors.IsNotFound(err) {
			ts.t.Logf("AssetGroup %q not found. Retry...", key)
			return false, nil
		}
		if err != nil {
			return false, err
		}

		return true, nil
	})

	require.NoError(ts.t, err)
}

func (ts *testSuite) assertClusterAssetGroupExist(name string) {
	var clusterAssetGroup dtv1beta1.ClusterAssetGroup

	err := wait.Poll(1*time.Second, 30*time.Second, func() (done bool, err error) {
		key := types.NamespacedName{Name: name}
		err = ts.dynamicClient.Get(context.TODO(), key, &clusterAssetGroup)
		if apierrors.IsNotFound(err) {
			ts.t.Logf("ClusterAssetGroup %q not found. Retry...", key)
			return false, nil
		}
		if err != nil {
			return false, err
		}

		return true, nil
	})

	require.NoError(ts.t, err)
}

func (ts *testSuite) assertAssetGroupListIsEmpty() {
	var assetGroupList dtv1beta1.AssetGroupList

	err := wait.Poll(1*time.Second, 30*time.Second, func() (done bool, err error) {
		err = ts.dynamicClient.List(context.TODO(), &assetGroupList)
		if err != nil {
			return false, err
		}
		if len(assetGroupList.Items) != 0 {
			ts.t.Logf("AssetGroupList is not empty, current size %d. Retry...", len(assetGroupList.Items))
			return false, nil
		}

		return true, nil
	})

	require.NoError(ts.t, err)
}

func (ts *testSuite) assertClusterAssetGroupListIsEmpty() {
	var clusterAssetGroupList dtv1beta1.ClusterAssetGroupList

	err := wait.Poll(1*time.Second, 30*time.Second, func() (done bool, err error) {
		err = ts.dynamicClient.List(context.TODO(), &clusterAssetGroupList)
		if err != nil {
			return false, err
		}
		if len(clusterAssetGroupList.Items) != 0 {
			ts.t.Logf("ClusterAssetGroupList is not empty, current size %d. Retry...", len(clusterAssetGroupList.Items))
			return false, nil
		}

		return true, nil
	})

	require.NoError(ts.t, err)
}

func (ts *testSuite) IsRepoServerBroker() bool {
	return ts.isRepoServerBroken
}
