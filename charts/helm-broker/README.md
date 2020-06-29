# Helm Broker

## Overview

The Helm Broker is a [Service Broker](https://kyma-project.io/docs/master/components/service-catalog/#service-brokers-overview) which exposes Helm charts as ServiceClasses in the [Service Catalog](https://kyma-project.io/docs/master/components/service-catalog/#overview-overview). To do so, the Helm Broker uses the concept of addons. An addon is an abstraction layer over a Helm chart which provides all information required to convert the chart into a ServiceClass.

The Helm Broker implements the [Open Service Broker API](https://github.com/openservicebrokerapi/servicebroker/blob/master/spec.md). For more information about the Helm Broker, read the [documentation](https://kyma-project.io/docs/master/components/helm-broker/).
