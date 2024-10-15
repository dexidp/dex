export PATH := $(abspath bin/protoc/bin/):$(abspath bin/):${PATH}
export SHELL := env PATH=$(PATH) /bin/sh

OS = $(shell uname | tr A-Z a-z)

user=$(shell id -u -n)
group=$(shell id -g -n)

$( shell mkdir -p bin )

PROJ      = dex
ORG_PATH  = github.com/dexidp
REPO_PATH = $(ORG_PATH)/$(PROJ)
VERSION  ?= $(shell ./scripts/git-version)


export GOBIN=$(PWD)/bin
LD_FLAGS="-w -X main.version=$(VERSION)"

# Dependency versions
GOLANGCI_VERSION   = 1.56.2
GOTESTSUM_VERSION ?= 1.10.1

PROTOC_VERSION             = 24.4
PROTOC_GEN_GO_VERSION      = 1.32.0
PROTOC_GEN_GO_GRPC_VERSION = 1.3.0

KIND_VERSION    = 0.22.0
KIND_NODE_IMAGE = "kindest/node:v1.25.3@sha256:cd248d1438192f7814fbca8fede13cfe5b9918746dfa12583976158a834fd5c5"
KIND_TMP_DIR    = "$(PWD)/bin/test/dex-kind-kubeconfig"


##@ Build

build: bin/dex ## Build Dex binaries.

examples: bin/grpc-client bin/example-app ## Build example app.

.PHONY: release-binary
release-binary: LD_FLAGS = "-w -X main.version=$(VERSION) -extldflags \"-static\""
release-binary: ## Build release binaries (used to build a final container image).
	@go build -o /go/bin/dex -v -ldflags $(LD_FLAGS) $(REPO_PATH)/cmd/dex
	@go build -o /go/bin/docker-entrypoint -v -ldflags $(LD_FLAGS) $(REPO_PATH)/cmd/docker-entrypoint

bin/dex:
	@mkdir -p bin/
	@go install -v -ldflags $(LD_FLAGS) $(REPO_PATH)/cmd/dex

bin/grpc-client:
	@mkdir -p bin/
	@cd examples/ && go install -v -ldflags $(LD_FLAGS) $(REPO_PATH)/examples/grpc-client

bin/example-app:
	@mkdir -p bin/
	@cd examples/ && go install -v -ldflags $(LD_FLAGS) $(REPO_PATH)/examples/example-app


##@ Generate

.PHONY: generate
generate: generate-proto generate-proto-internal generate-ent go-mod-tidy ## Run all generators.

.PHONY: generate-ent
generate-ent: ## Generate code for database ORM.
	@go generate $(REPO_PATH)/storage/ent/

