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

// ConnectorsConfig maps each built-in connector type to its config factory. It
// is handed to connectors.Resolver so the connectors package itself imports no
// connector implementation; a library consumer can pass a different map.
var ConnectorsConfig = map[string]func() connectors.ConnectorConfig{
	"keystone":        func() connectors.ConnectorConfig { return new(keystone.Config) },
	"mockCallback":    func() connectors.ConnectorConfig { return new(mock.CallbackConfig) },
	"mockPassword":    func() connectors.ConnectorConfig { return new(mock.PasswordConfig) },
	"ldap":            func() connectors.ConnectorConfig { return new(ldap.Config) },
	"gitea":           func() connectors.ConnectorConfig { return new(gitea.Config) },
	"github":          func() connectors.ConnectorConfig { return new(github.Config) },
	"gitlab":          func() connectors.ConnectorConfig { return new(gitlab.Config) },
	"google":          func() connectors.ConnectorConfig { return new(google.Config) },
	"oidc":            func() connectors.ConnectorConfig { return new(oidc.Config) },
	"oauth":           func() connectors.ConnectorConfig { return new(oauth.Config) },
	"saml":            func() connectors.ConnectorConfig { return new(saml.Config) },
	"authproxy":       func() connectors.ConnectorConfig { return new(authproxy.Config) },
	"linkedin":        func() connectors.ConnectorConfig { return new(linkedin.Config) },
	"microsoft":       func() connectors.ConnectorConfig { return new(microsoft.Config) },
	"bitbucket-cloud": func() connectors.ConnectorConfig { return new(bitbucketcloud.Config) },
	"openshift":       func() connectors.ConnectorConfig { return new(openshift.Config) },
	"atlassian-crowd": func() connectors.ConnectorConfig { return new(atlassiancrowd.Config) },
	// Keep around for backwards compatibility.
	"samlExperimental": func() connectors.ConnectorConfig { return new(saml.Config) },
}
