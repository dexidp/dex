PROJ=poke
ORG_PATH=github.com/coreos
REPO_PATH=$(ORG_PATH)/$(PROJ)
export PATH := $(PWD)/bin:$(PATH)

export GOBIN=$(PWD)/bin
export GO15VENDOREXPERIMENT=1
export CGO_ENABLED:=0

LD_FLAGS="-w -X $(REPO_PATH)/version.Version=$(shell ./scripts/git-version)"

GOOS=$(shell go env GOOS)
GOARCH=$(shell go env GOARCH)

build: bin/poke bin/example-app

bin/poke: FORCE
	@go install -ldflags $(LD_FLAGS) $(REPO_PATH)/cmd/poke

bin/example-app: FORCE
	@go install -ldflags $(LD_FLAGS) $(REPO_PATH)/cmd/example-app

test:
	@go test $(shell go list ./... | grep -v '/vendor/')

testrace:
	@go test --race $(shell go list ./... | grep -v '/vendor/')

vet:
	@go vet $(shell go list ./... | grep -v '/vendor/')

fmt:
	@go fmt $(shell go list ./... | grep -v '/vendor/')

lint:
	@for package in $(shell go list ./... | grep -v '/vendor/' | grep -v 'api/apipb'); do \
      golint $$package; \
	done

clean:
	@rm bin/*

testall: testrace vet fmt lint

FORCE:

.PHONY: test testrace vet fmt lint testall
