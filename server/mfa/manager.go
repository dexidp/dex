package mfa

import (
	"context"
	"fmt"
	"log/slog"
	"net/http"
	"net/url"
	"time"

	"github.com/dexidp/dex/server/connectors"
	"github.com/dexidp/dex/server/internal"
	"github.com/dexidp/dex/server/oauth2"
	"github.com/dexidp/dex/server/render"
	"github.com/dexidp/dex/server/router"
	"github.com/dexidp/dex/server/templates"
	"github.com/dexidp/dex/storage"
)

type Provider interface {
	// Type returns the authenticator type identifier (e.g., "TOTP").
	Type() string
	// EnabledForConnectorType returns true if this provider applies to the given connector type.
	// If no connector types are configured, the provider applies to all.
	EnabledForConnectorType(connectorType string) bool
}

// TOTPProvider implements TOTP-based multi-factor authentication.
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
	*render.UI
	Storage         storage.Storage
	Templates       *templates.Templates
	Logger          *slog.Logger
	MFAProviders    map[string]Provider
	DefaultMFAChain []string
	Now             func() time.Time
	Connectors      *connectors.Cache
}

func (m *Manager) validateMFARequest(w http.ResponseWriter, r *http.Request) (*mfaRequestContext, bool) {
	macEncoded := r.FormValue("hmac")
	if macEncoded == "" {
		m.RenderError(r, w, http.StatusUnauthorized, "Unauthorized request.")
		return nil, false
	}

	ctx := r.Context()

	authReq, err := m.Storage.GetAuthRequest(ctx, r.FormValue("req"))
	if err != nil {
		m.Logger.ErrorContext(ctx, "failed to get auth request", "err", err)
		m.RenderError(r, w, http.StatusInternalServerError, "Database error.")
		return nil, false
	}
	if !authReq.LoggedIn {
		m.Logger.ErrorContext(ctx, "auth request does not have an identity for MFA verification")
		m.RenderError(r, w, http.StatusInternalServerError, "Login process not yet finalized.")
		return nil, false
	}

	authenticatorID := r.FormValue("authenticator")

	if !internal.VerifyHMAC(authReq.HMACKey, macEncoded, authReq.ID, authenticatorID) {
		m.RenderError(r, w, http.StatusUnauthorized, "Unauthorized request.")
		return nil, false
	}

	identity, err := m.Storage.GetUserIdentity(ctx, authReq.Claims.UserID, authReq.ConnectorID)
	if err != nil {
		m.Logger.ErrorContext(ctx, "failed to get user identity", "err", err)
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

func (m *Manager) ChainForClient(ctx context.Context, clientID, connectorID string) ([]string, error) {
	if len(m.MFAProviders) == 0 {
		return nil, nil
	}

	client, err := m.Storage.GetClient(ctx, clientID)
	if err != nil {
		return nil, err
	}

	// nil means "not set" — fall back to default.
	// Explicit empty slice ([]string{}) means "no MFA" — don't fall back.
	source := client.MFAChain
	if source == nil {
		source = m.DefaultMFAChain
	}

	// Resolve connector type from connector ID.
	connectorType, err := m.getConnectorType(ctx, connectorID)
	if err != nil {
		return nil, err
	}

	var chain []string
	for _, authID := range source {
		provider, ok := m.MFAProviders[authID]
		if ok && provider.EnabledForConnectorType(connectorType) {
			chain = append(chain, authID)
		}
	}
	return chain, nil
}

// getConnectorType returns the type of the connector with the given ID.
func (m *Manager) getConnectorType(ctx context.Context, connectorID string) (string, error) {
	conn, err := m.Connectors.Get(ctx, connectorID)
	if err != nil {
		return "", fmt.Errorf("get connector %q: %w", connectorID, err)
	}
	return conn.Type, nil
}

// mfaPagePath returns the page URL path for the given MFA provider type.
func (m *Manager) mfaPagePath(authenticatorID string) string {
	provider, ok := m.MFAProviders[authenticatorID]
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
	if err := m.Storage.UpdateAuthRequest(ctx, authReq.ID, func(old storage.AuthRequest) (storage.AuthRequest, error) {
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

// handleStart is the MFA gate: the first step after login. It decides whether
// the request needs MFA and either sends the user to the first pending factor or
// forwards to the consent step. Login routes every request here, so the gate is
// always mounted even when no authenticators are configured.
func (m *Manager) handleStart(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	mac := r.FormValue("hmac")
	if mac == "" {
		m.RenderError(r, w, http.StatusUnauthorized, "Unauthorized request")
		return
	}
	authReq, err := m.Storage.GetAuthRequest(ctx, r.FormValue("req"))
	if err != nil {
		if err == storage.ErrNotFound {
			m.RenderError(r, w, http.StatusBadRequest, "User session error.")
			return
		}
		m.Logger.ErrorContext(ctx, "failed to get auth request", "err", err)
		m.RenderError(r, w, http.StatusInternalServerError, "Database error.")
		return
	}
	if !authReq.LoggedIn {
		m.Logger.ErrorContext(ctx, "MFA gate reached for auth request without an identity")
		m.RenderError(r, w, http.StatusInternalServerError, "Login process not yet finalized.")
		return
	}
	if !internal.VerifyHMAC(authReq.HMACKey, mac, authReq.ID, "mfa") {
		m.RenderError(r, w, http.StatusUnauthorized, "Unauthorized request")
		return
	}

	chain, err := m.ChainForClient(ctx, authReq.ClientID, authReq.ConnectorID)
	if err != nil {
		m.Logger.ErrorContext(ctx, "failed to determine MFA chain", "err", err)
		m.RenderError(r, w, http.StatusInternalServerError, "Internal server error.")
		return
	}
	if len(chain) > 0 && !authReq.MFAValidated {
		// prompt=none forbids the MFA interaction.
		if prompt, _ := oauth2.ParsePrompt(authReq.Prompt); prompt.None() {
			m.RedirectAuthError(w, r, authReq, oauth2.InteractionRequired, "User interaction required")
			return
		}
		http.Redirect(w, r, m.BuildRedirectURL(authReq, chain[0]), http.StatusSeeOther)
		return
	}

	// MFA satisfied or not required — hand off to the consent step.
	http.Redirect(w, r, m.BuildApprovalURL(authReq), http.StatusSeeOther)
}

// Mount registers the MFA endpoints. The gate is always mounted so login can
// route every request through it; the factor endpoints are mounted only when at
// least one authenticator is configured. MFA is independent of sessions.
func (m *Manager) Mount(mux router.Mux) {
	mux.HandleFunc("/mfa/start", m.handleStart)
	if len(m.MFAProviders) == 0 {
		return
	}
	mux.HandleFunc("/mfa/totp", m.handleTOTP)
	mux.HandleFunc("/mfa/webauthn", m.handleWebAuthn)
	mux.HandleFunc("/mfa/webauthn/register/begin", m.handleWebAuthnRegisterBegin)
	mux.HandleFunc("/mfa/webauthn/register/finish", m.handleWebAuthnRegisterFinish)
	mux.HandleFunc("/mfa/webauthn/login/begin", m.handleWebAuthnLoginBegin)
	mux.HandleFunc("/mfa/webauthn/login/finish", m.handleWebAuthnLoginFinish)
}
