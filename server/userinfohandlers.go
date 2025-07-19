package server

import (
	"encoding/json"
	"net/http"
	"strings"

	"github.com/coreos/go-oidc/v3/oidc"

	"github.com/dexidp/dex/pkg/otel/traces"
)

func (s *Server) handleUserInfo(w http.ResponseWriter, r *http.Request) {
	ctx, span := traces.InstrumentHandler(r)
	defer span.End()
	const prefix = "Bearer "

	auth := r.Header.Get("authorization")
	if len(auth) < len(prefix) || !strings.EqualFold(prefix, auth[:len(prefix)]) {
		w.Header().Set("WWW-Authenticate", "Bearer")
		s.tokenErrHelper(ctx, w, errAccessDenied, "Invalid bearer token.", http.StatusUnauthorized)
		return
	}
	rawIDToken := auth[len(prefix):]

	verifier := oidc.NewVerifier(s.issuerURL.String(), &storageKeySet{s.storage}, &oidc.Config{SkipClientIDCheck: true})
	idToken, err := verifier.Verify(ctx, rawIDToken)
	if err != nil {
		s.tokenErrHelper(ctx, w, errAccessDenied, err.Error(), http.StatusForbidden)
		return
	}

	var claims json.RawMessage
	if err := idToken.Claims(&claims); err != nil {
		s.tokenErrHelper(ctx, w, errServerError, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.Write(claims)
}
