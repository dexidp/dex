// Code generated by entc, DO NOT EDIT.

package db

import (
	"context"
	"fmt"
	"log"

	"github.com/dexidp/dex/storage/ent/db/migrate"

	"github.com/dexidp/dex/storage/ent/db/authcode"
	"github.com/dexidp/dex/storage/ent/db/authrequest"
	"github.com/dexidp/dex/storage/ent/db/connector"
	"github.com/dexidp/dex/storage/ent/db/devicerequest"
	"github.com/dexidp/dex/storage/ent/db/devicetoken"
	"github.com/dexidp/dex/storage/ent/db/keys"
	"github.com/dexidp/dex/storage/ent/db/oauth2client"
	"github.com/dexidp/dex/storage/ent/db/offlinesession"
	"github.com/dexidp/dex/storage/ent/db/password"
	"github.com/dexidp/dex/storage/ent/db/refreshtoken"

	"entgo.io/ent/dialect"
	"entgo.io/ent/dialect/sql"
)

// Client is the client that holds all ent builders.
type Client struct {
	config
	// Schema is the client for creating, migrating and dropping schema.
	Schema *migrate.Schema
	// AuthCode is the client for interacting with the AuthCode builders.
	AuthCode *AuthCodeClient
	// AuthRequest is the client for interacting with the AuthRequest builders.
	AuthRequest *AuthRequestClient
	// Connector is the client for interacting with the Connector builders.
	Connector *ConnectorClient
	// DeviceRequest is the client for interacting with the DeviceRequest builders.
	DeviceRequest *DeviceRequestClient
	// DeviceToken is the client for interacting with the DeviceToken builders.
	DeviceToken *DeviceTokenClient
	// Keys is the client for interacting with the Keys builders.
	Keys *KeysClient
	// OAuth2Client is the client for interacting with the OAuth2Client builders.
	OAuth2Client *OAuth2ClientClient
	// OfflineSession is the client for interacting with the OfflineSession builders.
	OfflineSession *OfflineSessionClient
	// Password is the client for interacting with the Password builders.
	Password *PasswordClient
	// RefreshToken is the client for interacting with the RefreshToken builders.
	RefreshToken *RefreshTokenClient
}

// NewClient creates a new client configured with the given options.
func NewClient(opts ...Option) *Client {
	cfg := config{log: log.Println, hooks: &hooks{}}
	cfg.options(opts...)
	client := &Client{config: cfg}
	client.init()
	return client
}

func (c *Client) init() {
	c.Schema = migrate.NewSchema(c.driver)
	c.AuthCode = NewAuthCodeClient(c.config)
	c.AuthRequest = NewAuthRequestClient(c.config)
	c.Connector = NewConnectorClient(c.config)
	c.DeviceRequest = NewDeviceRequestClient(c.config)
	c.DeviceToken = NewDeviceTokenClient(c.config)
	c.Keys = NewKeysClient(c.config)
	c.OAuth2Client = NewOAuth2ClientClient(c.config)
	c.OfflineSession = NewOfflineSessionClient(c.config)
	c.Password = NewPasswordClient(c.config)
	c.RefreshToken = NewRefreshTokenClient(c.config)
}

// Open opens a database/sql.DB specified by the driver name and
// the data source name, and returns a new client attached to it.
// Optional parameters can be added for configuring the client.
func Open(driverName, dataSourceName string, options ...Option) (*Client, error) {
	switch driverName {
	case dialect.MySQL, dialect.Postgres, dialect.SQLite:
		drv, err := sql.Open(driverName, dataSourceName)
		if err != nil {
			return nil, err
		}
		return NewClient(append(options, Driver(drv))...), nil
	default:
		return nil, fmt.Errorf("unsupported driver: %q", driverName)
	}
}

// Tx returns a new transactional client. The provided context
// is used until the transaction is committed or rolled back.
func (c *Client) Tx(ctx context.Context) (*Tx, error) {
	if _, ok := c.driver.(*txDriver); ok {
		return nil, fmt.Errorf("db: cannot start a transaction within a transaction")
	}
	tx, err := newTx(ctx, c.driver)
	if err != nil {
		return nil, fmt.Errorf("db: starting a transaction: %w", err)
	}
	cfg := c.config
	cfg.driver = tx
	return &Tx{
		ctx:            ctx,
		config:         cfg,
		AuthCode:       NewAuthCodeClient(cfg),
		AuthRequest:    NewAuthRequestClient(cfg),
		Connector:      NewConnectorClient(cfg),
		DeviceRequest:  NewDeviceRequestClient(cfg),
		DeviceToken:    NewDeviceTokenClient(cfg),
		Keys:           NewKeysClient(cfg),
		OAuth2Client:   NewOAuth2ClientClient(cfg),
		OfflineSession: NewOfflineSessionClient(cfg),
		Password:       NewPasswordClient(cfg),
		RefreshToken:   NewRefreshTokenClient(cfg),
	}, nil
}

// BeginTx returns a transactional client with specified options.
func (c *Client) BeginTx(ctx context.Context, opts *sql.TxOptions) (*Tx, error) {
	if _, ok := c.driver.(*txDriver); ok {
		return nil, fmt.Errorf("ent: cannot start a transaction within a transaction")
	}
	tx, err := c.driver.(interface {
		BeginTx(context.Context, *sql.TxOptions) (dialect.Tx, error)
	}).BeginTx(ctx, opts)
	if err != nil {
		return nil, fmt.Errorf("ent: starting a transaction: %w", err)
	}
	cfg := c.config
	cfg.driver = &txDriver{tx: tx, drv: c.driver}
	return &Tx{
		config:         cfg,
		AuthCode:       NewAuthCodeClient(cfg),
		AuthRequest:    NewAuthRequestClient(cfg),
		Connector:      NewConnectorClient(cfg),
		DeviceRequest:  NewDeviceRequestClient(cfg),
		DeviceToken:    NewDeviceTokenClient(cfg),
		Keys:           NewKeysClient(cfg),
		OAuth2Client:   NewOAuth2ClientClient(cfg),
		OfflineSession: NewOfflineSessionClient(cfg),
		Password:       NewPasswordClient(cfg),
		RefreshToken:   NewRefreshTokenClient(cfg),
	}, nil
}

