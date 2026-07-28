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
	DeviceSupported bool
	PKCE            bool
	SessionCheck    time.Duration

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
}

// AdminClient is one OAuth2 client as the API reports it.
type AdminClient struct {
	ID           string
	Name         string
	RedirectURIs []string
	Public       bool
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
	Notice       string
	Error        string
	Clients      []AdminClient
	Passwords    []AdminPassword
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
