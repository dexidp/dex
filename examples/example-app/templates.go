package main

import (
	"context"
	"crypto/rsa"
	"crypto/x509"
	"embed"
	"encoding/base64"
	"encoding/json"
	"encoding/pem"
	"html/template"
	"io/fs"
	"log"
	"math/big"
	"net/http"
	"net/url"

	"github.com/coreos/go-oidc/v3/oidc"
)

//go:embed templates/*.html
var templatesFS embed.FS

//go:embed static/*
var staticFS embed.FS

const dexLogoDataURI = "/static/dex-glyph-color.svg"

var (
	indexTmpl     *template.Template
	tokenTmpl     *template.Template
	staticHandler http.Handler
)

func init() {
	var err error
	indexTmpl, err = template.ParseFS(templatesFS, "templates/index.html")
	if err != nil {
		log.Fatalf("failed to parse index template: %v", err)
	}

	tokenTmpl, err = template.ParseFS(templatesFS, "templates/token.html")
	if err != nil {
		log.Fatalf("failed to parse token template: %v", err)
	}

	// Create handler for static files
	staticSubFS, err := fs.Sub(staticFS, "static")
	if err != nil {
		log.Fatalf("failed to create static sub filesystem: %v", err)
	}
	staticHandler = http.FileServer(http.FS(staticSubFS))
}

func renderIndex(w http.ResponseWriter, data indexPageData) {
	renderTemplate(w, indexTmpl, data)
}

type indexPageData struct {
	ScopesSupported []string
	LogoURI         string
}

type tokenTmplData struct {
	IDToken            string
	IDTokenJWTLink     string
	AccessToken        string
	AccessTokenJWTLink string
	RefreshToken       string
	RedirectURL        string
	Claims             string
	PublicKeyPEM       string
}

func generateJWTIOLink(token string, provider *oidc.Provider, ctx context.Context) string {
	// JWT.io doesn't support automatic public key via URL parameter
	// The public key is displayed separately on the page for manual copy-paste
	return "https://jwt.io/#debugger-io?token=" + url.QueryEscape(token)
}

func getPublicKeyPEM(provider *oidc.Provider) string {
	if provider == nil {
		return ""
	}

	jwksURL := provider.Endpoint().AuthURL
	if len(jwksURL) > 5 {
		jwksURL = jwksURL[:len(jwksURL)-5] + "/keys"
	} else {
		return ""
	}

	resp, err := http.Get(jwksURL)
	if err != nil {
		return ""
	}
	defer resp.Body.Close()

	var jwks struct {
		Keys []json.RawMessage `json:"keys"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&jwks); err != nil || len(jwks.Keys) == 0 {
		return ""
	}

	var key struct {
		N   string `json:"n"`
		E   string `json:"e"`
		Kty string `json:"kty"`
	}
	if err := json.Unmarshal(jwks.Keys[0], &key); err != nil || key.Kty != "RSA" {
		return ""
	}

	nBytes, err1 := base64.RawURLEncoding.DecodeString(key.N)
	eBytes, err2 := base64.RawURLEncoding.DecodeString(key.E)
	if err1 != nil || err2 != nil {
		return ""
	}

	var eInt int
	for _, b := range eBytes {
		eInt = eInt<<8 | int(b)
	}

	pubKey := &rsa.PublicKey{
		N: new(big.Int).SetBytes(nBytes),
		E: eInt,
	}

	pubKeyBytes, err := x509.MarshalPKIXPublicKey(pubKey)
	if err != nil {
		return ""
	}

	pubKeyPEM := pem.EncodeToMemory(&pem.Block{
		Type:  "PUBLIC KEY",
		Bytes: pubKeyBytes,
	})

	return string(pubKeyPEM)
}

func renderToken(w http.ResponseWriter, ctx context.Context, provider *oidc.Provider, redirectURL, idToken, accessToken, refreshToken, claims string) {
	data := tokenTmplData{
		IDToken:            idToken,
		IDTokenJWTLink:     generateJWTIOLink(idToken, provider, ctx),
		AccessToken:        accessToken,
		AccessTokenJWTLink: generateJWTIOLink(accessToken, provider, ctx),
		RefreshToken:       refreshToken,
		RedirectURL:        redirectURL,
		Claims:             claims,
		PublicKeyPEM:       getPublicKeyPEM(provider),
	}
	renderTemplate(w, tokenTmpl, data)
}

func renderTemplate(w http.ResponseWriter, tmpl *template.Template, data interface{}) {
	err := tmpl.Execute(w, data)
	if err == nil {
		return
	}

	switch err := err.(type) {
	case *template.Error:
		log.Printf("Error rendering template %s: %s", tmpl.Name(), err)
		http.Error(w, "Internal server error", http.StatusInternalServerError)
	default:
		// An error with the underlying write, such as the connection being dropped. Ignore for now.
	}
}
