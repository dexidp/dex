package server

import (
	"context"
	"encoding/base64"
	"fmt"
	"net/http"
	"net/url"
	"strings"
	"time"

	api "github.com/dexidp/dex/api/v2"
)

// The API page lists objects; these pages open one. A list can only show the
// fields that fit, and for a connector the interesting part — its config — never
// does.

// handleAdminClientDetail shows one client in full.
func (s *Server) handleAdminClientDetail(w http.ResponseWriter, r *http.Request) {
	ctx, cancel := context.WithTimeout(r.Context(), 10*time.Second)
	defer cancel()

	id := r.PathValue("id")
	data := AdminDetailPageData{
		LogoURI:      dexLogoDataURI,
		AdminEnabled: true,
		Kind:         "client",
		Title:        id,
		BackURL:      "/admin?section=clients",
	}

	resp, err := s.admin.api.GetClient(ctx, &api.GetClientReq{Id: id})
	switch {
	case err != nil:
		data.Error = err.Error()
	case resp.Client == nil:
		data.Error = fmt.Sprintf("client %q not found", id)
	default:
		c := resp.Client
		data.Client = &AdminClient{
			ID:                c.Id,
			Name:              c.Name,
			Secret:            c.Secret,
			RedirectURIs:      c.RedirectUris,
			TrustedPeers:      c.TrustedPeers,
			Public:            c.Public,
			LogoURL:           c.LogoUrl,
			AllowedConnectors: c.AllowedConnectors,
			SSOSharedWith:     c.SsoSharedWith,
		}
	}

	s.renderer.RenderAdminDetailPage(w, data)
}

// handleAdminConnectorDetail shows one connector, including the configuration
// it was created with.
func (s *Server) handleAdminConnectorDetail(w http.ResponseWriter, r *http.Request) {
	ctx, cancel := context.WithTimeout(r.Context(), 10*time.Second)
	defer cancel()

	id := r.PathValue("id")
	data := AdminDetailPageData{
		LogoURI:      dexLogoDataURI,
		AdminEnabled: true,
		Kind:         "connector",
		Title:        id,
		BackURL:      "/admin?section=connectors",

		ConnectorGrantTypes: []string{
			grantAuthorizationCode, grantRefreshToken, grantDeviceCode,
			grantTokenExchange, grantClientCredentials, grantPassword,
		},
	}

	// The API lists connectors but has no getter, so the list is the lookup.
	resp, err := s.admin.api.ListConnectors(ctx, &api.ListConnectorReq{})
	if err != nil {
		data.Error = err.Error()
		s.renderer.RenderAdminDetailPage(w, data)
		return
	}

	for _, c := range resp.Connectors {
		if c.Id != id {
			continue
		}
		data.Connector = &AdminConnector{
			ID:         c.Id,
			Type:       c.Type,
			Name:       c.Name,
			GrantTypes: c.GrantTypes,
			Config:     indentJSON(c.Config),
		}
		break
	}
	if data.Connector == nil {
		data.Error = fmt.Sprintf("connector %q not found — connectors from the config file are not stored", id)
	}

	s.renderer.RenderAdminDetailPage(w, data)
}

// handleAdminUserDetail shows one identity: its claims, what it has consented
// to, and what it has enrolled for MFA.
func (s *Server) handleAdminUserDetail(w http.ResponseWriter, r *http.Request) {
	ctx, cancel := context.WithTimeout(r.Context(), 10*time.Second)
	defer cancel()

	userID, connectorID := r.PathValue("user"), r.PathValue("connector")
	data := AdminDetailPageData{
		LogoURI:      dexLogoDataURI,
		AdminEnabled: true,
		Kind:         "user",
		Title:        userID,
		BackURL:      "/admin?section=identities",
	}

	resp, err := s.admin.api.GetUserIdentity(ctx, &api.GetUserIdentityReq{
		UserId:      userID,
		ConnectorId: connectorID,
	})
	switch {
	case err != nil:
		data.Error = err.Error()
	case resp.Identity == nil:
		data.Error = fmt.Sprintf("no identity for user %q on connector %q", userID, connectorID)
	default:
		data.Identity = adminIdentity(resp.Identity)
		data.Title = data.Identity.Email
		if data.Title == "" {
			data.Title = userID
		}
	}

	s.renderer.RenderAdminDetailPage(w, data)
}

// adminIdentity converts an identity, keeping everything the API returned.
func adminIdentity(u *api.UserIdentity) *AdminIdentity {
	identity := &AdminIdentity{
		UserID:        u.UserId,
		ConnectorID:   u.ConnectorId,
		Email:         u.Email,
		EmailVerified: u.EmailVerified,
		Username:      u.Username,
		Groups:        u.Groups,
		Created:       epochText(u.CreatedAt),
		LastLogin:     epochText(u.LastLogin),
	}

	for _, c := range u.Consents {
		identity.Consents = append(identity.Consents, AdminConsent{ClientID: c.ClientId, Scopes: c.Scopes})
	}

	for _, d := range u.MfaDevices {
		device := AdminMFADevice{
			AuthenticatorID: d.AuthenticatorId,
			HasSecret:       d.MfaSecret != nil,
		}
		for _, cred := range d.WebauthnCredentials {
			device.Credentials = append(device.Credentials, AdminWebAuthnCredential{
				ID:          base64.RawURLEncoding.EncodeToString(cred.CredentialId),
				DisplayName: cred.DisplayName,
				Transport:   cred.Transport,
				SignCount:   cred.SignCount,
				Created:     epochText(cred.CreatedAt),
			})
		}
		identity.MFADevices = append(identity.MFADevices, device)
	}

	return identity
}

