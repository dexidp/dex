PROJ=dex
ORG_PATH=github.com/coreos
REPO_PATH=$(ORG_PATH)/$(PROJ)

export GOBIN=$(PWD)/bin
export GOPATH=$(shell ./scripts/gopath)

# Makefile environment vairables aren't passed to "shell" calls. Need to set
# them explicitly.
PACKAGES=$(shell ./scripts/go list github.com/coreos/dex/... | grep -v '/vendor/')

VERSION ?= $(shell ./scripts/git-version)

DOCKER_REPO=quay.io/coreos/dex
DOCKER_IMAGE=$(DOCKER_REPO):$(VERSION)


LD_FLAGS="-w -X $(REPO_PATH)/version.Version=$(VERSION)"

build: bin/dex bin/example-app bin/grpc-client

.PHONY: pre-build
pre-build:
	@mkdir -p bin
	@./scripts/check-go-version

bin/dex: pre-build
	@go install -v -ldflags $(LD_FLAGS) $(REPO_PATH)/cmd/dex

bin/example-app: pre-build
	@go install -v -ldflags $(LD_FLAGS) $(REPO_PATH)/cmd/example-app

bin/grpc-client: pre-build
	@go install -v -ldflags $(LD_FLAGS) $(REPO_PATH)/examples/grpc-client

.PHONY: revendor
revendor:
	@glide up -v
	@glide-vc --use-lock-file --no-tests --only-code

test:
	@go test -v -i $(PACKAGES)
	@go test -v $(PACKAGES)

testrace:
	@go test -v -i --race $(PACKAGES)
	@go test -v --race $(PACKAGES)

vet:
	@go vet $(PACKAGES)

fmt:
	@go fmt $(PACKAGES)

lint:
	@for package in $(shell ./scripts/go list github.com/coreos/dex/... | grep -v '/vendor/' | grep -v '/api' | grep -v '/server/internal'); do \
      golint -set_exit_status $$package $$i || exit 1; \
	done

_output/bin/dex:
	@./scripts/docker-build
	@user=$(shell id -u -n)
	@group=$(shell id -g -n)
	@sudo chown $(user):$(group) _output/bin/dex

.PHONY: docker-image
docker-image: clean-release _output/bin/dex
	@sudo docker build -t $(DOCKER_IMAGE) .

.PHONY: proto
proto: api/api.pb.go server/internal/types.pb.go

api/api.pb.go: api/api.proto bin/protoc bin/protoc-gen-go
	@./bin/protoc --go_out=plugins=grpc:. api/*.proto

server/internal/types.pb.go: server/internal/types.proto bin/protoc bin/protoc-gen-go
	@./bin/protoc --go_out=. server/internal/*.proto

bin/protoc: scripts/get-protoc
	@./scripts/get-protoc bin/protoc

bin/protoc-gen-go:
	@go install -v $(REPO_PATH)/vendor/github.com/golang/protobuf/protoc-gen-go

clean: clean-release
	@rm -rf bin/
	@unlink _gopath/src/github.com/coreos/dex
	@rm -r _gopath

.PHONY: clean-release
clean-release:
	@rm -rf _output/

testall: testrace vet fmt lint

FORCE:

.PHONY: test testrace vet fmt lint testall
