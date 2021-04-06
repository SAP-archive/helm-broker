## Usage

To install Helm Broker on your cluster, run this command:

```
helm install https://github.com/kyma-project/helm-broker/releases/download/__RELEASE_VERSION__/helm-broker-chart.tar.gz --wait
```

Provide the latest addons for Helm Broker by creating the `ClusterAddonsConfiguration` custom resource:

```yaml
apiVersion: addons.kyma-project.io/v1alpha1
kind: ClusterAddonsConfiguration
metadata:
  name: my-addons
spec:
  repositories:
  - url: "https://github.com/kyma-project/addons/releases/download/latest/index.yaml"
```

To learn more about Helm Broker, read the [documentation](https://github.com/kyma-project/helm-broker/blob/__RELEASE_VERSION__/docs/README.md).
