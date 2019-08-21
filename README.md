# Helm Broker

## Overview

The Helm Broker is a [Service Broker](https://kyma-project.io/docs/master/components/service-catalog/#service-brokers-overview) which exposes Helm charts as Service Classes in the [Service Catalog](https://kyma-project.io/docs/master/components/service-catalog/#overview-overview). To do so, the Helm Broker uses the concept of addons. An addon is an abstraction layer over a Helm chart which provides all information required to convert the chart into a Service Class.

For more information, read the [Helm Broker documentation](https://kyma-project.io/docs/master/components/helm-broker/). If you want to use the Helm Broker with all dependencies, try out [Kyma](https://kyma-project.io/).

## Prerequisites

To run the project, download these tools:

* [kubectl](https://kubernetes.io/docs/tasks/tools/install-kubectl/)
* [Helm CLI](https://github.com/kubernetes/helm#install)
* [Minikube](https://kubernetes.io/docs/tasks/tools/install-minikube/) for local installation

## Installation 

To run the Helm Broker, you need a Kubernetes cluster with Tiller and Service Catalog. Follow these steps to set up the Helm Broker on Minikube with all necessary dependencies:

1. Run the Minikube:
```bash
minikube start
``` 

2. Install Tiller into your cluster:
```bash
helm init
```

3. Install the Service Catalog as a Helm chart:
```bash
helm repo add svc-cat https://svc-catalog-charts.storage.googleapis.com
helm install svc-cat/catalog --name catalog --namespace catalog
```

4. Install the Helm Broker chart:
```bash
helm install charts/helm-broker --name helm-broker --namespace helm-broker
```
## Usage



If you have installed the Helm Broker with the Service Catalog, you can add your addon repositories and provision ServiceInstances. Read [this](https://kyma-project.io/docs/master/components/helm-broker#details-create-addons-repository) document to learn how. You can find more ready-to-use addons [here](https://github.com/kyma-project/addons). Follow this example to configure the Helm Broker and provision the Redis instance:

1. Configure the Helm Broker to use the addons repository that contains the Redis addon:
```bash
kubectl apply -f contrib/sample-addons.yaml
```


After the Helm Broker processes the addons' configuration, you can see the Redis ClusterServiceClass:

```bash
kubectl get clusterserviceclass
```

2. Provision the Redis instance:
```bash
kubectl apply -f contrib/redis-instance.yaml
```

Now you can check the status of the Redis instance:
```bash
kubectl get serviceinstance
```

4. Create a binding for the Redis instance:
```bash
kubectl apply -f contrib/redis-binding.yaml
```

Check the secret containing Redis credentials:
```bash
kubectl get secret redis -o yaml
```

You can see decoded values using the following commands:
```bash
kubectl get secret redis -o=jsonpath="{.data.HOST}" | base64 -D
kubectl get secret redis -o=jsonpath="{.data.PORT}" | base64 -D
kubectl get secret redis -o=jsonpath="{.data.REDIS_PASSWORD}" | base64 -D
```

### Use environment variables

Use the following environment variables to configure the application:

#### Broker

| Name | Required | Default | Description |
|-----|---------|--------|------------|
| **APP_PORT** | No | `8080` | The port on which the HTTP server listens. |
| **APP_KUBECONFIG_PATH** | No |  | The path to the `kubeconfig` file that you need to run an application outside of the cluster. |
| **APP_CONFIG_FILE_NAME** | No | | Specifies the path to a config YAML file.|
| **APP_HELM_TILLER_TLS_ENABLED** | No | true | Specifies the tls configuration with the Tiller. If `true` the TLS communication with Tiller is required. |
| **APP_HELM_TILLER_HOST** | No | | Specifies the host address of the Tiller release server. |
| **APP_HELM_TILLER_INSECURE** | No | false | Controls whether the broker verifies the Tiller's certificate. |
| **APP_HELM_TILLER_KEY** | No | | A path to PEM-encoded private key file. |
| **APP_HELM_TILLER_CRT** | No | | A path to PEM-encoded certificate file. |

#### Controller

| Name | Required | Default | Description |
|-----|---------|--------|------------|
| **APP_CONFIG_FILE_NAME** | No | | Specifies the path to a config YAML file.|
| **APP_TMP_DIR** | Yes | | Path to a temporary directory, used for unpacking addons archives or cloning Git repositories. |
| **APP_KUBECONFIG_PATH** | No |  | The path to the `kubeconfig` file that you need to run an application outside of the cluster. |
| **APP_NAMESPACE** | Yes | | The namespace, where Helm-Broker is installed. |
| **APP_SERVICE_NAME** | Yes | | The name of the Kubernetes Service which is exposing the broker. |
| **APP_CLUSTER_SERVICE_BROKER_NAME** | Yes | | The name of the ClusterServiceBroker resource which registers the Helm-Broker into the Service-Catalog. |
| **APP_DEVELOP_MODE** | No | false | If `true` insecure HTTP based repositories are allowed. |
| **APP_DOCUMENTATION_ENABLED** | No | false | Indicates if Helm-Broker should upload addons documentation to the [Headless CMS](https://kyma-project.io/docs/components/headless-cms/). |

## Development

To set up the project, download these tools:

* [Go](https://golang.org/dl/) 1.11.4
* [Dep](https://github.com/golang/dep) v0.5.0
* [Docker](https://www.docker.com/)

These Go and Dep versions are compliant with the `buildpack` used by Prow. For more details read [this](https://github.com/kyma-project/test-infra/blob/master/prow/images/buildpack-golang/README.md) document.

Before each commit, use the `before-commit.sh` script, which tests your changes and builds binaries. If you want to run Helm-Broker locally, please read [running locally document](/docs/run-local.md)

Integration tests checks if all part of Helm-Broker works together. It starts `kube-apiserver` with `etcd`, setups the controller and the broker. Before you run it, please install [Kubebuilder](https://github.com/kubernetes-sigs/kubebuilder). You can run integration tests using the follwoing command:
```bash
make integration-test
```

If you want to build Docker images with your changes and push them to a registry, follow these steps:
1. Run tests and build binaries:
```bash
make build
```

2. Build Docker images:
```bash
make build-image
```

3. Configure environent variables pointing to your registry, for example:
```bash
export DOCKER_PUSH_REPOSITORY=eu.gcr.io/
export DOCKER_PUSH_DIRECTORY=your-project
export DOCKER_TAG=latest
```

4. Push to the registry:
```bash
make push-image
```

Now you can install Helm-Broker with you custom image using the following command:
```bash
helm install charts/helm-broker \
 --name helm-broker \
 --namespace helm-broker \
 --set global.helm_broker.image="${DOCKER_PUSH_REPOSITORY}${DOCKER_PUSH_DIRECTORY}/helm-broker" \
 --set global.helm_broker.version=${DOCKER_TAG} \
 --set global.helm_controller.image="${DOCKER_PUSH_REPOSITORY}${DOCKER_PUSH_DIRECTORY}/helm-controller" \
 --set global.helm_controller.version=${DOCKER_TAG}
```

If you have installed Helm-Broker already, you can upgrade it to use new images:
```bash
helm upgrade helm-broker charts/helm-broker \
 --set global.helm_broker.image="${DOCKER_PUSH_REPOSITORY}${DOCKER_PUSH_DIRECTORY}/helm-broker" \
 --set global.helm_broker.version=${DOCKER_TAG} \
 --set global.helm_controller.image="${DOCKER_PUSH_REPOSITORY}${DOCKER_PUSH_DIRECTORY}/helm-controller" \
 --set global.helm_controller.version=${DOCKER_TAG}
```
