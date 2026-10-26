package httpclient_test

import (
	"bufio"
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/tls"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/base64"
	"encoding/pem"
	"fmt"
	"io"
	"math/big"
	"net"
	"net/http"
	"net/http/httptest"
	"net/http/httptrace"
	"os"
	"path/filepath"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/dexidp/dex/pkg/httpclient"
)

// TestMain starts a single CONNECT proxy for TestRootCAFileReloadThroughHTTPSProxy.
// HTTPS_PROXY has to be set before the first request of the process, because
// http.ProxyFromEnvironment snapshots the environment once. Every other test in
// this package talks to a loopback address, which is never proxied.
func TestMain(m *testing.M) {
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		fmt.Fprintln(os.Stderr, "failed to start test proxy:", err)
		os.Exit(1)
	}
	os.Setenv("HTTPS_PROXY", "https://"+ln.Addr().String())
	os.Unsetenv("NO_PROXY")
	os.Unsetenv("no_proxy")
	go serveCONNECTProxy(ln)
	code := m.Run()
	ln.Close()
	os.Exit(code)
}

// proxy holds the TLS certificate and the upstream address of the test proxy.
// Only the proxy test sets them.
var proxy struct {
	mu      sync.Mutex
	cert    *tls.Certificate
	backend string
}

func serveCONNECTProxy(ln net.Listener) {
	tlsLn := tls.NewListener(ln, &tls.Config{
		GetCertificate: func(*tls.ClientHelloInfo) (*tls.Certificate, error) {
			proxy.mu.Lock()
			defer proxy.mu.Unlock()
			if proxy.cert == nil {
				return nil, fmt.Errorf("test proxy is not configured")
			}
			return proxy.cert, nil
		},
	})

	for {
		conn, err := tlsLn.Accept()
		if err != nil {
			return
		}

		go func() {
			defer conn.Close()

			if _, err := http.ReadRequest(bufio.NewReader(conn)); err != nil {
				return
			}

			proxy.mu.Lock()
			backend := proxy.backend
			proxy.mu.Unlock()

			upstream, err := net.Dial("tcp", backend)
			if err != nil {
				return
			}
			defer upstream.Close()

			if _, err := io.WriteString(conn, "HTTP/1.1 200 OK\r\n\r\n"); err != nil {
				return
			}
			go io.Copy(upstream, conn)
			io.Copy(conn, upstream)
		}()
	}
}

type testCA struct {
	cert *x509.Certificate
	key  *ecdsa.PrivateKey
	pem  []byte
}

func newTestCA(t *testing.T, name string) testCA {
	t.Helper()

	key, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	require.NoError(t, err)

	template := &x509.Certificate{
		SerialNumber:          big.NewInt(time.Now().UnixNano()),
		Subject:               pkix.Name{CommonName: name},
		NotBefore:             time.Now().Add(-time.Hour),
		NotAfter:              time.Now().Add(time.Hour),
		IsCA:                  true,
		KeyUsage:              x509.KeyUsageCertSign | x509.KeyUsageDigitalSignature,
		BasicConstraintsValid: true,
	}

	der, err := x509.CreateCertificate(rand.Reader, template, template, &key.PublicKey, key)
	require.NoError(t, err)

	cert, err := x509.ParseCertificate(der)
	require.NoError(t, err)

	return testCA{
		cert: cert,
		key:  key,
		pem:  pem.EncodeToMemory(&pem.Block{Type: "CERTIFICATE", Bytes: der}),
	}
}

