package server

import (
	"context"
	"crypto"
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rsa"
	"crypto/sha256"
	"crypto/sha512"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"hash"
	"io"
	"net"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"

	"github.com/go-jose/go-jose/v4"

	"github.com/dexidp/dex/connector"
	"github.com/dexidp/dex/server/internal"
	"github.com/dexidp/dex/storage"
)

// TODO(ericchiang): clean this file up and figure out more idiomatic error handling.

// See: https://tools.ietf.org/html/rfc6749#section-4.1.2.1

// displayedAuthErr is an error that should be displayed to the user as a web page
type displayedAuthErr struct {
	Status      int
	Description string
}

func (err *displayedAuthErr) Error() string {
	return err.Description
}

func newDisplayedErr(status int, format string, a ...interface{}) *displayedAuthErr {
	return &displayedAuthErr{status, fmt.Sprintf(format, a...)}
}

// redirectedAuthErr is an error that should be reported back to the client by 302 redirect
type redirectedAuthErr struct {
	State       string
	RedirectURI string
	Type        string
	Description string
}

func (err *redirectedAuthErr) Error() string {
	return err.Description
}

func (err *redirectedAuthErr) Handler() http.Handler {
	hf := func(w http.ResponseWriter, r *http.Request) {
		v := url.Values{}
		v.Add("state", err.State)
		v.Add("error", err.Type)
		if err.Description != "" {
			v.Add("error_description", err.Description)
		}
		var redirectURI string
		if strings.Contains(err.RedirectURI, "?") {
			redirectURI = err.RedirectURI + "&" + v.Encode()
		} else {
			redirectURI = err.RedirectURI + "?" + v.Encode()
		}
		http.Redirect(w, r, redirectURI, http.StatusSeeOther)
	}
	return http.HandlerFunc(hf)
}

func tokenErr(w http.ResponseWriter, typ, description string, statusCode int) error {
	data := struct {
		Error       string `json:"error"`
		Description string `json:"error_description,omitempty"`
	}{typ, description}
	body, err := json.Marshal(data)
	if err != nil {
		return fmt.Errorf("failed to marshal token error response: %v", err)
	}
	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Content-Length", strconv.Itoa(len(body)))
	w.WriteHeader(statusCode)
	w.Write(body)
	return nil
}

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
)

const (
	scopeOfflineAccess     = "offline_access" // Request a refresh token.
	scopeOpenID            = "openid"
	scopeGroups            = "groups"
	scopeEmail             = "email"
	scopeProfile           = "profile"
	scopeFederatedID       = "federated:id"
	scopeCrossClientPrefix = "audience:server:client_id:"
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
)

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
		case scopeOfflineAccess:
			s.OfflineAccess = true
		case scopeGroups:
			s.Groups = true
		}
	}
	return s
}

// Determine the signature algorithm for a JWT.
func signatureAlgorithm(jwk *jose.JSONWebKey) (alg jose.SignatureAlgorithm, err error) {
	if jwk.Key == nil {
		return alg, errors.New("no signing key")
	}
	switch key := jwk.Key.(type) {
	case *rsa.PrivateKey:
		// Because OIDC mandates that we support RS256, we always return that
		// value. In the future, we might want to make this configurable on a
		// per client basis. For example allowing PS256 or ECDSA variants.
		//
		// See https://github.com/dexidp/dex/issues/692
		return jose.RS256, nil
	case *ecdsa.PrivateKey:
		// We don't actually support ECDSA keys yet, but they're tested for
		// in case we want to in the future.
		//
		// These values are prescribed depending on the ECDSA key type. We
		// can't return different values.
		switch key.Params() {
		case elliptic.P256().Params():
			return jose.ES256, nil
		case elliptic.P384().Params():
			return jose.ES384, nil
		case elliptic.P521().Params():
			return jose.ES512, nil
		default:
			return alg, errors.New("unsupported ecdsa curve")
		}
	default:
		return alg, fmt.Errorf("unsupported signing key type %T", key)
	}
}

