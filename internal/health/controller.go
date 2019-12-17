package health

import (
	"fmt"
	"net/http"
	"time"

	"github.com/gorilla/mux"
	"github.com/kyma-project/helm-broker/pkg/apis/addons/v1alpha1"
	"github.com/sirupsen/logrus"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
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
		if err := c.runAddonsConfigurationControllerCycle(req, client, lg); err != nil {
			w.WriteHeader(http.StatusInternalServerError)
			return
		}

		w.WriteHeader(http.StatusOK)
		fmt.Fprint(w, "OK")
	}
}

func (c *ControllerHealth) runAddonsConfigurationControllerCycle(req *http.Request, client client.Client, lg *logrus.Entry) error {
	addonsConfiguration := &v1alpha1.AddonsConfiguration{
		ObjectMeta: v1.ObjectMeta{
			Name:      probeName,
			Namespace: c.livenessProbeNamespace,
		},
		Spec: v1alpha1.AddonsConfigurationSpec{
			CommonAddonsConfigurationSpec: v1alpha1.CommonAddonsConfigurationSpec{
				Repositories: []v1alpha1.SpecRepository{{URL: ""}},
			},
		},
	}

	lg.Infof("[liveness-probe] Creating liveness probe addonsConfiguration in %q namespace", c.livenessProbeNamespace)
	err := client.Create(req.Context(), addonsConfiguration)
	if err != nil && !apierrors.IsAlreadyExists(err) {
		lg.Errorf("[liveness-probe] Cannot create liveness probe addonsConfiguration: %s", err)
		return err
	}

	lg.Info("[liveness-probe] Waiting for liveness probe addonsConfiguration desirable status")
	err = wait.Poll(1*time.Second, 10*time.Second, func() (done bool, err error) {
		key := types.NamespacedName{Name: probeName, Namespace: c.livenessProbeNamespace}
		err = client.Get(req.Context(), key, addonsConfiguration)
		if apierrors.IsNotFound(err) {
			lg.Info("[liveness-probe] Liveness probe addonsConfiguration not found")
			return false, nil
		}
		if err != nil {
			lg.Errorf("[liveness-probe] Cannot get probe addonsConfiguration: %s", err)
			return false, nil
		}

		if len(addonsConfiguration.Status.Repositories) != 1 {
			lg.Info("[liveness-probe] Liveness probe addonsConfiguration repositories status not set")
			return false, nil
		}

		status := addonsConfiguration.Status.Repositories[0].Status
		reason := addonsConfiguration.Status.Repositories[0].Reason
		if status == v1alpha1.RepositoryStatusFailed {
			if reason == v1alpha1.RepositoryURLFetchingError {
				lg.Info("[liveness-probe] Liveness probe addonsConfiguration has achieved the desired status")
				return true, nil
			}
		}

		lg.Infof("[liveness-probe] Liveness probe addonsConfiguration current status: %s: %s", status, reason)
		return false, nil
	})
	if err != nil {
		lg.Errorf("[liveness-probe] Waiting for liveness probe addonsConfiguration failed: %s", err)
		return err
	}

	lg.Info("[liveness-probe] Removing liveness probe addonsConfiguration")
	err = client.Delete(req.Context(), addonsConfiguration)
	if err != nil {
		lg.Errorf("[liveness-probe] Cannot delete liveness probe addonsConfiguration: %s", err)
		return err
	}

	lg.Info("[liveness-probe] AddonsConfiguration controller is live")
	return nil
}
