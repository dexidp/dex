package main

//go:generate go-bindata -pkg main -o assets.go data/
//go:generate gofmt -w assets.go

import (
	"bytes"
	"crypto/tls"
	"crypto/x509"
	"encoding/json"
	"flag"
	"fmt"
	"html/template"
	"io/ioutil"
	"net"
	"net/http"
	"net/url"
	"os"
	"strings"
	"time"

	"github.com/coreos/go-oidc/jose"
	"github.com/coreos/go-oidc/oauth2"
	"github.com/coreos/go-oidc/oidc"

	"github.com/coreos/dex/client"
	pflag "github.com/coreos/dex/pkg/flag"
	phttp "github.com/coreos/dex/pkg/http"
	"github.com/coreos/dex/pkg/log"
	"github.com/coreos/dex/scope"
)

var indexTemplate *template.Template

func main() {
	fs := flag.NewFlagSet("oidc-app", flag.ExitOnError)
	listen := fs.String("listen", "http://127.0.0.1:5555", "")
	redirectURL := fs.String("redirect-url", "http://127.0.0.1:5555/callback", "")
	clientID := fs.String("client-id", "example-app", "")
	clientSecret := fs.String("client-secret", "ZXhhbXBsZS1hcHAtc2VjcmV0", "")
	caFile := fs.String("trusted-ca-file", "", "the TLS CA file, if empty then the host's root CA will be used")

	certFile := fs.String("tls-cert-file", "", "the TLS cert file. If empty, the app will listen on HTTP")
	keyFile := fs.String("tls-key-file", "", "the TLS key file. If empty, the app will listen on HTTP")

	discovery := fs.String("discovery", "http://127.0.0.1:5556", "")
	logDebug := fs.Bool("log-debug", false, "log debug-level information")
	logTimestamps := fs.Bool("log-timestamps", false, "prefix log lines with timestamps")

	if err := fs.Parse(os.Args[1:]); err != nil {
		fmt.Fprintln(os.Stderr, err.Error())
		os.Exit(1)
	}

	if err := pflag.SetFlagsFromEnv(fs, "EXAMPLE_APP"); err != nil {
		fmt.Fprintln(os.Stderr, err.Error())
		os.Exit(1)
	}

	if *logDebug {
		log.EnableDebug()
	}
	if *logTimestamps {
		log.EnableTimestamps()
	}

	if *clientID == "" {
		log.Fatal("--client-id must be set")
	}

	if *clientSecret == "" {
		log.Fatal("--client-secret must be set")
	}

	l, err := url.Parse(*listen)
	if err != nil {
		log.Fatalf("Unable to use --listen flag: %v", err)
	}

	_, p, err := net.SplitHostPort(l.Host)
	if err != nil {
		log.Fatalf("Unable to parse host from --listen flag: %v", err)
	}

	redirectURLParsed, err := url.Parse(*redirectURL)
	if err != nil {
		log.Fatalf("Unable to parse url from --redirect-url flag: %v", err)
	}

	useTLS := *keyFile != "" && *certFile != ""
	if useTLS && (redirectURLParsed.Scheme != "https" || l.Scheme != "https") {
		log.Fatalf(`TLS Cert File and Key File were provided. Ensure listen and redirect URLs are using the "https://" scheme.`)
	}

	cc := oidc.ClientCredentials{
		ID:     *clientID,
		Secret: *clientSecret,
	}

	var tlsConfig tls.Config
	if *caFile != "" {
		roots := x509.NewCertPool()
		pemBlock, err := ioutil.ReadFile(*caFile)
		if err != nil {
			log.Fatalf("Unable to read ca file: %v", err)
		}
		roots.AppendCertsFromPEM(pemBlock)
		tlsConfig.RootCAs = roots
	}

	httpClient := &http.Client{Transport: &http.Transport{TLSClientConfig: &tlsConfig}}

	var cfg oidc.ProviderConfig
	for {
		cfg, err = oidc.FetchProviderConfig(httpClient, *discovery)
		if err == nil {
			break
		}

		sleep := 3 * time.Second
		log.Errorf("Failed fetching provider config, trying again in %v: %v", sleep, err)
		time.Sleep(sleep)
	}

	log.Infof("Fetched provider config from %s: %#v", *discovery, cfg)

	ccfg := oidc.ClientConfig{
		HTTPClient:     httpClient,
		ProviderConfig: cfg,
		Credentials:    cc,
		RedirectURL:    *redirectURL,
		Scope:          append(oidc.DefaultScope, "offline_access"),
	}

	client, err := oidc.NewClient(ccfg)
	if err != nil {
		log.Fatalf("Unable to create Client: %v", err)
	}

	client.SyncProviderConfig(*discovery)

	hdlr := NewClientHandler(client, *discovery, *redirectURLParsed)
	httpsrv := &http.Server{
		Addr:    fmt.Sprintf(":%s", p),
		Handler: hdlr,
	}

	indexBytes, err := Asset("data/index.html")
	if err != nil {
		log.Fatalf("could not load template: %q", err)
	}

	indexTemplate = template.Must(template.New("root").Parse(string(indexBytes)))

	log.Infof("Binding to %s...", httpsrv.Addr)
	if useTLS {
		log.Info("Key and cert file provided. Using TLS")
		log.Fatal(httpsrv.ListenAndServeTLS(*certFile, *keyFile))
	} else {
		log.Fatal(httpsrv.ListenAndServe())
	}
}

