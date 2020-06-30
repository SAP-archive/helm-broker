## Usage

To install the Helm Broker on your cluster, run this command:

```
helm install https://github.com/kyma-project/helm-broker/releases/download/v1.1.0/helm-broker-chart.tar.gz --wait
```

Provide the latest addons for the Helm Broker by creating the `ClusterAddonsConfiguration` custom resource:

```yaml
apiVersion: addons.kyma-project.io/v1alpha1
kind: ClusterAddonsConfiguration
metadata:
  name: my-addons
spec:
  repositories:
  - url: "https://github.com/kyma-project/addons/releases/download/latest/index.yaml"
```

To learn more about the Helm Broker, read the [documentation](https://github.com/kyma-project/helm-broker/blob/v1.1.0/docs/README.md).
