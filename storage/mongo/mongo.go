package mongo

import (
	"context"
	"strings"
	"time"

	"github.com/dexidp/dex/pkg/log"
	"github.com/dexidp/dex/storage"
	"github.com/pkg/errors"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
)

var (
	emptyFilter = func() bson.D {
		return bson.D{}
	}
	filterMatchID = func(id string) bson.D {
		return bson.D{{Key: "id", Value: id}}
	}
	filterMatch_ID = func(id string) bson.D { //nolint:golint,stylecheck // The underscore is needed to distinguish between mongo and DEX id format
		return bson.D{{Key: "_id", Value: id}}
	}
	filterMatchEmail = func(email string) bson.D {
		return bson.D{{Key: "email", Value: email}}
	}
	filterMatchUserIDConnID = func(userID, connID string) bson.D {
		return bson.D{{Key: "user_id", Value: userID}, {Key: "conn_id", Value: connID}}
	}
	filterMatchUserCode = func(userCode string) bson.D {
		return bson.D{{Key: "user_code", Value: userCode}}
	}
	filterMatchDeviceCode = func(deviceCode string) bson.D {
		return bson.D{{Key: "device_code", Value: deviceCode}}
	}
	filterNotExpired = func(t time.Time) bson.D {
		return bson.D{{Key: "expiry", Value: bson.D{{Key: "$lte", Value: t}}}}
	}
)

func keyEmail(email string) string { return strings.ToLower(email) }

type mongoStorage struct {
	session  *Session
	database *mongo.Database

	config Mongo

	logger log.Logger
}

func (c *mongoStorage) Close() error {
	ctx, cancel := context.WithTimeout(context.Background(), c.config.DatabaseTimeout)

	defer cancel()

	c.session.End(ctx)

	return nil
}

func (c *mongoStorage) GarbageCollect(t time.Time) (storage.GCResult, error) {
	if !c.config.UseGCInsteadOfIndexes {
		//HINT: let mongodb use expire indexes
		return storage.GCResult{}, nil
	}

	ctx, cancel := context.WithTimeout(context.Background(), c.config.DatabaseTimeout)
	defer cancel()

	var gcResult storage.GCResult

	if err := c.session.Operate(ctx, func(ctx context.Context) error {
		deletedAuthCodeDocuments, err := c.authCodeCollection().DeleteMany(ctx, filterNotExpired(t))
		if err != nil && !errors.Is(err, storage.ErrNotFound) {
			return errors.Wrap(err, "unable to delete auth code documents")
		}

		deletedAuthRequestDocuments, err := c.authReqCollection().DeleteMany(ctx, filterNotExpired(t))
		if err != nil && !errors.Is(err, storage.ErrNotFound) {
			return errors.Wrap(err, "unable to delete auth code documents")
		}

		deletedDeviceRequestDocuments, err := c.deviceRequestCollection().DeleteMany(ctx, filterNotExpired(t))
		if err != nil && !errors.Is(err, storage.ErrNotFound) {
			return errors.Wrap(err, "unable to delete auth code documents")
		}

		deletedDeviceTokenDocuments, err := c.deviceTokenCollection().DeleteMany(ctx, filterNotExpired(t))
		if err != nil && !errors.Is(err, storage.ErrNotFound) {
			return errors.Wrap(err, "unable to delete auth code documents")
		}

		gcResult = storage.GCResult{
			AuthRequests:   deletedAuthRequestDocuments,
			AuthCodes:      deletedAuthCodeDocuments,
			DeviceRequests: deletedDeviceRequestDocuments,
			DeviceTokens:   deletedDeviceTokenDocuments,
		}

		return nil
	}); err != nil {
		return storage.GCResult{}, err
	}

	return gcResult, nil
}

func (c *mongoStorage) CreateAuthRequest(a storage.AuthRequest) error {
	ctx, cancel := context.WithTimeout(context.Background(), c.config.DatabaseTimeout)
	defer cancel()

	if err := c.session.Operate(ctx, func(ctx context.Context) error {
		if err := c.authReqCollection().Insert(ctx, fromStorageAuthRequest(a)); err != nil {
			return err
		}

		return nil
	}); err != nil {
		return err
	}

	return nil
}

