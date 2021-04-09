# Helm Broker

[![Go Report Card](https://goreportcard.com/badge/github.com/kyma-project/helm-broker)](https://goreportcard.com/report/github.com/kyma-project/helm-broker)
[![Sourcegraph](https://sourcegraph.com/github.com/kyma-project/helm-broker/-/badge.svg)](https://sourcegraph.com/github.com/kyma-project/helm-broker?badge)

## Overview

Helm Broker is a [Service Broker](https://kyma-project.io/docs/master/components/service-catalog/#overview-service-brokers) that exposes Helm charts as Service Classes in [Service Catalog](https://kyma-project.io/docs/master/components/service-catalog/#overview-service-catalog). To do so, Helm Broker uses the concept of addons. An addon is an abstraction layer over a Helm chart which provides all information required to convert the chart into a Service Class.

Helm Broker fetches default cluster-wide addons defined by the [helm-repos-urls](https://github.com/kyma-project/kyma/blob/master/resources/helm-broker/templates/default-addons-cfg.yaml) custom resource (CR). This CR contains URLs that point to the release of the [`addons`](https://github.com/kyma-project/addons/releases) repository compatible with a given [Kyma release](https://github.com/kyma-project/kyma/releases). You can also configure the Helm Broker to fetch addons definitions from other addons repositories.

You can install Helm Broker either as a standalone project, or as part of [Kyma](https://kyma-project.io/). In Kyma, you can use addons to install the following Service Brokers:

* Azure Service Broker
* AWS Service Broker
* GCP Service Broker

>**NOTE:** Starting from Kyma 2.0, Helm Broker will no longer be supported.

To see all addons that Helm Broker provides, go to the [`addons`](https://github.com/kyma-project/addons) repository.

Helm Broker implements the [Open Service Broker API](https://github.com/openservicebrokerapi/servicebroker/blob/v2.14/profile.md#service-metadata) (OSB API). To be compliant with the Service Catalog version used in Kyma, the Helm Broker supports only the following OSB API versions:
- v2.13
- v2.12
- v2.11

> **NOTE:** The Helm Broker does not implement the OSB API update operation.


## Next steps

To install Helm Broker and develop the project, read [this](./docs/01-installation.md) document. For more details, tutorials, and troubleshooting, explore the [documentation](./docs) directory.
