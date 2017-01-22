PROJ=dex
ORG_PATH=github.com/coreos
REPO_PATH=$(ORG_PATH)/$(PROJ)
export PATH := $(PWD)/bin:$(PATH)

VERSION=$(shell ./scripts/git-version)

DOCKER_REPO=quay.io/coreos/dex
DOCKER_IMAGE=$(DOCKER_REPO):$(VERSION)

$( shell mkdir -p bin )
$( shell mkdir -p _output/images )
$( shell mkdir -p _output/bin )

user=$(shell id -u -n)
group=$(shell id -g -n)

export GOBIN=$(PWD)/bin
# Prefer ./bin instead of system packages for things like protoc, where we want
# to use the version dex uses, not whatever a developer has installed.
export PATH=$(GOBIN):$(shell printenv PATH)
export GO15VENDOREXPERIMENT=1

LD_FLAGS="-w -X $(REPO_PATH)/version.Version=$(VERSION)"

build: bin/dex bin/example-app

bin/dex: check-go-version
	@go install -v -ldflags $(LD_FLAGS) $(REPO_PATH)/cmd/dex

bin/example-app: check-go-version
	@go install -v -ldflags $(LD_FLAGS) $(REPO_PATH)/cmd/example-app

.PHONY: release-binary
release-binary:
	@go build -o _output/bin/dex -v -ldflags $(LD_FLAGS) $(REPO_PATH)/cmd/dex

.PHONY: revendor
revendor:
	@glide up -v
	@glide-vc --use-lock-file --no-tests --only-code

test:
	@go test -v -i $(shell go list ./... | grep -v '/vendor/')
	@go test -v $(shell go list ./... | grep -v '/vendor/')

testrace:
	@go test -v -i --race $(shell go list ./... | grep -v '/vendor/')
	@go test -v --race $(shell go list ./... | grep -v '/vendor/')

vet:
	@go vet $(shell go list ./... | grep -v '/vendor/')

fmt:
	@go fmt $(shell go list ./... | grep -v '/vendor/')

lint:
	@for package in $(shell go list ./... | grep -v '/vendor/' | grep -v '/api' | grep -v '/server/internal'); do \
      golint -set_exit_status $$package $$i || exit 1; \
	done

_output/bin/dex:
	# Using rkt to build the dex binary.
	@./scripts/rkt-build
	@sudo chown $(user):$(group) _output/bin/dex

_output/images/library-alpine-3.4.aci:
	@mkdir -p _output/images
	# Using docker2aci to get a base ACI to build from.
	@docker2aci docker://alpine:3.4
	@mv library-alpine-3.4.aci _output/images/library-alpine-3.4.aci

.PHONY: aci
aci: clean-release _output/bin/dex _output/images/library-alpine-3.4.aci
	# Using acbuild to build a application container image.
	@sudo ./scripts/build-aci ./_output/images/library-alpine-3.4.aci
	@sudo chown $(user):$(group) _output/images/dex.aci
	@mv _output/images/dex.aci _output/images/dex-$(VERSION)-linux-amd64.aci

.PHONY: docker-image
docker-image: clean-release _output/bin/dex
	@sudo docker build -t $(DOCKER_IMAGE) .

.PHONY: proto
proto: api/api.pb.go server/internal/types.pb.go

api/api.pb.go: api/api.proto bin/protoc bin/protoc-gen-go
	@protoc --go_out=plugins=grpc:. api/*.proto

server/internal/types.pb.go: server/internal/types.proto bin/protoc bin/protoc-gen-go
	@protoc --go_out=. server/internal/*.proto

bin/protoc: scripts/get-protoc
	@./scripts/get-protoc bin/protoc

bin/protoc-gen-go:
	@go install -v $(REPO_PATH)/vendor/github.com/golang/protobuf/protoc-gen-go

.PHONY: check-go-version
check-go-version:
	@./scripts/check-go-version

clean: clean-release
	@rm -rf bin/

.PHONY: clean-release
clean-release:
	@rm -rf _output/

testall: testrace vet fmt lint

FORCE:

.PHONY: test testrace vet fmt lint testall
