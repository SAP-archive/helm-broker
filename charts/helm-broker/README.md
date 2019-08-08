# Helm Broker

## Overview

Helm Broker provides addons in the [Service Catalog](https://github.com/kubernetes-sigs/service-catalog). An addon is an abstraction layer over a Helm chart which enables you to provide more information about the Helm chart. For example, an addon can provide plan definitions or binding details. Service Catalog requires this information. Addons are services available in Service Catalog.

Helm Broker implements the [Service Broker API](https://github.com/openservicebrokerapi/servicebroker/blob/master/spec.md). For more information about the Service Brokers, see the [Service Brokers overview](https://github.com/kyma-project/kyma/blob/master/docs/helm-broker/03-01-create-addons.md) documentation. Find the details about the Helm Broker in the [docs](https://github.com/kyma-project/kyma/tree/master/docs/helm-broker) repository.