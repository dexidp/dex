package migrations

// To download go-bindata run `go get -u github.com/jteeuwen/go-bindata/...`

//go:generate go-bindata -modtime=1 -pkg migrations -o assets.go -ignore \.go$ -prefix "../.." ../../db/migrations
//go:generate gofmt -w assets.go