func (ca testCA) serverCert(t *testing.T, hosts ...string) tls.Certificate {
	t.Helper()

	key, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	require.NoError(t, err)

	template := &x509.Certificate{
		SerialNumber: big.NewInt(time.Now().UnixNano()),
		Subject:      pkix.Name{CommonName: hosts[0]},
		NotBefore:    time.Now().Add(-time.Hour),
		NotAfter:     time.Now().Add(time.Hour),
		KeyUsage:     x509.KeyUsageDigitalSignature | x509.KeyUsageKeyEncipherment,
		ExtKeyUsage:  []x509.ExtKeyUsage{x509.ExtKeyUsageServerAuth},
	}
	for _, host := range hosts {
		if ip := net.ParseIP(host); ip != nil {
			template.IPAddresses = append(template.IPAddresses, ip)
		} else {
			template.DNSNames = append(template.DNSNames, host)
		}
	}

	der, err := x509.CreateCertificate(rand.Reader, template, ca.cert, &key.PublicKey, ca.key)
	require.NoError(t, err)

	return tls.Certificate{Certificate: [][]byte{der}, PrivateKey: key}
}

// newTLSServerWithCert starts a loopback HTTPS server serving the given certificate.
func newTLSServerWithCert(t *testing.T, cert tls.Certificate) *httptest.Server {
	t.Helper()

	ts := httptest.NewUnstartedServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprint(w, "Hello, client")
	}))
	ts.TLS = &tls.Config{Certificates: []tls.Certificate{cert}}
	ts.StartTLS()
	t.Cleanup(ts.Close)

	return ts
}

// writeCAFile replaces path atomically, the way a well-behaved CA rotation does.
func writeCAFile(t *testing.T, path string, data []byte) {
	t.Helper()

	tmp := path + ".tmp"
	require.NoError(t, os.WriteFile(tmp, data, 0o600))
	require.NoError(t, os.Rename(tmp, path))
}

func get(client *http.Client, url string) error {
	res, err := client.Get(url)
	if err != nil {
		return err
	}
	defer res.Body.Close()
	_, err = io.ReadAll(res.Body)
	return err
}

func TestRootCAFileRotation(t *testing.T) {
	oldCA := newTestCA(t, "old")
	newCA := newTestCA(t, "new")

	oldServer := newTLSServerWithCert(t, oldCA.serverCert(t, "127.0.0.1"))
	newServer := newTLSServerWithCert(t, newCA.serverCert(t, "127.0.0.1"))

	caFile := filepath.Join(t.TempDir(), "ca.crt")
	writeCAFile(t, caFile, oldCA.pem)

	client, err := httpclient.NewHTTPClient([]string{caFile}, false)
	require.NoError(t, err)

	// Warm up the connection pool against the old CA.
	require.NoError(t, get(client, oldServer.URL))
	require.Error(t, get(client, newServer.URL), "new CA must not be trusted yet")

	writeCAFile(t, caFile, newCA.pem)

	assert.NoError(t, get(client, newServer.URL), "rotated CA must be picked up without a restart")
	assert.Error(t, get(client, oldServer.URL), "removed root must no longer be trusted, even with a warm connection")

	// Both roots at once, then back to the old one only.
	writeCAFile(t, caFile, append(append([]byte{}, oldCA.pem...), newCA.pem...))
	assert.NoError(t, get(client, oldServer.URL))
	assert.NoError(t, get(client, newServer.URL))

	writeCAFile(t, caFile, oldCA.pem)
	assert.NoError(t, get(client, oldServer.URL))
	assert.Error(t, get(client, newServer.URL))
}

func TestRootCAFileFailsClosed(t *testing.T) {
	ca := newTestCA(t, "ca")
	server := newTLSServerWithCert(t, ca.serverCert(t, "127.0.0.1"))

	caFile := filepath.Join(t.TempDir(), "ca.crt")
	writeCAFile(t, caFile, ca.pem)

	client, err := httpclient.NewHTTPClient([]string{caFile}, false)
	require.NoError(t, err)
	require.NoError(t, get(client, server.URL))

	for _, tc := range []struct {
		name    string
		prepare func()
	}{
		{"missing", func() { require.NoError(t, os.Remove(caFile)) }},
		{"empty", func() { writeCAFile(t, caFile, nil) }},
		{"malformed", func() { writeCAFile(t, caFile, []byte("-----BEGIN CERTIFICATE-----\nnope\n")) }},
	} {
		t.Run(tc.name, func(t *testing.T) {
			tc.prepare()
			assert.Error(t, get(client, server.URL), "request must fail instead of using the previous bundle")

			writeCAFile(t, caFile, ca.pem)
			assert.NoError(t, get(client, server.URL), "client must recover once the file is valid again")
		})
	}
}