func signPayload(key *jose.JSONWebKey, alg jose.SignatureAlgorithm, payload []byte) (jws string, err error) {
	signingKey := jose.SigningKey{Key: key, Algorithm: alg}

	signer, err := jose.NewSigner(signingKey, &jose.SignerOptions{})
	if err != nil {
		return "", fmt.Errorf("new signer: %v", err)
	}
	signature, err := signer.Sign(payload)
	if err != nil {
		return "", fmt.Errorf("signing payload: %v", err)
	}
	return signature.CompactSerialize()
}

// The hash algorithm for the at_hash is determined by the signing
// algorithm used for the id_token. From the spec:
//
//	...the hash algorithm used is the hash algorithm used in the alg Header
//	Parameter of the ID Token's JOSE Header. For instance, if the alg is RS256,
//	hash the access_token value with SHA-256
//
// https://openid.net/specs/openid-connect-core-1_0.html#ImplicitIDToken
var hashForSigAlg = map[jose.SignatureAlgorithm]func() hash.Hash{
	jose.RS256: sha256.New,
	jose.RS384: sha512.New384,
	jose.RS512: sha512.New,
	jose.ES256: sha256.New,
	jose.ES384: sha512.New384,
	jose.ES512: sha512.New,
}

// Compute an at_hash from a raw access token and a signature algorithm
//
// See: https://openid.net/specs/openid-connect-core-1_0.html#ImplicitIDToken
func accessTokenHash(alg jose.SignatureAlgorithm, accessToken string) (string, error) {
	newHash, ok := hashForSigAlg[alg]
	if !ok {
		return "", fmt.Errorf("unsupported signature algorithm: %s", alg)
	}

	hashFunc := newHash()
	if _, err := io.WriteString(hashFunc, accessToken); err != nil {
		return "", fmt.Errorf("computing hash: %v", err)
	}
	sum := hashFunc.Sum(nil)
	return base64.RawURLEncoding.EncodeToString(sum[:len(sum)/2]), nil
}

type audience []string

func (a audience) contains(aud string) bool {
	for _, e := range a {
		if aud == e {
			return true
		}
	}
	return false
}

func (a audience) MarshalJSON() ([]byte, error) {
	if len(a) == 1 {
		return json.Marshal(a[0])
	}
	return json.Marshal([]string(a))
}

type idTokenClaims struct {
	Issuer           string   `json:"iss"`
	Subject          string   `json:"sub"`
	Audience         audience `json:"aud"`
	Expiry           int64    `json:"exp"`
	IssuedAt         int64    `json:"iat"`
	AuthorizingParty string   `json:"azp,omitempty"`
	Nonce            string   `json:"nonce,omitempty"`

	AccessTokenHash string `json:"at_hash,omitempty"`
	CodeHash        string `json:"c_hash,omitempty"`

	Email         string `json:"email,omitempty"`
	EmailVerified *bool  `json:"email_verified,omitempty"`

	Groups []string `json:"groups,omitempty"`

	Name              string `json:"name,omitempty"`
	PreferredUsername string `json:"preferred_username,omitempty"`

	FederatedIDClaims *federatedIDClaims `json:"federated_claims,omitempty"`
}

type federatedIDClaims struct {
	ConnectorID string `json:"connector_id,omitempty"`
	UserID      string `json:"user_id,omitempty"`
}

func (s *Server) newAccessToken(clientID string, claims storage.Claims, scopes []string, nonce, connID string) (accessToken string, expiry time.Time, err error) {
	return s.newIDToken(clientID, claims, scopes, nonce, storage.NewID(), "", connID)
}

