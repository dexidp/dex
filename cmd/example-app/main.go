package main

import (
	"bytes"
	"context"
	"crypto/tls"
	"crypto/x509"
	"encoding/json"
	"errors"
	"fmt"
	"io/ioutil"
	"log"
	"net"
	"net/http"
	"net/http/httputil"
	"net/url"
	"os"
	"strings"
	"time"

	"github.com/coreos/go-oidc"
	"github.com/spf13/cobra"
	"golang.org/x/oauth2"
)

const exampleAppState = "I wish to wash my irish wristwatch"

type app struct {
	clientID     string
	clientSecret string
	redirectURI  string

	verifier *oidc.IDTokenVerifier
	provider *oidc.Provider

	// Does the provider use "offline_access" scope to request a refresh token
	// or does it use "access_type=offline" (e.g. Google)?
	offlineAsScope bool

	client *http.Client
}

// return an HTTP client which trusts the provided root CAs.
func httpClientForRootCAs(rootCAs string) (*http.Client, error) {
	tlsConfig := tls.Config{RootCAs: x509.NewCertPool()}
	rootCABytes, err := ioutil.ReadFile(rootCAs)
	if err != nil {
		return nil, fmt.Errorf("failed to read root-ca: %v", err)
	}
	if !tlsConfig.RootCAs.AppendCertsFromPEM(rootCABytes) {
		return nil, fmt.Errorf("no certs found in root CA file %q", rootCAs)
	}
	return &http.Client{
		Transport: &http.Transport{
			TLSClientConfig: &tlsConfig,
			Proxy:           http.ProxyFromEnvironment,
			Dial: (&net.Dialer{
				Timeout:   30 * time.Second,
				KeepAlive: 30 * time.Second,
			}).Dial,
			TLSHandshakeTimeout:   10 * time.Second,
			ExpectContinueTimeout: 1 * time.Second,
		},
	}, nil
}

type debugTransport struct {
	t http.RoundTripper
}

func (d debugTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	reqDump, err := httputil.DumpRequest(req, true)
	if err != nil {
		return nil, err
	}
	log.Printf("%s", reqDump)

	resp, err := d.t.RoundTrip(req)
	if err != nil {
		return nil, err
	}

	respDump, err := httputil.DumpResponse(resp, true)
	if err != nil {
		resp.Body.Close()
		return nil, err
	}
	log.Printf("%s", respDump)
	return resp, nil
}

func cmd() *cobra.Command {
	var (
		a         app
		issuerURL string
		listen    string
		tlsCert   string
		tlsKey    string
		rootCAs   string
		debug     bool
	)
	c := cobra.Command{
		Use:   "example-app",
		Short: "An example OpenID Connect client",
		Long:  "",
		RunE: func(cmd *cobra.Command, args []string) error {
			if len(args) != 0 {
				return errors.New("surplus arguments provided")
			}

			u, err := url.Parse(a.redirectURI)
			if err != nil {
				return fmt.Errorf("parse redirect-uri: %v", err)
			}
			listenURL, err := url.Parse(listen)
			if err != nil {
				return fmt.Errorf("parse listen address: %v", err)
			}

			if rootCAs != "" {
				client, err := httpClientForRootCAs(rootCAs)
				if err != nil {
					return err
				}
				a.client = client
			}

			if debug {
				if a.client == nil {
					a.client = &http.Client{
						Transport: debugTransport{http.DefaultTransport},
					}
				} else {
					a.client.Transport = debugTransport{a.client.Transport}
				}
			}

			if a.client == nil {
				a.client = http.DefaultClient
			}

			// TODO(ericchiang): Retry with backoff
			ctx := oidc.ClientContext(context.Background(), a.client)
			provider, err := oidc.NewProvider(ctx, issuerURL)
			if err != nil {
				return fmt.Errorf("failed to query provider %q: %v", issuerURL, err)
			}

			var s struct {
				// What scopes does a provider support?
				//
				// See: https://openid.net/specs/openid-connect-discovery-1_0.html#ProviderMetadata
				ScopesSupported []string `json:"scopes_supported"`
			}
			if err := provider.Claims(&s); err != nil {
				return fmt.Errorf("failed to parse provider scopes_supported: %v", err)
			}

			if len(s.ScopesSupported) == 0 {
				// scopes_supported is a "RECOMMENDED" discovery claim, not a required
				// one. If missing, assume that the provider follows the spec and has
				// an "offline_access" scope.
				a.offlineAsScope = true
			} else {
				// See if scopes_supported has the "offline_access" scope.
				a.offlineAsScope = func() bool {
					for _, scope := range s.ScopesSupported {
						if scope == oidc.ScopeOfflineAccess {
							return true
						}
					}
					return false
				}()
			}

			a.provider = provider
			a.verifier = provider.Verifier(&oidc.Config{ClientID: a.clientID})

			http.HandleFunc("/", a.handleIndex)
			http.HandleFunc("/login", a.handleLogin)
			http.HandleFunc(u.Path, a.handleCallback)

			switch listenURL.Scheme {
			case "http":
				log.Printf("listening on %s", listen)
				return http.ListenAndServe(listenURL.Host, nil)
			case "https":
				log.Printf("listening on %s", listen)
				return http.ListenAndServeTLS(listenURL.Host, tlsCert, tlsKey, nil)
			default:
				return fmt.Errorf("listen address %q is not using http or https", listen)
			}
		},
	}
	c.Flags().StringVar(&a.clientID, "client-id", "example-app", "OAuth2 client ID of this application.")
	c.Flags().StringVar(&a.clientSecret, "client-secret", "ZXhhbXBsZS1hcHAtc2VjcmV0", "OAuth2 client secret of this application.")
	c.Flags().StringVar(&a.redirectURI, "redirect-uri", "http://127.0.0.1:5555/callback", "Callback URL for OAuth2 responses.")
	c.Flags().StringVar(&issuerURL, "issuer", "http://127.0.0.1:5556/dex", "URL of the OpenID Connect issuer.")
	c.Flags().StringVar(&listen, "listen", "http://127.0.0.1:5555", "HTTP(S) address to listen at.")
	c.Flags().StringVar(&tlsCert, "tls-cert", "", "X509 cert file to present when serving HTTPS.")
	c.Flags().StringVar(&tlsKey, "tls-key", "", "Private key for the HTTPS cert.")
	c.Flags().StringVar(&rootCAs, "issuer-root-ca", "", "Root certificate authorities for the issuer. Defaults to host certs.")
	c.Flags().BoolVar(&debug, "debug", false, "Print all request and responses from the OpenID Connect issuer.")
	return &c
}

