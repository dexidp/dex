package main

import (
	"context"
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/tls"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/pem"
	"fmt"
	"io"
	"log/slog"
	"math/big"
	"net"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"syscall"
	"testing"
	"time"
)

type noSNITLSLog chan string

func (l noSNITLSLog) Write(p []byte) (int, error) {
	select {
	case l <- string(p):
	default:
	}
	return len(p), nil
}

func (l noSNITLSLog) wait(t *testing.T, text string) {
	t.Helper()
	timer := time.NewTimer(5 * time.Second)
	defer timer.Stop()
	for {
		select {
		case line := <-l:
			if strings.Contains(line, text) {
				return
			}
		case <-timer.C:
			t.Fatalf("no log containing %q", text)
		}
	}
}

func noSNITLSPair(t *testing.T, serial int64) ([]byte, []byte) {
	t.Helper()
	key, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	if err != nil {
		t.Fatal(err)
	}
	cert := &x509.Certificate{SerialNumber: big.NewInt(serial), Subject: pkix.Name{CommonName: "localhost"}, DNSNames: []string{"localhost"}, IPAddresses: []net.IP{net.ParseIP("127.0.0.1")}, NotBefore: time.Now().Add(-time.Hour), NotAfter: time.Now().Add(time.Hour), IsCA: true, BasicConstraintsValid: true, KeyUsage: x509.KeyUsageCertSign | x509.KeyUsageDigitalSignature, ExtKeyUsage: []x509.ExtKeyUsage{x509.ExtKeyUsageServerAuth, x509.ExtKeyUsageClientAuth}}
	der, err := x509.CreateCertificate(rand.Reader, cert, cert, &key.PublicKey, key)
	if err != nil {
		t.Fatal(err)
	}
	raw, err := x509.MarshalECPrivateKey(key)
	if err != nil {
		t.Fatal(err)
	}
	return pem.EncodeToMemory(&pem.Block{Type: "CERTIFICATE", Bytes: der}), pem.EncodeToMemory(&pem.Block{Type: "EC PRIVATE KEY", Bytes: raw})
}

func noSNITLSWrite(t *testing.T, path string, b []byte) {
	t.Helper()
	if err := os.WriteFile(path, b, 0o600); err != nil {
		t.Fatal(err)
	}
}

func noSNITLSSerial(addr string, roots *x509.CertPool, name string, client *tls.Certificate) (int64, error) {
	cfg := &tls.Config{RootCAs: roots, ServerName: name, MinVersion: tls.VersionTLS12}
	if client != nil {
		cfg.Certificates = []tls.Certificate{*client}
	}
	tr := &http.Transport{TLSClientConfig: cfg, DisableKeepAlives: true}
	defer tr.CloseIdleConnections()
	c := &http.Client{Transport: tr, Timeout: time.Second}
	resp, err := c.Get("https://" + addr)
	if err != nil {
		return 0, err
	}
	defer resp.Body.Close()
	_, err = io.Copy(io.Discard, resp.Body)
	return resp.TLS.PeerCertificates[0].SerialNumber.Int64(), err
}

func noSNITLSExpect(t *testing.T, addr string, roots *x509.CertPool, name string, client *tls.Certificate, want int64) {
	t.Helper()
	deadline := time.Now().Add(5 * time.Second)
	for {
		got, err := noSNITLSSerial(addr, roots, name, client)
		if err == nil && got == want {
			return
		}
		if time.Now().After(deadline) {
			t.Fatalf("certificate: got %d, error %v; want %d", got, err, want)
		}
		time.Sleep(20 * time.Millisecond)
	}
}

func noSNITLSSignal(t *testing.T, logs noSNITLSLog) {
	t.Helper()
	if err := syscall.Kill(os.Getpid(), syscall.SIGHUP); err != nil {
		t.Fatal(err)
	}
	logs.wait(t, "reloading cert from signal")
}

func TestTLSReloadWithoutSNI(t *testing.T) {
	const env = "DEX_NOSNI_TEST_CHILD"
	if os.Getenv(env) == "" {
		for _, mode := range []string{"https"} {
			t.Run(mode, func(t *testing.T) {
				ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
				defer cancel()
				cmd := exec.CommandContext(ctx, os.Args[0], "-test.run=^TestTLSReloadWithoutSNI$", "-test.v")
				cmd.Env = append(os.Environ(), env+"="+mode)
				out, err := cmd.CombinedOutput()
				if err != nil {
					t.Fatalf("isolated TLS test: %v\n%s", err, out)
				}
			})
		}
		return
	}
	dir := t.TempDir()
	certPath := filepath.Join(dir, "tls.crt")
	keyPath := filepath.Join(dir, "tls.key")
	a, ak := noSNITLSPair(t, 1)
	b, bk := noSNITLSPair(t, 2)
	roots := x509.NewCertPool()
	roots.AppendCertsFromPEM(a)
	roots.AppendCertsFromPEM(b)
	noSNITLSWrite(t, certPath, a)
	noSNITLSWrite(t, keyPath, ak)
	logs := make(noSNITLSLog, 64)
	logger := slog.New(slog.NewTextHandler(logs, &slog.HandlerOptions{Level: slog.LevelDebug}))
	caPath := ""
	var client *tls.Certificate

	cfg, err := newTLSReloader(logger, certPath, keyPath, caPath, &tls.Config{MinVersion: tls.VersionTLS12})
	if err != nil {
		t.Fatal(err)
	}
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	srv := &http.Server{Handler: http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) { fmt.Fprint(w, "ok") }), TLSConfig: cfg, ReadHeaderTimeout: time.Second}
	defer srv.Close()
	go func() { _ = srv.ServeTLS(ln, "", "") }()
	addr := ln.Addr().String()

	for _, name := range []string{"localhost", ""} {
		noSNITLSExpect(t, addr, roots, name, client, 1)
	}
	noSNITLSWrite(t, certPath, b)
	noSNITLSWrite(t, keyPath, bk)
	noSNITLSSignal(t, logs)
	noSNITLSExpect(t, addr, roots, "localhost", client, 2)
	noSNITLSExpect(t, addr, roots, "", client, 2)
	// Forcing the callback for IP clients must not expose a failed reload.
	noSNITLSWrite(t, certPath, []byte("partial certificate"))
	noSNITLSSignal(t, logs)
	logs.wait(t, "reload TLS config")
	noSNITLSSignal(t, logs)
	logs.wait(t, "reload TLS config")
	got, err := noSNITLSSerial(addr, roots, "", client)
	if err != nil || got != 2 {
		t.Fatalf("failed reload lost IP certificate: serial=%d err=%v", got, err)
	}
	noSNITLSWrite(t, certPath, a)
	noSNITLSWrite(t, keyPath, ak)
	noSNITLSSignal(t, logs)
	noSNITLSExpect(t, addr, roots, "", client, 1)
}
