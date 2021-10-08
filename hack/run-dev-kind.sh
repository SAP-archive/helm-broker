#!/usr/bin/env bash

set -o errexit

readonly CURRENT_DIR=$( cd "$( dirname "${BASH_SOURCE[0]}" )" && pwd )
readonly REPO_ROOT_DIR=$( cd ${CURRENT_DIR}/../ && pwd )

readonly SC_RELEASE_NAMESPACE="catalog"
readonly SC_RELEASE_NAME="catalog"

readonly HB_NAMESPACE="helm-broker"
readonly HB_CHART_NAME="helm-broker"

readonly HELM_BINARY=helm

source "${CURRENT_DIR}/ci/lib/utilities.sh" || { echo 'Cannot load CI utilities.'; exit 1; }
source "${CURRENT_DIR}/ci/lib/deps_ver.sh" || { echo 'Cannot load dependencies versions.'; exit 1; }

print_warning() {
  echo -e "\033[33m $1 \033[39m"
}

print_error() {
  echo -e "\033[31m $1 \033[39m"
}

print_done() {
  echo -e "\033[32m $1 \033[39m"
}

kind::check_kind() {
  if ! which kind >/dev/null; then
    print_error "Kind is not installed on your host, install it and try again"
    exit 1
  fi

  local version=$(kind version)
  if [[ "${version}" != "${STABLE_KIND_VERSION}" ]]; then
    print_warning "Your kind is in ${version}. Your version is not equal than the supported version of kind ${STABLE_KIND_VERSION}"
  fi
}

helm::check_helm() {
  if ! which helm >/dev/null; then
    print_error "Helm is not installed on your host, install it and try again"
    exit 1
  fi
}

install::helm_broker() {
  shout '- Provisioning Helm Broker chart...'

  ${HELM_BINARY} install ${HB_CHART_NAME} ${REPO_ROOT_DIR}/charts/helm-broker --namespace ${HB_NAMESPACE} --create-namespace
}

# Installs service catalog on cluster.
install::service_catalog() {
  shout "- Provisioning Service Catalog chart in ${SC_RELEASE_NAMESPACE} namespace..."

  ${HELM_BINARY} repo add svc-cat https://kubernetes-sigs.github.io/service-catalog
  ${HELM_BINARY} install "${SC_RELEASE_NAME}" svc-cat/catalog --namespace "${SC_RELEASE_NAMESPACE}" --wait --create-namespace
}

main() {
  # check if docker is running; docker ps -q should only work if the daemon is ready
  docker ps -q > /dev/null

  # check if kind and helm exist and have proper supported version if required
  kind::check_kind
  helm::check_helm

  export KUBERNETES_VERSION=${STABLE_KUBERNETES_VERSION}
  kind::create_cluster

  install::service_catalog
  install::helm_broker

  print_done "Cluster creation complete. You can now use the cluster with:"
  print_done "export KUBECONFIG=\"\$(kind get kubeconfig-path --name=\"${KIND_CLUSTER_NAME}\")\""
}

main
