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

.PHONY: generate-changelog
generate-changelog:
	$(eval FLAGS:=)
ifneq ($(shell ./scripts/get_last_tag.sh),)
	$(eval FLAGS+= --since-tag)
	$(eval FLAGS+= $(shell ./scripts/get_last_tag.sh))
endif
ifneq ($(GIT_TAG),)
	$(eval FLAGS+= --future-release)
	$(eval FLAGS+= $(GIT_TAG))
endif
	@docker run --rm -v $(ROOT_PATH):/usr/local/src/your-app ferrarimarco/github-changelog-generator -u $(REPO_OWNER) -p $(REPO_NAME) -t $(GITHUB_TOKEN) $(FLAGS) ||:

.PHONY: release
release: tag-release-images tar-chart generate-changelog release-branch
	./scripts/push_release.sh $(GIT_TAG) $(GIT_REPO)

.PHONY: latest-release
latest-release: tar-chart generate-changelog
	./scripts/create_latest_tag.sh $(GIT_REPO)
	./scripts/remove_latest_tag.sh $(GIT_REPO)
	./scripts/push_release.sh $(GIT_TAG) $(GIT_REPO)

.PHONY: tar-chart
tar-chart:
	tar -czvf helm-broker-chart.tar.gz -C charts/helm-broker/ . &> /dev/null

.PHONY: tag-release-images
tag-release-images:
	$(eval TAG:=$(GIT_TAG))
ifeq ($(TAG),)
	$(eval TAG:=$(VERSION))
endif
	@docker run --rm -v $(ROOT_PATH):/workdir mikefarah/yq yq w -i charts/helm-broker/values.yaml global.helm_broker.version $(TAG) ||:
	@docker run --rm -v $(ROOT_PATH):/workdir mikefarah/yq yq w -i charts/helm-broker/values.yaml global.helm_controller.version $(TAG) ||:
	@docker run --rm -v $(ROOT_PATH):/workdir mikefarah/yq yq w -i charts/helm-broker/values.yaml tests.tag $(TAG) ||:

.PHONY: cut-release
cut-release: tag-release-images
	git add charts/helm-broker/values.yaml
	git commit -m "Bump release images"
	git tag $(VERSION)

.PHONY: release-branch
release-branch:
# release branch named `release-x.y` will be created if the GIT_TAG matches the `x.y.0` version pattern.
	./scripts/create_release_branch.sh $(GIT_TAG) $(GIT_REPO)

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
ci-master: build integration-test build-image push-image latest-release

.PHONY: ci-release
ci-release: build integration-test build-image push-image charts-test release

.PHONY: clean
clean:
	rm -f broker
	rm -f targz
	rm -f indexbuilder

.PHONY: path-to-referenced-charts
path-to-referenced-charts:
	@echo "resources/helm-broker"
