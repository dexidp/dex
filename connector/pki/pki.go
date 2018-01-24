package pki

import (
	"context"
	"encoding/json"
	"fmt"
	"github.com/coreos/dex/connector"
	"github.com/sirupsen/logrus"
	"github.com/coreos/dex/pkiutil"
)

// Config holds configuration options for pki logins.
type Config struct {
	// RedirectURI string `json:"redirectURI"`
}

// Open returns a strategy for logging in through GitHub.
func (c *Config) Open(id string, logger logrus.FieldLogger) (connector.Connector, error) {
	p := pkiConnector{
		logger: logger,
	}

	return &p, nil
}

type connectorData struct {
	// GitHub's OAuth2 tokens never expire. We don't need a refresh token.
	AccessToken string `json:"accessToken"`
}

var (
	_ connector.PasswordConnector = (*pkiConnector)(nil)
	_ connector.RefreshConnector  = (*pkiConnector)(nil)
)

type pkiConnector struct {
	logger logrus.FieldLogger
}

type refreshData struct {
	DN string `json:"dn"`
}

func (c *pkiConnector) Prompt() string {
	return "PROMPT"
}

func (c *pkiConnector) Login(ctx context.Context, s connector.Scopes, username, password string) (identity connector.Identity, validPass bool, err error) {
	dn, ok := pkiutil.DistinguishedNameFromContext(ctx)
	if !ok {
		return connector.Identity{}, false, fmt.Errorf("pki: no peer certificate found")
	}

	identity.UserID = dn
	identity.UserDN = dn

	if s.OfflineAccess {
		refresh := refreshData{
			DN: dn,
		}
		if identity.ConnectorData, err = json.Marshal(refresh); err != nil {
			return connector.Identity{}, false, fmt.Errorf("pki: marshal entry: %v", err)
		}
	}

	return identity, true, nil
}

func (c *pkiConnector) Refresh(ctx context.Context, s connector.Scopes, identity connector.Identity) (connector.Identity, error) {
	dn, ok := pkiutil.DistinguishedNameFromContext(ctx)
	if !ok {
		return connector.Identity{}, fmt.Errorf("pki: no peer certificate found")
	}

	var data refreshData
	if err := json.Unmarshal(identity.ConnectorData, &data); err != nil {
		return connector.Identity{}, fmt.Errorf("pki: failed to unamrshal internal data: %v", err)
	}

	if data.DN != dn {
		return connector.Identity{}, fmt.Errorf("pki: refresh expected DN %q but got %q", data.DN, dn)
	}

	//identity.Email = TODO (???)
	identity.UserID = dn
	identity.UserDN = dn

	return identity, nil
}
