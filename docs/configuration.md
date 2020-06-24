# Configuration

Use the following environment variables to configure the `Broker` container of the Helm Broker:

| Name | Required | Default | Description |
|-----|:---------:|--------|------------|
| **APP_PORT** | No | `8080` | The port on which the HTTP server listens. |
| **APP_KUBECONFIG_PATH** | No |  | Provides the path to the `kubeconfig` file that you need to run an application outside of the cluster. |
| **APP_CONFIG_FILE_NAME** | No | | Specifies the path to the configuration `.yaml` file. |

Use the following environment variables to configure the `Controller` container of the Helm Broker:

| Name | Required | Default | Description |
|-----|:---------:|--------|------------|
| **APP_CONFIG_FILE_NAME** | No | | Specifies the path to the configuration `.yaml` file.  |
| **APP_TMP_DIR** | Yes | | Provides a path to a temporary directory that is used to unpack addons archives or to clone Git repositories. |
| **APP_KUBECONFIG_PATH** | No |  | Provides the path to the `kubeconfig` file that you need to run an application outside of the cluster. |
| **APP_NAMESPACE** | Yes | | Specifies the Namespace where the Helm Broker is installed. |
| **APP_SERVICE_NAME** | Yes | | Specifies the name of the Kubernetes service that exposes the Broker. |
| **APP_CLUSTER_SERVICE_BROKER_NAME** | Yes | | Specifies the name of the ClusterServiceBroker resource which registers the Helm Broker in the Service Catalog. |
| **APP_DEVELOP_MODE** | No | `false` | If set to `true`, you can use unsecured HTTP-based repositories URLs. |
| **APP_DOCUMENTATION_ENABLED** | No | `false` | If set to `true`, the Helm Broker uploads addons documentation to the [Rafter](https://kyma-project.io/docs/components/headless-cms/). |
