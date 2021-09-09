# Directory to put `go install`ed binaries in.
export GOBIN ?= $(shell pwd)/bin

ROOT_PATH := $(shell pwd)
REPO = $(DOCKER_PUSH_REPOSITORY)$(DOCKER_PUSH_DIRECTORY)/
TAG = $(DOCKER_TAG)
GIT_TAG=$(PULL_BASE_REF)
GIT_REPO=$(REPO_OWNER)/$(REPO_NAME)
export GITHUB_TOKEN=$(BOT_GITHUB_TOKEN)

APP_NAME = helm-broker
TOOLS_NAME = helm-broker-tools
TESTS_NAME = helm-broker-tests
CONTROLLER_NAME = helm-controller
WEBHOOK_NAME = helm-broker-webhook

# VERIFY_IGNORE is a grep pattern to exclude files and directories from verification
VERIFY_IGNORE := /vendor\|/automock
# FILES_TO_CHECK is a command used to determine which files should be verified
FILES_TO_CHECK = find . -type f -name "*.go" | grep -v "$(VERIFY_IGNORE)"
# DIRS_TO_CHECK is a command used to determine which directories should be verified
DIRS_TO_CHECK = go list ./... | grep -v "$(VERIFY_IGNORE)"

build:: build-binaries verify test

verify:: vet check-imports-local check-fmt-local

format:: vet goimports fmt golint clean

test:: unit-test integration-test

.PHONY: build-binaries
build-binaries:
	./hack/build-binaries.sh

.PHONY: integration-test
integration-test:
	export KUBEBUILDER_CONTROLPLANE_START_TIMEOUT=2m
	go test -tags=integration ./test/integration/

.PHONY: unit-test
unit-test:
	go test ./internal/... ./cmd/...

.PHONY: charts-test
charts-test:
	./hack/ci/run-chart-test.sh

# Run go fmt against code
.PHONY: fmt
fmt:
	go fmt ./internal/... ./cmd/...

# Run go vet against code
.PHONY: vet
vet:
	go vet $$($(DIRS_TO_CHECK))

.PHONY: golint
golint:
	@go install golang.org/x/lint/golint
	@$(GOBIN)/golint $$($(FILES_TO_CHECK))

.PHONY: goimports
goimports:
	@go install golang.org/x/tools/cmd/goimports
	@$(GOBIN)/goimports  -w -l $$($(FILES_TO_CHECK))

.PHONY: check-imports-local
check-imports-local:
	@if [ -n "$$(goimports -l $$($(FILES_TO_CHECK)))" ]; then \
		echo "✗ some files are not properly formatted or contain not formatted imports. To repair run make goimports"; \
		goimports -l $$($(FILES_TO_CHECK)); \
		exit 1; \
	fi;

.PHONY: check-fmt-local
check-fmt-local:
	@if [ -n "$$(gofmt -l $$($(FILES_TO_CHECK)))" ]; then \
		gofmt -l $$($(FILES_TO_CHECK)); \
		echo "✗ some files are not properly formatted. To repair run make fmt"; \
		exit 1; \
	fi;

.PHONY: pull-licenses
pull-licenses:
ifdef LICENSE_PULLER_PATH
	bash $(LICENSE_PULLER_PATH)
else
	mkdir -p licenses
endif

.PHONY: generates
# Generate CRD manifests, clients etc.
generates: crd-manifests client

.PHONY: crd-manifests
# Generate CRD manifests
crd-manifests: get-yaml-editor
	go run vendor/sigs.k8s.io/controller-tools/cmd/controller-gen/main.go crd paths=./pkg/apis/... output:crd:dir=charts/helm-broker/templates/crd
	mv charts/helm-broker/templates/crd/addons.kyma-project.io_addonsconfigurations.yaml charts/helm-broker/templates/crd/addons-configuration.crd.yaml
	mv charts/helm-broker/templates/crd/addons.kyma-project.io_clusteraddonsconfigurations.yaml charts/helm-broker/templates/crd/cluster-addons-configuration.crd.yaml
	$(YAML_EDITOR) d -i charts/helm-broker/templates/crd/addons-configuration.crd.yaml metadata.annotations
	$(YAML_EDITOR) w -i charts/helm-broker/templates/crd/addons-configuration.crd.yaml metadata.annotations["helm.sh/hook"]  "crd-install"
	$(YAML_EDITOR) d -i charts/helm-broker/templates/crd/cluster-addons-configuration.crd.yaml metadata.annotations
	$(YAML_EDITOR) w -i charts/helm-broker/templates/crd/cluster-addons-configuration.crd.yaml metadata.annotations["helm.sh/hook"]  "crd-install"

