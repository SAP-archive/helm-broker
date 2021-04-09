# Documentation

## Overview

This directory contains Helm Broker documentation.

The Helm Broker fetches addons which contain a set of specific [files](#details-create-addons). You must place your addons in a repository of an appropriate [format](#details-create-addons-repository). The Helm Broker fetches default cluster-wide addons defined by the [helm-repos-urls](https://github.com/kyma-project/kyma/blob/master/resources/helm-broker/templates/default-addons-cfg.yaml) custom resource (CR). This CR contains URLs that point to the release of [`addons`](https://github.com/kyma-project/addons/releases) repository compatible with a given [Kyma release](https://github.com/kyma-project/kyma/releases). You can also configure the Helm Broker to fetch addons definitions from other addons repositories.

If you want to create your addons and store them in your own repository, start with these documents:
  - [Create addons](./04-create-addons.md)
  - [Bind addons](./05-bind-addons.md)
  - [Test addons](./06-test-addons.md)
  - [Create addons repository](./07-create-addons-repo.md)

If you want to learn more about the architecture, read these docs:
  - [Architecture](./02-architecture.md)
  - [Architecture deep dive](./03-architecture-deep-dive.md)

Here are the custom resources that Helm Broker uses:
  - [AddonsConfiguration](./13-cr-addonsconfiguration.md)
  - [ClusterAddonsConfiguration](./14-cr-clusteraddonsconfiguration.md)

For more detailed information, [configuration](./12-configuration.md), and [troubleshooting](./14-troubleshooting.md), read the other docs in this directory. If you want to know more about Helm Broker release process, read [this](./release/hb-release.md) document.


### Project structure

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
