# Helm Broker

## Overview

Helm Broker is a [Service Broker](https://kyma-project.io/docs/components/service-catalog/#service-brokers-overview) which exposes Helm charts as ServiceClasses in [Service Catalog](https://kyma-project.io/docs/components/service-catalog/#overview-overview). To do so, Helm Broker uses the concept of addons. An addon is an abstraction layer over a Helm chart which provides all information required to convert the chart into a ServiceClass.

Helm Broker implements the [Open Service Broker API](https://github.com/openservicebrokerapi/servicebroker/blob/master/spec.md). For more information about the Helm Broker, read the [documentation](https://kyma-project.io/docs/components/helm-broker/).