// handleAdminRevokeConsent withdraws a user's approval for one client, which
// puts the consent screen back in front of them next time.
func (s *Server) handleAdminRevokeConsent(w http.ResponseWriter, r *http.Request) {
	if err := r.ParseForm(); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	ctx, cancel := context.WithTimeout(r.Context(), 10*time.Second)
	defer cancel()

	userID, connectorID, clientID := r.FormValue("user_id"), r.FormValue("connector_id"), r.FormValue("client_id")
	resp, err := s.admin.api.RevokeConsent(ctx, &api.RevokeConsentReq{
		UserId:      userID,
		ConnectorId: connectorID,
		ClientId:    clientID,
	})
	switch {
	case err != nil:
		s.detailRedirect(w, r, "", err.Error())
	case resp.NotFound:
		s.detailRedirect(w, r, "", fmt.Sprintf("no consent for client %q", clientID))
	default:
		s.detailRedirect(w, r, fmt.Sprintf("revoked consent for client %q", clientID), "")
	}
}

// handleAdminDeleteWebAuthn removes one registered key.
func (s *Server) handleAdminDeleteWebAuthn(w http.ResponseWriter, r *http.Request) {
	if err := r.ParseForm(); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	credentialID, err := base64.RawURLEncoding.DecodeString(r.FormValue("credential_id"))
	if err != nil {
		s.detailRedirect(w, r, "", "credential id is not base64: "+err.Error())
		return
	}

	ctx, cancel := context.WithTimeout(r.Context(), 10*time.Second)
	defer cancel()

	resp, err := s.admin.api.DeleteWebAuthnCredential(ctx, &api.DeleteWebAuthnCredentialReq{
		UserId:       r.FormValue("user_id"),
		ConnectorId:  r.FormValue("connector_id"),
		CredentialId: credentialID,
	})
	switch {
	case err != nil:
		s.detailRedirect(w, r, "", err.Error())
	case resp.NotFound:
		s.detailRedirect(w, r, "", "credential not found")
	default:
		s.detailRedirect(w, r, "deleted the credential", "")
	}
}

// detailRedirect returns to the detail page an action was started from.
func (s *Server) detailRedirect(w http.ResponseWriter, r *http.Request, notice, errMsg string) {
	back := r.FormValue("back")
	if back == "" {
		back = "/admin"
	}

	sep := "?"
	if strings.Contains(back, "?") {
		sep = "&"
	}
	switch {
	case errMsg != "":
		back += sep + "error=" + url.QueryEscape(errMsg)
	case notice != "":
		back += sep + "notice=" + url.QueryEscape(notice)
	}

	http.Redirect(w, r, back, http.StatusSeeOther)
}

// handleAdminUpdateConnector changes a connector in place. Its type, name,
// config and grant types are all updatable; its id is what looks it up.
func (s *Server) handleAdminUpdateConnector(w http.ResponseWriter, r *http.Request) {
	if err := r.ParseForm(); err != nil {
		s.detailRedirect(w, r, "", err.Error())
		return
	}

	ctx, cancel := context.WithTimeout(r.Context(), 10*time.Second)
	defer cancel()

	id := r.FormValue("id")
	req := &api.UpdateConnectorReq{
		Id:      id,
		NewType: r.FormValue("type"),
		NewName: r.FormValue("name"),
	}
	if config := r.FormValue("config"); config != "" {
		req.NewConfig = []byte(config)
	}
	// An empty list means "no restriction", which is different from leaving the
	// field alone, so it is only sent when the form says to.
	if r.FormValue("set_grant_types") != "" {
		req.NewGrantTypes = &api.GrantTypes{GrantTypes: r.Form["grant_types"]}
	}

	resp, err := s.admin.api.UpdateConnector(ctx, req)
	switch {
	case err != nil:
		s.detailRedirect(w, r, "", err.Error())
	case resp.NotFound:
		s.detailRedirect(w, r, "", fmt.Sprintf("connector %q not found", id))
	default:
		s.detailRedirect(w, r, fmt.Sprintf("updated connector %q", id), "")
	}
}

// handleDiscovery shows the provider's metadata document. The app reads it at
// startup to find endpoints; this is the rest of what it says.
func (s *Server) handleDiscovery(w http.ResponseWriter, r *http.Request) {
	data := DiscoveryPageData{
		LogoURI:      dexLogoDataURI,
		AdminEnabled: s.admin != nil,
		IssuerURL:    s.provider.Endpoint().AuthURL,
	}

	var raw map[string]any
	if err := s.provider.Claims(&raw); err == nil {
		data.Document = jsonOf(raw)

		// The fields worth reading first, rather than hunting for in the
		// document below.
		for _, name := range []string{
			"issuer", "authorization_endpoint", "token_endpoint", "userinfo_endpoint",
			"jwks_uri", "device_authorization_endpoint", "introspection_endpoint",
			"end_session_endpoint", "registration_endpoint",
		} {
			if value, ok := raw[name].(string); ok && value != "" {
				data.Summary = append(data.Summary, DiscoveryEntry{Name: name, Value: value})
			}
		}
		for _, name := range []string{
			"scopes_supported", "response_types_supported", "grant_types_supported",
			"id_token_signing_alg_values_supported", "code_challenge_methods_supported",
			"token_endpoint_auth_methods_supported", "claims_supported", "subject_types_supported",
		} {
			values, ok := raw[name].([]any)
			if !ok || len(values) == 0 {
				continue
			}
			entry := DiscoveryEntry{Name: name}
			for _, v := range values {
				entry.Values = append(entry.Values, fmt.Sprint(v))
			}
			data.Summary = append(data.Summary, entry)
		}
	}

	s.renderer.RenderDiscoveryPage(w, data)
}
