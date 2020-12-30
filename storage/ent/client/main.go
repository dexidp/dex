package client

import (
	"context"
	"hash"
	"time"

	"github.com/dexidp/dex/storage"
	"github.com/dexidp/dex/storage/ent/db"
	"github.com/dexidp/dex/storage/ent/db/authcode"
	"github.com/dexidp/dex/storage/ent/db/authrequest"
	"github.com/dexidp/dex/storage/ent/db/devicerequest"
	"github.com/dexidp/dex/storage/ent/db/devicetoken"
	"github.com/dexidp/dex/storage/ent/db/migrate"
)

var _ storage.Storage = (*Database)(nil)

type Database struct {
	client *db.Client
	hasher func() hash.Hash
}

// NewDatabase returns new database client with set options.
func NewDatabase(opts ...func(*Database)) *Database {
	database := &Database{}
	for _, f := range opts {
		f(database)
	}
	return database
}

// WithClient sets client option of a Database object.
func WithClient(c *db.Client) func(*Database) {
	return func(s *Database) {
		s.client = c
	}
}

// WithHasher sets client option of a Database object.
func WithHasher(h func() hash.Hash) func(*Database) {
	return func(s *Database) {
		s.hasher = h
	}
}

// Schema exposes migration schema to perform migrations.
func (d *Database) Schema() *migrate.Schema {
	return d.client.Schema
}

// Close calls the corresponding method of the ent database client.
func (d *Database) Close() error {
	return d.client.Close()
}

// GarbageCollect removes expired entities from the database.
func (d *Database) GarbageCollect(now time.Time) (storage.GCResult, error) {
	result := storage.GCResult{}
	utcNow := now.UTC()

	q, err := d.client.AuthRequest.Delete().
		Where(authrequest.ExpiryLT(utcNow)).
		Exec(context.TODO())
	if err != nil {
		return result, convertDBError("gc auth request: %w", err)
	}
	result.AuthRequests = int64(q)

	q, err = d.client.AuthCode.Delete().
		Where(authcode.ExpiryLT(utcNow)).
		Exec(context.TODO())
	if err != nil {
		return result, convertDBError("gc auth code: %w", err)
	}
	result.AuthCodes = int64(q)

	q, err = d.client.DeviceRequest.Delete().
		Where(devicerequest.ExpiryLT(utcNow)).
		Exec(context.TODO())
	if err != nil {
		return result, convertDBError("gc device request: %w", err)
	}
	result.DeviceRequests = int64(q)

	q, err = d.client.DeviceToken.Delete().
		Where(devicetoken.ExpiryLT(utcNow)).
		Exec(context.TODO())
	if err != nil {
		return result, convertDBError("gc device token: %w", err)
	}
	result.DeviceTokens = int64(q)

	return result, err
}
