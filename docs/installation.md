## Installation 

Learn how to install Helm Broker in this document.

### Prerequisites

To run the project, download these tools:

* [kubectl](https://kubernetes.io/docs/tasks/tools/install-kubectl/) 1.16
* [Helm CLI](https://github.com/kubernetes/helm#install) 2.14
* [Docker](https://docs.docker.com/install/) 19.03 (for local installation)
* [Kind](https://github.com/kubernetes-sigs/kind#installation-and-usage) 0.5 (for local installation) 

>**NOTE:** For non-local installation, use Kubernetes v1.15.

### Run on kind

To run the Helm Broker, you need a Kubernetes cluster with Tiller and Service Catalog. Run the `./hack/run-dev-kind.sh` script, or follow these steps to set up the Helm Broker on Kind with all necessary dependencies:

1. Create a local cluster on Kind:
```bash
kind create cluster
``` 

2. Install Tiller into your cluster:
```bash
kubectl create serviceaccount --namespace kube-system tiller
kubectl create clusterrolebinding tiller-cluster-rule --clusterrole=cluster-admin --serviceaccount=kube-system:tiller
helm init --service-account tiller --upgrade --wait
```

3. Install Service Catalog as a Helm chart:
```bash
helm repo add svc-cat https://svc-catalog-charts.storage.googleapis.com
helm install svc-cat/catalog --name catalog --namespace catalog
```

4. Clone the Helm Broker repository:
```bash
git clone git@github.com:kyma-project/helm-broker.git
```

5. Install the Helm Broker chart from the cloned repository:
```bash
helm install charts/helm-broker --name helm-broker --namespace helm-broker
```

### Run the Helm Broker locally

To run the Helm Broker without building a binary file, follow these steps:

1. Start Minikube:
```bash
minikube start
```

2. Create necessary CRDs:
```bash
kubectl apply -f config/crds/
```

3. Start etcd in a Docker container:
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

4. Start the Broker:
```bash
APP_HELM_TILLER_TLS_ENABLED=false \
APP_KUBECONFIG_PATH=/Users/$User/.kube/config \
APP_CONFIG_FILE_NAME=hack/examples/local-etcd-config.yaml \
go run cmd/broker/main.go
```

Now you can test the Broker using the **/v2/catalog** endpoint.

```bash
curl -H "X-Broker-API-Version: 2.13" localhost:8080/cluster/v2/catalog
```

5. Start the Controller:
```bash
APP_KUBECONFIG_PATH=/Users/$User/.kube/config \
APP_DOCUMENTATION_ENABLED=false \
APP_TMP_DIR=/tmp APP_NAMESPACE=default \
APP_SERVICE_NAME=helm-broker \
APP_CONFIG_FILE_NAME=hack/examples/local-etcd-config.yaml \
APP_CLUSTER_SERVICE_BROKER_NAME=helm-broker \
APP_DEVELOP_MODE=true \
go run cmd/controller/main.go -metrics-addr ":8081"
```

>**NOTE:** Not all features are available when you run the Helm Broker locally. All features which perform actions with Tiller do not work. Moreover, the Controller performs operations on ClusterServiceBroker/ServiceBroker resources, which needs the Service Catalog to work properly.

You can run the Controller and the Broker configured with the in-memory storage, but then the Broker cannot read data stored by the Controller. To run the Broker and the Controller without etcd, run these commands:

```bash
APP_HELM_TILLER_TLS_ENABLED=false \
APP_KUBECONFIG_PATH=/Users/$User/.kube/config \
APP_CONFIG_FILE_NAME=hack/examples/minimal-config.yaml \
APP_NAMESPACE=kyma-system go run cmd/broker/main.go
```

```bash
APP_KUBECONFIG_PATH=/Users/$User/.kube/config \
APP_DOCUMENTATION_ENABLED=false \
APP_TMP_DIR=/tmp APP_NAMESPACE=default \
APP_SERVICE_NAME=helm-broker \
APP_CONFIG_FILE_NAME=hack/examples/minimal-config.yaml \
APP_CLUSTER_SERVICE_BROKER_NAME=helm-broker \
APP_DEVELOP_MODE=true \
go run cmd/controller/main.go -metrics-addr ":8081"
```
