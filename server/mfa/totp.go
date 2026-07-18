package mfa

import (
	"bytes"
	"context"
	"crypto/subtle"
	"encoding/base64"
	"errors"
	"fmt"
	"image/png"
	"net/http"
	"time"

	"github.com/pquerna/otp"
	"github.com/pquerna/otp/totp"

	"github.com/dexidp/dex/storage"
)

const (
	totpPeriod = 30 // seconds per TOTP time step
	totpSkew   = 1  // number of steps tolerated on either side of the current one
)

var (
	// errTOTPReplay signals the presented code's time-step was already used.
	errTOTPReplay = errors.New("totp code already used")
	// errTOTPNotEnrolled signals the secret disappeared between validation and update.
	errTOTPNotEnrolled = errors.New("totp not enrolled")
)

// validateTOTPCode checks code against secret around now (tolerating totpSkew
// steps) and returns the matched TOTP time-step counter. It returns ok=false
// when the code does not match, or when the matched counter is <= lastCounter —
// the latter enforces single use so a captured code cannot be replayed within
// its validity window.
func validateTOTPCode(secret, code string, now time.Time, lastCounter int64) (ok bool, counter int64) {
	opts := totp.ValidateOpts{
		Period:    totpPeriod,
		Skew:      totpSkew,
		Digits:    otp.DigitsSix,
		Algorithm: otp.AlgorithmSHA1,
	}

	base := now.Unix() / totpPeriod
	// Walk the skew window newest-first so the returned counter is the highest
	// matching step.
	for d := int64(totpSkew); d >= -int64(totpSkew); d-- {
		c := base + d
		if c < 0 {
			continue
		}
		candidate, err := totp.GenerateCodeCustom(secret, time.Unix(c*totpPeriod, 0), opts)
		if err != nil {
			continue
		}
		if subtle.ConstantTimeCompare([]byte(candidate), []byte(code)) == 1 {
			if c <= lastCounter {
				return false, c
			}
			return true, c
		}
	}
	return false, 0
}

// Provider is a pluggable multi-factor authentication method.
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

// mfaRequestContext holds validated MFA request data shared across handlers.
func (h *Handler) handleTOTP(w http.ResponseWriter, r *http.Request) {
	mfa, ok := h.validateMFARequest(w, r)
	if !ok {
		return
	}

	provider, ok := h.MFAProviders[mfa.authenticatorID]
	if !ok {
		h.RenderError(r, w, http.StatusBadRequest, "Unknown authenticator.")
		return
	}
	totpProvider, ok := provider.(*TOTPProvider)
	if !ok {
		h.RenderError(r, w, http.StatusBadRequest, "Not a TOTP authenticator.")
		return
	}

	h.handleTOTPVerify(w, r, r.Context(), mfa.authReq, mfa.identity, mfa.authenticatorID, totpProvider)
}

