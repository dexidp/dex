package etcd

import (
	"time"

	"github.com/go-jose/go-jose/v4"

	"github.com/dexidp/dex/storage"
)

// AuthCode is a mirrored struct from storage with JSON struct tags
type AuthCode struct {
	ID          string   `json:"ID"`
	ClientID    string   `json:"clientID"`
	RedirectURI string   `json:"redirectURI"`
	Nonce       string   `json:"nonce,omitempty"`
	Scopes      []string `json:"scopes,omitempty"`

	ConnectorID   string `json:"connectorID,omitempty"`
	ConnectorData []byte `json:"connectorData,omitempty"`
	Claims        Claims `json:"claims,omitempty"`

	Expiry time.Time `json:"expiry"`

	CodeChallenge       string `json:"code_challenge,omitempty"`
	CodeChallengeMethod string `json:"code_challenge_method,omitempty"`
}

func toStorageAuthCode(a AuthCode) storage.AuthCode {
	return storage.AuthCode{
		ID:            a.ID,
		ClientID:      a.ClientID,
		RedirectURI:   a.RedirectURI,
		ConnectorID:   a.ConnectorID,
		ConnectorData: a.ConnectorData,
		Nonce:         a.Nonce,
		Scopes:        a.Scopes,
		Claims:        toStorageClaims(a.Claims),
		Expiry:        a.Expiry,
		PKCE: storage.PKCE{
			CodeChallenge:       a.CodeChallenge,
			CodeChallengeMethod: a.CodeChallengeMethod,
		},
	}
}

func fromStorageAuthCode(a storage.AuthCode) AuthCode {
	return AuthCode{
		ID:                  a.ID,
		ClientID:            a.ClientID,
		RedirectURI:         a.RedirectURI,
		ConnectorID:         a.ConnectorID,
		ConnectorData:       a.ConnectorData,
		Nonce:               a.Nonce,
		Scopes:              a.Scopes,
		Claims:              fromStorageClaims(a.Claims),
		Expiry:              a.Expiry,
		CodeChallenge:       a.PKCE.CodeChallenge,
		CodeChallengeMethod: a.PKCE.CodeChallengeMethod,
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

	CodeChallenge       string `json:"code_challenge,omitempty"`
	CodeChallengeMethod string `json:"code_challenge_method,omitempty"`

	HMACKey []byte `json:"hmac_key"`
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
		ConnectorData:       a.ConnectorData,
		CodeChallenge:       a.PKCE.CodeChallenge,
		CodeChallengeMethod: a.PKCE.CodeChallengeMethod,
		HMACKey:             a.HMACKey,
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
		ConnectorData:       a.ConnectorData,
		Expiry:              a.Expiry,
		Claims:              toStorageClaims(a.Claims),
		PKCE: storage.PKCE{
			CodeChallenge:       a.CodeChallenge,
			CodeChallengeMethod: a.CodeChallengeMethod,
		},
		HMACKey: a.HMACKey,
	}
}

// RefreshToken is a mirrored struct from storage with JSON struct tags
type RefreshToken struct {
	ID string `json:"id"`

	Token         string `json:"token"`
	ObsoleteToken string `json:"obsolete_token"`

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
		ID:            r.ID,
		Token:         r.Token,
		ObsoleteToken: r.ObsoleteToken,
		CreatedAt:     r.CreatedAt,
		LastUsed:      r.LastUsed,
		ClientID:      r.ClientID,
		ConnectorID:   r.ConnectorID,
		ConnectorData: r.ConnectorData,
		Scopes:        r.Scopes,
		Nonce:         r.Nonce,
		Claims:        toStorageClaims(r.Claims),
	}
}

func fromStorageRefreshToken(r storage.RefreshToken) RefreshToken {
	return RefreshToken{
		ID:            r.ID,
		Token:         r.Token,
		ObsoleteToken: r.ObsoleteToken,
		CreatedAt:     r.CreatedAt,
		LastUsed:      r.LastUsed,
		ClientID:      r.ClientID,
		ConnectorID:   r.ConnectorID,
		ConnectorData: r.ConnectorData,
		Scopes:        r.Scopes,
		Nonce:         r.Nonce,
		Claims:        fromStorageClaims(r.Claims),
	}
}

