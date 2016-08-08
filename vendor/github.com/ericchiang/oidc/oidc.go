package oidc

import (
	"encoding/json"
	"errors"
	"fmt"
	"io/ioutil"
	"net/http"
	"strings"
	"time"

	"golang.org/x/net/context"
	"golang.org/x/oauth2"
)

var (
	// ErrTokenExpired indicates that a token parsed by a verifier has expired.
	ErrTokenExpired = errors.New("oidc: ID Token expired")
	// ErrNotSupported indicates that the requested optional OpenID Connect endpoint is not supported by the provider.
	ErrNotSupported = errors.New("oidc: endpoint not supported")
)

const (
	// ScopeOpenID is the mandatory scope for all OpenID Connect OAuth2 requests.
	ScopeOpenID = "openid"

	// ScopeOfflineAccess is an optional scope defined by OpenID Connect for requesting
	// OAuth2 refresh tokens.
	//
	// Support for this scope differs between OpenID Connect providers. For instance
	// Google rejects it, favoring appending "access_type=offline" as part of the
	// authorization request instead.
	//
	// See: https://openid.net/specs/openid-connect-core-1_0.html#OfflineAccess
	ScopeOfflineAccess = "offline_access"
)

// Provider contains the subset of the OpenID Connect provider metadata needed to request
// and verify ID Tokens.
type Provider struct {
	Issuer      string `json:"issuer"`
	AuthURL     string `json:"authorization_endpoint"`
	TokenURL    string `json:"token_endpoint"`
	JWKSURL     string `json:"jwks_uri"`
	UserInfoURL string `json:"userinfo_endpoint"`

	// Raw claims returned by the server.
	rawClaims []byte
}

// NewProvider uses the OpenID Connect disovery mechanism to construct a Provider.
func NewProvider(ctx context.Context, issuer string) (*Provider, error) {
	wellKnown := strings.TrimSuffix(issuer, "/") + "/.well-known/openid-configuration"
	resp, err := contextClient(ctx).Get(wellKnown)
	if err != nil {
		return nil, err
	}
	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("%s: %s", resp.Status, body)
	}
	defer resp.Body.Close()
	var p Provider
	if err := json.Unmarshal(body, &p); err != nil {
		return nil, fmt.Errorf("oidc: failed to decode provider discovery object: %v", err)
	}
	p.rawClaims = body
	if p.Issuer != issuer {
		return nil, fmt.Errorf("oidc: issuer did not match the issuer returned by provider, expected %q got %q", issuer, p.Issuer)
	}
	return &p, nil
}

// Claims returns additional fields returned by the server during discovery.
func (p *Provider) Claims(v interface{}) error {
	if p.rawClaims == nil {
		return errors.New("oidc: claims not set")
	}
	return json.Unmarshal(p.rawClaims, v)
}

// Endpoint returns the OAuth2 auth and token endpoints for the given provider.
func (p *Provider) Endpoint() oauth2.Endpoint {
	return oauth2.Endpoint{AuthURL: p.AuthURL, TokenURL: p.TokenURL}
}

// UserInfo represents the OpenID Connect userinfo claims.
type UserInfo struct {
	Subject       string `json:"sub"`
	Profile       string `json:"profile"`
	Email         string `json:"email"`
	EmailVerified bool   `json:"email_verified"`

	claims []byte
}

// Claims unmarshals the raw JSON object claims into the provided object.
func (u *UserInfo) Claims(v interface{}) error {
	if u.claims == nil {
		return errors.New("oidc: claims not set")
	}
	return json.Unmarshal(u.claims, v)
}

// UserInfo uses the token source to query the provider's user info endpoint.
func (p *Provider) UserInfo(ctx context.Context, tokenSource oauth2.TokenSource) (*UserInfo, error) {
	if p.UserInfoURL == "" {
		return nil, ErrNotSupported
	}
	cli := oauth2.NewClient(ctx, tokenSource)
	resp, err := cli.Get(p.UserInfoURL)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("%s: %s", resp.Status, body)
	}

	var userInfo UserInfo
	if err := json.Unmarshal(body, &userInfo); err != nil {
		return nil, fmt.Errorf("oidc: failed to decode userinfo: %v", err)
	}
	userInfo.claims = body
	return &userInfo, nil
}

// IDToken is an OpenID Connect extension that provides a predictable representation
// of an authorization event.
//
// The ID Token only holds fields OpenID Connect requires. To access additional
// claims returned by the server, use the Claims method.
//
//		idToken, err := idTokenVerifier.Verify(rawIDToken)
//		if err != nil {
//			// handle error
//		}
//		var claims struct {
//			Email         string `json:"email"`
//			EmailVerified bool   `json:"email_verified"`
//		}
//		if err := idToken.Claims(&claims); err != nil {
//			// handle error
//		}
//
type IDToken struct {
	// The URL of the server which issued this token. This will always be the same
	// as the URL used for initial discovery.
	Issuer string

	// The client, or set of clients, that this token is issued for.
	Audience []string

	// A unique string which identifies the end user.
	Subject string

	IssuedAt time.Time
	Expiry   time.Time
	Nonce    string

	claims []byte
}

