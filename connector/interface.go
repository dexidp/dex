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
	ID() string
	LoginURL(sessionKey, prompt string) (string, error)
	Register(mux *http.ServeMux, errorURL url.URL)

	// Sync triggers any long-running tasks needed to maintain the
	// Connector's operation. For example, this would encompass
	// repeatedly caching any remote resources for local use.
	Sync() chan struct{}

	// TrustedEmailProvider indicates whether or not we can trust that email claims coming from this provider.
	TrustedEmailProvider() bool

	health.Checkable
}

//go:generate genconfig -o config.go connector Connector
type ConnectorConfig interface {
	ConnectorID() string
	ConnectorType() string
	Connector(ns url.URL, loginFunc oidc.LoginFunc, tpls *template.Template) (Connector, error)
}

type ConnectorConfigRepo interface {
	All() ([]ConnectorConfig, error)
	GetConnectorByID(repo.Transaction, string) (ConnectorConfig, error)
}
