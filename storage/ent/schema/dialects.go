package schema

import (
	"entgo.io/ent/dialect"
)

var textSchema = map[string]string{
	dialect.Postgres: "text",
	dialect.SQLite:   "text",
	// MySQL doesn't support indices on text fields w/o
	// specifying key length. Use varchar instead (767 byte
	// is the max key length for InnoDB with 4k pages).
	// For compound indexes (with two keys) even less.
	dialect.MySQL: "varchar(384)",
}

var timeSchema = map[string]string{
	dialect.Postgres: "timestamptz",
	dialect.SQLite:   "timestamp",
	dialect.MySQL:    "datetime(3)",
}
