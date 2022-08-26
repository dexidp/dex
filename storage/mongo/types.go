package mongo

import (
	"reflect"
	"time"

	"github.com/dexidp/dex/storage"
	"github.com/pkg/errors"
	"gopkg.in/square/go-jose.v2"
)

// AuthCode is a mirrored struct from storage with BSON struct tags
type AuthCode struct {
	ID          string   `bson:"id"`
	ClientID    string   `bson:"client_id"`
	RedirectURI string   `bson:"redirect_uri"`
	Nonce       string   `bson:"nonce,omitempty"`
	Scopes      []string `bson:"scopes,omitempty"`

	ConnectorID   string `bson:"connector_id,omitempty"`
	ConnectorData []byte `bson:"connector_data,omitempty"`
	Claims        Claims `bson:"claims,omitempty"`

	Expiry time.Time `bson:"expiry"`

	LoginHint           string `bson:"login_hint"`
	HD                  string `bson:"hd"`
	ClaimsLocation      string `bson:"claims_location"`
	ClaimsLocationScore int    `bson:"claims_location_score"`
	CodeChallenge       string `bson:"code_challenge,omitempty"`
	CodeChallengeMethod string `bson:"code_challenge_method,omitempty"`
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

// AuthRequest is a mirrored struct from storage with BSON struct tags
type AuthRequest struct {
	ID       string `bson:"id"`
	ClientID string `bson:"client_id"`

	ResponseTypes []string `bson:"response_types"`
	Scopes        []string `bson:"scopes"`
	RedirectURI   string   `bson:"redirect_uri"`
	Nonce         string   `bson:"nonce"`
	State         string   `bson:"state"`

	ForceApprovalPrompt bool `bson:"force_approval_prompt"`

	Expiry time.Time `bson:"expiry"`

	LoggedIn bool `bson:"logged_in"`

	Claims Claims `bson:"claims"`

	ConnectorID   string `bson:"connector_id"`
	ConnectorData []byte `bson:"connector_data"`

	CodeChallenge       string `bson:"code_challenge,omitempty"`
	CodeChallengeMethod string `bson:"code_challenge_method,omitempty"`
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
	}
}

