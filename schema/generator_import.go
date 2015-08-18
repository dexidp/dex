package schema

// This file exists to ensure Godep manages a vendored copy of the
// `google-api-go-generator` library, used by the 'generate' script.
// Unfortunately since this is a binary package and hence is not importable, we
// need to trick godep into managing it. To update the dependency, do the following steps:
// 1. Uncomment the import line below
// 2. Update the package in GOPATH as appropriate (e.g. `go get -u google.golang.org/api/google-api-go-generator`)
// 3. Run `godep save` as usual across the entire project (e.g. `godep save ./...`)
// 4. Revert this file (i.e. comment the line again, and revert to the original import) as it may not build properly

// import _ "google.golang.org/api/google-api-go-generator"
