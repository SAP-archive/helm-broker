package controller

import (
	"testing"
	"time"

	"context"

	"github.com/kyma-project/helm-broker/internal"
	"github.com/kyma-project/helm-broker/pkg/apis/addons/v1alpha1"
	"github.com/sirupsen/logrus"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
)

func TestCommon_OnAdd(t *testing.T) {
	configuration := fixClusterAddonsConfiguration()
	failedConfiguration := fixFailedClusterAddonsConfiguration()
	for tn, tc := range map[string]struct {
		obj        []runtime.Object
		addon      *internal.CommonAddon
		lastStatus v1alpha1.CommonAddonsConfigurationStatus
	}{
		"success": {
			obj:   []runtime.Object{configuration, failedConfiguration},
			addon: fixCommonClusterAddon(),
			lastStatus: v1alpha1.CommonAddonsConfigurationStatus{
				Phase:        v1alpha1.AddonsConfigurationReady,
				Repositories: fixRepositories(),
			},
		},
		"failed": {
			obj: []runtime.Object{configuration, failedConfiguration},
			addon: &internal.CommonAddon{
				Meta:   fixCommonClusterAddon().Meta,
				Spec:   fixCommonClusterAddon().Spec,
				Status: v1alpha1.CommonAddonsConfigurationStatus{Phase: v1alpha1.AddonsConfigurationFailed},
			},
			lastStatus: v1alpha1.CommonAddonsConfigurationStatus{
				Phase:        v1alpha1.AddonsConfigurationReady,
				Repositories: fixRepositories(),
			},
		},
	} {
		t.Run(tn, func(t *testing.T) {
			ts := getClusterTestSuite(t, tc.obj...)
			defer ts.assertExpectations()
			common := newControllerCommon(ts.mgr.GetClient(), ts.addonGetterFactory, ts.addonStorage, ts.chartStorage,
				ts.docsProvider, ts.brokerSyncer, ts.brokerFacade, ts.templateService, "", time.Second, logrus.New())

			ts.brokerFacade.On("Exist").Return(false, nil)

			err := common.OnAdd(tc.addon, tc.lastStatus)
			require.NoError(t, err)

			result := &v1alpha1.ClusterAddonsConfiguration{}
			err = ts.mgr.GetClient().Get(context.Background(), types.NamespacedName{Name: failedConfiguration.Name, Namespace: failedConfiguration.Namespace}, result)
			require.NoError(t, err)

			assert.Equal(t, int64(1), result.Spec.ReprocessRequest)
			assert.Equal(t, v1alpha1.AddonsConfigurationFailed, result.Status.Phase)
		})
	}
}

func TestCommon_OnDelete(t *testing.T) {
	commonAddon := fixCommonClusterAddon()
	for tn, tc := range map[string]struct {
		obj        []runtime.Object
		addon      *internal.CommonAddon
		lastStatus v1alpha1.CommonAddonsConfigurationStatus
	}{
		"success": {
			obj: []runtime.Object{fixClusterAddonsConfiguration(), fixFailedClusterAddonsConfiguration()},
			addon: &internal.CommonAddon{
				Meta:   commonAddon.Meta,
				Status: v1alpha1.CommonAddonsConfigurationStatus{Phase: v1alpha1.AddonsConfigurationReady},
				Spec:   commonAddon.Spec,
			},
		},
		"success-broker-stay": {
			obj: []runtime.Object{fixClusterAddonsConfiguration(), fixReadyClusterAddonsConfiguration()},
			addon: &internal.CommonAddon{
				Meta:   commonAddon.Meta,
				Status: v1alpha1.CommonAddonsConfigurationStatus{Phase: v1alpha1.AddonsConfigurationReady},
				Spec:   commonAddon.Spec,
			},
		},
		"success-addon-removed": {
			obj: []runtime.Object{fixClusterAddonsConfiguration(), fixFailedClusterAddonsConfiguration()},
			addon: &internal.CommonAddon{
				Meta:   commonAddon.Meta,
				Status: v1alpha1.CommonAddonsConfigurationStatus{Phase: v1alpha1.AddonsConfigurationReady, Repositories: fixRepositories()},
				Spec:   commonAddon.Spec,
			},
		},
		"success-only-finalizer": {
			obj: []runtime.Object{fixClusterAddonsConfiguration(), fixReadyClusterAddonsConfiguration()},
			addon: &internal.CommonAddon{
				Meta:   commonAddon.Meta,
				Status: v1alpha1.CommonAddonsConfigurationStatus{Phase: v1alpha1.AddonsConfigurationFailed},
				Spec:   commonAddon.Spec,
			},
		},
	} {
		t.Run(tn, func(t *testing.T) {
			ts := getClusterTestSuite(t, tc.obj...)
			common := newControllerCommon(ts.mgr.GetClient(), ts.addonGetterFactory, ts.addonStorage, ts.chartStorage,
				ts.docsProvider, ts.brokerSyncer, ts.brokerFacade, ts.templateService, "", time.Second, logrus.New())

			err := common.OnDelete(tc.addon)
			require.NoError(t, err)
		})
	}
}