// Debug returns a new debug-client. It's used to get verbose logging on specific operations.
//
//	client.Debug().
//		AuthCode.
//		Query().
//		Count(ctx)
//
func (c *Client) Debug() *Client {
	if c.debug {
		return c
	}
	cfg := c.config
	cfg.driver = dialect.Debug(c.driver, c.log)
	client := &Client{config: cfg}
	client.init()
	return client
}

// Close closes the database connection and prevents new queries from starting.
func (c *Client) Close() error {
	return c.driver.Close()
}

// Use adds the mutation hooks to all the entity clients.
// In order to add hooks to a specific client, call: `client.Node.Use(...)`.
func (c *Client) Use(hooks ...Hook) {
	c.AuthCode.Use(hooks...)
	c.AuthRequest.Use(hooks...)
	c.Connector.Use(hooks...)
	c.DeviceRequest.Use(hooks...)
	c.DeviceToken.Use(hooks...)
	c.Keys.Use(hooks...)
	c.OAuth2Client.Use(hooks...)
	c.OfflineSession.Use(hooks...)
	c.Password.Use(hooks...)
	c.RefreshToken.Use(hooks...)
}

// AuthCodeClient is a client for the AuthCode schema.
type AuthCodeClient struct {
	config
}

// NewAuthCodeClient returns a client for the AuthCode from the given config.
func NewAuthCodeClient(c config) *AuthCodeClient {
	return &AuthCodeClient{config: c}
}

// Use adds a list of mutation hooks to the hooks stack.
// A call to `Use(f, g, h)` equals to `authcode.Hooks(f(g(h())))`.
func (c *AuthCodeClient) Use(hooks ...Hook) {
	c.hooks.AuthCode = append(c.hooks.AuthCode, hooks...)
}

// Create returns a create builder for AuthCode.
func (c *AuthCodeClient) Create() *AuthCodeCreate {
	mutation := newAuthCodeMutation(c.config, OpCreate)
	return &AuthCodeCreate{config: c.config, hooks: c.Hooks(), mutation: mutation}
}

// CreateBulk returns a builder for creating a bulk of AuthCode entities.
func (c *AuthCodeClient) CreateBulk(builders ...*AuthCodeCreate) *AuthCodeCreateBulk {
	return &AuthCodeCreateBulk{config: c.config, builders: builders}
}

// Update returns an update builder for AuthCode.
func (c *AuthCodeClient) Update() *AuthCodeUpdate {
	mutation := newAuthCodeMutation(c.config, OpUpdate)
	return &AuthCodeUpdate{config: c.config, hooks: c.Hooks(), mutation: mutation}
}

// UpdateOne returns an update builder for the given entity.
func (c *AuthCodeClient) UpdateOne(ac *AuthCode) *AuthCodeUpdateOne {
	mutation := newAuthCodeMutation(c.config, OpUpdateOne, withAuthCode(ac))
	return &AuthCodeUpdateOne{config: c.config, hooks: c.Hooks(), mutation: mutation}
}

// UpdateOneID returns an update builder for the given id.
func (c *AuthCodeClient) UpdateOneID(id string) *AuthCodeUpdateOne {
	mutation := newAuthCodeMutation(c.config, OpUpdateOne, withAuthCodeID(id))
	return &AuthCodeUpdateOne{config: c.config, hooks: c.Hooks(), mutation: mutation}
}

// Delete returns a delete builder for AuthCode.
func (c *AuthCodeClient) Delete() *AuthCodeDelete {
	mutation := newAuthCodeMutation(c.config, OpDelete)
	return &AuthCodeDelete{config: c.config, hooks: c.Hooks(), mutation: mutation}
}

// DeleteOne returns a delete builder for the given entity.
func (c *AuthCodeClient) DeleteOne(ac *AuthCode) *AuthCodeDeleteOne {
	return c.DeleteOneID(ac.ID)
}

// DeleteOneID returns a delete builder for the given id.
func (c *AuthCodeClient) DeleteOneID(id string) *AuthCodeDeleteOne {
	builder := c.Delete().Where(authcode.ID(id))
	builder.mutation.id = &id
	builder.mutation.op = OpDeleteOne
	return &AuthCodeDeleteOne{builder}
}

// Query returns a query builder for AuthCode.
func (c *AuthCodeClient) Query() *AuthCodeQuery {
	return &AuthCodeQuery{
		config: c.config,
	}
}

// Get returns a AuthCode entity by its id.
func (c *AuthCodeClient) Get(ctx context.Context, id string) (*AuthCode, error) {
	return c.Query().Where(authcode.ID(id)).Only(ctx)
}

// GetX is like Get, but panics if an error occurs.
func (c *AuthCodeClient) GetX(ctx context.Context, id string) *AuthCode {
	obj, err := c.Get(ctx, id)
	if err != nil {
		panic(err)
	}
	return obj
}

// Hooks returns the client hooks.
func (c *AuthCodeClient) Hooks() []Hook {
	return c.hooks.AuthCode
}

// AuthRequestClient is a client for the AuthRequest schema.
type AuthRequestClient struct {
	config
}

