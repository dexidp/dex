devsetup:
	go get "github.com/kisielk/errcheck"
	go get "golang.org/x/lint/golint"
	go get "github.com/gordonklaus/ineffassign"
	go get "github.com/client9/misspell/cmd/misspell"

test:
	go test ./ ./cbft
fasttest:
	go test -short ./ ./cbft

cover:
	go test -coverprofile=cover.out ./ ./cbft

checkerrs:
	errcheck -blank -asserts -ignoretests ./ ./cbft

checkfmt:
	! gofmt -l -d ./ ./cbft 2>&1 | read

checkvet:
	go vet

checkiea:
	ineffassign ./
	ineffassign ./cbft

checkspell:
	misspell -error ./
	misspell -error ./cbft

lint: checkfmt checkerrs checkvet checkiea checkspell
	golint -set_exit_status -min_confidence 0.81 ./
	golint -set_exit_status -min_confidence 0.81 ./cbft

check: lint
	go test -cover -race ./ ./cbft

.PHONY: all test devsetup fasttest lint cover checkerrs checkfmt checkvet checkiea checkspell check