func main() {
	if err := cmd().Execute(); err != nil {
		fmt.Fprintf(os.Stderr, "error: %v\n", err)
		os.Exit(2)
	}
}

func (a *app) handleIndex(w http.ResponseWriter, r *http.Request) {
	renderIndex(w)
}

func (a *app) oauth2Config(scopes []string) *oauth2.Config {
	return &oauth2.Config{
		ClientID:     a.clientID,
		ClientSecret: a.clientSecret,
		Endpoint:     a.provider.Endpoint(),
		Scopes:       scopes,
		RedirectURL:  a.redirectURI,
	}
}

func (a *app) handleLogin(w http.ResponseWriter, r *http.Request) {
	var scopes []string
	if extraScopes := r.FormValue("extra_scopes"); extraScopes != "" {
		scopes = strings.Split(extraScopes, " ")
	}
	var clients []string
	if crossClients := r.FormValue("cross_client"); crossClients != "" {
		clients = strings.Split(crossClients, " ")
	}
	for _, client := range clients {
		scopes = append(scopes, "audience:server:client_id:"+client)
	}
	connectorID := ""
	if id := r.FormValue("connector_id"); id != "" {
		connectorID = id
	}

	authCodeURL := ""
	scopes = append(scopes, "openid", "profile", "email")
	if r.FormValue("offline_access") != "yes" {
		authCodeURL = a.oauth2Config(scopes).AuthCodeURL(exampleAppState)
	} else if a.offlineAsScope {
		scopes = append(scopes, "offline_access")
		authCodeURL = a.oauth2Config(scopes).AuthCodeURL(exampleAppState)
	} else {
		authCodeURL = a.oauth2Config(scopes).AuthCodeURL(exampleAppState, oauth2.AccessTypeOffline)
	}
	if connectorID != "" {
		authCodeURL = authCodeURL + "&connector_id=" + connectorID
	}

	http.Redirect(w, r, authCodeURL, http.StatusSeeOther)
}

func (a *app) handleCallback(w http.ResponseWriter, r *http.Request) {
	var (
		err   error
		token *oauth2.Token
	)

	ctx := oidc.ClientContext(r.Context(), a.client)
	oauth2Config := a.oauth2Config(nil)
	switch r.Method {
	case http.MethodGet:
		// Authorization redirect callback from OAuth2 auth flow.
		if errMsg := r.FormValue("error"); errMsg != "" {
			http.Error(w, errMsg+": "+r.FormValue("error_description"), http.StatusBadRequest)
			return
		}
		code := r.FormValue("code")
		if code == "" {
			http.Error(w, fmt.Sprintf("no code in request: %q", r.Form), http.StatusBadRequest)
			return
		}
		if state := r.FormValue("state"); state != exampleAppState {
			http.Error(w, fmt.Sprintf("expected state %q got %q", exampleAppState, state), http.StatusBadRequest)
			return
		}
		token, err = oauth2Config.Exchange(ctx, code)
	case http.MethodPost:
		// Form request from frontend to refresh a token.
		refresh := r.FormValue("refresh_token")
		if refresh == "" {
			http.Error(w, fmt.Sprintf("no refresh_token in request: %q", r.Form), http.StatusBadRequest)
			return
		}
		t := &oauth2.Token{
			RefreshToken: refresh,
			Expiry:       time.Now().Add(-time.Hour),
		}
		token, err = oauth2Config.TokenSource(ctx, t).Token()
	default:
		http.Error(w, fmt.Sprintf("method not implemented: %s", r.Method), http.StatusBadRequest)
		return
	}

	if err != nil {
		http.Error(w, fmt.Sprintf("failed to get token: %v", err), http.StatusInternalServerError)
		return
	}

	rawIDToken, ok := token.Extra("id_token").(string)
	if !ok {
		http.Error(w, "no id_token in token response", http.StatusInternalServerError)
		return
	}

	idToken, err := a.verifier.Verify(r.Context(), rawIDToken)
	if err != nil {
		http.Error(w, fmt.Sprintf("failed to verify ID token: %v", err), http.StatusInternalServerError)
		return
	}

	accessToken, ok := token.Extra("access_token").(string)
	if !ok {
		http.Error(w, "no access_token in token response", http.StatusInternalServerError)
		return
	}

	var claims json.RawMessage
	if err := idToken.Claims(&claims); err != nil {
		http.Error(w, fmt.Sprintf("error decoding ID token claims: %v", err), http.StatusInternalServerError)
		return
	}

	buff := new(bytes.Buffer)
	if err := json.Indent(buff, []byte(claims), "", "  "); err != nil {
		http.Error(w, fmt.Sprintf("error indenting ID token claims: %v", err), http.StatusInternalServerError)
		return
	}

	renderToken(w, a.redirectURI, rawIDToken, accessToken, token.RefreshToken, buff.String())
}
