# Development

Read this document to learn how to develop the project.

## Prerequisites

* [Go](https://golang.org/dl/) 1.12
* [Dep](https://github.com/golang/dep) 0.5
* [Docker](https://www.docker.com/)

>**NOTE:** The versions of Go and Dep are compliant with the `buildpack` used by Prow. For more details, read [this](https://github.com/kyma-project/test-infra/blob/master/prow/images/buildpack-golang/README.md) document.

## Run tests

Before each commit, use the `before-commit.sh` script. The script runs unit tests that check your changes and build binaries. If you want to run the Helm Broker locally, read [this](/docs/installation.md) document.

You can also run integration tests that check if all parts of the Helm Broker work together. 
These are the prerequisites for integration tests:

- [Kubebuilder](https://github.com/kubernetes-sigs/kubebuilder) 1.0.8
- [Etcd](https://github.com/etcd-io/etcd#etcd) 3.4
- [Minio](https://min.io/download) RELEASE.2019-10-12T01-39-57Z

Run integration tests using this command:

```bash
make integration-test
```

## Update chart's images tag

To change the chart's tags version, run this command:

```bash
make VERSION=v0.0.1 DIR=/pr tag-chart-images
```

This command overrides the images tag in the `charts/helm-broker/values.yaml` file to:

```
eu.gcr.io/kyma-project/helm-broker/pr:v0.0.1
```

## Build Docker images

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

4. Push the image to the registry:
```bash
make push-image
```

5. Install the Helm Broker with your custom image using the following command:
```bash
helm install charts/helm-broker \
 --name helm-broker \
 --namespace helm-broker \
 --set global.helm_broker.image="${DOCKER_PUSH_REPOSITORY}${DOCKER_PUSH_DIRECTORY}/helm-broker" \
 --set global.helm_broker.version=${DOCKER_TAG} \
 --set global.helm_controller.image="${DOCKER_PUSH_REPOSITORY}${DOCKER_PUSH_DIRECTORY}/helm-controller" \
 --set global.helm_controller.version=${DOCKER_TAG}
```

If you already have the Helm Broker installed, you can upgrade it to use new images:
```bash
helm upgrade helm-broker charts/helm-broker \
 --set global.helm_broker.image="${DOCKER_PUSH_REPOSITORY}${DOCKER_PUSH_DIRECTORY}/helm-broker" \
 --set global.helm_broker.version=${DOCKER_TAG} \
 --set global.helm_controller.image="${DOCKER_PUSH_REPOSITORY}${DOCKER_PUSH_DIRECTORY}/helm-controller" \
 --set global.helm_controller.version=${DOCKER_TAG}
```
