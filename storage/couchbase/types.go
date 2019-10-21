package couchbase

import (
	"strconv"
	"time"

	jose "gopkg.in/square/go-jose.v2"

	"github.com/dexidp/dex/storage"
)

type RefreshTokenRef struct {
	ID        string `json:"ID"`
	ClientID  string `json:"clientID"`
	CreatedAt int64  `json:"created_at"`
	LastUsed  int64  `json:"last_used"`
}

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

	Expiry int64 `json:"expiry"`
}

func fromStorageAuthCode(a storage.AuthCode) AuthCode {
	return AuthCode{
		ID:            a.ID,
		ClientID:      a.ClientID,
		RedirectURI:   a.RedirectURI,
		ConnectorID:   a.ConnectorID,
		ConnectorData: a.ConnectorData,
		Nonce:         a.Nonce,
		Scopes:        a.Scopes,
		Claims:        fromStorageClaims(a.Claims),
		Expiry:        a.Expiry.Unix(),
	}
}

func toStorageAuthCode(a AuthCode) storage.AuthCode {
	i, _ := strconv.ParseInt(strconv.FormatInt(a.Expiry, 10), 10, 64)
	tm := time.Unix(i, 0)
	return storage.AuthCode{
		ID:            a.ID,
		ClientID:      a.ClientID,
		RedirectURI:   a.RedirectURI,
		ConnectorID:   a.ConnectorID,
		ConnectorData: a.ConnectorData,
		Nonce:         a.Nonce,
		Scopes:        a.Scopes,
		Claims:        toStorageClaims(a.Claims),
		Expiry:        tm,
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

	Expiry int64 `json:"expiry"`

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
		Expiry:              a.Expiry.Unix(),
		LoggedIn:            a.LoggedIn,
		Claims:              fromStorageClaims(a.Claims),
		ConnectorID:         a.ConnectorID,
		ConnectorData:       a.ConnectorData,
	}
}

func toStorageAuthRequest(a AuthRequest) storage.AuthRequest {
	i, _ := strconv.ParseInt(strconv.FormatInt(a.Expiry, 10), 10, 64)
	tm := time.Unix(i, 0)

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
		Expiry:              tm,
		Claims:              toStorageClaims(a.Claims),
	}
}

// RefreshToken is a mirrored struct from storage with JSON struct tags
type RefreshToken struct {
	ID string `json:"id"`

	Token string `json:"token"`

	CreatedAt int64 `json:"created_at"`
	LastUsed  int64 `json:"last_used"`

	ClientID string `json:"client_id"`

	ConnectorID   string `json:"connector_id"`
	ConnectorData []byte `json:"connector_data"`
	Claims        Claims `json:"claims"`

	Scopes []string `json:"scopes"`

	Nonce string `json:"nonce"`
}

func fromStorageRefreshToken(r storage.RefreshToken) RefreshToken {
	return RefreshToken{
		ID:            r.ID,
		Token:         r.Token,
		CreatedAt:     r.CreatedAt.Unix(),
		LastUsed:      r.LastUsed.Unix(),
		ClientID:      r.ClientID,
		ConnectorID:   r.ConnectorID,
		ConnectorData: r.ConnectorData,
		Scopes:        r.Scopes,
		Nonce:         r.Nonce,
		Claims:        fromStorageClaims(r.Claims),
	}
}

