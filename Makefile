# The old school Makefile, following are required targets. The Makefile is written
# to allow building multiple binaries. You are free to add more targets or change
# existing implementations, as long as the semantics are preserved.
#
#   make                - default to 'build' target
#   make lint           - code analysis
#   make test           - run unit test (or plus integration test)
#   make build          - alias to build-local target
#   make build-local    - build local binary targets
#   make build-linux    - build linux binary targets
#   make build-coverage - build local binary targets for code-coverage
#   make container      - build containers
#   $ docker login registry -u username -p xxxxx
#   make push           - push containers
#   make clean          - clean up targets
#
# Not included but recommended targets:
#   make e2e-test
#
# The makefile is also responsible to populate project version information.
#

#
# Tweak the variables based on your project.
#

# This repo's root import path (under GOPATH).
ROOT := github.com/qiankunli/workflow

# Module name.
NAME := workflow

# Container image prefix and suffix added to targets.
# The final built images are:
#   $[REGISTRY]/$[IMAGE_PREFIX]$[TARGET]$[IMAGE_SUFFIX]:$[VERSION]
# $[REGISTRY] is an item from $[REGISTRIES], $[TARGET] is an item from $[TARGETS].
IMAGE_PREFIX ?= $(strip )
IMAGE_SUFFIX ?= $(strip )

# Container registries.
REGISTRY ?= hub.docker.com/qiankunli

# Container registry for base images.
BASE_REGISTRY ?= hub.docker.com/qiankunli

# Helm chart repo
CHART_REPO ?= charts

#
# These variables should not need tweaking.
#

# It's necessary to set this because some environments don't link sh -> bash.
export SHELL := /bin/bash

# It's necessary to set the errexit flags for the bash shell.
export SHELLOPTS := errexit

# Project main package location.
CMD_DIR := ./cmd

# Project output directory.
OUTPUT_DIR := ./bin

# Build directory.
BUILD_DIR := ./build

IMAGE_NAME := $(IMAGE_PREFIX)$(NAME)$(IMAGE_SUFFIX)

# Current version of the project.
VERSION      ?= $(shell git describe --tags --always --dirty)
BRANCH       ?= $(shell git branch | grep \* | cut -d ' ' -f2)
GITCOMMIT    ?= $(shell git rev-parse HEAD)
GITTREESTATE ?= $(if $(shell git status --porcelain),dirty,clean)
BUILDDATE    ?= $(shell date -u +"%Y-%m-%dT%H:%M:%SZ")
appVersion   ?= $(VERSION)

# Available cpus for compiling, please refer to https://github.com/caicloud/engineering/issues/8186#issuecomment-518656946 for more information.
CPUS ?= $(shell /bin/bash hack/read_cpus_available.sh)

# Track code version with Docker Label.
DOCKER_LABELS ?= git-describe="$(shell date -u +v%Y%m%d)-$(shell git describe --tags --always --dirty)"

# Golang standard bin directory.
GOPATH ?= $(shell go env GOPATH)
BIN_DIR := $(GOPATH)/bin
GOLANGCI_LINT := $(BIN_DIR)/golangci-lint
HELM_LINT := /usr/local/bin/helm
NIRVANA := $(OUTPUT_DIR)/nirvana

# Default golang flags used in build and test
# -count: run each test and benchmark 1 times. Set this flag to disable test cache
export GOFLAGS ?= -count=1

#
# Define all targets. At least the following commands are required:
#

# All targets.
.PHONY: lint test build container push build-coverage

build: build-local

# more info about `GOGC` env: https://github.com/golangci/golangci-lint#memory-usage-of-golangci-lint
lint: $(GOLANGCI_LINT) $(HELM_LINT)
	@$(GOLANGCI_LINT) run
	@bash hack/helm-lint.sh

$(GOLANGCI_LINT):
	curl -sfL https://install.goreleaser.com/github.com/golangci/golangci-lint.sh | sh -s -- -b $(BIN_DIR) v1.23.6

$(HELM_LINT):
	curl https://raw.githubusercontent.com/helm/helm/master/scripts/get-helm-3 | sudo bash

test:
	@go test -gcflags=all=-l -gcflags=all=-d=checkptr=0 -race -coverpkg=./... -coverprofile=coverage.out.tmp $(shell go list ./... | grep /pkg | grep -v /pkg/generated)
	@sed -e  '/kitex_gen/d' coverage.out.tmp > coverage.out
	@go tool cover -func coverage.out | tail -n 1 | awk '{ print "Total coverage: " $$3 }'

