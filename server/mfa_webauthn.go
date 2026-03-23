package server

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"time"

	gowebauthn "github.com/go-webauthn/webauthn/protocol"
	"github.com/go-webauthn/webauthn/webauthn"

	"github.com/dexidp/dex/storage"
)

// WebAuthnProvider implements WebAuthn-based multi-factor authentication.
type WebAuthnProvider struct {
	wan            *webauthn.WebAuthn
	connectorTypes map[string]struct{}
}

// NewWebAuthnProvider creates a new WebAuthn MFA provider.
// If rpID or rpOrigins are empty, they are derived from the issuerURL.
func NewWebAuthnProvider(rpDisplayName, rpID string, rpOrigins []string,
	attestationPreference, userVerification, authenticatorAttachment, timeout, issuerURL string,
	connectorTypes []string,
) (*WebAuthnProvider, error) {
	parsed, err := url.Parse(issuerURL)
	if err != nil {
		return nil, fmt.Errorf("parse issuer URL: %w", err)
	}

	if rpID == "" {
		rpID = parsed.Hostname()
	}
	if len(rpOrigins) == 0 {
		rpOrigins = []string{parsed.Scheme + "://" + parsed.Host}
	}
	if rpDisplayName == "" {
		rpDisplayName = rpID
	}

	cfg := &webauthn.Config{
		RPID:          rpID,
		RPDisplayName: rpDisplayName,
		RPOrigins:     rpOrigins,
	}

	switch attestationPreference {
	case "none":
		cfg.AttestationPreference = gowebauthn.PreferNoAttestation
	case "direct":
		cfg.AttestationPreference = gowebauthn.PreferDirectAttestation
	case "", "indirect":
		cfg.AttestationPreference = gowebauthn.PreferIndirectAttestation
	}

	if timeout != "" {
		d, err := time.ParseDuration(timeout)
		if err != nil {
			return nil, fmt.Errorf("parse timeout: %w", err)
		}
		timeoutCfg := webauthn.TimeoutConfig{Enforce: true, Timeout: d}
		cfg.Timeouts = webauthn.TimeoutsConfig{
			Login:        timeoutCfg,
			Registration: timeoutCfg,
		}
	}

	wan, err := webauthn.New(cfg)
	if err != nil {
		return nil, fmt.Errorf("create webauthn: %w", err)
	}

	m := make(map[string]struct{}, len(connectorTypes))
	for _, t := range connectorTypes {
		m[t] = struct{}{}
	}

	return &WebAuthnProvider{wan: wan, connectorTypes: m}, nil
}

func (p *WebAuthnProvider) Type() string { return "WebAuthn" }

func (p *WebAuthnProvider) EnabledForConnectorType(connectorType string) bool {
	if len(p.connectorTypes) == 0 {
		return true
	}
	_, ok := p.connectorTypes[connectorType]
	return ok
}

// webauthnUser implements the webauthn.User interface.
type webauthnUser struct {
	id          []byte
	name        string
	displayName string
	credentials []webauthn.Credential
}

func (u *webauthnUser) WebAuthnID() []byte                         { return u.id }
func (u *webauthnUser) WebAuthnName() string                       { return u.name }
func (u *webauthnUser) WebAuthnDisplayName() string                { return u.displayName }
func (u *webauthnUser) WebAuthnCredentials() []webauthn.Credential { return u.credentials }

// buildWebAuthnUser creates a webauthn.User from a UserIdentity, using credentials for the given authenticatorID.
func buildWebAuthnUser(identity storage.UserIdentity, authenticatorID string) *webauthnUser {
	stored := identity.WebAuthnCredentials[authenticatorID]
	creds := make([]webauthn.Credential, 0, len(stored))
	for _, c := range stored {
		transports := make([]gowebauthn.AuthenticatorTransport, len(c.Transport))
		for i, t := range c.Transport {
			transports[i] = gowebauthn.AuthenticatorTransport(t)
		}
		creds = append(creds, webauthn.Credential{
			ID:              c.CredentialID,
			PublicKey:       c.PublicKey,
			AttestationType: c.AttestationType,
			Transport:       transports,
			Flags: webauthn.CredentialFlags{
				BackupEligible: c.BackupEligible,
				BackupState:    c.BackupState,
			},
			Authenticator: webauthn.Authenticator{
				AAGUID:       c.AAGUID,
				SignCount:    c.SignCount,
				CloneWarning: c.CloneWarning,
			},
		})
	}

	return &webauthnUser{
		id:          []byte(identity.UserID + "|" + identity.ConnectorID),
		name:        identity.Claims.Email,
		displayName: identity.Claims.PreferredUsername,
		credentials: creds,
	}
}

