# Helm Operator (Flux) vs Helm

In this document you can read about findings about [helm 3](https://helm.sh/docs/) pros and cons in comparision with the [helm operator](https://docs.fluxcd.io/projects/helm-operator/en/latest/).

## Helm 3 features

The Helm 3 has numerous new features, but a few deserve highlighting here:

- Releases are stored in a new format
- There is no in-cluster (Tiller) component
- Helm 3 includes support for a new version of Helm charts (Charts v2)
- Helm 3 also supports library charts -- charts that are used primarily as a resource for other charts.
- Experimental support for storing Helm charts in OCI registries (e.g. Docker Distribution) is available for testing.
- A 3-way strategic merge patch is now applied when upgrading Kubernetes resources.
- A chart's supplied values can now be validated against a JSON schema
- A number of small improvements have been made to make Helm more secure, usable, and robust.

## Helm Operator features

The Helm operator is a [Flux](https://github.com/fluxcd/flux) extension which automates Helm Charts in a GitOps manner.

See the features it provides:
- Declarative helm install/upgrade/delete of charts
- Pulls charts from public or private Helm repositories over HTTPS
- Pulls charts from public or private Git repositories over SSH
- Chart release values can be specified inline in the HelmRelease object or via secrets, configmaps or URLs
- Automated chart upgrades based on container image tag policies (requires Flux)
- Automatic purging on chart install failures
- Automatic rollback on chart upgrade failures

## Pros and cons

The Helm Operator beyond the Helm features like managing the charts provides also a lot of features for the Github.

With actual implementation of the Helm Broker, implementing the Helm Operator or Helm 3 should be similar.

In Helm Broker we use go-getter library which help us to store AddOns in the Github repositories, that approach can be improved easily with the Helm Operator features.

If possible we should migrate to Helm Operator when the first use-case for it will show up.

## For future

We should remember about features that Helm Operator provides, and track the issues which are blocking us from migration to it.

The Helm Operator will support Helm 3 after implementing this [issue](https://github.com/fluxcd/helm-operator/issues/8)