build-local:
	@go build -v -o $(OUTPUT_DIR)/$(NAME)                                  \
	  -ldflags "-s -w -X $(ROOT)/pkg/version.module=$(NAME)                \
	    -X $(ROOT)/pkg/version.version=$(VERSION)                          \
	    -X $(ROOT)/pkg/version.branch=$(BRANCH)                            \
	    -X $(ROOT)/pkg/version.gitCommit=$(GITCOMMIT)                      \
	    -X $(ROOT)/pkg/version.gitTreeState=$(GITTREESTATE)                \
	    -X $(ROOT)/pkg/version.buildDate=$(BUILDDATE)"                     \
	  $(CMD_DIR);

build-linux:
	/bin/bash -c 'GOOS=linux GOARCH=amd64 GOFLAGS="$(GOFLAGS)"  \
	  go build -v -o $(OUTPUT_DIR)/$(NAME)                                 \
	    -ldflags "-s -w -X $(ROOT)/pkg/version.module=$(NAME)              \
	      -X $(ROOT)/pkg/version.version=$(VERSION)                        \
	      -X $(ROOT)/pkg/version.branch=$(BRANCH)                          \
	      -X $(ROOT)/pkg/version.gitCommit=$(GITCOMMIT)                    \
	      -X $(ROOT)/pkg/version.gitTreeState=$(GITTREESTATE)              \
	      -X $(ROOT)/pkg/version.buildDate=$(BUILDDATE)"                   \
		$(CMD_DIR)'

build-coverage:
	# skip specified dir with --skip-files, separated by comma
	@go_coverage annotate --main-folder=$(CMD_DIR)
	@echo '{"branch_name": "$(BRANCH)", "commit_id":"$(GITCOMMIT)"}' > revision_file.txt
	/bin/bash -c 'GOOS=linux GOARCH=amd64 GOPATH=/go GOFLAGS="$(GOFLAGS)"  \
	  go build -v -o $(OUTPUT_DIR)/$(NAME)                                 \
	    -ldflags "-s -w -X $(ROOT)/pkg/version.module=$(NAME)              \
	      -X $(ROOT)/pkg/version.version=$(VERSION)                        \
	      -X $(ROOT)/pkg/version.branch=$(BRANCH)                          \
	      -X $(ROOT)/pkg/version.gitCommit=$(GITCOMMIT)                    \
	      -X $(ROOT)/pkg/version.gitTreeState=$(GITTREESTATE)              \
	      -X $(ROOT)/pkg/version.buildDate=$(BUILDDATE)"                   \
		$(CMD_DIR)'

container:
	@docker build -t $(REGISTRY)/$(IMAGE_NAME):$(VERSION)                  \
	  --label $(DOCKER_LABELS)                                             \
	  -f $(BUILD_DIR)/Dockerfile .;

push: container
	@docker push $(REGISTRY)/$(IMAGE_NAME):$(VERSION);

.PHONY: clean
clean:
	@-rm -vrf ${OUTPUT_DIR}

clientset-clean:
	@rm -rf ./pkg/generated/*
	@find ./pkg/apis -maxdepth 3 -mindepth 2 -name 'zz_generated.*.go' -exec rm -f {} \;

clientset-gen: clientset-clean
	@bash hack/update-codegen.sh

crd-clean:
	@rm -rf ./crds/*
crd-gen: crd-clean
	@go install sigs.k8s.io/controller-tools/cmd/controller-gen@v0.6.0
	controller-gen crd:crdVersions=v1 paths=./pkg/apis/... output:crd:artifacts:config=./manifests/workflow-controller/crds

# check if make clientset-gen and make client are executed
.PHONY: check
check: clientset-gen
	@bash hack/check.sh

.PHONY: gen
gen: clientset-gen crd-gen

gen-clean: crd-clean clientset-clean

.PHONY: images-list check-images

images-list:
	@find manifests -type f | sed -n 's/Chart.yaml//p' \
	| xargs -L1 helm template --set platformConfig.imageRegistry=REGISTRY_DOMAIN \
	--set platformConfig.imageRepositoryRelease=release \
	--set platformConfig.imageRepositoryLibrary=library \
	| grep -Eo "REGISTRY_DOMAIN/.*:.*" | sed "s#^REGISTRY_DOMAIN/##g" \
	| tr -d "[:blank:],']{}\[\]\`\"" | sort -u | tee /tmp/images.list
	@echo "" >> /tmp/images.list
	@find manifests -type f -name "images*.list" \
	| xargs -L1 grep -E '^library|^release' >> /tmp/images.list || true

.PHONY: changelog
changelog:
	@git fetch --prune-tags
	@git-chglog --next-tag $(VERSION) -o CHANGELOG.md

generate:
	@go generate ./...


