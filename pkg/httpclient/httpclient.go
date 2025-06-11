package httpclient

import (
	"crypto/tls"
	"crypto/x509"
	"encoding/base64"
	"fmt"
	"net"
	"net/http"
	"os"
	"time"
)

func extractCAs(input []string) [][]byte {
	result := make([][]byte, 0, len(input))
	for _, ca := range input {
		if ca == "" {
			continue
		}

		pemData, err := os.ReadFile(ca)
		if err != nil {
			pemData, err = base64.StdEncoding.DecodeString(ca)
			if err != nil {
				pemData = []byte(ca)
			}
		}

		result = append(result, pemData)
	}
	return result
}

func NewHTTPClient(rootCAs []string, insecureSkipVerify bool) (*http.Client, error) {
	pool, err := x509.SystemCertPool()
	if err != nil {
		return nil, err
	}

	tlsConfig := tls.Config{RootCAs: pool, InsecureSkipVerify: insecureSkipVerify}
	for index, rootCABytes := range extractCAs(rootCAs) {
		if !tlsConfig.RootCAs.AppendCertsFromPEM(rootCABytes) {
			return nil, fmt.Errorf("rootCAs.%d is not in PEM format, certificate must be "+
				"a PEM encoded string, a base64 encoded bytes that contain PEM encoded string, "+
				"or a path to a PEM encoded certificate", index)
		}
	}

	return &http.Client{
		Transport: &http.Transport{
			TLSClientConfig: &tlsConfig,
			Proxy:           http.ProxyFromEnvironment,
			DialContext: (&net.Dialer{
				Timeout:   30 * time.Second,
				KeepAlive: 30 * time.Second,
				DualStack: true,
			}).DialContext,
			MaxIdleConns:          100,
			IdleConnTimeout:       90 * time.Second,
			TLSHandshakeTimeout:   10 * time.Second,
			ExpectContinueTimeout: 1 * time.Second,
		},
	}, nil
}
