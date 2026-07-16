package grants

import (
	"crypto/subtle"
	"log/slog"
	"net/http"
	"net/url"
	"time"

	"github.com/dexidp/dex/server/oauth2"
	"github.com/dexidp/dex/server/tokens"
	"github.com/dexidp/dex/storage"
)

// Grant serves one OAuth2 grant type at the token endpoint. Each grant is a
// self-contained handler registered with the Endpoint, which dispatches to it by
// grant_type — the token endpoint's equivalent of the router.Handler abstraction.
type Grant interface {
	// GrantType is the grant_type value this grant serves.
	GrantType() string
	// RequiresClientAuth reports whether the endpoint must authenticate the
	// client before Handle. It is false for grants that authenticate differently
	// (e.g. the device_code grant, which is bound to the device code).
	RequiresClientAuth() bool
	// Handle serves a token request. client is the authenticated client when
	// RequiresClientAuth is true, and the zero Client otherwise.
	Handle(w http.ResponseWriter, r *http.Request, client storage.Client)
}

// Endpoint dispatches the /token endpoint to its registered grants.
type Endpoint struct {
	Storage storage.Storage
	Logger  *slog.Logger
	Now     func() time.Time

	grants map[string]Grant
}

// Register adds grants to the dispatch table.
func (e *Endpoint) Register(gs ...Grant) {
	if e.grants == nil {
		e.grants = make(map[string]Grant, len(gs))
	}
	for _, g := range gs {
		e.grants[g.GrantType()] = g
	}
}

// Dispatch serves the grant registered for grantType, authenticating the client
// first when the grant requires it. It reports whether a grant handled the
// request, so the caller can fall back to grants that have not been migrated yet.
func (e *Endpoint) Dispatch(w http.ResponseWriter, r *http.Request, grantType string) bool {
	grant, ok := e.grants[grantType]
	if !ok {
		return false
	}
	if grant.RequiresClientAuth() {
		e.withClientFromStorage(w, r, grant.Handle)
	} else {
		grant.Handle(w, r, storage.Client{})
	}
	return true
}

// withClientFromStorage authenticates the client from HTTP Basic or form
// credentials and calls handler with the resolved client.
func (e *Endpoint) withClientFromStorage(w http.ResponseWriter, r *http.Request, handler func(http.ResponseWriter, *http.Request, storage.Client)) {
	ctx := r.Context()
	clientID, clientSecret, ok := r.BasicAuth()
	if ok {
		var err error
		if clientID, err = url.QueryUnescape(clientID); err != nil {
			e.writeError(w, oauth2.InvalidRequest, "client_id improperly encoded", http.StatusBadRequest)
			return
		}
		if clientSecret, err = url.QueryUnescape(clientSecret); err != nil {
			e.writeError(w, oauth2.InvalidRequest, "client_secret improperly encoded", http.StatusBadRequest)
			return
		}
	} else {
		clientID = r.PostFormValue("client_id")
		clientSecret = r.PostFormValue("client_secret")
	}

	client, err := e.Storage.GetClient(ctx, clientID)
	if err != nil {
		if err != storage.ErrNotFound {
			e.Logger.ErrorContext(ctx, "failed to get client", "err", err)
			e.writeError(w, oauth2.ServerError, "", http.StatusInternalServerError)
		} else {
			e.writeError(w, oauth2.InvalidClient, "Invalid client credentials.", http.StatusUnauthorized)
		}
		return
	}

	if subtle.ConstantTimeCompare([]byte(client.Secret), []byte(clientSecret)) != 1 {
		if clientSecret == "" {
			e.Logger.InfoContext(ctx, "missing client_secret on token request", "client_id", client.ID)
		} else {
			e.Logger.InfoContext(ctx, "invalid client_secret on token request", "client_id", client.ID)
		}
		e.writeError(w, oauth2.InvalidClient, "Invalid client credentials.", http.StatusUnauthorized)
		return
	}

	handler(w, r, client)
}

func (e *Endpoint) writeError(w http.ResponseWriter, typ, description string, statusCode int) {
	writeError(e.Logger, w, typ, description, statusCode)
}

// writeError writes a JSON OAuth2 error response.
func writeError(logger *slog.Logger, w http.ResponseWriter, typ, description string, statusCode int) {
	if err := oauth2.WriteError(w, typ, description, statusCode); err != nil {
		logger.Error("token error response", "err", err)
	}
}

// writeTokenResponse writes a token set as an OAuth2 token response.
func writeTokenResponse(w http.ResponseWriter, ts tokens.TokenSet, now time.Time) error {
	return ts.Response(now).Write(w)
}
