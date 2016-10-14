package server

import (
	"errors"
	"fmt"
	"log"
	"net/http"
	"net/url"
	"path"
	"sync/atomic"
	"time"

	"golang.org/x/crypto/bcrypt"
	"golang.org/x/net/context"

	"github.com/gorilla/mux"

	"github.com/coreos/dex/connector"
	"github.com/coreos/dex/storage"
)

// Connector is a connector with metadata.
type Connector struct {
	ID          string
	DisplayName string
	Connector   connector.Connector
}

// Config holds the server's configuration options.
//
// Multiple servers using the same storage are expected to be configured identically.
type Config struct {
	Issuer string

	// The backing persistence layer.
	Storage storage.Storage

	// Strategies for federated identity.
	Connectors []Connector

	// Valid values are "code" to enable the code flow and "token" to enable the implicit
	// flow. If no response types are supplied this value defaults to "code".
	SupportedResponseTypes []string

	// If enabled, the server won't prompt the user to approve authorization requests.
	// Logging in implies approval.
	SkipApprovalScreen bool

	RotateKeysAfter  time.Duration // Defaults to 6 hours.
	IDTokensValidFor time.Duration // Defaults to 24 hours

	GCFrequency time.Duration // Defaults to 5 minutes

	// If specified, the server will use this function for determining time.
	Now func() time.Time

	EnablePasswordDB bool

	TemplateConfig TemplateConfig
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

	templates *templates

	// If enabled, don't prompt user for approval after logging in through connector.
	skipApproval bool

	supportedResponseTypes map[string]bool

	now func() time.Time

	idTokensValidFor time.Duration
}

// NewServer constructs a server from the provided config.
func NewServer(ctx context.Context, c Config) (*Server, error) {
	return newServer(ctx, c, defaultRotationStrategy(
		value(c.RotateKeysAfter, 6*time.Hour),
		value(c.IDTokensValidFor, 24*time.Hour),
	))
}

func newServer(ctx context.Context, c Config, rotationStrategy rotationStrategy) (*Server, error) {
	issuerURL, err := url.Parse(c.Issuer)
	if err != nil {
		return nil, fmt.Errorf("server: can't parse issuer URL")
	}
	if c.EnablePasswordDB {
		c.Connectors = append(c.Connectors, Connector{
			ID:          "local",
			DisplayName: "Email",
			Connector:   newPasswordDB(c.Storage),
		})
	}

	if len(c.Connectors) == 0 {
		return nil, errors.New("server: no connectors specified")
	}
	if c.Storage == nil {
		return nil, errors.New("server: storage cannot be nil")
	}
	if len(c.SupportedResponseTypes) == 0 {
		c.SupportedResponseTypes = []string{responseTypeCode}
	}

	supported := make(map[string]bool)
	for _, respType := range c.SupportedResponseTypes {
		switch respType {
		case responseTypeCode, responseTypeToken:
		default:
			return nil, fmt.Errorf("unsupported response_type %q", respType)
		}
		supported[respType] = true
	}

	tmpls, err := loadTemplates(c.TemplateConfig)
	if err != nil {
		return nil, fmt.Errorf("server: failed to load templates: %v", err)
	}

	now := c.Now
	if now == nil {
		now = time.Now
	}

	s := &Server{
		issuerURL:              *issuerURL,
		connectors:             make(map[string]Connector),
		storage:                newKeyCacher(c.Storage, now),
		supportedResponseTypes: supported,
		idTokensValidFor:       value(c.IDTokensValidFor, 24*time.Hour),
		skipApproval:           c.SkipApprovalScreen,
		now:                    now,
		templates:              tmpls,
	}

	for _, conn := range c.Connectors {
		s.connectors[conn.ID] = conn
	}

	r := mux.NewRouter()
	handleFunc := func(p string, h http.HandlerFunc) {
		r.HandleFunc(path.Join(issuerURL.Path, p), h)
	}
	r.NotFoundHandler = http.HandlerFunc(s.notFound)

	discoveryHandler, err := s.discoveryHandler()
	if err != nil {
		return nil, err
	}
	handleFunc("/.well-known/openid-configuration", discoveryHandler)

	// TODO(ericchiang): rate limit certain paths based on IP.
	handleFunc("/token", s.handleToken)
	handleFunc("/keys", s.handlePublicKeys)
	handleFunc("/auth", s.handleAuthorization)
	handleFunc("/auth/{connector}", s.handleConnectorLogin)
	handleFunc("/callback/{connector}", s.handleConnectorCallback)
	handleFunc("/approval", s.handleApproval)
	handleFunc("/healthz", s.handleHealth)
	s.mux = r

	startKeyRotation(ctx, c.Storage, rotationStrategy, now)
	startGarbageCollection(ctx, c.Storage, value(c.GCFrequency, 5*time.Minute), now)

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

func newPasswordDB(s storage.Storage) interface {
	connector.Connector
	connector.PasswordConnector
} {
	return passwordDB{s}
}

type passwordDB struct {
	s storage.Storage
}

func (db passwordDB) Close() error { return nil }

func (db passwordDB) Login(email, password string) (connector.Identity, bool, error) {
	p, err := db.s.GetPassword(email)
	if err != nil {
		if err != storage.ErrNotFound {
			log.Printf("get password: %v", err)
		}
		return connector.Identity{}, false, err
	}
	if err := bcrypt.CompareHashAndPassword(p.Hash, []byte(password)); err != nil {
		return connector.Identity{}, false, nil
	}
	return connector.Identity{
		UserID:        p.UserID,
		Username:      p.Username,
		Email:         p.Email,
		EmailVerified: true,
	}, true, nil
}

// newKeyCacher returns a storage which caches keys so long as the next
func newKeyCacher(s storage.Storage, now func() time.Time) storage.Storage {
	if now == nil {
		now = time.Now
	}
	return &keyCacher{Storage: s, now: now}
}

type keyCacher struct {
	storage.Storage

	now  func() time.Time
	keys atomic.Value // Always holds nil or type *storage.Keys.
}

func (k *keyCacher) GetKeys() (storage.Keys, error) {
	keys, ok := k.keys.Load().(*storage.Keys)
	if ok && keys != nil && k.now().Before(keys.NextRotation) {
		return *keys, nil
	}

	storageKeys, err := k.Storage.GetKeys()
	if err != nil {
		return storageKeys, err
	}

	if k.now().Before(storageKeys.NextRotation) {
		k.keys.Store(&storageKeys)
	}
	return storageKeys, nil
}

func startGarbageCollection(ctx context.Context, s storage.Storage, frequency time.Duration, now func() time.Time) {
	go func() {
		for {
			select {
			case <-ctx.Done():
				return
			case <-time.After(frequency):
				if r, err := s.GarbageCollect(now()); err != nil {
					log.Printf("garbage collection failed: %v", err)
				} else if r.AuthRequests > 0 || r.AuthCodes > 0 {
					log.Printf("garbage collection run, delete auth requests=%d, auth codes=%d", r.AuthRequests, r.AuthCodes)
				}
			}
		}
	}()
	return
}
