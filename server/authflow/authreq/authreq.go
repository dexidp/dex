// Package authreq parses and validates the OAuth2 /auth authorization request
// into a storage.AuthRequest, and owns the request-error surface (displayed vs
// redirected). The interactive auth flow delegates parsing to a Parser.
package authreq

import (
	"context"
	"crypto"
	"fmt"
	"log/slog"
	"net/http"
	"net/url"
	"path"
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

// PKCEConfig holds PKCE (Proof Key for Code Exchange) settings.
type PKCEConfig struct {
	// If true, PKCE is required for all authorization code flows.
	Enforce bool
	// Supported code challenge methods. Defaults to ["S256", "plain"].
	CodeChallengeMethodsSupported []string
}

// Parser parses and validates authorization requests.
type Parser struct {
	storage                storage.Storage
	logger                 *slog.Logger
	signer                 signer.Signer
	issuerURL              url.URL
	pkce                   PKCEConfig
	supportedResponseTypes map[string]bool
}

// New builds a Parser.
func New(store storage.Storage, logger *slog.Logger, sig signer.Signer, issuerURL url.URL, pkce PKCEConfig, supportedResponseTypes map[string]bool) *Parser {
	return &Parser{
		storage:                store,
		logger:                 logger,
		signer:                 sig,
		issuerURL:              issuerURL,
		pkce:                   pkce,
		supportedResponseTypes: supportedResponseTypes,
	}
}

func (p *Parser) absPath(pathItems ...string) string {
	paths := make([]string, len(pathItems)+1)
	paths[0] = p.issuerURL.Path
	copy(paths[1:], pathItems)
	return path.Join(paths...)
}

// ValidateIDTokenHint verifies the signature and issuer of an id_token_hint.
// Expired tokens are accepted per OIDC Core 1.0 §3.1.2.1. It returns the verified
// token so callers can extract Subject, Audience, etc.
func (p *Parser) ValidateIDTokenHint(ctx context.Context, hint string) (*oidc.IDToken, error) {
	verifier := oidc.NewVerifier(p.issuerURL.String(), &signer.KeySet{Signer: p.signer}, &oidc.Config{
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
// error (a *DisplayedErr or *RedirectedErr).
func (p *Parser) Parse(r *http.Request) (*storage.AuthRequest, string, error) {
	ctx := r.Context()
	if err := r.ParseForm(); err != nil {
		return nil, "", NewDisplayedErr(http.StatusBadRequest, "Failed to parse request.")
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

	client, err := p.storage.GetClient(ctx, clientID)
	if err != nil {
		if err == storage.ErrNotFound {
			p.logger.ErrorContext(ctx, "invalid client_id provided", "client_id", clientID)
			return nil, "", NewDisplayedErr(http.StatusNotFound, "Invalid client_id.")
		}
		p.logger.ErrorContext(ctx, "failed to get client", "err", err)
		return nil, "", NewDisplayedErr(http.StatusInternalServerError, "Database error.")
	}

	if !validateRedirectURI(client, redirectURI) {
		p.logger.ErrorContext(ctx, "unregistered redirect_uri", "redirect_uri", redirectURI, "client_id", clientID)
		return nil, "", NewDisplayedErr(http.StatusBadRequest, "Unregistered redirect_uri.")
	}
	if redirectURI == oauth2.DeviceCallbackURI && client.Public {
		redirectURI = p.absPath(oauth2.DeviceCallbackURI)
	}

	// From here on out, we want to redirect back to the client with an error.
	newRedirectedErr := func(typ, format string, a ...interface{}) *RedirectedErr {
		return &RedirectedErr{state, redirectURI, typ, fmt.Sprintf(format, a...)}
	}

	if connectorID != "" {
		connectors, err := p.storage.ListConnectors(ctx)
		if err != nil {
			p.logger.ErrorContext(ctx, "failed to list connectors", "err", err)
			return nil, "", newRedirectedErr(oauth2.ServerError, "Unable to retrieve connectors")
		}
		if !validateConnectorID(connectors, connectorID) {
			return nil, "", newRedirectedErr(oauth2.InvalidRequest, "Invalid ConnectorID")
		}
		if !conns.ConnectorAllowed(client.AllowedConnectors, connectorID) {
			return nil, "", newRedirectedErr(oauth2.InvalidRequest, "Connector not allowed for this client")
		}
	}

	// dex doesn't support the request parameter and must return request_not_supported.
	// https://openid.net/specs/openid-connect-core-1_0.html#6.1
	if q.Get("request") != "" {
		return nil, "", newRedirectedErr(oauth2.RequestNotSupported, "Server does not support request parameter.")
	}

	if codeChallenge != "" && !slices.Contains(p.pkce.CodeChallengeMethodsSupported, codeChallengeMethod) {
		return nil, "", newRedirectedErr(oauth2.InvalidRequest, "Unsupported PKCE challenge method (%q).", codeChallengeMethod)
	}

	// Enforce PKCE if configured.
	// https://datatracker.ietf.org/doc/html/draft-ietf-oauth-v2-1-12#section-4.1.1
	if p.pkce.Enforce && codeChallenge == "" {
		return nil, "", newRedirectedErr(oauth2.InvalidRequest, "PKCE is required. The code_challenge parameter must be provided.")
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

			isTrusted, err := tokens.CrossClientTrusted(ctx, p.storage, clientID, peerID)
			if err != nil {
				return nil, "", newRedirectedErr(oauth2.ServerError, "Internal server error.")
			}
			if !isTrusted {
				invalidScopes = append(invalidScopes, scope)
			}
		}
	}
	if !hasOpenIDScope {
		return nil, "", newRedirectedErr(oauth2.InvalidScope, `Missing required scope(s) ["openid"].`)
	}
	if len(unrecognized) > 0 {
		return nil, "", newRedirectedErr(oauth2.InvalidScope, "Unrecognized scope(s) %q", unrecognized)
	}
	if len(invalidScopes) > 0 {
		return nil, "", newRedirectedErr(oauth2.InvalidScope, "Client can't request scope(s) %q", invalidScopes)
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
			return nil, "", newRedirectedErr(oauth2.InvalidRequest, "Invalid response type %q", responseType)
		}

		if !p.supportedResponseTypes[responseType] {
			return nil, "", newRedirectedErr(oauth2.UnsupportedResponseType, "Unsupported response type %q", responseType)
		}
	}

	if len(responseTypes) == 0 {
		return nil, "", newRedirectedErr(oauth2.InvalidRequest, "No response_type provided")
	}

	if rt.token && !rt.code && !rt.idToken {
		// "token" can't be provided on its own.
		// https://openid.net/specs/openid-connect-core-1_0.html#Authentication
		return nil, "", newRedirectedErr(oauth2.InvalidRequest, "Response type 'token' must be provided with type 'id_token' and/or 'code'")
	}
	if !rt.code {
		// Either "id_token token" or "id_token" implies the implicit flow, which
		// requires a nonce value.
		// https://openid.net/specs/openid-connect-core-1_0.html#ImplicitAuthRequest
		if nonce == "" {
			return nil, "", newRedirectedErr(oauth2.InvalidRequest, "Response type 'token' requires a 'nonce' value.")
		}
	}
	if rt.token {
		if redirectURI == oauth2.RedirectURIOOB {
			return nil, "", newRedirectedErr(oauth2.InvalidRequest, "Cannot use response type 'token' with redirect_uri '%s'.", oauth2.RedirectURIOOB)
		}
	}

	prompt, err := oauth2.ParsePrompt(q.Get("prompt"))
	if err != nil {
		return nil, "", newRedirectedErr(oauth2.InvalidRequest, "Invalid prompt parameter: %v", err)
	}

	// Parse max_age: -1 means not specified.
	maxAge := -1
	if maxAgeStr := q.Get("max_age"); maxAgeStr != "" {
		v, err := strconv.Atoi(maxAgeStr)
		if err != nil || v < 0 {
			return nil, "", newRedirectedErr(oauth2.InvalidRequest, "Invalid max_age value %q", maxAgeStr)
		}
		maxAge = v
	}

	// OIDC prompt=consent implies force approval.
	forceApproval := q.Get("approval_prompt") == "force" || prompt.Consent()

	// Validate id_token_hint if provided (OIDC Core 1.0 §3.1.2.1).
	var idTokenHintSubject string
	if hint := q.Get("id_token_hint"); hint != "" {
		idToken, err := p.ValidateIDTokenHint(ctx, hint)
		if err != nil {
			return nil, "", newRedirectedErr(oauth2.InvalidRequest, "Invalid id_token_hint.")
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