func (s *Server) newIDToken(clientID string, claims storage.Claims, scopes []string, nonce, accessToken, code, connID string) (idToken string, expiry time.Time, err error) {
	keys, err := s.storage.GetKeys()
	if err != nil {
		s.logger.Errorf("Failed to get keys: %v", err)
		return "", expiry, err
	}

	signingKey := keys.SigningKey
	if signingKey == nil {
		return "", expiry, fmt.Errorf("no key to sign payload with")
	}
	signingAlg, err := signatureAlgorithm(signingKey)
	if err != nil {
		return "", expiry, err
	}

	issuedAt := s.now()
	expiry = issuedAt.Add(s.idTokensValidFor)

	sub := &internal.IDTokenSubject{
		UserId: claims.UserID,
		ConnId: connID,
	}

	subjectString, err := internal.Marshal(sub)
	if err != nil {
		s.logger.Errorf("failed to marshal offline session ID: %v", err)
		return "", expiry, fmt.Errorf("failed to marshal offline session ID: %v", err)
	}

	tok := idTokenClaims{
		Issuer:   s.issuerURL.String(),
		Subject:  subjectString,
		Nonce:    nonce,
		Expiry:   expiry.Unix(),
		IssuedAt: issuedAt.Unix(),
	}

	if accessToken != "" {
		atHash, err := accessTokenHash(signingAlg, accessToken)
		if err != nil {
			s.logger.Errorf("error computing at_hash: %v", err)
			return "", expiry, fmt.Errorf("error computing at_hash: %v", err)
		}
		tok.AccessTokenHash = atHash
	}

	if code != "" {
		cHash, err := accessTokenHash(signingAlg, code)
		if err != nil {
			s.logger.Errorf("error computing c_hash: %v", err)
			return "", expiry, fmt.Errorf("error computing c_hash: #{err}")
		}
		tok.CodeHash = cHash
	}

	for _, scope := range scopes {
		switch {
		case scope == scopeEmail:
			tok.Email = claims.Email
			tok.EmailVerified = &claims.EmailVerified
		case scope == scopeGroups:
			tok.Groups = claims.Groups
		case scope == scopeProfile:
			tok.Name = claims.Username
			tok.PreferredUsername = claims.PreferredUsername
		case scope == scopeFederatedID:
			tok.FederatedIDClaims = &federatedIDClaims{
				ConnectorID: connID,
				UserID:      claims.UserID,
			}
		default:
			peerID, ok := parseCrossClientScope(scope)
			if !ok {
				// Ignore unknown scopes. These are already validated during the
				// initial auth request.
				continue
			}
			isTrusted, err := s.validateCrossClientTrust(clientID, peerID)
			if err != nil {
				return "", expiry, err
			}
			if !isTrusted {
				// TODO(ericchiang): propagate this error to the client.
				return "", expiry, fmt.Errorf("peer (%s) does not trust client", peerID)
			}
			tok.Audience = append(tok.Audience, peerID)
		}
	}

	if len(tok.Audience) == 0 {
		// Client didn't ask for cross client audience. Set the current
		// client as the audience.
		tok.Audience = audience{clientID}
	} else {
		// Client asked for cross client audience:
		// if the current client was not requested explicitly
		if !tok.Audience.contains(clientID) {
			// by default it becomes one of entries in Audience
			tok.Audience = append(tok.Audience, clientID)
		}
		// The current client becomes the authorizing party.
		tok.AuthorizingParty = clientID
	}

	payload, err := json.Marshal(tok)
	if err != nil {
		return "", expiry, fmt.Errorf("could not serialize claims: %v", err)
	}

	if idToken, err = signPayload(signingKey, signingAlg, payload); err != nil {
		return "", expiry, fmt.Errorf("failed to sign payload: %v", err)
	}
	return idToken, expiry, nil
}