func (c *mongoStorage) GetAuthRequest(id string) (storage.AuthRequest, error) {
	ctx, cancel := context.WithTimeout(context.Background(), c.config.DatabaseTimeout)
	defer cancel()

	var authRequest AuthRequest

	if err := c.session.Operate(ctx, func(ctx context.Context) error {
		if err := c.authReqCollection().FindOne(ctx, &authRequest, filterMatchID(id)); err != nil {
			return err
		}

		return nil
	}); err != nil {
		return storage.AuthRequest{}, err
	}

	return toStorageAuthRequest(authRequest), nil
}

func (c *mongoStorage) UpdateAuthRequest(id string, updater func(a storage.AuthRequest) (storage.AuthRequest, error)) error {
	ctx, cancel := context.WithTimeout(context.Background(), c.config.DatabaseTimeout)
	defer cancel()

	if err := c.session.Operate(ctx, func(ctx context.Context) error {
		var current AuthRequest
		if err := c.authReqCollection().FindOne(ctx, &current, filterMatchID(id)); err != nil {
			return err
		}

		updated, err := updater(toStorageAuthRequest(current))
		if err != nil {
			return err
		}

		if err := c.authReqCollection().ReplaceOne(ctx, filterMatchID(id), fromStorageAuthRequest(updated)); err != nil {
			return err
		}

		return nil
	}); err != nil {
		return err
	}

	return nil
}

func (c *mongoStorage) DeleteAuthRequest(id string) error {
	ctx, cancel := context.WithTimeout(context.Background(), c.config.DatabaseTimeout)
	defer cancel()

	return c.session.Operate(ctx, func(ctx context.Context) error {
		return c.authReqCollection().DeleteOne(ctx, filterMatchID(id))
	})
}

func (c *mongoStorage) CreateAuthCode(a storage.AuthCode) error {
	ctx, cancel := context.WithTimeout(context.Background(), c.config.DatabaseTimeout)
	defer cancel()

	authCode := fromStorageAuthCode(a)

	c.logger.Debug("[mongo.go] CreateAuthCode: ", authCode)

	return c.session.Operate(ctx, func(ctx context.Context) error {
		return c.authCodeCollection().Insert(ctx, authCode)
	})
}

func (c *mongoStorage) GetAuthCode(id string) (storage.AuthCode, error) {
	ctx, cancel := context.WithTimeout(context.Background(), c.config.DatabaseTimeout)
	defer cancel()

	var authCode AuthCode

	if err := c.session.Operate(ctx, func(ctx context.Context) error {
		return c.authCodeCollection().FindOne(ctx, &authCode, filterMatchID(id))
	}); err != nil {
		return storage.AuthCode{}, err
	}

	return toStorageAuthCode(authCode), nil
}

func (c *mongoStorage) DeleteAuthCode(id string) error {
	ctx, cancel := context.WithTimeout(context.Background(), c.config.DatabaseTimeout)
	defer cancel()

	return c.session.Operate(ctx, func(ctx context.Context) error {
		return c.authCodeCollection().DeleteOne(ctx, filterMatchID(id))
	})
}

func (c *mongoStorage) CreateRefresh(r storage.RefreshToken) error {
	ctx, cancel := context.WithTimeout(context.Background(), c.config.DatabaseTimeout)
	defer cancel()

	return c.session.Operate(ctx, func(ctx context.Context) error {
		return c.refreshTokenCollection().Insert(ctx, fromStorageRefreshToken(r))
	})
}

func (c *mongoStorage) GetRefresh(id string) (storage.RefreshToken, error) {
	ctx, cancel := context.WithTimeout(context.Background(), c.config.DatabaseTimeout)
	defer cancel()

	var token RefreshToken

	if err := c.session.Operate(ctx, func(ctx context.Context) error {
		return c.refreshTokenCollection().FindOne(ctx, &token, filterMatchID(id))
	}); err != nil {
		return storage.RefreshToken{}, err
	}

	return toStorageRefreshToken(token), nil
}

