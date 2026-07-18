package authflow

import (
	"context"
	"crypto"
	"fmt"
	"net"
	"net/http"
	"net/url"
	"slices"
	"strconv"
	"strings"

	"github.com/coreos/go-oidc/v3/oidc"

	conns "github.com/dexidp/dex/server/connectors"
	"github.com/dexidp/dex/server/oauth2"
	"github.com/dexidp/dex/server/signer"
	"github.com/dexidp/dex/server/tokens"
	"github.com/dexidp/dex/storage"
)

// request.go parses and validates the OAuth2 /auth authorization request into a
// storage.AuthRequest, and defines the request-error surface (displayed vs
// redirected).

// displayedAuthErr is an error that should be displayed to the user as a web page.
// See RFC 6749 §4.1.2.1: an invalid client_id or redirect_uri is shown, not
// redirected.
type displayedAuthErr struct {
	Status      int
	Description string
}

func (err *displayedAuthErr) Error() string { return err.Description }

// newDisplayedErr builds a displayedAuthErr.
func newDisplayedErr(status int, format string, a ...interface{}) *displayedAuthErr {
	return &displayedAuthErr{status, fmt.Sprintf(format, a...)}
}

// redirectedAuthErr is an error reported back to the client by 302 redirect.
type redirectedAuthErr struct {
	State       string
	RedirectURI string
	Type        string
	Description string
}

func (err *redirectedAuthErr) Error() string { return err.Description }

// Handler returns an http.Handler that redirects to the client with the error.
func (err *redirectedAuthErr) Handler() http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		v := url.Values{}
		v.Add("state", err.State)
		v.Add("error", err.Type)
		if err.Description != "" {
			v.Add("error_description", err.Description)
		}

		// Parse the redirect URI to ensure it's valid before redirecting.
		u, parseErr := url.Parse(err.RedirectURI)
		if parseErr != nil {
			http.Error(w, "Invalid redirect URI", http.StatusBadRequest)
			return
		}

		query := u.Query()
		for key, values := range v {
			for _, value := range values {
				query.Add(key, value)
			}
		}
		u.RawQuery = query.Encode()

		http.Redirect(w, r, u.String(), http.StatusSeeOther)
	})
}

// redirectWithError redirects back to the client with an OAuth2 error response.
// Used for prompt=none when login or consent is required.
func (h *Handler) redirectWithError(w http.ResponseWriter, r *http.Request, authReq *storage.AuthRequest, errType, description string) {
	err := &redirectedAuthErr{State: authReq.State, RedirectURI: authReq.RedirectURI, Type: errType, Description: description}
	err.Handler().ServeHTTP(w, r)
}

func validateRedirectURI(client storage.Client, redirectURI string) bool {
	// Allow named RedirectURIs for both public and non-public clients.
	// This is required to make PKCE-enabled web apps work when configured as public clients.
	for _, uri := range client.RedirectURIs {
		if redirectURI == uri {
			return true
		}
	}
	// For non-public clients or when RedirectURIs is set, we allow only explicitly named RedirectURIs.
	if !client.Public || len(client.RedirectURIs) > 0 {
		return false
	}

	if redirectURI == oauth2.RedirectURIOOB || redirectURI == oauth2.DeviceCallbackURI {
		return true
	}

	// Verify the host is a loopback form ("http://localhost:(port)(path)" etc).
	u, err := url.Parse(redirectURI)
	if err != nil {
		return false
	}
	if u.Scheme != "http" {
		return false
	}
	return isHostLocal(u.Host)
}

func isHostLocal(host string) bool {
	if host == "localhost" || net.ParseIP(host).IsLoopback() {
		return true
	}

	host, _, err := net.SplitHostPort(host)
	if err != nil {
		return false
	}

	return host == "localhost" || net.ParseIP(host).IsLoopback()
}

func validateConnectorID(connectors []storage.Connector, connectorID string) bool {
	for _, c := range connectors {
		if c.ID == connectorID {
			return true
		}
	}
	return false
}

// sessionMatchesHint checks whether the session's user identity matches the
// subject from an id_token_hint by encoding the session's (userID, connectorID)
// via GenSubject and doing a string comparison.
func sessionMatchesHint(session *storage.AuthSession, hintSubject string) bool {
	if session == nil {
		return false
	}
	encoded, err := tokens.GenSubject(session.UserID, session.ConnectorID)
	if err != nil {
		return false
	}
	return encoded == hintSubject
}

// PKCEConfig holds PKCE (Proof Key for Code Exchange) settings.
type PKCEConfig struct {
	// If true, PKCE is required for all authorization code flows.
	Enforce bool
	// Supported code challenge methods. Defaults to ["S256", "plain"].
	CodeChallengeMethodsSupported []string
}

