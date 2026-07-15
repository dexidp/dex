package server

// userinfo.go serves the OIDC /userinfo endpoint.

import (
	"encoding/json"
	"net/http"
	"strings"

	"github.com/coreos/go-oidc/v3/oidc"
)

func (s *Server) handleUserInfo(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	const prefix = "Bearer "

	auth := r.Header.Get("authorization")
	if len(auth) < len(prefix) || !strings.EqualFold(prefix, auth[:len(prefix)]) {
		w.Header().Set("WWW-Authenticate", "Bearer")
		s.tokenErrHelper(w, errAccessDenied, "Invalid bearer token.", http.StatusUnauthorized)
		return
	}
	rawIDToken := auth[len(prefix):]

	verifier := oidc.NewVerifier(s.issuerURL.String(), &signerKeySet{s.signer}, &oidc.Config{
		SupportedSigningAlgs: supportedSigningAlgStrings(),
		SkipClientIDCheck:    true,
	})
	idToken, err := verifier.Verify(ctx, rawIDToken)
	if err != nil {
		s.logger.ErrorContext(r.Context(), "failed to verify ID token", "err", err)
		s.tokenErrHelper(w, errAccessDenied, "Invalid bearer token.", http.StatusForbidden)
		return
	}

	var claims json.RawMessage
	if err := idToken.Claims(&claims); err != nil {
		s.logger.ErrorContext(r.Context(), "failed to decode ID token claims", "err", err)
		s.tokenErrHelper(w, errServerError, "", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.Write(claims)
}
