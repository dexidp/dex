package client

import (
	"encoding/json"
	"strings"

	"github.com/dexidp/dex/storage"
	"github.com/dexidp/dex/storage/ent/db"
)

const keysRowID = "keys"

func toStorageKeys(keys *db.Keys) storage.Keys {
	return storage.Keys{
		SigningKey:       &keys.SigningKey,
		SigningKeyPub:    &keys.SigningKeyPub,
		VerificationKeys: keys.VerificationKeys,
		NextRotation:     keys.NextRotation,
	}
}

func toStorageAuthRequest(a *db.AuthRequest) storage.AuthRequest {
	return storage.AuthRequest{
		ID:                  a.ID,
		ClientID:            a.ClientID,
		ResponseTypes:       a.ResponseTypes,
		Scopes:              a.Scopes,
		RedirectURI:         a.RedirectURI,
		Nonce:               a.Nonce,
		State:               a.State,
		ForceApprovalPrompt: a.ForceApprovalPrompt,
		LoggedIn:            a.LoggedIn,
		ConnectorID:         a.ConnectorID,
		ConnectorData:       *a.ConnectorData,
		Expiry:              a.Expiry,
		Claims: storage.Claims{
			UserID:            a.ClaimsUserID,
			Username:          a.ClaimsUsername,
			PreferredUsername: a.ClaimsPreferredUsername,
			Email:             a.ClaimsEmail,
			EmailVerified:     a.ClaimsEmailVerified,
			Groups:            a.ClaimsGroups,
		},
		PKCE: storage.PKCE{
			CodeChallenge:       a.CodeChallenge,
			CodeChallengeMethod: a.CodeChallengeMethod,
		},
		HMACKey:      a.HmacKey,
		MFAValidated: a.MfaValidated,
	}
}

func toStorageAuthCode(a *db.AuthCode) storage.AuthCode {
	return storage.AuthCode{
		ID:            a.ID,
		ClientID:      a.ClientID,
		Scopes:        a.Scopes,
		RedirectURI:   a.RedirectURI,
		Nonce:         a.Nonce,
		ConnectorID:   a.ConnectorID,
		ConnectorData: *a.ConnectorData,
		Expiry:        a.Expiry,
		Claims: storage.Claims{
			UserID:            a.ClaimsUserID,
			Username:          a.ClaimsUsername,
			PreferredUsername: a.ClaimsPreferredUsername,
			Email:             a.ClaimsEmail,
			EmailVerified:     a.ClaimsEmailVerified,
			Groups:            a.ClaimsGroups,
		},
		PKCE: storage.PKCE{
			CodeChallenge:       a.CodeChallenge,
			CodeChallengeMethod: a.CodeChallengeMethod,
		},
	}
}

func toStorageClient(c *db.OAuth2Client) storage.Client {
	return storage.Client{
		ID:                c.ID,
		Secret:            c.Secret,
		RedirectURIs:      c.RedirectUris,
		TrustedPeers:      c.TrustedPeers,
		Public:            c.Public,
		Name:              c.Name,
		LogoURL:           c.LogoURL,
		AllowedConnectors: c.AllowedConnectors,
		MFAChain:          c.MfaChain,
	}
}

func toStorageConnector(c *db.Connector) storage.Connector {
	return storage.Connector{
		ID:         c.ID,
		Type:       c.Type,
		Name:       c.Name,
		Config:     c.Config,
		GrantTypes: c.GrantTypes,
	}
}

func toStorageOfflineSession(o *db.OfflineSession) storage.OfflineSessions {
	s := storage.OfflineSessions{
		UserID:        o.UserID,
		ConnID:        o.ConnID,
		ConnectorData: *o.ConnectorData,
	}

	if o.Refresh != nil {
		if err := json.Unmarshal(o.Refresh, &s.Refresh); err != nil {
			// Correctness of json structure if guaranteed on uploading
			panic(err)
		}
	} else {
		// Server code assumes this will be non-nil.
		s.Refresh = make(map[string]*storage.RefreshTokenRef)
	}
	return s
}

