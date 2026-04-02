package server

import (
	"crypto/tls"
	"crypto/x509"
	"fmt"
	"log"
	"net"
	"net/http"
	"net/http/httputil"
	"os"
	"slices"
	"time"
)

// newHTTPClient creates an *http.Client with optional custom root CAs and debug logging.
func newHTTPClient(rootCAs string, debug bool) (*http.Client, error) {
	var client *http.Client

	if rootCAs != "" {
		tlsConfig := &tls.Config{RootCAs: x509.NewCertPool()}
		rootCABytes, err := os.ReadFile(rootCAs)
		if err != nil {
			return nil, fmt.Errorf("failed to read root-ca: %v", err)
		}
		if !tlsConfig.RootCAs.AppendCertsFromPEM(rootCABytes) {
			return nil, fmt.Errorf("no certs found in root CA file %q", rootCAs)
		}
		client = &http.Client{
			Transport: &http.Transport{
				TLSClientConfig: tlsConfig,
				Proxy:           http.ProxyFromEnvironment,
				DialContext: (&net.Dialer{
					Timeout:   30 * time.Second,
					KeepAlive: 30 * time.Second,
				}).DialContext,
				TLSHandshakeTimeout:   10 * time.Second,
				ExpectContinueTimeout: 1 * time.Second,
			},
		}
	}

	if debug {
		if client == nil {
			client = &http.Client{
				Transport: debugTransport{http.DefaultTransport},
			}
		} else {
			client.Transport = debugTransport{client.Transport}
		}
	}

	if client == nil {
		client = http.DefaultClient
	}

	return client, nil
}

// debugTransport wraps an http.RoundTripper and logs full request/response details.
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

// buildScopes constructs a scope list from base scopes and cross-client IDs.
func buildScopes(baseScopes, crossClients []string) []string {
	scopes := make([]string, len(baseScopes))
	copy(scopes, baseScopes)

	for _, client := range crossClients {
		if client != "" {
			scopes = append(scopes, "audience:server:client_id:"+client)
		}
	}

	return uniqueStrings(scopes)
}

// uniqueStrings deduplicates and sorts a string slice in place.
func uniqueStrings(values []string) []string {
	slices.Sort(values)
	return slices.Compact(values)
}