func TestCommon_OnAdd_NamespaceScoped(t *testing.T) {
	configuration := fixAddonsConfiguration()
	failedConfiguration := fixFailedAddonsConfiguration()
	for tn, tc := range map[string]struct {
		obj        []runtime.Object
		addon      *internal.CommonAddon
		lastStatus v1alpha1.CommonAddonsConfigurationStatus
	}{
		"success": {
			obj:   []runtime.Object{configuration, failedConfiguration},
			addon: fixCommonAddon(),
			lastStatus: v1alpha1.CommonAddonsConfigurationStatus{
				Phase:        v1alpha1.AddonsConfigurationReady,
				Repositories: fixRepositories(),
			},
		},
		"failed": {
			obj: []runtime.Object{configuration, failedConfiguration},
			addon: &internal.CommonAddon{
				Meta:   fixCommonAddon().Meta,
				Spec:   fixCommonAddon().Spec,
				Status: v1alpha1.CommonAddonsConfigurationStatus{Phase: v1alpha1.AddonsConfigurationFailed},
			},
			lastStatus: v1alpha1.CommonAddonsConfigurationStatus{
				Phase:        v1alpha1.AddonsConfigurationReady,
				Repositories: fixRepositories(),
			},
		},
	} {
		t.Run(tn, func(t *testing.T) {
			ts := getTestSuite(t, tc.obj...)
			defer ts.assertExpectations()
			common := newControllerCommon(ts.mgr.GetClient(), ts.addonGetterFactory, ts.addonStorage, ts.chartStorage,
				ts.docsProvider, ts.brokerSyncer, ts.brokerFacade, ts.templateService, "", time.Second, logrus.New())

			ts.brokerFacade.On("Exist").Return(false, nil)

			common.SetWorkingNamespace(tc.addon.Meta.Namespace)

			err := common.OnAdd(tc.addon, tc.lastStatus)
			require.NoError(t, err)

			result := &v1alpha1.AddonsConfiguration{}
			err = ts.mgr.GetClient().Get(context.Background(), types.NamespacedName{Name: failedConfiguration.Name, Namespace: failedConfiguration.Namespace}, result)
			require.NoError(t, err)

			assert.Equal(t, int64(1), result.Spec.ReprocessRequest)
			assert.Equal(t, v1alpha1.AddonsConfigurationFailed, result.Status.Phase)
		})

	}

}