// NewAuthRequestClient returns a client for the AuthRequest from the given config.
func NewAuthRequestClient(c config) *AuthRequestClient {
	return &AuthRequestClient{config: c}
}

// Use adds a list of mutation hooks to the hooks stack.
// A call to `Use(f, g, h)` equals to `authrequest.Hooks(f(g(h())))`.
func (c *AuthRequestClient) Use(hooks ...Hook) {
	c.hooks.AuthRequest = append(c.hooks.AuthRequest, hooks...)
}

// Create returns a create builder for AuthRequest.
func (c *AuthRequestClient) Create() *AuthRequestCreate {
	mutation := newAuthRequestMutation(c.config, OpCreate)
	return &AuthRequestCreate{config: c.config, hooks: c.Hooks(), mutation: mutation}
}

// CreateBulk returns a builder for creating a bulk of AuthRequest entities.
func (c *AuthRequestClient) CreateBulk(builders ...*AuthRequestCreate) *AuthRequestCreateBulk {
	return &AuthRequestCreateBulk{config: c.config, builders: builders}
}

// Update returns an update builder for AuthRequest.
func (c *AuthRequestClient) Update() *AuthRequestUpdate {
	mutation := newAuthRequestMutation(c.config, OpUpdate)
	return &AuthRequestUpdate{config: c.config, hooks: c.Hooks(), mutation: mutation}
}

// UpdateOne returns an update builder for the given entity.
func (c *AuthRequestClient) UpdateOne(ar *AuthRequest) *AuthRequestUpdateOne {
	mutation := newAuthRequestMutation(c.config, OpUpdateOne, withAuthRequest(ar))
	return &AuthRequestUpdateOne{config: c.config, hooks: c.Hooks(), mutation: mutation}
}

// UpdateOneID returns an update builder for the given id.
func (c *AuthRequestClient) UpdateOneID(id string) *AuthRequestUpdateOne {
	mutation := newAuthRequestMutation(c.config, OpUpdateOne, withAuthRequestID(id))
	return &AuthRequestUpdateOne{config: c.config, hooks: c.Hooks(), mutation: mutation}
}

// Delete returns a delete builder for AuthRequest.
func (c *AuthRequestClient) Delete() *AuthRequestDelete {
	mutation := newAuthRequestMutation(c.config, OpDelete)
	return &AuthRequestDelete{config: c.config, hooks: c.Hooks(), mutation: mutation}
}

// DeleteOne returns a delete builder for the given entity.
func (c *AuthRequestClient) DeleteOne(ar *AuthRequest) *AuthRequestDeleteOne {
	return c.DeleteOneID(ar.ID)
}

// DeleteOneID returns a delete builder for the given id.
func (c *AuthRequestClient) DeleteOneID(id string) *AuthRequestDeleteOne {
	builder := c.Delete().Where(authrequest.ID(id))
	builder.mutation.id = &id
	builder.mutation.op = OpDeleteOne
	return &AuthRequestDeleteOne{builder}
}

// Query returns a query builder for AuthRequest.
func (c *AuthRequestClient) Query() *AuthRequestQuery {
	return &AuthRequestQuery{
		config: c.config,
	}
}

// Get returns a AuthRequest entity by its id.
func (c *AuthRequestClient) Get(ctx context.Context, id string) (*AuthRequest, error) {
	return c.Query().Where(authrequest.ID(id)).Only(ctx)
}

// GetX is like Get, but panics if an error occurs.
func (c *AuthRequestClient) GetX(ctx context.Context, id string) *AuthRequest {
	obj, err := c.Get(ctx, id)
	if err != nil {
		panic(err)
	}
	return obj
}

// Hooks returns the client hooks.
func (c *AuthRequestClient) Hooks() []Hook {
	return c.hooks.AuthRequest
}

// ConnectorClient is a client for the Connector schema.
type ConnectorClient struct {
	config
}

// NewConnectorClient returns a client for the Connector from the given config.
func NewConnectorClient(c config) *ConnectorClient {
	return &ConnectorClient{config: c}
}

// Use adds a list of mutation hooks to the hooks stack.
// A call to `Use(f, g, h)` equals to `connector.Hooks(f(g(h())))`.
func (c *ConnectorClient) Use(hooks ...Hook) {
	c.hooks.Connector = append(c.hooks.Connector, hooks...)
}

// Create returns a create builder for Connector.
func (c *ConnectorClient) Create() *ConnectorCreate {
	mutation := newConnectorMutation(c.config, OpCreate)
	return &ConnectorCreate{config: c.config, hooks: c.Hooks(), mutation: mutation}
}

// CreateBulk returns a builder for creating a bulk of Connector entities.
func (c *ConnectorClient) CreateBulk(builders ...*ConnectorCreate) *ConnectorCreateBulk {
	return &ConnectorCreateBulk{config: c.config, builders: builders}
}

// Update returns an update builder for Connector.
func (c *ConnectorClient) Update() *ConnectorUpdate {
	mutation := newConnectorMutation(c.config, OpUpdate)
	return &ConnectorUpdate{config: c.config, hooks: c.Hooks(), mutation: mutation}
}

// UpdateOne returns an update builder for the given entity.
func (c *ConnectorClient) UpdateOne(co *Connector) *ConnectorUpdateOne {
	mutation := newConnectorMutation(c.config, OpUpdateOne, withConnector(co))
	return &ConnectorUpdateOne{config: c.config, hooks: c.Hooks(), mutation: mutation}
}

// UpdateOneID returns an update builder for the given id.
func (c *ConnectorClient) UpdateOneID(id string) *ConnectorUpdateOne {
	mutation := newConnectorMutation(c.config, OpUpdateOne, withConnectorID(id))
	return &ConnectorUpdateOne{config: c.config, hooks: c.Hooks(), mutation: mutation}
}

