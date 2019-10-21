# Helm Broker's Change Log

## [Unreleased](https://github.com/kyma-project/helm-broker/tree/HEAD)

:rocket: **Implemented enhancements:**

- Improve controller readiness probe [\#29](https://github.com/kyma-project/helm-broker/issues/29)
- Add Helm Chart testing scripts [\#32](https://github.com/kyma-project/helm-broker/pull/32) ([mszostok](https://github.com/mszostok))
- S3 integration test [\#28](https://github.com/kyma-project/helm-broker/pull/28) ([jasiu001](https://github.com/jasiu001))
- Add GCS,S3, and HG protocols support [\#22](https://github.com/kyma-project/helm-broker/pull/22) ([mszostok](https://github.com/mszostok))
- Change structs to more readable and improve common logic [\#10](https://github.com/kyma-project/helm-broker/pull/10) ([polskikiel](https://github.com/polskikiel))

:bug: **Fixed bugs:**

- No retry when DocsTopic creation fails [\#33](https://github.com/kyma-project/helm-broker/issues/33)
- Apply minor changes to utilities.sh script [\#37](https://github.com/kyma-project/helm-broker/pull/37) ([mszostok](https://github.com/mszostok))

:no_entry_sign: **Closed issues:**

- Test EPIC [\#44](https://github.com/kyma-project/helm-broker/issues/44)
- Helm Broker chart testing [\#23](https://github.com/kyma-project/helm-broker/issues/23)

:heavy_check_mark: **Other improvements:**

- Upgrade to alpine 3.10 [\#42](https://github.com/kyma-project/helm-broker/pull/42) ([piotrmiskiewicz](https://github.com/piotrmiskiewicz))
- Fix etcd client timeout [\#40](https://github.com/kyma-project/helm-broker/pull/40) ([polskikiel](https://github.com/polskikiel))
- Add Maja and Alex to CODEOWNERS [\#39](https://github.com/kyma-project/helm-broker/pull/39) ([klaudiagrz](https://github.com/klaudiagrz))
- Add waiter for DocsTopics [\#38](https://github.com/kyma-project/helm-broker/pull/38) ([polskikiel](https://github.com/polskikiel))
- Add controller liveness probe with controller flow [\#36](https://github.com/kyma-project/helm-broker/pull/36) ([jasiu001](https://github.com/jasiu001))
- Fix OWNERS file regex [\#27](https://github.com/kyma-project/helm-broker/pull/27) ([mszostok](https://github.com/mszostok))
- Change reprocessRequest type to int [\#26](https://github.com/kyma-project/helm-broker/pull/26) ([polskikiel](https://github.com/polskikiel))
- Add OWNERS files [\#25](https://github.com/kyma-project/helm-broker/pull/25) ([mszostok](https://github.com/mszostok))
- Implement health checking for HelmBroker [\#24](https://github.com/kyma-project/helm-broker/pull/24) ([polskikiel](https://github.com/polskikiel))
- Bump Helm Broker docker image version [\#21](https://github.com/kyma-project/helm-broker/pull/21) ([piotrmiskiewicz](https://github.com/piotrmiskiewicz))
- Disable upload service usage if documentation enabled flag is set to … [\#20](https://github.com/kyma-project/helm-broker/pull/20) ([piotrmiskiewicz](https://github.com/piotrmiskiewicz))
- Fix helm chart yaml file [\#18](https://github.com/kyma-project/helm-broker/pull/18) ([piotrmiskiewicz](https://github.com/piotrmiskiewicz))
- Git SSH support [\#17](https://github.com/kyma-project/helm-broker/pull/17) ([piotrmiskiewicz](https://github.com/piotrmiskiewicz))
- Align HB chart after introducing templating [\#16](https://github.com/kyma-project/helm-broker/pull/16) ([polskikiel](https://github.com/polskikiel))
- Implement templating from secret [\#14](https://github.com/kyma-project/helm-broker/pull/14) ([polskikiel](https://github.com/polskikiel))
- Templating IT test [\#13](https://github.com/kyma-project/helm-broker/pull/13) ([piotrmiskiewicz](https://github.com/piotrmiskiewicz))
- Extend AddonsConfiguration CRD with secretRef [\#12](https://github.com/kyma-project/helm-broker/pull/12) ([polskikiel](https://github.com/polskikiel))
- Enable prometheus metrics endpoint [\#11](https://github.com/kyma-project/helm-broker/pull/11) ([jasiu001](https://github.com/jasiu001))
- Make tiller TLS enabled by configuration [\#9](https://github.com/kyma-project/helm-broker/pull/9) ([piotrmiskiewicz](https://github.com/piotrmiskiewicz))
- Add helm broker chart [\#8](https://github.com/kyma-project/helm-broker/pull/8) ([jasiu001](https://github.com/jasiu001))
- Improve docs [\#7](https://github.com/kyma-project/helm-broker/pull/7) ([piotrmiskiewicz](https://github.com/piotrmiskiewicz))
- Delete addons configs migration [\#6](https://github.com/kyma-project/helm-broker/pull/6) ([polskikiel](https://github.com/polskikiel))
- Addons docs configurable [\#4](https://github.com/kyma-project/helm-broker/pull/4) ([piotrmiskiewicz](https://github.com/piotrmiskiewicz))
- Add repo templates [\#3](https://github.com/kyma-project/helm-broker/pull/3) ([adamwalach](https://github.com/adamwalach))
- Add CODEOWNERS [\#2](https://github.com/kyma-project/helm-broker/pull/2) ([adamwalach](https://github.com/adamwalach))
- Update imports after repository migration [\#1](https://github.com/kyma-project/helm-broker/pull/1) ([adamwalach](https://github.com/adamwalach))



\* *This Change Log was automatically generated by [github_changelog_generator](https://github.com/skywinder/Github-Changelog-Generator)*