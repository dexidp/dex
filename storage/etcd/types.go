package etcd

import (
	"time"

	jose "gopkg.in/square/go-jose.v2"

	"github.com/dexidp/dex/storage"
)

// AuthCode is a mirrored struct from storage with JSON struct tags
type AuthCode struct {
	ID          string   `json:"ID"`
	ClientID    string   `json:"clientID"`
	RedirectURI string   `json:"redirectURI"`
	Nonce       string   `json:"nonce,omitempty"`
	Scopes      []string `json:"scopes,omitempty"`

	ConnectorID string `json:"connectorID,omitempty"`
	Claims      Claims `json:"claims,omitempty"`

	Expiry time.Time `json:"expiry"`
}

func fromStorageAuthCode(a storage.AuthCode) AuthCode {
	return AuthCode{
		ID:          a.ID,
		ClientID:    a.ClientID,
		RedirectURI: a.RedirectURI,
		ConnectorID: a.ConnectorID,
		Nonce:       a.Nonce,
		Scopes:      a.Scopes,
		Claims:      fromStorageClaims(a.Claims),
		Expiry:      a.Expiry,
	}
}

// AuthRequest is a mirrored struct from storage with JSON struct tags
type AuthRequest struct {
	ID       string `json:"id"`
	ClientID string `json:"client_id"`

	ResponseTypes []string `json:"response_types"`
	Scopes        []string `json:"scopes"`
	RedirectURI   string   `json:"redirect_uri"`
	Nonce         string   `json:"nonce"`
	State         string   `json:"state"`

	ForceApprovalPrompt bool `json:"force_approval_prompt"`

	Expiry time.Time `json:"expiry"`

	LoggedIn bool `json:"logged_in"`

	Claims Claims `json:"claims"`

	ConnectorID   string `json:"connector_id"`
	ConnectorData []byte `json:"connector_data"`
}

func fromStorageAuthRequest(a storage.AuthRequest) AuthRequest {
	return AuthRequest{
		ID:                  a.ID,
		ClientID:            a.ClientID,
		ResponseTypes:       a.ResponseTypes,
		Scopes:              a.Scopes,
		RedirectURI:         a.RedirectURI,
		Nonce:               a.Nonce,
		State:               a.State,
		ForceApprovalPrompt: a.ForceApprovalPrompt,
		Expiry:              a.Expiry,
		LoggedIn:            a.LoggedIn,
		Claims:              fromStorageClaims(a.Claims),
		ConnectorID:         a.ConnectorID,
	}
}

func toStorageAuthRequest(a AuthRequest) storage.AuthRequest {
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
		Expiry:              a.Expiry,
		Claims:              toStorageClaims(a.Claims),
	}
}

// RefreshToken is a mirrored struct from storage with JSON struct tags
type RefreshToken struct {
	ID string `json:"id"`

	Token string `json:"token"`

	CreatedAt time.Time `json:"created_at"`
	LastUsed  time.Time `json:"last_used"`

	ClientID string `json:"client_id"`

	ConnectorID   string `json:"connector_id"`
	ConnectorData []byte `json:"connector_data"`
	Claims        Claims `json:"claims"`

	Scopes []string `json:"scopes"`

	Nonce string `json:"nonce"`
}

func toStorageRefreshToken(r RefreshToken) storage.RefreshToken {
	return storage.RefreshToken{
		ID:          r.ID,
		Token:       r.Token,
		CreatedAt:   r.CreatedAt,
		LastUsed:    r.LastUsed,
		ClientID:    r.ClientID,
		ConnectorID: r.ConnectorID,
		Scopes:      r.Scopes,
		Nonce:       r.Nonce,
		Claims:      toStorageClaims(r.Claims),
	}
}

func fromStorageRefreshToken(r storage.RefreshToken) RefreshToken {
	return RefreshToken{
		ID:          r.ID,
		Token:       r.Token,
		CreatedAt:   r.CreatedAt,
		LastUsed:    r.LastUsed,
		ClientID:    r.ClientID,
		ConnectorID: r.ConnectorID,
		Scopes:      r.Scopes,
		Nonce:       r.Nonce,
		Claims:      fromStorageClaims(r.Claims),
	}
}

// Claims is a mirrored struct from storage with JSON struct tags.
type Claims struct {
	UserID        string   `json:"userID"`
	Username      string   `json:"username"`
	Email         string   `json:"email"`
	EmailVerified bool     `json:"emailVerified"`
	Groups        []string `json:"groups,omitempty"`
}

func fromStorageClaims(i storage.Claims) Claims {
	return Claims{
		UserID:        i.UserID,
		Username:      i.Username,
		Email:         i.Email,
		EmailVerified: i.EmailVerified,
		Groups:        i.Groups,
	}
}

func toStorageClaims(i Claims) storage.Claims {
	return storage.Claims{
		UserID:        i.UserID,
		Username:      i.Username,
		Email:         i.Email,
		EmailVerified: i.EmailVerified,
		Groups:        i.Groups,
	}
}

// Keys is a mirrored struct from storage with JSON struct tags
type Keys struct {
	SigningKey       *jose.JSONWebKey          `json:"signing_key,omitempty"`
	SigningKeyPub    *jose.JSONWebKey          `json:"signing_key_pub,omitempty"`
	VerificationKeys []storage.VerificationKey `json:"verification_keys"`
	NextRotation     time.Time                 `json:"next_rotation"`
}

func fromStorageKeys(keys storage.Keys) Keys {
	return Keys{
		SigningKey:       keys.SigningKey,
		SigningKeyPub:    keys.SigningKeyPub,
		VerificationKeys: keys.VerificationKeys,
		NextRotation:     keys.NextRotation,
	}
}

func toStorageKeys(keys Keys) storage.Keys {
	return storage.Keys{
		SigningKey:       keys.SigningKey,
		SigningKeyPub:    keys.SigningKeyPub,
		VerificationKeys: keys.VerificationKeys,
		NextRotation:     keys.NextRotation,
	}
}

// OfflineSessions is a mirrored struct from storage with JSON struct tags
type OfflineSessions struct {
	UserID        string                              `json:"user_id,omitempty"`
	ConnID        string                              `json:"conn_id,omitempty"`
	Refresh       map[string]*storage.RefreshTokenRef `json:"refresh,omitempty"`
	ConnectorData []byte                              `json:"connectorData,omitempty"`
}

func fromStorageOfflineSessions(o storage.OfflineSessions) OfflineSessions {
	return OfflineSessions{
		UserID:        o.UserID,
		ConnID:        o.ConnID,
		Refresh:       o.Refresh,
		ConnectorData: o.ConnectorData,
	}
}

func toStorageOfflineSessions(o OfflineSessions) storage.OfflineSessions {
	s := storage.OfflineSessions{
		UserID:        o.UserID,
		ConnID:        o.ConnID,
		Refresh:       o.Refresh,
		ConnectorData: o.ConnectorData,
	}
	if s.Refresh == nil {
		// Server code assumes this will be non-nil.
		s.Refresh = make(map[string]*storage.RefreshTokenRef)
	}
	return s
}
