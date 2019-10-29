# Helm Broker

## Overview

The Helm Broker is a [Service Broker](https://kyma-project.io/docs/master/components/service-catalog/#service-brokers-overview) which exposes Helm charts as Service Classes in the [Service Catalog](https://kyma-project.io/docs/master/components/service-catalog/#overview-overview). To do so, the Helm Broker uses the concept of addons. An addon is an abstraction layer over a Helm chart which provides all information required to convert the chart into a Service Class.

If you want to use the Helm Broker with all dependencies, try out [Kyma](https://kyma-project.io/).

### Table of contents

- [Installation](https://github.com/kyma-project/helm-broker/blob/master/docs/installation.md)
- [Usage](https://github.com/kyma-project/helm-broker/blob/master/docs/usage.md)
- [Development](https://github.com/kyma-project/helm-broker/blob/master/docs/development.md)
- [Releasing](https://github.com/kyma-project/helm-broker/blob/master/docs/releasing.md)

### Project structure

The repository has the following structure:

```
  ├── .github                   # Pull request and issue templates    
  ├── charts                    # Charts to install by Helm
  ├── cmd                       # Main applications for project                                     
  ├── config                    # Configuration file templates or default configurations
  ├── deploy                    # Dockerfiles to build applications image
  │
  ├── docs                      # Documentation files
  │    ├── proposals                # Documentation about proposed architecture decisions
  │    └── release                  # Documentation used during release process
  │
  ├── hack                      # Scripts used by the Helm Broker developers
  │    ├── boilerplate              # Header used in code generation
  │    ├── ci                       # Chart-test source
  │    ├── examples                 # Example Kubernetes objects  
  │    └── release                  # Release pipeline scripts
  │
  ├── internal                  # Private application and library code
  │    ├── addon                    # Package providing logic for fetching Addons from different remote repositories
  │    ├── assetstore               # Contains client for upload service which allows Helm Broker to upload a documentation
  │    ├── bind                     # Provides logic to render binding data
  │    ├── broker                   # Contains implementation of the OSB API contract
  │    ├── config                   # Contains configurations structs for both controller and broker
  │    ├── controller               # Contains logic of two controllers - `ClusterAddonsConfigurations` and `AddonsConfigurations`
  │    ├── health                   # Provides the handlers of the liveness and readiness probes
  │    ├── helm                     # Provides a client for Helm
  │    ├── platform                 # Contains internal minor packages like logger or idProvider
  │    ├── storage                  # Storage layer for both memory and ETCD provider, based on factory design pattern
  │    └── model.go                 # Contains all structs used in the project
  │
  ├── pkg                       # Library code to use by external applications
  │    ├── apis                     # Contains `ClusterAddonsConfigurations` and `AddonsConfigurations` structs definitions
  │    └── client                   # Provides a typed client for `ClusterAddonsConfigurations` and `AddonsConfigurations`
  │
  └── test                      # Additional external test applications and test data
       ├── charts                   # Contains implementation of the helm-broker's chart test
       └── integration              # Contains implementation of the integration test
```