func NewClientHandler(c *oidc.Client, issuer string, cbURL url.URL) http.Handler {
	mux := http.NewServeMux()

	oob := cbURL.String() == client.OOBRedirectURI

	issuerURL, err := url.Parse(issuer)
	if err != nil {
		log.Fatalf("Could not parse issuer url: %v", err)
	}

	mux.HandleFunc("/", handleIndexFunc(oob))
	mux.HandleFunc("/login", handleLoginFunc(c))
	mux.HandleFunc("/register", handleRegisterFunc(c))
	if cbURL.String() != client.OOBRedirectURI {
		mux.HandleFunc(cbURL.Path, handleCallbackFunc(c))
	} else {
		mux.HandleFunc("/callback", handleCallbackFunc(c))
	}

	resendURL := *issuerURL
	resendURL.Path = "/resend-verify-email"

	mux.HandleFunc("/resend", handleResendFunc(c, *issuerURL, resendURL, cbURL))
	return mux
}

func handleIndexFunc(oob bool) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		err := indexTemplate.Execute(w, map[string]interface{}{
			"OOB": oob,
		})
		if err != nil {
			phttp.WriteError(w, http.StatusInternalServerError,
				fmt.Sprintf("unable to execute template: %v", err))

		}
	}
}

func handleLoginFunc(c *oidc.Client) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		err := r.ParseForm()
		if err != nil {
			phttp.WriteError(w, http.StatusBadRequest,
				fmt.Sprintf("Could not parse request: %v", err))
		}

		oac, err := c.OAuthClient()
		if err != nil {
			panic("unable to proceed")
		}

		u, err := url.Parse(oac.AuthCodeURL("", "", ""))
		if err != nil {
			panic("unable to proceed")
		}

		var scopes []string
		q := u.Query()
		if scope := q.Get("scope"); scope != "" {
			scopes = strings.Split(scope, " ")
		}

		if xClient := r.Form.Get("cross_client"); xClient != "" {
			xClients := strings.Split(xClient, ",")
			for _, x := range xClients {
				scopes = append(scopes, scope.ScopeGoogleCrossClient+x)
			}
		}

		if extraScopes := r.Form.Get("extra_scopes"); extraScopes != "" {
			scopes = append(scopes, strings.Split(extraScopes, ",")...)
		}

		if scopes != nil {
			q.Set("scope", strings.Join(scopes, " "))
			u.RawQuery = q.Encode()
		}

		http.Redirect(w, r, u.String(), http.StatusFound)
	}
}

