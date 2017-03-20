package main

import (
	"crypto/tls"
	"io"
	"net"
	"net/http"
	"net/http/httputil"
	"net/url"
	"strings"
	"time"

	"github.com/gorilla/websocket"
	"golang.org/x/net/http2"
)

type proxyConfig struct {
	// Should the proxy support proxying websocket requests?
	enableWS bool
	// TLS config for connecting to the backend service.
	tlsConfig *tls.Config

	logger logger
}

// A single host reverse HTTP proxy with optional support for websocket requests.
type proxy struct {
	httpProxy *httputil.ReverseProxy
	wsProxy   *wsProxy // nil if unsupported
	logger    logger
}

func newProxy(target *url.URL, cfg *proxyConfig) *proxy {
	p := &proxy{logger: cfg.logger}
	p.httpProxy = httputil.NewSingleHostReverseProxy(target)
	p.httpProxy.Transport = newTransport(cfg.tlsConfig)

	if cfg.enableWS {
		p.wsProxy = &wsProxy{
			upstreamURL: target,
			upgrader: &websocket.Upgrader{
				Error: func(w http.ResponseWriter, r *http.Request, status int, reason error) {
					cfg.logger.Errorf("websocket upgrade failed: %v", reason)
					http.Error(w, reason.Error(), status)
				},
			},
			dialer: &websocket.Dialer{
				Proxy:           http.ProxyFromEnvironment,
				TLSClientConfig: cfg.tlsConfig,
			},
		}
	}
	return p
}

func (p *proxy) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if p.wsProxy != nil && isWSRequest(r) {
		p.logger.Debugf("got a websocket request")
		p.wsProxy.ServeHTTP(w, r)
		return
	}
	p.httpProxy.ServeHTTP(w, r)
}

// wsProxy is a single host proxy for HTTP websocket requests.
type wsProxy struct {
	upstreamURL *url.URL

	upgrader *websocket.Upgrader
	dialer   *websocket.Dialer
}

func (p *wsProxy) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	// Handle incoming requests.
	clientConn, err := p.upgrader.Upgrade(w, r, http.Header{})
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	defer clientConn.Close()

	// Shallow copy. TODO: remove headers?
	u := new(url.URL)
	*u = *r.URL

	u.Scheme = p.upstreamURL.Scheme
	u.Host = p.upstreamURL.Host
	u.Path = singleJoiningSlash(p.upstreamURL.Path, r.URL.Path)

	// Dial the backend.
	upstreamConn, _, err := p.dialer.Dial(u.String(), nil)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadGateway)
		return
	}
	defer upstreamConn.Close()

	cp := func(dest, src *websocket.Conn) error {
		for {
			msgType, r, err := src.NextReader()
			if err != nil {
				return err
			}
			wc, err := dest.NextWriter(msgType)
			if err != nil {
				return err
			}
			_, err = io.Copy(wc, r)
			wc.Close()
			if err != nil {
				return err
			}
		}
	}

	// Stream requests in both directions until one fails.
	errc := make(chan error, 2)
	go func() { errc <- cp(clientConn, upstreamConn) }()
	go func() { errc <- cp(upstreamConn, clientConn) }()
	<-errc
	return
}

func isWSRequest(r *http.Request) bool {
	if !strings.EqualFold(r.Header.Get("Upgrade"), "websocket") {
		return false
	}

	// The "Connection:" header can sometimes hold multiple comma
	// separated values.
	for _, v := range strings.Split(r.Header.Get("Connection"), ",") {
		if strings.EqualFold(v, "upgrade") {
			return true
		}
	}
	return false
}

func newTransport(tlsConfig *tls.Config) http.RoundTripper {
	t := &http.Transport{
		Proxy:           http.ProxyFromEnvironment,
		TLSClientConfig: tlsConfig,
		DialContext: (&net.Dialer{
			Timeout:   30 * time.Second,
			KeepAlive: 30 * time.Second,
			DualStack: true,
		}).DialContext,
		MaxIdleConns:          100,
		IdleConnTimeout:       90 * time.Second,
		TLSHandshakeTimeout:   10 * time.Second,
		ExpectContinueTimeout: 1 * time.Second,
	}

	// Must explicitly configure this since we provide a custom
	// TLS config.
	http2.ConfigureTransport(t)
	return t
}

// Copied from net/http/httputil.
func singleJoiningSlash(a, b string) string {
	aslash := strings.HasSuffix(a, "/")
	bslash := strings.HasPrefix(b, "/")
	switch {
	case aslash && bslash:
		return a + b[1:]
	case !aslash && !bslash:
		return a + "/" + b
	}
	return a + b
}