// parse the initial request from the OAuth2 client.
func (s *Server) parseAuthorizationRequest(r *http.Request) (*storage.AuthRequest, error) {
	if err := r.ParseForm(); err != nil {
		return nil, newDisplayedErr(http.StatusBadRequest, "Failed to parse request.")
	}
	q := r.Form
	redirectURI, err := url.QueryUnescape(q.Get("redirect_uri"))
	if err != nil {
		return nil, newDisplayedErr(http.StatusBadRequest, "No redirect_uri provided.")
	}

	clientID := q.Get("client_id")
	state := q.Get("state")
	nonce := q.Get("nonce")
	connectorID := q.Get("connector_id")
	// Some clients, like the old go-oidc, provide extra whitespace. Tolerate this.
	scopes := strings.Fields(q.Get("scope"))
	responseTypes := strings.Fields(q.Get("response_type"))

	codeChallenge := q.Get("code_challenge")
	codeChallengeMethod := q.Get("code_challenge_method")

	if codeChallengeMethod == "" {
		codeChallengeMethod = codeChallengeMethodPlain
	}

	client, err := s.storage.GetClient(clientID)
	if err != nil {
		if err == storage.ErrNotFound {
			return nil, newDisplayedErr(http.StatusNotFound, "Invalid client_id (%q).", clientID)
		}
		s.logger.Errorf("Failed to get client: %v", err)
		return nil, newDisplayedErr(http.StatusInternalServerError, "Database error.")
	}

	if !validateRedirectURI(client, redirectURI) {
		return nil, newDisplayedErr(http.StatusBadRequest, "Unregistered redirect_uri (%q).", redirectURI)
	}
	if redirectURI == deviceCallbackURI && client.Public {
		redirectURI = s.issuerURL.Path + deviceCallbackURI
	}

	// From here on out, we want to redirect back to the client with an error.
	newRedirectedErr := func(typ, format string, a ...interface{}) *redirectedAuthErr {
		return &redirectedAuthErr{state, redirectURI, typ, fmt.Sprintf(format, a...)}
	}

	if connectorID != "" {
		connectors, err := s.storage.ListConnectors()
		if err != nil {
			s.logger.Errorf("Failed to list connectors: %v", err)
			return nil, newRedirectedErr(errServerError, "Unable to retrieve connectors")
		}
		if !validateConnectorID(connectors, connectorID) {
			return nil, newRedirectedErr(errInvalidRequest, "Invalid ConnectorID")
		}
	}

	// dex doesn't support request parameter and must return request_not_supported error
	// https://openid.net/specs/openid-connect-core-1_0.html#6.1
	if q.Get("request") != "" {
		return nil, newRedirectedErr(errRequestNotSupported, "Server does not support request parameter.")
	}

	if codeChallengeMethod != codeChallengeMethodS256 && codeChallengeMethod != codeChallengeMethodPlain {
		description := fmt.Sprintf("Unsupported PKCE challenge method (%q).", codeChallengeMethod)
		return nil, newRedirectedErr(errInvalidRequest, description)
	}

	var (
		unrecognized  []string
		invalidScopes []string
	)
	hasOpenIDScope := false
	for _, scope := range scopes {
		switch scope {
		case scopeOpenID:
			hasOpenIDScope = true
		case scopeOfflineAccess, scopeEmail, scopeProfile, scopeGroups, scopeFederatedID:
		default:
			peerID, ok := parseCrossClientScope(scope)
			if !ok {
				unrecognized = append(unrecognized, scope)
				continue
			}

			isTrusted, err := s.validateCrossClientTrust(clientID, peerID)
			if err != nil {
				return nil, newRedirectedErr(errServerError, "Internal server error.")
			}
			if !isTrusted {
				invalidScopes = append(invalidScopes, scope)
			}
		}
	}
	if !hasOpenIDScope {
		return nil, newRedirectedErr(errInvalidScope, `Missing required scope(s) ["openid"].`)
	}
	if len(unrecognized) > 0 {
		return nil, newRedirectedErr(errInvalidScope, "Unrecognized scope(s) %q", unrecognized)
	}
	if len(invalidScopes) > 0 {
		return nil, newRedirectedErr(errInvalidScope, "Client can't request scope(s) %q", invalidScopes)
	}

	var rt struct {
		code    bool
		idToken bool
		token   bool
	}

	for _, responseType := range responseTypes {
		switch responseType {
		case responseTypeCode:
			rt.code = true
		case responseTypeIDToken:
			rt.idToken = true
		case responseTypeToken:
			rt.token = true
		default:
			return nil, newRedirectedErr(errInvalidRequest, "Invalid response type %q", responseType)
		}

		if !s.supportedResponseTypes[responseType] {
			return nil, newRedirectedErr(errUnsupportedResponseType, "Unsupported response type %q", responseType)
		}
	}

	if len(responseTypes) == 0 {
		return nil, newRedirectedErr(errInvalidRequest, "No response_type provided")
	}

	if rt.token && !rt.code && !rt.idToken {
		// "token" can't be provided by its own.
		//
		// https://openid.net/specs/openid-connect-core-1_0.html#Authentication
		return nil, newRedirectedErr(errInvalidRequest, "Response type 'token' must be provided with type 'id_token' and/or 'code'")
	}
	if !rt.code {
		// Either "id_token token" or "id_token" has been provided which implies the
		// implicit flow. Implicit flow requires a nonce value.
		//
		// https://openid.net/specs/openid-connect-core-1_0.html#ImplicitAuthRequest
		if nonce == "" {
			return nil, newRedirectedErr(errInvalidRequest, "Response type 'token' requires a 'nonce' value.")
		}
	}
	if rt.token {
		if redirectURI == redirectURIOOB {
			err := fmt.Sprintf("Cannot use response type 'token' with redirect_uri '%s'.", redirectURIOOB)
			return nil, newRedirectedErr(errInvalidRequest, err)
		}
	}

	return &storage.AuthRequest{
		ID:                  storage.NewID(),
		ClientID:            client.ID,
		State:               state,
		Nonce:               nonce,
		ForceApprovalPrompt: q.Get("approval_prompt") == "force",
		Scopes:              scopes,
		RedirectURI:         redirectURI,
		ResponseTypes:       responseTypes,
		ConnectorID:         connectorID,
		PKCE: storage.PKCE{
			CodeChallenge:       codeChallenge,
			CodeChallengeMethod: codeChallengeMethod,
		},
		HMACKey: storage.NewHMACKey(crypto.SHA256),
	}, nil
}

