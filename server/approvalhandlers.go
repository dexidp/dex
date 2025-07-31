package server

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/base64"
	"net/http"

	"github.com/dexidp/dex/pkg/otel/traces"
)

func (s *Server) handleApproval(w http.ResponseWriter, r *http.Request) {
	ctx, span := traces.InstrumentHandler(r)
	defer span.End()
	macEncoded := r.FormValue("hmac")
	if macEncoded == "" {
		s.renderError(r, w, http.StatusUnauthorized, "Unauthorized request")
		return
	}
	mac, err := base64.RawURLEncoding.DecodeString(macEncoded)
	if err != nil {
		s.renderError(r, w, http.StatusUnauthorized, "Unauthorized request")
		return
	}

	authReq, err := s.storage.GetAuthRequest(ctx, r.FormValue("req"))
	if err != nil {
		s.logger.ErrorContext(ctx, "failed to get auth request", "err", err)
		s.renderError(r, w, http.StatusInternalServerError, "Database error.")
		return
	}
	if !authReq.LoggedIn {
		s.logger.ErrorContext(ctx, "auth request does not have an identity for approval")
		s.renderError(r, w, http.StatusInternalServerError, "Login process not yet finalized.")
		return
	}

	// build expected hmac with secret key
	h := hmac.New(sha256.New, authReq.HMACKey)
	h.Write([]byte(authReq.ID))
	expectedMAC := h.Sum(nil)
	// constant time comparison
	if !hmac.Equal(mac, expectedMAC) {
		s.renderError(r, w, http.StatusUnauthorized, "Unauthorized request")
		return
	}

	switch r.Method {
	case http.MethodGet:
		client, err := s.storage.GetClient(ctx, authReq.ClientID)
		if err != nil {
			s.logger.ErrorContext(ctx, "Failed to get client", "client_id", authReq.ClientID, "err", err)
			s.renderError(r, w, http.StatusInternalServerError, "Failed to retrieve client.")
			return
		}
		if err := s.templates.approval(r, w, authReq.ID, authReq.Claims.Username, client.Name, authReq.Scopes); err != nil {
			s.logger.ErrorContext(ctx, "server template error", "err", err)
		}
	case http.MethodPost:
		if r.FormValue("approval") != "approve" {
			s.renderError(r, w, http.StatusInternalServerError, "Approval rejected.")
			return
		}
		s.sendCodeResponse(w, r, authReq)
	}
}
