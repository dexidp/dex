package schema

// This file exists to ensure Godep manages a vendored copy of the
// `google-api-go-generator` library, used by the 'generate' script.
// Unfortunately since this is a binary package and hence is not importable, we
// need to trick godep into managing it by providing an import statement.

// This file is not expected to build.

import _ "google.golang.org/api/google-api-go-generator"
