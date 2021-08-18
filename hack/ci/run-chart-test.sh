#!/usr/bin/env bash

set -o errexit
set -o nounset
set -o pipefail

readonly TMP_DIR=$(mktemp -d)

readonly SC_RELEASE_NAMESPACE="catalog"
readonly SC_RELEASE_NAME="catalog"

readonly CURRENT_DIR=$( cd "$( dirname "${BASH_SOURCE[0]}" )" && pwd )
readonly LOCAL_REPO_ROOT_DIR=$( cd "${CURRENT_DIR}/../../" && pwd )
readonly CONTAINER_REPO_ROOT_DIR="/workdir"

source "${CURRENT_DIR}/lib/utilities.sh" || { echo 'Cannot load CI utilities.'; exit 1; }
source "${CURRENT_DIR}/lib/deps_ver.sh" || { echo 'Cannot load dependencies versions.'; exit 1; }

cleanup() {
    shout '- Removing ct container...'
    docker kill ct > /dev/null 2>&1
    kind::delete_cluster || true

    rm -rf "${TMP_DIR}" > /dev/null 2>&1 || true
    shout 'Cleanup Done!'
}

run_ct_container() {
    shout '- Running ct container...'
    docker run --rm --interactive --detach --network host --name ct \
        --volume "$LOCAL_REPO_ROOT_DIR":"$CONTAINER_REPO_ROOT_DIR" \
        --workdir "$CONTAINER_REPO_ROOT_DIR" \
        "quay.io/helmpack/chart-testing:$CT_VERSION" \
        cat
}

docker_ct_exec() {
    docker exec --interactive ct "$@"
}

chart::lint() {
    shout '- Linting Helm Broker chart...'
    docker_ct_exec ct lint --charts ${CONTAINER_REPO_ROOT_DIR}/charts/helm-broker/
}

chart::install_and_test() {
    pushd "${LOCAL_REPO_ROOT_DIR}"
    shout "- Building Helm Broker images from sources..."
    make build-binaries
    make build-image

    shout "- Loading Helm Broker images into kind cluster..."
    kind::load_image helm-broker-tests:latest
    kind::load_image helm-controller:latest
    kind::load_image helm-broker:latest
    kind::load_image helm-broker-webhook:latest

    shout '- Installing and testing Helm Broker chart...'
    docker_ct_exec ct install --charts ${CONTAINER_REPO_ROOT_DIR}/charts/helm-broker/

    popd
}

chart::setup() {
    # This is required because chart-testing tool expects that origin will be set
    # but when prow checkouts repository then remote info is empty, so we need to do that by our own
    docker_ct_exec git remote add origin https://github.com/kyma-project/helm-broker.git
}

setup_kubectl_in_ct_container() {
    docker_ct_exec mkdir -p /root/.kube

    shout '- Copying KUBECONFIG to container...'
    docker cp "$KUBECONFIG" ct:/root/.kube/config

    shout '- Checking connection to cluster...'
    docker_ct_exec kubectl cluster-info
}

# Installs service catalog on cluster.
install::service_catalog() {
  shout "- Provisioning Service Catalog chart in ${SC_RELEASE_NAMESPACE} namespace..."

  docker_ct_exec helm repo add svc-cat https://kubernetes-sigs.github.io/service-catalog
  docker_ct_exec kubectl create ns "${SC_RELEASE_NAMESPACE}"
  docker_ct_exec helm install "${SC_RELEASE_NAME}" svc-cat/catalog  --namespace "${SC_RELEASE_NAMESPACE}" --wait
}

install_local-path-provisioner() {
    # kind doesn't support Dynamic PVC provisioning yet https://github.com/kubernetes-sigs/kind/issues/118,
    # this is one ways to get it working
    # https://github.com/rancher/local-path-provisioner


    # Remove default storage class. It will be recreated by local-path-provisioner
    docker_ct_exec kubectl delete storageclass standard

    shout '- Installing local-path-provisioner...'
    docker_ct_exec kubectl apply -f https://raw.githubusercontent.com/rancher/local-path-provisioner/master/deploy/local-path-storage.yaml

    shout '- Setting local-path-provisioner as default class...'
    kubectl patch storageclass local-path -p '{"metadata": {"annotations":{"storageclass.kubernetes.io/is-default-class":"true"}}}'

}

main() {
    if [[ "${RUN_ON_PROW-no}" = "true" ]]; then
        # This is a workaround for our CI. More info you can find in this issue:
        # https://github.com/kyma-project/test-infra/issues/1499
        ensure_docker
    fi

    run_ct_container
    trap cleanup EXIT
    if [[ "${RUN_ON_PROW-no}" = "true" ]]; then
        chart::setup
    fi

    export INSTALL_DIR=${TMP_DIR} KIND_VERSION=${STABLE_KIND_VERSION} HELM_VERSION=${STABLE_HELM_VERSION}
    install::kind

    export KUBERNETES_VERSION=${STABLE_KUBERNETES_VERSION}
    kind::create_cluster
    setup_kubectl_in_ct_container
    install_local-path-provisioner
    install::service_catalog

    chart::lint
    chart::install_and_test
}

main
