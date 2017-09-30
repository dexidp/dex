package server

import (
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

	jose "gopkg.in/square/go-jose.v2"

	"github.com/coreos/dex/connector"
	"github.com/coreos/dex/server/internal"
	"github.com/coreos/dex/storage"
)

// TODO(ericchiang): clean this file up and figure out more idiomatic error handling.

// authErr is an error response to an authorization request.
// See: https://tools.ietf.org/html/rfc6749#section-4.1.2.1
type authErr struct {
	State       string
	RedirectURI string
	Type        string
	Description string
}

func (err *authErr) Status() int {
	if err.State == errServerError {
		return http.StatusInternalServerError
	}
	return http.StatusBadRequest
}

func (err *authErr) Error() string {
	return err.Description
}

func (err *authErr) Handle() (http.Handler, bool) {
	// Didn't get a valid redirect URI.
	if err.RedirectURI == "" {
		return nil, false
	}

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
	return http.HandlerFunc(hf), true
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
	scopeCrossClientPrefix = "audience:server:client_id:"
)

const (
	redirectURIOOB = "urn:ietf:wg:oauth:2.0:oob"
)

const (
	grantTypeAuthorizationCode = "authorization_code"
	grantTypeRefreshToken      = "refresh_token"
)

const (
	responseTypeCode    = "code"     // "Regular" flow
	responseTypeToken   = "token"    // Implicit flow for frontend apps.
	responseTypeIDToken = "id_token" // ID Token in url fragment
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
		// See https://github.com/coreos/dex/issues/692
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
		return "", fmt.Errorf("new signier: %v", err)
	}
	signature, err := signer.Sign(payload)
	if err != nil {
		return "", fmt.Errorf("signing payload: %v", err)
	}
	return signature.CompactSerialize()
}

// The hash algorithm for the at_hash is detemrined by the signing
// algorithm used for the id_token. From the spec:
//
//    ...the hash algorithm used is the hash algorithm used in the alg Header
//    Parameter of the ID Token's JOSE Header. For instance, if the alg is RS256,
//    hash the access_token value with SHA-256
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

	hash := newHash()
	if _, err := io.WriteString(hash, accessToken); err != nil {
		return "", fmt.Errorf("computing hash: %v", err)
	}
	sum := hash.Sum(nil)
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

	Email         string `json:"email,omitempty"`
	EmailVerified *bool  `json:"email_verified,omitempty"`

	Groups []string `json:"groups,omitempty"`

	Name string `json:"name,omitempty"`
}

func (s *Server) newIDToken(clientID string, claims storage.Claims, scopes []string, nonce, accessToken, connID string) (idToken string, expiry time.Time, err error) {
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

	for _, scope := range scopes {
		switch {
		case scope == scopeEmail:
			tok.Email = claims.Email
			tok.EmailVerified = &claims.EmailVerified
		case scope == scopeGroups:
			tok.Groups = claims.Groups
		case scope == scopeProfile:
			tok.Name = claims.Username
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
func (s *Server) parseAuthorizationRequest(r *http.Request) (req storage.AuthRequest, oauth2Err *authErr) {
	if err := r.ParseForm(); err != nil {
		return req, &authErr{"", "", errInvalidRequest, "Failed to parse request body."}
	}
	q := r.Form
	redirectURI, err := url.QueryUnescape(q.Get("redirect_uri"))
	if err != nil {
		return req, &authErr{"", "", errInvalidRequest, "No redirect_uri provided."}
	}

	clientID := q.Get("client_id")
	state := q.Get("state")
	nonce := q.Get("nonce")
	// Some clients, like the old go-oidc, provide extra whitespace. Tolerate this.
	scopes := strings.Fields(q.Get("scope"))
	responseTypes := strings.Fields(q.Get("response_type"))

	client, err := s.storage.GetClient(clientID)
	if err != nil {
		if err == storage.ErrNotFound {
			description := fmt.Sprintf("Invalid client_id (%q).", clientID)
			return req, &authErr{"", "", errUnauthorizedClient, description}
		}
		s.logger.Errorf("Failed to get client: %v", err)
		return req, &authErr{"", "", errServerError, ""}
	}

	if !validateRedirectURI(client, redirectURI) {
		description := fmt.Sprintf("Unregistered redirect_uri (%q).", redirectURI)
		return req, &authErr{"", "", errInvalidRequest, description}
	}

	// From here on out, we want to redirect back to the client with an error.
	newErr := func(typ, format string, a ...interface{}) *authErr {
		return &authErr{state, redirectURI, typ, fmt.Sprintf(format, a...)}
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
		case scopeOfflineAccess, scopeEmail, scopeProfile, scopeGroups:
		default:
			peerID, ok := parseCrossClientScope(scope)
			if !ok {
				unrecognized = append(unrecognized, scope)
				continue
			}

			isTrusted, err := s.validateCrossClientTrust(clientID, peerID)
			if err != nil {
				return req, newErr(errServerError, "Internal server error.")
			}
			if !isTrusted {
				invalidScopes = append(invalidScopes, scope)
			}
		}
	}
	if !hasOpenIDScope {
		return req, newErr("invalid_scope", `Missing required scope(s) ["openid"].`)
	}
	if len(unrecognized) > 0 {
		return req, newErr("invalid_scope", "Unrecognized scope(s) %q", unrecognized)
	}
	if len(invalidScopes) > 0 {
		return req, newErr("invalid_scope", "Client can't request scope(s) %q", invalidScopes)
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
			return req, newErr("invalid_request", "Invalid response type %q", responseType)
		}

		if !s.supportedResponseTypes[responseType] {
			return req, newErr(errUnsupportedResponseType, "Unsupported response type %q", responseType)
		}
	}

	if len(responseTypes) == 0 {
		return req, newErr("invalid_requests", "No response_type provided")
	}

	if rt.token && !rt.code && !rt.idToken {
		// "token" can't be provided by its own.
		//
		// https://openid.net/specs/openid-connect-core-1_0.html#Authentication
		return req, newErr("invalid_request", "Response type 'token' must be provided with type 'id_token' and/or 'code'")
	}
	if !rt.code {
		// Either "id_token code" or "id_token" has been provided which implies the
		// implicit flow. Implicit flow requires a nonce value.
		//
		// https://openid.net/specs/openid-connect-core-1_0.html#ImplicitAuthRequest
		if nonce == "" {
			return req, newErr("invalid_request", "Response type 'token' requires a 'nonce' value.")
		}
	}
	if rt.token {
		if redirectURI == redirectURIOOB {
			err := fmt.Sprintf("Cannot use response type 'token' with redirect_uri '%s'.", redirectURIOOB)
			return req, newErr("invalid_request", err)
		}
	}

	return storage.AuthRequest{
		ID:                  storage.NewID(),
		ClientID:            client.ID,
		State:               state,
		Nonce:               nonce,
		ForceApprovalPrompt: q.Get("approval_prompt") == "force",
		Scopes:              scopes,
		RedirectURI:         redirectURI,
		ResponseTypes:       responseTypes,
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
	if !client.Public {
		for _, uri := range client.RedirectURIs {
			if redirectURI == uri {
				return true
			}
		}
		return false
	}

	if redirectURI == redirectURIOOB {
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
