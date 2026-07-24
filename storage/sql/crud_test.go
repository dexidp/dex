//go:build cgo
// +build cgo

package sql

import (
	"database/sql"
	"reflect"
	"testing"
)

func TestDecoder(t *testing.T) {
	db, err := sql.Open("sqlite3", ":memory:")
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()

	if _, err := db.Exec(`create table foo ( id integer primary key, bar blob );`); err != nil {
		t.Fatal(err)
	}
	if _, err := db.Exec(`insert into foo ( id, bar ) values (1, ?);`, []byte(`["a", "b"]`)); err != nil {
		t.Fatal(err)
	}
	var got []string
	if err := db.QueryRow(`select bar from foo where id = 1;`).Scan(decoder(&got)); err != nil {
		t.Fatal(err)
	}
	want := []string{"a", "b"}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("wanted %q got %q", want, got)
	}
}

func TestEncoder(t *testing.T) {
	db, err := sql.Open("sqlite3", ":memory:")
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()

	if _, err := db.Exec(`create table foo ( id integer primary key, bar blob );`); err != nil {
		t.Fatal(err)
	}
	put := []string{"a", "b"}
	if _, err := db.Exec(`insert into foo ( id, bar ) values (1, ?)`, encoder(put)); err != nil {
		t.Fatal(err)
	}

	var got []byte
	if err := db.QueryRow(`select bar from foo where id = 1;`).Scan(&got); err != nil {
		t.Fatal(err)
	}
	want := []byte(`["a","b"]`)
	if !reflect.DeepEqual(got, want) {
		t.Errorf("wanted %q got %q", want, got)
	}
}

func TestScanConnectorNullExpiry(t *testing.T) {
	db, err := sql.Open("sqlite3", ":memory:")
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()

	if _, err := db.Exec(`create table connector ( id text, type text, name text, resource_version text, config blob, grant_types blob, expiry blob );`); err != nil {
		t.Fatal(err)
	}
	// Dex always writes SQL NULL for an unset override, but an out-of-band
	// writer can leave the JSON literal null; both must scan to a nil Expiry.
	if _, err := db.Exec(`insert into connector values ('a', 'oidc', 'A', '1', '{}', '[]', null), ('b', 'oidc', 'B', '1', '{}', '[]', 'null');`); err != nil {
		t.Fatal(err)
	}
	for _, id := range []string{"a", "b"} {
		c, err := scanConnector(db.QueryRow(`select id, type, name, resource_version, config, grant_types, expiry from connector where id = ?;`, id))
		if err != nil {
			t.Fatal(err)
		}
		if c.Expiry != nil {
			t.Errorf("connector %q: want nil expiry, got %+v", id, c.Expiry)
		}
	}
}
