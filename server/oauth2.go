package server

import (
	"context"

	"github.com/dexidp/dex/connector"
	"github.com/dexidp/dex/server/tokens"
	"github.com/dexidp/dex/storage"
)

const (
	errInvalidRequest          = "invalid_request"
	errUnauthorizedClient      = "unauthorized_client"
	errAccessDenied            = "access_denied"
	errUnsupportedResponseType = "unsupported_response_type"
	errRequestNotSupported     = "request_not_supported"
	errInvalidScope            = "invalid_scope"
	errServerError             = "server_error"
	errTemporarilyUnavailable  = "temporarily_unavailable"
	errUnsupportedGrantType    = "unsupported_grant_type"
	errInvalidGrant            = "invalid_grant"
	errInvalidClient           = "invalid_client"
	errInactiveToken           = "inactive_token"
	errLoginRequired           = "login_required"
	errInteractionRequired     = "interaction_required"
	errConsentRequired         = "consent_required"
)

const (
	deviceCallbackURI = "/device/callback"
)

const (
	redirectURIOOB = "urn:ietf:wg:oauth:2.0:oob"
)

const (
	grantTypeAuthorizationCode = "authorization_code"
	grantTypeRefreshToken      = "refresh_token"
	grantTypeImplicit          = "implicit"
	grantTypePassword          = "password"
	grantTypeDeviceCode        = "urn:ietf:params:oauth:grant-type:device_code"
	grantTypeTokenExchange     = "urn:ietf:params:oauth:grant-type:token-exchange"
	grantTypeClientCredentials = "client_credentials"
)

// ConnectorGrantTypes is the set of grant types that can be restricted per connector.
var ConnectorGrantTypes = map[string]bool{
	grantTypeAuthorizationCode: true,
	grantTypeRefreshToken:      true,
	grantTypeImplicit:          true,
	grantTypePassword:          true,
	grantTypeDeviceCode:        true,
	grantTypeTokenExchange:     true,
}

const (
	// https://www.rfc-editor.org/rfc/rfc8693.html#section-3
	tokenTypeAccess  = "urn:ietf:params:oauth:token-type:access_token"
	tokenTypeRefresh = "urn:ietf:params:oauth:token-type:refresh_token"
	tokenTypeID      = "urn:ietf:params:oauth:token-type:id_token"
	tokenTypeSAML1   = "urn:ietf:params:oauth:token-type:saml1"
	tokenTypeSAML2   = "urn:ietf:params:oauth:token-type:saml2"
	tokenTypeJWT     = "urn:ietf:params:oauth:token-type:jwt"
)

const (
	responseTypeCode             = "code"                // "Regular" flow
	responseTypeToken            = "token"               // Implicit flow for frontend apps.
	responseTypeIDToken          = "id_token"            // ID Token in url fragment
	responseTypeCodeToken        = "code token"          // "Regular" flow + Implicit flow
	responseTypeCodeIDToken      = "code id_token"       // "Regular" flow + ID Token
	responseTypeIDTokenToken     = "id_token token"      // ID Token + Implicit flow
	responseTypeCodeIDTokenToken = "code id_token token" // "Regular" flow + ID Token + Implicit flow
)

const (
	deviceTokenPending  = "authorization_pending"
	deviceTokenComplete = "complete"
	deviceTokenSlowDown = "slow_down"
	deviceTokenExpired  = "expired_token"
)

func parseScopes(scopes []string) connector.Scopes {
	var s connector.Scopes
	for _, scope := range scopes {
		switch scope {
		case tokens.ScopeOfflineAccess:
			s.OfflineAccess = true
		case tokens.ScopeGroups:
			s.Groups = true
		}
	}
	return s
}

func (s *Server) validateCrossClientTrust(ctx context.Context, clientID, peerID string) (trusted bool, err error) {
	if peerID == clientID {
		return true, nil
	}
	peer, err := s.storage.GetClient(ctx, peerID)
	if err != nil {
		if err != storage.ErrNotFound {
			s.logger.ErrorContext(ctx, "failed to get client", "err", err)
			return false, err
		}
		return false, nil
	}
	for _, id := range peer.TrustedPeers {
		if id == clientID {
			return true, nil
		}
	}
	return false, nil
}
