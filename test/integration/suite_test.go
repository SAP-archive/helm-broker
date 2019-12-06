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

	"github.com/kubernetes-incubator/service-catalog/pkg/apis/servicecatalog/v1beta1"
	osb "github.com/pmorie/go-open-service-broker-client/v2"
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

	"github.com/kyma-project/helm-broker/internal/assetstore"
	"github.com/kyma-project/helm-broker/internal/assetstore/automock"
	"github.com/kyma-project/helm-broker/internal/bind"
	"github.com/kyma-project/helm-broker/internal/broker"
	"github.com/kyma-project/helm-broker/internal/config"
	"github.com/kyma-project/helm-broker/internal/controller"
	"github.com/kyma-project/helm-broker/internal/storage"
	"github.com/kyma-project/helm-broker/internal/storage/testdata"
	"github.com/kyma-project/helm-broker/pkg/apis"
	"github.com/kyma-project/helm-broker/pkg/apis/addons/v1alpha1"
	dtv1alpha1 "github.com/kyma-project/kyma/components/cms-controller-manager/pkg/apis/cms/v1alpha1"
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
	require.NoError(t, dtv1alpha1.AddToScheme(sch))

	k8sClientset := kubernetes.NewSimpleClientset()

	cfg := &config.Config{
		TmpDir:  os.TempDir(),
		Storage: testdata.GoldenConfigMemorySingleAll(),
	}
	storageConfig := storage.ConfigList(cfg.Storage)
	sFact, err := storage.NewFactory(&storageConfig)
	require.NoError(t, err)
	logger := logrus.New()

	brokerServer := broker.New(sFact.Addon(), sFact.Chart(), sFact.InstanceOperation(), sFact.BindOperation(), sFact.Instance(), sFact.InstanceBindData(),
		bind.NewRenderer(), bind.NewResolver(k8sClientset.CoreV1()), nil, logger.WithField("test", "int"))

	// OSB API Server
	server := httptest.NewServer(brokerServer.CreateHandler())

	// server with addons repository
	staticSvr := http.FileServer(http.Dir("testdata"))
	repoServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if httpBasicAuth {
			u, p, ok := r.BasicAuth()
			assert.True(t, ok, "basic auth required")
			assert.Equal(t, basicUsername, u)
			assert.Equal(t, basicPassword, p)
		}

		staticSvr.ServeHTTP(w, r)
	}))

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

	// create a client for managing (cluster) addons configurations
	dynamicClient, err := client.New(restConfig, client.Options{Scheme: sch})

	// initialize git repositoryDirName
	gitRepository, err := newGitRepository(t, addonSource)
	require.NoError(t, err)
	stopCh := make(chan struct{})

	return &testSuite{
		t: t,

		dynamicClient: dynamicClient,
		repoServer:    repoServer,
		server:        server,
		k8sClient:     k8sClientset,

		stopCh:         stopCh,
		tmpDir:         cfg.TmpDir,
		gitRepository:  gitRepository,
		restConfig:     restConfig,
		storageFactory: sFact,

		logger: logger,
	}
}

func (ts *testSuite) StartControllers(docsEnabled bool) {
	uploadClient := &automock.Client{}
	if docsEnabled {
		uploadClient.On("Upload", mock.AnythingOfType("string"), mock.Anything).Return(assetstore.UploadedFile{}, nil)
	} else {
		uploadClient.On("Upload", mock.AnythingOfType("string"), mock.Anything).Return(assetstore.UploadedFile{}, errors.New("Upload must not be called, the service does not exists"))
	}

	mgr := controller.SetupAndStartController(ts.restConfig, &config.ControllerConfig{
		DevelopMode:              true, // DevelopMode allows "http" urls
		ClusterServiceBrokerName: "helm-broker",
		TmpDir:                   os.TempDir(),
		DocumentationEnabled:     docsEnabled,
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
	config.APIVersion = osb.Version2_13()

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

	tmpDir         string
	stopCh         chan struct{}
	storageFactory storage.Factory

	logger logrus.FieldLogger
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

func (ts *testSuite) initMinioServer() {
	minioServer, err := runMinioServer(ts.t, ts.tmpDir)
	require.NoError(ts.t, err)

	ts.minio = minioServer
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
		return []v1alpha1.SpecRepository{{}}
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

func (ts *testSuite) assertDocsTopicExist(namespace, name string) {
	var docsTopic dtv1alpha1.DocsTopic

	err := wait.Poll(1*time.Second, 30*time.Second, func() (done bool, err error) {
		key := types.NamespacedName{Name: name, Namespace: namespace}
		err = ts.dynamicClient.Get(context.TODO(), key, &docsTopic)
		if apierrors.IsNotFound(err) {
			ts.t.Logf("DocsTopic %q not found. Retry...", key)
			return false, nil
		}
		if err != nil {
			return false, err
		}

		return true, nil
	})

	require.NoError(ts.t, err)
}

func (ts *testSuite) assertClusterDocsTopicExist(name string) {
	var clusterDocsTopic dtv1alpha1.ClusterDocsTopic

	err := wait.Poll(1*time.Second, 30*time.Second, func() (done bool, err error) {
		key := types.NamespacedName{Name: name}
		err = ts.dynamicClient.Get(context.TODO(), key, &clusterDocsTopic)
		if apierrors.IsNotFound(err) {
			ts.t.Logf("ClusterDocsTopic %q not found. Retry...", key)
			return false, nil
		}
		if err != nil {
			return false, err
		}

		return true, nil
	})

	require.NoError(ts.t, err)
}

func (ts *testSuite) assertDocsTopicListIsEmpty() {
	var docsTopicList dtv1alpha1.DocsTopicList

	err := wait.Poll(1*time.Second, 30*time.Second, func() (done bool, err error) {
		err = ts.dynamicClient.List(context.TODO(), &client.ListOptions{}, &docsTopicList)
		if err != nil {
			return false, err
		}
		if len(docsTopicList.Items) != 0 {
			ts.t.Logf("DocsTopicList is not empty, current size %d. Retry...", len(docsTopicList.Items))
			return false, nil
		}

		return true, nil
	})

	require.NoError(ts.t, err)
}

func (ts *testSuite) assertClusterDocsTopicListIsEmpty() {
	var clusterDocsTopicList dtv1alpha1.ClusterDocsTopicList

	err := wait.Poll(1*time.Second, 30*time.Second, func() (done bool, err error) {
		err = ts.dynamicClient.List(context.TODO(), &client.ListOptions{}, &clusterDocsTopicList)
		if err != nil {
			return false, err
		}
		if len(clusterDocsTopicList.Items) != 0 {
			ts.t.Logf("ClusterDocsTopicList is not empty, current size %d. Retry...", len(clusterDocsTopicList.Items))
			return false, nil
		}

		return true, nil
	})

	require.NoError(ts.t, err)
}
