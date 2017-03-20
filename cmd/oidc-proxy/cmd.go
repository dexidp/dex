package main

import (
	"crypto/tls"
	"crypto/x509"
	"encoding/base64"
	"fmt"
	"io/ioutil"
	"net"
	"net/http"
	"os"
	"runtime"
	"strings"

	"github.com/Sirupsen/logrus"
	"github.com/spf13/cobra"
	"github.com/spf13/pflag"
)

const version = "v0.1.0"

func main() {
	cmd := &cobra.Command{
		Use:          "oidc-proxy",
		Short:        "An authentication proxy using OpenID Connect.",
		SilenceUsage: true,
	}

	var (
		allowedDomains string
		allowedEmails  string
		c              config
		scopes         string
		tlsKey         string
		tlsCert        string
		backendCA      string
		issuerCA       string
		listenHTTP     string
		listenHTTPS    string
		logLevel       string
		sessionSecret  string
		version        bool
	)
	l := &logrus.Logger{
		Out:       os.Stderr,
		Level:     logrus.DebugLevel,
		Formatter: &logrus.TextFormatter{DisableColors: true},
	}

	fs := cmd.Flags()
	fs.StringVar(&c.issuerURL, "issuer-url", "",
		"Canonical URL of the OpenID Connect provider. Example: 'https://account.google.com'.")
	fs.StringVar(&issuerCA, "issuer-ca", "",
		"Path to a PEM encoded root CA for the issuer.")
	fs.StringVar(&c.clientID, "client-id", "",
		"OAuth2 client ID.")
	fs.StringVar(&c.clientSecret, "client-secret", "",
		"OAuth2 client secret.")
	fs.StringVar(&scopes, "scopes", "openid,profile,email",
		"Comma separated list of scopes to request from the OpenID Connect provider.")
	fs.StringVar(&c.redirectURI, "redirect-uri", "",
		"OAuth2 redirect URI of the proxy.")

	fs.StringVar(&c.backendURL, "backend-url", "",
		"URL to proxy.")
	fs.StringVar(&backendCA, "backend-ca", "",
		"Path of a PEM encoded root CA for the backend.")

	fs.StringVar(&listenHTTP, "listen-http", "localhost:11080",
		"Address to listen for HTTP request. If listen-https is provided, this will redirect to that port instead.")
	fs.StringVar(&listenHTTPS, "listen-https", "",
		"Address to listen for HTTPS requests.")
	fs.StringVar(&tlsCert, "tls-cert", "",
		"Path of a PEM encoded TLS cert to listen with.")
	fs.StringVar(&tlsKey, "tls-key", "",
		"Path of a PEM encoded TLS key to listen with.")

	fs.StringVar(&sessionSecret, "session-secret", "",
		"32 random bytes for encrypting HTTP cookies, base64 encoded. If not provided, a random secret will be generated.")

	fs.BoolVar(&c.websocketSupport, "proxy-websockets", true,
		"Support proxying websocket requests.")

	fs.StringVar(&allowedEmails, "allow-emails", "",
		"Comma separated list of emails allowed to access the backend.")
	fs.StringVar(&allowedDomains, "allow-domains", "",
		"Comma separated list of email domains allowed to access the backend.")

	fs.StringVar(&logLevel, "log-level", "info",
		"Logging level. Valid values are 'error', 'info', and 'debug'. The 'debug' level potentially prints sensitive information to the log.")
	fs.BoolVar(&version, "version", false,
		"Print the version of the binary to stdout and exit.")

	cmd.Run = func(cmd *cobra.Command, args []string) {
		// If the user hasn't provided any arguments, print the usage and exit.
		changed := false
		cmd.Flags().VisitAll(func(f *pflag.Flag) {
			if f.Changed {
				changed = true
			}
		})
		if !changed {
			cmd.Usage()
			os.Exit(1)
		}

		if len(args) != 0 {
			l.Fatalf("surplus arguments provided")
		}
		if version {
			printVersion()
			os.Exit(0)
		}

		level, err := parseLogLevel(logLevel)
		if err != nil {
			l.Fatalf("failed to parse log level: %v", err)
		}
		l.Level = level
		if level == logrus.DebugLevel {
			l.Warn("a debug log level will print sensitive information to the log and is not recommended outside of debugging")
		}

		c.logger = l

		if len(allowedEmails) != 0 {
			a := allowEmailWhitelist(strings.Split(allowedEmails, ",")...)
			c.authorizers = append(c.authorizers, a)
		}
		if len(allowedDomains) != 0 {
			a := allowEmailDomains(strings.Split(allowedDomains, ",")...)
			c.authorizers = append(c.authorizers, a)
		}
		c.scopes = strings.Split(scopes, ",")

		if c.sessionSecret, err = parseSessionSecret(sessionSecret); err != nil {
			l.Fatalf("prasing session-secret: %v", err)
		}

		if c.backendTLSConfig, err = loadTLSConfig(backendCA); err != nil {
			l.Fatalf("failed to load backend CA: %v", err)
		}
		if c.issuerTLSConfig, err = loadTLSConfig(issuerCA); err != nil {
			l.Fatalf("failed to load issuer CA: %v", err)
		}

		// Allow insecure cookies if the redirectURI is insecure.
		c.insecureAllowHTTPCookies = !strings.HasPrefix(c.redirectURI, "https://")
		if c.insecureAllowHTTPCookies {
			l.Warnf("allowing insecure HTTP cookies because the redirect-uri is insecure (%s)", c.redirectURI)
		}

		server, err := newServer(&c)
		if err != nil {
			l.Fatal(err)
		}

		errc := make(chan error, 2)

		if len(listenHTTPS) != 0 {
			if len(listenHTTP) != 0 {
				httpsPort := ""
				if _, port, err := net.SplitHostPort(listenHTTPS); err == nil {
					httpsPort = port
				}
				l.Infof("listening for HTTP request at %s", listenHTTP)
				go func() {
					errc <- http.ListenAndServe(listenHTTP, redirectToHTTPS(l, httpsPort))
				}()
			}
			l.Infof("listening for HTTPS requests at %s", listenHTTPS)
			go func() {
				errc <- http.ListenAndServeTLS(listenHTTPS, tlsCert, tlsKey, server)
			}()
		} else {
			l.Infof("listening for HTTP request at %s", listenHTTP)
			go func() {
				errc <- http.ListenAndServe(listenHTTP, server)
			}()
		}

		l.Fatal(<-errc)
	}

	if err := cmd.Execute(); err != nil {
		l.Fatal(err)
		os.Exit(2)
	}
}

