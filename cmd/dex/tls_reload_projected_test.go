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
	"runtime"
	"strings"
	"syscall"
	"testing"
	"time"
)

type projectedTLSLog chan string

func (l projectedTLSLog) Write(p []byte) (int, error) {
	select {
	case l <- string(p):
	default:
	}
	return len(p), nil
}

func (l projectedTLSLog) wait(t *testing.T, text string) {
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

func projectedTLSPair(t *testing.T, serial int64, key *ecdsa.PrivateKey) ([]byte, []byte) {
	t.Helper()
	if key == nil {
		var err error
		key, err = ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
		if err != nil {
			t.Fatal(err)
		}
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

func projectedTLSWrite(t *testing.T, path string, b []byte) {
	t.Helper()
	if err := os.WriteFile(path, b, 0o600); err != nil {
		t.Fatal(err)
	}
}

func projectedTLSSerial(addr string, roots *x509.CertPool, name string, client *tls.Certificate) (int64, error) {
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

func projectedTLSExpect(t *testing.T, addr string, roots *x509.CertPool, name string, client *tls.Certificate, want int64) {
	t.Helper()
	deadline := time.Now().Add(5 * time.Second)
	for {
		got, err := projectedTLSSerial(addr, roots, name, client)
		if err == nil && got == want {
			return
		}
		if time.Now().After(deadline) {
			t.Fatalf("certificate: got %d, error %v; want %d", got, err, want)
		}
		time.Sleep(20 * time.Millisecond)
	}
}

func projectedTLSSignal(t *testing.T, logs projectedTLSLog) {
	t.Helper()
	if err := syscall.Kill(os.Getpid(), syscall.SIGHUP); err != nil {
		t.Fatal(err)
	}
	logs.wait(t, "reloading cert from signal")
}

func TestTLSReloadProjectedSecret(t *testing.T) {
	const env = "DEX_PROJECTED_TEST_CHILD"
	if os.Getenv(env) == "" {
		for _, mode := range []string{"projected", "write", "rename"} {
			t.Run(mode, func(t *testing.T) {
				if mode == "projected" && runtime.GOOS != "linux" {
					t.Skip("Kubernetes projected volume notifications require Linux inotify")
				}
				ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
				defer cancel()
				cmd := exec.CommandContext(ctx, os.Args[0], "-test.run=^TestTLSReloadProjectedSecret$", "-test.v")
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
	a, ak := projectedTLSPair(t, 1, nil)
	block, _ := pem.Decode(ak)
	key, err := x509.ParseECPrivateKey(block.Bytes)
	if err != nil {
		t.Fatal(err)
	}
	b, bk := projectedTLSPair(t, 2, key)
	roots := x509.NewCertPool()
	roots.AppendCertsFromPEM(a)
	roots.AppendCertsFromPEM(b)
	projectedTLSWrite(t, certPath, a)
	projectedTLSWrite(t, keyPath, ak)
	logs := make(projectedTLSLog, 64)
	logger := slog.New(slog.NewTextHandler(logs, &slog.HandlerOptions{Level: slog.LevelDebug}))
	caPath := ""
	var client *tls.Certificate

	if os.Getenv(env) == "projected" {
		version := filepath.Join(dir, "..v1")
		if err := os.Mkdir(version, 0o700); err != nil {
			t.Fatal(err)
		}
		projectedTLSWrite(t, filepath.Join(version, "tls.crt"), a)
		projectedTLSWrite(t, filepath.Join(version, "tls.key"), ak)
		if err := os.Remove(certPath); err != nil {
			t.Fatal(err)
		}
		if err := os.Remove(keyPath); err != nil {
			t.Fatal(err)
		}
		for name, target := range map[string]string{"..data": "..v1", "tls.crt": "..data/tls.crt", "tls.key": "..data/tls.key"} {
			if err := os.Symlink(target, filepath.Join(dir, name)); err != nil {
				t.Fatal(err)
			}
		}
	}

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

	projectedTLSExpect(t, addr, roots, "localhost", client, 1)
	switch os.Getenv(env) {
	case "projected":
		version := filepath.Join(dir, "..v2")
		if err := os.Mkdir(version, 0o700); err != nil {
			t.Fatal(err)
		}
		projectedTLSWrite(t, filepath.Join(version, "tls.crt"), b)
		projectedTLSWrite(t, filepath.Join(version, "tls.key"), bk)
		if err := os.Symlink("..v2", filepath.Join(dir, "..data_tmp")); err != nil {
			t.Fatal(err)
		}
		if err := os.Rename(filepath.Join(dir, "..data_tmp"), filepath.Join(dir, "..data")); err != nil {
			t.Fatal(err)
		}
	case "write":
		projectedTLSWrite(t, certPath, b)
	case "rename":
		tmp := filepath.Join(dir, "cert.tmp")
		projectedTLSWrite(t, tmp, b)
		if err := os.Rename(tmp, certPath); err != nil {
			t.Fatal(err)
		}
	}

	projectedTLSExpect(t, addr, roots, "localhost", client, 2)
	if os.Getenv(env) == "write" {
		// A truncate or partial write must not discard the last valid keypair.
		projectedTLSWrite(t, certPath, []byte("partial certificate"))
		logs.wait(t, "reload TLS config")
		projectedTLSSignal(t, logs)
		logs.wait(t, "reload TLS config")
		got, err := projectedTLSSerial(addr, roots, "localhost", client)
		if err != nil || got != 2 {
			t.Fatalf("partial write discarded certificate: serial=%d err=%v", got, err)
		}
		projectedTLSWrite(t, certPath, b)
		projectedTLSSignal(t, logs)
		projectedTLSExpect(t, addr, roots, "localhost", client, 2)
	}
	// A later key rename provides a positive event after the unrelated file.
	// Match paths so a delayed certificate event cannot fail this assertion.
	unrelated := filepath.Join(dir, "unrelated.tmp")
	projectedTLSWrite(t, unrelated, []byte("data"))
	keyTemp := filepath.Join(dir, "key.tmp")
	projectedTLSWrite(t, keyTemp, bk)
	if err := os.Rename(keyTemp, keyPath); err != nil {
		t.Fatal(err)
	}
	timer := time.NewTimer(5 * time.Second)
	defer timer.Stop()
events:
	for {
		select {
		case line := <-logs:
			if strings.Contains(line, unrelated) {
				t.Fatalf("unrelated event: %s", line)
			}
			if strings.Contains(line, "reloading cert from fsnotify") && strings.Contains(line, keyPath) {
				break events
			}
		case <-timer.C:
			t.Fatal("key rename event was not processed")
		}
	}
	// Keep the signal path working as well.
	projectedTLSSignal(t, logs)
	projectedTLSExpect(t, addr, roots, "localhost", client, 2)
}
