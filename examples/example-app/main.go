package main

import (
	"context"
	"errors"
	"fmt"
	"log"
	"net/http"
	"net/url"
	"os"
	"sync"

	"github.com/coreos/go-oidc/v3/oidc"
	"github.com/spf13/cobra"
	"golang.org/x/oauth2"
)

const exampleAppState = "I wish to wash my irish wristwatch"

var (
	codeVerifier  string
	codeChallenge string
)

func init() {
	codeVerifier = oauth2.GenerateVerifier()
	codeChallenge = oauth2.S256ChallengeFromVerifier(codeVerifier)
}

type app struct {
	clientID     string
	clientSecret string
	pkce         bool
	redirectURI  string

	verifier        *oidc.IDTokenVerifier
	provider        *oidc.Provider
	scopesSupported []string

	// Does the provider use "offline_access" scope to request a refresh token
	// or does it use "access_type=offline" (e.g. Google)?
	offlineAsScope bool

	client *http.Client

	// Device flow state
	// Only one session is possible at a time
	// Since it is an example, we don't bother locking', this is a simplicity tradeoff
	deviceFlowMutex sync.Mutex
	deviceFlowData  struct {
		sessionID       string // Unique ID for current flow session
		deviceCode      string
		userCode        string
		verificationURI string
		pollInterval    int
		token           *oauth2.Token
	}
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
			a.scopesSupported = s.ScopesSupported

			http.Handle("/static/", http.StripPrefix("/static/", staticHandler))
			http.HandleFunc("/", a.handleIndex)
			http.HandleFunc("/login", a.handleLogin)
			http.HandleFunc("/device/login", a.handleDeviceLogin)
			http.HandleFunc("/device", a.handleDevicePage)
			http.HandleFunc("/device/poll", a.handleDevicePoll)
			http.HandleFunc("/device/result", a.handleDeviceResult)
			http.HandleFunc("/userinfo", a.handleUserInfo)
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
	c.Flags().BoolVar(&a.pkce, "pkce", true, "Use PKCE flow for the code exchange.")
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
