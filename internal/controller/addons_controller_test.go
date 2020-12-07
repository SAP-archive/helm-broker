package controller

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"os"
	"path"
	"testing"
	"time"

	"github.com/kyma-project/helm-broker/internal"
	"github.com/kyma-project/helm-broker/internal/controller/automock"
	"github.com/kyma-project/helm-broker/internal/controller/repository"
	"github.com/kyma-project/helm-broker/internal/platform/logger/spy"
	"github.com/kyma-project/helm-broker/internal/storage"
	"github.com/kyma-project/helm-broker/pkg/apis"
	"github.com/kyma-project/helm-broker/pkg/apis/addons/v1alpha1"

	"github.com/Masterminds/semver"
	"github.com/go-logr/logr"
	"github.com/kubernetes-sigs/service-catalog/pkg/apis/servicecatalog/v1beta1"
	rafter "github.com/kyma-project/rafter/pkg/apis/rafter/v1beta1"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"helm.sh/helm/v3/pkg/chart"
	v1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/record"
	"sigs.k8s.io/controller-runtime/pkg/cache"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
	"sigs.k8s.io/controller-runtime/pkg/healthz"
	"sigs.k8s.io/controller-runtime/pkg/manager"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
	"sigs.k8s.io/controller-runtime/pkg/webhook"
)

func TestReconcileAddonsConfiguration_AddAddonsProcess(t *testing.T) {
	for tn, ac := range map[string]*v1alpha1.AddonsConfiguration{
		"empty addons configuration":   fixAddonsConfiguration(),
		"pending addons configuration": fixPendingAddonsConfiguration(),
	} {
		t.Run(tn, func(t *testing.T) {
			// GIVEN
			fixAddonsCfg := ac
			ts := getTestSuite(t, fixAddonsCfg)
			indexDTO := fixIndexDTO()
			tmpDir := os.TempDir()

			ts.addonGetter.On("GetIndex").Return(indexDTO, nil)
			ts.addonGetter.On("Cleanup").Return(nil)
			for _, entry := range indexDTO.Entries {
				for _, e := range entry {
					completeAddon := fixAddonWithDocsURL(string(e.Name), string(e.Name), "example.com", "example.com")

					ts.addonGetter.On("GetCompleteAddon", e).
						Return(completeAddon, nil)
					ts.docsProvider.On("EnsureAssetGroup", completeAddon.Addon).Return(nil)
				}
			}
			ts.brokerFacade.On("Exist").Return(false, nil).Once()
			ts.addonGetterFactory.On("NewGetter", fixAddonsCfg.Spec.Repositories[0].URL, path.Join(tmpDir, "addon-loader-dst")).Return(ts.addonGetter, nil).Once()
			defer ts.assertExpectations()

			// WHEN
			reconciler := NewReconcileAddonsConfiguration(ts.mgr, ts.addonGetterFactory, ts.chartStorage, ts.addonStorage,
				ts.brokerFacade, ts.docsProvider, ts.brokerSyncer, ts.templateService, tmpDir, time.Second, spy.NewLogDummy())

			// THEN
			result, err := reconciler.Reconcile(reconcile.Request{NamespacedName: types.NamespacedName{Namespace: fixAddonsCfg.Namespace, Name: fixAddonsCfg.Name}})
			assert.NoError(t, err)
			assert.False(t, result.Requeue)

			res := v1alpha1.AddonsConfiguration{}
			err = ts.mgr.GetClient().Get(context.Background(), types.NamespacedName{Namespace: fixAddonsCfg.Namespace, Name: fixAddonsCfg.Name}, &res)
			assert.NoError(t, err)
			assert.Contains(t, res.Finalizers, v1alpha1.FinalizerAddonsConfiguration)
			assert.Equal(t, res.Status.Phase, v1alpha1.AddonsConfigurationReady)
		})
	}
}

