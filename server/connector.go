package server

import (
	"github.com/dexidp/dex/connector/atlassiancrowd"
	"github.com/dexidp/dex/connector/authproxy"
	"github.com/dexidp/dex/connector/bitbucketcloud"
	"github.com/dexidp/dex/connector/gitea"
	"github.com/dexidp/dex/connector/github"
	"github.com/dexidp/dex/connector/gitlab"
	"github.com/dexidp/dex/connector/google"
	"github.com/dexidp/dex/connector/keystone"
	"github.com/dexidp/dex/connector/ldap"
	"github.com/dexidp/dex/connector/linkedin"
	"github.com/dexidp/dex/connector/microsoft"
	"github.com/dexidp/dex/connector/mock"
	"github.com/dexidp/dex/connector/oauth"
	"github.com/dexidp/dex/connector/oidc"
	"github.com/dexidp/dex/connector/openshift"
	"github.com/dexidp/dex/connector/saml"
	"github.com/dexidp/dex/server/connectors"
)

// ConnectorConfig is a configuration that can open a connector. The resolution
// machinery lives in the connectors package; this alias is kept for callers of
// the server package.
type ConnectorConfig = connectors.ConnectorConfig

// LocalConnector is the built-in local password-DB connector type, re-exported
// from the connectors package.
const LocalConnector = connectors.LocalConnector

// ConnectorsConfig maps each built-in connector type to its config factory. It
// is handed to connectors.Resolver so the connectors package itself imports no
// connector implementation; a library consumer can pass a different map.
var ConnectorsConfig = map[string]func() ConnectorConfig{
	"keystone":        func() ConnectorConfig { return new(keystone.Config) },
	"mockCallback":    func() ConnectorConfig { return new(mock.CallbackConfig) },
	"mockPassword":    func() ConnectorConfig { return new(mock.PasswordConfig) },
	"ldap":            func() ConnectorConfig { return new(ldap.Config) },
	"gitea":           func() ConnectorConfig { return new(gitea.Config) },
	"github":          func() ConnectorConfig { return new(github.Config) },
	"gitlab":          func() ConnectorConfig { return new(gitlab.Config) },
	"google":          func() ConnectorConfig { return new(google.Config) },
	"oidc":            func() ConnectorConfig { return new(oidc.Config) },
	"oauth":           func() ConnectorConfig { return new(oauth.Config) },
	"saml":            func() ConnectorConfig { return new(saml.Config) },
	"authproxy":       func() ConnectorConfig { return new(authproxy.Config) },
	"linkedin":        func() ConnectorConfig { return new(linkedin.Config) },
	"microsoft":       func() ConnectorConfig { return new(microsoft.Config) },
	"bitbucket-cloud": func() ConnectorConfig { return new(bitbucketcloud.Config) },
	"openshift":       func() ConnectorConfig { return new(openshift.Config) },
	"atlassian-crowd": func() ConnectorConfig { return new(atlassiancrowd.Config) },
	// Keep around for backwards compatibility.
	"samlExperimental": func() ConnectorConfig { return new(saml.Config) },
}