.PHONY: client
client:
	./hack/update-codegen.sh

.PHONY: release
release: tar-chart append-changelog release-branch
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
	cp webhook deploy/webhook/webhook
	cp hb_chart_test deploy/tests/hb_chart_test

	docker build -t $(APP_NAME) deploy/broker
	docker build -t $(CONTROLLER_NAME) deploy/controller
	docker build -t $(WEBHOOK_NAME) deploy/webhook
	docker build -t $(TOOLS_NAME) deploy/tools
	docker build -t $(TESTS_NAME) deploy/tests

.PHONY: push-image
push-image:
	docker tag $(APP_NAME) $(REPO)$(APP_NAME):$(TAG)
	docker push $(REPO)$(APP_NAME):$(TAG)
ifeq ($(JOB_TYPE), postsubmit)
	@echo "Sign image with Cosign"
	cosign version
	cosign sign -key ${KMS_KEY_URL} $(REPO)$(APP_NAME):$(TAG)
else
	@echo "Image signing skipped"
endif

	docker tag $(CONTROLLER_NAME) $(REPO)$(CONTROLLER_NAME):$(TAG)
	docker push $(REPO)$(CONTROLLER_NAME):$(TAG)
ifeq ($(JOB_TYPE), postsubmit)
	@echo "Sign image with Cosign"
	cosign version
	cosign sign -key ${KMS_KEY_URL} $(REPO)$(CONTROLLER_NAME):$(TAG)
else
	@echo "Image signing skipped"
endif

	docker tag $(WEBHOOK_NAME) $(REPO)$(WEBHOOK_NAME):$(TAG)
	docker push $(REPO)$(WEBHOOK_NAME):$(TAG)
ifeq ($(JOB_TYPE), postsubmit)
	@echo "Sign image with Cosign"
	cosign version
	cosign sign -key ${KMS_KEY_URL} $(REPO)$(WEBHOOK_NAME):$(TAG)
else
	@echo "Image signing skipped"
endif

	docker tag $(TOOLS_NAME) $(REPO)$(TOOLS_NAME):$(TAG)
	docker push $(REPO)$(TOOLS_NAME):$(TAG)
ifeq ($(JOB_TYPE), postsubmit)
	@echo "Sign image with Cosign"
	cosign version
	cosign sign -key ${KMS_KEY_URL} $(REPO)$(TOOLS_NAME):$(TAG)
else
	@echo "Image signing skipped"
endif

	docker tag $(TESTS_NAME) $(REPO)$(TESTS_NAME):$(TAG)
	docker push $(REPO)$(TESTS_NAME):$(TAG)
ifeq ($(JOB_TYPE), postsubmit)
	@echo "Sign image with Cosign"
	cosign version
	cosign sign -key ${KMS_KEY_URL} $(REPO)$(TESTS_NAME):$(TAG)
else
	@echo "Image signing skipped"
endif

.PHONY: ci-pr
ci-pr: build build-image push-image

.PHONY: ci-master
ci-master: build build-image push-image

.PHONY: ci-release
ci-release: build build-image push-image charts-test release

.PHONY: clean
clean:
	rm -f broker
	rm -f controller
	rm -f webhook
	rm -f targz
	rm -f indexbuilder
	rm -f hb_chart_test
	rm -rf bin/

.PHONY: path-to-referenced-charts
path-to-referenced-charts:
	@echo "resources/helm-broker"

.PHONY: get-yaml-editor
get-yaml-editor:
	$(eval YAML_EDITOR=$(shell ./hack/release/get_yq.sh))
