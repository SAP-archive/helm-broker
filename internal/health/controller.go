package health

import (
	"context"
	"fmt"
	"net/http"
	"time"

	"k8s.io/apimachinery/pkg/types"

	"github.com/gorilla/mux"
	"github.com/kyma-project/helm-broker/pkg/apis/addons/v1alpha1"
	"github.com/sirupsen/logrus"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/wait"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

const (
	probeName = "liveness-probe"
)

// ControllerHealth holds logic for controller's probes
type ControllerHealth struct {
	port                   string
	etcdURL                string
	client                 client.Client
	livenessProbeNamespace string
	lg                     *logrus.Entry
}

// NewControllerProbes creates a ControllerHealth
func NewControllerProbes(port string, etcdURL string, client client.Client, livenessProbeNamespace string) *ControllerHealth {
	return &ControllerHealth{
		port:                   port,
		etcdURL:                etcdURL,
		client:                 client,
		livenessProbeNamespace: livenessProbeNamespace,
		lg:                     logrus.WithField("health", "controller"),
	}
}

// Handle handles probes for controller
func (c *ControllerHealth) Handle() {
	rtr := mux.NewRouter()
	rtr.HandleFunc(c.liveProbe(c.client, c.lg)).Methods("GET")
	rtr.HandleFunc(c.handleReady(c.etcdURL)).Methods("GET")
	http.ListenAndServe(c.port, rtr)
}

func (c *ControllerHealth) handleReady(etcdURL string) (string, func(w http.ResponseWriter, req *http.Request)) {
	return "/ready", handleHealth(etcdURL)
}

func (c *ControllerHealth) liveProbe(client client.Client, lg *logrus.Entry) (string, func(w http.ResponseWriter, req *http.Request)) {
	return "/live", c.runFullControllersCycle(client, lg)
}

func (c *ControllerHealth) runFullControllersCycle(client client.Client, lg *logrus.Entry) func(w http.ResponseWriter, req *http.Request) {
	return func(w http.ResponseWriter, req *http.Request) {
		if err := c.runAddonsConfigurationControllerCycle(client, lg); err != nil {
			w.WriteHeader(http.StatusInternalServerError)
			return
		}

		w.WriteHeader(http.StatusOK)
		fmt.Fprint(w, "OK")
	}
}

func (c *ControllerHealth) runAddonsConfigurationControllerCycle(client client.Client, lg *logrus.Entry) error {
	clusterAddonsConfiguration := &v1alpha1.ClusterAddonsConfiguration{
		ObjectMeta: v1.ObjectMeta{
			Name: probeName,
		},
		Spec: v1alpha1.ClusterAddonsConfigurationSpec{
			CommonAddonsConfigurationSpec: v1alpha1.CommonAddonsConfigurationSpec{
				Repositories: []v1alpha1.SpecRepository{{URL: ""}},
			},
		},
	}

	ctx := context.Background()
	lg.Info("[liveness-probe] Creating liveness probe clusterAddonsConfiguration")
	err := client.Create(ctx, clusterAddonsConfiguration)
	if err != nil && !apierrors.IsAlreadyExists(err) {
		lg.Errorf("[liveness-probe] Cannot create liveness probe clusterAddonsConfiguration: %s", err)
		return err
	}

	lg.Info("[liveness-probe] Waiting for liveness probe clusterAddonsConfiguration desirable status")
	err = wait.Poll(1*time.Second, 10*time.Second, func() (done bool, err error) {
		key := types.NamespacedName{Name: probeName, Namespace: ""}
		err = client.Get(ctx, key, clusterAddonsConfiguration)
		if apierrors.IsNotFound(err) {
			lg.Info("[liveness-probe] Liveness probe clusterAddonsConfiguration not found")
			return false, nil
		}
		if err != nil {
			lg.Errorf("[liveness-probe] Cannot get probe clusterAddonsConfiguration: %s", err)
			return false, nil
		}

		if len(clusterAddonsConfiguration.Status.Repositories) != 1 {
			lg.Infof("[liveness-probe] Liveness probe clusterAddonsConfiguration repositories status not set, number of repositories: %d", len(clusterAddonsConfiguration.Status.Repositories))
			return false, nil
		}

		status := clusterAddonsConfiguration.Status.Repositories[0].Status
		reason := clusterAddonsConfiguration.Status.Repositories[0].Reason
		if status == v1alpha1.RepositoryStatusFailed {
			if reason == v1alpha1.RepositoryEmptyURLError {
				lg.Info("[liveness-probe] Liveness probe clusterAddonsConfiguration has achieved the desired status")
				return true, nil
			}
		}

		lg.Infof("[liveness-probe] Liveness probe clusterAddonsConfiguration current status: %s: %s", status, reason)
		return false, nil
	})
	if err != nil {
		lg.Errorf("[liveness-probe] Waiting for liveness probe clusterAddonsConfiguration failed: %s", err)
		return err
	}

	lg.Info("[liveness-probe] Removing liveness probe clusterAddonsConfiguration")
	err = client.Delete(ctx, clusterAddonsConfiguration)
	if err != nil {
		lg.Errorf("[liveness-probe] Cannot delete liveness probe clusterAddonsConfiguration: %s", err)
		return err
	}

	lg.Info("[liveness-probe] ClusterAddonsConfiguration controller is live")
	return nil
}
