package server

import (
	"embed"
	"html/template"
	"io/fs"
	"log"
	"net/http"
	"time"

	"golang.org/x/oauth2"

	"github.com/dexidp/dex/examples/example-app/session"
)

//go:embed templates/*.html
var templatesFS embed.FS

//go:embed static/*
var staticFS embed.FS

const dexLogoDataURI = "/static/dex-glyph-color.svg"

// staticHandler serves embedded static assets.
var staticHandler http.Handler

func init() {
	staticSubFS, err := fs.Sub(staticFS, "static")
	if err != nil {
		log.Fatalf("failed to create static sub filesystem: %v", err)
	}
	staticHandler = http.FileServer(http.FS(staticSubFS))
}

// TokenSummary is what the index page says about the tokens this browser holds.
type TokenSummary struct {
	AccessToken  string
	IDToken      string
	RefreshToken bool
	Expiry       string
	Expired      bool
}

// tokenSummary describes a token set without printing it in full.
func tokenSummary(token *oauth2.Token, rawIDToken string) *TokenSummary {
	if token == nil {
		return nil
	}
	s := &TokenSummary{
		AccessToken:  abbreviate(token.AccessToken),
		IDToken:      abbreviate(rawIDToken),
		RefreshToken: token.RefreshToken != "",
	}
	if !token.Expiry.IsZero() {
		s.Expiry = token.Expiry.Format(time.RFC3339)
		s.Expired = time.Now().After(token.Expiry)
	}
	return s
}

func abbreviate(token string) string {
	const keep = 12
	if len(token) <= keep*2+1 {
		return token
	}
	return token[:keep] + "…" + token[len(token)-keep:]
}

// IndexPageData holds data for the index page template.
type IndexPageData struct {
	ScopesSupported []string
	LogoURI         string
	AdminEnabled    bool
	PKCE            bool
	SessionCheck    time.Duration

	// Which flows the provider says it supports. A button for a grant dex will
	// refuse is a button that only teaches you it does not work.
	DeviceSupported            bool
	RefreshSupported           bool
	ClientCredentialsSupported bool
	PasswordSupported          bool
	TokenExchangeSupported     bool

	User  *session.UserClaims
	Token *TokenSummary
}

// TokenPageData holds data for the token display template.
type TokenPageData struct {
	LogoURI            string
	AdminEnabled       bool
	Grant              string
	IDToken            string
	IDTokenJWTLink     string
	AccessToken        string
	AccessTokenJWTLink string
	RefreshToken       string
	IssuedTokenType    string
	ExpiresIn          string
	RedirectURL        string
	Claims             string
	RawResponse        string
	PublicKeyPEM       string
}

// DevicePageData holds data for the device flow template.
type DevicePageData struct {
	AdminEnabled    bool
	DeviceCode      string
	UserCode        string
	VerificationURI string
	PollInterval    int
	LogoURI         string
}

// ToolsPageData holds data for the token tools page.
type ToolsPageData struct {
	LogoURI      string
	AdminEnabled bool
}

// ResultPageData holds the output of a tool.
type ResultPageData struct {
	LogoURI      string
	AdminEnabled bool
	Title        string
	Verdict      string
	Body         string
	// LastGrant names the flow whose tokens are still on the session, so a tool
	// result is not a dead end with the tokens left behind on a page you can no
	// longer reach.
	LastGrant string
}

// AdminConnector is one connector as the API reports it.
type AdminConnector struct {
	ID         string
	Type       string
	Name       string
	GrantTypes []string
	Config     string
}

// AdminIdentity is one user identity as the API reports it.
type AdminIdentity struct {
	UserID        string
	ConnectorID   string
	Email         string
	EmailVerified bool
	Username      string
	Groups        []string
	MFADevices    []AdminMFADevice
	Consents      []AdminConsent
	Created       string
	LastLogin     string
}

// AdminConsent is one client a user has approved, and for what.
type AdminConsent struct {
	ClientID string
	Scopes   []string
}

// AdminMFADevice is one enrolled authenticator.
type AdminMFADevice struct {
	AuthenticatorID string
	HasSecret       bool
	Credentials     []AdminWebAuthnCredential
}

// AdminWebAuthnCredential is one registered key.
type AdminWebAuthnCredential struct {
	ID          string
	DisplayName string
	Transport   []string
	SignCount   uint32
	Created     string
}

// AdminDetailPageData is one object with everything the API knows about it.
// The lists say what a thing is; this says what it holds.
type AdminDetailPageData struct {
	LogoURI      string
	AdminEnabled bool
	Kind         string // client, connector, user
	Title        string
	BackURL      string
	Error        string

	Client    *AdminClient
	Connector *AdminConnector
	Identity  *AdminIdentity

	// ConnectorGrantTypes are the grants a connector can be restricted to.
	ConnectorGrantTypes []Option
}

// DiscoveryEntry is one metadata field worth reading without scrolling.
type DiscoveryEntry struct {
	Name   string
	Value  string
	Values []string
}

// AdminSession is one of dex's own sessions as the API reports it.
type AdminSession struct {
	UserID      string
	ConnectorID string
	IPAddress   string
	Created     string
	Expires     string
}

// AdminRefreshToken is one refresh token as the API reports it.
type AdminRefreshToken struct {
	ID       string
	ClientID string
	Created  string
	LastUsed string
}

