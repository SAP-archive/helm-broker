package bind

import (
	"fmt"

	"github.com/kyma-project/helm-broker/internal"
	"github.com/pkg/errors"
	"helm.sh/helm/v3/pkg/chart"
	"helm.sh/helm/v3/pkg/chartutil"
	"helm.sh/helm/v3/pkg/engine"
)

const (
	bindFile = "bindTmpl"
)

//go:generate mockery -name=chartGoTemplateRenderer -output=automock -outpkg=automock -case=underscore
type chartGoTemplateRenderer interface {
	Render(*chart.Chart, chartutil.Values) (map[string]string, error)
}

type toRenderValuesCaps func(*chart.Chart, map[string]interface{}, chartutil.ReleaseOptions, *chartutil.Capabilities) (chartutil.Values, error)

// Renderer purpose is to render helm template directives, like: {{ .Release.Namespace }}
type Renderer struct {
	renderEngine       chartGoTemplateRenderer
	toRenderValuesCaps toRenderValuesCaps
}

// NewRenderer creates new instance of Renderer.
func NewRenderer() *Renderer {
	return &Renderer{
		renderEngine:       &engine.Engine{},
		toRenderValuesCaps: chartutil.ToRenderValues,
	}
}

// Render renders given bindTemplate in context of helm Chart by e.g. replacing directives like: {{ .Release.Namespace }}
func (r *Renderer) Render(bindTemplate internal.AddonPlanBindTemplate, instance *internal.Instance, ch *chart.Chart) (RenderedBindYAML, error) {

	options := r.createReleaseOptions(instance)
	chartCap := &chartutil.Capabilities{}

	valsToRender, err := r.toRenderValuesCaps(ch, instance.ReleaseInfo.ConfigValues, options, chartCap)
	if err != nil {
		return nil, errors.Wrap(err, "while merging values to render")
	}

	ch.Templates = append(ch.Templates, &chart.File{Name: bindFile, Data: bindTemplate})

	files, err := r.renderEngine.Render(ch, valsToRender)
	if err != nil {
		return nil, errors.Wrap(err, "while rendering files")
	}

	rendered, exits := files[fmt.Sprintf("%s/%s", ch.Metadata.Name, bindFile)]
	if !exits {
		return nil, fmt.Errorf("%v file was not resolved after rendering", bindFile)
	}

	return RenderedBindYAML(rendered), nil
}

func (*Renderer) createReleaseOptions(instance *internal.Instance) chartutil.ReleaseOptions {
	return chartutil.ReleaseOptions{
		Name:      string(instance.ReleaseName),
		Namespace: string(instance.Namespace),
		Revision:  instance.ReleaseInfo.Revision,
		IsInstall: true,
	}
}
