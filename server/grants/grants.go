package grants

import (
	"crypto/subtle"
	"errors"
	"log/slog"
	"net/http"
	"net/url"
	"strings"

	"github.com/dexidp/dex/connector"
	"github.com/dexidp/dex/server/connectors"
	"github.com/dexidp/dex/server/oauth2"
	"github.com/dexidp/dex/server/tokens"
	"github.com/dexidp/dex/storage"
)

// Grant serves one OAuth2 grant type at the token endpoint. It is a set of hooks
// the Endpoint calls in order — the shared phases (client auth, scope validation,
// minting, response writing) live on the Endpoint, so a grant only fills in the
// parts unique to it and cannot forget a shared step.
type Grant interface {
	// GrantType is the grant_type value this grant serves.
	GrantType() string
	// RequiresClientAuth reports whether the endpoint must authenticate the
	// client before the request is processed.
	RequiresClientAuth() bool
	// Scopes reports how the endpoint validates the requested scopes for this
	// grant.
	Scopes() ScopePolicy
	// ConnectorID is the connector this grant authenticates against, or "" when
	// it uses none (client_credentials). The endpoint resolves it and enforces
	// the connector-authorization invariant before Authorize, so a grant cannot
	// forget the check.
	ConnectorID(r *http.Request) string
	// Authorize turns the validated request into the authorization to issue
	// tokens for, proving the resource owner's identity against conn (the zero
	// Connector when ConnectorID is ""). Returning an *oauth2.Error makes the
	// endpoint write that error.
	Authorize(r *http.Request, client storage.Client, scopes []string, conn connectors.Connector) (*Result, error)
}

// Minter is implemented by a grant whose response is not the standard token set,
// such as RFC 8693 token exchange. When a grant implements it, the endpoint calls
// Mint instead of the standard issuer.Issue mint.
type Minter interface {
	Mint(r *http.Request, res *Result) (tokens.Response, error)
}

// Result is what a grant's Authorize produces for the standard mint.
type Result struct {
	Authorization tokens.Authorization
	IssueRefresh  bool
}

// ScopePolicy configures the shared scope-validation phase for a grant. It is
// the single place scope rules are enforced, so no grant re-implements the
// cross-client trust check or scope filtering.
type ScopePolicy struct {
	// Standard is the set of standard (non cross-client) scopes the grant
	// accepts. When nil, scopes are passed through unvalidated (token exchange).
	Standard map[string]bool
	// RequireOpenID rejects the request when the openid scope is absent.
	RequireOpenID bool
	// Rejected maps an explicitly refused scope to its rejection message.
	Rejected map[string]string
	// ErrorType is the OAuth2 error code returned for scope violations.
	ErrorType string
}

// Endpoint is the /token endpoint. It owns the phases shared by every grant —
// dispatch by grant_type, client authentication, scope validation, minting and
// writing the response or error — while each grant carries only its own narrow
// dependencies.
type Endpoint struct {
	issuer     *tokens.Issuer
	storage    storage.Storage
	connectors *connectors.Cache
	logger     *slog.Logger

	grants map[string]Grant
}

// NewEndpoint wires the token endpoint and its grants from the shared
// dependencies. Each grant is constructed with only what it needs.
func NewEndpoint(issuer *tokens.Issuer, s storage.Storage, conns *connectors.Cache, logger *slog.Logger, passwordConnector string) *Endpoint {
	e := &Endpoint{issuer: issuer, storage: s, connectors: conns, logger: logger, grants: map[string]Grant{}}
	e.register(
		&clientCredentials{},
		&password{logger: logger, connectorID: passwordConnector},
		&tokenExchange{issuer: issuer, logger: logger},
	)
	return e
}

func (e *Endpoint) register(gs ...Grant) {
	for _, g := range gs {
		e.grants[g.GrantType()] = g
	}
}

