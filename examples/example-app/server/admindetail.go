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
			ID:                     c.Id,
			Name:                   c.Name,
			Secret:                 c.Secret,
			RedirectURIs:           c.RedirectUris,
			TrustedPeers:           c.TrustedPeers,
			Public:                 c.Public,
			LogoURL:                c.LogoUrl,
			AllowedConnectors:      c.AllowedConnectors,
			SSOSharedWith:          c.SsoSharedWith,
			BackchannelLogoutURI:   c.BackchannelLogoutUri,
			PostLogoutRedirectURIs: c.PostLogoutRedirectUris,
			RefreshTokenLifetime:   c.RefreshTokenLifetime,
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

		ConnectorGrantTypes: connectorGrantTypes(),
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

// localPath reports whether a redirect target stays inside this app.
//
// A leading slash is not enough: a browser reads "//host" as another host, and
// so does it read "/\host", because it turns backslashes into slashes before
// resolving. Normalising first and then insisting the parsed URL carries
// neither scheme nor host covers both spellings.
func localPath(target string) bool {
	if !strings.HasPrefix(target, "/") {
		return false
	}

	u, err := url.Parse(strings.ReplaceAll(target, `\`, "/"))
	if err != nil {
		return false
	}

	return u.Scheme == "" && u.Host == "" && strings.HasPrefix(u.Path, "/") && !strings.HasPrefix(u.Path, "//")
}

// detailRedirect returns to the detail page an action was started from.
//
// The target comes from the form, so it is only honoured when it points back
// into this app. Anything else would make these endpoints a way to bounce
// someone off to a site of the sender's choosing.
func (s *Server) detailRedirect(w http.ResponseWriter, r *http.Request, notice, errMsg string) {
	back := r.FormValue("back")
	if !localPath(back) {
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
	// The field is a tri-state: absent leaves the restriction alone, an empty
	// list lifts it, a list replaces it.
	switch r.FormValue("grant_mode") {
	case "any":
		req.NewGrantTypes = &api.GrantTypes{}
	case "restrict":
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

// discoveryEntries asks dex what it says about itself, over the API rather than
// by reading the document it publishes for clients.
func (s *Server) discoveryEntries(ctx context.Context) (endpoints, capabilities []DiscoveryEntry, err error) {
	resp, err := s.admin.api.GetDiscovery(ctx, &api.DiscoveryReq{})
	if err != nil {
		return nil, nil, err
	}

	for _, e := range []DiscoveryEntry{
		{Name: "issuer", Value: resp.Issuer},
		{Name: "authorization_endpoint", Value: resp.AuthorizationEndpoint},
		{Name: "token_endpoint", Value: resp.TokenEndpoint},
		{Name: "userinfo_endpoint", Value: resp.UserinfoEndpoint},
		{Name: "jwks_uri", Value: resp.JwksUri},
		{Name: "device_authorization_endpoint", Value: resp.DeviceAuthorizationEndpoint},
		{Name: "introspection_endpoint", Value: resp.IntrospectionEndpoint},
	} {
		if e.Value != "" {
			endpoints = append(endpoints, e)
		}
	}

	for _, e := range []DiscoveryEntry{
		{Name: "scopes_supported", Values: resp.ScopesSupported},
		{Name: "grant_types_supported", Values: resp.GrantTypesSupported},
		{Name: "response_types_supported", Values: resp.ResponseTypesSupported},
		{Name: "claims_supported", Values: resp.ClaimsSupported},
		{Name: "subject_types_supported", Values: resp.SubjectTypesSupported},
		{Name: "id_token_signing_alg_values_supported", Values: resp.IdTokenSigningAlgValuesSupported},
		{Name: "code_challenge_methods_supported", Values: resp.CodeChallengeMethodsSupported},
		{Name: "token_endpoint_auth_methods_supported", Values: resp.TokenEndpointAuthMethodsSupported},
	} {
		if len(e.Values) > 0 {
			capabilities = append(capabilities, e)
		}
	}

	return endpoints, capabilities, nil
}