// AdminClient is one OAuth2 client as the API reports it — every field it has,
// since a list that shows three of nine invites you to guess the rest.
type AdminClient struct {
	ID                     string
	Name                   string
	Secret                 string
	RedirectURIs           []string
	TrustedPeers           []string
	Public                 bool
	LogoURL                string
	AllowedConnectors      []string
	SSOSharedWith          []string
	BackchannelLogoutURI   string
	PostLogoutRedirectURIs []string
	RefreshTokenLifetime   string
}

// AdminPassword is one local password entry as the API reports it.
type AdminPassword struct {
	Email    string
	Username string
	UserID   string
}

// AdminPageData holds data for the gRPC API page. Configured is false when the
// app was started without --grpc-addr: the page still exists and says what to
// pass, because a feature that is simply absent reads as a feature that is
// missing.
type AdminPageData struct {
	LogoURI      string
	AdminEnabled bool
	Configured   bool
	Version      string
	Issuer       string
	Notice       string
	Error        string

	// Section is which part of the API the page is showing. The API has more
	// than twenty methods; one page of all of them is the stack this replaced.
	Section  string
	Sections []AdminSection

	Clients       []AdminClient
	Passwords     []AdminPassword
	Connectors    []AdminConnector
	Identities    []AdminIdentity
	Sessions      []AdminSession
	RefreshTokens []AdminRefreshToken
	// UserID and ConnectorID are who the sessions section is about. The refresh
	// listing needs the sub claim, which the app derives from the pair rather
	// than asking for a value nobody has to hand.
	UserID      string
	ConnectorID string

	// Mode is which form the section is showing, if any: create, edit, verify.
	// One at a time — three stacked forms is a page you have to read to find
	// the one you wanted.
	Mode string

	// ConnectorTypes are the types dex accepts, and ConnectorGrantTypes the
	// grants a connector can be restricted to, both for the create form.
	ConnectorTypes      []string
	ConnectorGrantTypes []Option

	// Endpoints and Capabilities are what GetDiscovery reports.
	Endpoints    []DiscoveryEntry
	Capabilities []DiscoveryEntry

	// EditClient and EditPassword are prefilled from the row an edit was
	// started from, so updating is something you do to a thing you can see.
	EditClient   *AdminClient
	EditPassword *AdminPassword
}

// AdminSection is one tab of the API page.
type AdminSection struct {
	ID      string
	Label   string
	Current bool
}

// Renderer renders HTML pages for the application.
type Renderer interface {
	RenderIndexPage(w http.ResponseWriter, data IndexPageData)
	RenderTokenPage(w http.ResponseWriter, data TokenPageData)
	RenderDevicePage(w http.ResponseWriter, data DevicePageData)
	RenderToolsPage(w http.ResponseWriter, data ToolsPageData)
	RenderFormPage(w http.ResponseWriter, data FormPageData)
	RenderResultPage(w http.ResponseWriter, data ResultPageData)
	RenderAdminPage(w http.ResponseWriter, data AdminPageData)
	RenderAdminDetailPage(w http.ResponseWriter, data AdminDetailPageData)
}

// templateRenderer implements Renderer using Go html/template.
type templateRenderer struct {
	index  *template.Template
	token  *template.Template
	device *template.Template
	tools  *template.Template
	form   *template.Template
	result *template.Template
	admin  *template.Template
	detail *template.Template
}

// newTemplateRenderer parses embedded templates and returns a Renderer.
func newTemplateRenderer() Renderer {
	parse := func(name string) *template.Template {
		t, err := template.ParseFS(templatesFS, "templates/layout.html", "templates/"+name)
		if err != nil {
			log.Fatalf("failed to parse template %s: %v", name, err)
		}
		return t
	}

	return &templateRenderer{
		index:  parse("index.html"),
		token:  parse("token.html"),
		device: parse("device.html"),
		tools:  parse("tools.html"),
		form:   parse("form.html"),
		result: parse("result.html"),
		admin:  parse("admin.html"),
		detail: parse("detail.html"),
	}
}

func (r *templateRenderer) RenderIndexPage(w http.ResponseWriter, data IndexPageData) {
	renderTemplate(w, r.index, data)
}

func (r *templateRenderer) RenderTokenPage(w http.ResponseWriter, data TokenPageData) {
	renderTemplate(w, r.token, data)
}

func (r *templateRenderer) RenderDevicePage(w http.ResponseWriter, data DevicePageData) {
	renderTemplate(w, r.device, data)
}

func (r *templateRenderer) RenderToolsPage(w http.ResponseWriter, data ToolsPageData) {
	renderTemplate(w, r.tools, data)
}

func (r *templateRenderer) RenderFormPage(w http.ResponseWriter, data FormPageData) {
	renderTemplate(w, r.form, data)
}

func (r *templateRenderer) RenderResultPage(w http.ResponseWriter, data ResultPageData) {
	renderTemplate(w, r.result, data)
}

func (r *templateRenderer) RenderAdminPage(w http.ResponseWriter, data AdminPageData) {
	renderTemplate(w, r.admin, data)
}

func (r *templateRenderer) RenderAdminDetailPage(w http.ResponseWriter, data AdminDetailPageData) {
	renderTemplate(w, r.detail, data)
}

func renderTemplate(w http.ResponseWriter, tmpl *template.Template, data any) {
	err := tmpl.ExecuteTemplate(w, "page", data)
	if err == nil {
		return
	}

	switch err := err.(type) {
	case *template.Error:
		log.Printf("Error rendering template %s: %s", tmpl.Name(), err)
		http.Error(w, "Internal server error", http.StatusInternalServerError)
	default:
		// An error with the underlying writer (e.g. connection dropped). Ignore.
	}
}