// Dispatch runs the token-endpoint pipeline for the grant registered for
// grantType. It reports whether a grant handled the request, so the caller can
// fall back to grants that have not been migrated yet.
func (e *Endpoint) Dispatch(w http.ResponseWriter, r *http.Request, grantType string) bool {
	grant, ok := e.grants[grantType]
	if !ok {
		return false
	}

	// 1. Authenticate the client.
	client := storage.Client{}
	if grant.RequiresClientAuth() {
		client, ok = e.authenticateClient(w, r)
		if !ok {
			return true
		}
	}

	// 2. Parse and validate the requested scopes.
	scopes, oerr := e.validateScopes(r, client, grant.Scopes())
	if oerr != nil {
		e.writeError(w, r, oerr)
		return true
	}

	// 3. Resolve the grant's connector and enforce the connector-authorization
	// invariant. A grant that uses no connector resolves to the zero Connector.
	conn, oerr := e.resolveConnector(r, grant, client)
	if oerr != nil {
		e.writeError(w, r, oerr)
		return true
	}

	// 4. Let the grant prove the identity against the resolved connector.
	res, err := grant.Authorize(r, client, scopes, conn)
	if err != nil {
		e.writeError(w, r, err)
		return true
	}

	// 5. Mint the response — the standard token set, or the grant's own.
	resp, err := e.mint(r, grant, res)
	if err != nil {
		e.writeError(w, r, err)
		return true
	}

	// 6. Write the response.
	if err := resp.Write(w); err != nil {
		e.logger.ErrorContext(r.Context(), "failed to write token response", "err", err)
	}
	return true
}

// mint produces the token response: the grant's own when it is a Minter (RFC 8693
// token exchange), otherwise the single standard mint every grant shares.
func (e *Endpoint) mint(r *http.Request, grant Grant, res *Result) (tokens.Response, error) {
	if m, ok := grant.(Minter); ok {
		return m.Mint(r, res)
	}
	resp, err := e.issuer.IssueResponse(r.Context(), res.Authorization, res.IssueRefresh)
	if err != nil {
		e.logger.ErrorContext(r.Context(), "failed to issue tokens", "grant_type", grant.GrantType(), "err", err)
		return tokens.Response{}, &oauth2.Error{Type: oauth2.ServerError, Status: http.StatusInternalServerError}
	}
	return resp, nil
}

// authenticateClient resolves the client from HTTP Basic or form credentials. On
// failure it writes the error response and returns ok=false.
func (e *Endpoint) authenticateClient(w http.ResponseWriter, r *http.Request) (storage.Client, bool) {
	ctx := r.Context()
	clientID, clientSecret, ok := r.BasicAuth()
	if ok {
		var err error
		if clientID, err = url.QueryUnescape(clientID); err != nil {
			e.writeError(w, r, &oauth2.Error{Type: oauth2.InvalidRequest, Description: "client_id improperly encoded", Status: http.StatusBadRequest})
			return storage.Client{}, false
		}
		if clientSecret, err = url.QueryUnescape(clientSecret); err != nil {
			e.writeError(w, r, &oauth2.Error{Type: oauth2.InvalidRequest, Description: "client_secret improperly encoded", Status: http.StatusBadRequest})
			return storage.Client{}, false
		}
	} else {
		clientID = r.PostFormValue("client_id")
		clientSecret = r.PostFormValue("client_secret")
	}

	client, err := e.storage.GetClient(ctx, clientID)
	if err != nil {
		if err != storage.ErrNotFound {
			e.logger.ErrorContext(ctx, "failed to get client", "err", err)
			e.writeError(w, r, &oauth2.Error{Type: oauth2.ServerError, Status: http.StatusInternalServerError})
		} else {
			e.writeError(w, r, &oauth2.Error{Type: oauth2.InvalidClient, Description: "Invalid client credentials.", Status: http.StatusUnauthorized})
		}
		return storage.Client{}, false
	}

	if subtle.ConstantTimeCompare([]byte(client.Secret), []byte(clientSecret)) != 1 {
		if clientSecret == "" {
			e.logger.InfoContext(ctx, "missing client_secret on token request", "client_id", client.ID)
		} else {
			e.logger.InfoContext(ctx, "invalid client_secret on token request", "client_id", client.ID)
		}
		e.writeError(w, r, &oauth2.Error{Type: oauth2.InvalidClient, Description: "Invalid client credentials.", Status: http.StatusUnauthorized})
		return storage.Client{}, false
	}

	return client, true
}

// writeError writes err as an OAuth2 error response. An *oauth2.Error carries its
// own type/description/status; anything else is reported as a server error.
func (e *Endpoint) writeError(w http.ResponseWriter, r *http.Request, err error) {
	oerr := &oauth2.Error{Type: oauth2.ServerError, Status: http.StatusInternalServerError}
	if !errors.As(err, &oerr) {
		e.logger.ErrorContext(r.Context(), "token request failed", "err", err)
	}
	if werr := oauth2.WriteError(w, oerr.Type, oerr.Description, oerr.Status); werr != nil {
		e.logger.ErrorContext(r.Context(), "failed to write token error response", "err", werr)
	}
}