.PHONY: generate-proto
generate-proto: ## Generate the Dex client's protobuf code.
	@protoc --go_out=paths=source_relative:. --go-grpc_out=paths=source_relative:. api/v2/*.proto
	@protoc --go_out=paths=source_relative:. --go-grpc_out=paths=source_relative:. api/*.proto

.PHONY: generate-proto-internal
generate-proto-internal: ## Generate protobuf code for token encoding.
	@protoc --go_out=paths=source_relative:. server/internal/*.proto

go-mod-tidy: ## Run go mod tidy for all targets.
	@go mod tidy
	@cd examples/ && go mod tidy
	@cd api/v2/ && go mod tidy

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

##@ Verify

verify: generate ## Verify that all the code was generated and committed to repository.
	@git diff --exit-code

.PHONY: verify-proto
verify-proto: generate-proto ## Verify that the Dex client's protobuf code was generated.
	@git diff --exit-code

.PHONY: verify-proto
verify-proto-internal: generate-proto-internal ## Verify internal protobuf code for token encoding was generated.
	@git diff --exit-code

.PHONY: verify-ent
verify-ent: generate-ent ## Verify code for database ORM was generated.
	@git diff --exit-code

.PHONY: verify-go-mod
verify-go-mod: go-mod-tidy ## Check that go.mod and go.sum formatted according to the changes.
	@git diff --exit-code

##@ Test and Lint

deps: bin/gotestsum bin/golangci-lint bin/protoc bin/protoc-gen-go bin/protoc-gen-go-grpc bin/kind ## Install dev dependencies.

.PHONY: test testrace testall
test: ## Test go code.
	@go test -v ./...

testrace: ## Test go code and check for possible race conditions.
	@go test -v --race ./...

testall: testrace ## Run all tests for go code.

.PHONY: lint lint-fix
lint: ## Run linter.
	@golangci-lint version
	@golangci-lint run

.PHONY: fix
fix: ## Fix lint violations.
	@golangci-lint version
	@golangci-lint run --fix

docker-compose.override.yaml:
	cp docker-compose.override.yaml.dist docker-compose.override.yaml

.PHONY: up
up: docker-compose.override.yaml ## Launch the development environment.
	@ if [ docker-compose.override.yaml -ot docker-compose.override.yaml.dist ]; then diff -u docker-compose.override.yaml docker-compose.override.yaml.dist || (echo "!!! The distributed docker-compose.override.yaml example changed. Please update your file accordingly (or at least touch it). !!!" && false); fi
	docker-compose up -d

.PHONY: down
down: clear ## Destroy the development environment.
	docker-compose down --volumes --remove-orphans --rmi local

.PHONY: kind-up kind-down kind-tests
kind-up: ## Create a kind cluster.
	@mkdir -p bin/test
	@kind create cluster --image ${KIND_NODE_IMAGE} --kubeconfig ${KIND_TMP_DIR} --name dex-tests

kind-tests: export DEX_KUBERNETES_CONFIG_PATH=${KIND_TMP_DIR}
kind-tests: testall ## Run test on kind cluster (kind cluster must be created).

kind-down: ## Delete the kind cluster.
	@kind delete cluster --name dex-tests
	rm ${KIND_TMP_DIR}

bin/golangci-lint:
	@mkdir -p bin
	curl -sSfL https://raw.githubusercontent.com/golangci/golangci-lint/master/install.sh | BINARY=golangci-lint bash -s -- v${GOLANGCI_VERSION}

bin/gotestsum:
	@mkdir -p bin
	curl -L https://github.com/gotestyourself/gotestsum/releases/download/v${GOTESTSUM_VERSION}/gotestsum_${GOTESTSUM_VERSION}_$(shell uname | tr A-Z a-z)_amd64.tar.gz | tar -zOxf - gotestsum > ./bin/gotestsum
	@chmod +x ./bin/gotestsum

bin/kind:
	@mkdir -p bin
	curl -L https://github.com/kubernetes-sigs/kind/releases/download/v${KIND_VERSION}/kind-$(shell uname | tr A-Z a-z)-amd64 > ./bin/kind
	@chmod +x ./bin/kind

##@ Clean
clean: ## Delete all builds and downloaded dependencies.
	@rm -rf bin/


FORMATTING_BEGIN_YELLOW = \033[0;33m
FORMATTING_BEGIN_BLUE = \033[36m
FORMATTING_END = \033[0m

.PHONY: help
help:
	@printf -- "${FORMATTING_BEGIN_BLUE}%s${FORMATTING_END}\n" \
	"" \
	"       ___             " \
	"      / _ \_____ __    " \
	"     / // / -_) \ /    " \
	"    /____/\__/_\_\     " \
	"" \
	"-----------------------" \
	""
	@awk 'BEGIN {\
	    FS = ":.*##"; \
	    printf                "Usage: ${FORMATTING_BEGIN_BLUE}OPTION${FORMATTING_END}=<value> make ${FORMATTING_BEGIN_YELLOW}<target>${FORMATTING_END}\n"\
	  } \
	  /^[a-zA-Z0-9_-]+:.*?##/ { printf "  ${FORMATTING_BEGIN_BLUE}%-46s${FORMATTING_END} %s\n", $$1, $$2 } \
	  /^.?.?##~/              { printf "   %-46s${FORMATTING_BEGIN_YELLOW}%-46s${FORMATTING_END}\n", "", substr($$1, 6) } \
	  /^##@/                  { printf "\n\033[1m%s\033[0m\n", substr($$0, 5) } ' $(MAKEFILE_LIST)
