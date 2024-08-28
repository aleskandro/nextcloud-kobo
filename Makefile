ifeq ($(shell test -f .env && echo -n yes),yes)
 include .env
endif

ARTIFACT_DIR ?= ./_output

#### Tool Versions ####
# https://github.com/golangci/golangci-lint/releases
GOLINT_VERSION = v1.60.1
NO_DOCKER ?= 0

# Get the currently used golang install path (in GOPATH/bin, unless GOBIN is set)
ifeq (,$(shell go env GOBIN))
GOBIN=$(shell go env GOPATH)/bin
else
GOBIN=$(shell go env GOBIN)
endif

# Setting SHELL to bash allows bash commands to be executed by recipes.
# Options are set to exit when a recipe line exits non-zero or a piped command fails.
SHELL = /usr/bin/env bash -o pipefail
.SHELLFLAGS = -ec

SRC_IMAGE ?= golang:1.23

GO111MODULE = on
export GO111MODULE

ifeq ($(shell command -v podman > /dev/null 2>&1 ; echo $$? ), 0)
	ENGINE=podman
else ifeq ($(shell command -v docker > /dev/null 2>&1 ; echo $$? ), 0)
	ENGINE=docker
else
	NO_DOCKER=1
endif

FORCE_DOCKER ?= 0
ifeq ($(FORCE_DOCKER), 1)
	ENGINE=docker
endif

ifeq ($(NO_DOCKER), 1)
  DOCKER_CMD =
else
  DOCKER_CMD := $(ENGINE) run --env GO111MODULE=$(GO111MODULE) --env GOLINT_VERSION=$(GOLINT_VERSION) --rm -v "$(PWD)":/go/src/github.com/aleskandro/nextcloud-kobo:Z -w /go/src/github.com/aleskandro/nextcloud-kobo $(SRC_IMAGE)
endif

.PHONY: static
static: fmt vet goimports gosec lint

.PHONY: fmt
fmt:
	$(DOCKER_CMD) hack/go-fmt.sh ./

.PHONY: vet
vet:
	$(DOCKER_CMD) go vet ./...

.PHONY: lint
lint:
	GOLINT_VERSION=$(GOLINT_VERSION) $(DOCKER_CMD) hack/golangci-lint.sh ./...

.PHONY: goimports
goimports:
	$(DOCKER_CMD) hack/goimports.sh .

.PHONY: gosec
gosec:
	$(DOCKER_CMD) hack/gosec.sh ./...

.PHONY: koboroot
koboroot:
	hack/package.sh

.PHONY: unit
unit:
	$(DOCKER_CMD) go test -v ./...