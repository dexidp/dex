package server

import "net/http"

// A form page is one flow's parameters and nothing else. The front page lists
// flows; choosing one opens its form. That split is the whole point: parameters
// for six grants on one screen is a wall of inputs with no way to tell which
// belong together, and every one of them is irrelevant to whatever you came to
// do.
type Field struct {
	Name        string
	Label       string
	Hint        string
	Type        string // text, password, textarea, select, scopes
	Placeholder string
	Value       string
	Options     []Option
	Required    bool
	Wide        bool // full-width control, for tokens rather than identifiers
}

// scopeField offers the scopes the provider advertises, with the usual ones
// ticked — the same control the browser flows use, rather than a text box you
// have to know the spelling for.
func (s *Server) scopeField(defaults ...string) Field {
	chosen := make(map[string]bool, len(defaults))
	for _, d := range defaults {
		chosen[d] = true
	}

	options := make([]Option, 0, len(s.scopesSupported))
	for _, scope := range s.scopesSupported {
		options = append(options, Option{Value: scope, Label: scope, Selected: chosen[scope]})
	}

	return Field{
		Name:    "scopes",
		Label:   "Scopes",
		Type:    "scopes",
		Wide:    true,
		Options: options,
		Hint:    "What the token should be good for. Add any the provider does not advertise.",
	}
}

// Option is one choice in a select or scope field.
type Option struct {
	Value    string
	Label    string
	Selected bool
}

// FormPageData is a page that asks for parameters and posts them somewhere.
type FormPageData struct {
	LogoURI      string
	AdminEnabled bool
	Title        string
	Description  string
	Note         string
	Action       string
	Submit       string
	Fields       []Field
}

// handleGrantForm serves the parameters for one direct grant.
func (s *Server) handleGrantForm(w http.ResponseWriter, r *http.Request) {
	sess := s.session(w, r)

	var data FormPageData
	switch r.PathValue("grant") {
	case "client-credentials":
		data = FormPageData{
			Title:       "Client credentials",
			Description: "A token for this application itself. There is no user in this flow, and so no ID token — what comes back says only which client it belongs to.",
			Action:      "/grant/client-credentials",
			Submit:      "Request token",
			Fields:      []Field{s.scopeField("openid")},
		}
	case "password":
		data = FormPageData{
			Title:       "Password",
			Description: "Exchanges a username and password for tokens directly.",
			Note:        "Deprecated. The password passes through this application instead of staying at the provider, which is the thing OAuth exists to avoid. dex allows it only when oauth2.passwordConnector names a connector.",
			Action:      "/grant/password",
			Submit:      "Request token",
			Fields: []Field{
				{Name: "username", Label: "Username", Type: "text", Placeholder: "user@example.com", Required: true},
				{Name: "password", Label: "Password", Type: "password", Required: true},
				s.scopeField("openid", "profile", "email", "offline_access"),
			},
		}
	case "token-exchange":
		data = FormPageData{
			Title:       "Token exchange",
			Description: "RFC 8693. Trades a token issued elsewhere for one issued by dex.",
			Note:        "dex requires connector_id, which is an extension to the RFC: the connector is what verifies the subject token, so it must be one that can.",
			Action:      "/grant/token-exchange",
			Submit:      "Exchange",
			Fields: []Field{
				{Name: "subject_token", Label: "Subject token", Type: "textarea", Required: true, Wide: true, Hint: "The token you are trading in."},
				{Name: "connector_id", Label: "Connector", Type: "text", Placeholder: "mock", Required: true, Hint: "Which connector verifies the subject token."},
				{
					Name: "subject_token_type", Label: "Subject token type", Type: "select",
					Options: []Option{
						{Value: tokenTypeAccess, Label: "Access token"},
						{Value: tokenTypeID, Label: "ID token"},
					},
				},
				{
					Name: "requested_token_type", Label: "Requested token type", Type: "select",
					Options: []Option{
						{Value: "", Label: "Default (access token)"},
						{Value: tokenTypeID, Label: "ID token"},
						{Value: tokenTypeAccess, Label: "Access token"},
					},
				},
				s.scopeField("openid"),
			},
		}
	case "refresh":
		value := ""
		if sess.LastTokens != nil {
			value = sess.LastTokens.RefreshToken
		}
		if value == "" && sess.Token != nil {
			value = sess.Token.RefreshToken
		}
		data = FormPageData{
			Title:       "Refresh",
			Description: "Redeems a refresh token for a new set of tokens. dex rotates the refresh token, so the one that comes back replaces the one you sent.",
			Action:      "/grant/refresh",
			Submit:      "Redeem",
			Fields: []Field{
				{Name: "refresh_token", Label: "Refresh token", Type: "textarea", Required: true, Wide: true, Value: value},
			},
		}
	default:
		http.NotFound(w, r)
		return
	}

	data.LogoURI = dexLogoDataURI
	data.AdminEnabled = s.admin != nil
	s.renderer.RenderFormPage(w, data)
}

// handleToolForm serves the parameters for one token tool.
func (s *Server) handleToolForm(w http.ResponseWriter, r *http.Request) {
	sess := s.session(w, r)

	// Whatever this browser last got back is the likeliest thing to want to look
	// at, so it is filled in already — including from flows that sign nobody in.
	var accessToken, idToken string
	if sess.LastTokens != nil {
		accessToken = sess.LastTokens.AccessToken
		idToken = sess.LastIDToken
	} else if sess.Token != nil {
		accessToken = sess.Token.AccessToken
		idToken = sess.IDToken
	}

	var data FormPageData
	switch r.PathValue("tool") {
	case "introspect":
		data = FormPageData{
			Title:       "Introspection",
			Description: "Asks dex what it makes of a token — access, ID or refresh. This is the only way to learn that one has been revoked: a revoked token still has a valid signature.",
			Action:      "/tools/introspect",
			Submit:      "Introspect",
			Fields: []Field{
				{Name: "token", Label: "Token", Type: "textarea", Required: true, Wide: true, Value: accessToken},
				{
					Name: "token_type_hint", Label: "Type hint", Type: "select",
					Hint: "Optional — dex works the type out from the token itself and only logs a mismatch.",
					Options: []Option{
						{Value: "", Label: "no hint"},
						{Value: "access_token", Label: "access_token"},
						{Value: "id_token", Label: "id_token"},
						{Value: "refresh_token", Label: "refresh_token"},
					},
				},
			},
		}
	case "verify":
		data = FormPageData{
			Title:       "Local verification",
			Description: "Checks the signature against the issuer's published keys and reads the claims, the way a resource server would. It says nothing about revocation.",
			Action:      "/tools/verify",
			Submit:      "Verify",
			Fields: []Field{
				{Name: "token", Label: "JWT", Type: "textarea", Required: true, Wide: true, Value: idToken},
			},
		}
	case "userinfo":
		data = FormPageData{
			Title:       "UserInfo",
			Description: "Calls the provider's UserInfo endpoint with an access token, which is how a client gets claims it did not ask to have in the ID token.",
			Action:      "/userinfo",
			Submit:      "Call UserInfo",
			Fields: []Field{
				{Name: "access_token", Label: "Access token", Type: "textarea", Required: true, Wide: true, Value: accessToken},
			},
		}
	default:
		http.NotFound(w, r)
		return
	}

	data.LogoURI = dexLogoDataURI
	data.AdminEnabled = s.admin != nil
	s.renderer.RenderFormPage(w, data)
}
