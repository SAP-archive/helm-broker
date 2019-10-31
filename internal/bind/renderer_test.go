package bind_test

import (
	"errors"
	"fmt"
	"testing"

	google_protobuf "github.com/golang/protobuf/ptypes/timestamp"
	"github.com/kyma-project/helm-broker/internal"
	"github.com/kyma-project/helm-broker/internal/bind"
	"github.com/kyma-project/helm-broker/internal/bind/automock"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	"k8s.io/helm/pkg/chartutil"
	"k8s.io/helm/pkg/proto/hapi/chart"
	hapi_release5 "k8s.io/helm/pkg/proto/hapi/release"
	"k8s.io/helm/pkg/proto/hapi/services"
)

func TestRenderSuccess(t *testing.T) {
	// given
	fixResp := fixInstallReleaseResponse(fixChart())
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

	toRenderFake := toRenderValuesFake{t}.WithInputAssertion(fixChart(), fixResp)
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
	fixResp := fixInstallReleaseResponse(fixChart())
	fixedInstance := fixInstance()
	fixedChart := fixChart()
	fixErr := errors.New("fix err")
	tplToRender := internal.AddonPlanBindTemplate("template-body-to-render")

	toRenderFake := toRenderValuesFake{t}.WithInputAssertion(fixChart(), fixResp)

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
	fixResp := fixInstallReleaseResponse(fixChart())
	fixedInstance := fixInstance()
	fixedChart := fixChart()
	tplToRender := internal.AddonPlanBindTemplate("template-body-to-render")

	engineRenderMock := &automock.ChartGoTemplateRenderer{}
	defer engineRenderMock.AssertExpectations(t)
	engineRenderMock.On("Render", mock.MatchedBy(chartWithTpl(t, tplToRender)), fixChartutilValues()).
		Return(map[string]string{}, nil)

	toRenderFake := toRenderValuesFake{t}.WithInputAssertion(fixChart(), fixResp)
	renderer := bind.NewRendererWithDeps(engineRenderMock, toRenderFake)

	// when
	out, err := renderer.Render(tplToRender, &fixedInstance, &fixedChart)

	// then
	assert.EqualError(t, err, "bindTmpl file was not resolved after rendering")
	assert.Nil(t, out)
}

func chartWithTpl(t *testing.T, expTpl internal.AddonPlanBindTemplate) func(*chart.Chart) bool {
	return func(ch *chart.Chart) bool {
		assert.Contains(t, ch.Templates, &chart.Template{Name: "bindTmpl", Data: expTpl})
		return true
	}
}

type toRenderValuesFake struct {
	t *testing.T
}

func (r toRenderValuesFake) WithInputAssertion(expChrt chart.Chart, expResp *services.InstallReleaseResponse) func(*chart.Chart, *chart.Config, chartutil.ReleaseOptions, *chartutil.Capabilities) (chartutil.Values, error) {
	return func(chrt *chart.Chart, chrtVals *chart.Config, options chartutil.ReleaseOptions, caps *chartutil.Capabilities) (chartutil.Values, error) {
		assert.Equal(r.t, expChrt, *chrt)
		assert.Equal(r.t, expResp.Release.Config, chrtVals)
		assert.Equal(r.t, chartutil.ReleaseOptions{
			Name:      expResp.Release.Name,
			Time:      expResp.Release.Info.LastDeployed,
			Namespace: expResp.Release.Namespace,
			Revision:  int(expResp.Release.Version),
			IsInstall: true,
		}, options)
		assert.Equal(r.t, &chartutil.Capabilities{}, caps)
		return fixChartutilValues(), nil
	}
}

func (r toRenderValuesFake) WithForcedError(err error) func(*chart.Chart, *chart.Config, chartutil.ReleaseOptions, *chartutil.Capabilities) (chartutil.Values, error) {
	return func(chrt *chart.Chart, chrtVals *chart.Config, options chartutil.ReleaseOptions, caps *chartutil.Capabilities) (chartutil.Values, error) {
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
		ParamsHash:    "test-hash",
		ReleaseInfo:   internal.ReleaseInfo{
			Time:     &google_protobuf.Timestamp{
				Seconds: 123123123,
				Nanos:   1,
			},
			Revision: 123,
			Config:  &chart.Config{
				Raw: "raw-config",
			},
		},
	}
}
func fixInstallReleaseResponse(ch chart.Chart) *services.InstallReleaseResponse {
	return &services.InstallReleaseResponse{
		Release: &hapi_release5.Release{
			Info: &hapi_release5.Info{
				LastDeployed: &google_protobuf.Timestamp{
					Seconds: 123123123,
					Nanos:   1,
				},
			},
			Config: &chart.Config{
				Raw: "raw-config",
			},
			Name:      "test-release",
			Namespace: "test-ns",
			Version:   int32(123),
			Chart:     &ch,
		},
	}
}