func TestCommon_OnDelete_NamespaceScoped(t *testing.T) {
	commonAddon := fixCommonAddon()
	for tn, tc := range map[string]struct {
		obj        []runtime.Object
		addon      *internal.CommonAddon
		lastStatus v1alpha1.CommonAddonsConfigurationStatus
	}{
		"success": {
			obj: []runtime.Object{fixAddonsConfiguration(), fixFailedAddonsConfiguration()},
			addon: &internal.CommonAddon{
				Meta:   commonAddon.Meta,
				Status: v1alpha1.CommonAddonsConfigurationStatus{Phase: v1alpha1.AddonsConfigurationReady},
				Spec:   commonAddon.Spec,
			},
		},
		"success-broker-stay": {
			obj: []runtime.Object{fixAddonsConfiguration(), fixReadyAddonsConfiguration()},
			addon: &internal.CommonAddon{
				Meta:   commonAddon.Meta,
				Status: v1alpha1.CommonAddonsConfigurationStatus{Phase: v1alpha1.AddonsConfigurationReady},
				Spec:   commonAddon.Spec,
			},
		},
		"success-addon-removed": {
			obj: []runtime.Object{fixAddonsConfiguration(), fixFailedAddonsConfiguration()},
			addon: &internal.CommonAddon{
				Meta:   commonAddon.Meta,
				Status: v1alpha1.CommonAddonsConfigurationStatus{Phase: v1alpha1.AddonsConfigurationReady, Repositories: fixRepositories()},
				Spec:   commonAddon.Spec,
			},
		},
		"success-only-finalizer": {
			obj: []runtime.Object{fixAddonsConfiguration(), fixReadyAddonsConfiguration()},
			addon: &internal.CommonAddon{
				Meta:   commonAddon.Meta,
				Status: v1alpha1.CommonAddonsConfigurationStatus{Phase: v1alpha1.AddonsConfigurationFailed},
				Spec:   commonAddon.Spec,
			},
		},
	} {
		t.Run(tn, func(t *testing.T) {
			ts := getTestSuite(t, tc.obj...)
			common := newControllerCommon(ts.mgr.GetClient(), ts.addonGetterFactory, ts.addonStorage, ts.chartStorage,
				ts.docsProvider, ts.brokerSyncer, ts.brokerFacade, ts.templateService, "", time.Second, logrus.New())

			common.SetWorkingNamespace(tc.addon.Meta.Namespace)

			err := common.OnDelete(tc.addon)
			require.NoError(t, err)
		})
	}
}

func TestCommon_PrepareForProcessing(t *testing.T) {
	for tn, tc := range map[string]struct {
		obj   []runtime.Object
		addon *internal.CommonAddon
	}{
		"success": {
			obj:   []runtime.Object{fixClusterAddonsConfiguration(), fixReadyClusterAddonsConfiguration()},
			addon: fixCommonClusterAddon(),
		},
	} {
		t.Run(tn, func(t *testing.T) {
			ts := getTestSuite(t, tc.obj...)
			common := newControllerCommon(ts.mgr.GetClient(), ts.addonGetterFactory, ts.addonStorage, ts.chartStorage,
				ts.docsProvider, ts.brokerSyncer, ts.brokerFacade, ts.templateService, "", time.Second, logrus.New())

			err := common.PrepareForProcessing(tc.addon)
			require.NoError(t, err)
		})
	}
}

func TestCommon_PrepareForProcessing_NamespaceScoped(t *testing.T) {
	for tn, tc := range map[string]struct {
		obj   []runtime.Object
		addon *internal.CommonAddon
	}{
		"success": {
			obj:   []runtime.Object{fixAddonsConfiguration(), fixReadyAddonsConfiguration()},
			addon: fixCommonAddon(),
		},
	} {
		t.Run(tn, func(t *testing.T) {
			ts := getTestSuite(t, tc.obj...)
			common := newControllerCommon(ts.mgr.GetClient(), ts.addonGetterFactory, ts.addonStorage, ts.chartStorage,
				ts.docsProvider, ts.brokerSyncer, ts.brokerFacade, ts.templateService, "", time.Second, logrus.New())

			common.SetWorkingNamespace(tc.addon.Meta.Namespace)

			err := common.PrepareForProcessing(tc.addon)
			require.NoError(t, err)
		})
	}
}

func fixCommonAddon() *internal.CommonAddon {
	return &internal.CommonAddon{
		Meta: v1.ObjectMeta{
			Name:      "test",
			Namespace: "test",
		},
		Spec:   v1alpha1.CommonAddonsConfigurationSpec{},
		Status: v1alpha1.CommonAddonsConfigurationStatus{},
	}
}

func fixCommonClusterAddon() *internal.CommonAddon {
	return &internal.CommonAddon{
		Meta: v1.ObjectMeta{
			Name: "test",
		},
		Spec:   v1alpha1.CommonAddonsConfigurationSpec{},
		Status: v1alpha1.CommonAddonsConfigurationStatus{},
	}
}
