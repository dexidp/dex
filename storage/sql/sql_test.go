package sql

import "testing"

func TestTranslate(t *testing.T) {
	tests := []struct {
		testCase string
		flavor   flavor
		query    string
		exp      string
	}{
		{
			"sqlite3 query bind replacement",
			flavorSQLite3,
			`select foo from bar where foo.zam = $1;`,
			`select foo from bar where foo.zam = ?;`,
		},
		{
			"sqlite3 query bind replacement at newline",
			flavorSQLite3,
			`select foo from bar where foo.zam = $1`,
			`select foo from bar where foo.zam = ?`,
		},
		{
			"sqlite3 query true",
			flavorSQLite3,
			`select foo from bar where foo.zam = true`,
			`select foo from bar where foo.zam = 1`,
		},
		{
			"sqlite3 query false",
			flavorSQLite3,
			`select foo from bar where foo.zam = false`,
			`select foo from bar where foo.zam = 0`,
		},
		{
			"sqlite3 bytea",
			flavorSQLite3,
			`"connector_data" bytea not null,`,
			`"connector_data" blob not null,`,
		},
		{
			"sqlite3 now",
			flavorSQLite3,
			`now(),`,
			`date('now'),`,
		},
	}

	for _, tc := range tests {
		if got := tc.flavor.translate(tc.query); got != tc.exp {
			t.Errorf("%s: want=%q, got=%q", tc.testCase, tc.exp, got)
		}
	}
}
