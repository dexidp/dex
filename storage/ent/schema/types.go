package schema

import (
	"github.com/facebook/ent/dialect"
)

var textSchema = map[string]string{
	dialect.SQLite: "text",
}