func TestReconcileAddonsConfiguration_AddAddonsProcess_ErrorIfBrokerExist(t *testing.T) {
	// GIVEN
	fixAddonsCfg := fixAddonsConfiguration()
	ts := getTestSuite(t, fixAddonsCfg)
	indexDTO := fixIndexDTO()
	tmpDir := os.TempDir()

	ts.addonGetter.On("GetIndex").Return(indexDTO, nil)
	ts.addonGetter.On("Cleanup").Return(nil)
	for _, entry := range indexDTO.Entries {
		for _, e := range entry {
			completeAddon := fixAddonWithDocsURL(string(e.Name), string(e.Name), "example.com", "example.com")

			ts.addonGetter.On("GetCompleteAddon", e).
				Return(completeAddon, nil)
			ts.docsProvider.On("EnsureAssetGroup", completeAddon.Addon).Return(nil)
		}
	}
	ts.brokerFacade.On("Exist").Return(false, errors.New("")).Once()
	ts.addonGetterFactory.On("NewGetter", fixAddonsCfg.Spec.Repositories[0].URL, path.Join(tmpDir, "addon-loader-dst")).Return(ts.addonGetter, nil).Once()
	defer ts.assertExpectations()

	// WHEN
	reconciler := NewReconcileAddonsConfiguration(ts.mgr, ts.addonGetterFactory, ts.chartStorage, ts.addonStorage,
		ts.brokerFacade, ts.docsProvider, ts.brokerSyncer, ts.templateService, tmpDir, time.Second, spy.NewLogDummy())

	// THEN
	result, err := reconciler.Reconcile(reconcile.Request{NamespacedName: types.NamespacedName{Namespace: fixAddonsCfg.Namespace, Name: fixAddonsCfg.Name}})
	assert.Error(t, err)
	assert.False(t, result.Requeue)

	res := v1alpha1.AddonsConfiguration{}
	err = ts.mgr.GetClient().Get(context.Background(), types.NamespacedName{Namespace: fixAddonsCfg.Namespace, Name: fixAddonsCfg.Name}, &res)
	assert.NoError(t, err)
	assert.Contains(t, res.Finalizers, v1alpha1.FinalizerAddonsConfiguration)
}

func TestReconcileAddonsConfiguration_UpdateAddonsProcess(t *testing.T) {
	// GIVEN
	fixAddonsCfg := fixAddonsConfiguration()
	fixAddonsCfg.Generation = 2
	fixAddonsCfg.Status.ObservedGeneration = 1
	ts := getTestSuite(t, fixAddonsCfg)
	indexDTO := fixIndexDTO()
	tmpDir := os.TempDir()

	ts.addonGetter.On("GetIndex").Return(indexDTO, nil)
	ts.addonGetter.On("Cleanup").Return(nil)
	for _, entry := range indexDTO.Entries {
		for _, e := range entry {
			completeAddon := fixAddonWithDocsURL(string(e.Name), string(e.Name), "example.com", "example.com")

			ts.addonGetter.On("GetCompleteAddon", e).
				Return(completeAddon, nil)
			ts.docsProvider.On("EnsureAssetGroup", completeAddon.Addon).Return(nil)
		}
	}
	ts.brokerFacade.On("Exist").Return(false, nil).Once()
	ts.addonGetterFactory.On("NewGetter", fixAddonsCfg.Spec.Repositories[0].URL, path.Join(tmpDir, "addon-loader-dst")).Return(ts.addonGetter, nil).Once()
	defer ts.assertExpectations()

	// WHEN
	reconciler := NewReconcileAddonsConfiguration(ts.mgr, ts.addonGetterFactory, ts.chartStorage, ts.addonStorage,
		ts.brokerFacade, ts.docsProvider, ts.brokerSyncer, ts.templateService, tmpDir, time.Second, spy.NewLogDummy())

	// THEN
	result, err := reconciler.Reconcile(reconcile.Request{NamespacedName: types.NamespacedName{Namespace: fixAddonsCfg.Namespace, Name: fixAddonsCfg.Name}})
	assert.NoError(t, err)
	assert.False(t, result.Requeue)
}

