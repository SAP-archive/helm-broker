package webhook

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"

	log "github.com/sirupsen/logrus"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/client-go/kubernetes/scheme"
	cli "sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/apiutil"
	"sigs.k8s.io/controller-runtime/pkg/webhook/admission"
)

const (
	// deprecatedAzureBrokerImagePath is the missing image deleted from DockerHub, used in kyma-project/addons v0.14.0
	deprecatedAzureBrokerImagePath = "microsoft/azure-service-broker:v1.5.0"
	// externalAzureBrokerImagePath is mirrored microsoft/azure-service-broker:v1.5.0 pushed to kyma-project gcr
	externalAzureBrokerImagePath = "eu.gcr.io/kyma-project/external/azure-service-broker:v1.5.0"
)

type handler struct {
	client  cli.Client
	decoder *admission.Decoder
	log     log.FieldLogger
}

func NewWebhookHandler(k8sCli cli.Client, log log.FieldLogger) *handler {
	return &handler{
		client: k8sCli,
		log:    log,
	}
}

func (h *handler) Handle(ctx context.Context, req admission.Request) admission.Response {
	h.log.Infof("webhook: handling request %q", req.UID)
	pod := &corev1.Pod{}
	if err := MatchKinds(pod, req.Kind); err != nil {
		h.log.Errorf("kind does not match: %s", err)
		return admission.Errored(http.StatusBadRequest, err)
	}

	if err := h.decoder.Decode(req, pod); err != nil {
		h.log.Errorf("cannot decode Pod: %s", err)
		return admission.Errored(http.StatusBadRequest, err)
	}

	h.log.Infof("mutating pod %s", pod.ObjectMeta.Name)
	err := h.mutatePod(pod)
	if err != nil {
		h.log.Errorf("cannot mutate Pod: %s", err)
		return admission.Errored(http.StatusInternalServerError, err)
	}

	rawPod, err := json.Marshal(pod)
	if err != nil {
		h.log.Errorf("cannot marshal mutated pod: %s", err)
		return admission.Errored(http.StatusInternalServerError, err)
	}

	h.log.Infof("finish handling pod: %s", req.UID)
	return admission.PatchResponseFromRaw(req.Object.Raw, rawPod)
}

func (h *handler) mutatePod(pod *corev1.Pod) error {
	for i, ctr := range pod.Spec.Containers {
		h.log.Infof("found container %s using image %q", ctr.Name, ctr.Image)
		if ctr.Image == deprecatedAzureBrokerImagePath {
			h.log.Infof("swapping image %q with %q", ctr.Image, externalAzureBrokerImagePath)
			ctr.Image = externalAzureBrokerImagePath
			pod.Spec.Containers[i] = ctr
		}
	}
	return nil
}

func (h *handler) InjectDecoder(d *admission.Decoder) error {
	h.decoder = d
	return nil
}

func EqualGVK(a metav1.GroupVersionKind, b schema.GroupVersionKind) bool {
	return a.Kind == b.Kind && a.Version == b.Version && a.Group == b.Group
}

// matchKinds returns error if given obj GVK is not equal to the reqKind GVK
func MatchKinds(obj runtime.Object, reqKind metav1.GroupVersionKind) error {
	gvk, err := apiutil.GVKForObject(obj, scheme.Scheme)
	if err != nil {
		return err
	}

	if !EqualGVK(reqKind, gvk) {
		return fmt.Errorf("type mismatch: want: %s got: %s", gvk, reqKind)
	}
	return nil
}