// Delete returns a delete builder for Connector.
func (c *ConnectorClient) Delete() *ConnectorDelete {
	mutation := newConnectorMutation(c.config, OpDelete)
	return &ConnectorDelete{config: c.config, hooks: c.Hooks(), mutation: mutation}
}

// DeleteOne returns a delete builder for the given entity.
func (c *ConnectorClient) DeleteOne(co *Connector) *ConnectorDeleteOne {
	return c.DeleteOneID(co.ID)
}

// DeleteOneID returns a delete builder for the given id.
func (c *ConnectorClient) DeleteOneID(id string) *ConnectorDeleteOne {
	builder := c.Delete().Where(connector.ID(id))
	builder.mutation.id = &id
	builder.mutation.op = OpDeleteOne
	return &ConnectorDeleteOne{builder}
}

// Query returns a query builder for Connector.
func (c *ConnectorClient) Query() *ConnectorQuery {
	return &ConnectorQuery{
		config: c.config,
	}
}

// Get returns a Connector entity by its id.
func (c *ConnectorClient) Get(ctx context.Context, id string) (*Connector, error) {
	return c.Query().Where(connector.ID(id)).Only(ctx)
}

// GetX is like Get, but panics if an error occurs.
func (c *ConnectorClient) GetX(ctx context.Context, id string) *Connector {
	obj, err := c.Get(ctx, id)
	if err != nil {
		panic(err)
	}
	return obj
}

// Hooks returns the client hooks.
func (c *ConnectorClient) Hooks() []Hook {
	return c.hooks.Connector
}

// DeviceRequestClient is a client for the DeviceRequest schema.
type DeviceRequestClient struct {
	config
}

// NewDeviceRequestClient returns a client for the DeviceRequest from the given config.
func NewDeviceRequestClient(c config) *DeviceRequestClient {
	return &DeviceRequestClient{config: c}
}

// Use adds a list of mutation hooks to the hooks stack.
// A call to `Use(f, g, h)` equals to `devicerequest.Hooks(f(g(h())))`.
func (c *DeviceRequestClient) Use(hooks ...Hook) {
	c.hooks.DeviceRequest = append(c.hooks.DeviceRequest, hooks...)
}

// Create returns a create builder for DeviceRequest.
func (c *DeviceRequestClient) Create() *DeviceRequestCreate {
	mutation := newDeviceRequestMutation(c.config, OpCreate)
	return &DeviceRequestCreate{config: c.config, hooks: c.Hooks(), mutation: mutation}
}

// CreateBulk returns a builder for creating a bulk of DeviceRequest entities.
func (c *DeviceRequestClient) CreateBulk(builders ...*DeviceRequestCreate) *DeviceRequestCreateBulk {
	return &DeviceRequestCreateBulk{config: c.config, builders: builders}
}

// Update returns an update builder for DeviceRequest.
func (c *DeviceRequestClient) Update() *DeviceRequestUpdate {
	mutation := newDeviceRequestMutation(c.config, OpUpdate)
	return &DeviceRequestUpdate{config: c.config, hooks: c.Hooks(), mutation: mutation}
}

// UpdateOne returns an update builder for the given entity.
func (c *DeviceRequestClient) UpdateOne(dr *DeviceRequest) *DeviceRequestUpdateOne {
	mutation := newDeviceRequestMutation(c.config, OpUpdateOne, withDeviceRequest(dr))
	return &DeviceRequestUpdateOne{config: c.config, hooks: c.Hooks(), mutation: mutation}
}

// UpdateOneID returns an update builder for the given id.
func (c *DeviceRequestClient) UpdateOneID(id int) *DeviceRequestUpdateOne {
	mutation := newDeviceRequestMutation(c.config, OpUpdateOne, withDeviceRequestID(id))
	return &DeviceRequestUpdateOne{config: c.config, hooks: c.Hooks(), mutation: mutation}
}

// Delete returns a delete builder for DeviceRequest.
func (c *DeviceRequestClient) Delete() *DeviceRequestDelete {
	mutation := newDeviceRequestMutation(c.config, OpDelete)
	return &DeviceRequestDelete{config: c.config, hooks: c.Hooks(), mutation: mutation}
}

// DeleteOne returns a delete builder for the given entity.
func (c *DeviceRequestClient) DeleteOne(dr *DeviceRequest) *DeviceRequestDeleteOne {
	return c.DeleteOneID(dr.ID)
}

// DeleteOneID returns a delete builder for the given id.
func (c *DeviceRequestClient) DeleteOneID(id int) *DeviceRequestDeleteOne {
	builder := c.Delete().Where(devicerequest.ID(id))
	builder.mutation.id = &id
	builder.mutation.op = OpDeleteOne
	return &DeviceRequestDeleteOne{builder}
}

// Query returns a query builder for DeviceRequest.
func (c *DeviceRequestClient) Query() *DeviceRequestQuery {
	return &DeviceRequestQuery{
		config: c.config,
	}
}

// Get returns a DeviceRequest entity by its id.
func (c *DeviceRequestClient) Get(ctx context.Context, id int) (*DeviceRequest, error) {
	return c.Query().Where(devicerequest.ID(id)).Only(ctx)
}

// GetX is like Get, but panics if an error occurs.
func (c *DeviceRequestClient) GetX(ctx context.Context, id int) *DeviceRequest {
	obj, err := c.Get(ctx, id)
	if err != nil {
		panic(err)
	}
	return obj
}

// Hooks returns the client hooks.
func (c *DeviceRequestClient) Hooks() []Hook {
	return c.hooks.DeviceRequest
}

// DeviceTokenClient is a client for the DeviceToken schema.
type DeviceTokenClient struct {
	config
}

