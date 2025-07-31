package server

import (
	"net/http"

	"github.com/dexidp/dex/pkg/otel/traces"
)

func (s *Server) handleToken(w http.ResponseWriter, r *http.Request) {
	ctx, span := traces.InstrumentHandler(r)
	defer span.End()
	w.Header().Set("Content-Type", "application/json")
	if r.Method != http.MethodPost {
		s.tokenErrHelper(ctx, w, errInvalidRequest, "method not allowed", http.StatusBadRequest)
		return
	}

	err := r.ParseForm()
	if err != nil {
		s.logger.ErrorContext(ctx, "could not parse request body", "err", err)
		s.tokenErrHelper(ctx, w, errInvalidRequest, "", http.StatusBadRequest)
		return
	}

	grantType := r.PostFormValue("grant_type")
	if !contains(s.supportedGrantTypes, grantType) {
		s.logger.ErrorContext(ctx, "unsupported grant type", "grant_type", grantType)
		s.tokenErrHelper(ctx, w, errUnsupportedGrantType, "", http.StatusBadRequest)
		return
	}
	switch grantType {
	case grantTypeDeviceCode:
		s.handleDeviceToken(w, r)
	case grantTypeAuthorizationCode:
		s.withClientFromStorage(w, r, s.handleAuthCode)
	case grantTypeRefreshToken:
		s.withClientFromStorage(w, r, s.handleRefreshToken)
	case grantTypePassword:
		s.withClientFromStorage(w, r, s.handlePasswordGrant)
	case grantTypeTokenExchange:
		s.withClientFromStorage(w, r, s.handleTokenExchange)
	default:
		s.tokenErrHelper(ctx, w, errUnsupportedGrantType, "", http.StatusBadRequest)
	}
	s.logger.InfoContext(ctx, "handled grant type", "grant_type", grantType)
}
