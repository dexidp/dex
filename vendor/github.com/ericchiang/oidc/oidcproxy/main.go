package main

import (
	"crypto/rand"
	"encoding/gob"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"log"
	"net/http"
	"net/http/httputil"
	"net/url"
	"os"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/ericchiang/oidc"
	"github.com/gorilla/securecookie"

	"golang.org/x/net/context"
	"golang.org/x/oauth2"
)

const (
	cookieName = "oidc-proxy"
	// This header will be set by oidcproxy during authentication and
	// passed to the backend.
	emailHeaderName = "X-User-Email"
)

// Session represents a logged in user's active session.
type Session struct {
	Email   string
	Expires time.Time
}

func init() {
	gob.Register(&Session{})
}

var (
	// Flags.
	issuer    string
	backend   string
	scopes    string
	allow     string
	httpAddr  string
	httpsAddr string
	cookieExp time.Duration

	// Set up during initial configuration.
	oauth2Config   = new(oauth2.Config)
	oidcProvider   *oidc.Provider
	backendHandler *httputil.ReverseProxy
	verifier       *oidc.IDTokenVerifier

	// Regexps of emails to allow.
	allowEmail []*regexp.Regexp

	nonceSource *memNonceSource

	cookieEncrypter *securecookie.SecureCookie
)

func main() {
	flag.StringVar(&issuer, "issuer", "https://accounts.google.com", "The issuer URL of the OpenID Connect provider.")
	flag.StringVar(&backend, "backend", "", "The URL of the backened to proxy to.")
	flag.StringVar(&oauth2Config.RedirectURL, "redirect-url", "", "A full OAuth2 redirect URL.")
	flag.StringVar(&oauth2Config.ClientID, "client-id", "", "The client ID of the OAuth2 client.")
	flag.StringVar(&oauth2Config.ClientSecret, "client-secret", "", "The client secret of the OAuth2 client.")
	flag.StringVar(&scopes, "scopes", "openid,email,profile", `A comma seprated list of OAuth2 scopes to request ("openid" required).`)
	flag.StringVar(&allow, "allow-email", ".*", "Comma seperated list of email regexp's to match for access to the backend.")
	flag.StringVar(&httpAddr, "http", "127.0.0.1:5556", "Default address to listen on.")
	flag.DurationVar(&cookieExp, "cookie-exp", time.Hour*24, "Duration for which a login cookie is valid for.")
	flag.Parse()

	// Set flags from environment variables.
	flag.VisitAll(func(f *flag.Flag) {
		if f.Value.String() != f.DefValue {
			return
		}

		// Convert flag name, e.g. "redirect-url" becomes "OIDC_PROXY_REDIRECT_URL"
		envVar := "OIDC_PROXY_" + strings.ToUpper(strings.Replace(f.Name, "-", "_", -1))

		if envVal := os.Getenv(envVar); envVal != "" {
			if err := flag.Set(f.Name, envVal); err != nil {
				log.Fatal(err)
			}
		}
		// All flags are manditory.
		if f.Value.String() == "" {
			flag.Usage()
			os.Exit(2)
		}
	})

	// compile email regexps
	for _, expr := range strings.Split(allow, ",") {
		allowEmailRegexp, err := regexp.Compile(expr)
		if err != nil {
			log.Fatalf("invalid regexp: %q %v", expr, err)
		}
		allowEmail = append(allowEmail, allowEmailRegexp)
	}

	// configure reverse proxy
	backendURL, err := url.Parse(backend)
	if err != nil {
		log.Fatalf("failed to parse backend: %v", err)
	}
	backendHandler = httputil.NewSingleHostReverseProxy(backendURL)

	redirectURL, err := url.Parse(oauth2Config.RedirectURL)
	if err != nil {
		log.Fatalf("failed to parse redirect URL: %v", err)
	}

	// Query for the provider.
	oidcProvider, err = oidc.NewProvider(context.TODO(), issuer)
	if err != nil {
		log.Fatalf("failed to get provider: %v", err)
	}

	nonceSource = newNonceSource(context.TODO())
	verifier = oidcProvider.NewVerifier(context.TODO(), oidc.VerifyNonce(nonceSource))

	oauth2Config.Endpoint = oidcProvider.Endpoint()
	oauth2Config.Scopes = strings.Split(scopes, ",")

	// Initialize secure cookies.
	// TODO(ericchiang): make these configurable
	hashKey := make([]byte, 64)
	blockKey := make([]byte, 32)
	if _, err := io.ReadFull(rand.Reader, hashKey); err != nil {
		log.Fatalf("failed to initialize hash key: %v", err)
	}
	if _, err := io.ReadFull(rand.Reader, blockKey); err != nil {
		log.Fatalf("failed to initialize block key: %v", err)
	}
	cookieEncrypter = securecookie.New(hashKey, blockKey)

	mux := http.NewServeMux()
	mux.HandleFunc("/", handleProxy)
	mux.HandleFunc("/login", handleRedirect)
	mux.HandleFunc("/logout", handleLogout)
	mux.HandleFunc(redirectURL.Path, handleCallback)

	log.Printf("Listening on: %s", httpAddr)
	http.ListenAndServe(httpAddr, mux)
}