// NewDeviceTokenClient returns a client for the DeviceToken from the given config.
func NewDeviceTokenClient(c config) *DeviceTokenClient {
	return &DeviceTokenClient{config: c}
}

// Use adds a list of mutation hooks to the hooks stack.
// A call to `Use(f, g, h)` equals to `devicetoken.Hooks(f(g(h())))`.
func (c *DeviceTokenClient) Use(hooks ...Hook) {
	c.hooks.DeviceToken = append(c.hooks.DeviceToken, hooks...)
}

// Create returns a create builder for DeviceToken.
func (c *DeviceTokenClient) Create() *DeviceTokenCreate {
	mutation := newDeviceTokenMutation(c.config, OpCreate)
	return &DeviceTokenCreate{config: c.config, hooks: c.Hooks(), mutation: mutation}
}

// CreateBulk returns a builder for creating a bulk of DeviceToken entities.
func (c *DeviceTokenClient) CreateBulk(builders ...*DeviceTokenCreate) *DeviceTokenCreateBulk {
	return &DeviceTokenCreateBulk{config: c.config, builders: builders}
}

// Update returns an update builder for DeviceToken.
func (c *DeviceTokenClient) Update() *DeviceTokenUpdate {
	mutation := newDeviceTokenMutation(c.config, OpUpdate)
	return &DeviceTokenUpdate{config: c.config, hooks: c.Hooks(), mutation: mutation}
}

// UpdateOne returns an update builder for the given entity.
func (c *DeviceTokenClient) UpdateOne(dt *DeviceToken) *DeviceTokenUpdateOne {
	mutation := newDeviceTokenMutation(c.config, OpUpdateOne, withDeviceToken(dt))
	return &DeviceTokenUpdateOne{config: c.config, hooks: c.Hooks(), mutation: mutation}
}

// UpdateOneID returns an update builder for the given id.
func (c *DeviceTokenClient) UpdateOneID(id int) *DeviceTokenUpdateOne {
	mutation := newDeviceTokenMutation(c.config, OpUpdateOne, withDeviceTokenID(id))
	return &DeviceTokenUpdateOne{config: c.config, hooks: c.Hooks(), mutation: mutation}
}

// Delete returns a delete builder for DeviceToken.
func (c *DeviceTokenClient) Delete() *DeviceTokenDelete {
	mutation := newDeviceTokenMutation(c.config, OpDelete)
	return &DeviceTokenDelete{config: c.config, hooks: c.Hooks(), mutation: mutation}
}

// DeleteOne returns a delete builder for the given entity.
func (c *DeviceTokenClient) DeleteOne(dt *DeviceToken) *DeviceTokenDeleteOne {
	return c.DeleteOneID(dt.ID)
}

// DeleteOneID returns a delete builder for the given id.
func (c *DeviceTokenClient) DeleteOneID(id int) *DeviceTokenDeleteOne {
	builder := c.Delete().Where(devicetoken.ID(id))
	builder.mutation.id = &id
	builder.mutation.op = OpDeleteOne
	return &DeviceTokenDeleteOne{builder}
}

// Query returns a query builder for DeviceToken.
func (c *DeviceTokenClient) Query() *DeviceTokenQuery {
	return &DeviceTokenQuery{
		config: c.config,
	}
}

// Get returns a DeviceToken entity by its id.
func (c *DeviceTokenClient) Get(ctx context.Context, id int) (*DeviceToken, error) {
	return c.Query().Where(devicetoken.ID(id)).Only(ctx)
}

// GetX is like Get, but panics if an error occurs.
func (c *DeviceTokenClient) GetX(ctx context.Context, id int) *DeviceToken {
	obj, err := c.Get(ctx, id)
	if err != nil {
		panic(err)
	}
	return obj
}

// Hooks returns the client hooks.
func (c *DeviceTokenClient) Hooks() []Hook {
	return c.hooks.DeviceToken
}

// KeysClient is a client for the Keys schema.
type KeysClient struct {
	config
}

// NewKeysClient returns a client for the Keys from the given config.
func NewKeysClient(c config) *KeysClient {
	return &KeysClient{config: c}
}

// Use adds a list of mutation hooks to the hooks stack.
// A call to `Use(f, g, h)` equals to `keys.Hooks(f(g(h())))`.
func (c *KeysClient) Use(hooks ...Hook) {
	c.hooks.Keys = append(c.hooks.Keys, hooks...)
}

// Create returns a create builder for Keys.
func (c *KeysClient) Create() *KeysCreate {
	mutation := newKeysMutation(c.config, OpCreate)
	return &KeysCreate{config: c.config, hooks: c.Hooks(), mutation: mutation}
}

// CreateBulk returns a builder for creating a bulk of Keys entities.
func (c *KeysClient) CreateBulk(builders ...*KeysCreate) *KeysCreateBulk {
	return &KeysCreateBulk{config: c.config, builders: builders}
}

// Update returns an update builder for Keys.
func (c *KeysClient) Update() *KeysUpdate {
	mutation := newKeysMutation(c.config, OpUpdate)
	return &KeysUpdate{config: c.config, hooks: c.Hooks(), mutation: mutation}
}

// UpdateOne returns an update builder for the given entity.
func (c *KeysClient) UpdateOne(k *Keys) *KeysUpdateOne {
	mutation := newKeysMutation(c.config, OpUpdateOne, withKeys(k))
	return &KeysUpdateOne{config: c.config, hooks: c.Hooks(), mutation: mutation}
}

// UpdateOneID returns an update builder for the given id.
func (c *KeysClient) UpdateOneID(id string) *KeysUpdateOne {
	mutation := newKeysMutation(c.config, OpUpdateOne, withKeysID(id))
	return &KeysUpdateOne{config: c.config, hooks: c.Hooks(), mutation: mutation}
}

