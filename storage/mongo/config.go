package mongo

import (
	"context"
	"log"
	"time"

	"github.com/dexidp/dex/storage"
	"github.com/pkg/errors"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
	"go.mongodb.org/mongo-driver/mongo/readconcern"
	"go.mongodb.org/mongo-driver/mongo/writeconcern"
)

type Mongo struct {
	URI                   string        `json:"uri" yaml:"uri"`
	Database              string        `json:"database" yaml:"database"`
	ConnectionTimeout     time.Duration `json:"connection_timeout" yaml:"connection_timeout"`
	DatabaseTimeout       time.Duration `json:"database_timeout" yaml:"database_timeout"`
	UseGCInsteadOfIndexes bool          `json:"not_set_expire_index" yaml:"not_set_expire_index"`
}

func (p *Mongo) Open(logger log.Logger) (storage.Storage, error) {
	mongoStorage, err := p.open(logger)
	if err != nil {
		return nil, err
	}

	return mongoStorage, nil
}

func (p *Mongo) open(logger log.Logger) (*mongoStorage, error) {
	ctx, contextCancel := context.WithTimeout(context.Background(), p.ConnectionTimeout)
	defer contextCancel()

	clientOptions := options.Client().ApplyURI(p.URI)

	client, err := mongo.Connect(ctx, clientOptions)
	if err != nil {
		return nil, errors.Wrap(err, "unable to connect to mongo")
	}

	if err := client.Ping(ctx, nil); err != nil {
		return nil, errors.Wrap(err, "unable to ping mongo")
	}

	s, err := client.StartSession()
	if err != nil {
		return nil, errors.Wrap(err, "unable to start a mongo session")
	}

	txOptions := options.
		Transaction().
		SetWriteConcern(writeconcern.New(writeconcern.WMajority())).
		SetReadConcern(readconcern.Majority())

	c := &mongoStorage{
		session:  NewSession(s, txOptions),
		database: client.Database(p.Database),
		logger:   logger,
		config:   *p,
	}

	if err := c.initAuthCodeCollection(ctx); err != nil {
		return nil, errors.Wrap(err, "unable to initialize auth code collection")
	}
	if err := c.initAuthRequestCollection(ctx); err != nil {
		return nil, errors.Wrap(err, "unable to initialize auth request collection")
	}
	if err := c.initPasswordCollection(ctx); err != nil {
		return nil, errors.Wrap(err, "unable to initialize password collection")
	}
	if err := c.initDeviceRequestCollection(ctx); err != nil {
		return nil, errors.Wrap(err, "unable to initialize device request collection")
	}
	if err := c.initDeviceTokenCollection(ctx); err != nil {
		return nil, errors.Wrap(err, "unable to initialize device token collection")
	}
	if err := c.initClientCollection(ctx); err != nil {
		return nil, errors.Wrap(err, "unable to initialize client collection")
	}
	if err := c.initRefreshTokenCollection(ctx); err != nil {
		return nil, errors.Wrap(err, "unable to initialize refresh token collection")
	}
	if err := c.initOfflineSessionCollection(ctx); err != nil {
		return nil, errors.Wrap(err, "unable to initialize offline session collection")
	}

	return c, nil
}
