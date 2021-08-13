package webhook

import (
	"context"
	"fmt"
	"testing"

	"github.com/sirupsen/logrus"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gomodules.xyz/jsonpatch/v2"
	"k8s.io/api/admission/v1beta1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/kubernetes/scheme"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
	"sigs.k8s.io/controller-runtime/pkg/webhook/admission"
)

func TestHandler_Handle(t *testing.T) {
	// given

	request := admission.Request{
		AdmissionRequest: v1beta1.AdmissionRequest{
			UID:       "1234-abcd",
			Operation: v1beta1.Create,
			Name:      "test-pod",
			Namespace: "namespace",
			Kind: metav1.GroupVersionKind{
				Kind:    "Pod",
				Version: "v1",
				Group:   "",
			},
			Object: runtime.RawExtension{Raw: rawPod()},
		},
	}

	fakeClient := fake.NewFakeClientWithScheme(scheme.Scheme)
	decoder, err := admission.NewDecoder(scheme.Scheme)
	require.NoError(t, err)

	handler := NewWebhookHandler(fakeClient, logrus.New())
	err = handler.InjectDecoder(decoder)
	require.NoError(t, err)

	// when
	response := handler.Handle(context.TODO(), request)

	// then
	assert.True(t, response.Allowed)

	// filtering out status cause k8s api-server will discard this too
	patches := filterOutStatusPatch(response.Patches)
	assert.Len(t, patches, 1)

	for _, patch := range patches {
		assert.Equal(t, "replace", patch.Operation)
		assert.Contains(t, []string{"/spec/containers/1/image"}, patch.Path)
		assert.Equal(t, patch.Value, externalAzureBrokerImagePath)
	}
}

func rawPod() []byte {
	return []byte(fmt.Sprintf(`{
		"apiVersion": "v1",
		"kind": "Pod",
		"metadata": {
		  "creationTimestamp": null,
		  "name": "test-pod",
		  "labels": {
			"%s": "%s"
		  }
		},
		"spec": {
		  "containers": [
			{
			  "name": "test",
			  "image": "test",
			  "resources": {}
            },
			{
			  "name": "open-service-broker-azure",
			  "image": "microsoft/azure-service-broker:v1.5.0",
			  "resources": {}
			}
		  ]
		}
	}`, "chart", "azure-service-broker-0.0.1"))
}

func filterOutStatusPatch(operations []jsonpatch.JsonPatchOperation) []jsonpatch.JsonPatchOperation {
	var filtered []jsonpatch.JsonPatchOperation
	for _, op := range operations {
		if op.Path != "/status" {
			filtered = append(filtered, op)
		}
	}

	return filtered
}
