//go:build cgo
// +build cgo

package sql

import (
	"context"
	"database/sql"
	"log/slog"
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

func TestGetPasswordWithTextGroups(t *testing.T) {
	logger := slog.New(slog.DiscardHandler)
	conn, err := (&SQLite3{File: ":memory:"}).open(logger)
	if err != nil {
		t.Fatal(err)
	}
	defer conn.Close()

	email := "text-groups@example.com"
	_, err = conn.Exec(`
		insert into password (
			email, hash, username, preferred_username, user_id, groups, name, email_verified
		)
		values (
			$1, $2, $3, $4, $5, CAST('[]' AS TEXT), $6, $7
		);`,
		email, []byte("hash"), "username", "", "user-id", "", false,
	)
	if err != nil {
		t.Fatal(err)
	}

	var storageClass string
	if err := conn.QueryRow(`select typeof(groups) from password where email = $1;`, email).Scan(&storageClass); err != nil {
		t.Fatal(err)
	}
	if storageClass != "text" {
		t.Fatalf("expected groups storage class text, got %q", storageClass)
	}

	p, err := conn.GetPassword(context.Background(), email)
	if err != nil {
		t.Fatal(err)
	}
	if len(p.Groups) != 0 {
		t.Fatalf("expected empty groups, got %q", p.Groups)
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
