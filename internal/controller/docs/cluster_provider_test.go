package docs

import (
	"context"
	"testing"

	"github.com/kyma-project/helm-broker/internal"
	"github.com/kyma-project/kyma/components/cms-controller-manager/pkg/apis/cms/v1alpha1"
	"github.com/sirupsen/logrus"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"k8s.io/apimachinery/pkg/api/errors"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes/scheme"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
)

func TestProvider_EnsureClusterDocsTopic(t *testing.T) {
	// given
	err := v1alpha1.AddToScheme(scheme.Scheme)
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
			cdt := fixClusterDocsTopic(id)
			docsProvider := NewClusterProvider(c, logrus.New())

			// when
			err = docsProvider.EnsureDocsTopic(tc.givenAddon.Addon)
			require.NoError(t, err)

			// then
			err = c.Get(context.Background(), client.ObjectKey{Name: cdt.Name}, cdt)
			require.NoError(t, err)
			assert.Equal(t, tc.givenAddon.Addon.Docs[0].Template, cdt.Spec.CommonDocsTopicSpec)
		})
	}
}

func TestProvider_EnsureClusterDocsTopic_UpdateIfExist(t *testing.T) {
	// given
	err := v1alpha1.AddToScheme(scheme.Scheme)
	require.NoError(t, err)

	const id = "123"
	cdt := fixClusterDocsTopic(id)
	addonWithEmptyDocsURL := fixAddonWithEmptyDocs(id, "test", "url")
	addonWithEmptyDocsURL.Addon.Docs[0].Template.Description = "new description"

	c := fake.NewFakeClient(cdt)
	docsProvider := NewClusterProvider(c, logrus.New())

	// when
	err = docsProvider.EnsureDocsTopic(addonWithEmptyDocsURL.Addon)
	require.NoError(t, err)

	// then
	err = c.Get(context.Background(), client.ObjectKey{Name: cdt.Name}, cdt)
	require.NoError(t, err)
	assert.Equal(t, addonWithEmptyDocsURL.Addon.Docs[0].Template, cdt.Spec.CommonDocsTopicSpec)
}

func TestDocsProvider_EnsureClusterDocsTopicRemoved(t *testing.T) {
	// given
	err := v1alpha1.AddToScheme(scheme.Scheme)
	require.NoError(t, err)

	const id = "123"
	cdt := fixClusterDocsTopic(id)
	c := fake.NewFakeClient(cdt)
	docsProvider := NewClusterProvider(c, logrus.New())

	// when
	err = docsProvider.EnsureDocsTopicRemoved(id)
	require.NoError(t, err)

	// then
	err = c.Get(context.Background(), client.ObjectKey{Name: cdt.Name}, cdt)
	assert.True(t, errors.IsNotFound(err))
}

func TestDocsProvider_EnsureClusterDocsTopicRemoved_NotExists(t *testing.T) {
	// given
	err := v1alpha1.AddToScheme(scheme.Scheme)
	require.NoError(t, err)

	const id = "123"
	cdt := fixClusterDocsTopic(id)
	c := fake.NewFakeClient()
	docsProvider := NewClusterProvider(c, logrus.New())

	// when
	err = docsProvider.EnsureDocsTopicRemoved(id)
	require.NoError(t, err)

	// then
	err = c.Get(context.Background(), client.ObjectKey{Name: cdt.Name}, cdt)
	assert.True(t, errors.IsNotFound(err))
}

func fixClusterDocsTopic(id string) *v1alpha1.ClusterDocsTopic {
	return &v1alpha1.ClusterDocsTopic{
		ObjectMeta: v1.ObjectMeta{
			Name: id,
		},
	}
}