// Delete returns a delete builder for Keys.
func (c *KeysClient) Delete() *KeysDelete {
	mutation := newKeysMutation(c.config, OpDelete)
	return &KeysDelete{config: c.config, hooks: c.Hooks(), mutation: mutation}
}

// DeleteOne returns a delete builder for the given entity.
func (c *KeysClient) DeleteOne(k *Keys) *KeysDeleteOne {
	return c.DeleteOneID(k.ID)
}

// DeleteOneID returns a delete builder for the given id.
func (c *KeysClient) DeleteOneID(id string) *KeysDeleteOne {
	builder := c.Delete().Where(keys.ID(id))
	builder.mutation.id = &id
	builder.mutation.op = OpDeleteOne
	return &KeysDeleteOne{builder}
}

// Query returns a query builder for Keys.
func (c *KeysClient) Query() *KeysQuery {
	return &KeysQuery{
		config: c.config,
	}
}

// Get returns a Keys entity by its id.
func (c *KeysClient) Get(ctx context.Context, id string) (*Keys, error) {
	return c.Query().Where(keys.ID(id)).Only(ctx)
}

// GetX is like Get, but panics if an error occurs.
func (c *KeysClient) GetX(ctx context.Context, id string) *Keys {
	obj, err := c.Get(ctx, id)
	if err != nil {
		panic(err)
	}
	return obj
}

// Hooks returns the client hooks.
func (c *KeysClient) Hooks() []Hook {
	return c.hooks.Keys
}

// OAuth2ClientClient is a client for the OAuth2Client schema.
type OAuth2ClientClient struct {
	config
}

// NewOAuth2ClientClient returns a client for the OAuth2Client from the given config.
func NewOAuth2ClientClient(c config) *OAuth2ClientClient {
	return &OAuth2ClientClient{config: c}
}

// Use adds a list of mutation hooks to the hooks stack.
// A call to `Use(f, g, h)` equals to `oauth2client.Hooks(f(g(h())))`.
func (c *OAuth2ClientClient) Use(hooks ...Hook) {
	c.hooks.OAuth2Client = append(c.hooks.OAuth2Client, hooks...)
}

// Create returns a create builder for OAuth2Client.
func (c *OAuth2ClientClient) Create() *OAuth2ClientCreate {
	mutation := newOAuth2ClientMutation(c.config, OpCreate)
	return &OAuth2ClientCreate{config: c.config, hooks: c.Hooks(), mutation: mutation}
}

// CreateBulk returns a builder for creating a bulk of OAuth2Client entities.
func (c *OAuth2ClientClient) CreateBulk(builders ...*OAuth2ClientCreate) *OAuth2ClientCreateBulk {
	return &OAuth2ClientCreateBulk{config: c.config, builders: builders}
}

// Update returns an update builder for OAuth2Client.
func (c *OAuth2ClientClient) Update() *OAuth2ClientUpdate {
	mutation := newOAuth2ClientMutation(c.config, OpUpdate)
	return &OAuth2ClientUpdate{config: c.config, hooks: c.Hooks(), mutation: mutation}
}

// UpdateOne returns an update builder for the given entity.
func (c *OAuth2ClientClient) UpdateOne(o *OAuth2Client) *OAuth2ClientUpdateOne {
	mutation := newOAuth2ClientMutation(c.config, OpUpdateOne, withOAuth2Client(o))
	return &OAuth2ClientUpdateOne{config: c.config, hooks: c.Hooks(), mutation: mutation}
}

// UpdateOneID returns an update builder for the given id.
func (c *OAuth2ClientClient) UpdateOneID(id string) *OAuth2ClientUpdateOne {
	mutation := newOAuth2ClientMutation(c.config, OpUpdateOne, withOAuth2ClientID(id))
	return &OAuth2ClientUpdateOne{config: c.config, hooks: c.Hooks(), mutation: mutation}
}

// Delete returns a delete builder for OAuth2Client.
func (c *OAuth2ClientClient) Delete() *OAuth2ClientDelete {
	mutation := newOAuth2ClientMutation(c.config, OpDelete)
	return &OAuth2ClientDelete{config: c.config, hooks: c.Hooks(), mutation: mutation}
}

// DeleteOne returns a delete builder for the given entity.
func (c *OAuth2ClientClient) DeleteOne(o *OAuth2Client) *OAuth2ClientDeleteOne {
	return c.DeleteOneID(o.ID)
}

// DeleteOneID returns a delete builder for the given id.
func (c *OAuth2ClientClient) DeleteOneID(id string) *OAuth2ClientDeleteOne {
	builder := c.Delete().Where(oauth2client.ID(id))
	builder.mutation.id = &id
	builder.mutation.op = OpDeleteOne
	return &OAuth2ClientDeleteOne{builder}
}

// Query returns a query builder for OAuth2Client.
func (c *OAuth2ClientClient) Query() *OAuth2ClientQuery {
	return &OAuth2ClientQuery{
		config: c.config,
	}
}

// Get returns a OAuth2Client entity by its id.
func (c *OAuth2ClientClient) Get(ctx context.Context, id string) (*OAuth2Client, error) {
	return c.Query().Where(oauth2client.ID(id)).Only(ctx)
}

// GetX is like Get, but panics if an error occurs.
func (c *OAuth2ClientClient) GetX(ctx context.Context, id string) *OAuth2Client {
	obj, err := c.Get(ctx, id)
	if err != nil {
		panic(err)
	}
	return obj
}

// Hooks returns the client hooks.
func (c *OAuth2ClientClient) Hooks() []Hook {
	return c.hooks.OAuth2Client
}

