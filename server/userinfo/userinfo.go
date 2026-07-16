// Package userinfo serves the OIDC /userinfo endpoint.
package userinfo

import (
	"encoding/json"
	"log/slog"
	"net/http"
	"strings"

	"github.com/coreos/go-oidc/v3/oidc"

	"github.com/dexidp/dex/server/router"
	"github.com/dexidp/dex/server/signer"
)

// OAuth2 error codes emitted by the endpoint.
const (
	errAccessDenied = "access_denied"
	errServerError  = "server_error"
)

// Config holds the userinfo handler's dependencies. WriteError writes an OAuth2
// token error response; it is supplied by the server so the handler does not
// depend on the whole Server.
type Config struct {
	Issuer     string
	Signer     signer.Signer
	Logger     *slog.Logger
	WriteError func(w http.ResponseWriter, typ, desc string, code int)
}

// Handler serves the /userinfo endpoint.
type Handler struct {
	Config
}

// New returns a userinfo handler.
func New(c Config) *Handler {
	return &Handler{Config: c}
}

// Mount registers the userinfo route.
func (h *Handler) Mount(m router.Mux) {
	m.HandleCORS("/userinfo", h.handle)
}

func (h *Handler) handle(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	const prefix = "Bearer "

	auth := r.Header.Get("authorization")
	if len(auth) < len(prefix) || !strings.EqualFold(prefix, auth[:len(prefix)]) {
		w.Header().Set("WWW-Authenticate", "Bearer")
		h.WriteError(w, errAccessDenied, "Invalid bearer token.", http.StatusUnauthorized)
		return
	}
	rawIDToken := auth[len(prefix):]

	verifier := oidc.NewVerifier(h.Issuer, &signer.KeySet{Signer: h.Signer}, &oidc.Config{SkipClientIDCheck: true})
	idToken, err := verifier.Verify(ctx, rawIDToken)
	if err != nil {
		h.Logger.ErrorContext(ctx, "failed to verify ID token", "err", err)
		h.WriteError(w, errAccessDenied, "Invalid bearer token.", http.StatusForbidden)
		return
	}

	var claims json.RawMessage
	if err := idToken.Claims(&claims); err != nil {
		h.Logger.ErrorContext(ctx, "failed to decode ID token claims", "err", err)
		h.WriteError(w, errServerError, "", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.Write(claims)
}