// ValidateIDTokenHint verifies the signature and issuer of an id_token_hint.
// Expired tokens are accepted per OIDC Core 1.0 §3.1.2.1. It returns the verified
// token so callers can extract Subject, Audience, etc.
func (h *Handler) validateIDTokenHint(ctx context.Context, hint string) (*oidc.IDToken, error) {
	verifier := oidc.NewVerifier(h.issuerURL.String(), &signer.KeySet{Signer: h.signer}, &oidc.Config{
		SkipExpiryCheck: true,
		// SkipClientIDCheck is set because the hint may originate from any client that
		// Dex issued a token to — the caller does not know the expected audience in advance.
		// The signature verification via signer.KeySet already guarantees the token was
		// issued by this server. Dex does the client id check later during session validation.
		SkipClientIDCheck: true,
	})
	return verifier.Verify(ctx, hint)
}

// Parse parses the initial request from the OAuth2 client. It returns the auth
// request, the raw subject from id_token_hint (empty if not provided), and any
// error (a *displayedAuthErr or *redirectedAuthErr).
func (h *Handler) parseAuthorizationRequest(r *http.Request) (*storage.AuthRequest, string, error) {
	ctx := r.Context()
	if err := r.ParseForm(); err != nil {
		return nil, "", newDisplayedErr(http.StatusBadRequest, "Failed to parse request.")
	}
	q := r.Form
	// r.ParseForm already URL-decodes query values once; decoding redirect_uri a
	// second time created a normalization differential with the token endpoint.
	redirectURI := q.Get("redirect_uri")

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
		codeChallengeMethod = oauth2.PKCEMethodPlain
	}

	client, err := h.storage.GetClient(ctx, clientID)
	if err != nil {
		if err == storage.ErrNotFound {
			h.logger.ErrorContext(ctx, "invalid client_id provided", "client_id", clientID)
			return nil, "", newDisplayedErr(http.StatusNotFound, "Invalid client_id.")
		}
		h.logger.ErrorContext(ctx, "failed to get client", "err", err)
		return nil, "", newDisplayedErr(http.StatusInternalServerError, "Database error.")
	}

	if !validateRedirectURI(client, redirectURI) {
		h.logger.ErrorContext(ctx, "unregistered redirect_uri", "redirect_uri", redirectURI, "client_id", clientID)
		return nil, "", newDisplayedErr(http.StatusBadRequest, "Unregistered redirect_uri.")
	}
	if redirectURI == oauth2.DeviceCallbackURI && client.Public {
		redirectURI = h.absPath(oauth2.DeviceCallbackURI)
	}

	// From here on out, we want to redirect back to the client with an error.
	newredirectedAuthErr := func(typ, format string, a ...interface{}) *redirectedAuthErr {
		return &redirectedAuthErr{state, redirectURI, typ, fmt.Sprintf(format, a...)}
	}

	if connectorID != "" {
		connectors, err := h.storage.ListConnectors(ctx)
		if err != nil {
			h.logger.ErrorContext(ctx, "failed to list connectors", "err", err)
			return nil, "", newredirectedAuthErr(oauth2.ServerError, "Unable to retrieve connectors")
		}
		if !validateConnectorID(connectors, connectorID) {
			return nil, "", newredirectedAuthErr(oauth2.InvalidRequest, "Invalid ConnectorID")
		}
		if !conns.ConnectorAllowed(client.AllowedConnectors, connectorID) {
			return nil, "", newredirectedAuthErr(oauth2.InvalidRequest, "Connector not allowed for this client")
		}
	}

	// dex doesn't support the request parameter and must return request_not_supported.
	// https://openid.net/specs/openid-connect-core-1_0.html#6.1
	if q.Get("request") != "" {
		return nil, "", newredirectedAuthErr(oauth2.RequestNotSupported, "Server does not support request parameter.")
	}

	if codeChallenge != "" && !slices.Contains(h.pkce.CodeChallengeMethodsSupported, codeChallengeMethod) {
		return nil, "", newredirectedAuthErr(oauth2.InvalidRequest, "Unsupported PKCE challenge method (%q).", codeChallengeMethod)
	}

	// Enforce PKCE if configured.
	// https://datatracker.ietf.org/doc/html/draft-ietf-oauth-v2-1-12#section-4.1.1
	if h.pkce.Enforce && codeChallenge == "" {
		return nil, "", newredirectedAuthErr(oauth2.InvalidRequest, "PKCE is required. The code_challenge parameter must be provided.")
	}

	var (
		unrecognized  []string
		invalidScopes []string
	)
	hasOpenIDScope := false
	for _, scope := range scopes {
		switch scope {
		case tokens.ScopeOpenID:
			hasOpenIDScope = true
		case tokens.ScopeOfflineAccess, tokens.ScopeEmail, tokens.ScopeProfile, tokens.ScopeGroups, tokens.ScopeFederatedID:
		default:
			peerID, ok := tokens.ParseCrossClientScope(scope)
			if !ok {
				unrecognized = append(unrecognized, scope)
				continue
			}

			isTrusted, err := tokens.CrossClientTrusted(ctx, h.storage, clientID, peerID)
			if err != nil {
				return nil, "", newredirectedAuthErr(oauth2.ServerError, "Internal server error.")
			}
			if !isTrusted {
				invalidScopes = append(invalidScopes, scope)
			}
		}
	}
	if !hasOpenIDScope {
		return nil, "", newredirectedAuthErr(oauth2.InvalidScope, `Missing required scope(s) ["openid"].`)
	}
	if len(unrecognized) > 0 {
		return nil, "", newredirectedAuthErr(oauth2.InvalidScope, "Unrecognized scope(s) %q", unrecognized)
	}
	if len(invalidScopes) > 0 {
		return nil, "", newredirectedAuthErr(oauth2.InvalidScope, "Client can't request scope(s) %q", invalidScopes)
	}

	var rt struct {
		code    bool
		idToken bool
		token   bool
	}

	for _, responseType := range responseTypes {
		switch responseType {
		case oauth2.ResponseTypeCode:
			rt.code = true
		case oauth2.ResponseTypeIDToken:
			rt.idToken = true
		case oauth2.ResponseTypeToken:
			rt.token = true
		default:
			return nil, "", newredirectedAuthErr(oauth2.InvalidRequest, "Invalid response type %q", responseType)
		}

		if !h.supportedResponseTypes[responseType] {
			return nil, "", newredirectedAuthErr(oauth2.UnsupportedResponseType, "Unsupported response type %q", responseType)
		}
	}

	if len(responseTypes) == 0 {
		return nil, "", newredirectedAuthErr(oauth2.InvalidRequest, "No response_type provided")
	}

	if rt.token && !rt.code && !rt.idToken {
		// "token" can't be provided on its own.
		// https://openid.net/specs/openid-connect-core-1_0.html#Authentication
		return nil, "", newredirectedAuthErr(oauth2.InvalidRequest, "Response type 'token' must be provided with type 'id_token' and/or 'code'")
	}
	if !rt.code {
		// Either "id_token token" or "id_token" implies the implicit flow, which
		// requires a nonce value.
		// https://openid.net/specs/openid-connect-core-1_0.html#ImplicitAuthRequest
		if nonce == "" {
			return nil, "", newredirectedAuthErr(oauth2.InvalidRequest, "Response type 'token' requires a 'nonce' value.")
		}
	}
	if rt.token {
		if redirectURI == oauth2.RedirectURIOOB {
			return nil, "", newredirectedAuthErr(oauth2.InvalidRequest, "Cannot use response type 'token' with redirect_uri '%s'.", oauth2.RedirectURIOOB)
		}
	}

	prompt, err := oauth2.ParsePrompt(q.Get("prompt"))
	if err != nil {
		return nil, "", newredirectedAuthErr(oauth2.InvalidRequest, "Invalid prompt parameter: %v", err)
	}

	// Parse max_age: -1 means not specified.
	maxAge := -1
	if maxAgeStr := q.Get("max_age"); maxAgeStr != "" {
		v, err := strconv.Atoi(maxAgeStr)
		if err != nil || v < 0 {
			return nil, "", newredirectedAuthErr(oauth2.InvalidRequest, "Invalid max_age value %q", maxAgeStr)
		}
		maxAge = v
	}

	// OIDC prompt=consent implies force approval.
	forceApproval := q.Get("approval_prompt") == "force" || prompt.Consent()

	// Validate id_token_hint if provided (OIDC Core 1.0 §3.1.2.1).
	var idTokenHintSubject string
	if hint := q.Get("id_token_hint"); hint != "" {
		idToken, err := h.validateIDTokenHint(ctx, hint)
		if err != nil {
			return nil, "", newredirectedAuthErr(oauth2.InvalidRequest, "Invalid id_token_hint.")
		}
		idTokenHintSubject = idToken.Subject
	}

	return &storage.AuthRequest{
		ID:                  storage.NewID(),
		ClientID:            client.ID,
		State:               state,
		Nonce:               nonce,
		ForceApprovalPrompt: forceApproval,
		Prompt:              prompt.String(),
		MaxAge:              maxAge,
		Scopes:              scopes,
		RedirectURI:         redirectURI,
		ResponseTypes:       responseTypes,
		ConnectorID:         connectorID,
		PKCE: storage.PKCE{
			CodeChallenge:       codeChallenge,
			CodeChallengeMethod: codeChallengeMethod,
		},
		HMACKey: storage.NewHMACKey(crypto.SHA256),
	}, idTokenHintSubject, nil
}