func (h *Handler) handleTOTPVerify(w http.ResponseWriter, r *http.Request, ctx context.Context,
	authReq storage.AuthRequest, identity storage.UserIdentity,
	authenticatorID string, totpProvider *TOTPProvider,
) {
	secret := identity.MFASecrets[authenticatorID]

	switch r.Method {
	case http.MethodGet:
		if secret == nil {
			// First-time enrollment: generate a new TOTP key.
			// TODO(nabokihms): clean up stale unconfirmed secrets. If a user starts
			// enrollment multiple times without completing it, old secrets accumulate.
			generated, err := totpProvider.generate(authReq.ConnectorID, authReq.Claims.Email)
			if err != nil {
				h.Logger.ErrorContext(ctx, "failed to generate TOTP key", "err", err)
				h.RenderError(r, w, http.StatusInternalServerError, "Internal server error.")
				return
			}

			secret = &storage.MFASecret{
				AuthenticatorID: authenticatorID,
				Type:            "TOTP",
				Secret:          generated.String(),
				Confirmed:       false,
				CreatedAt:       h.Now(),
			}

			if err := h.Storage.UpdateUserIdentity(ctx, authReq.Claims.UserID, authReq.ConnectorID, func(old storage.UserIdentity) (storage.UserIdentity, error) {
				if old.MFASecrets == nil {
					old.MFASecrets = make(map[string]*storage.MFASecret)
				}
				old.MFASecrets[authenticatorID] = secret
				return old, nil
			}); err != nil {
				h.Logger.ErrorContext(ctx, "failed to store MFA secret", "err", err)
				h.RenderError(r, w, http.StatusInternalServerError, "Internal server error.")
				return
			}
		}

		h.renderTOTPPage(secret, false, totpProvider.issuer, authReq.ConnectorID, w, r)

	case http.MethodPost:
		// TODO(nabokihms): this endpoint should be protected with a rate limit (like the auth endpoint).
		// TOTP has a limited keyspace (6 digits) with a 30-second validity window,
		// making it particularly vulnerable to brute-force without rate limiting.
		//
		// For now the best way is to use external rate limiting solutions.
		if secret == nil || secret.Secret == "" {
			h.RenderError(r, w, http.StatusBadRequest, "MFA not enrolled.")
			return
		}

		code := r.FormValue("totp")
		generated, err := otp.NewKeyFromURL(secret.Secret)
		if err != nil {
			h.Logger.ErrorContext(ctx, "failed to load TOTP key", "err", err)
			h.RenderError(r, w, http.StatusInternalServerError, "Internal server error.")
			return
		}

		ok, counter := validateTOTPCode(generated.Secret(), code, h.Now(), secret.LastUsedCounter)
		if !ok {
			h.renderTOTPPage(secret, true, totpProvider.issuer, authReq.ConnectorID, w, r)
			return
		}

		// Record the accepted time-step (single-use / replay protection) and
		// confirm the secret on first successful use. The counter is re-checked
		// against the latest stored value inside the updater so two concurrent
		// requests with the same code cannot both succeed. This burn commits
		// before CompleteStep marks the challenge passed, so a code can never
		// pass the challenge without its counter being recorded first.
		if err := h.Storage.UpdateUserIdentity(ctx, authReq.Claims.UserID, authReq.ConnectorID, func(old storage.UserIdentity) (storage.UserIdentity, error) {
			sec := old.MFASecrets[authenticatorID]
			if sec == nil {
				return old, errTOTPNotEnrolled
			}
			if sec.LastUsedCounter >= counter {
				return old, errTOTPReplay
			}
			sec.Confirmed = true
			sec.LastUsedCounter = counter
			return old, nil
		}); err != nil {
			if errors.Is(err, errTOTPReplay) || errors.Is(err, errTOTPNotEnrolled) {
				h.renderTOTPPage(secret, true, totpProvider.issuer, authReq.ConnectorID, w, r)
				return
			}
			h.Logger.ErrorContext(ctx, "failed to update MFA secret", "err", err)
			h.RenderError(r, w, http.StatusInternalServerError, "Internal server error.")
			return
		}

		redirectURL, err := h.CompleteStep(ctx, authReq, authenticatorID)
		if err != nil {
			h.Logger.ErrorContext(ctx, "failed to complete MFA step", "err", err)
			h.RenderError(r, w, http.StatusInternalServerError, "Internal server error.")
			return
		}

		// CompleteStep returns the next factor URL or the dispatcher URL; the
		// dispatcher decides what follows once MFA is complete.
		http.Redirect(w, r, redirectURL, http.StatusSeeOther)

	default:
		h.RenderError(r, w, http.StatusBadRequest, "Unsupported request method.")
	}
}

func (h *Handler) renderTOTPPage(secret *storage.MFASecret, lastFail bool, issuer, connectorID string, w http.ResponseWriter, r *http.Request) {
	// Prevent browser from caching the TOTP page (contains QR code with secret).
	w.Header().Set("Cache-Control", "no-store")
	var qrCode string
	if !secret.Confirmed {
		var err error
		qrCode, err = generateTOTPQRCode(secret.Secret)
		if err != nil {
			h.Logger.ErrorContext(r.Context(), "failed to generate QR code", "err", err)
			h.RenderError(r, w, http.StatusInternalServerError, "Internal server error.")
			return
		}
	}
	if err := h.Templates.TOTPVerify(r, w, r.URL.String(), issuer, connectorID, qrCode, lastFail); err != nil {
		h.Logger.ErrorContext(r.Context(), "server template error", "err", err)
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

// ChainForClient returns the MFA chain for a client filtered by connector type,
// falling back to the server's defaultMFAChain if the client has none.
// Returns nil if no MFA is configured/applicable.
