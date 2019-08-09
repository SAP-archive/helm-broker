# Helm Broker

## Overview

The Helm Broker is a [Service Broker](https://kyma-project.io/docs/master/components/service-catalog/#service-brokers-overview) which exposes Helm charts as Service Classes in the [Service Catalog](https://kyma-project.io/docs/master/components/service-catalog/#overview-overview). To do so, the Helm Broker uses the concept of addons. An addon is an abstraction layer over a Helm chart which provides all information required to convert the chart into a Service Class.

Helm Broker implements the [Service Broker API](https://github.com/openservicebrokerapi/servicebroker/blob/master/spec.md). For more information about the Service Brokers, see the [Service Brokers overview](https://github.com/kyma-project/kyma/blob/master/docs/helm-broker/03-01-create-addons.md) documentation. Find the details about the Helm Broker in the [docs](https://github.com/kyma-project/kyma/tree/master/docs/helm-broker) repository.
