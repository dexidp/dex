package mongo

import (
	"context"

	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
	"go.mongodb.org/mongo-driver/x/bsonx"
)

func (c *mongoStorage) authReqCollection() *Collection {
	return &Collection{c.database.Collection("auth_request")}
}

func getUniqueIndex(key string) mongo.IndexModel {
	index := mongo.IndexModel{
		Keys:    bsonx.Doc{{Key: key, Value: bsonx.Int32(1)}},
		Options: &options.IndexOptions{},
	}

	index.Options.SetUnique(true)

	return index
}

func getExpirationIndex(key string, expirationAfterSeconds int32) mongo.IndexModel { //nolint:unparam
	indexExpire := mongo.IndexModel{
		Keys:    bsonx.Doc{{Key: key, Value: bsonx.Int32(1)}},
		Options: &options.IndexOptions{},
	}

	indexExpire.Options.SetExpireAfterSeconds(expirationAfterSeconds)

	return indexExpire
}

func (c *mongoStorage) initAuthRequestCollection(ctx context.Context) error {
	indexes := []mongo.IndexModel{}

	indexes = append(indexes, getUniqueIndex("id"))

	if !c.config.UseGCInsteadOfIndexes {
		indexes = append(indexes, getExpirationIndex("expiry", 0))
	}

	return c.authReqCollection().CreateIndexes(ctx, indexes...)
}

func (c *mongoStorage) authCodeCollection() *Collection {
	return &Collection{c.database.Collection("auth_code")}
}

func (c *mongoStorage) initAuthCodeCollection(ctx context.Context) error {
	indexes := []mongo.IndexModel{}

	indexes = append(indexes, getUniqueIndex("id"))

	if !c.config.UseGCInsteadOfIndexes {
		indexes = append(indexes, getExpirationIndex("expiry", 0))
	}

	return c.authCodeCollection().CreateIndexes(ctx, indexes...)
}

func (c *mongoStorage) refreshTokenCollection() *Collection {
	return &Collection{c.database.Collection("refresh_token")}
}

func (c *mongoStorage) initRefreshTokenCollection(ctx context.Context) error {
	return c.refreshTokenCollection().CreateIndexes(ctx, getUniqueIndex("id"))
}

func (c *mongoStorage) clientCollection() *Collection {
	return &Collection{c.database.Collection("client")}
}

func (c *mongoStorage) initClientCollection(ctx context.Context) error {
	return c.clientCollection().CreateIndexes(ctx, getUniqueIndex("id"))
}

func (c *mongoStorage) passwordCollection() *Collection {
	return &Collection{c.database.Collection("password")}
}

func (c *mongoStorage) initPasswordCollection(ctx context.Context) error {
	return c.passwordCollection().CreateIndexes(ctx, getUniqueIndex("email"))
}

func (c *mongoStorage) offlineSessionCollection() *Collection {
	return &Collection{c.database.Collection("offline_session")}
}

func (c *mongoStorage) initOfflineSessionCollection(ctx context.Context) error {
	indexUserConnUnique := mongo.IndexModel{
		Keys:    bsonx.Doc{{Key: "user_id", Value: bsonx.Int32(1)}, {Key: "conn_id", Value: bsonx.Int32(1)}},
		Options: &options.IndexOptions{},
	}

	indexUserConnUnique.Options.SetUnique(true)

	return c.offlineSessionCollection().CreateIndexes(ctx, indexUserConnUnique)
}

func (c *mongoStorage) connectorCollection() *Collection {
	return &Collection{c.database.Collection("connector")}
}

func (c *mongoStorage) keysCollection() *Collection {
	return &Collection{c.database.Collection("openid_connect_keys")}
}

func (c *mongoStorage) deviceRequestCollection() *Collection {
	return &Collection{c.database.Collection("device_request")}
}

func (c *mongoStorage) initDeviceRequestCollection(ctx context.Context) error {
	indexes := []mongo.IndexModel{}

	indexUserDeviceCodeUnique := mongo.IndexModel{
		Keys:    bsonx.Doc{{Key: "user_code", Value: bsonx.Int32(1)}, {Key: "device_code", Value: bsonx.Int32(1)}},
		Options: &options.IndexOptions{},
	}

	indexUserDeviceCodeUnique.Options.SetUnique(true)

	indexes = append(indexes, indexUserDeviceCodeUnique)

	if !c.config.UseGCInsteadOfIndexes {
		indexes = append(indexes, getExpirationIndex("expiry", 0))
	}

	return c.deviceRequestCollection().CreateIndexes(ctx, indexes...)
}

func (c *mongoStorage) deviceTokenCollection() *Collection {
	return &Collection{c.database.Collection("device_token")}
}

func (c *mongoStorage) initDeviceTokenCollection(ctx context.Context) error {
	indexes := []mongo.IndexModel{}

	indexes = append(indexes, getUniqueIndex("device_code"))

	if !c.config.UseGCInsteadOfIndexes {
		indexes = append(indexes, getExpirationIndex("expiry", 0))
	}

	return c.deviceTokenCollection().CreateIndexes(ctx, indexes...)
}