func TestRootCAFileErrorClosesRequestBody(t *testing.T) {
	ca := newTestCA(t, "ca")
	caFile := filepath.Join(t.TempDir(), "ca.crt")
	writeCAFile(t, caFile, ca.pem)
	client, err := httpclient.NewHTTPClient([]string{caFile}, false)
	require.NoError(t, err)
	body, err := os.Open(caFile)
	require.NoError(t, err)
	t.Cleanup(func() { body.Close() })
	req, err := http.NewRequest(http.MethodPost, "https://upstream.invalid/", body)
	require.NoError(t, err)
	require.NoError(t, os.Remove(caFile))
	_, err = client.Transport.RoundTrip(req)
	require.Error(t, err)
	_, err = body.Read(make([]byte, 1))
	assert.ErrorIs(t, err, os.ErrClosed, "RoundTripper must close the body even on a CA read error")
}

func TestRootCAFileWithInsecureSkipVerify(t *testing.T) {
	ca := newTestCA(t, "unrelated")
	server := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprint(w, "ok")
	}))
	defer server.Close()
	caFile := filepath.Join(t.TempDir(), "ca.crt")
	writeCAFile(t, caFile, ca.pem)
	client, err := httpclient.NewHTTPClient([]string{caFile}, true)
	require.NoError(t, err)
	require.NoError(t, get(client, server.URL))
	other := newTestCA(t, "rotated")
	writeCAFile(t, caFile, other.pem)
	require.NoError(t, get(client, server.URL), "reload must preserve InsecureSkipVerify")
	writeCAFile(t, caFile, nil)
	assert.Error(t, get(client, server.URL), "configured CA files must still be valid")
	writeCAFile(t, caFile, ca.pem)
	assert.NoError(t, get(client, server.URL))
}

func TestRootCAFilesAllOrNothing(t *testing.T) {
	first := newTestCA(t, "first")
	second := newTestCA(t, "second")

	firstServer := newTLSServerWithCert(t, first.serverCert(t, "127.0.0.1"))
	secondServer := newTLSServerWithCert(t, second.serverCert(t, "127.0.0.1"))

	dir := t.TempDir()
	firstFile := filepath.Join(dir, "first.crt")
	secondFile := filepath.Join(dir, "second.crt")
	writeCAFile(t, firstFile, first.pem)
	writeCAFile(t, secondFile, second.pem)

	client, err := httpclient.NewHTTPClient([]string{firstFile, secondFile}, false)
	require.NoError(t, err)
	require.NoError(t, get(client, firstServer.URL))
	require.NoError(t, get(client, secondServer.URL))

	writeCAFile(t, secondFile, []byte("garbage"))

	assert.Error(t, get(client, firstServer.URL), "an invalid bundle must not be published partially")
	assert.Error(t, get(client, secondServer.URL))

	writeCAFile(t, secondFile, second.pem)
	assert.NoError(t, get(client, firstServer.URL))
	assert.NoError(t, get(client, secondServer.URL))
}