// handleWebAuthnRegisterBegin starts the WebAuthn registration ceremony.
func (s *Server) handleWebAuthnRegisterBegin(w http.ResponseWriter, r *http.Request) {
	mfa, provider, ok := s.validateWebAuthnAPIRequest(w, r)
	if !ok {
		return
	}

	ctx := r.Context()
	user := buildWebAuthnUser(mfa.identity, mfa.authenticatorID)
	user.credentials = nil // don't exclude existing credentials during registration

	creation, session, err := provider.wan.BeginRegistration(user)
	if err != nil {
		s.logger.ErrorContext(ctx, "failed to begin webauthn registration", "err", err)
		writeJSONError(w, http.StatusInternalServerError, "Internal server error.")
		return
	}

	if err := s.storeWebAuthnSession(ctx, mfa.authReq.ID, session); err != nil {
		s.logger.ErrorContext(ctx, "failed to store session data", "err", err)
		writeJSONError(w, http.StatusInternalServerError, "Internal server error.")
		return
	}

	writeJSON(w, creation)
}

// handleWebAuthnRegisterFinish completes the WebAuthn registration ceremony.
func (s *Server) handleWebAuthnRegisterFinish(w http.ResponseWriter, r *http.Request) {
	mfa, provider, ok := s.validateWebAuthnAPIRequest(w, r)
	if !ok {
		return
	}

	ctx := r.Context()
	session, err := s.loadWebAuthnSession(mfa.authReq)
	if err != nil {
		s.logger.ErrorContext(ctx, "failed to load session data", "err", err)
		writeJSONError(w, http.StatusBadRequest, "Invalid session.")
		return
	}

	user := buildWebAuthnUser(mfa.identity, mfa.authenticatorID)
	user.credentials = nil

	credential, err := provider.wan.FinishRegistration(user, *session, r)
	if err != nil {
		s.logger.ErrorContext(ctx, "webauthn registration failed", "err", err)
		writeJSONError(w, http.StatusBadRequest, "Registration failed: "+err.Error())
		return
	}

	newCred := convertCredential(credential, s.now())
	if err := s.storage.UpdateUserIdentity(ctx, mfa.authReq.Claims.UserID, mfa.authReq.ConnectorID, func(old storage.UserIdentity) (storage.UserIdentity, error) {
		if old.WebAuthnCredentials == nil {
			old.WebAuthnCredentials = make(map[string][]storage.WebAuthnCredential)
		}
		old.WebAuthnCredentials[mfa.authenticatorID] = append(old.WebAuthnCredentials[mfa.authenticatorID], newCred)
		return old, nil
	}); err != nil {
		s.logger.ErrorContext(ctx, "failed to store credential", "err", err)
		writeJSONError(w, http.StatusInternalServerError, "Internal server error.")
		return
	}

	s.writeCompleteMFAStepResponse(w, r, mfa)
}

// handleWebAuthnLoginBegin starts the WebAuthn login ceremony.
func (s *Server) handleWebAuthnLoginBegin(w http.ResponseWriter, r *http.Request) {
	mfa, provider, ok := s.validateWebAuthnAPIRequest(w, r)
	if !ok {
		return
	}

	ctx := r.Context()
	user := buildWebAuthnUser(mfa.identity, mfa.authenticatorID)
	if len(user.credentials) == 0 {
		writeJSONError(w, http.StatusBadRequest, "No WebAuthn credentials registered.")
		return
	}

	assertion, session, err := provider.wan.BeginLogin(user)
	if err != nil {
		s.logger.ErrorContext(ctx, "failed to begin webauthn login", "err", err)
		writeJSONError(w, http.StatusInternalServerError, "Internal server error.")
		return
	}

	if err := s.storeWebAuthnSession(ctx, mfa.authReq.ID, session); err != nil {
		s.logger.ErrorContext(ctx, "failed to store session data", "err", err)
		writeJSONError(w, http.StatusInternalServerError, "Internal server error.")
		return
	}

	writeJSON(w, assertion)
}