// RefreshToken is a mirrored struct from storage with BSON struct tags
type RefreshToken struct {
	ID string `bson:"id"`

	Token         string `bson:"token"`
	ObsoleteToken string `bson:"obsolete_token"`

	CreatedAt time.Time `bson:"created_at"`
	LastUsed  time.Time `bson:"last_used"`

	ClientID string `bson:"client_id"`

	ConnectorID   string `bson:"connector_id"`
	ConnectorData []byte `bson:"connector_data"`
	Claims        Claims `bson:"claims"`

	Scopes []string `bson:"scopes"`

	Nonce string `bson:"nonce"`
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

// Claims is a mirrored struct from storage with BSON struct tags.
type Claims struct {
	UserID            string   `bson:"userID"`
	Username          string   `bson:"username"`
	PreferredUsername string   `bson:"preferredUsername"`
	Email             string   `bson:"email"`
	EmailVerified     bool     `bson:"emailVerified"`
	Groups            []string `bson:"groups,omitempty"`
}

func fromStorageClaims(i storage.Claims) Claims {
	return Claims(i)
}

func toStorageClaims(i Claims) storage.Claims {
	return storage.Claims(i)
}

// Keys is a mirrored struct from storage with BSON struct tags
type Keys struct {
	SigningKey       []byte            `bson:"signing_key_bytes,omitempty"`
	SigningKeyPub    []byte            `bson:"signing_key_pub_bytes,omitempty"`
	VerificationKeys []VerificationKey `bson:"verification_keys"`
	NextRotation     time.Time         `bson:"next_rotation"`
}

func isEmptyJSONWebKey(key jose.JSONWebKey) bool {
	return reflect.DeepEqual(key, jose.JSONWebKey{})
}

func fromStorageKeys(k storage.Keys) (Keys, error) {
	if reflect.DeepEqual(k, storage.Keys{}) {
		return Keys{}, nil
	}

	keys := Keys{
		VerificationKeys: []VerificationKey{},
		NextRotation:     k.NextRotation,
	}

	if isEmptyJSONWebKey(*k.SigningKey) {
		k.SigningKey = &jose.JSONWebKey{}
	} else {
		skb, err := k.SigningKey.MarshalJSON()
		if err != nil {
			return Keys{}, errors.Wrap(err, "marshal signing key")
		}

		keys.SigningKey = skb
	}

	if isEmptyJSONWebKey(*k.SigningKeyPub) {
		k.SigningKeyPub = &jose.JSONWebKey{}
	} else {
		spkb, err := k.SigningKeyPub.MarshalJSON()
		if err != nil {
			return Keys{}, errors.Wrap(err, "marshal signing pub key")
		}

		keys.SigningKeyPub = spkb
	}

	for _, verificationKey := range k.VerificationKeys {
		vk, err := fromStorageVerificationKeys(verificationKey)
		if err != nil {
			return Keys{}, errors.Wrap(err, "form storage verification")
		}

		keys.VerificationKeys = append(keys.VerificationKeys, vk)
	}

	return keys, nil
}

func toStorageKeys(k Keys) (storage.Keys, error) {
	if reflect.DeepEqual(k, Keys{}) {
		return storage.Keys{}, nil
	}

	keys := storage.Keys{
		SigningKey:       &jose.JSONWebKey{},
		SigningKeyPub:    &jose.JSONWebKey{},
		VerificationKeys: []storage.VerificationKey{},
		NextRotation:     k.NextRotation,
	}

	if len(k.SigningKey) > 0 {
		if err := keys.SigningKey.UnmarshalJSON(k.SigningKey); err != nil {
			return storage.Keys{}, err
		}
	}

	if len(k.SigningKeyPub) > 0 {
		if err := keys.SigningKeyPub.UnmarshalJSON(k.SigningKeyPub); err != nil {
			return storage.Keys{}, err
		}
	}

	for _, verificationKey := range k.VerificationKeys {
		vk, err := toStorageVerificationKeys(verificationKey)
		if err != nil {
			return storage.Keys{}, err
		}

		keys.VerificationKeys = append(keys.VerificationKeys, vk)
	}

	return keys, nil
}

type VerificationKey struct {
	PublicKey []byte    `bson:"public_key_bytes"`
	Expiry    time.Time `bson:"expiry"`
}

func fromStorageVerificationKeys(vk storage.VerificationKey) (VerificationKey, error) {
	verificationKey := VerificationKey{
		Expiry:    vk.Expiry,
		PublicKey: []byte{},
	}
	if !isEmptyJSONWebKey(*vk.PublicKey) {
		publicBytes, err := vk.PublicKey.MarshalJSON()
		if err != nil {
			return VerificationKey{}, errors.Wrap(err, "marshal verification public key")
		}
		verificationKey.PublicKey = publicBytes
	}
	return verificationKey, nil
}

func toStorageVerificationKeys(vk VerificationKey) (storage.VerificationKey, error) {
	svk := storage.VerificationKey{
		Expiry:    vk.Expiry,
		PublicKey: &jose.JSONWebKey{},
	}
	if svk.PublicKey != nil {
		if err := svk.PublicKey.UnmarshalJSON(vk.PublicKey); err != nil {
			return storage.VerificationKey{}, err
		}
	}

	return svk, nil
}

// OfflineSessions is a mirrored struct from storage with BSON struct tags
type OfflineSessions struct {
	UserID        string                              `bson:"user_id,omitempty"`
	ConnID        string                              `bson:"conn_id,omitempty"`
	Refresh       map[string]*storage.RefreshTokenRef `bson:"refresh,omitempty"`
	ConnectorData []byte                              `bson:"connector_data,omitempty"`
}

func fromStorageOfflineSessions(o storage.OfflineSessions) OfflineSessions {
	return OfflineSessions(o)
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

// DeviceRequest is a mirrored struct from storage with BSON struct tags
type DeviceRequest struct {
	UserCode     string    `bson:"user_code"`
	DeviceCode   string    `bson:"device_code"`
	ClientID     string    `bson:"client_id"`
	ClientSecret string    `bson:"client_secret"`
	Scopes       []string  `bson:"scopes"`
	Expiry       time.Time `bson:"expiry"`
}

func fromStorageDeviceRequest(d storage.DeviceRequest) DeviceRequest {
	return DeviceRequest(d)
}

func toStorageDeviceRequest(d DeviceRequest) storage.DeviceRequest {
	return storage.DeviceRequest(d)
}

// DeviceToken is a mirrored struct from storage with BSON struct tags
type DeviceToken struct {
	DeviceCode          string    `bson:"device_code"`
	Status              string    `bson:"status"`
	Token               string    `bson:"token"`
	Expiry              time.Time `bson:"expiry"`
	LastRequestTime     time.Time `bson:"last_request"`
	PollIntervalSeconds int       `bson:"poll_interval"`
	CodeChallenge       string    `bson:"code_challenge,omitempty"`
	CodeChallengeMethod string    `bson:"code_challenge_method,omitempty"`
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
