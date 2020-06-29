package helm

import (
	"k8s.io/helm/pkg/helm"
	"k8s.io/helm/pkg/proto/hapi/chart"
	rls "k8s.io/helm/pkg/proto/hapi/services"
)

// DeleteInstaller defines necessary methods for installing and deleting releases
type DeleteInstaller interface {
	InstallReleaseFromChart(chart *chart.Chart, ns string, opts ...helm.InstallOption) (*rls.InstallReleaseResponse, error)
	DeleteRelease(rlsName string, opts ...helm.DeleteOption) (*rls.UninstallReleaseResponse, error)
}
