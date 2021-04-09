---
title: Helm Broker configuration
type: Configuration
---

To configure the Helm Broker chart, override the default values of its `values.yaml` file. This document describes parameters that you can configure.

>**TIP:** To learn more about how to use overrides in Kyma, see the following documents:
>* [Helm overrides for Kyma installation](https://kyma-project.io/docs/root/kyma#configuration-helm-overrides-for-kyma-installation)
>* [Top-level charts overrides](https://kyma-project.io/docs/root/kyma/#configuration-helm-overrides-for-kyma-installation-top-level-charts-overrides)

## Helm Broker chart

This table lists the configurable parameters, their descriptions, and default values for the Helm Broker chart:

| Parameter | Description | Default value |
|-----------|-------------|---------------|
| **ctrl.resources.limits.cpu** | Defines limits for CPU resources. | `100m` |
| **ctrl.resources.limits.memory** | Defines limits for memory resources. During the clone action, the Git binary loads the whole repository into memory. You may need to adjust this value if you want to clone a bigger repository.| `76Mi` |
| **ctrl.resources.requests.cpu** | Defines requests for CPU resources. | `80m` |
| **ctrl.resources.requests.memory** | Defines requests for memory resources. | `32Mi` |
| **ctrl.tmpDirSizeLimit** | Specifies a size limit on the `tmp` directory in the Helm Pod. This directory is used to store processed addons. Eviction manager monitors the disk space used by the Pod and evicts it when the usage exceeds the limit. Then, the Pod is marked as `Evicted`. The limit is enforced with a time delay, usually about 10s. | `1Gi` |
| **global.cfgReposUrlName** | Specifies the name of the default ConfigMap which provides the URLs of addons repositories. | `helm-repos-urls` |
| **global.isDevelopMode** | Defines whether to accept URL prefixes from the **global.urlRepoPrefixes.additionalDevelopMode** list. If set to `true`, Helm Broker accepts the prefixes from the list. | `false` |
| **global.urlRepoPrefixes.default** | Defines a list of accepted prefixes for repository URLs. | `'https://', 'git::', 'github.com/', 'bitbucket.org/'` |
| **global.urlRepoPrefixes.additionalDevelopMode** | Defines a list of accepted prefixes for repository URLs when develop mode is enabled. | `'http://'` |
| **additionalAddonsRepositories.myRepo** | Provides a map of additional ClusterAddonsConfiguration repositories to create by default. The key is used as a name and the value is used as a URL for the repository. | `github.com/myOrg/myRepo//addons/index.yaml` |

## Etcd-stateful sub-chart

This table lists the configurable parameters, their descriptions, and default values:

| Parameter | Description | Default value |
|-----------|-------------|---------------|
| **etcd.resources.limits.cpu** | Defines limits for CPU resources. | `200m` |
| **etcd.resources.limits.memory** | Defines limits for memory resources. | `256Mi` |
| **etcd.resources.requests.cpu** | Defines requests for CPU resources. | `50m` |
| **etcd.resources.requests.memory** | Defines requests for memory resources. | `64Mi` |
| **replicaCount** | Defines the size of the etcd cluster. | `1` |

## Broker container

Use the following environment variables to configure the `Broker` container of Helm Broker:

| Name | Required | Default | Description |
|-----|:---------:|--------|------------|
| **APP_PORT** | No | `8080` | The port on which the HTTP server listens. |
| **APP_KUBECONFIG_PATH** | No |  | Provides the path to the `kubeconfig` file that you need to run an application outside of the cluster. |
| **APP_CONFIG_FILE_NAME** | No | | Specifies the path to the configuration `.yaml` file. |
| **APP_HELM_DRIVER** | Yes| `secrets` | Specifies how Helm releases are stored. The possible values are `secrets` and `configmaps`. |

## Controller container

Use the following environment variables to configure the `Controller` container of Helm Broker:

| Name | Required | Default | Description |
|-----|:---------:|--------|------------|
| **APP_CONFIG_FILE_NAME** | No | | Specifies the path to the configuration `.yaml` file.  |
| **APP_TMP_DIR** | Yes | | Provides a path to a temporary directory that is used to unpack addons archives or to clone Git repositories. |
| **APP_KUBECONFIG_PATH** | No |  | Provides the path to the `kubeconfig` file that you need to run an application outside of the cluster. |
| **APP_NAMESPACE** | Yes | | Specifies the Namespace where Helm Broker is installed. |
| **APP_SERVICE_NAME** | Yes | | Specifies the name of the Kubernetes service that exposes the Broker. |
| **APP_CLUSTER_SERVICE_BROKER_NAME** | Yes | | Specifies the name of the ClusterServiceBroker resource which registers Helm Broker in Service Catalog. |
| **APP_DEVELOP_MODE** | No | `false` | If set to `true`, you can use unsecured HTTP-based repositories URLs. |
| **APP_DOCUMENTATION_ENABLED** | No | `false` | If set to `true`, Helm Broker uploads addons documentation to [Rafter](https://kyma-project.io/docs/components/rafter/). |
| **APP_REPROCESS_ON_ERROR_DURATION** | No | `5m` | Specifies the time after which Helm Broker performs the repository connection retry that has previously failed. |
