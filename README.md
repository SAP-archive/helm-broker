# Helm Broker

[![Go Report Card](https://goreportcard.com/badge/github.com/kyma-project/helm-broker)](https://goreportcard.com/report/github.com/kyma-project/helm-broker)
[![Sourcegraph](https://sourcegraph.com/github.com/kyma-project/helm-broker/-/badge.svg)](https://sourcegraph.com/github.com/kyma-project/helm-broker?badge)

## Overview

Helm Broker is a [Service Broker](https://kyma-project.io/docs/components/service-catalog/#overview-service-brokers) that exposes Helm charts as Service Classes in [Service Catalog](https://kyma-project.io/docs/components/service-catalog/#overview-service-catalog). To do so, Helm Broker uses the concept of addons. An addon is an abstraction layer over a Helm chart which provides all information required to convert the chart into a Service Class.

Helm Broker fetches default cluster-wide addons defined by the [helm-repos-urls](https://github.com/kyma-project/kyma/blob/main/resources/helm-broker/templates/default-addons-cfg.yaml) custom resource (CR). This CR contains URLs that point to the release of the [`addons`](https://github.com/kyma-project/addons/releases) repository compatible with a given [Kyma release](https://github.com/kyma-project/kyma/releases). You can also configure Helm Broker to fetch addons definitions from other addons repositories.

You can install Helm Broker either as a standalone project, or as part of [Kyma](https://kyma-project.io/). In Kyma, you can use addons to install the following Service Brokers:

* Azure Service Broker
* AWS Service Broker
* GCP Service Broker

>**NOTE:** Starting from Kyma 2.0, Helm Broker will no longer be supported.

To see all addons that Helm Broker provides, go to the [`addons`](https://github.com/kyma-project/addons) repository.

Helm Broker implements the [Open Service Broker API](https://github.com/openservicebrokerapi/servicebroker/blob/v2.14/profile.md#service-metadata) (OSB API). To be compliant with Service Catalog version used in Kyma, Helm Broker supports only the following OSB API versions:
- v2.13
- v2.12
- v2.11

> **NOTE:** Helm Broker does not implement the OSB API update operation.

### Next steps

To install Helm Broker and develop the project, read the [installation](./docs/01-installation.md) document. For more details, tutorials, and troubleshooting, explore the [documentation](./docs) directory.

## Project structure

The `helm-broker` repository has the following structure:

```
  ├── .github                   # Pull request and issue templates    
  ├── charts                    # Charts to install by Helm
  ├── cmd                       # Main applications of the project                                     
  ├── config                    # Configuration file templates or default configurations
  ├── deploy                    # Dockerfiles to build applications image
  │
  ├── docs                      # Documentation related to the project
  │    ├── assets                  # Diagrams and assets used in the documentation
  │    └── internal                # Proposals and release-related documentation
  │
  ├── hack                      # Scripts used by the Helm Broker developers
  │    ├── boilerplate              # Header used while generating code
  │    ├── ci                       # Source of the test for charts
  │    ├── examples                 # Example Kubernetes objects  
  │    └── release                  # Release pipeline scripts
  │
  ├── internal                  # Private application and library code
  │    ├── addon                    # Package that provides logic for fetching addons from different remote repositories
  │    ├── rafter                   # Client for the upload service which allows the Helm Broker to upload documentation
  │    ├── bind                     # Logic that renders the binding data
  │    ├── broker                   # Implementation of the OSB API contract
  │    ├── config                   # Configurations structs for both Controller and Broker
  │    ├── controller               # Logic of the ClusterAddonsConfigurations and AddonsConfigurations controllers
  │    ├── health                   # Handlers of the liveness and readiness probes
  │    ├── helm                     # Client for Helm
  │    ├── platform                 # Internal minor packages, such as logger or idProvider
  │    ├── storage                  # Storage layer for both memory and ETCD provider, based on factory design pattern
  │    └── model.go                 # All structs used in the project
  │
  ├── pkg                       # Library code to use by external applications
  │    ├── apis                     # Structs definitions for ClusterAddonsConfigurations and AddonsConfigurations
  │    └── client                   # Typed client for ClusterAddonsConfigurations and AddonsConfigurations
  │
  └── test                      # Additional external test applications and test data
       ├── charts                   # Implementation of the test for the `helm-broker` chart
       └── integration              # Implementation of the integration test
```
