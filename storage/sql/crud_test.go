//go:build cgo
// +build cgo

package sql

import (
	"context"
	"database/sql"
	"log/slog"
	"reflect"
	"sort"
	"testing"

	"github.com/go-jose/go-jose/v4/json"
	"github.com/stretchr/testify/require"

	"github.com/dexidp/dex/storage"
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

func Test_conn_ListClients(t *testing.T) {
	type fields struct {
		db                 *sql.DB
		flavor             *flavor
		logger             *slog.Logger
		alreadyExistsCheck func(err error) bool
	}
	type args struct {
		ctx context.Context
	}
	tests := []struct {
		name    string
		fields  fields
		args    args
		setup   func(db *sql.DB)
		want    []storage.Client
		wantErr bool
	}{
		{
			name: "empty database",
			fields: fields{
				flavor:             &flavorSQLite3,
				logger:             slog.Default(),
				alreadyExistsCheck: func(err error) bool { return false },
			},
			args: args{
				ctx: context.Background(),
			},
			setup: func(db *sql.DB) {
				// Create table with BLOB for JSON fields
				_, err := db.Exec(`
                    CREATE TABLE client (
                        id TEXT PRIMARY KEY,
                        secret TEXT,
                        redirect_uris BLOB,
                        trusted_peers BLOB,
                        public INTEGER,
                        name TEXT,
                        logo_url TEXT
                    );
                `)
				require.NoError(t, err)
			},
			want: nil,
		},
		{
			name: "multiple clients",
			fields: fields{
				flavor:             &flavorSQLite3,
				logger:             slog.Default(),
				alreadyExistsCheck: func(err error) bool { return false },
			},
			args: args{
				ctx: context.Background(),
			},
			setup: func(db *sql.DB) {
				// Create table with BLOB for JSON fields
				_, err := db.Exec(`
                    CREATE TABLE client (
                        id TEXT PRIMARY KEY,
                        secret TEXT,
                        redirect_uris BLOB,
                        trusted_peers BLOB,
                        public INTEGER,
                        name TEXT,
                        logo_url TEXT
                    );
                `)
				require.NoError(t, err)

				// Insert data with []byte JSON
				redirect1, _ := json.Marshal([]string{"uri1"})
				trusted1, _ := json.Marshal([]string{"peer1"})
				_, err = db.Exec(`
                    INSERT INTO client (id, secret, redirect_uris, trusted_peers, public, name, logo_url)
                    VALUES ('client1', 'secret1', ?, ?, 1, 'name1', 'logo1');
                `, redirect1, trusted1)
				require.NoError(t, err)

				redirect2, _ := json.Marshal([]string{"uri2"})
				trusted2, _ := json.Marshal([]string{"peer2"})
				_, err = db.Exec(`
                    INSERT INTO client (id, secret, redirect_uris, trusted_peers, public, name, logo_url)
                    VALUES ('client2', 'secret2', ?, ?, 0, 'name2', 'logo2');
                `, redirect2, trusted2)
				require.NoError(t, err)
			},
			want: []storage.Client{
				{ID: "client1", Secret: "secret1", RedirectURIs: []string{"uri1"}, TrustedPeers: []string{"peer1"}, Public: true, Name: "name1", LogoURL: "logo1"},
				{ID: "client2", Secret: "secret2", RedirectURIs: []string{"uri2"}, TrustedPeers: []string{"peer2"}, Public: false, Name: "name2", LogoURL: "logo2"},
			},
		},
		{
			name: "multiple broken clients",
			fields: fields{
				flavor:             &flavorSQLite3,
				logger:             slog.Default(),
				alreadyExistsCheck: func(err error) bool { return false },
			},
			args: args{
				ctx: context.Background(),
			},
			setup: func(db *sql.DB) {
				// Create table with BLOB for JSON fields
				_, err := db.Exec(`
                    CREATE TABLE client (
                        id TEXT PRIMARY KEY,
                        secret TEXT,
                        redirect_uris BLOB,
                        trusted_peers BLOB,
                        public INTEGER,
                        name TEXT,
                        logo_url TEXT
                    );
                `)
				require.NoError(t, err)

				// Insert data with []byte JSON
				redirect1, _ := json.Marshal([]string{"uri1"})
				trusted1, _ := json.Marshal([]string{"peer1"})
				_, err = db.Exec(`
                    INSERT INTO client (id, secret, redirect_uris, trusted_peers, public, name, logo_url)
                    VALUES ('client1', 'secret1', ?, ?, 1, 'name1', 'logo1');
                `, string(redirect1), string(trusted1))
				require.NoError(t, err)

				redirect2, _ := json.Marshal([]string{"uri2"})
				trusted2, _ := json.Marshal([]string{"peer2"})
				_, err = db.Exec(`
                    INSERT INTO client (id, secret, redirect_uris, trusted_peers, public, name, logo_url)
                    VALUES ('client2', 'secret2', ?, ?, 0, 'name2', 'logo2');
                `, string(redirect2), string(trusted2))
				require.NoError(t, err)
			},
			want:    nil,
			wantErr: true, // Expect error due to broken JSON
		},
		{
			name: "query error",
			fields: fields{
				flavor:             &flavorSQLite3,
				logger:             slog.Default(),
				alreadyExistsCheck: func(err error) bool { return false },
			},
			args: args{
				ctx: context.Background(),
			},
			setup: func(db *sql.DB) {
				db.Close() // Cause query to fail
			},
			wantErr: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			db, err := sql.Open("sqlite3", ":memory:")
			require.NoError(t, err)
			defer db.Close()

			tt.fields.db = db
			if tt.setup != nil {
				tt.setup(db)
			}

			c := &conn{
				db:                 tt.fields.db,
				flavor:             tt.fields.flavor,
				logger:             tt.fields.logger,
				alreadyExistsCheck: tt.fields.alreadyExistsCheck,
			}
			got, err := c.ListClients(tt.args.ctx)
			if tt.wantErr {
				require.Error(t, err)
				return
			}
			require.NoError(t, err)
			// Sort for consistent order
			sort.Slice(got, func(i, j int) bool { return got[i].ID < got[j].ID })
			sort.Slice(tt.want, func(i, j int) bool { return tt.want[i].ID < tt.want[j].ID })
			require.Equal(t, tt.want, got)
		})
	}
}
