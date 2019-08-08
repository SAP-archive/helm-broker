## Run Helm-Broker local version

To run the application without building a binary file, follow these steps:

Start minikube:
```bash
minikube start
```

Create necessary CRDs:
```bash
kubectl apply -f config/crds/
```

Start Etcd in a Docker container:
```bash
docker run \
  -p 2379:2379 \
  -p 2380:2380 \
  -d \
  quay.io/coreos/etcd:v3.3 \
  /usr/local/bin/etcd \
  --data-dir /etcd-data \
  --listen-client-urls http://0.0.0.0:2379 \
  --advertise-client-urls http://0.0.0.0:2379 \
  --listen-peer-urls http://0.0.0.0:2380 \
  --initial-advertise-peer-urls http://0.0.0.0:2380
```

start the broker:
```bash
APP_HELM_TILLER_TLS_ENABLED=false \
APP_KUBECONFIG_PATH=/Users/$User/.kube/config \
APP_CONFIG_FILE_NAME=contrib/local-etcd-config.yaml \
go run cmd/broker/main.go
```

Now you can test the broker using the **/v2/catalog** endpoint

```bash
curl -H "X-Broker-API-Version: 2.13" localhost:8080/cluster/v2/catalog
```

Start the controller:
```bash
APP_KUBECONFIG_PATH=/Users/$User/.kube/config \
APP_DOCUMENTATION_ENABLED=false \
APP_TMP_DIR=/tmp APP_NAMESPACE=default \
APP_SERVICE_NAME=helm-broker \
APP_CONFIG_FILE_NAME=contrib/local-etcd-config.yaml \
APP_CLUSTER_SERVICE_BROKER_NAME=helm-broker \
APP_DEVELOP_MODE=true \
go run cmd/controller/main.go -metrics-addr ":8081"
```

>**NOTE:**  Not all features are available when you run the Helm Broker locally. All features which perform actions with Tiller do not work. Additionally, the controller performs operation on ClusterServiceBroker/ServiceBroker resources, which are part of Service Catalog.

You can run the controller and the broker configured with in-memory storage (without etcd), but then the broker cannot read data stored by the controller:

```bash
APP_HELM_TILLER_TLS_ENABLED=false \
APP_KUBECONFIG_PATH=/Users/$User/.kube/config \
APP_CONFIG_FILE_NAME=contrib/minimal-config.yaml \
APP_NAMESPACE=kyma-system go run cmd/broker/main.go
```

```bash
APP_KUBECONFIG_PATH=/Users/$User/.kube/config \
APP_DOCUMENTATION_ENABLED=false \
APP_TMP_DIR=/tmp APP_NAMESPACE=default \
APP_SERVICE_NAME=helm-broker \
APP_CONFIG_FILE_NAME=contrib/minimal-config.yaml \
APP_CLUSTER_SERVICE_BROKER_NAME=helm-broker \
APP_DEVELOP_MODE=true \
go run cmd/controller/main.go -metrics-addr ":8081"
```