---
title: Overview
---

The Helm Broker is a [Service Broker](/components/service-catalog/#overview-service-brokers) which exposes Helm charts as Service Classes in the [Service Catalog](/components/service-catalog/). To do so, the Helm Broker uses the concept of addons. An addon is an abstraction layer over a Helm chart which provides all information required to convert the chart into a Service Class.

The Helm Broker fetches addons which contain a set of specific [files](#details-create-addons). You must place your addons in a repository of an appropriate [format](#details-create-addons-repository). The Helm Broker fetches default cluster-wide addons defined by the [helm-repos-urls](https://github.com/kyma-project/kyma/blob/master/resources/helm-broker/templates/default-addons-cfg.yaml) custom resource (CR). This CR contains URLs that point to the release of  [`addons`](https://github.com/kyma-project/addons/releases) repository compatible with a given [Kyma release](https://github.com/kyma-project/kyma/releases). You can also configure the Helm Broker to fetch addons definitions from other addons repositories.
