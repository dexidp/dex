OS = $(shell uname | tr A-Z a-z)

export PATH := $(abspath bin/protoc/bin/):$(abspath bin/):${PATH}

PROJ=dex
ORG_PATH=github.com/dexidp
REPO_PATH=$(ORG_PATH)/$(PROJ)

VERSION ?= $(shell ./scripts/git-version)

DOCKER_REPO=quay.io/dexidp/dex
DOCKER_IMAGE=$(DOCKER_REPO):$(VERSION)

$( shell mkdir -p bin )

user=$(shell id -u -n)
group=$(shell id -g -n)

export GOBIN=$(PWD)/bin

LD_FLAGS="-w -X main.version=$(VERSION)"

# Dependency versions

KIND_NODE_IMAGE = "kindest/node:v1.19.11@sha256:07db187ae84b4b7de440a73886f008cf903fcf5764ba8106a9fd5243d6f32729"
KIND_TMP_DIR = "$(PWD)/bin/test/dex-kind-kubeconfig"

.PHONY: generate
generate:
	@go generate $(REPO_PATH)/storage/ent/

build: generate bin/dex

bin/dex:
	@mkdir -p bin/
	@go install -v -ldflags $(LD_FLAGS) $(REPO_PATH)/cmd/dex

examples: bin/grpc-client bin/example-app

bin/grpc-client:
	@mkdir -p bin/
	@cd examples/ && go install -v -ldflags $(LD_FLAGS) $(REPO_PATH)/examples/grpc-client

bin/example-app:
	@mkdir -p bin/
	@cd examples/ && go install -v -ldflags $(LD_FLAGS) $(REPO_PATH)/examples/example-app

.PHONY: release-binary
release-binary: LD_FLAGS = "-w -X main.version=$(VERSION) -extldflags \"-static\""
release-binary: generate
	@go build -o /go/bin/dex -v -ldflags $(LD_FLAGS) $(REPO_PATH)/cmd/dex
	@go build -o /go/bin/docker-entrypoint -v -ldflags $(LD_FLAGS) $(REPO_PATH)/cmd/docker-entrypoint

docker-compose.override.yaml:
	cp docker-compose.override.yaml.dist docker-compose.override.yaml

.PHONY: up
up: docker-compose.override.yaml ## Launch the development environment
	@ if [ docker-compose.override.yaml -ot docker-compose.override.yaml.dist ]; then diff -u docker-compose.override.yaml docker-compose.override.yaml.dist || (echo "!!! The distributed docker-compose.override.yaml example changed. Please update your file accordingly (or at least touch it). !!!" && false); fi
	docker-compose up -d

.PHONY: down
down: clear ## Destroy the development environment
	docker-compose down --volumes --remove-orphans --rmi local

test:
	@go test -v ./...

testrace:
	@go test -v --race ./...

.PHONY: kind-up kind-down kind-tests
kind-up:
	@mkdir -p bin/test
	@kind create cluster --image ${KIND_NODE_IMAGE} --kubeconfig ${KIND_TMP_DIR}

kind-down:
	@kind delete cluster
	rm ${KIND_TMP_DIR}

kind-tests: export DEX_KUBERNETES_CONFIG_PATH=${KIND_TMP_DIR}
kind-tests: testall

.PHONY: lint lint-fix
lint: ## Run linter
	golangci-lint run

.PHONY: fix
fix: ## Fix lint violations
	golangci-lint run --fix

.PHONY: docker-image
docker-image:
	@sudo docker build -t $(DOCKER_IMAGE) .

.PHONY: verify-proto
verify-proto: proto
	@./scripts/git-diff

clean:
	@rm -rf bin/

testall: testrace

FORCE:

.PHONY: test testrace testall

.PHONY: proto
proto:
	@protoc --go_out=paths=source_relative:. --go-grpc_out=paths=source_relative:. api/v2/*.proto
	@protoc --go_out=paths=source_relative:. --go-grpc_out=paths=source_relative:. api/*.proto
	#@cp api/v2/*.proto api/

.PHONY: proto-internal
proto-internal:
	@protoc --go_out=paths=source_relative:. server/internal/*.proto

# Dependency versions
GOLANGCI_VERSION = 1.46.0
GOTESTSUM_VERSION ?= 1.7.0
PROTOC_VERSION = 3.15.6
PROTOC_GEN_GO_VERSION = 1.26.0
PROTOC_GEN_GO_GRPC_VERSION = 1.1.0
KIND_VERSION = 0.11.1

deps: bin/gotestsum bin/golangci-lint bin/protoc bin/protoc-gen-go bin/protoc-gen-go-grpc bin/kind

bin/gotestsum:
	@mkdir -p bin
	curl -L https://github.com/gotestyourself/gotestsum/releases/download/v${GOTESTSUM_VERSION}/gotestsum_${GOTESTSUM_VERSION}_$(shell uname | tr A-Z a-z)_amd64.tar.gz | tar -zOxf - gotestsum > ./bin/gotestsum
	@chmod +x ./bin/gotestsum

bin/golangci-lint:
	@mkdir -p bin
	curl -sSfL https://raw.githubusercontent.com/golangci/golangci-lint/master/install.sh | BINARY=golangci-lint bash -s -- v${GOLANGCI_VERSION}

bin/protoc:
	@mkdir -p bin/protoc
ifeq ($(shell uname | tr A-Z a-z), darwin)
	curl -L https://github.com/protocolbuffers/protobuf/releases/download/v${PROTOC_VERSION}/protoc-${PROTOC_VERSION}-osx-x86_64.zip > bin/protoc.zip
endif
ifeq ($(shell uname | tr A-Z a-z), linux)
	curl -L https://github.com/protocolbuffers/protobuf/releases/download/v${PROTOC_VERSION}/protoc-${PROTOC_VERSION}-linux-x86_64.zip > bin/protoc.zip
endif
	unzip bin/protoc.zip -d bin/protoc
	rm bin/protoc.zip

bin/protoc-gen-go:
	@mkdir -p bin
	curl -L https://github.com/protocolbuffers/protobuf-go/releases/download/v${PROTOC_GEN_GO_VERSION}/protoc-gen-go.v${PROTOC_GEN_GO_VERSION}.$(shell uname | tr A-Z a-z).amd64.tar.gz | tar -zOxf - protoc-gen-go > ./bin/protoc-gen-go
	@chmod +x ./bin/protoc-gen-go

bin/protoc-gen-go-grpc:
	@mkdir -p bin
	curl -L https://github.com/grpc/grpc-go/releases/download/cmd/protoc-gen-go-grpc/v${PROTOC_GEN_GO_GRPC_VERSION}/protoc-gen-go-grpc.v${PROTOC_GEN_GO_GRPC_VERSION}.$(shell uname | tr A-Z a-z).amd64.tar.gz | tar -zOxf - ./protoc-gen-go-grpc > ./bin/protoc-gen-go-grpc
	@chmod +x ./bin/protoc-gen-go-grpc

bin/kind:
	@mkdir -p bin
	curl -L https://github.com/kubernetes-sigs/kind/releases/download/v${KIND_VERSION}/kind-$(shell uname | tr A-Z a-z)-amd64 > ./bin/kind
	@chmod +x ./bin/kind