func toStorageRefreshToken(r *db.RefreshToken) storage.RefreshToken {
	return storage.RefreshToken{
		ID:            r.ID,
		Token:         r.Token,
		ObsoleteToken: r.ObsoleteToken,
		CreatedAt:     r.CreatedAt,
		LastUsed:      r.LastUsed,
		ClientID:      r.ClientID,
		ConnectorID:   r.ConnectorID,
		ConnectorData: *r.ConnectorData,
		Scopes:        r.Scopes,
		Nonce:         r.Nonce,
		Claims: storage.Claims{
			UserID:            r.ClaimsUserID,
			Username:          r.ClaimsUsername,
			PreferredUsername: r.ClaimsPreferredUsername,
			Email:             r.ClaimsEmail,
			EmailVerified:     r.ClaimsEmailVerified,
			Groups:            r.ClaimsGroups,
		},
	}
}

func toStoragePassword(p *db.Password) storage.Password {
	return storage.Password{
		Email:             p.Email,
		Hash:              p.Hash,
		Username:          p.Username,
		Name:              p.Name,
		PreferredUsername: p.PreferredUsername,
		EmailVerified:     p.EmailVerified,
		UserID:            p.UserID,
		Groups:            p.Groups,
	}
}

func toStorageDeviceRequest(r *db.DeviceRequest) storage.DeviceRequest {
	return storage.DeviceRequest{
		UserCode:     strings.ToUpper(r.UserCode),
		DeviceCode:   r.DeviceCode,
		ClientID:     r.ClientID,
		ClientSecret: r.ClientSecret,
		Scopes:       r.Scopes,
		Expiry:       r.Expiry,
	}
}

func toStorageUserIdentity(u *db.UserIdentity) storage.UserIdentity {
	s := storage.UserIdentity{
		UserID:      u.UserID,
		ConnectorID: u.ConnectorID,
		Claims: storage.Claims{
			UserID:            u.ClaimsUserID,
			Username:          u.ClaimsUsername,
			PreferredUsername: u.ClaimsPreferredUsername,
			Email:             u.ClaimsEmail,
			EmailVerified:     u.ClaimsEmailVerified,
			Groups:            u.ClaimsGroups,
		},
		CreatedAt:    u.CreatedAt,
		LastLogin:    u.LastLogin,
		BlockedUntil: u.BlockedUntil,
	}

	if u.Consents != nil {
		if err := json.Unmarshal(u.Consents, &s.Consents); err != nil {
			// Correctness of json structure is guaranteed on uploading
			panic(err)
		}
		if s.Consents == nil {
			// Ensure Consents is non-nil even if JSON was "null".
			s.Consents = make(map[string][]string)
		}
	} else {
		// Server code assumes this will be non-nil.
		s.Consents = make(map[string][]string)
	}

	if u.MfaSecrets != nil {
		if err := json.Unmarshal(*u.MfaSecrets, &s.MFASecrets); err != nil {
			// Correctness of json structure is guaranteed on uploading
			panic(err)
		}
		if s.MFASecrets == nil {
			s.MFASecrets = make(map[string]*storage.MFASecret)
		}
	} else {
		s.MFASecrets = make(map[string]*storage.MFASecret)
	}
	return s
}

func toStorageAuthSession(s *db.AuthSession) storage.AuthSession {
	result := storage.AuthSession{
		UserID:       s.UserID,
		ConnectorID:  s.ConnectorID,
		Nonce:        s.Nonce,
		CreatedAt:    s.CreatedAt,
		LastActivity: s.LastActivity,
		IPAddress:    s.IPAddress,
		UserAgent:    s.UserAgent,
	}

	if s.ClientStates != nil {
		if err := json.Unmarshal(s.ClientStates, &result.ClientStates); err != nil {
			panic(err)
		}
		if result.ClientStates == nil {
			result.ClientStates = make(map[string]*storage.ClientAuthState)
		}
	} else {
		result.ClientStates = make(map[string]*storage.ClientAuthState)
	}
	return result
}

func toStorageDeviceToken(t *db.DeviceToken) storage.DeviceToken {
	return storage.DeviceToken{
		DeviceCode:          t.DeviceCode,
		Status:              t.Status,
		Token:               string(*t.Token),
		Expiry:              t.Expiry,
		LastRequestTime:     t.LastRequest,
		PollIntervalSeconds: t.PollInterval,
		PKCE: storage.PKCE{
			CodeChallenge:       t.CodeChallenge,
			CodeChallengeMethod: t.CodeChallengeMethod,
		},
	}
}