// httpRedirect returns a handler which redirects to the provided path.
func httpRedirect(path string) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Redirect(w, r, path, http.StatusFound)
	})
}

// httpError returns a handler which presents an error to the end user.
func httpError(status int, format string, a ...interface{}) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Error(w, fmt.Sprintf(format, a...), http.StatusInternalServerError)
	})
}

func handleCallback(w http.ResponseWriter, r *http.Request) {
	func() http.Handler {
		state := r.URL.Query().Get("state")
		if state == "" {
			log.Printf("State not set")
			return httpError(http.StatusInternalServerError, "Authentication failed")
		}
		if err := nonceSource.ClaimNonce(state); err != nil {
			log.Printf("Failed to claim nonce: %v", err)
			return httpError(http.StatusInternalServerError, "Authentication failed")
		}

		oauth2Token, err := oauth2Config.Exchange(context.TODO(), r.URL.Query().Get("code"))
		if err != nil {
			log.Printf("Failed to exchange token: %v", err)
			return httpError(http.StatusInternalServerError, "Authentication failed")
		}

		// Extract the ID Token from oauth2 token.
		rawIDToken, ok := oauth2Token.Extra("id_token").(string)
		if !ok {
			log.Println("No ID Token found")
			return httpError(http.StatusInternalServerError, "Authentication failed")
		}

		payload, err := verifier.Verify(rawIDToken)
		if err != nil {
			log.Printf("Failed to verify token: %v", err)
			return httpError(http.StatusInternalServerError, "Authentication failed")
		}
		var claims struct {
			Email         string `json:"email"`
			EmailVerified bool   `json:"email_verified"`
		}
		if err := json.Unmarshal(payload, &claims); err != nil {
			log.Printf("Failed to decode claims: %v", err)
			return httpError(http.StatusInternalServerError, "Authentication failed")
		}
		if !claims.EmailVerified || claims.Email == "" {
			log.Println("Failed to verify email")
			return httpError(http.StatusInternalServerError, "Authentication failed")
		}

		s := Session{Email: claims.Email, Expires: time.Now().Add(cookieExp)}
		encoded, err := cookieEncrypter.Encode(cookieName, s)
		if err != nil {
			log.Printf("Failed to encrypt session: %v", err)
			return httpError(http.StatusInternalServerError, "Authentication failed")
		}

		// Set the encoded cookie
		cookie := &http.Cookie{Name: cookieName, Value: encoded, HttpOnly: true, Path: "/"}
		http.SetCookie(w, cookie)
		return httpRedirect("/")

	}().ServeHTTP(w, r)
}

func handleRedirect(w http.ResponseWriter, r *http.Request) {
	// TODO(ericchiang): since arbitrary requests can create nonces, rate limit this endpoint.
	func() http.Handler {
		nonce, err := nonceSource.Nonce()
		if err != nil {
			log.Printf("Failed to create nonce: %v", err)
			return httpError(http.StatusInternalServerError, "Failed to generate redirect")
		}
		state, err := nonceSource.Nonce()
		if err != nil {
			log.Printf("Failed to create state: %v", err)
			return httpError(http.StatusInternalServerError, "Failed to generate redirect")
		}
		return httpRedirect(oauth2Config.AuthCodeURL(state, oauth2.ApprovalForce, oidc.Nonce(nonce)))
	}().ServeHTTP(w, r)
}

func handleLogout(w http.ResponseWriter, r *http.Request) {
	cookie := &http.Cookie{Name: cookieName, Value: "", HttpOnly: true, Path: "/"}
	http.SetCookie(w, cookie)
	httpRedirect("/login").ServeHTTP(w, r)
}

func handleProxy(w http.ResponseWriter, r *http.Request) {
	func() http.Handler {
		cookie, err := r.Cookie(cookieName)
		if err != nil {
			// Only error can be ErrNoCookie https://goo.gl/o5fZ49
			return httpRedirect("/login")
		}
		var s Session
		if err := cookieEncrypter.Decode(cookieName, cookie.Value, &s); err != nil {
			log.Printf("Failed to decode cookie: %v", err)
			return http.HandlerFunc(handleLogout) // clear the cookie
		}
		if time.Now().After(s.Expires) {
			log.Printf("Cookie for %q expired", s.Email)
			return http.HandlerFunc(handleLogout) // clear the cookie
		}

		for _, allow := range allowEmail {
			if allow.MatchString(s.Email) {
				r.Header.Set(emailHeaderName, s.Email)
				return backendHandler
			}
		}
		log.Printf("Denying %q", s.Email)
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			resp := []byte(`<html><head></head><body>Provided email does not have permission to login. <a href="/logout">Try a different account.</a></body></html>`)
			w.Header().Set("Content-Type", "text/html")
			w.Header().Set("Content-Length", strconv.Itoa(len(resp)))
			w.WriteHeader(http.StatusForbidden)
			w.Write(resp)
		})
	}().ServeHTTP(w, r)
}