func TestReconcileAddonsConfiguration_UpdateAddonsProcess_ConflictingAddons(t *testing.T) {
	// GIVEN
	fixAddonsCfg := fixAddonsConfiguration()
	fixAddonsCfg.Generation = 2
	fixAddonsCfg.Status.ObservedGeneration = 1

	ts := getTestSuite(t, fixAddonsCfg, fixReadyAddonsConfiguration())
	indexDTO := fixIndexDTO()
	tmpDir := os.TempDir()

	ts.addonGetter.On("GetIndex").Return(indexDTO, nil)
	ts.addonGetter.On("Cleanup").Return(nil)
	for _, entry := range indexDTO.Entries {
		for _, e := range entry {
			completeAddon := fixAddonWithDocsURL(string(e.Name), string(e.Name), "example.com", "example.com")
			ts.addonGetter.On("GetCompleteAddon", e).
				Return(completeAddon, nil)
		}
	}
	ts.addonGetterFactory.On("NewGetter", fixAddonsCfg.Spec.Repositories[0].URL, path.Join(tmpDir, "addon-loader-dst")).Return(ts.addonGetter, nil).Once()
	defer ts.assertExpectations()

	// WHEN
	reconciler := NewReconcileAddonsConfiguration(ts.mgr, ts.addonGetterFactory, ts.chartStorage, ts.addonStorage,
		ts.brokerFacade, ts.docsProvider, ts.brokerSyncer, ts.templateService, tmpDir, time.Second, spy.NewLogDummy())

	// THEN
	result, err := reconciler.Reconcile(reconcile.Request{NamespacedName: types.NamespacedName{Namespace: fixAddonsCfg.Namespace, Name: fixAddonsCfg.Name}})
	assert.NoError(t, err)
	assert.False(t, result.Requeue)

	res := v1alpha1.AddonsConfiguration{}
	err = ts.mgr.GetClient().Get(context.Background(), types.NamespacedName{Namespace: fixAddonsCfg.Namespace, Name: fixAddonsCfg.Name}, &res)
	assert.NoError(t, err)
	assert.Equal(t, res.Status.ObservedGeneration, int64(2))
	assert.Equal(t, res.Status.Phase, v1alpha1.AddonsConfigurationFailed)
}

func TestReconcileAddonsConfiguration_DeleteAddonsProcess(t *testing.T) {
	// GIVEN
	fixAddonsCfg := fixDeletedAddonsConfiguration()
	ts := getTestSuite(t, fixAddonsCfg)
	tmpDir := os.TempDir()

	defer ts.assertExpectations()

	// WHEN
	reconciler := NewReconcileAddonsConfiguration(ts.mgr, ts.addonGetterFactory, ts.chartStorage, ts.addonStorage,
		ts.brokerFacade, ts.docsProvider, ts.brokerSyncer, ts.templateService, tmpDir, time.Second, spy.NewLogDummy())

	// THEN
	result, err := reconciler.Reconcile(reconcile.Request{NamespacedName: types.NamespacedName{Namespace: fixAddonsCfg.Namespace, Name: fixAddonsCfg.Name}})
	assert.NoError(t, err)
	assert.False(t, result.Requeue)

	res := v1alpha1.AddonsConfiguration{}
	err = ts.mgr.GetClient().Get(context.Background(), types.NamespacedName{Namespace: fixAddonsCfg.Namespace, Name: fixAddonsCfg.Name}, &res)
	assert.NoError(t, err)
	assert.NotContains(t, res.Finalizers, v1alpha1.FinalizerAddonsConfiguration)
}