// OfflineSessionClient is a client for the OfflineSession schema.
type OfflineSessionClient struct {
	config
}

// NewOfflineSessionClient returns a client for the OfflineSession from the given config.
func NewOfflineSessionClient(c config) *OfflineSessionClient {
	return &OfflineSessionClient{config: c}
}

// Use adds a list of mutation hooks to the hooks stack.
// A call to `Use(f, g, h)` equals to `offlinesession.Hooks(f(g(h())))`.
func (c *OfflineSessionClient) Use(hooks ...Hook) {
	c.hooks.OfflineSession = append(c.hooks.OfflineSession, hooks...)
}

// Create returns a create builder for OfflineSession.
func (c *OfflineSessionClient) Create() *OfflineSessionCreate {
	mutation := newOfflineSessionMutation(c.config, OpCreate)
	return &OfflineSessionCreate{config: c.config, hooks: c.Hooks(), mutation: mutation}
}

// CreateBulk returns a builder for creating a bulk of OfflineSession entities.
func (c *OfflineSessionClient) CreateBulk(builders ...*OfflineSessionCreate) *OfflineSessionCreateBulk {
	return &OfflineSessionCreateBulk{config: c.config, builders: builders}
}

// Update returns an update builder for OfflineSession.
func (c *OfflineSessionClient) Update() *OfflineSessionUpdate {
	mutation := newOfflineSessionMutation(c.config, OpUpdate)
	return &OfflineSessionUpdate{config: c.config, hooks: c.Hooks(), mutation: mutation}
}

// UpdateOne returns an update builder for the given entity.
func (c *OfflineSessionClient) UpdateOne(os *OfflineSession) *OfflineSessionUpdateOne {
	mutation := newOfflineSessionMutation(c.config, OpUpdateOne, withOfflineSession(os))
	return &OfflineSessionUpdateOne{config: c.config, hooks: c.Hooks(), mutation: mutation}
}

// UpdateOneID returns an update builder for the given id.
func (c *OfflineSessionClient) UpdateOneID(id string) *OfflineSessionUpdateOne {
	mutation := newOfflineSessionMutation(c.config, OpUpdateOne, withOfflineSessionID(id))
	return &OfflineSessionUpdateOne{config: c.config, hooks: c.Hooks(), mutation: mutation}
}

// Delete returns a delete builder for OfflineSession.
func (c *OfflineSessionClient) Delete() *OfflineSessionDelete {
	mutation := newOfflineSessionMutation(c.config, OpDelete)
	return &OfflineSessionDelete{config: c.config, hooks: c.Hooks(), mutation: mutation}
}

// DeleteOne returns a delete builder for the given entity.
func (c *OfflineSessionClient) DeleteOne(os *OfflineSession) *OfflineSessionDeleteOne {
	return c.DeleteOneID(os.ID)
}

// DeleteOneID returns a delete builder for the given id.
func (c *OfflineSessionClient) DeleteOneID(id string) *OfflineSessionDeleteOne {
	builder := c.Delete().Where(offlinesession.ID(id))
	builder.mutation.id = &id
	builder.mutation.op = OpDeleteOne
	return &OfflineSessionDeleteOne{builder}
}

// Query returns a query builder for OfflineSession.
func (c *OfflineSessionClient) Query() *OfflineSessionQuery {
	return &OfflineSessionQuery{
		config: c.config,
	}
}

// Get returns a OfflineSession entity by its id.
func (c *OfflineSessionClient) Get(ctx context.Context, id string) (*OfflineSession, error) {
	return c.Query().Where(offlinesession.ID(id)).Only(ctx)
}

// GetX is like Get, but panics if an error occurs.
func (c *OfflineSessionClient) GetX(ctx context.Context, id string) *OfflineSession {
	obj, err := c.Get(ctx, id)
	if err != nil {
		panic(err)
	}
	return obj
}

// Hooks returns the client hooks.
func (c *OfflineSessionClient) Hooks() []Hook {
	return c.hooks.OfflineSession
}

// PasswordClient is a client for the Password schema.
type PasswordClient struct {
	config
}

// NewPasswordClient returns a client for the Password from the given config.
func NewPasswordClient(c config) *PasswordClient {
	return &PasswordClient{config: c}
}

// Use adds a list of mutation hooks to the hooks stack.
// A call to `Use(f, g, h)` equals to `password.Hooks(f(g(h())))`.
func (c *PasswordClient) Use(hooks ...Hook) {
	c.hooks.Password = append(c.hooks.Password, hooks...)
}

// Create returns a create builder for Password.
func (c *PasswordClient) Create() *PasswordCreate {
	mutation := newPasswordMutation(c.config, OpCreate)
	return &PasswordCreate{config: c.config, hooks: c.Hooks(), mutation: mutation}
}

// CreateBulk returns a builder for creating a bulk of Password entities.
func (c *PasswordClient) CreateBulk(builders ...*PasswordCreate) *PasswordCreateBulk {
	return &PasswordCreateBulk{config: c.config, builders: builders}
}

// Update returns an update builder for Password.
func (c *PasswordClient) Update() *PasswordUpdate {
	mutation := newPasswordMutation(c.config, OpUpdate)
	return &PasswordUpdate{config: c.config, hooks: c.Hooks(), mutation: mutation}
}

// UpdateOne returns an update builder for the given entity.
func (c *PasswordClient) UpdateOne(pa *Password) *PasswordUpdateOne {
	mutation := newPasswordMutation(c.config, OpUpdateOne, withPassword(pa))
	return &PasswordUpdateOne{config: c.config, hooks: c.Hooks(), mutation: mutation}
}