func loadTLSConfig(caPath string) (*tls.Config, error) {
	if len(caPath) == 0 {
		return nil, nil
	}
	caData, err := ioutil.ReadFile(caPath)
	if err != nil {
		return nil, fmt.Errorf("read ca file: %v", err)
	}
	pool := x509.NewCertPool()
	if !pool.AppendCertsFromPEM(caData) {
		return nil, fmt.Errorf("ca file containe no certificates: %s", caPath)
	}
	return &tls.Config{RootCAs: pool}, nil
}

func redirectToHTTPS(l logger, httpsPort string) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		host := r.Host
		if h, _, err := net.SplitHostPort(host); err == nil {
			host = h
		}
		if len(httpsPort) != 0 {
			host = net.JoinHostPort(host, httpsPort)
		}
		target := "https://" + host + r.URL.Path
		l.Infof("redirecting HTTP request to: %s", target)

		// Don't log URL query.
		if len(r.URL.RawQuery) != 0 {
			target += "?" + r.URL.RawQuery
		}
		http.Redirect(w, r, target, http.StatusTemporaryRedirect)
	}
}

func parseLogLevel(s string) (logrus.Level, error) {
	switch s {
	case "error":
		return logrus.ErrorLevel, nil
	case "info":
		return logrus.InfoLevel, nil
	case "debug":
		return logrus.DebugLevel, nil
	default:
		return logrus.InfoLevel, fmt.Errorf("unrecognized log level: %s", s)
	}
}

func parseSessionSecret(s string) (*[32]byte, error) {
	if len(s) == 0 {
		return nil, nil
	}
	b, err := base64.StdEncoding.DecodeString(s)
	if err != nil {
		return nil, fmt.Errorf("session secret is not base64 encoded: %v", err)
	}
	if n := len(b); n != 32 {
		return nil, fmt.Errorf("session secret must be 32 bytes, got secret of length %d", n)
	}
	var secret [32]byte
	copy(secret[:], b)
	return &secret, nil
}

func printVersion() {
	fmt.Fprintf(os.Stdout, `oidc-proxy version: %s
Go version: %s
Go OS/ARCH: %s/%s
`, version, runtime.Version(), runtime.GOOS, runtime.GOARCH)
}
