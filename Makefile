APP_NAME = dex
# BASE_PKG is a root packge of the component
BASE_PKG := github.com/kyma-incubator/dex
# IMG_GOPATH is a path to go path in the container
IMG_GOPATH := /go
BUILDPACK = eu.gcr.io/kyma-project/test-infra/buildpack-golang-toolbox:v20190913-65b55d1

IMG_NAME := $(DOCKER_PUSH_REPOSITORY)$(DOCKER_PUSH_DIRECTORY)/$(APP_NAME)
TAG := $(DOCKER_TAG)

# COMPONENT_DIR is a local path to commponent
COMPONENT_DIR = $(shell pwd)
# WORKSPACE_LOCAL_DIR is a path to the scripts folder in the container
WORKSPACE_LOCAL_DIR = $(IMG_GOPATH)/src/$(BASE_PKG)
# WORKSPACE_COMPONENT_DIR is a path to commponent in the container
WORKSPACE_COMPONENT_DIR = $(IMG_GOPATH)/src/$(BASE_PKG)

# Base docker configuration
DOCKER_CREATE_OPTS := -v $(LOCAL_DIR):$(WORKSPACE_LOCAL_DIR):delegated --rm -w $(WORKSPACE_COMPONENT_DIR) $(BUILDPACK)

# Check if go is available
ifneq (,$(shell go version 2>/dev/null))
DOCKER_CREATE_OPTS := -v $(shell go env GOCACHE):$(IMG_GOCACHE):delegated -v $(shell go env GOPATH)/pkg/dep:$(IMG_GOPATH)/pkg/dep:delegated $(DOCKER_CREATE_OPTS)
endif

# Check if running with TTY
ifeq (1, $(shell [ -t 0 ] && echo 1))
DOCKER_INTERACTIVE := -i
DOCKER_CREATE_OPTS := -t $(DOCKER_CREATE_OPTS)
else
DOCKER_INTERACTIVE_START := --attach
endif

# Buildpack directives
define buildpack-mount
.PHONY: $(1)-local $(1)
$(1):
	@echo make $(1)
	@docker run $(DOCKER_INTERACTIVE) \
		-v $(COMPONENT_DIR):$(WORKSPACE_COMPONENT_DIR):delegated \
		$(DOCKER_CREATE_OPTS) make $(1)-local
endef

VERIFY_IGNORE := /vendor\|/automock
DIRS_TO_CHECK = go list ./... | grep -v "$(VERIFY_IGNORE)"
FILES_TO_CHECK = find . -type f -name "*.go" | grep -v "$(VERIFY_IGNORE)"

release: test vet check-fmt build-image push-image

# Targets mounting sources to buildpack
MOUNT_TARGETS = test vet check-fmt fmt fix-fmt
$(foreach t,$(MOUNT_TARGETS),$(eval $(call buildpack-mount,$(t))))

test-local:
	go test -v ./...

vet-local:
	go vet $$($(DIRS_TO_CHECK))

check-fmt-local:
	exit $(shell gofmt -l $$($(FILES_TO_CHECK)) | wc -l | xargs)

fmt-local:
	gofmt -l $$($(FILES_TO_CHECK))

fix-fmt-local:
	gofmt -w $$($(FILES_TO_CHECK))

build-image: pull-licenses
	docker build -t $(IMG_NAME) .

push-image:
	docker tag $(IMG_NAME) $(IMG_NAME):$(TAG)
	docker push $(IMG_NAME):$(TAG)

pull-licenses:
ifdef LICENSE_PULLER_PATH
	bash $(LICENSE_PULLER_PATH)
else
	mkdir -p licenses
endif