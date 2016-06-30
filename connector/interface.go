package connector

import (
	"errors"
	"html/template"
	"net/http"
	"net/url"

	"github.com/coreos/dex/repo"
	"github.com/coreos/go-oidc/oidc"
	"github.com/coreos/pkg/health"
)

var ErrorNotFound = errors.New("connector not found in repository")

type Connector interface {
	// ID returns the ID of the ConnectorConfig used to create the Connector.
	ID() string

	// LoginURL returns the backend's authorization URL for a sessionKey
	// and OAuth2 prompt type.
	LoginURL(sessionKey, prompt string) (string, error)

	// Register allows connectors to register a callback handler with the
	// dex server.
	//
	// Connectors should register with a path that extends the namespace
	// URL provided when the Connector is instantiated.
	Register(mux *http.ServeMux, errorURL url.URL)

	// Sync triggers any long-running tasks needed to maintain the
	// Connector's operation. For example, this would encompass
	// repeatedly caching any remote resources for local use.
	Sync() chan struct{}

	// TrustedEmailProvider indicates whether or not we can trust that email
	// claims coming from this provider.
	TrustedEmailProvider() bool

	health.Checkable
}

type NewSessionFunc func(ipdcID, clientID, clientState string, redirectURL url.URL, nonce string, register bool, scope []string) (string, error)

//go:generate genconfig -o config.go connector Connector
type ConnectorConfig interface {
	// ConnectorID returns a unique end user facing identifier. For example "google".
	ConnectorID() string

	// ConnectorType returns an implementation specific identifier. For example "oidc".
	ConnectorType() string

	// Connector is invoked by the dex server and returns a Connector configured
	// to use the provided arguments. URL namespace is used to register callbacks.
	// loginFunc is used to associate remote identies with dex session keys.
	//
	// The returned Connector must call loginFunc once upon successful
	// identification of a user.
	//
	// Additional templates are passed for connectors that require rendering HTML
	// pages, such as the "local" connector.
	Connector(ns url.URL, loginFunc oidc.LoginFunc, newSessionFunc NewSessionFunc, tpls *template.Template) (Connector, error)
}

type ConnectorConfigRepo interface {
	All() ([]ConnectorConfig, error)
	GetConnectorByID(repo.Transaction, string) (ConnectorConfig, error)
	Set(cfgs []ConnectorConfig) error
}