func (c *mongoStorage) UpdateRefreshToken(id string, updater func(old storage.RefreshToken) (storage.RefreshToken, error)) error {
	ctx, cancel := context.WithTimeout(context.Background(), c.config.DatabaseTimeout)
	defer cancel()

	return c.session.Operate(ctx, func(ctx context.Context) error {
		var token RefreshToken

		err := c.refreshTokenCollection().FindOne(ctx, &token, filterMatchID(id))
		if err != nil {
			return err
		}

		updated, err := updater(toStorageRefreshToken(token))
		if err != nil {
			return err
		}

		if err := c.refreshTokenCollection().ReplaceOne(ctx, filterMatchID(id), fromStorageRefreshToken(updated)); err != nil {
			return err
		}

		return nil
	})
}

func (c *mongoStorage) DeleteRefresh(id string) error {
	ctx, cancel := context.WithTimeout(context.Background(), c.config.DatabaseTimeout)
	defer cancel()

	return c.session.Operate(ctx, func(ctx context.Context) error {
		return c.refreshTokenCollection().DeleteOne(ctx, filterMatchID(id))
	})
}

func (c *mongoStorage) ListRefreshTokens() ([]storage.RefreshToken, error) {
	ctx, cancel := context.WithTimeout(context.Background(), c.config.DatabaseTimeout)
	defer cancel()

	tokens := []storage.RefreshToken{}

	if err := c.session.Operate(ctx, func(ctx context.Context) error {
		refreshTokens := []RefreshToken{}
		if err := c.refreshTokenCollection().Find(ctx, &refreshTokens, emptyFilter()); err != nil {
			return err
		}

		for _, refreshToken := range refreshTokens {
			tokens = append(tokens, toStorageRefreshToken(refreshToken))
		}

		return nil
	}); err != nil {
		return nil, err
	}

	return tokens, nil
}

func (c *mongoStorage) CreateClient(cli storage.Client) error {
	ctx, cancel := context.WithTimeout(context.Background(), c.config.DatabaseTimeout)
	defer cancel()

	return c.session.Operate(ctx, func(ctx context.Context) error {
		return c.clientCollection().Insert(ctx, cli)
	})
}

func (c *mongoStorage) GetClient(id string) (storage.Client, error) {
	ctx, cancel := context.WithTimeout(context.Background(), c.config.DatabaseTimeout)
	defer cancel()

	var cli storage.Client

	if err := c.session.Operate(ctx, func(ctx context.Context) error {
		return c.clientCollection().FindOne(ctx, &cli, filterMatchID(id))
	}); err != nil {
		return storage.Client{}, err
	}

	return cli, nil
}

func (c *mongoStorage) UpdateClient(id string, updater func(old storage.Client) (storage.Client, error)) error {
	ctx, cancel := context.WithTimeout(context.Background(), c.config.DatabaseTimeout)
	defer cancel()

	if err := c.session.Operate(ctx, func(ctx context.Context) error {
		var current storage.Client
		if err := c.clientCollection().FindOne(ctx, &current, filterMatchID(id)); err != nil {
			return err
		}

		updated, err := updater(current)
		if err != nil {
			return err
		}

		if err := c.clientCollection().ReplaceOne(ctx, filterMatchID(id), updated); err != nil {
			return err
		}
		return nil
	}); err != nil {
		return err
	}

	return nil
}

func (c *mongoStorage) DeleteClient(id string) error {
	ctx, cancel := context.WithTimeout(context.Background(), c.config.DatabaseTimeout)
	defer cancel()

	if err := c.clientCollection().DeleteOne(ctx, filterMatchID(id)); err != nil {
		return err
	}
	return nil
}

func (c *mongoStorage) ListClients() ([]storage.Client, error) {
	ctx, cancel := context.WithTimeout(context.Background(), c.config.DatabaseTimeout)
	defer cancel()

	var clients []storage.Client

	if err := c.clientCollection().Find(ctx, &clients, emptyFilter()); err != nil {
		return nil, err
	}

	return clients, nil
}