// TestRootCAFileKubernetesSymlinkSwap mimics how kubelet rotates a projected
// secret: ca.crt -> ..data/ca.crt, and ..data is an atomically swapped symlink.
func TestRootCAFileKubernetesSymlinkSwap(t *testing.T) {
	oldCA := newTestCA(t, "old")
	newCA := newTestCA(t, "new")

	oldServer := newTLSServerWithCert(t, oldCA.serverCert(t, "127.0.0.1"))
	newServer := newTLSServerWithCert(t, newCA.serverCert(t, "127.0.0.1"))

	dir := t.TempDir()
	writeVersion := func(version string, ca testCA) string {
		versionDir := filepath.Join(dir, ".."+version)
		require.NoError(t, os.Mkdir(versionDir, 0o700))
		require.NoError(t, os.WriteFile(filepath.Join(versionDir, "ca.crt"), ca.pem, 0o600))
		return ".." + version
	}

	require.NoError(t, os.Symlink(writeVersion("2026_09_22_00", oldCA), filepath.Join(dir, "..data")))
	require.NoError(t, os.Symlink(filepath.Join("..data", "ca.crt"), filepath.Join(dir, "ca.crt")))

	client, err := httpclient.NewHTTPClient([]string{filepath.Join(dir, "ca.crt")}, false)
	require.NoError(t, err)
	require.NoError(t, get(client, oldServer.URL))

	next := writeVersion("2026_09_22_01", newCA)
	require.NoError(t, os.Symlink(next, filepath.Join(dir, "..data_tmp")))
	require.NoError(t, os.Rename(filepath.Join(dir, "..data_tmp"), filepath.Join(dir, "..data")))

	assert.NoError(t, get(client, newServer.URL))
	assert.Error(t, get(client, oldServer.URL))
}

func TestRootCAFileUnchangedKeepsConnection(t *testing.T) {
	ca := newTestCA(t, "ca")
	server := newTLSServerWithCert(t, ca.serverCert(t, "127.0.0.1"))

	caFile := filepath.Join(t.TempDir(), "ca.crt")
	writeCAFile(t, caFile, ca.pem)

	client, err := httpclient.NewHTTPClient([]string{caFile}, false)
	require.NoError(t, err)

	reused := func() bool {
		var reused bool
		req, err := http.NewRequest(http.MethodGet, server.URL, nil)
		require.NoError(t, err)
		req = req.WithContext(httptrace.WithClientTrace(req.Context(), &httptrace.ClientTrace{
			GotConn: func(info httptrace.GotConnInfo) { reused = info.Reused },
		}))

		res, err := client.Do(req)
		require.NoError(t, err)
		io.Copy(io.Discard, res.Body)
		res.Body.Close()
		return reused
	}

	require.False(t, reused())
	assert.True(t, reused(), "unchanged CA file must not throw away the keepalive connection")

	// Rewriting identical content is not a change either.
	writeCAFile(t, caFile, ca.pem)
	assert.True(t, reused(), "identical content must reuse the existing transport")

	client.CloseIdleConnections()
	assert.False(t, reused(), "CloseIdleConnections must reach the underlying transport")
}

func TestRootCAFileConcurrentRotation(t *testing.T) {
	ca := newTestCA(t, "ca")
	other := newTestCA(t, "other")
	server := newTLSServerWithCert(t, ca.serverCert(t, "127.0.0.1"))

	caFile := filepath.Join(t.TempDir(), "ca.crt")
	writeCAFile(t, caFile, ca.pem)

	client, err := httpclient.NewHTTPClient([]string{caFile}, false)
	require.NoError(t, err)

	bundles := [][]byte{ca.pem, append(append([]byte{}, ca.pem...), other.pem...)}

	done := make(chan struct{})
	rotated := make(chan error, 1)
	go func() {
		// t.FailNow must not be called from this goroutine, so report instead.
		var err error
		for i := 0; err == nil; i++ {
			select {
			case <-done:
				rotated <- nil
				return
			default:
			}
			if err = os.WriteFile(caFile+".tmp", bundles[i%len(bundles)], 0o600); err == nil {
				err = os.Rename(caFile+".tmp", caFile)
			}
		}
		rotated <- err
	}()

	var requests sync.WaitGroup
	errs := make(chan error, 8*10)
	for i := 0; i < 8; i++ {
		requests.Add(1)
		go func() {
			defer requests.Done()
			for j := 0; j < 10; j++ {
				if err := get(client, server.URL); err != nil {
					errs <- err
				}
			}
		}()
	}
	requests.Wait()
	close(done)
	require.NoError(t, <-rotated)
	close(errs)

	for err := range errs {
		// Every published bundle contains the server's CA.
		assert.NoError(t, err)
	}
}