// Claims unmarshals the raw JSON payload of the ID Token into a provided struct.
func (i *IDToken) Claims(v interface{}) error {
	if i.claims == nil {
		return errors.New("oidc: claims not set")
	}
	return json.Unmarshal(i.claims, v)
}

type audience []string

func (a *audience) UnmarshalJSON(b []byte) error {
	var s string
	if json.Unmarshal(b, &s) == nil {
		*a = audience{s}
		return nil
	}
	var auds []string
	if err := json.Unmarshal(b, &auds); err != nil {
		return err
	}
	*a = audience(auds)
	return nil
}

type jsonTime time.Time

func (j *jsonTime) UnmarshalJSON(b []byte) error {
	var n json.Number
	if err := json.Unmarshal(b, &n); err != nil {
		return err
	}
	var unix int64

	if t, err := n.Int64(); err == nil {
		unix = t
	} else {
		f, err := n.Float64()
		if err != nil {
			return err
		}
		unix = int64(f)
	}
	*j = jsonTime(time.Unix(unix, 0))
	return nil
}

type idToken struct {
	Issuer   string   `json:"iss"`
	Subject  string   `json:"sub"`
	Audience audience `json:"aud"`
	Expiry   jsonTime `json:"exp"`
	IssuedAt jsonTime `json:"iat"`
	Nonce    string   `json:"nonce"`
}

// IDTokenVerifier provides verification for ID Tokens.
type IDTokenVerifier struct {
	issuer  string
	keySet  *remoteKeySet
	options []VerificationOption
}

// Verify parse the raw ID Token, verifies it's been signed by the provider, preforms
// additional verification, and returns the claims.
func (v *IDTokenVerifier) Verify(rawIDToken string) (*IDToken, error) {
	payload, err := v.keySet.verifyJWT(rawIDToken)
	if err != nil {
		return nil, err
	}
	var token idToken
	if err := json.Unmarshal(payload, &token); err != nil {
		return nil, fmt.Errorf("oidc: failed to unmarshal claims: %v", err)
	}
	if v.issuer != token.Issuer {
		return nil, fmt.Errorf("oidc: iss field did not match provider issuer")
	}
	t := &IDToken{
		Issuer:   token.Issuer,
		Subject:  token.Subject,
		Audience: []string(token.Audience),
		Expiry:   time.Time(token.Expiry),
		IssuedAt: time.Time(token.Expiry),
		Nonce:    token.Nonce,
		claims:   payload,
	}
	for _, option := range v.options {
		if err := option.verifyIDToken(t); err != nil {
			return nil, err
		}
	}
	return t, nil
}

// NewVerifier returns an IDTokenVerifier that uses the provider's key set to verify JWTs.
//
// The verifier queries the provider to update keys when a signature cannot be verified by the
// set of keys cached from the previous request.
func (p *Provider) NewVerifier(ctx context.Context, options ...VerificationOption) *IDTokenVerifier {
	return &IDTokenVerifier{
		issuer:  p.Issuer,
		keySet:  newRemoteKeySet(ctx, p.JWKSURL),
		options: options,
	}
}

// VerificationOption is an option provided to Provider.NewVerifier.
type VerificationOption interface {
	verifyIDToken(token *IDToken) error
}

// VerifyAudience ensures that an ID Token was issued for the specific client.
//
// Note that a verified token may be valid for other clients, as OpenID Connect allows a token to have
// multiple audiences.
func VerifyAudience(clientID string) VerificationOption {
	return clientVerifier{clientID}
}

type clientVerifier struct {
	clientID string
}

func (c clientVerifier) verifyIDToken(token *IDToken) error {
	for _, aud := range token.Audience {
		if aud == c.clientID {
			return nil
		}
	}
	return errors.New("oidc: id token aud field did not match client_id")
}

// VerifyExpiry ensures that an ID Token has not expired.
func VerifyExpiry() VerificationOption {
	return expiryVerifier{time.Now}
}

type expiryVerifier struct {
	now func() time.Time
}

func (e expiryVerifier) verifyIDToken(token *IDToken) error {
	if e.now().After(token.Expiry) {
		return ErrTokenExpired
	}
	return nil
}

// This method is internal to golang.org/x/oauth2. Just copy it.
func contextClient(ctx context.Context) *http.Client {
	if ctx != nil {
		if hc, ok := ctx.Value(oauth2.HTTPClient).(*http.Client); ok {
			return hc
		}
	}
	return http.DefaultClient
}
