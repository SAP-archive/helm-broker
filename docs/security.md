# Security

Read this document to learn how to secure the Helm Broker on your cluster.

## Vulnerabilities

The Helm Broker with the current architecture have vulnerabilities. Read below points to know how to protect yourself.

### Addons Configurations

In the AddonsConfiguration CR, you can provide the URLs to addon repositories. If the server will provide too much payload the Helm Broker can crash with the `OOM killed` reason.
That's a weak part of the Helm Broker which can be used by the third persons to damage the cluster or to increase the costs.
To mitigate that issue, you can authorize access to the AddonsConfigurations CR. Read this [document](https://github.com/kyma-project/kyma/blob/master/docs/security/03-05-roles-in-kyma.md) to see how it's done in Kyma. 


> **NOTE:** The amount of memory and storage size determine the maximum size of your addons repository. These limits are set in the
[Helm Broker chart](https://kyma-project.io/docs/components/helm-broker/#configuration-helm-broker-chart).