func (c *mongoStorage) CreatePassword(p storage.Password) error {
	ctx, cancel := context.WithTimeout(context.Background(), c.config.DatabaseTimeout)
	defer cancel()

	p.Email = keyEmail(p.Email)

	if err := c.session.Operate(ctx, func(ctx context.Context) error {
		if err := c.passwordCollection().Insert(ctx, p); err != nil {
			return err
		}
		return nil
	}); err != nil {
		return err
	}

	return nil
}

func (c *mongoStorage) GetPassword(email string) (storage.Password, error) {
	ctx, cancel := context.WithTimeout(context.Background(), c.config.DatabaseTimeout)
	defer cancel()

	var password storage.Password

	if err := c.session.Operate(ctx, func(ctx context.Context) error {
		return c.passwordCollection().FindOne(ctx, &password, filterMatchEmail(keyEmail(email)))
	}); err != nil {
		return storage.Password{}, err
	}

	return password, nil
}

func (c *mongoStorage) UpdatePassword(email string, updater func(p storage.Password) (storage.Password, error)) error {
	ctx, cancel := context.WithTimeout(context.Background(), c.config.DatabaseTimeout)
	defer cancel()

	return c.session.Operate(ctx, func(ctx context.Context) error {
		var current storage.Password
		if err := c.passwordCollection().FindOne(ctx, &current, filterMatchEmail(keyEmail(email))); err != nil {
			return err
		}

		updated, err := updater(current)
		if err != nil {
			return err
		}

		updated.Email = keyEmail(updated.Email)

		if err := c.passwordCollection().ReplaceOne(ctx, filterMatchEmail(keyEmail(email)), updated); err != nil {
			return err
		}

		return nil
	})
}

func (c *mongoStorage) DeletePassword(email string) error {
	ctx, cancel := context.WithTimeout(context.Background(), c.config.DatabaseTimeout)
	defer cancel()

	return c.session.Operate(ctx, func(ctx context.Context) error {
		return c.passwordCollection().DeleteOne(ctx, filterMatchEmail(keyEmail(email)))
	})
}

func (c *mongoStorage) ListPasswords() ([]storage.Password, error) {
	ctx, cancel := context.WithTimeout(context.Background(), c.config.DatabaseTimeout)
	defer cancel()

	var passwords []storage.Password

	if err := c.session.Operate(ctx, func(ctx context.Context) error {
		return c.passwordCollection().Find(ctx, &passwords, emptyFilter())
	}); err != nil {
		return []storage.Password{}, nil
	}

	return passwords, nil
}

func (c *mongoStorage) CreateOfflineSessions(s storage.OfflineSessions) error {
	ctx, cancel := context.WithTimeout(context.Background(), c.config.DatabaseTimeout)
	defer cancel()

	return c.session.Operate(ctx, func(ctx context.Context) error {
		return c.offlineSessionCollection().Insert(ctx, fromStorageOfflineSessions(s))
	})
}

func (c *mongoStorage) UpdateOfflineSessions(userID string, connID string, updater func(s storage.OfflineSessions) (storage.OfflineSessions, error)) error {
	ctx, cancel := context.WithTimeout(context.Background(), c.config.DatabaseTimeout)
	defer cancel()

	return c.session.Operate(ctx, func(ctx context.Context) error {
		var currentOfflineSessions OfflineSessions

		if err := c.offlineSessionCollection().FindOne(ctx, &currentOfflineSessions, filterMatchUserIDConnID(userID, connID)); err != nil {
			return err
		}

		updated, err := updater(toStorageOfflineSessions(currentOfflineSessions))
		if err != nil {
			return err
		}

		if err := c.offlineSessionCollection().ReplaceOne(ctx, filterMatchUserIDConnID(userID, connID), fromStorageOfflineSessions(updated)); err != nil {
			return err
		}

		return nil
	})
}