func TestRootCAFileInflightRequestCompletes(t *testing.T) {
	oldCA := newTestCA(t, "old")
	newCA := newTestCA(t, "new")

	release := make(chan struct{})
	started := make(chan struct{})
	var once sync.Once
	slow := httptest.NewUnstartedServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		once.Do(func() { close(started) })
		<-release
		fmt.Fprint(w, "Hello, client")
	}))
	slow.TLS = &tls.Config{Certificates: []tls.Certificate{oldCA.serverCert(t, "127.0.0.1")}}
	slow.StartTLS()
	defer slow.Close()

	newServer := newTLSServerWithCert(t, newCA.serverCert(t, "127.0.0.1"))

	caFile := filepath.Join(t.TempDir(), "ca.crt")
	writeCAFile(t, caFile, oldCA.pem)

	client, err := httpclient.NewHTTPClient([]string{caFile}, false)
	require.NoError(t, err)

	inflight := make(chan error, 1)
	go func() { inflight <- get(client, slow.URL) }()
	<-started

	// Rotate and force a transport swap while the first request is still open.
	writeCAFile(t, caFile, newCA.pem)
	assert.NoError(t, get(client, newServer.URL))

	close(release)
	assert.NoError(t, <-inflight, "in-flight request must survive the transport swap")
}

func TestRootCAFileReloadThroughHTTPSProxy(t *testing.T) {
	backendCA := newTestCA(t, "backend")
	proxyCA := newTestCA(t, "proxy")

	backend := newTLSServerWithCert(t, backendCA.serverCert(t, "upstream.test"))
	proxyCert := proxyCA.serverCert(t, "127.0.0.1")

	proxy.mu.Lock()
	proxy.cert = &proxyCert
	proxy.backend = backend.Listener.Addr().String()
	proxy.mu.Unlock()

	caFile := filepath.Join(t.TempDir(), "ca.crt")
	writeCAFile(t, caFile, backendCA.pem)

	client, err := httpclient.NewHTTPClient([]string{caFile}, false)
	require.NoError(t, err)

	// The proxy certificate is not trusted yet, so the CONNECT tunnel fails.
	require.Error(t, get(client, "https://upstream.test/"))

	writeCAFile(t, caFile, append(append([]byte{}, backendCA.pem...), proxyCA.pem...))

	assert.NoError(t, get(client, "https://upstream.test/"), "CONNECT proxy must use the reloaded roots")
}

func TestRootCAInlineSourcesAreNotReloaded(t *testing.T) {
	inlineCA := newTestCA(t, "inline")
	fileCA := newTestCA(t, "file")

	inlineServer := newTLSServerWithCert(t, inlineCA.serverCert(t, "127.0.0.1"))
	fileServer := newTLSServerWithCert(t, fileCA.serverCert(t, "127.0.0.1"))

	caFile := filepath.Join(t.TempDir(), "ca.crt")
	writeCAFile(t, caFile, fileCA.pem)

	client, err := httpclient.NewHTTPClient([]string{
		base64.StdEncoding.EncodeToString(inlineCA.pem),
		caFile,
	}, false)
	require.NoError(t, err)

	require.NoError(t, get(client, inlineServer.URL))
	require.NoError(t, get(client, fileServer.URL))

	// Rotating the file keeps the inline root, and never re-reads the inline value.
	other := newTestCA(t, "other")
	writeCAFile(t, caFile, other.pem)

	assert.NoError(t, get(client, inlineServer.URL), "inline roots must survive a file reload")
	assert.Error(t, get(client, fileServer.URL))
}
