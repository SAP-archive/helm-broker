# Helm Operator (Flux) vs Helm

In this document you can read about findings about the [Helm 3](https://helm.sh/docs/) in comparision with the [Helm operator](https://docs.fluxcd.io/projects/helm-operator/en/latest/).

## Helm 3 features

The new version of Helm has numerous new features, but a few deserve highlighting here:

- Releases are stored in a new format (in secrets, not like before in the config maps)
- There is no in-cluster (Tiller) component
- New version of [Helm charts](https://helm.sh/docs/topics/charts/)
- [Library charts](https://helm.sh/docs/topics/library_charts/) - charts that are used primarily as a resource for other charts
- Experimental support for storing Helm charts in OCI registries is available for testing
- A 3-way strategic merge patch is now applied when upgrading Kubernetes resources
- A chart's supplied values can now be validated against a JSON schema
- A number of small improvements have been made to make Helm more secure, usable, and robust

## Helm Operator features

The Helm operator is a [FluxCD](https://github.com/fluxcd/flux) extension which automates Helm Charts in a GitOps manner.

See the features it provides:
- Declarative install, upgrade, and delete of HelmRelease CR
- Pulls chart from any chart source:
    - Public or private Helm repositories over HTTP/S
    - Public or private Git repositories over HTTPS or SSH
    - Any other public or private chart source using one of the available Helm downloader plugins
- Allows Helm values to be specified:
    - In-line in the HelmRelease resource
    - In (external) sources, e.g. ConfigMap and Secret resources, or a (local) URL
- Automated purging on release install failures
- Automated (optional) rollback on upgrade failures
- Automated image upgrades (Flux)
- Automated (configurable) chart dependency updates for Helm charts from Git sources on install or upgrade (Flux)
- Detection and recovery from Helm storage mutations (e.g. a manual Helm release that was made but conflicts with the declared configuration for the release)
- Parallel and scalable processing of different HelmRelease resources using workers
- Supports both Helm 2 and 3

## Implementation

Read below sections for more about implementation details in Helm Broker.

### Helm 3

The Helm 3 provides a support for `v2` charts, so it shouldn't be a problem to implement it in the Helm Broker. Even without adjusting the AddOns itself.

To download the Helm 3 binary, use the following command:

```
curl https://raw.githubusercontent.com/helm/helm/master/scripts/get-helm-3 | bash
```

### Helm Operator

In order to use Helm Operator approach, we may need to provide the following dependencies to your cluster:

- [Helm Operator](https://github.com/fluxcd/helm-operator/tree/v1.1.0/chart/helm-operator)
- [HelmRelease CR](https://raw.githubusercontent.com/fluxcd/helm-operator/1.1.0/deploy/crds.yaml)
- **Optional** [FluxCD](https://github.com/fluxcd/helm-operator/tree/master/chart/helm-operator)

>**NOTE:** When working with FluxCD use [FluxCTL](https://github.com/fluxcd/flux/releases/tag/1.19.0).

With this solution the Helm Broker would need to create a HelmRelease CR in the provisioning process. It wouldn't need the Helm client anymore.

The Helm Operator beyond the Helm features like managing the charts provides also a lot of features for the Github, but most of them can be used only when FluxCD is configured and installed on the cluster.

Using Flux you can 

### Migration

The Helm 3 provides a migration [CLI](https://github.com/helm/helm-2to3) along with the [documentation](https://helm.sh/docs/topics/v2_v3_migration/).

## Links

Explore the following links to expand your knowledge about above things:

- [Example app deploy with Helm 3](https://www.civo.com/learn/guide-to-helm-3-with-an-express-js-microservice)
- [Example app automation with Helm Operator and Flux](https://www.civo.com/learn/gitops-using-helm3-and-flux-for-an-node-js-and-express-js-microservice)
- [Explanation of the Helm 3 features](https://thenewstack.io/helm-3-is-almost-boring-and-thats-a-great-sign-of-maturity/)