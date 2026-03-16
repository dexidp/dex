package server

import (
	"bytes"
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/base64"
	"fmt"
	"image/png"
	"net/http"
	"net/url"
	"path"

	"github.com/pquerna/otp"
	"github.com/pquerna/otp/totp"

	"github.com/dexidp/dex/storage"
)

// MFAProvider is a pluggable multi-factor authentication method.
type MFAProvider interface {
	// Type returns the authenticator type identifier (e.g., "TOTP").
	Type() string
	// EnabledForConnectorType returns true if this provider applies to the given connector type.
	// If no connector types are configured, the provider applies to all.
	EnabledForConnectorType(connectorType string) bool
}

// TOTPProvider implements TOTP-based multi-factor authentication.
type TOTPProvider struct {
	issuer         string
	connectorTypes map[string]struct{}
}

// NewTOTPProvider creates a new TOTP MFA provider.
func NewTOTPProvider(issuer string, connectorTypes []string) *TOTPProvider {
	m := make(map[string]struct{}, len(connectorTypes))
	for _, t := range connectorTypes {
		m[t] = struct{}{}
	}
	return &TOTPProvider{issuer: issuer, connectorTypes: m}
}

func (p *TOTPProvider) EnabledForConnectorType(connectorType string) bool {
	if len(p.connectorTypes) == 0 {
		return true
	}
	_, ok := p.connectorTypes[connectorType]
	return ok
}

func (p *TOTPProvider) Type() string { return "TOTP" }

func (p *TOTPProvider) generate(connID, email string) (*otp.Key, error) {
	return totp.Generate(totp.GenerateOpts{
		Issuer:      p.issuer,
		AccountName: fmt.Sprintf("(%s) %s", connID, email),
	})
}

