package userinfo

import (
	"encoding/json"
	"log/slog"
	"net/http"
	"strings"

	"github.com/coreos/go-oidc/v3/oidc"

	"github.com/dexidp/dex/server/oauth2"
	"github.com/dexidp/dex/server/router"
	"github.com/dexidp/dex/server/signer"
)

// Handler serves the OIDC /userinfo endpoint.
type Handler struct {
	Issuer string
	Signer signer.Signer
	Logger *slog.Logger
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
		h.writeError(w, oauth2.AccessDenied, "Invalid bearer token.", http.StatusUnauthorized)
		return
	}
	rawIDToken := auth[len(prefix):]

	verifier := oidc.NewVerifier(h.Issuer, &signer.KeySet{Signer: h.Signer}, &oidc.Config{SkipClientIDCheck: true})
	idToken, err := verifier.Verify(ctx, rawIDToken)
	if err != nil {
		h.Logger.ErrorContext(ctx, "failed to verify ID token", "err", err)
		h.writeError(w, oauth2.AccessDenied, "Invalid bearer token.", http.StatusForbidden)
		return
	}

	var claims json.RawMessage
	if err := idToken.Claims(&claims); err != nil {
		h.Logger.ErrorContext(ctx, "failed to decode ID token claims", "err", err)
		h.writeError(w, oauth2.ServerError, "", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.Write(claims)
}

func (h *Handler) writeError(w http.ResponseWriter, typ, description string, statusCode int) {
	if err := oauth2.WriteError(w, typ, description, statusCode); err != nil {
		h.Logger.Error("userinfo error response", "err", err)
	}
}