func TestReconcileAddonsConfiguration_DeleteAddonsProcess_ReconcileOtherAddons(t *testing.T) {
	// GIVEN
	failedAddCfg := fixFailedAddonsConfiguration()
	fixAddonsCfg := fixDeletedAddonsConfiguration()
	ts := getTestSuite(t, fixAddonsCfg, failedAddCfg)
	tmpDir := os.TempDir()

	defer ts.assertExpectations()

	// WHEN
	reconciler := NewReconcileAddonsConfiguration(ts.mgr, ts.addonGetterFactory, ts.chartStorage, ts.addonStorage,
		ts.brokerFacade, ts.docsProvider, ts.brokerSyncer, ts.templateService, tmpDir, time.Second, spy.NewLogDummy())

	// THEN
	result, err := reconciler.Reconcile(reconcile.Request{NamespacedName: types.NamespacedName{Namespace: fixAddonsCfg.Namespace, Name: fixAddonsCfg.Name}})
	assert.NoError(t, err)
	assert.False(t, result.Requeue)

	otherAddon := v1alpha1.AddonsConfiguration{}
	err = ts.mgr.GetClient().Get(context.Background(), types.NamespacedName{Namespace: failedAddCfg.Namespace, Name: failedAddCfg.Name}, &otherAddon)
	assert.NoError(t, err)
	assert.Equal(t, int(otherAddon.Spec.ReprocessRequest), 1)

	res := v1alpha1.AddonsConfiguration{}
	err = ts.mgr.GetClient().Get(context.Background(), types.NamespacedName{Namespace: fixAddonsCfg.Namespace, Name: fixAddonsCfg.Name}, &res)
	assert.NoError(t, err)
	assert.NotContains(t, res.Finalizers, v1alpha1.FinalizerAddonsConfiguration)
}

func fixRepositories() []v1alpha1.StatusRepository {
	return []v1alpha1.StatusRepository{
		{
			Status: v1alpha1.RepositoryStatusReady,
			Addons: []v1alpha1.Addon{
				{
					Status:  v1alpha1.AddonStatusReady,
					Name:    "redis",
					Version: "0.0.1",
				},
			},
		},
	}
}

func fixRepositoriesFailed() []v1alpha1.StatusRepository {
	return []v1alpha1.StatusRepository{
		{
			Status: v1alpha1.RepositoryStatusFailed,
			Addons: []v1alpha1.Addon{
				{
					Status:  v1alpha1.AddonStatusFailed,
					Name:    "redis",
					Version: "0.0.1",
				},
			},
		},
	}
}

func fixPendingAddonsConfiguration() *v1alpha1.AddonsConfiguration {
	return &v1alpha1.AddonsConfiguration{
		ObjectMeta: metav1.ObjectMeta{
			Name:       "test",
			Namespace:  "test",
			Generation: 1,
		},
		Spec: v1alpha1.AddonsConfigurationSpec{
			CommonAddonsConfigurationSpec: v1alpha1.CommonAddonsConfigurationSpec{
				ReprocessRequest: 0,
				Repositories: []v1alpha1.SpecRepository{
					{
						URL: "http://example.com/index.yaml",
					},
				},
			},
		},
		Status: v1alpha1.AddonsConfigurationStatus{
			CommonAddonsConfigurationStatus: v1alpha1.CommonAddonsConfigurationStatus{
				Phase:              v1alpha1.AddonsConfigurationPending,
				Repositories:       fixRepositories(),
				ObservedGeneration: 1,
			},
		},
	}
}

func fixAddonsConfiguration() *v1alpha1.AddonsConfiguration {
	return &v1alpha1.AddonsConfiguration{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test",
			Namespace: "test",
		},
		Spec: v1alpha1.AddonsConfigurationSpec{
			CommonAddonsConfigurationSpec: v1alpha1.CommonAddonsConfigurationSpec{
				ReprocessRequest: 0,
				Repositories: []v1alpha1.SpecRepository{
					{
						URL: "http://example.com/index.yaml",
					},
				},
			},
		},
	}
}

