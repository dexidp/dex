package sql

import (
	"database/sql"
	"testing"
)

func TestMigrate(t *testing.T) {
	db, err := sql.Open("sqlite3", ":memory:")
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()

	c := &conn{db, flavorSQLite3}
	for _, want := range []int{len(migrations), 0} {
		got, err := c.migrate()
		if err != nil {
			t.Fatal(err)
		}
		if got != want {
			t.Errorf("expected %d migrations, got %d", want, got)
		}
	}
}