func (c *mongoStorage) GetOfflineSessions(userID string, connID string) (storage.OfflineSessions, error) {
	ctx, cancel := context.WithTimeout(context.Background(), c.config.DatabaseTimeout)
	defer cancel()

	var os OfflineSessions

	if err := c.session.Operate(ctx, func(ctx context.Context) error {
		return c.offlineSessionCollection().FindOne(ctx, &os, filterMatchUserIDConnID(userID, connID))
	}); err != nil {
		return storage.OfflineSessions{}, err
	}

	return toStorageOfflineSessions(os), nil
}

func (c *mongoStorage) DeleteOfflineSessions(userID string, connID string) error {
	ctx, cancel := context.WithTimeout(context.Background(), c.config.DatabaseTimeout)
	defer cancel()

	return c.session.Operate(ctx, func(ctx context.Context) error {
		return c.offlineSessionCollection().DeleteOne(ctx, filterMatchUserIDConnID(userID, connID))
	})
}

func (c *mongoStorage) CreateConnector(connector storage.Connector) error {
	ctx, cancel := context.WithTimeout(context.Background(), c.config.DatabaseTimeout)
	defer cancel()

	return c.session.Operate(ctx, func(ctx context.Context) error {
		return c.connectorCollection().Insert(ctx, connector)
	})
}

func (c *mongoStorage) GetConnector(id string) (storage.Connector, error) {
	ctx, cancel := context.WithTimeout(context.Background(), c.config.DatabaseTimeout)
	defer cancel()

	var storageConnector storage.Connector

	if err := c.session.Operate(ctx, func(ctx context.Context) error {
		return c.connectorCollection().FindOne(ctx, &storageConnector, filterMatch_ID(id))
	}); err != nil {
		return storage.Connector{}, err
	}

	return storageConnector, nil
}

func (c *mongoStorage) UpdateConnector(id string, updater func(s storage.Connector) (storage.Connector, error)) error {
	ctx, cancel := context.WithTimeout(context.Background(), c.config.DatabaseTimeout)
	defer cancel()

	return c.session.Operate(ctx, func(ctx context.Context) error {
		var currentStorageConnector storage.Connector

		err := c.connectorCollection().FindOne(ctx, &currentStorageConnector, filterMatch_ID(id))
		if err != nil {
			return err
		}

		updated, err := updater(currentStorageConnector)
		if err != nil {
			return err
		}

		if err := c.connectorCollection().ReplaceOne(ctx, filterMatch_ID(id), updated); err != nil {
			return err
		}

		return nil
	})
}

func (c *mongoStorage) DeleteConnector(id string) error {
	ctx, cancel := context.WithTimeout(context.Background(), c.config.DatabaseTimeout)
	defer cancel()

	return c.session.Operate(ctx, func(ctx context.Context) error {
		return c.connectorCollection().DeleteOne(ctx, filterMatch_ID(id))
	})
}

func (c *mongoStorage) ListConnectors() ([]storage.Connector, error) {
	ctx, cancel := context.WithTimeout(context.Background(), c.config.DatabaseTimeout)
	defer cancel()

	var storageConnectors []storage.Connector

	if err := c.session.Operate(ctx, func(ctx context.Context) error {
		return c.connectorCollection().Find(ctx, &storageConnectors, emptyFilter())
	}); err != nil {
		return []storage.Connector{}, err
	}

	return storageConnectors, nil
}

func (c *mongoStorage) CreateDeviceRequest(d storage.DeviceRequest) error {
	ctx, cancel := context.WithTimeout(context.Background(), c.config.DatabaseTimeout)
	defer cancel()

	if err := c.deviceRequestCollection().Insert(ctx, fromStorageDeviceRequest(d)); err != nil {
		return err
	}

	return nil
}

func (c *mongoStorage) GetDeviceRequest(userCode string) (r storage.DeviceRequest, err error) {
	ctx, cancel := context.WithTimeout(context.Background(), c.config.DatabaseTimeout)
	defer cancel()

	var d DeviceRequest
	if err := c.deviceRequestCollection().FindOne(ctx, &d, filterMatchUserCode(userCode)); err != nil {
		return storage.DeviceRequest{}, err
	}

	return toStorageDeviceRequest(d), nil
}

