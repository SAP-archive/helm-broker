# Helm Broker

[![Go Report Card](https://goreportcard.com/badge/github.com/kyma-project/helm-broker)](https://goreportcard.com/report/github.com/kyma-project/helm-broker)
[![Sourcegraph](https://sourcegraph.com/github.com/kyma-project/helm-broker/-/badge.svg)](https://sourcegraph.com/github.com/kyma-project/helm-broker?badge)

## Overview

The Helm Broker is a [Service Broker](https://kyma-project.io/docs/master/components/service-catalog/#service-brokers-overview) which exposes Helm charts as Service Classes in the [Service Catalog](https://kyma-project.io/docs/master/components/service-catalog/#overview-overview). To do so, the Helm Broker uses the concept of [addons](https://github.com/kyma-project/addons). An addon is an abstraction layer over a Helm chart which provides all information required to convert the chart into a Service Class. To learn more about the Helm Broker, read the [documentation](https://github.com/kyma-project/helm-broker/blob/master/docs/README.md).

If you want to use the Helm Broker with all dependencies, try out [Kyma](https://kyma-project.io/).

### Project structure

The repository has the following structure:

```
  ├── .github                   # Pull request and issue templates    
  ├── charts                    # Charts to install by Helm
  ├── cmd                       # Main applications of the project                                     
  ├── config                    # Configuration file templates or default configurations
  ├── deploy                    # Dockerfiles to build applications image
  │
  ├── docs                      # Documentation related to the project
  │    ├── proposals                # Proposed architecture decisions
  │    └── release                  # Release notes template
  │
  ├── hack                      # Scripts used by the Helm Broker developers
  │    ├── boilerplate              # Header used while generating code
  │    ├── ci                       # Source of the test for charts
  │    ├── examples                 # Example Kubernetes objects  
  │    └── release                  # Release pipeline scripts
  │
  ├── internal                  # Private application and library code
  │    ├── addon                    # Package that provides logic for fetching addons from different remote repositories
  │    ├── assetstore               # Client for the upload service which allows the Helm Broker to upload documentation
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
