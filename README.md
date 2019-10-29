# Helm Broker

## Overview

The Helm Broker is a [Service Broker](https://kyma-project.io/docs/master/components/service-catalog/#service-brokers-overview) which exposes Helm charts as Service Classes in the [Service Catalog](https://kyma-project.io/docs/master/components/service-catalog/#overview-overview). To do so, the Helm Broker uses the concept of addons. An addon is an abstraction layer over a Helm chart which provides all information required to convert the chart into a Service Class.

If you want to use the Helm Broker with all dependencies, try out [Kyma](https://kyma-project.io/).

## Documentation

<center>

[Installation](https://github.com/kyma-project/helm-broker/blob/master/docs/installation.md)

[Usage](https://github.com/kyma-project/helm-broker/blob/master/docs/usage.md)

[Development](https://github.com/kyma-project/helm-broker/blob/master/docs/development.md)

[Releasing](https://github.com/kyma-project/helm-broker/blob/master/docs/releasing.md)

</center>

### Project structure

The repository has the following structure:

```
  ├── .github                     # Pull request and issue templates    
  ├── charts                      # Charts to install by Helm
  ├── cmd                         # Main applications for project                                     
  ├── config                      # Configuration file templates or default configurations
  ├── deploy                      # Dockerfiles to build images
  ├── docs                        # Documentation files
  ├── hack                        # Scripts used by the Helm Broker developers
  ├── internal                    # Private application and library code
  ├── pkg                         # Library code to use by external applications
  └── test                        # Additional external test applications and test data
```