func (c *mongoStorage) CreateDeviceToken(t storage.DeviceToken) error {
	ctx, cancel := context.WithTimeout(context.Background(), c.config.DatabaseTimeout)
	defer cancel()

	return c.session.Operate(ctx, func(ctx context.Context) error {
		return c.deviceTokenCollection().Insert(ctx, fromStorageDeviceToken(t))
	})
}

func (c *mongoStorage) GetDeviceToken(deviceCode string) (storage.DeviceToken, error) {
	ctx, cancel := context.WithTimeout(context.Background(), c.config.DatabaseTimeout)
	defer cancel()

	var deviceToken DeviceToken

	if err := c.session.Operate(ctx, func(ctx context.Context) error {
		return c.deviceTokenCollection().FindOne(ctx, &deviceToken, filterMatchDeviceCode(deviceCode))
	}); err != nil {
		return storage.DeviceToken{}, err
	}

	return toStorageDeviceToken(deviceToken), nil
}

func (c *mongoStorage) UpdateDeviceToken(deviceCode string, updater func(old storage.DeviceToken) (storage.DeviceToken, error)) error {
	ctx, cancel := context.WithTimeout(context.Background(), c.config.DatabaseTimeout)
	defer cancel()

	return c.session.Operate(ctx, func(ctx context.Context) error {
		var currentDeviceToken DeviceToken

		if err := c.deviceTokenCollection().FindOne(ctx, &currentDeviceToken, filterMatchDeviceCode(deviceCode)); err != nil {
			return err
		}

		updated, err := updater(toStorageDeviceToken(currentDeviceToken))
		if err != nil {
			return err
		}

		if err := c.deviceTokenCollection().ReplaceOne(ctx, filterMatchDeviceCode(deviceCode), fromStorageDeviceToken(updated)); err != nil {
			return err
		}

		return nil
	})
}

func (c *mongoStorage) GetKeys() (storage.Keys, error) {
	ctx, cancel := context.WithTimeout(context.Background(), c.config.DatabaseTimeout)
	defer cancel()

	var keys Keys

	if err := c.session.Operate(ctx, func(ctx context.Context) error {
		return c.keysCollection().FindOne(ctx, &keys, emptyFilter())
	}); err != nil {
		return storage.Keys{}, err
	}

	storageKeys, err := toStorageKeys(keys)
	if err != nil {
		return storage.Keys{}, err
	}

	return storageKeys, nil
}

func (c *mongoStorage) UpdateKeys(updater func(old storage.Keys) (storage.Keys, error)) error {
	ctx, cancel := context.WithTimeout(context.Background(), c.config.DatabaseTimeout)
	defer cancel()

	if err := c.session.Operate(ctx, func(ctx context.Context) error {
		firstUpdate := false

		current := Keys{}
		if err := c.keysCollection().FindOne(ctx, &current, emptyFilter()); err != nil {
			if !errors.Is(err, storage.ErrNotFound) {
				return err
			}

			firstUpdate = true
		}

		storageKeys, err := toStorageKeys(current)
		if err != nil {
			return errors.Wrap(err, "UpdateKeys: failed to transform keys to storage keys")
		}

		updated, err := updater(storageKeys)
		if err != nil {
			return errors.Wrap(err, "UpdateKeys: updater failed on storage keys")
		}

		keysUpdated, err := fromStorageKeys(updated)
		if err != nil {
			return errors.Wrap(err, "UpdateKeys: failed to transform keys to mongo keys")
		}

		if firstUpdate {
			if err := c.keysCollection().Insert(ctx, keysUpdated); err != nil {
				return errors.Wrap(err, "UpdateKeys: failed to insert into mongo")
			}
		} else {
			if err := c.keysCollection().ReplaceOne(ctx, emptyFilter(), keysUpdated); err != nil {
				return errors.Wrap(err, "UpdateKeys: failed to replace into mongo")
			}
		}

		return nil
	}); err != nil {
		return err
	}

	return nil
}
