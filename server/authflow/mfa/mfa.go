// Package mfa owns dex's MFA domain: the authenticator chain and the TOTP
// and WebAuthn endpoints. The interactive auth flow delegates to a Manager.
package mfa

import (
	"bytes"
	"context"
	"crypto/subtle"
	"encoding/base64"
	"errors"
	"fmt"
	"image/png"
	"log/slog"
	"net/http"
	"net/url"
	"time"

	"github.com/pquerna/otp"
	"github.com/pquerna/otp/totp"

	"github.com/dexidp/dex/server/authflow/web"
	"github.com/dexidp/dex/server/connectors"
	"github.com/dexidp/dex/server/internal"
	"github.com/dexidp/dex/server/templates"
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
type Provider interface {
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

// mfaRequestContext holds validated MFA request data shared across handlers.
type mfaRequestContext struct {
	authReq         storage.AuthRequest
	identity        storage.UserIdentity
	authenticatorID string
	approvalURL     string
}

// validateMFARequest performs common MFA request validation: HMAC check, auth request
// lookup, user identity lookup, and approval URL generation.
// Manager owns the MFA domain: the authenticator chain, the TOTP and WebAuthn
// endpoints, and challenge lifecycle. It embeds web for error rendering and URL
// building; the Handler delegates the MFA endpoints and chain lookups to it.
type Manager struct {
	*web.UI
	storage         storage.Storage
	templates       *templates.Templates
	logger          *slog.Logger
	mfaProviders    map[string]Provider
	defaultMFAChain []string
	now             func() time.Time
	connectors      *connectors.Cache
}

// New builds an MFA Manager.
func New(ui *web.UI, store storage.Storage, tmpls *templates.Templates, logger *slog.Logger, providers map[string]Provider, defaultChain []string, now func() time.Time, conns *connectors.Cache) *Manager {
	return &Manager{
		UI:              ui,
		storage:         store,
		templates:       tmpls,
		logger:          logger,
		mfaProviders:    providers,
		defaultMFAChain: defaultChain,
		now:             now,
		connectors:      conns,
	}
}

func (m *Manager) validateMFARequest(w http.ResponseWriter, r *http.Request) (*mfaRequestContext, bool) {
	macEncoded := r.FormValue("hmac")
	if macEncoded == "" {
		m.RenderError(r, w, http.StatusUnauthorized, "Unauthorized request.")
		return nil, false
	}

	ctx := r.Context()

	authReq, err := m.storage.GetAuthRequest(ctx, r.FormValue("req"))
	if err != nil {
		m.logger.ErrorContext(ctx, "failed to get auth request", "err", err)
		m.RenderError(r, w, http.StatusInternalServerError, "Database error.")
		return nil, false
	}
	if !authReq.LoggedIn {
		m.logger.ErrorContext(ctx, "auth request does not have an identity for MFA verification")
		m.RenderError(r, w, http.StatusInternalServerError, "Login process not yet finalized.")
		return nil, false
	}

	authenticatorID := r.FormValue("authenticator")

	if !internal.VerifyHMAC(authReq.HMACKey, macEncoded, authReq.ID, authenticatorID) {
		m.RenderError(r, w, http.StatusUnauthorized, "Unauthorized request.")
		return nil, false
	}

	identity, err := m.storage.GetUserIdentity(ctx, authReq.Claims.UserID, authReq.ConnectorID)
	if err != nil {
		m.logger.ErrorContext(ctx, "failed to get user identity", "err", err)
		m.RenderError(r, w, http.StatusInternalServerError, "Database error.")
		return nil, false
	}

	approvalURL := m.BuildApprovalURL(authReq)

	if authReq.MFAValidated {
		http.Redirect(w, r, approvalURL, http.StatusSeeOther)
		return nil, false
	}

	return &mfaRequestContext{
		authReq:         authReq,
		identity:        identity,
		authenticatorID: authenticatorID,
		approvalURL:     approvalURL,
	}, true
}

func (m *Manager) HandleTOTP(w http.ResponseWriter, r *http.Request) {
	mfa, ok := m.validateMFARequest(w, r)
	if !ok {
		return
	}

	provider, ok := m.mfaProviders[mfa.authenticatorID]
	if !ok {
		m.RenderError(r, w, http.StatusBadRequest, "Unknown authenticator.")
		return
	}
	totpProvider, ok := provider.(*TOTPProvider)
	if !ok {
		m.RenderError(r, w, http.StatusBadRequest, "Not a TOTP authenticator.")
		return
	}

	m.handleTOTPVerify(w, r, r.Context(), mfa.authReq, mfa.identity, mfa.authenticatorID, totpProvider, mfa.approvalURL)
}

func (m *Manager) HandleWebAuthn(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		m.RenderError(r, w, http.StatusMethodNotAllowed, "Unsupported request method.")
		return
	}

	mfa, ok := m.validateMFARequest(w, r)
	if !ok {
		return
	}

	w.Header().Set("Cache-Control", "no-store")

	user := buildWebAuthnUser(mfa.identity, mfa.authenticatorID)
	mode := "login"
	if len(user.credentials) == 0 {
		mode = "register"
	}

	if err := m.templates.WebAuthnVerify(r, w, mode, mfa.authenticatorID); err != nil {
		m.logger.ErrorContext(r.Context(), "server template error", "err", err)
	}
}

// handleTOTPVerify handles TOTP enrollment and verification.
func (m *Manager) handleTOTPVerify(w http.ResponseWriter, r *http.Request, ctx context.Context,
	authReq storage.AuthRequest, identity storage.UserIdentity,
	authenticatorID string, totpProvider *TOTPProvider, returnURL string,
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
				m.logger.ErrorContext(ctx, "failed to generate TOTP key", "err", err)
				m.RenderError(r, w, http.StatusInternalServerError, "Internal server error.")
				return
			}

			secret = &storage.MFASecret{
				AuthenticatorID: authenticatorID,
				Type:            "TOTP",
				Secret:          generated.String(),
				Confirmed:       false,
				CreatedAt:       m.now(),
			}

			if err := m.storage.UpdateUserIdentity(ctx, authReq.Claims.UserID, authReq.ConnectorID, func(old storage.UserIdentity) (storage.UserIdentity, error) {
				if old.MFASecrets == nil {
					old.MFASecrets = make(map[string]*storage.MFASecret)
				}
				old.MFASecrets[authenticatorID] = secret
				return old, nil
			}); err != nil {
				m.logger.ErrorContext(ctx, "failed to store MFA secret", "err", err)
				m.RenderError(r, w, http.StatusInternalServerError, "Internal server error.")
				return
			}
		}

		m.renderTOTPPage(secret, false, totpProvider.issuer, authReq.ConnectorID, w, r)

	case http.MethodPost:
		// TODO(nabokihms): this endpoint should be protected with a rate limit (like the auth endpoint).
		// TOTP has a limited keyspace (6 digits) with a 30-second validity window,
		// making it particularly vulnerable to brute-force without rate limiting.
		//
		// For now the best way is to use external rate limiting solutions.
		if secret == nil || secret.Secret == "" {
			m.RenderError(r, w, http.StatusBadRequest, "MFA not enrolled.")
			return
		}

		code := r.FormValue("totp")
		generated, err := otp.NewKeyFromURL(secret.Secret)
		if err != nil {
			m.logger.ErrorContext(ctx, "failed to load TOTP key", "err", err)
			m.RenderError(r, w, http.StatusInternalServerError, "Internal server error.")
			return
		}

		ok, counter := validateTOTPCode(generated.Secret(), code, m.now(), secret.LastUsedCounter)
		if !ok {
			m.renderTOTPPage(secret, true, totpProvider.issuer, authReq.ConnectorID, w, r)
			return
		}

		// Record the accepted time-step (single-use / replay protection) and
		// confirm the secret on first successful use. The counter is re-checked
		// against the latest stored value inside the updater so two concurrent
		// requests with the same code cannot both succeed. This burn commits
		// before CompleteStep marks the challenge passed, so a code can never
		// pass the challenge without its counter being recorded first.
		if err := m.storage.UpdateUserIdentity(ctx, authReq.Claims.UserID, authReq.ConnectorID, func(old storage.UserIdentity) (storage.UserIdentity, error) {
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
				m.renderTOTPPage(secret, true, totpProvider.issuer, authReq.ConnectorID, w, r)
				return
			}
			m.logger.ErrorContext(ctx, "failed to update MFA secret", "err", err)
			m.RenderError(r, w, http.StatusInternalServerError, "Internal server error.")
			return
		}

		redirectURL, err := m.CompleteStep(ctx, authReq, authenticatorID)
		if err != nil {
			m.logger.ErrorContext(ctx, "failed to complete MFA step", "err", err)
			m.RenderError(r, w, http.StatusInternalServerError, "Internal server error.")
			return
		}

		// CompleteStep returns either the next MFA step URL or the approval URL.
		// Redirect in both cases — the approval handler handles skipApproval logic.
		http.Redirect(w, r, redirectURL, http.StatusSeeOther)

	default:
		m.RenderError(r, w, http.StatusBadRequest, "Unsupported request method.")
	}
}