// handleWebAuthnLoginFinish completes the WebAuthn login ceremony.
func (s *Server) handleWebAuthnLoginFinish(w http.ResponseWriter, r *http.Request) {
	mfa, provider, ok := s.validateWebAuthnAPIRequest(w, r)
	if !ok {
		return
	}

	ctx := r.Context()
	session, err := s.loadWebAuthnSession(mfa.authReq)
	if err != nil {
		s.logger.ErrorContext(ctx, "failed to load session data", "err", err)
		writeJSONError(w, http.StatusBadRequest, "Invalid session.")
		return
	}

	user := buildWebAuthnUser(mfa.identity, mfa.authenticatorID)
	credential, err := provider.wan.FinishLogin(user, *session, r)
	if err != nil {
		s.logger.ErrorContext(ctx, "webauthn login failed", "err", err)
		writeJSONError(w, http.StatusUnauthorized, "Authentication failed.")
		return
	}

	// Update sign count and clone warning for the matched credential.
	if err := s.storage.UpdateUserIdentity(ctx, mfa.authReq.Claims.UserID, mfa.authReq.ConnectorID, func(old storage.UserIdentity) (storage.UserIdentity, error) {
		creds := old.WebAuthnCredentials[mfa.authenticatorID]
		for i := range creds {
			if string(creds[i].CredentialID) == string(credential.ID) {
				creds[i].SignCount = credential.Authenticator.SignCount
				creds[i].CloneWarning = credential.Authenticator.CloneWarning
				break
			}
		}
		old.WebAuthnCredentials[mfa.authenticatorID] = creds
		return old, nil
	}); err != nil {
		s.logger.ErrorContext(ctx, "failed to update credential", "err", err)
		writeJSONError(w, http.StatusInternalServerError, "Internal server error.")
		return
	}

	s.writeCompleteMFAStepResponse(w, r, mfa)
}

// validateWebAuthnAPIRequest validates a WebAuthn JSON API request.
// It reuses validateMFARequest for HMAC/auth checks, then asserts the provider type
// and loads the user identity.
func (s *Server) validateWebAuthnAPIRequest(w http.ResponseWriter, r *http.Request) (*mfaRequestContext, *WebAuthnProvider, bool) {
	if r.Method != http.MethodPost {
		writeJSONError(w, http.StatusMethodNotAllowed, "Method not allowed.")
		return nil, nil, false
	}

	mfa, ok := s.validateMFARequest(w, r)
	if !ok {
		return nil, nil, false
	}

	provider, ok := s.mfaProviders[mfa.authenticatorID]
	if !ok {
		writeJSONError(w, http.StatusBadRequest, "Unknown authenticator.")
		return nil, nil, false
	}
	webauthnProvider, ok := provider.(*WebAuthnProvider)
	if !ok {
		writeJSONError(w, http.StatusBadRequest, "Not a WebAuthn authenticator.")
		return nil, nil, false
	}

	return mfa, webauthnProvider, true
}

// storeWebAuthnSession marshals and stores WebAuthn session data in the auth request.
func (s *Server) storeWebAuthnSession(ctx context.Context, authReqID string, session *webauthn.SessionData) error {
	data, err := json.Marshal(session)
	if err != nil {
		return fmt.Errorf("marshal session: %w", err)
	}
	return s.storage.UpdateAuthRequest(ctx, authReqID, func(old storage.AuthRequest) (storage.AuthRequest, error) {
		old.WebAuthnSessionData = data
		return old, nil
	})
}

// loadWebAuthnSession unmarshals WebAuthn session data from an auth request.
func (s *Server) loadWebAuthnSession(authReq storage.AuthRequest) (*webauthn.SessionData, error) {
	var session webauthn.SessionData
	if err := json.Unmarshal(authReq.WebAuthnSessionData, &session); err != nil {
		return nil, fmt.Errorf("unmarshal session: %w", err)
	}
	return &session, nil
}

// writeCompleteMFAStepResponse completes the MFA step and writes a JSON redirect response.
func (s *Server) writeCompleteMFAStepResponse(w http.ResponseWriter, r *http.Request, mfa *mfaRequestContext) {
	ctx := r.Context()
	redirectURL, err := s.completeMFAStep(ctx, mfa.authReq, mfa.authenticatorID)
	if err != nil {
		s.logger.ErrorContext(ctx, "failed to complete MFA step", "err", err)
		writeJSONError(w, http.StatusInternalServerError, "Internal server error.")
		return
	}
	writeJSON(w, map[string]string{"status": "ok", "redirect": redirectURL})
}

// convertCredential converts a webauthn.Credential to a storage.WebAuthnCredential.
func convertCredential(cred *webauthn.Credential, now time.Time) storage.WebAuthnCredential {
	transports := make([]string, len(cred.Transport))
	for i, t := range cred.Transport {
		transports[i] = string(t)
	}
	return storage.WebAuthnCredential{
		CredentialID:    cred.ID,
		PublicKey:       cred.PublicKey,
		AttestationType: cred.AttestationType,
		AAGUID:          cred.Authenticator.AAGUID,
		SignCount:       cred.Authenticator.SignCount,
		CloneWarning:    cred.Authenticator.CloneWarning,
		Transport:       transports,
		BackupEligible:  cred.Flags.BackupEligible,
		BackupState:     cred.Flags.BackupState,
		DisplayName:     "Security Key",
		CreatedAt:       now,
	}
}

// Dex return JSON errors because of the dynamic nature of WebAuthn, they are received
// by a JavaScript code and printed for users.
func writeJSON(w http.ResponseWriter, v any) {
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(v)
}

func writeJSONError(w http.ResponseWriter, status int, message string) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(map[string]string{"error": message})
}
