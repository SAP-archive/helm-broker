package bind_test

import (
	"errors"
	"fmt"
	"testing"

	"github.com/kyma-project/helm-broker/internal"
	"github.com/kyma-project/helm-broker/internal/bind"
	"github.com/kyma-project/helm-broker/internal/bind/automock"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	"helm.sh/helm/v3/pkg/chart"
	"helm.sh/helm/v3/pkg/chartutil"

	helm2chart "k8s.io/helm/pkg/proto/hapi/chart"
)

func TestRenderSuccess(t *testing.T) {
	// given
	fixedInstance := fixInstance()
	fixedChart := fixChart()
	fixRenderOutFiles := map[string]string{
		fmt.Sprintf("%s/%s", fixChart().Metadata.Name, "bindTmpl"): "rendered-content",
	}
	tplToRender := internal.AddonPlanBindTemplate("template-body-to-render")

	engineRenderMock := &automock.ChartGoTemplateRenderer{}
	defer engineRenderMock.AssertExpectations(t)
	engineRenderMock.On("Render", mock.MatchedBy(chartWithTpl(t, tplToRender)), fixChartutilValues()).
		Return(fixRenderOutFiles, nil)

	toRenderFake := toRenderValuesFake{t}.WithInputAssertion(fixChart())
	renderer := bind.NewRendererWithDeps(engineRenderMock, toRenderFake)

	// when
	out, err := renderer.Render(tplToRender, &fixedInstance, &fixedChart)

	// then
	require.NoError(t, err)
	assert.EqualValues(t, "rendered-content", out)
}

func TestRenderFailureOnCreatingToRenderValues(t *testing.T) {
	// given
	fixedInstance := fixInstance()
	fixedChart := fixChart()
	fixErr := errors.New("fix err")
	tplToRender := internal.AddonPlanBindTemplate("template-body-to-render")

	toRenderFake := toRenderValuesFake{t}.WithForcedError(fixErr)
	renderer := bind.NewRendererWithDeps(nil, toRenderFake)

	// when
	out, err := renderer.Render(tplToRender, &fixedInstance, &fixedChart)

	// then
	require.EqualError(t, err, "while merging values to render: fix err")
	assert.Nil(t, out)
}

func TestRenderFailureOnEngineRender(t *testing.T) {
	// given
	fixedInstance := fixInstance()
	fixedChart := fixChart()
	fixErr := errors.New("fix err")
	tplToRender := internal.AddonPlanBindTemplate("template-body-to-render")

	toRenderFake := toRenderValuesFake{t}.WithInputAssertion(fixChart())

	engineRenderMock := &automock.ChartGoTemplateRenderer{}
	defer engineRenderMock.AssertExpectations(t)
	engineRenderMock.On("Render", mock.MatchedBy(chartWithTpl(t, tplToRender)), fixChartutilValues()).
		Return(nil, fixErr)

	renderer := bind.NewRendererWithDeps(engineRenderMock, toRenderFake)

	// when
	out, err := renderer.Render(tplToRender, &fixedInstance, &fixedChart)

	// then
	assert.EqualError(t, err, fmt.Sprintf("while rendering files: %s", fixErr))
	assert.Nil(t, out)
}

func TestRenderFailureOnExtractingResolveBindFile(t *testing.T) {
	// given
	fixedInstance := fixInstance()
	fixedChart := fixChart()
	tplToRender := internal.AddonPlanBindTemplate("template-body-to-render")

	engineRenderMock := &automock.ChartGoTemplateRenderer{}
	defer engineRenderMock.AssertExpectations(t)
	engineRenderMock.On("Render", mock.MatchedBy(chartWithTpl(t, tplToRender)), fixChartutilValues()).
		Return(map[string]string{}, nil)

	toRenderFake := toRenderValuesFake{t}.WithInputAssertion(fixChart())
	renderer := bind.NewRendererWithDeps(engineRenderMock, toRenderFake)

	// when
	out, err := renderer.Render(tplToRender, &fixedInstance, &fixedChart)

	// then
	assert.EqualError(t, err, "bindTmpl file was not resolved after rendering")
	assert.Nil(t, out)
}

func chartWithTpl(t *testing.T, expTpl internal.AddonPlanBindTemplate) func(*chart.Chart) bool {
	return func(ch *chart.Chart) bool {
		assert.Contains(t, ch.Templates, &chart.File{Name: "bindTmpl", Data: expTpl})
		return true
	}
}

type toRenderValuesFake struct {
	t *testing.T
}

func (r toRenderValuesFake) WithInputAssertion(expChrt chart.Chart) func(*chart.Chart, map[string]interface{}, chartutil.ReleaseOptions, *chartutil.Capabilities) (chartutil.Values, error) {
	return func(chrt *chart.Chart, chrtVals map[string]interface{}, options chartutil.ReleaseOptions, caps *chartutil.Capabilities) (chartutil.Values, error) {
		assert.Equal(r.t, expChrt, *chrt)
		//assert.Equal(r.t, expResp.Release.Config, chrtVals)
		assert.Equal(r.t, chartutil.ReleaseOptions{
			Name:      "test-release",
			Namespace: "test-ns",
			Revision:  123,
			IsInstall: true,
		}, options)
		assert.Equal(r.t, &chartutil.Capabilities{}, caps)
		return fixChartutilValues(), nil
	}
}

func (r toRenderValuesFake) WithForcedError(err error) func(*chart.Chart, map[string]interface{}, chartutil.ReleaseOptions, *chartutil.Capabilities) (chartutil.Values, error) {
	return func(chrt *chart.Chart, chrtVals map[string]interface{}, options chartutil.ReleaseOptions, caps *chartutil.Capabilities) (chartutil.Values, error) {
		return nil, err
	}
}

func fixChartutilValues() chartutil.Values {
	return chartutil.Values{"fix_val_key": "fix_val"}
}

func fixChart() chart.Chart {
	return chart.Chart{
		Metadata: &chart.Metadata{
			Name: "test-chart",
		},
	}
}

func fixInstance() internal.Instance {
	return internal.Instance{
		ID:            "test-instance-id",
		ServiceID:     "test-service-id",
		ServicePlanID: "test-service-plan-id",
		ReleaseName:   "test-release",
		Namespace:     "test-ns",
		ReleaseInfo: internal.ReleaseInfo{
			Revision: 123,
			Config: &helm2chart.Config{
				Raw: "raw-config",
			},
		},
		ProvisioningParameters: &internal.RequestParameters{
			Data: map[string]interface{}{
				"sample-parameter": "sample-value",
			},
		},
	}
}