// resolveConnector enforces the connector-authorization invariant for the grant
// and returns the opened connector: the client must allow the connector, and the
// connector must permit the grant type. A grant that uses no connector
// (ConnectorID == "") resolves to the zero Connector. Running here, before
// Authorize, means no grant can forget the check.
func (e *Endpoint) resolveConnector(r *http.Request, grant Grant, client storage.Client) (connectors.Connector, *oauth2.Error) {
	connID := grant.ConnectorID(r)
	if connID == "" {
		return connectors.Connector{}, nil
	}

	ctx := r.Context()
	if !connectors.ConnectorAllowed(client.AllowedConnectors, connID) {
		e.logger.WarnContext(ctx, "connector not allowed for client", "client_id", client.ID, "connector_id", connID)
		return connectors.Connector{}, &oauth2.Error{Type: oauth2.InvalidGrant, Description: "Connector not allowed for this client.", Status: http.StatusBadRequest}
	}
	conn, err := e.connectors.Get(ctx, connID)
	if err != nil {
		e.logger.ErrorContext(ctx, "failed to get connector", "connector_id", connID, "err", err)
		return connectors.Connector{}, &oauth2.Error{Type: oauth2.InvalidRequest, Description: "Requested connector does not exist.", Status: http.StatusBadRequest}
	}
	if !connectors.GrantTypeAllowed(conn.GrantTypes, grant.GrantType()) {
		e.logger.ErrorContext(ctx, "connector does not allow grant", "connector_id", connID, "grant_type", grant.GrantType())
		return connectors.Connector{}, &oauth2.Error{Type: oauth2.InvalidRequest, Description: "Requested connector does not support this grant type.", Status: http.StatusBadRequest}
	}
	return conn, nil
}

// shouldIssueRefreshToken reports whether a refresh token should be issued: the
// connector supports refresh, the connector permits the refresh_token grant, and
// offline_access was requested. A refresh token is never mandatory (RFC 6749 §1.5).
func shouldIssueRefreshToken(conn connectors.Connector, scopes []string) bool {
	if _, ok := conn.Connector.(connector.RefreshConnector); !ok {
		return false
	}
	if !connectors.GrantTypeAllowed(conn.GrantTypes, oauth2.GrantTypeRefreshToken) {
		return false
	}
	for _, scope := range scopes {
		if scope == tokens.ScopeOfflineAccess {
			return true
		}
	}
	return false
}

// validateScopes parses and validates the requested scopes per the grant's
// policy: it rejects refused scopes, filters unknown ones, enforces openid when
// required, and verifies cross-client trust — the security-sensitive check that
// must run for every grant. A nil policy set passes scopes through unvalidated.
func (e *Endpoint) validateScopes(r *http.Request, client storage.Client, p ScopePolicy) ([]string, *oauth2.Error) {
	scopes := strings.Fields(r.Form.Get("scope"))
	if p.Standard == nil {
		return scopes, nil
	}

	ctx := r.Context()
	var unrecognized, invalid []string
	for _, scope := range scopes {
		if msg, refused := p.Rejected[scope]; refused {
			return nil, &oauth2.Error{Type: p.ErrorType, Description: msg, Status: http.StatusBadRequest}
		}
		if p.Standard[scope] {
			continue
		}

		peerID, ok := tokens.ParseCrossClientScope(scope)
		if !ok {
			unrecognized = append(unrecognized, scope)
			continue
		}
		trusted, err := tokens.CrossClientTrusted(ctx, e.storage, client.ID, peerID)
		if err != nil {
			e.logger.ErrorContext(ctx, "error validating cross client trust", "client_id", client.ID, "peer_id", peerID, "err", err)
			return nil, &oauth2.Error{Type: oauth2.InvalidClient, Description: "Error validating cross client trust.", Status: http.StatusBadRequest}
		}
		if !trusted {
			invalid = append(invalid, scope)
		}
	}

	if p.RequireOpenID && !tokens.HasOpenID(scopes) {
		return nil, &oauth2.Error{Type: p.ErrorType, Description: `Missing required scope(s) ["openid"].`, Status: http.StatusBadRequest}
	}
	if len(unrecognized) > 0 {
		return nil, oauth2.Errorf(p.ErrorType, http.StatusBadRequest, "Unrecognized scope(s) %q", unrecognized)
	}
	if len(invalid) > 0 {
		return nil, oauth2.Errorf(p.ErrorType, http.StatusBadRequest, "Client can't request scope(s) %q", invalid)
	}
	return scopes, nil
}