func toStorageRefreshToken(r RefreshToken) storage.RefreshToken {
	created_at, _ := strconv.ParseInt(strconv.FormatInt(r.CreatedAt, 10), 10, 64)
	last_used, _ := strconv.ParseInt(strconv.FormatInt(r.LastUsed, 10), 10, 64)
	return storage.RefreshToken{
		ID:            r.ID,
		Token:         r.Token,
		CreatedAt:     time.Unix(created_at, 0),
		LastUsed:      time.Unix(last_used, 0),
		ClientID:      r.ClientID,
		ConnectorID:   r.ConnectorID,
		ConnectorData: r.ConnectorData,
		Scopes:        r.Scopes,
		Nonce:         r.Nonce,
		Claims:        toStorageClaims(r.Claims),
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

// signatures.
type VerificationKey struct {
	PublicKey *jose.JSONWebKey `json:"publicKey"`
	Expiry    int64            `json:"expiry"`
}

func fromStorageVerificationKey(v storage.VerificationKey) VerificationKey {
	return VerificationKey{
		PublicKey: v.PublicKey,
		Expiry:    v.Expiry.Unix(),
	}
}

func toStorageVerificationKey(v VerificationKey) storage.VerificationKey {
	expire, _ := strconv.ParseInt(strconv.FormatInt(v.Expiry, 10), 10, 64)

	return storage.VerificationKey{
		PublicKey: v.PublicKey,
		Expiry:    time.Unix(expire, 0),
	}
}

// Keys is a mirrored struct from storage with JSON struct tags
type Keys struct {
	SigningKey       *jose.JSONWebKey  `json:"signing_key,omitempty"`
	SigningKeyPub    *jose.JSONWebKey  `json:"signing_key_pub,omitempty"`
	VerificationKeys []VerificationKey `json:"verification_keys"`
	NextRotation     int64             `json:"next_rotation"`
}

// OfflineSessions is a mirrored struct from storage with JSON struct tags
type OfflineSessions struct {
	UserID  string                     `json:"user_id,omitempty"`
	ConnID  string                     `json:"conn_id,omitempty"`
	Refresh map[string]RefreshTokenRef `json:"refresh,omitempty"`
}

func fromStorageOfflineSessions(o storage.OfflineSessions) OfflineSessions {
	list_vk := make(map[string]RefreshTokenRef)
	for k, v := range o.Refresh {
		obj := RefreshTokenRef{
			ID:        v.ID,
			ClientID:  v.ClientID,
			CreatedAt: v.CreatedAt.Unix(),
			LastUsed:  v.LastUsed.Unix(),
		}
		list_vk[k] = obj
	}
	return OfflineSessions{
		UserID:  o.UserID,
		ConnID:  o.ConnID,
		Refresh: list_vk,
	}
}

func toStorageOfflineSessions(o OfflineSessions) storage.OfflineSessions {
	list_vk := make(map[string]*storage.RefreshTokenRef)
	for k, v := range o.Refresh {
		created_at, _ := strconv.ParseInt(strconv.FormatInt(v.CreatedAt, 10), 10, 64)
		last_used, _ := strconv.ParseInt(strconv.FormatInt(v.LastUsed, 10), 10, 64)

		obj := storage.RefreshTokenRef{
			ID:        v.ID,
			ClientID:  v.ClientID,
			CreatedAt: time.Unix(created_at, 0),
			LastUsed:  time.Unix(last_used, 0),
		}

		list_vk[k] = &obj
	}

	s := storage.OfflineSessions{
		UserID:  o.UserID,
		ConnID:  o.ConnID,
		Refresh: list_vk,
	}
	if s.Refresh == nil {
		// Server code assumes this will be non-nil.
		s.Refresh = make(map[string]*storage.RefreshTokenRef)
	}
	return s
}

func fromStorageKeys(o storage.Keys) Keys {
	var list_vk []VerificationKey
	for _, vk := range o.VerificationKeys {
		list_vk = append(list_vk, fromStorageVerificationKey(vk))
	}
	return Keys{
		SigningKey:       o.SigningKey,
		SigningKeyPub:    o.SigningKeyPub,
		VerificationKeys: list_vk,
		NextRotation:     o.NextRotation.Unix(),
	}
}

func toStorageKeys(o Keys) storage.Keys {
	time_parsed, _ := strconv.ParseInt(strconv.FormatInt(o.NextRotation, 10), 10, 64)
	var list_vk []storage.VerificationKey
	for _, vk := range o.VerificationKeys {
		list_vk = append(list_vk, toStorageVerificationKey(vk))
	}
	s := storage.Keys{
		SigningKey:       o.SigningKey,
		SigningKeyPub:    o.SigningKeyPub,
		VerificationKeys: list_vk,
		NextRotation:     time.Unix(time_parsed, 0),
	}
	return s
}
