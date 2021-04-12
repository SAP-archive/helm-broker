---
title: Security
---

This document presents the ways to secure Helm Broker on your cluster against possible vulnerabilities.

## Authorize access to AddonsConfigurations

In the [AddonsConfiguration](https://kyma-project.io/docs/components/helm-broker#custom-resource-addons-configuration) custom resource (CR), you can provide URLs to your external addon repositories. If a server delivers too much payload, Helm Broker may crash with the `OOM killed` reason. This may be used by third parties to damage your cluster or to increase costs. To mitigate this issue, authorize access to the AddonsConfiguration CR. Read about [how to grant roles and permissions in Kyma](https://kyma-project.io/docs/components/security/#details-authorization-in-kyma) to learn more.

> **NOTE:** The amount of memory and storage size determines the maximum size of your addons repository. These limits are set in the
[Helm Broker chart](https://kyma-project.io/docs/components/helm-broker/#configuration-helm-broker-chart).
