OS = $(shell uname | tr A-Z a-z)

PROJ=dex
ORG_PATH=github.com/dexidp
REPO_PATH=$(ORG_PATH)/$(PROJ)
export PATH := $(PWD)/bin:$(PATH)

VERSION ?= $(shell ./scripts/git-version)

DOCKER_REPO=quay.io/dexidp/dex
DOCKER_IMAGE=$(DOCKER_REPO):$(VERSION)

$( shell mkdir -p bin )

user=$(shell id -u -n)
group=$(shell id -g -n)

export GOBIN=$(PWD)/bin

LD_FLAGS="-w -X main.version=$(VERSION)"

# Dependency versions
GOLANGCI_VERSION = 1.42.0

PROTOC_VERSION = 3.15.6
PROTOC_GEN_GO_VERSION = 1.26.0
PROTOC_GEN_GO_GRPC_VERSION = 1.1.0

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
release-binary: generate
	@go build -o /go/bin/dex -v -ldflags $(LD_FLAGS) $(REPO_PATH)/cmd/dex

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

bin/golangci-lint: bin/golangci-lint-${GOLANGCI_VERSION}
	@ln -sf golangci-lint-${GOLANGCI_VERSION} bin/golangci-lint
bin/golangci-lint-${GOLANGCI_VERSION}:
	curl -sfL https://install.goreleaser.com/github.com/golangci/golangci-lint.sh | BINARY=golangci-lint bash -s -- v${GOLANGCI_VERSION}
	@mv bin/golangci-lint $@

.PHONY: lint lint-fix
lint: bin/golangci-lint ## Run linter
	bin/golangci-lint run

.PHONY: fix
fix: bin/golangci-lint ## Fix lint violations
	bin/golangci-lint run --fix

.PHONY: docker-image
docker-image:
	@sudo docker build -t $(DOCKER_IMAGE) .

.PHONY: proto-old
proto-old: bin/protoc-old bin/protoc-gen-go-old
	@./bin/protoc-old --go_out=plugins=grpc:. --plugin=protoc-gen-go=./bin/protoc-gen-go-old api/v2/*.proto
	@cp api/v2/*.proto api/
	@./bin/protoc-old --go_out=plugins=grpc:. --plugin=protoc-gen-go=./bin/protoc-gen-go-old api/*.proto

.PHONY: verify-proto
verify-proto: proto
	@./scripts/git-diff

bin/protoc-old: scripts/get-protoc
	@./scripts/get-protoc bin/protoc-old

bin/protoc-gen-go-old:
	@mkdir -p tmp
	@GOBIN=$$PWD/tmp go install -v github.com/golang/protobuf/protoc-gen-go@v1.3.2
	@mv tmp/protoc-gen-go bin/protoc-gen-go-old

clean:
	@rm -rf bin/

testall: testrace

FORCE:

.PHONY: test testrace testall

.PHONY: proto
proto: bin/protoc bin/protoc-gen-go bin/protoc-gen-go-grpc
	@./bin/protoc --go_out=paths=source_relative:. --go-grpc_out=paths=source_relative:. --plugin=protoc-gen-go=./bin/protoc-gen-go --plugin=protoc-gen-go-grpc=./bin/protoc-gen-go-grpc api/v2/*.proto
	@./bin/protoc --go_out=paths=source_relative:. --go-grpc_out=paths=source_relative:. --plugin=protoc-gen-go=./bin/protoc-gen-go --plugin=protoc-gen-go-grpc=./bin/protoc-gen-go-grpc api/*.proto
	#@cp api/v2/*.proto api/

.PHONY: proto-internal
proto-internal: bin/protoc bin/protoc-gen-go
	@./bin/protoc --go_out=paths=source_relative:. --plugin=protoc-gen-go=./bin/protoc-gen-go server/internal/*.proto

bin/protoc: bin/protoc-${PROTOC_VERSION}
	@ln -sf protoc-${PROTOC_VERSION}/bin/protoc bin/protoc
bin/protoc-${PROTOC_VERSION}:
	@mkdir -p bin/protoc-${PROTOC_VERSION}
ifeq (${OS}, darwin)
	curl -L https://github.com/protocolbuffers/protobuf/releases/download/v${PROTOC_VERSION}/protoc-${PROTOC_VERSION}-osx-x86_64.zip > bin/protoc.zip
endif
ifeq (${OS}, linux)
	curl -L https://github.com/protocolbuffers/protobuf/releases/download/v${PROTOC_VERSION}/protoc-${PROTOC_VERSION}-linux-x86_64.zip > bin/protoc.zip
endif
	unzip bin/protoc.zip -d bin/protoc-${PROTOC_VERSION}
	rm bin/protoc.zip

bin/protoc-gen-go: bin/protoc-gen-go-${PROTOC_GEN_GO_VERSION}
	@ln -sf protoc-gen-go-${PROTOC_GEN_GO_VERSION} bin/protoc-gen-go
bin/protoc-gen-go-${PROTOC_GEN_GO_VERSION}:
	@mkdir -p bin
	curl -L https://github.com/protocolbuffers/protobuf-go/releases/download/v${PROTOC_GEN_GO_VERSION}/protoc-gen-go.v${PROTOC_GEN_GO_VERSION}.${OS}.amd64.tar.gz | tar -zOxf - protoc-gen-go > ./bin/protoc-gen-go-${PROTOC_GEN_GO_VERSION} && chmod +x ./bin/protoc-gen-go-${PROTOC_GEN_GO_VERSION}

bin/protoc-gen-go-grpc: bin/protoc-gen-go-grpc-${PROTOC_GEN_GO_GRPC_VERSION}
	@ln -sf protoc-gen-go-grpc-${PROTOC_GEN_GO_GRPC_VERSION} bin/protoc-gen-go-grpc
bin/protoc-gen-go-grpc-${PROTOC_GEN_GO_GRPC_VERSION}:
	@mkdir -p bin
	curl -L https://github.com/grpc/grpc-go/releases/download/cmd%2Fprotoc-gen-go-grpc%2Fv${PROTOC_GEN_GO_GRPC_VERSION}/protoc-gen-go-grpc.v${PROTOC_GEN_GO_GRPC_VERSION}.${OS}.amd64.tar.gz | tar -zOxf - ./protoc-gen-go-grpc > ./bin/protoc-gen-go-grpc-${PROTOC_GEN_GO_GRPC_VERSION} && chmod +x ./bin/protoc-gen-go-grpc-${PROTOC_GEN_GO_GRPC_VERSION}