func handleRegisterFunc(c *oidc.Client) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		oac, err := c.OAuthClient()
		if err != nil {
			panic("unable to proceed")
		}

		u, err := url.Parse(oac.AuthCodeURL("", "", ""))
		q := u.Query()
		q.Set("register", "1")
		if err != nil {
			panic("unable to proceed")
		}
		u.RawQuery = q.Encode()
		log.Infof("URL: %v", u.String())
		http.Redirect(w, r, u.String(), http.StatusFound)
	}
}

func handleResendFunc(c *oidc.Client, issuerURL, resendURL, cbURL url.URL) http.HandlerFunc {
	trans := &oidc.AuthenticatedTransport{
		TokenRefresher: &oidc.ClientCredsTokenRefresher{
			Issuer:     issuerURL.String(),
			OIDCClient: c,
		},
		RoundTripper: http.DefaultTransport,
	}
	hc := &http.Client{Transport: trans}

	return func(w http.ResponseWriter, r *http.Request) {
		err := r.ParseForm()
		if err != nil {
			panic("unable to proceed")
		}

		tok := r.Form.Get("jwt")
		q := struct {
			Token       string `json:"token"`
			RedirectURI string `json:"redirectURI"`
		}{
			Token:       tok,
			RedirectURI: cbURL.String(),
		}
		qBytes, err := json.Marshal(&q)
		res, err := hc.Post(resendURL.String(), "application/json", bytes.NewReader(qBytes))
		if err != nil {
			log.Fatalf("error requesting email resend:", err)
		}

		w.Write([]byte(fmt.Sprintf("Status from Dex: %v", res.Status)))
	}
}

func handleCallbackFunc(c *oidc.Client) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		refreshToken := r.URL.Query().Get("refresh_token")
		code := r.URL.Query().Get("code")

		oac, err := c.OAuthClient()
		if err != nil {
			phttp.WriteError(w, http.StatusBadRequest, fmt.Sprintf("unable to create OAuth2 client: %v", err))
			return
		}

		var token oauth2.TokenResponse

		switch {
		case code != "":
			if token, err = oac.RequestToken(oauth2.GrantTypeAuthCode, code); err != nil {
				phttp.WriteError(w, http.StatusBadRequest, fmt.Sprintf("unable to verify auth code with issuer: %v", err))
				return
			}
		case refreshToken != "":
			if token, err = oac.RequestToken(oauth2.GrantTypeRefreshToken, refreshToken); err != nil {
				phttp.WriteError(w, http.StatusBadRequest, fmt.Sprintf("unable to refresh token: %v", err))
				return
			}
			if token.RefreshToken == "" {
				token.RefreshToken = refreshToken
			}
		default:
			phttp.WriteError(w, http.StatusBadRequest, "code query param must be set")
			return
		}

		tok, err := jose.ParseJWT(token.IDToken)
		if err != nil {
			phttp.WriteError(w, http.StatusBadRequest, fmt.Sprintf("unable to parse JWT: %v", err))
			return
		}

		claims := new(bytes.Buffer)
		if err := json.Indent(claims, tok.Payload, "", "  "); err != nil {
			phttp.WriteError(w, http.StatusBadRequest, fmt.Sprintf("unable to construct claims: %v", err))
			return
		}
		s := fmt.Sprintf(`
<html>
  <head>
    <style>
/* make pre wrap */
pre {
 white-space: pre-wrap;       /* css-3 */
 white-space: -moz-pre-wrap;  /* Mozilla, since 1999 */
 white-space: -pre-wrap;      /* Opera 4-6 */
 white-space: -o-pre-wrap;    /* Opera 7 */
 word-wrap: break-word;       /* Internet Explorer 5.5+ */
} 
    </style>
  </head>
  <body>
    <p> Token: <pre><code>%v</code></pre></p>
    <p> Claims: <pre><code>%v</code></pre></p>
    <p> Refresh Token: <pre><code>%v</code></pre></p>
    <p><a href="%s?refresh_token=%s">Redeem refresh token</a><p>
    <p><a href="/resend?jwt=%s">Resend Verification Email</a></p>
  </body>
</html>`, tok.Encode(), claims.String(), token.RefreshToken, r.URL.Path, token.RefreshToken, tok.Encode())
		w.Write([]byte(s))
	}
}