func (s *Server) handleMFAVerify(w http.ResponseWriter, r *http.Request) {
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

	ctx := r.Context()

	authReq, err := s.storage.GetAuthRequest(ctx, r.FormValue("req"))
	if err != nil {
		s.logger.ErrorContext(ctx, "failed to get auth request", "err", err)
		s.renderError(r, w, http.StatusInternalServerError, "Database error.")
		return
	}
	if !authReq.LoggedIn {
		s.logger.ErrorContext(ctx, "auth request does not have an identity for MFA verification")
		s.renderError(r, w, http.StatusInternalServerError, "Login process not yet finalized.")
		return
	}

	authenticatorID := r.FormValue("authenticator")

	// Verify HMAC — includes authenticatorID to prevent skipping steps in the MFA chain.
	h := hmac.New(sha256.New, authReq.HMACKey)
	h.Write([]byte(authReq.ID + "|" + authenticatorID))
	if !hmac.Equal(mac, h.Sum(nil)) {
		s.renderError(r, w, http.StatusUnauthorized, "Unauthorized request.")
		return
	}
	provider, ok := s.mfaProviders[authenticatorID]
	if !ok {
		s.renderError(r, w, http.StatusBadRequest, "Unknown authenticator.")
		return
	}

	totpProvider, ok := provider.(*TOTPProvider)
	if !ok {
		s.renderError(r, w, http.StatusInternalServerError, "Unsupported authenticator type.")
		return
	}

	identity, err := s.storage.GetUserIdentity(ctx, authReq.Claims.UserID, authReq.ConnectorID)
	if err != nil {
		s.logger.ErrorContext(ctx, "failed to get user identity", "err", err)
		s.renderError(r, w, http.StatusInternalServerError, "Database error.")
		return
	}

	// Build approval URL with an HMAC that covers only the request ID
	// (MFA HMAC includes authenticatorID and is not valid for approval).
	approvalH := hmac.New(sha256.New, authReq.HMACKey)
	approvalH.Write([]byte(authReq.ID))
	returnURL := path.Join(s.issuerURL.Path, "/approval") + "?req=" + authReq.ID +
		"&hmac=" + base64.RawURLEncoding.EncodeToString(approvalH.Sum(nil))

	if authReq.MFAValidated {
		http.Redirect(w, r, returnURL, http.StatusSeeOther)
		return
	}

	secret := identity.MFASecrets[authenticatorID]

	switch r.Method {
	case http.MethodGet:
		if secret == nil {
			// First-time enrollment: generate a new TOTP key.
			// TODO(nabokihms): clean up stale unconfirmed secrets. If a user starts
			// enrollment multiple times without completing it, old secrets accumulate.
			generated, err := totpProvider.generate(authReq.ConnectorID, authReq.Claims.Email)
			if err != nil {
				s.logger.ErrorContext(ctx, "failed to generate TOTP key", "err", err)
				s.renderError(r, w, http.StatusInternalServerError, "Internal server error.")
				return
			}

			secret = &storage.MFASecret{
				AuthenticatorID: authenticatorID,
				Type:            "TOTP",
				Secret:          generated.String(),
				Confirmed:       false,
				CreatedAt:       s.now(),
			}

			if err := s.storage.UpdateUserIdentity(ctx, authReq.Claims.UserID, authReq.ConnectorID, func(old storage.UserIdentity) (storage.UserIdentity, error) {
				if old.MFASecrets == nil {
					old.MFASecrets = make(map[string]*storage.MFASecret)
				}
				old.MFASecrets[authenticatorID] = secret
				return old, nil
			}); err != nil {
				s.logger.ErrorContext(ctx, "failed to store MFA secret", "err", err)
				s.renderError(r, w, http.StatusInternalServerError, "Internal server error.")
				return
			}
		}

		s.renderTOTPPage(secret, false, totpProvider.issuer, authReq.ConnectorID, w, r)

	case http.MethodPost:
		// TODO(nabokihms): this endpoint should be proteted with a rate limit (like the auth endpoint).
		// TOTP has a limited keyspace (6 digits) with a 30-second validity window,
		// making it particularly vulnerable to brute-force without rate limiting.
		// This endpoint should be protected similarly to the auth/login endpoints.
		//
		// For now the best way is to use external rate limitting solutions.
		if secret == nil || secret.Secret == "" {
			s.renderError(r, w, http.StatusBadRequest, "MFA not enrolled.")
			return
		}

		code := r.FormValue("totp")
		generated, err := otp.NewKeyFromURL(secret.Secret)
		if err != nil {
			s.logger.ErrorContext(ctx, "failed to load TOTP key", "err", err)
			s.renderError(r, w, http.StatusInternalServerError, "Internal server error.")
			return
		}

		if !totp.Validate(code, generated.Secret()) {
			s.renderTOTPPage(secret, true, totpProvider.issuer, authReq.ConnectorID, w, r)
			return
		}

		// Mark MFA secret as confirmed.
		if !secret.Confirmed {
			if err := s.storage.UpdateUserIdentity(ctx, authReq.Claims.UserID, authReq.ConnectorID, func(old storage.UserIdentity) (storage.UserIdentity, error) {
				if s := old.MFASecrets[authenticatorID]; s != nil {
					s.Confirmed = true
				}
				return old, nil
			}); err != nil {
				s.logger.ErrorContext(ctx, "failed to confirm MFA secret", "err", err)
				s.renderError(r, w, http.StatusInternalServerError, "Internal server error.")
				return
			}
		}

		// Check if there are more authenticators in the MFA chain.
		mfaChain, err := s.mfaChainForClient(ctx, authReq.ClientID, authReq.ConnectorID)
		if err != nil {
			s.logger.ErrorContext(ctx, "failed to get MFA chain", "err", err)
			s.renderError(r, w, http.StatusInternalServerError, "Internal server error.")
			return
		}

		// Find the next authenticator in the chain after the current one.
		var nextAuthenticator string
		for i, id := range mfaChain {
			if id == authenticatorID && i+1 < len(mfaChain) {
				nextAuthenticator = mfaChain[i+1]
				break
			}
		}

		if nextAuthenticator != "" {
			// Redirect to the next authenticator in the chain.
			h := hmac.New(sha256.New, authReq.HMACKey)
			h.Write([]byte(authReq.ID + "|" + nextAuthenticator))
			v := url.Values{}
			v.Set("req", authReq.ID)
			v.Set("hmac", base64.RawURLEncoding.EncodeToString(h.Sum(nil)))
			v.Set("authenticator", nextAuthenticator)
			nextURL := path.Join(s.issuerURL.Path, "/totp/verify") + "?" + v.Encode()
			http.Redirect(w, r, nextURL, http.StatusSeeOther)
			return
		}

		// All authenticators in the chain completed — mark as validated.
		if err := s.storage.UpdateAuthRequest(ctx, authReq.ID, func(old storage.AuthRequest) (storage.AuthRequest, error) {
			old.MFAValidated = true
			return old, nil
		}); err != nil {
			s.logger.ErrorContext(ctx, "failed to update auth request", "err", err)
			s.renderError(r, w, http.StatusInternalServerError, "Internal server error.")
			return
		}

		// Skip approval if configured.
		if s.skipApproval && !authReq.ForceApprovalPrompt {
			authReq, err = s.storage.GetAuthRequest(ctx, authReq.ID)
			if err != nil {
				s.logger.ErrorContext(ctx, "failed to get finalized auth request", "err", err)
				s.renderError(r, w, http.StatusInternalServerError, "Login error.")
				return
			}
			s.sendCodeResponse(w, r, authReq)
			return
		}

		// Skip approval if user already consented to the requested scopes.
		if !authReq.ForceApprovalPrompt {
			ui, err := s.storage.GetUserIdentity(ctx, authReq.Claims.UserID, authReq.ConnectorID)
			if err == nil && scopesCoveredByConsent(ui.Consents[authReq.ClientID], authReq.Scopes) {
				authReq, err = s.storage.GetAuthRequest(ctx, authReq.ID)
				if err != nil {
					s.logger.ErrorContext(ctx, "failed to get auth request", "err", err)
					s.renderError(r, w, http.StatusInternalServerError, "Login error.")
					return
				}
				s.sendCodeResponse(w, r, authReq)
				return
			}
		}

		http.Redirect(w, r, returnURL, http.StatusSeeOther)

	default:
		s.renderError(r, w, http.StatusBadRequest, "Unsupported request method.")
	}
}

