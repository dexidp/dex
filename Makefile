PROJ=dex
ORG_PATH=github.com/dexidp
REPO_PATH=$(ORG_PATH)/$(PROJ)
export PATH := $(PWD)/bin:$(PATH)

VERSION ?= $(shell ./scripts/git-version)

DOCKER_REPO=quay.io/giantswarm/dex
DOCKER_IMAGE=$(DOCKER_REPO):$(VERSION)

$( shell mkdir -p bin )

user=$(shell id -u -n)
group=$(shell id -g -n)

export GOBIN=$(PWD)/bin

LD_FLAGS="-w -X $(REPO_PATH)/version.Version=$(VERSION)"

# Dependency versions
GOLANGCI_VERSION = 1.32.2

build: bin/dex

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
release-binary:
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

test: bin/test/kube-apiserver bin/test/etcd
	@go test -v ./...

testrace: bin/test/kube-apiserver bin/test/etcd
	@go test -v --race ./...

export TEST_ASSET_KUBE_APISERVER=$(abspath bin/test/kube-apiserver)
export TEST_ASSET_ETCD=$(abspath bin/test/etcd)

bin/test/kube-apiserver:
	@mkdir -p bin/test
	curl -L https://storage.googleapis.com/k8s-c10s-test-binaries/kube-apiserver-$(shell uname)-x86_64 > bin/test/kube-apiserver
	chmod +x bin/test/kube-apiserver

bin/test/etcd:
	@mkdir -p bin/test
	curl -L https://storage.googleapis.com/k8s-c10s-test-binaries/etcd-$(shell uname)-x86_64 > bin/test/etcd
	chmod +x bin/test/etcd

bin/golangci-lint: bin/golangci-lint-${GOLANGCI_VERSION}
	@ln -sf golangci-lint-${GOLANGCI_VERSION} bin/golangci-lint
bin/golangci-lint-${GOLANGCI_VERSION}:
	curl -sfL https://install.goreleaser.com/github.com/golangci/golangci-lint.sh | BINARY=golangci-lint bash -s -- v${GOLANGCI_VERSION}
	@mv bin/golangci-lint $@

.PHONY: lint
lint: bin/golangci-lint ## Run linter
	bin/golangci-lint run

.PHONY: fix
fix: bin/golangci-lint ## Fix lint violations
	bin/golangci-lint run --fix

.PHONY: docker-image
docker-image:
	@sudo docker build -t $(DOCKER_IMAGE) .

.PHONY: proto
proto: bin/protoc bin/protoc-gen-go
	@./bin/protoc --go_out=plugins=grpc:. --plugin=protoc-gen-go=./bin/protoc-gen-go api/v2/*.proto
	@cp api/v2/*.proto api/
	@./bin/protoc --go_out=plugins=grpc:. --plugin=protoc-gen-go=./bin/protoc-gen-go api/*.proto
	@./bin/protoc --go_out=. --plugin=protoc-gen-go=./bin/protoc-gen-go server/internal/*.proto

.PHONY: verify-proto
verify-proto: proto
	@./scripts/git-diff

bin/protoc: scripts/get-protoc
	@./scripts/get-protoc bin/protoc

bin/protoc-gen-go:
	@go install -v github.com/golang/protobuf/protoc-gen-go

clean:
	@rm -rf bin/

testall: testrace

FORCE:

.PHONY: test testrace testall
