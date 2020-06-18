package helm_test

import (
	"context"
	"testing"

	"github.com/kyma-project/helm-broker/internal/helm"
	"github.com/sirupsen/logrus"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"helm.sh/helm/v3/pkg/chart/loader"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"sigs.k8s.io/controller-runtime/pkg/envtest"
)

func TestInstallListDelete(t *testing.T) {
	// given
	environment := &envtest.Environment{}
	restConfig, err := environment.Start()
	require.NoError(t, err)

	svc, err := helm.NewClient(restConfig, "secrets", logrus.New())
	require.NoError(t, err)

	chrt, err := loader.LoadDir("example/testing")
	require.NoError(t, err)
	k8sCS, err := kubernetes.NewForConfig(restConfig)
	require.NoError(t, err)

	// when
	_, err = svc.Install(chrt, map[string]interface{}{
		"planName":       "micro",
		"additionalData": "abc",
	}, "nice-alpaca", "playground")
	require.NoError(t, err)

	// then
	gotCM, err := k8sCS.CoreV1().ConfigMaps("playground").Get(context.TODO(), "nice-alpaca-testing", v1.GetOptions{})
	require.NoError(t, err)
	assert.Equal(t, "micro", gotCM.Data["planName"])
	assert.Equal(t, "abc", gotCM.Data["additionalData"])

	// check that the release exists
	// when
	rels, err := svc.ListReleases("playground")
	require.NoError(t, err)

	// then
	assert.Len(t, rels, 1)
	assert.Equal(t, "nice-alpaca", rels[0].Name)

	// delete
	err = svc.Delete("nice-alpaca", "playground")
	require.NoError(t, err)
	rels, err = svc.ListReleases("playground")
	require.NoError(t, err)
	assert.Len(t, rels, 0)

}