// UpdateOneID returns an update builder for the given id.
func (c *PasswordClient) UpdateOneID(id int) *PasswordUpdateOne {
	mutation := newPasswordMutation(c.config, OpUpdateOne, withPasswordID(id))
	return &PasswordUpdateOne{config: c.config, hooks: c.Hooks(), mutation: mutation}
}

// Delete returns a delete builder for Password.
func (c *PasswordClient) Delete() *PasswordDelete {
	mutation := newPasswordMutation(c.config, OpDelete)
	return &PasswordDelete{config: c.config, hooks: c.Hooks(), mutation: mutation}
}

// DeleteOne returns a delete builder for the given entity.
func (c *PasswordClient) DeleteOne(pa *Password) *PasswordDeleteOne {
	return c.DeleteOneID(pa.ID)
}

// DeleteOneID returns a delete builder for the given id.
func (c *PasswordClient) DeleteOneID(id int) *PasswordDeleteOne {
	builder := c.Delete().Where(password.ID(id))
	builder.mutation.id = &id
	builder.mutation.op = OpDeleteOne
	return &PasswordDeleteOne{builder}
}

// Query returns a query builder for Password.
func (c *PasswordClient) Query() *PasswordQuery {
	return &PasswordQuery{
		config: c.config,
	}
}

// Get returns a Password entity by its id.
func (c *PasswordClient) Get(ctx context.Context, id int) (*Password, error) {
	return c.Query().Where(password.ID(id)).Only(ctx)
}

// GetX is like Get, but panics if an error occurs.
func (c *PasswordClient) GetX(ctx context.Context, id int) *Password {
	obj, err := c.Get(ctx, id)
	if err != nil {
		panic(err)
	}
	return obj
}

// Hooks returns the client hooks.
func (c *PasswordClient) Hooks() []Hook {
	return c.hooks.Password
}

// RefreshTokenClient is a client for the RefreshToken schema.
type RefreshTokenClient struct {
	config
}

// NewRefreshTokenClient returns a client for the RefreshToken from the given config.
func NewRefreshTokenClient(c config) *RefreshTokenClient {
	return &RefreshTokenClient{config: c}
}

// Use adds a list of mutation hooks to the hooks stack.
// A call to `Use(f, g, h)` equals to `refreshtoken.Hooks(f(g(h())))`.
func (c *RefreshTokenClient) Use(hooks ...Hook) {
	c.hooks.RefreshToken = append(c.hooks.RefreshToken, hooks...)
}

// Create returns a create builder for RefreshToken.
func (c *RefreshTokenClient) Create() *RefreshTokenCreate {
	mutation := newRefreshTokenMutation(c.config, OpCreate)
	return &RefreshTokenCreate{config: c.config, hooks: c.Hooks(), mutation: mutation}
}

// CreateBulk returns a builder for creating a bulk of RefreshToken entities.
func (c *RefreshTokenClient) CreateBulk(builders ...*RefreshTokenCreate) *RefreshTokenCreateBulk {
	return &RefreshTokenCreateBulk{config: c.config, builders: builders}
}

// Update returns an update builder for RefreshToken.
func (c *RefreshTokenClient) Update() *RefreshTokenUpdate {
	mutation := newRefreshTokenMutation(c.config, OpUpdate)
	return &RefreshTokenUpdate{config: c.config, hooks: c.Hooks(), mutation: mutation}
}

// UpdateOne returns an update builder for the given entity.
func (c *RefreshTokenClient) UpdateOne(rt *RefreshToken) *RefreshTokenUpdateOne {
	mutation := newRefreshTokenMutation(c.config, OpUpdateOne, withRefreshToken(rt))
	return &RefreshTokenUpdateOne{config: c.config, hooks: c.Hooks(), mutation: mutation}
}

// UpdateOneID returns an update builder for the given id.
func (c *RefreshTokenClient) UpdateOneID(id string) *RefreshTokenUpdateOne {
	mutation := newRefreshTokenMutation(c.config, OpUpdateOne, withRefreshTokenID(id))
	return &RefreshTokenUpdateOne{config: c.config, hooks: c.Hooks(), mutation: mutation}
}

// Delete returns a delete builder for RefreshToken.
func (c *RefreshTokenClient) Delete() *RefreshTokenDelete {
	mutation := newRefreshTokenMutation(c.config, OpDelete)
	return &RefreshTokenDelete{config: c.config, hooks: c.Hooks(), mutation: mutation}
}

// DeleteOne returns a delete builder for the given entity.
func (c *RefreshTokenClient) DeleteOne(rt *RefreshToken) *RefreshTokenDeleteOne {
	return c.DeleteOneID(rt.ID)
}

// DeleteOneID returns a delete builder for the given id.
func (c *RefreshTokenClient) DeleteOneID(id string) *RefreshTokenDeleteOne {
	builder := c.Delete().Where(refreshtoken.ID(id))
	builder.mutation.id = &id
	builder.mutation.op = OpDeleteOne
	return &RefreshTokenDeleteOne{builder}
}

// Query returns a query builder for RefreshToken.
func (c *RefreshTokenClient) Query() *RefreshTokenQuery {
	return &RefreshTokenQuery{
		config: c.config,
	}
}

// Get returns a RefreshToken entity by its id.
func (c *RefreshTokenClient) Get(ctx context.Context, id string) (*RefreshToken, error) {
	return c.Query().Where(refreshtoken.ID(id)).Only(ctx)
}

// GetX is like Get, but panics if an error occurs.
func (c *RefreshTokenClient) GetX(ctx context.Context, id string) *RefreshToken {
	obj, err := c.Get(ctx, id)
	if err != nil {
		panic(err)
	}
	return obj
}

// Hooks returns the client hooks.
func (c *RefreshTokenClient) Hooks() []Hook {
	return c.hooks.RefreshToken
}