// Claims is a mirrored struct from storage with JSON struct tags.
type Claims struct {
	UserID            string   `json:"userID"`
	Username          string   `json:"username"`
	PreferredUsername string   `json:"preferredUsername"`
	Email             string   `json:"email"`
	EmailVerified     bool     `json:"emailVerified"`
	Groups            []string `json:"groups,omitempty"`
}

func fromStorageClaims(i storage.Claims) Claims {
	return Claims{
		UserID:            i.UserID,
		Username:          i.Username,
		PreferredUsername: i.PreferredUsername,
		Email:             i.Email,
		EmailVerified:     i.EmailVerified,
		Groups:            i.Groups,
	}
}

func toStorageClaims(i Claims) storage.Claims {
	return storage.Claims{
		UserID:            i.UserID,
		Username:          i.Username,
		PreferredUsername: i.PreferredUsername,
		Email:             i.Email,
		EmailVerified:     i.EmailVerified,
		Groups:            i.Groups,
	}
}

// Keys is a mirrored struct from storage with JSON struct tags
type Keys struct {
	SigningKey       *jose.JSONWebKey          `json:"signing_key,omitempty"`
	SigningKeyPub    *jose.JSONWebKey          `json:"signing_key_pub,omitempty"`
	VerificationKeys []storage.VerificationKey `json:"verification_keys"`
	NextRotation     time.Time                 `json:"next_rotation"`
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

// DeviceRequest is a mirrored struct from storage with JSON struct tags
type DeviceRequest struct {
	UserCode     string    `json:"user_code"`
	DeviceCode   string    `json:"device_code"`
	ClientID     string    `json:"client_id"`
	ClientSecret string    `json:"client_secret"`
	Scopes       []string  `json:"scopes"`
	Expiry       time.Time `json:"expiry"`
}

func fromStorageDeviceRequest(d storage.DeviceRequest) DeviceRequest {
	return DeviceRequest{
		UserCode:     d.UserCode,
		DeviceCode:   d.DeviceCode,
		ClientID:     d.ClientID,
		ClientSecret: d.ClientSecret,
		Scopes:       d.Scopes,
		Expiry:       d.Expiry,
	}
}

func toStorageDeviceRequest(d DeviceRequest) storage.DeviceRequest {
	return storage.DeviceRequest{
		UserCode:     d.UserCode,
		DeviceCode:   d.DeviceCode,
		ClientID:     d.ClientID,
		ClientSecret: d.ClientSecret,
		Scopes:       d.Scopes,
		Expiry:       d.Expiry,
	}
}

// DeviceToken is a mirrored struct from storage with JSON struct tags
type DeviceToken struct {
	DeviceCode          string    `json:"device_code"`
	Status              string    `json:"status"`
	Token               string    `json:"token"`
	Expiry              time.Time `json:"expiry"`
	LastRequestTime     time.Time `json:"last_request"`
	PollIntervalSeconds int       `json:"poll_interval"`
	CodeChallenge       string    `json:"code_challenge,omitempty"`
	CodeChallengeMethod string    `json:"code_challenge_method,omitempty"`
}

func fromStorageDeviceToken(t storage.DeviceToken) DeviceToken {
	return DeviceToken{
		DeviceCode:          t.DeviceCode,
		Status:              t.Status,
		Token:               t.Token,
		Expiry:              t.Expiry,
		LastRequestTime:     t.LastRequestTime,
		PollIntervalSeconds: t.PollIntervalSeconds,
		CodeChallenge:       t.PKCE.CodeChallenge,
		CodeChallengeMethod: t.PKCE.CodeChallengeMethod,
	}
}

func toStorageDeviceToken(t DeviceToken) storage.DeviceToken {
	return storage.DeviceToken{
		DeviceCode:          t.DeviceCode,
		Status:              t.Status,
		Token:               t.Token,
		Expiry:              t.Expiry,
		LastRequestTime:     t.LastRequestTime,
		PollIntervalSeconds: t.PollIntervalSeconds,
		PKCE: storage.PKCE{
			CodeChallenge:       t.CodeChallenge,
			CodeChallengeMethod: t.CodeChallengeMethod,
		},
	}
}
