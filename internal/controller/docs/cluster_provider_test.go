package docs

import (
	"context"
	"testing"

	"github.com/kyma-project/helm-broker/internal"
	"github.com/kyma-project/rafter/pkg/apis/rafter/v1beta1"
	"github.com/sirupsen/logrus"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"k8s.io/apimachinery/pkg/api/errors"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes/scheme"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
)

func TestProvider_EnsureClusterAssetGroup(t *testing.T) {
	// given
	err := v1beta1.AddToScheme(scheme.Scheme)
	require.NoError(t, err)
	const id = "123"

	for tn, tc := range map[string]struct {
		givenAddon internal.AddonWithCharts
	}{
		"URL set":   {fixAddonWithDocsURL(id, "test", "url", "url2")},
		"empty URL": {fixAddonWithEmptyDocs(id, "test", "url")},
	} {
		t.Run(tn, func(t *testing.T) {
			c := fake.NewFakeClient()
			cdt := fixClusterAssetGroup(id)
			docsProvider := NewClusterProvider(c, logrus.New())

			// when
			err = docsProvider.EnsureAssetGroup(tc.givenAddon.Addon)
			require.NoError(t, err)

			// then
			err = c.Get(context.Background(), client.ObjectKey{Name: cdt.Name}, cdt)
			require.NoError(t, err)
			assert.Equal(t, tc.givenAddon.Addon.Docs[0].Template, cdt.Spec.CommonAssetGroupSpec)
		})
	}
}

func TestProvider_EnsureClusterAssetGroup_UpdateIfExist(t *testing.T) {
	// given
	err := v1beta1.AddToScheme(scheme.Scheme)
	require.NoError(t, err)

	const id = "123"
	cdt := fixClusterAssetGroup(id)
	addonWithEmptyDocsURL := fixAddonWithEmptyDocs(id, "test", "url")
	addonWithEmptyDocsURL.Addon.Docs[0].Template.Description = "new description"

	c := fake.NewFakeClient(cdt)
	docsProvider := NewClusterProvider(c, logrus.New())

	// when
	err = docsProvider.EnsureAssetGroup(addonWithEmptyDocsURL.Addon)
	require.NoError(t, err)

	// then
	err = c.Get(context.Background(), client.ObjectKey{Name: cdt.Name}, cdt)
	require.NoError(t, err)
	assert.Equal(t, addonWithEmptyDocsURL.Addon.Docs[0].Template, cdt.Spec.CommonAssetGroupSpec)
}

func TestDocsProvider_EnsureClusterAssetGroupRemoved(t *testing.T) {
	// given
	err := v1beta1.AddToScheme(scheme.Scheme)
	require.NoError(t, err)

	const id = "123"
	cdt := fixClusterAssetGroup(id)
	c := fake.NewFakeClient(cdt)
	docsProvider := NewClusterProvider(c, logrus.New())

	// when
	err = docsProvider.EnsureAssetGroupRemoved(id)
	require.NoError(t, err)

	// then
	err = c.Get(context.Background(), client.ObjectKey{Name: cdt.Name}, cdt)
	assert.True(t, errors.IsNotFound(err))
}

func TestDocsProvider_EnsureClusterAssetGroupRemoved_NotExists(t *testing.T) {
	// given
	err := v1beta1.AddToScheme(scheme.Scheme)
	require.NoError(t, err)

	const id = "123"
	cdt := fixClusterAssetGroup(id)
	c := fake.NewFakeClient()
	docsProvider := NewClusterProvider(c, logrus.New())

	// when
	err = docsProvider.EnsureAssetGroupRemoved(id)
	require.NoError(t, err)

	// then
	err = c.Get(context.Background(), client.ObjectKey{Name: cdt.Name}, cdt)
	assert.True(t, errors.IsNotFound(err))
}

func fixClusterAssetGroup(id string) *v1beta1.ClusterAssetGroup {
	return &v1beta1.ClusterAssetGroup{
		ObjectMeta: v1.ObjectMeta{
			Name: id,
		},
	}
}
