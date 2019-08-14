package docs

import (
	"context"
	"testing"

	"fmt"

	"github.com/Masterminds/semver"
	"github.com/kyma-project/helm-broker/internal"
	"github.com/kyma-project/kyma/components/cms-controller-manager/pkg/apis/cms/v1alpha1"
	"github.com/sirupsen/logrus"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"k8s.io/apimachinery/pkg/api/errors"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes/scheme"
	"k8s.io/helm/pkg/proto/hapi/chart"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
)

func TestDocsProvider_EnsureDocsTopic(t *testing.T) {
	// given
	err := v1alpha1.AddToScheme(scheme.Scheme)
	require.NoError(t, err)

	for tn, tc := range map[string]struct {
		givenAddon internal.AddonWithCharts
	}{
		"URL set":   {fixAddonWithDocsURL("test", "test", "url", "url2")},
		"empty URL": {fixAddonWithEmptyDocs("test", "test", "url")},
	} {
		t.Run(tn, func(t *testing.T) {
			c := fake.NewFakeClient()
			docsProvider := NewProvider(c, logrus.New())

			// when
			docsProvider.SetNamespace("test")
			err = docsProvider.EnsureDocsTopic(tc.givenAddon.Addon)
			require.NoError(t, err)

			// then
			result := v1alpha1.DocsTopic{}
			err = c.Get(context.Background(), client.ObjectKey{Namespace: "test", Name: "test"}, &result)
			require.NoError(t, err)
			assert.Equal(t, tc.givenAddon.Addon.Docs[0].Template, result.Spec.CommonDocsTopicSpec)
		})
	}
}

func TestDocsProvider_EnsureDocsTopic_UpdateIfExist(t *testing.T) {
	// given
	err := v1alpha1.AddToScheme(scheme.Scheme)
	require.NoError(t, err)

	dt := fixDocsTopic()

	addonWithEmptyDocsURL := fixAddonWithEmptyDocs(dt.Name, "test", "url")
	addonWithEmptyDocsURL.Addon.Docs[0].Template.Description = "new description"

	c := fake.NewFakeClient(dt)
	docsProvider := NewProvider(c, logrus.New())

	// when
	docsProvider.SetNamespace("test")
	err = docsProvider.EnsureDocsTopic(addonWithEmptyDocsURL.Addon)
	require.NoError(t, err)

	// then
	result := v1alpha1.DocsTopic{}
	err = c.Get(context.Background(), client.ObjectKey{Namespace: dt.Namespace, Name: dt.Name}, &result)
	require.NoError(t, err)
	assert.Equal(t, addonWithEmptyDocsURL.Addon.Docs[0].Template, result.Spec.CommonDocsTopicSpec)
}

func TestDocsProvider_EnsureDocsTopicRemoved(t *testing.T) {
	// given
	err := v1alpha1.AddToScheme(scheme.Scheme)
	require.NoError(t, err)

	dt := fixDocsTopic()
	c := fake.NewFakeClient(dt)
	docsProvider := NewProvider(c, logrus.New())

	// when
	docsProvider.SetNamespace("test")
	err = docsProvider.EnsureDocsTopicRemoved(dt.Name)
	require.NoError(t, err)

	// then
	result := v1alpha1.DocsTopic{}
	err = c.Get(context.Background(), client.ObjectKey{Namespace: dt.Namespace, Name: dt.Name}, &result)
	assert.True(t, errors.IsNotFound(err))
}

func TestDocsProvider_EnsureDocsTopicRemoved_NotExists(t *testing.T) {
	// given
	err := v1alpha1.AddToScheme(scheme.Scheme)
	require.NoError(t, err)

	dt := fixDocsTopic()
	c := fake.NewFakeClient()
	docsProvider := NewProvider(c, logrus.New())

	// when
	docsProvider.SetNamespace("test")
	err = docsProvider.EnsureDocsTopicRemoved(dt.Name)
	require.NoError(t, err)

	// then
	result := v1alpha1.DocsTopic{}
	err = c.Get(context.Background(), client.ObjectKey{Namespace: dt.Namespace, Name: dt.Name}, &result)
	assert.True(t, errors.IsNotFound(err))
}

func fixDocsTopic() *v1alpha1.DocsTopic {
	return &v1alpha1.DocsTopic{
		ObjectMeta: v1.ObjectMeta{
			Name:      "test",
			Namespace: "test",
		},
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
					Template: v1alpha1.CommonDocsTopicSpec{
						Sources: []v1alpha1.Source{
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

func fixAddonWithEmptyDocs(id, name, url string) internal.AddonWithCharts {
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
					Template: v1alpha1.CommonDocsTopicSpec{
						Sources: []v1alpha1.Source{
							{},
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
