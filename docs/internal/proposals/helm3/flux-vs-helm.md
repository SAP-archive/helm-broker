# Helm Operator (Flux) vs Helm

This document presents a comparison between [Helm 3](https://helm.sh/docs/) and [Helm Operator](https://docs.fluxcd.io/projects/helm-operator/en/latest/).

## Helm 3 features

The new version of Helm has numerous new features, but a few deserve highlighting here:

- Releases are stored in a new format (in Secrets created in the release's Namespace, not in ConfigMaps in the `kube-system` Namespace). See the example of such a Secret: 

```
NAME                          TYPE              
sh.helm.release.v1.redis.v1   helm.sh/release.v1
```

- There are no in-cluster components (such as Tiller)
- New version of [Helm charts](https://helm.sh/docs/topics/charts/)
- [Library charts](https://helm.sh/docs/topics/library_charts/) that are used primarily as a resource for other charts
- Experimental support for storing [Helm charts in OCI registries](https://helm.sh/docs/topics/registries/) is available for testing
- A [3-way strategic merge patch](https://helm.sh/docs/faq/#improved-upgrade-strategy-3-way-strategic-merge-patches) is now applied when upgrading Kubernetes resources
- Chart's supplied values can now be [validated](https://helm.sh/docs/faq/#validating-chart-values-with-jsonschema) against a JSON schema
- A number of small improvements have been made to make Helm more secure, usable, and robust

## Helm Operator features

Helm Operator is a [FluxCD](https://github.com/fluxcd/flux) extension that automates Helm charts in a GitOps manner.

See the features it provides:
- Declarative install, upgrade, and delete of the HelmRelease CR
- Pulling charts from any chart source:
    - Public or private Helm repositories over HTTP/S
    - Public or private Git repositories over HTTPS or SSH
    - Any other public or private chart source using one of the available Helm downloader plugins
- Allowing Helm values to be specified:
    - In-line in the HelmRelease resource
    - In external sources, such as ConfigMap and Secret resources, or a local URL
- Automated purging on release install failures
- Automated (optional) rollback on upgrade failures
- Automated image upgrades (Flux)
- Automated (configurable) chart dependency updates for Helm charts from Git sources on install or upgrade (Flux)
- Detection and recovery from Helm storage mutations (for example, a manual Helm release that conflicts with the declared configuration for the release)
- Parallel and scalable processing of different HelmRelease resources using workers
- Support for both Helm 2 and 3

## Pros and cons

In the following tabs, you can find pros and cons of the above solutions.

<div tabs>
  <details>
  <summary>
  Helm 3
  </summary>
   
   <br/>:heavy_plus_sign: Is a more lightweight solution
   <br/>:heavy_plus_sign: Provides new types of charts
   <br/>:heavy_plus_sign: Does not need any in-cluster application
   <br/>:heavy_plus_sign: Can be extended with the Helm Operator
   <br/>:heavy_plus_sign: Does not need any CR to work
   
   <br/>:heavy_minus_sign: Does not provide automated processes
   <br/>:heavy_minus_sign: Does not provide parallel executions
   
  </details>

  <details>
  <summary>
  Helm Operator
  </summary>
   
   <br/>:heavy_plus_sign: Can automate many things and improve customer experience
   <br/>:heavy_plus_sign: Provides scalable parallel processing
   <br/>:heavy_plus_sign: User can still use vanilla Helm 3
   <br/>:heavy_plus_sign: Can download a chart from any repository
   <br/>:heavy_plus_sign: Can be extended with Flux
   
   <br/>:heavy_minus_sign: Maintenance
   <br/>:heavy_minus_sign: Increased resources consumption
   <br/>:heavy_minus_sign: Helm Operator Pod has super admin permissions like Tiller (will be changed)
   <br/>:heavy_minus_sign: New controller along with its CR
   <br/>:heavy_minus_sign: It requires more implementation in Helm Broker than the Helm 3 approach
   <br/>:heavy_minus_sign: Another dependency of Helm Broker, may require a feature flag 

  </details>
</div>

## Implementation

Read the sections below for more information about implementation details in Helm Broker.

### Helm 3

Helm 3 provides support for `v2` charts, so it shouldn't be a problem to implement it in the Helm Broker, even without adjusting the addons themselves.

To download the Helm 3 binary, use the following command:

```
curl https://raw.githubusercontent.com/helm/helm/master/scripts/get-helm-3 | bash
```

#### Migration

Helm 3 provides [CLI migration](https://github.com/helm/helm-2to3) along with the corresponding [documentation](https://helm.sh/docs/topics/v2_v3_migration/).

### Helm Operator

In order to use the Helm Operator approach, we may need to provide the following dependencies to our cluster:

- [Helm Operator](https://github.com/fluxcd/helm-operator/tree/v1.1.0/chart/helm-operator)
- [HelmRelease CR](https://raw.githubusercontent.com/fluxcd/helm-operator/1.1.0/deploy/crds.yaml)
- **Optional** [FluxCD](https://github.com/fluxcd/helm-operator/tree/master/chart/helm-operator) along with [memcached](https://github.com/memcached/memcached)

>**NOTE:** When working with FluxCD, use [FluxCTL](https://github.com/fluxcd/flux/releases/tag/1.19.0).

With this solution, the Helm Broker would need to create a HelmRelease CR in the provisioning process. It wouldn't need the Helm client anymore.

In order not to download the same charts two times, both by the Helm Broker and by the Helm Operator, the logic of downloading addons would need to be replaced by some other logic which could read only the `index.yaml` file and to create a proper response from its entries on the `/v2/catalog` endpoint. 

#### Flux

Helm Operator provides a lot of GitOps features, but most of them can be used only when FluxCD is configured and installed on the cluster.

Using Flux, you can automate the process of delivering the applications' latest changes after their docker images are built. For example, the Helm Broker could mark the HelmRelease created within instance provisioning with such a label:
```
fluxcd.io/automated: "true"
```

There are also other labels which you can use to configure the automation process.

## Summary

In my opinion, implementing the Helm Operator's support in Helm Broker would require a lot of effort to adjust the current implementation, but automated processes and other features look interesting and I wouldn't rule them out to implement in the future.  

It seems the best choice is to stay with plain Helm 3.

### Links

Explore the following links to expand your knowledge about above things:

- [Example app deploy with Helm 3](https://www.civo.com/learn/guide-to-helm-3-with-an-express-js-microservice)
- [Example app automation with Helm Operator and Flux](https://www.civo.com/learn/gitops-using-helm3-and-flux-for-an-node-js-and-express-js-microservice)
- [Explanation of the Helm 3 features](https://thenewstack.io/helm-3-is-almost-boring-and-thats-a-great-sign-of-maturity/)
- [Deep Dive: Flux the GitOps Operator for Kubernetes](https://www.youtube.com/watch?v=Fs_Oz-RzWWI)
- [Flagger - Kubernetes operator that automates the promotion of canary deployments](https://docs.flagger.app/)