func (m *Manager) renderTOTPPage(secret *storage.MFASecret, lastFail bool, issuer, connectorID string, w http.ResponseWriter, r *http.Request) {
	// Prevent browser from caching the TOTP page (contains QR code with secret).
	w.Header().Set("Cache-Control", "no-store")
	var qrCode string
	if !secret.Confirmed {
		var err error
		qrCode, err = generateTOTPQRCode(secret.Secret)
		if err != nil {
			m.logger.ErrorContext(r.Context(), "failed to generate QR code", "err", err)
			m.RenderError(r, w, http.StatusInternalServerError, "Internal server error.")
			return
		}
	}
	if err := m.templates.TOTPVerify(r, w, r.URL.String(), issuer, connectorID, qrCode, lastFail); err != nil {
		m.logger.ErrorContext(r.Context(), "server template error", "err", err)
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
func (m *Manager) ChainForClient(ctx context.Context, clientID, connectorID string) ([]string, error) {
	if len(m.mfaProviders) == 0 {
		return nil, nil
	}

	client, err := m.storage.GetClient(ctx, clientID)
	if err != nil {
		return nil, err
	}

	// nil means "not set" — fall back to default.
	// Explicit empty slice ([]string{}) means "no MFA" — don't fall back.
	source := client.MFAChain
	if source == nil {
		source = m.defaultMFAChain
	}

	// Resolve connector type from connector ID.
	connectorType, err := m.getConnectorType(ctx, connectorID)
	if err != nil {
		return nil, err
	}

	var chain []string
	for _, authID := range source {
		provider, ok := m.mfaProviders[authID]
		if ok && provider.EnabledForConnectorType(connectorType) {
			chain = append(chain, authID)
		}
	}
	return chain, nil
}

// getConnectorType returns the type of the connector with the given ID.
func (m *Manager) getConnectorType(ctx context.Context, connectorID string) (string, error) {
	conn, err := m.connectors.Get(ctx, connectorID)
	if err != nil {
		return "", fmt.Errorf("get connector %q: %w", connectorID, err)
	}
	return conn.Type, nil
}

// mfaPagePath returns the page URL path for the given MFA provider type.
func (m *Manager) mfaPagePath(authenticatorID string) string {
	provider, ok := m.mfaProviders[authenticatorID]
	if ok && provider.Type() == "WebAuthn" {
		return "/mfa/webauthn"
	}
	return "/mfa/totp"
}

// CompleteStep checks for the next authenticator in the MFA chain and either
// returns the URL for the next step or marks MFA as validated and returns the approval URL.
func (m *Manager) CompleteStep(ctx context.Context, authReq storage.AuthRequest, authenticatorID string) (string, error) {
	mfaChain, err := m.ChainForClient(ctx, authReq.ClientID, authReq.ConnectorID)
	if err != nil {
		return "", fmt.Errorf("get MFA chain: %w", err)
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
		return m.BuildRedirectURL(authReq, nextAuthenticator), nil
	}

	// All authenticators completed — mark as validated.
	if err := m.storage.UpdateAuthRequest(ctx, authReq.ID, func(old storage.AuthRequest) (storage.AuthRequest, error) {
		old.MFAValidated = true
		return old, nil
	}); err != nil {
		return "", fmt.Errorf("update auth request: %w", err)
	}

	return m.BuildApprovalURL(authReq), nil
}

// BuildRedirectURL builds an HMAC-protected redirect URL for the given authenticator.
func (m *Manager) BuildRedirectURL(authReq storage.AuthRequest, authenticatorID string) string {
	v := url.Values{}
	v.Set("req", authReq.ID)
	v.Set("hmac", internal.ComputeHMAC(authReq.HMACKey, authReq.ID, authenticatorID))
	v.Set("authenticator", authenticatorID)
	return m.AbsPath(m.mfaPagePath(authenticatorID)) + "?" + v.Encode()
}
