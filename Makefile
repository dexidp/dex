PROJ=dex
ORG_PATH=github.com/coreos
REPO_PATH=$(ORG_PATH)/$(PROJ)
export PATH := $(PWD)/bin:$(PATH)

VERSION=$(shell ./scripts/git-version)

DOCKER_REPO=quay.io/ericchiang/dex
DOCKER_IMAGE=$(DOCKER_REPO):$(VERSION)

$( shell mkdir -p bin )

export GOBIN=$(PWD)/bin
# Prefer ./bin instead of system packages for things like protoc, where we want
# to use the version dex uses, not whatever a developer has installed.
export PATH=$(GOBIN):$(shell printenv PATH)
export GO15VENDOREXPERIMENT=1

LD_FLAGS="-w -X $(REPO_PATH)/version.Version=$(VERSION)"

build: bin/dex bin/example-app

bin/dex: FORCE generated
	@go install -v -ldflags $(LD_FLAGS) $(REPO_PATH)/cmd/dex

bin/example-app: FORCE
	@go install -v -ldflags $(LD_FLAGS) $(REPO_PATH)/cmd/example-app

.PHONY: generated
generated: server/templates_default.go

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
	@for package in $(shell go list ./... | grep -v '/vendor/' | grep -v '/api'); do \
      golint -set_exit_status $$package; \
	done

server/templates_default.go: $(wildcard web/templates/**)
	@go run server/templates_default_gen.go

.PHONY: docker-build
docker-build: bin/dex
	@docker build -t $(DOCKER_IMAGE) .

.PHONY: docker-push
docker-push: docker-build
	@docker tag $(DOCKER_IMAGE) $(DOCKER_REPO):latest
	@docker push $(DOCKER_IMAGE)
	@docker push $(DOCKER_REPO):latest

.PHONY: grpc
grpc: api/api.pb.go

api/api.pb.go: api/api.proto bin/protoc bin/protoc-gen-go
	@protoc --go_out=plugins=grpc:. api/*.proto

bin/protoc: scripts/get-protoc
	@./scripts/get-protoc bin/protoc

bin/protoc-gen-go:
	@go install -v $(REPO_PATH)/vendor/github.com/golang/protobuf/protoc-gen-go

clean:
	@rm bin/*

testall: testrace vet fmt lint

FORCE:

.PHONY: test testrace vet fmt lint testall
