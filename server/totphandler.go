package server

import (
	"bytes"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/base64"
	"fmt"
	"image/png"
	"net/http"
	"strings"

	"github.com/pquerna/otp"
	"github.com/pquerna/otp/totp"

	"github.com/dexidp/dex/storage"
)

func (s *Server) handleTOTPVerify(w http.ResponseWriter, r *http.Request) {
	macEncoded := r.FormValue("hmac")
	if macEncoded == "" {
		s.renderError(r, w, http.StatusUnauthorized, "Unauthorized request.")
		return
	}
	mac, err := base64.RawURLEncoding.DecodeString(macEncoded)
	if err != nil {
		s.renderError(r, w, http.StatusUnauthorized, "Unauthorized request.")
		return
	}

	authReq, err := s.storage.GetAuthRequest(r.Context(), r.FormValue("req"))
	if err != nil {
		s.logger.ErrorContext(r.Context(), "failed to get auth request", "err", err)
		s.renderError(r, w, http.StatusInternalServerError, "Database error.")
		return
	}
	if !authReq.LoggedIn {
		s.logger.ErrorContext(r.Context(), "auth request does not have an identity for TOTP verification")
		s.renderError(r, w, http.StatusInternalServerError, "Login process not yet finalized.")
		return
	}

	// build expected hmac with secret key
	h := hmac.New(sha256.New, authReq.HMACKey)
	h.Write([]byte(authReq.ID))
	expectedMAC := h.Sum(nil)
	// constant time comparison
	if !hmac.Equal(mac, expectedMAC) {
		s.renderError(r, w, http.StatusUnauthorized, "Unauthorized request.")
		return
	}

	offlineSession, err := s.storage.GetOfflineSessions(r.Context(), authReq.Claims.UserID, authReq.ConnectorID)
	if err != nil {
		s.logger.ErrorContext(r.Context(), "failed to get offline session", "err", err, "connector_id", authReq.ConnectorID, "user_id", authReq.Claims.UserID)
		s.renderError(r, w, http.StatusInternalServerError, "Database error.")
		return
	}

	// TODO(nabokihms): compose the redirect URL the right way
	returnURL := strings.ReplaceAll(r.URL.String(), "/totp", "/approval")
	if offlineSession.TOTP == "" || authReq.TOTPValidated {
		http.Redirect(w, r, returnURL, http.StatusSeeOther)
		return
	}

	switch r.Method {
	case http.MethodGet:
		s.renderTOTPValidatePage(offlineSession, false, w, r)
		return
	case http.MethodPost:
		password := r.FormValue("totp")

		generated, err := otp.NewKeyFromURL(offlineSession.TOTP)
		if err != nil {
			s.logger.ErrorContext(r.Context(), "failed to load TOTP QR code", "err", err, "connector_id", offlineSession.ConnID, "user_id", offlineSession.ConnID)
			s.renderError(r, w, http.StatusInternalServerError, "Internal server error.")
			return
		}

		ok := totp.Validate(password, generated.Secret())
		if !ok {
			s.renderTOTPValidatePage(offlineSession, true, w, r)
			s.logger.ErrorContext(r.Context(), "failed TOTP attempt: Invalid credentials.", "user", "????")
			return
		}

		// If the TOTP is valid, update the offline session and auth request to reflect that.
		if err := s.storage.UpdateOfflineSessions(r.Context(), offlineSession.UserID, offlineSession.ConnID, func(old storage.OfflineSessions) (storage.OfflineSessions, error) {
			old.TOTPConfirmed = true
			return old, nil
		}); err != nil {
			s.logger.ErrorContext(r.Context(), "failed to update offline session", "err", err, "connector_id", offlineSession.ConnID, "user_id", offlineSession.ConnID)
			s.renderError(r, w, http.StatusInternalServerError, "Internal server error.")
			return
		}
		if err := s.storage.UpdateAuthRequest(r.Context(), authReq.ID, func(old storage.AuthRequest) (storage.AuthRequest, error) {
			old.TOTPValidated = true
			return old, nil
		}); err != nil {
			s.logger.ErrorContext(r.Context(), "failed to update auth request", "err", err, "auth_request_id", authReq.ID)
			s.renderError(r, w, http.StatusInternalServerError, "Internal server error.")
			return
		}

		// we can skip the redirect to /approval and go ahead and send code if it's not required
		if s.skipApproval && !authReq.ForceApprovalPrompt {
			authReq, err = s.storage.GetAuthRequest(r.Context(), authReq.ID)
			if err != nil {
				s.logger.ErrorContext(r.Context(), "failed to get finalized auth request", "err", err)
				s.renderError(r, w, http.StatusInternalServerError, "Login error.")
				return
			}
			s.sendCodeResponse(w, r, authReq)
			return
		}
		http.Redirect(w, r, returnURL, http.StatusSeeOther)
	default:
		s.renderError(r, w, http.StatusBadRequest, "Unsupported request method.")
	}
}

// generateQRCode generates a QR code image for the OTP key.
// Returned value is a base64 encoded PNG image.
func generateQRCode(o storage.OfflineSessions) (string, error) {
	generated, err := otp.NewKeyFromURL(o.TOTP)
	if err != nil {
		return "", fmt.Errorf("failed to load TOTP QR code: %w", err)
	}

	qrCodeImage, err := generated.Image(300, 300)
	if err != nil {
		return "", fmt.Errorf("failed to generate TOTP QR code: %w", err)
	}

	var buf bytes.Buffer
	err = png.Encode(&buf, qrCodeImage)
	if err != nil {
		return "", fmt.Errorf("failed to encode TOTP QR code: %w", err)
	}

	return base64.StdEncoding.EncodeToString(buf.Bytes()), nil
}

func (s *Server) renderTOTPValidatePage(o storage.OfflineSessions, lastFail bool, w http.ResponseWriter, r *http.Request) {
	qrCode := ""
	var err error

	// Show QR code only once when the offline session is registered
	if o.TOTP != "" && !o.TOTPConfirmed {
		qrCode, err = generateQRCode(o)
		if err != nil {
			s.logger.ErrorContext(r.Context(), "failed to generate QR code", "err", err, "connector_id", o.ConnID, "user_id", o.ConnID)
			s.renderError(r, w, http.StatusInternalServerError, "Internal server error.")
			return
		}
	}
	if err := s.templates.totpVerify(r, w, r.URL.String(), s.totp.issuer, o.ConnID, qrCode, lastFail); err != nil {
		s.logger.ErrorContext(r.Context(), "server template error", "err", err)
	}
}

type secondFactorAuthenticator struct {
	issuer string
	// To check that TOTP is enabled for the connector.
	connectors map[string]struct{}
}

func newSecondFactorAuthenticator(issuer string, connectors []string) *secondFactorAuthenticator {
	c := make(map[string]struct{})
	for _, conn := range connectors {
		c[conn] = struct{}{}
	}
	return &secondFactorAuthenticator{issuer: issuer, connectors: c}
}

func (s *secondFactorAuthenticator) generate(connID, email string) (*otp.Key, error) {
	return totp.Generate(totp.GenerateOpts{
		Issuer:      s.issuer,
		AccountName: fmt.Sprintf("(%s) %s", connID, email),
	})
}

func (s *secondFactorAuthenticator) enabledForConnector(connID string) bool {
	_, ok := s.connectors[connID]
	return ok
}