func fixFailedAddonsConfiguration() *v1alpha1.AddonsConfiguration {
	return &v1alpha1.AddonsConfiguration{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "failed",
			Namespace: "test",
		},
		Spec: v1alpha1.AddonsConfigurationSpec{
			CommonAddonsConfigurationSpec: v1alpha1.CommonAddonsConfigurationSpec{
				ReprocessRequest: 0,
				Repositories: []v1alpha1.SpecRepository{
					{
						URL: "http://example.com/index.yaml",
					},
				},
			},
		},
		Status: v1alpha1.AddonsConfigurationStatus{
			CommonAddonsConfigurationStatus: v1alpha1.CommonAddonsConfigurationStatus{
				Phase:        v1alpha1.AddonsConfigurationFailed,
				Repositories: fixRepositoriesFailed(),
			},
		},
	}
}

func fixReadyAddonsConfiguration() *v1alpha1.AddonsConfiguration {
	return &v1alpha1.AddonsConfiguration{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "ready",
			Namespace: "test",
		},
		Spec: v1alpha1.AddonsConfigurationSpec{
			CommonAddonsConfigurationSpec: v1alpha1.CommonAddonsConfigurationSpec{
				ReprocessRequest: 0,
				Repositories: []v1alpha1.SpecRepository{
					{
						URL: "http://example.com/index.yaml",
					},
				},
			},
		},
		Status: v1alpha1.AddonsConfigurationStatus{
			CommonAddonsConfigurationStatus: v1alpha1.CommonAddonsConfigurationStatus{
				Phase:        v1alpha1.AddonsConfigurationReady,
				Repositories: fixRepositories(),
			},
		},
	}
}

func fixDeletedAddonsConfiguration() *v1alpha1.AddonsConfiguration {
	return &v1alpha1.AddonsConfiguration{
		ObjectMeta: metav1.ObjectMeta{
			Name:              "deleted",
			Namespace:         "test",
			DeletionTimestamp: &metav1.Time{Time: time.Now()},
			Finalizers:        []string{v1alpha1.FinalizerAddonsConfiguration},
		},
		Spec: v1alpha1.AddonsConfigurationSpec{
			CommonAddonsConfigurationSpec: v1alpha1.CommonAddonsConfigurationSpec{
				ReprocessRequest: 0,
				Repositories: []v1alpha1.SpecRepository{
					{
						URL: "http://example.com/index.yaml",
					},
				},
			},
		},
		Status: v1alpha1.AddonsConfigurationStatus{
			CommonAddonsConfigurationStatus: v1alpha1.CommonAddonsConfigurationStatus{
				Phase:        v1alpha1.AddonsConfigurationReady,
				Repositories: fixRepositories(),
			},
		},
	}
}

func fixIndexDTO() *internal.Index {
	return &internal.Index{
		Entries: map[internal.AddonName][]internal.IndexEntry{
			"redis": {
				{
					Name:        "redis",
					Version:     "0.0.1",
					Description: "desc",
				}},
			"testing": {
				{
					Name:        "testing",
					Version:     "0.0.1",
					Description: "desc",
				},
			},
		},
	}
}

type testSuite struct {
	t                  *testing.T
	mgr                manager.Manager
	addonGetterFactory *automock.AddonGetterFactory
	addonGetter        *automock.AddonGetter
	brokerFacade       *automock.BrokerFacade
	docsProvider       *automock.DocsProvider
	brokerSyncer       *automock.BrokerSyncer
	templateService    *repository.Template
	addonStorage       storage.Addon
	chartStorage       storage.Chart
}

func getTestSuite(t *testing.T, objects ...runtime.Object) *testSuite {
	sch, err := v1alpha1.SchemeBuilder.Build()
	require.NoError(t, err)
	require.NoError(t, apis.AddToScheme(sch))
	require.NoError(t, v1beta1.AddToScheme(sch))
	require.NoError(t, v1.AddToScheme(sch))

	sFact, err := storage.NewFactory(storage.NewConfigListAllMemory())
	require.NoError(t, err)

	cli := fake.NewFakeClientWithScheme(sch, objects...)
	ts := &testSuite{
		t:                  t,
		mgr:                getFakeManager(t, cli, sch),
		brokerFacade:       &automock.BrokerFacade{},
		addonGetterFactory: &automock.AddonGetterFactory{},
		addonGetter:        &automock.AddonGetter{},
		brokerSyncer:       &automock.BrokerSyncer{},
		docsProvider:       &automock.DocsProvider{},
		templateService:    repository.NewTemplate(cli),

		addonStorage: sFact.Addon(),
		chartStorage: sFact.Chart(),
	}

	ts.brokerFacade.On("SetNamespace", fixAddonsConfiguration().Namespace).Return(nil).Once()
	ts.brokerSyncer.On("SetNamespace", fixAddonsConfiguration().Namespace).Return(nil).Once()
	ts.docsProvider.On("SetNamespace", fixAddonsConfiguration().Namespace).Return(nil).Once()

	return ts
}