func parseCrossClientScope(scope string) (peerID string, ok bool) {
	if ok = strings.HasPrefix(scope, scopeCrossClientPrefix); ok {
		peerID = scope[len(scopeCrossClientPrefix):]
	}
	return
}

func (s *Server) validateCrossClientTrust(clientID, peerID string) (trusted bool, err error) {
	if peerID == clientID {
		return true, nil
	}
	peer, err := s.storage.GetClient(peerID)
	if err != nil {
		if err != storage.ErrNotFound {
			s.logger.Errorf("Failed to get client: %v", err)
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

func validateRedirectURI(client storage.Client, redirectURI string) bool {
	// Allow named RedirectURIs for both public and non-public clients.
	// This is required make PKCE-enabled web apps work, when configured as public clients.
	for _, uri := range client.RedirectURIs {
		if redirectURI == uri {
			return true
		}
	}
	// For non-public clients or when RedirectURIs is set, we allow only explicitly named RedirectURIs.
	// Otherwise, we check below for special URIs used for desktop or mobile apps.
	if !client.Public || len(client.RedirectURIs) > 0 {
		return false
	}

	if redirectURI == redirectURIOOB || redirectURI == deviceCallbackURI {
		return true
	}

	// verify that the host is of form "http://localhost:(port)(path)" or "http://localhost(path)"
	u, err := url.Parse(redirectURI)
	if err != nil {
		return false
	}
	if u.Scheme != "http" {
		return false
	}
	if u.Host == "localhost" {
		return true
	}
	host, _, err := net.SplitHostPort(u.Host)
	return err == nil && host == "localhost"
}

func validateConnectorID(connectors []storage.Connector, connectorID string) bool {
	for _, c := range connectors {
		if c.ID == connectorID {
			return true
		}
	}
	return false
}

// storageKeySet implements the oidc.KeySet interface backed by Dex storage
type storageKeySet struct {
	storage.Storage
}

func (s *storageKeySet) VerifySignature(_ context.Context, jwt string) (payload []byte, err error) {
	jws, err := jose.ParseSigned(jwt, []jose.SignatureAlgorithm{jose.RS256, jose.RS384, jose.RS512, jose.ES256, jose.ES384, jose.ES512})
	if err != nil {
		return nil, err
	}

	keyID := ""
	for _, sig := range jws.Signatures {
		keyID = sig.Header.KeyID
		break
	}

	skeys, err := s.Storage.GetKeys()
	if err != nil {
		return nil, err
	}

	keys := []*jose.JSONWebKey{skeys.SigningKeyPub}
	for _, vk := range skeys.VerificationKeys {
		keys = append(keys, vk.PublicKey)
	}

	for _, key := range keys {
		if keyID == "" || key.KeyID == keyID {
			if payload, err := jws.Verify(key); err == nil {
				return payload, nil
			}
		}
	}

	return nil, errors.New("failed to verify id token signature")
}
