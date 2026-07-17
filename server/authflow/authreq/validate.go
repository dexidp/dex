package authreq

import (
	"net"
	"net/url"

	"github.com/dexidp/dex/server/oauth2"
	"github.com/dexidp/dex/server/tokens"
	"github.com/dexidp/dex/storage"
)

func validateRedirectURI(client storage.Client, redirectURI string) bool {
	// Allow named RedirectURIs for both public and non-public clients.
	// This is required to make PKCE-enabled web apps work when configured as public clients.
	for _, uri := range client.RedirectURIs {
		if redirectURI == uri {
			return true
		}
	}
	// For non-public clients or when RedirectURIs is set, we allow only explicitly named RedirectURIs.
	if !client.Public || len(client.RedirectURIs) > 0 {
		return false
	}

	if redirectURI == oauth2.RedirectURIOOB || redirectURI == oauth2.DeviceCallbackURI {
		return true
	}

	// Verify the host is a loopback form ("http://localhost:(port)(path)" etc).
	u, err := url.Parse(redirectURI)
	if err != nil {
		return false
	}
	if u.Scheme != "http" {
		return false
	}
	return isHostLocal(u.Host)
}

func isHostLocal(host string) bool {
	if host == "localhost" || net.ParseIP(host).IsLoopback() {
		return true
	}

	host, _, err := net.SplitHostPort(host)
	if err != nil {
		return false
	}

	return host == "localhost" || net.ParseIP(host).IsLoopback()
}

func validateConnectorID(connectors []storage.Connector, connectorID string) bool {
	for _, c := range connectors {
		if c.ID == connectorID {
			return true
		}
	}
	return false
}

// SessionMatchesHint checks whether the session's user identity matches the
// subject from an id_token_hint by encoding the session's (userID, connectorID)
// via GenSubject and doing a string comparison.
func SessionMatchesHint(session *storage.AuthSession, hintSubject string) bool {
	if session == nil {
		return false
	}
	encoded, err := tokens.GenSubject(session.UserID, session.ConnectorID)
	if err != nil {
		return false
	}
	return encoded == hintSubject
}
