package httpclient

import (
	"bytes"
	"crypto/tls"
	"crypto/x509"
	"encoding/base64"
	"fmt"
	"net"
	"net/http"
	"os"
	"sync"
	"time"
)

// caSource is one configured rootCAs entry. Entries that could be read as a
// file when the client was built keep their path and are re-read before every
// request; inline PEM and base64 entries keep the bytes they were built with.
type caSource struct {
	path string
	data []byte
}

func extractCAs(input []string) []caSource {
	result := make([]caSource, 0, len(input))
	for _, ca := range input {
		if ca == "" {
			continue
		}

		pemData, err := os.ReadFile(ca)
		if err == nil {
			result = append(result, caSource{path: ca, data: pemData})
			continue
		}

		pemData, err = base64.StdEncoding.DecodeString(ca)
		if err != nil {
			pemData = []byte(ca)
		}

		result = append(result, caSource{data: pemData})
	}
	return result
}

// newTLSConfig builds a fresh root pool from the system roots plus every
// configured source, so roots removed from a file are gone after a reload.
func newTLSConfig(systemRoots *x509.CertPool, sources []caSource, insecureSkipVerify bool) (*tls.Config, error) {
	pool := systemRoots.Clone()

	for index, source := range sources {
		if !pool.AppendCertsFromPEM(source.data) {
			return nil, fmt.Errorf("rootCAs.%d is not in PEM format, certificate must be "+
				"a PEM encoded string, a base64 encoded bytes that contain PEM encoded string, "+
				"or a path to a PEM encoded certificate", index)
		}
	}

	return &tls.Config{RootCAs: pool, InsecureSkipVerify: insecureSkipVerify}, nil
}

// reloadingTransport re-reads file backed rootCAs before every request, so a
// rotated CA bundle is picked up without restarting Dex. It fails closed: a
// missing, empty or malformed file aborts the request instead of falling back
// to the bundle loaded earlier, and recovers as soon as the file is valid again.
type reloadingTransport struct {
	template           *http.Transport // immutable, only cloned, never used for a request
	systemRoots        *x509.CertPool  // startup roots, never mutated
	insecureSkipVerify bool

	// ponytail: a single mutex serializes the local file reads of all requests
	// on this client; split it only if profiling ever shows contention.
	mu      sync.Mutex
	sources []caSource
	current *http.Transport
}

func (t *reloadingTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	transport, err := t.transport()
	if err != nil {
		if req.Body != nil {
			req.Body.Close()
		}
		return nil, err
	}
	return transport.RoundTrip(req)
}

func (t *reloadingTransport) CloseIdleConnections() {
	t.mu.Lock()
	transport := t.current
	t.mu.Unlock()

	transport.CloseIdleConnections()
}

// transport returns the transport to use for the next request, rebuilding it
// when any file source changed on disk.
func (t *reloadingTransport) transport() (*http.Transport, error) {
	t.mu.Lock()
	defer t.mu.Unlock()

	sources := make([]caSource, len(t.sources))
	copy(sources, t.sources)

	changed := false
	for index, source := range sources {
		if source.path == "" {
			continue
		}

		pemData, err := os.ReadFile(source.path)
		if err != nil {
			return nil, fmt.Errorf("rootCAs.%d: %w", index, err)
		}

		if !bytes.Equal(pemData, source.data) {
			sources[index].data = pemData
			changed = true
		}
	}

	if !changed {
		return t.current, nil
	}

	tlsConfig, err := newTLSConfig(t.systemRoots, sources, t.insecureSkipVerify)
	if err != nil {
		return nil, err
	}

	transport := t.template.Clone()
	transport.TLSClientConfig = tlsConfig

	previous := t.current
	t.current, t.sources = transport, sources
	previous.CloseIdleConnections()

	return transport, nil
}

// NewHTTPClient creates a client with system roots and the configured root CAs.
// Entries read as files at construction are re-read before each request. Invalid
// or unreadable files fail that request; inline PEM and base64 entries stay fixed.
func NewHTTPClient(rootCAs []string, insecureSkipVerify bool) (*http.Client, error) {
	systemRoots, err := x509.SystemCertPool()
	if err != nil {
		return nil, err
	}
	sources := extractCAs(rootCAs)

	tlsConfig, err := newTLSConfig(systemRoots, sources, insecureSkipVerify)
	if err != nil {
		return nil, err
	}

	template := &http.Transport{
		Proxy: http.ProxyFromEnvironment,
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

	transport := template.Clone()
	transport.TLSClientConfig = tlsConfig

	for _, source := range sources {
		if source.path != "" {
			return &http.Client{Transport: &reloadingTransport{
				template:           template,
				systemRoots:        systemRoots,
				insecureSkipVerify: insecureSkipVerify,
				sources:            sources,
				current:            transport,
			}}, nil
		}
	}

	return &http.Client{Transport: transport}, nil
}
