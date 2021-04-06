# Documentation

## Overview
This directory contains the following documents that relate to the project:

- [Installation](https://github.com/kyma-project/helm-broker/blob/master/docs/installation.md) describes how to install the Helm Broker.
- [Configuration](https://github.com/kyma-project/helm-broker/blob/master/docs/configuration.md) describes the environment variables you can configure.
- [Development](https://github.com/kyma-project/helm-broker/blob/master/docs/development.md) describes how to develop the project.
- [Releasing](https://github.com/kyma-project/helm-broker/blob/master/docs/releasing.md) describes the Helm Broker release process.
- [Example usage](https://github.com/kyma-project/helm-broker/blob/master/docs/example-usage.md) describes how to provision a Redis instance using the Helm Broker and addons.



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