func (ts *testSuite) assertExpectations() {
	ts.addonGetter.AssertExpectations(ts.t)
	ts.brokerFacade.AssertExpectations(ts.t)
	ts.docsProvider.AssertExpectations(ts.t)
	ts.brokerSyncer.AssertExpectations(ts.t)
	ts.addonGetterFactory.AssertExpectations(ts.t)
}

func getFakeManager(t *testing.T, cli client.Client, sch *runtime.Scheme) manager.Manager {
	return &fakeManager{
		t:      t,
		client: cli,
		sch:    sch,
	}
}

func fixAddonWithDocsURL(id, name, url, docsURL string) internal.AddonWithCharts {
	chartName := fmt.Sprintf("chart-%s", name)
	chartVersion := semver.MustParse("1.0.0")
	return internal.AddonWithCharts{
		Addon: &internal.Addon{
			ID:            internal.AddonID(id),
			Name:          internal.AddonName(name),
			Description:   "simple description",
			Version:       *semver.MustParse("0.0.1"),
			RepositoryURL: url,
			Plans: map[internal.AddonPlanID]internal.AddonPlan{
				internal.AddonPlanID(fmt.Sprintf("plan-%s", name)): {
					ChartRef: internal.ChartRef{
						Name:    internal.ChartName(chartName),
						Version: *chartVersion,
					},
				},
			},
			Docs: []internal.AddonDocs{
				{
					Template: rafter.CommonAssetGroupSpec{
						Sources: []rafter.Source{
							{
								URL: docsURL,
							},
						},
					},
				},
			},
		},
		Charts: []*chart.Chart{
			{
				Metadata: &chart.Metadata{
					Name:    chartName,
					Version: chartVersion.String(),
				},
			},
		},
	}
}

type fakeManager struct {
	t      *testing.T
	client client.Client
	sch    *runtime.Scheme
}

func (f fakeManager) Elected() <-chan struct{} {
	panic("implement me")
}

func (f fakeManager) AddMetricsExtraHandler(path string, handler http.Handler) error {
	panic("implement me")
}

func (f fakeManager) AddHealthzCheck(name string, check healthz.Checker) error {
	return nil
}

func (f fakeManager) AddReadyzCheck(name string, check healthz.Checker) error {
	return nil
}

func (f fakeManager) GetEventRecorderFor(name string) record.EventRecorder {
	return nil
}

func (f fakeManager) GetAPIReader() client.Reader {
	return nil
}

func (f fakeManager) GetWebhookServer() *webhook.Server {
	return nil
}

func (fakeManager) Add(manager.Runnable) error {
	return nil
}

func (fakeManager) SetFields(interface{}) error {
	return nil
}

func (fakeManager) Start(<-chan struct{}) error {
	return nil
}

func (fakeManager) GetConfig() *rest.Config {
	return &rest.Config{}
}

func (f *fakeManager) GetScheme() *runtime.Scheme {
	// Setup schemes for all resources
	return f.sch
}

func (f *fakeManager) GetClient() client.Client {
	return f.client
}

func (fakeManager) GetFieldIndexer() client.FieldIndexer {
	return nil
}

func (fakeManager) GetCache() cache.Cache {
	return nil
}

func (fakeManager) GetRESTMapper() meta.RESTMapper {
	return nil
}

func (fakeManager) GetLogger() logr.Logger {
	return nil
}
