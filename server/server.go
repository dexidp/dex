package server

import (
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"path"
	"time"

	"github.com/gorilla/mux"

	"github.com/coreos/poke/connector"
	"github.com/coreos/poke/storage"
)

// Connector is a connector with metadata.
type Connector struct {
	ID          string
	DisplayName string
	Connector   connector.Connector
}

// Config holds the server's configuration options.
type Config struct {
	Issuer string

	// The backing persistence layer.
	Storage storage.Storage

	// Strategies for federated identity.
	Connectors []Connector

	// NOTE: Multiple servers using the same storage are expected to set rotation and
	// validity periods to the same values.
	RotateKeysAfter  time.Duration // Defaults to 6 hours.
	IDTokensValidFor time.Duration // Defaults to 24 hours

	// If specified, the server will use this function for determining time.
	Now func() time.Time
}

func value(val, defaultValue time.Duration) time.Duration {
	if val == 0 {
		return defaultValue
	}
	return val
}

// Server is the top level object.
type Server struct {
	issuerURL url.URL

	// Read-only map of connector IDs to connectors.
	connectors map[string]Connector

	storage storage.Storage

	mux http.Handler

	// If enabled, don't prompt user for approval after logging in through connector.
	// No package level API to set this, only used in tests.
	skipApproval bool

	now func() time.Time

	idTokensValidFor time.Duration
}

// New constructs a server from the provided config.
func New(c Config) (*Server, error) {
	return newServer(c, defaultRotationStrategy(
		value(c.RotateKeysAfter, 6*time.Hour),
		value(c.IDTokensValidFor, 24*time.Hour),
	))
}

func newServer(c Config, rotationStrategy rotationStrategy) (*Server, error) {
	issuerURL, err := url.Parse(c.Issuer)
	if err != nil {
		return nil, fmt.Errorf("server: can't parse issuer URL")
	}
	if len(c.Connectors) == 0 {
		return nil, errors.New("server: no connectors specified")
	}
	if c.Storage == nil {
		return nil, errors.New("server: storage cannot be nil")
	}

	now := c.Now
	if now == nil {
		now = time.Now
	}

	s := &Server{
		issuerURL:        *issuerURL,
		connectors:       make(map[string]Connector),
		storage:          storageWithKeyRotation(c.Storage, rotationStrategy, now),
		idTokensValidFor: value(c.IDTokensValidFor, 24*time.Hour),
		now:              now,
	}

	for _, conn := range c.Connectors {
		s.connectors[conn.ID] = conn
	}

	r := mux.NewRouter()
	handleFunc := func(p string, h http.HandlerFunc) {
		r.HandleFunc(path.Join(issuerURL.Path, p), h)
	}
	r.NotFoundHandler = http.HandlerFunc(s.notFound)

	// TODO(ericchiang): rate limit certain paths based on IP.
	handleFunc("/.well-known/openid-configuration", s.handleDiscovery)
	handleFunc("/token", s.handleToken)
	handleFunc("/keys", s.handlePublicKeys)
	handleFunc("/auth", s.handleAuthorization)
	handleFunc("/auth/{connector}", s.handleConnectorLogin)
	handleFunc("/callback/{connector}", s.handleConnectorCallback)
	handleFunc("/approval", s.handleApproval)
	s.mux = r

	return s, nil
}

func (s *Server) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	s.mux.ServeHTTP(w, r)
}

func (s *Server) absPath(pathItems ...string) string {
	paths := make([]string, len(pathItems)+1)
	paths[0] = s.issuerURL.Path
	copy(paths[1:], pathItems)
	return path.Join(paths...)
}

func (s *Server) absURL(pathItems ...string) string {
	u := s.issuerURL
	u.Path = s.absPath(pathItems...)
	return u.String()
}
