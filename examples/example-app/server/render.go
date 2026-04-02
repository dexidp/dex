package server

import (
	"embed"
	"html/template"
	"io/fs"
	"log"
	"net/http"

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

// LoginPageData holds data for the login page template.
type LoginPageData struct {
	ScopesSupported []string
	LogoURI         string
	User            *session.UserClaims
	NotLoggedIn     bool
	LogoutURL       string
}

// TokenPageData holds data for the token display template.
type TokenPageData struct {
	IDToken            string
	IDTokenJWTLink     string
	AccessToken        string
	AccessTokenJWTLink string
	RefreshToken       string
	RedirectURL        string
	Claims             string
	PublicKeyPEM       string
}

// DevicePageData holds data for the device flow template.
type DevicePageData struct {
	SessionID       string
	DeviceCode      string
	UserCode        string
	VerificationURI string
	PollInterval    int
	LogoURI         string
}

// Renderer renders HTML pages for the application.
type Renderer interface {
	RenderLoginPage(w http.ResponseWriter, data LoginPageData)
	RenderTokenPage(w http.ResponseWriter, data TokenPageData)
	RenderDevicePage(w http.ResponseWriter, data DevicePageData)
}

// templateRenderer implements Renderer using Go html/template.
type templateRenderer struct {
	index  *template.Template
	token  *template.Template
	device *template.Template
}

// newTemplateRenderer parses embedded templates and returns a Renderer.
func newTemplateRenderer() Renderer {
	parse := func(name string) *template.Template {
		t, err := template.ParseFS(templatesFS, name)
		if err != nil {
			log.Fatalf("failed to parse template %s: %v", name, err)
		}
		return t
	}

	return &templateRenderer{
		index:  parse("templates/index.html"),
		token:  parse("templates/token.html"),
		device: parse("templates/device.html"),
	}
}

func (r *templateRenderer) RenderLoginPage(w http.ResponseWriter, data LoginPageData) {
	renderTemplate(w, r.index, data)
}

func (r *templateRenderer) RenderTokenPage(w http.ResponseWriter, data TokenPageData) {
	renderTemplate(w, r.token, data)
}

func (r *templateRenderer) RenderDevicePage(w http.ResponseWriter, data DevicePageData) {
	renderTemplate(w, r.device, data)
}

func renderTemplate(w http.ResponseWriter, tmpl *template.Template, data any) {
	err := tmpl.Execute(w, data)
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