func (s *Server) renderTOTPPage(secret *storage.MFASecret, lastFail bool, issuer, connectorID string, w http.ResponseWriter, r *http.Request) {
	var qrCode string
	if !secret.Confirmed {
		var err error
		qrCode, err = generateTOTPQRCode(secret.Secret)
		if err != nil {
			s.logger.ErrorContext(r.Context(), "failed to generate QR code", "err", err)
			s.renderError(r, w, http.StatusInternalServerError, "Internal server error.")
			return
		}
	}
	if err := s.templates.totpVerify(r, w, r.URL.String(), issuer, connectorID, qrCode, lastFail); err != nil {
		s.logger.ErrorContext(r.Context(), "server template error", "err", err)
	}
}

func generateTOTPQRCode(keyURL string) (string, error) {
	generated, err := otp.NewKeyFromURL(keyURL)
	if err != nil {
		return "", fmt.Errorf("failed to load TOTP key: %w", err)
	}

	qrCodeImage, err := generated.Image(300, 300)
	if err != nil {
		return "", fmt.Errorf("failed to generate TOTP QR code: %w", err)
	}

	var buf bytes.Buffer
	if err := png.Encode(&buf, qrCodeImage); err != nil {
		return "", fmt.Errorf("failed to encode TOTP QR code: %w", err)
	}

	return base64.StdEncoding.EncodeToString(buf.Bytes()), nil
}

// mfaChainForClient returns the MFA chain for a client filtered by connector type,
// falling back to the server's defaultMFAChain if the client has none.
// Returns nil if no MFA is configured/applicable.
func (s *Server) mfaChainForClient(ctx context.Context, clientID, connectorID string) ([]string, error) {
	if len(s.mfaProviders) == 0 {
		return nil, nil
	}

	client, err := s.storage.GetClient(ctx, clientID)
	if err != nil {
		return nil, err
	}

	// nil means "not set" — fall back to default.
	// Explicit empty slice ([]string{}) means "no MFA" — don't fall back.
	source := client.MFAChain
	if source == nil {
		source = s.defaultMFAChain
	}

	// Resolve connector type from connector ID.
	connectorType := s.getConnectorType(ctx, connectorID)

	var chain []string
	for _, authID := range source {
		provider, ok := s.mfaProviders[authID]
		if ok && provider.EnabledForConnectorType(connectorType) {
			chain = append(chain, authID)
		}
	}
	return chain, nil
}

// getConnectorType returns the type of the connector with the given ID.
func (s *Server) getConnectorType(ctx context.Context, connectorID string) string {
	conn, err := s.storage.GetConnector(ctx, connectorID)
	if err != nil {
		return ""
	}
	return conn.Type
}
