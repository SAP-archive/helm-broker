module github.com/kyma-project/helm-broker

go 1.13

require (
	github.com/Masterminds/semver v1.5.0
	github.com/Masterminds/sprig v2.22.0+incompatible // indirect
	github.com/alecthomas/jsonschema v0.0.0-20200123075451-43663a393755
	github.com/asaskevich/govalidator v0.0.0-20200428143746-21a406dcc535
	github.com/dsnet/compress v0.0.1 // indirect
	github.com/fatih/structs v1.1.0
	github.com/frankban/quicktest v1.10.0 // indirect
	github.com/ghodss/yaml v1.0.0
	github.com/go-logr/logr v0.2.0
	github.com/golang/protobuf v1.4.2
	github.com/golang/snappy v0.0.1 // indirect
	github.com/gorilla/mux v1.7.4
	github.com/hashicorp/go-cleanhttp v0.5.1
	github.com/hashicorp/go-getter v1.4.1
	github.com/hashicorp/go-multierror v1.1.0
	github.com/imdario/mergo v0.3.9
	github.com/kubernetes-sigs/go-open-service-broker-client v0.0.0-20190909175253-906fa5f9c249
	github.com/kubernetes-sigs/service-catalog v0.3.0
	github.com/kyma-project/kyma v0.5.1-0.20200317154738-0bb20217c2cb
	github.com/kyma-project/rafter v0.0.0-20200413150919-1a89277ac3d8
	github.com/lithammer/dedent v1.1.0
	github.com/mcuadros/go-defaults v1.2.0
	github.com/meatballhat/negroni-logrus v1.1.0
	github.com/mholt/archiver v3.1.1+incompatible
	github.com/minio/minio-go/v6 v6.0.56
	github.com/nwaples/rardecode v1.1.0 // indirect
	github.com/oklog/ulid v1.3.1
	github.com/pborman/uuid v1.2.0
	github.com/pierrec/lz4 v2.5.2+incompatible // indirect
	github.com/pkg/errors v0.9.1
	github.com/prometheus/client_golang v1.7.1
	github.com/sirupsen/logrus v1.7.0
	github.com/stretchr/testify v1.6.1
	github.com/urfave/negroni v1.0.0
	github.com/vrischmann/envconfig v1.2.0
	github.com/xi2/xz v0.0.0-20171230120015-48954b6210f8 // indirect
	go.etcd.io/etcd v0.5.0-alpha.5.0.20200819165624-17cef6e3e9d5
	gopkg.in/yaml.v2 v2.3.0
	helm.sh/helm/v3 v3.4.1
	k8s.io/api v0.19.3
	k8s.io/apimachinery v0.19.3
	k8s.io/cli-runtime v0.19.3
	k8s.io/client-go v0.19.3
	k8s.io/helm v2.16.11+incompatible
	rsc.io/letsencrypt v0.0.3 // indirect
	sigs.k8s.io/controller-runtime v0.6.4
	sigs.k8s.io/yaml v1.2.0
)

replace (
	github.com/go-logr/logr => github.com/go-logr/logr v0.3.0
	gopkg.in/yaml.v2 => gopkg.in/yaml.v2 v2.2.8
	sigs.k8s.io/structured-merge-diff/v3 => sigs.k8s.io/structured-merge-diff/v3 v3.0.0
)
