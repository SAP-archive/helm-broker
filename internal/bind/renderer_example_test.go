//go:build integration
// +build integration

package bind_test

import (
	"fmt"
	"io/ioutil"
	"log"
	"os"
	"path/filepath"

	"github.com/kyma-project/helm-broker/internal"
	"github.com/kyma-project/helm-broker/internal/bind"
	yhelm "github.com/kyma-project/helm-broker/internal/helm"
	"github.com/kyma-project/helm-broker/internal/platform/logger/spy"
	"helm.sh/helm/v3/pkg/chart/loader"
	"k8s.io/client-go/tools/clientcmd"
)

// To run it you need to configure KUBECONFIG env
func ExampleNewRenderer() {
	cfg, err := clientcmd.BuildConfigFromFlags("", os.Getenv("KUBECONFIG"))
	fatalOnErr(err)
	const releaseName = "example-renderer-test"
	bindTmplRenderer := bind.NewRenderer()

	// loadChart
	ch, err := loader.LoadDir("testdata/repository/redis-0.0.3/chart/redis")
	fatalOnErr(err)

	// load bind template for above chart
	b, err := ioutil.ReadFile(filepath.Join("testdata/repository", "redis-0.0.3/plans/micro/bind.yaml"))
	fatalOnErr(err)

	hClient, err := yhelm.NewClient(cfg, spy.NewLogDummy())
	fatalOnErr(err)

	// install chart in same way as we are doing in our business logic
	resp, err := hClient.Install(ch, internal.ChartValues{}, releaseName, "ns-name")

	// clean-up, even if install error occurred
	defer hClient.Delete(releaseName, "ns-name")
	fatalOnErr(err)

	rendered, err := bindTmplRenderer.Render(internal.AddonPlanBindTemplate(b), &internal.Instance{
		Namespace:   "ns-name",
		ReleaseName: internal.ReleaseName(resp.Name),
	}, ch)
	fatalOnErr(err)

	fmt.Println(string(rendered))

	// Output:
	// credential:
	//   - name: HOST
	//     value: example-renderer-test-redis.ns-name.svc.cluster.local
	//   - name: PORT
	//     valueFrom:
	//       serviceRef:
	//         name: example-renderer-test-redis
	//         jsonpath: '{ .spec.ports[?(@.name=="redis")].port }'
	//   - name: REDIS_PASSWORD
	//     valueFrom:
	//       secretKeyRef:
	//         name: example-renderer-test-redis
	//         key: redis-password
}

func fatalOnErr(err error) {
	if err != nil {
		log.Fatal(err)
	}
}
