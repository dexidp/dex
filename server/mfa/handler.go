package mfa

import (
	"context"
	"fmt"
	"log/slog"
	"net/http"
	"net/url"
	"path"
	"time"

	"github.com/dexidp/dex/server/connectors"
	"github.com/dexidp/dex/server/internal"
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
}

// Handler owns the MFA domain: the authenticator chain (ChainForClient, which the
// /auth dispatcher queries), the TOTP and WebAuthn factor endpoints, and the
// challenge lifecycle. A completed factor returns to the dispatcher.
type Handler struct {
	Storage         storage.Storage
	Templates       *templates.Templates
	Logger          *slog.Logger
	IssuerURL       url.URL
	MFAProviders    map[string]Provider
	DefaultMFAChain []string
	Now             func() time.Time
	Connectors      *connectors.Cache
}

// renderError renders a user-facing HTML error page.
func (h *Handler) renderError(r *http.Request, w http.ResponseWriter, status int, description string) {
	if err := h.Templates.Err(r, w, status, description); err != nil {
		h.Logger.ErrorContext(r.Context(), "server template error", "err", err)
	}
}

// absPath returns the issuer path joined with the given path items.
func (h *Handler) absPath(pathItems ...string) string {
	return path.Join(append([]string{h.IssuerURL.Path}, pathItems...)...)
}

func (h *Handler) validateMFARequest(w http.ResponseWriter, r *http.Request) (*mfaRequestContext, bool) {
	macEncoded := r.FormValue("hmac")
	if macEncoded == "" {
		h.renderError(r, w, http.StatusUnauthorized, "Unauthorized request.")
		return nil, false
	}

	ctx := r.Context()

	authReq, err := h.Storage.GetAuthRequest(ctx, r.FormValue("req"))
	if err != nil {
		h.Logger.ErrorContext(ctx, "failed to get auth request", "err", err)
		h.renderError(r, w, http.StatusInternalServerError, "Database error.")
		return nil, false
	}
	if !authReq.LoggedIn {
		h.Logger.ErrorContext(ctx, "auth request does not have an identity for MFA verification")
		h.renderError(r, w, http.StatusInternalServerError, "Login process not yet finalized.")
		return nil, false
	}

	authenticatorID := r.FormValue("authenticator")

	if !internal.VerifyHMAC(authReq.HMACKey, macEncoded, authReq.ID, authenticatorID) {
		h.renderError(r, w, http.StatusUnauthorized, "Unauthorized request.")
		return nil, false
	}

	identity, err := h.Storage.GetUserIdentity(ctx, authReq.Claims.UserID, authReq.ConnectorID)
	if err != nil {
		h.Logger.ErrorContext(ctx, "failed to get user identity", "err", err)
		h.renderError(r, w, http.StatusInternalServerError, "Database error.")
		return nil, false
	}

	// Already satisfied — move on to consent instead of showing a factor.
	if authReq.MFAValidated {
		http.Redirect(w, r, h.buildContinueURL(authReq), http.StatusSeeOther)
		return nil, false
	}

	return &mfaRequestContext{
		authReq:         authReq,
		identity:        identity,
		authenticatorID: authenticatorID,
	}, true
}

func (h *Handler) ChainForClient(ctx context.Context, clientID, connectorID string) ([]string, error) {
	if len(h.MFAProviders) == 0 {
		return nil, nil
	}

	client, err := h.Storage.GetClient(ctx, clientID)
	if err != nil {
		return nil, err
	}

	// nil means "not set" — fall back to default.
	// Explicit empty slice ([]string{}) means "no MFA" — don't fall back.
	source := client.MFAChain
	if source == nil {
		source = h.DefaultMFAChain
	}

	// Resolve connector type from connector ID.
	connectorType, err := h.getConnectorType(ctx, connectorID)
	if err != nil {
		return nil, err
	}

	var chain []string
	for _, authID := range source {
		provider, ok := h.MFAProviders[authID]
		if ok && provider.EnabledForConnectorType(connectorType) {
			chain = append(chain, authID)
		}
	}
	return chain, nil
}

// getConnectorType returns the type of the connector with the given ID.
func (h *Handler) getConnectorType(ctx context.Context, connectorID string) (string, error) {
	conn, err := h.Connectors.Get(ctx, connectorID)
	if err != nil {
		return "", fmt.Errorf("get connector %q: %w", connectorID, err)
	}
	return conn.Type, nil
}

// mfaPagePath returns the page URL path for the given MFA provider type.
func (h *Handler) mfaPagePath(authenticatorID string) string {
	provider, ok := h.MFAProviders[authenticatorID]
	if ok && provider.Type() == "WebAuthn" {
		return "/mfa/webauthn"
	}
	return "/mfa/totp"
}

// CompleteStep checks for the next authenticator in the MFA chain and either
// returns the URL for the next factor or marks MFA validated and returns the
// consent URL — the next step in the chain.
func (h *Handler) CompleteStep(ctx context.Context, authReq storage.AuthRequest, authenticatorID string) (string, error) {
	mfaChain, err := h.ChainForClient(ctx, authReq.ClientID, authReq.ConnectorID)
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
		return h.BuildRedirectURL(authReq, nextAuthenticator), nil
	}

	// All authenticators completed — mark as validated.
	if err := h.Storage.UpdateAuthRequest(ctx, authReq.ID, func(old storage.AuthRequest) (storage.AuthRequest, error) {
		old.MFAValidated = true
		return old, nil
	}); err != nil {
		return "", fmt.Errorf("update auth request: %w", err)
	}

	return h.buildContinueURL(authReq), nil
}

// BuildRedirectURL builds an HMAC-protected redirect URL for the given authenticator.
func (h *Handler) BuildRedirectURL(authReq storage.AuthRequest, authenticatorID string) string {
	v := url.Values{}
	v.Set("req", authReq.ID)
	v.Set("hmac", internal.ComputeHMAC(authReq.HMACKey, authReq.ID, authenticatorID))
	v.Set("authenticator", authenticatorID)
	return h.absPath(h.mfaPagePath(authenticatorID)) + "?" + v.Encode()
}

// buildContinueURL builds the HMAC-protected URL that returns to the authorize
// dispatcher (/auth) once a factor is done, so it can decide the next step.
func (h *Handler) buildContinueURL(authReq storage.AuthRequest) string {
	v := url.Values{}
	v.Set("req", authReq.ID)
	v.Set("hmac", internal.ComputeHMAC(authReq.HMACKey, authReq.ID, "continue"))
	return h.absPath("/auth") + "?" + v.Encode()
}

// Mount registers the MFA factor endpoints, only when at least one authenticator
// is configured. The dispatcher (/auth) decides whether MFA applies and sends the
// user to a factor; MFA verifies it and returns to the dispatcher.
func (h *Handler) Mount(mux router.Mux) {
	if len(h.MFAProviders) == 0 {
		return
	}
	mux.HandleFunc("/mfa/totp", h.handleTOTP)
	mux.HandleFunc("/mfa/webauthn", h.handleWebAuthn)
	mux.HandleFunc("/mfa/webauthn/register/begin", h.handleWebAuthnRegisterBegin)
	mux.HandleFunc("/mfa/webauthn/register/finish", h.handleWebAuthnRegisterFinish)
	mux.HandleFunc("/mfa/webauthn/login/begin", h.handleWebAuthnLoginBegin)
	mux.HandleFunc("/mfa/webauthn/login/finish", h.handleWebAuthnLoginFinish)
}
