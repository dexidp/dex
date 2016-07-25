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
	ErrTokenExpired = errors.New("ID Token expired")
	// ErrNotSupported indicates that the requested optional OpenID Connect endpoint is not supported by the provider.
	ErrNotSupported = errors.New("endpoint not supported")
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

	// Optionally contains extra claims.
	raw map[string]interface{}
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
	// raw claims do not get error checks
	json.Unmarshal(body, &p.raw)
	if p.Issuer != issuer {
		return nil, fmt.Errorf("oidc: issuer did not match the issuer returned by provider, expected %q got %q", issuer, p.Issuer)
	}
	return &p, nil
}

// Extra returns additional fields returned by the server during discovery.
func (p *Provider) Extra(key string) interface{} {
	if p.raw != nil {
		return p.raw[key]
	}
	return nil
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

	// Optionally contains extra claims.
	raw map[string]interface{}
}

// Extra returns additional claims returned by the server.
func (u *UserInfo) Extra(key string) interface{} {
	if u.raw != nil {
		return u.raw[key]
	}
	return nil
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
	// raw claims do not get error checks
	json.Unmarshal(body, &userInfo.raw)
	return &userInfo, nil
}

// IDTokenVerifier provides verification for ID Tokens.
type IDTokenVerifier struct {
	issuer  string
	keySet  *remoteKeySet
	options []VerificationOption
}

// Verify parse the raw ID Token, verifies it's been signed by the provider, preforms
// additional verification, such as checking the expiration, and returns the claims.
func (v *IDTokenVerifier) Verify(rawIDToken string) (payload []byte, err error) {
	payload, err = v.keySet.verifyJWT(rawIDToken)
	if err != nil {
		return nil, err
	}
	var token struct {
		Exp    float64 `json:"exp"` // JSON numbers are always float64s.
		Issuer string  `json:"iss"`
	}
	if err := json.Unmarshal(payload, &token); err != nil {
		return nil, fmt.Errorf("oidc: failed to unmarshal claims: %v", err)
	}
	if v.issuer != token.Issuer {
		return nil, fmt.Errorf("oidc: iss field did not match provider issuer")
	}
	if time.Unix(int64(token.Exp), 0).Before(time.Now().Round(time.Second)) {
		return nil, ErrTokenExpired
	}
	for _, option := range v.options {
		if err := option.verifyIDTokenPayload(payload); err != nil {
			return nil, err
		}
	}
	return payload, nil
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
	verifyIDTokenPayload(raw []byte) error
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

func (c clientVerifier) verifyIDTokenPayload(payload []byte) error {
	var token struct {
		Aud string `json:"aud"`
	}
	if err := json.Unmarshal(payload, &token); err == nil {
		if token.Aud != c.clientID {
			return errors.New("oidc: id token aud field did not match client_id")
		}
		return nil
	}

	// Aud can optionally be an array of strings
	var token2 struct {
		Aud []string `json:"aud"`
	}
	if err := json.Unmarshal(payload, &token2); err != nil {
		return fmt.Errorf("oidc: failed to unmarshal aud claim: %v", err)
	}
	for _, aud := range token2.Aud {
		if aud == c.clientID {
			return nil
		}
	}
	return errors.New("oidc: id token aud field did not match client_id")
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
