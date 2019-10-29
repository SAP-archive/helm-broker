## Usage

Learn more about Helm Broker usage in this document.

### Use environment variables

Use the following environment variables to configure the `Broker` container of the Helm Broker:

| Name | Required | Default | Description |
|-----|:---------:|--------|------------|
| **APP_PORT** | No | `8080` | The port on which the HTTP server listens. |
| **APP_KUBECONFIG_PATH** | No |  | Provides the path to the `kubeconfig` file that you need to run an application outside of the cluster. |
| **APP_CONFIG_FILE_NAME** | No | | Specifies the path to the configuration `.yaml` file. |
| **APP_HELM_TILLER_TLS_ENABLED** | No | `true` | Specifies the TLS configuration for the Tiller. If set to `true`, the TLS communication with Tiller is required. |
| **APP_HELM_TILLER_HOST** | No | | Specifies the host address of the Tiller release server. |
| **APP_HELM_TILLER_INSECURE** | No | `false` | If set to `true`, the Broker verifies the Tiller's certificate. |
| **APP_HELM_TILLER_KEY** | No | | Provides the path to the PEM-encoded private key file. |
| **APP_HELM_TILLER_CRT** | No | | Provides the path to the PEM-encoded certificate file. |

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
| **APP_DOCUMENTATION_ENABLED** | No | `false` | If set to `true`, the Helm Broker uploads addons documentation to the [Headless CMS](https://kyma-project.io/docs/components/headless-cms/). |

### Example

If you have installed the Helm Broker with the Service Catalog, you can add your addon repositories and provision ServiceInstances. Read [this](https://kyma-project.io/docs/master/components/helm-broker#details-create-addons-repository) document to learn how. You can find more ready-to-use addons [here](https://github.com/kyma-project/addons). Follow this example to configure the Helm Broker and provision the Redis instance:

1. Configure the Helm Broker to use the addons repository that contains the Redis addon:
```bash
kubectl apply -f hack/examples/sample-addons.yaml
```

After the Helm Broker processes the addons' configuration, you can see the Redis ClusterServiceClass:

```bash
kubectl get clusterserviceclass
```

2. Provision the Redis instance:
```bash
kubectl apply -f hack/examples/redis-instance.yaml
```

3. Check the status of the Redis instance:
```bash
kubectl get serviceinstance
```

4. Create a binding for the Redis instance:
```bash
kubectl apply -f hack/examples/redis-binding.yaml
```

5. Check the Secret that contains Redis credentials:
```bash
kubectl get secret redis -o yaml
```

Use the following commands to see the decoded values:
```bash
kubectl get secret redis -o=jsonpath="{.data.HOST}" | base64 -D
kubectl get secret redis -o=jsonpath="{.data.PORT}" | base64 -D
kubectl get secret redis -o=jsonpath="{.data.REDIS_PASSWORD}" | base64 -D
```
