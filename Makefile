ROOT_PATH := $(shell pwd)
GIT_TAG=$(PULL_BASE_REF)
GIT_REPO=$(REPO_OWNER)/$(REPO_NAME)
export GITHUB_TOKEN=$(BOT_GITHUB_TOKEN)

APP_NAME = helm-broker
TOOLS_NAME = helm-broker-tools
TESTS_NAME = helm-broker-tests
CONTROLLER_NAME = helm-controller

REPO = $(DOCKER_PUSH_REPOSITORY)$(DOCKER_PUSH_DIRECTORY)/
TAG = $(DOCKER_TAG)

.PHONY: build
build:
	./before-commit.sh ci

.PHONY: integration-test
integration-test:
	export KUBEBUILDER_CONTROLPLANE_START_TIMEOUT=2m
	go test -tags=integration ./test/integration/

.PHONY: charts-test
charts-test:
	./hack/ci/run-chart-test.sh

.PHONY: pull-licenses
pull-licenses:
ifdef LICENSE_PULLER_PATH
	bash $(LICENSE_PULLER_PATH)
else
	mkdir -p licenses
endif

# Caution! Remove the “namespace: v.namespace” parameter after regeneration of
# “components/helm-broker/pkg/client/informers/externalversions/addons/v1alpha1/interface.go” file.
# clusterAddonsConfigurationInformer doesn’t have the “namespace” field
.PHONY: generates
# Generate CRD manifests, clients etc.
generates: crd-manifests client

.PHONY: crd-manifests
# Generate CRD manifests
crd-manifests:
	go run vendor/sigs.k8s.io/controller-tools/cmd/controller-gen/main.go crd --domain kyma-project.io

.PHONY: client
client:
	./hack/update-codegen.sh

.PHONY: release
release: tar-chart append-changelog release-branch
	./hack/release/push_release.sh $(GIT_TAG) $(GIT_REPO)

.PHONY: latest-release
latest-release: set-latest-tag tag-chart-images tar-chart update-release-docs append-changelog
	./hack/release/create_latest_tag.sh $(GIT_REPO)
	./hack/release/remove_latest_tag.sh $(GIT_REPO)
	./hack/release/push_release.sh $(GIT_TAG) $(GIT_REPO)

.PHONY: generate-changelog
generate-changelog:
	./hack/release/generate_changelog.sh $(GIT_TAG) $(REPO_NAME) $(REPO_OWNER)

.PHONY: append-changelog
append-changelog: generate-changelog
	./hack/release/append_changelog.sh

.PHONY: release-branch
release-branch:
# release branch named `release-x.y` will be created if the GIT_TAG matches the `x.y.0` version pattern.
	./hack/release/create_release_branch.sh $(GIT_TAG) $(GIT_REPO)

.PHONY: set-latest-tag
set-latest-tag:
	$(eval GIT_TAG=latest)
	$(eval TAG=latest)
	$(eval VERSION=latest)

.PHONY: tar-chart
tar-chart: create-release-dir
	@tar -czvf toCopy/helm-broker-chart.tar.gz -C charts/helm-broker/ . ||:

.PHONY: create-release-dir
create-release-dir:
	mkdir -p toCopy

.PHONY: cut-release
cut-release: tag-chart-images update-release-docs
	git add .
	git commit -m "Bump version to: $(VERSION)"
	git tag $(VERSION)

.PHONY: update-release-docs
update-release-docs:
	./hack/release/update_release_docs.sh $(VERSION)

.PHONY: tag-chart-images
tag-chart-images: get-yaml-editor
	$(YAML_EDITOR) w -i charts/helm-broker/values.yaml global.helm_broker.version $(VERSION)
	$(YAML_EDITOR) w -i charts/helm-broker/values.yaml global.helm_broker.dir '$(DIR)'
	$(YAML_EDITOR) w -i charts/helm-broker/values.yaml global.helm_controller.version $(VERSION)
	$(YAML_EDITOR) w -i charts/helm-broker/values.yaml global.helm_controller.dir '$(DIR)'
	$(YAML_EDITOR) w -i charts/helm-broker/values.yaml tests.tag $(VERSION)
	$(YAML_EDITOR) w -i charts/helm-broker/values.yaml tests.dir '$(DIR)'

.PHONY: build-image
build-image: pull-licenses
	cp broker deploy/broker/helm-broker
	cp targz deploy/tools/targz
	cp indexbuilder deploy/tools/indexbuilder
	cp controller deploy/controller/controller
	cp hb_chart_test deploy/tests/hb_chart_test

	docker build -t $(APP_NAME) deploy/broker
	docker build -t $(CONTROLLER_NAME) deploy/controller
	docker build -t $(TOOLS_NAME) deploy/tools
	docker build -t $(TESTS_NAME) deploy/tests

.PHONY: push-image
push-image:
	docker tag $(APP_NAME) $(REPO)$(APP_NAME):$(TAG)
	docker push $(REPO)$(APP_NAME):$(TAG)

	docker tag $(CONTROLLER_NAME) $(REPO)$(CONTROLLER_NAME):$(TAG)
	docker push $(REPO)$(CONTROLLER_NAME):$(TAG)

	docker tag $(TOOLS_NAME) $(REPO)$(TOOLS_NAME):$(TAG)
	docker push $(REPO)$(TOOLS_NAME):$(TAG)

	docker tag $(TESTS_NAME) $(REPO)$(TESTS_NAME):$(TAG)
	docker push $(REPO)$(TESTS_NAME):$(TAG)

.PHONY: ci-pr
ci-pr: build integration-test build-image push-image

.PHONY: ci-master
ci-master: build integration-test build-image push-image latest-release push-image

.PHONY: ci-release
ci-release: build integration-test build-image push-image charts-test release

.PHONY: clean
clean:
	rm -f broker
	rm -f controller
	rm -f targz
	rm -f indexbuilder

.PHONY: path-to-referenced-charts
path-to-referenced-charts:
	@echo "resources/helm-broker"

.PHONY: get-yaml-editor
get-yaml-editor:
	$(eval YAML_EDITOR=$(shell ./hack/release/get_yq.sh))
