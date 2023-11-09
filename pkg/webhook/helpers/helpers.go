//go:generate go run -mod mod go.uber.org/mock/mockgen -destination=mock_helpers.go -package=helpers --source=helpers.go WebhookHTTPHelpers
package helpers

import (
	"bytes"
	"crypto/tls"
	"crypto/x509"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"

	"github.com/dexidp/dex/pkg/webhook/config"
)

type WebhookHTTPHelpers interface {
	CallWebhook(jsonData []byte) ([]byte, error)
}

type webhookHTTPHelpersImpl struct {
	transport *http.Transport
	url       string
}

var _ WebhookHTTPHelpers = &webhookHTTPHelpersImpl{}

func NewWebhookHTTPHelpers(cfg *config.WebhookConfig) (WebhookHTTPHelpers, error) {
	if cfg == nil {
		return nil, errors.New("webhook config is nil")
	}
	if cfg.URL == "" {
		return nil, errors.New("webhook url is empty")
	}
	transport, err := createTransport(cfg)
	if err != nil {
		return nil, err
	}
	return &webhookHTTPHelpersImpl{
		transport: transport,
		url:       cfg.URL,
	}, nil
}

func (h *webhookHTTPHelpersImpl) CallWebhook(jsonData []byte) ([]byte, error) {
	req, err := http.NewRequest("POST", h.url, bytes.NewBuffer(jsonData))
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")

	client := &http.Client{Transport: h.transport}
	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to send request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("unexpected status code: %d", resp.StatusCode)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("could not read response body: %v", err)
	}

	return body, nil
}

func createTransport(cfg *config.WebhookConfig) (*http.Transport, error) {
	p, err := url.Parse(cfg.URL)
	if err != nil {
		return nil, fmt.Errorf("could not parse url: %v", err)
	}
	switch p.Scheme {
	case "http":
		return &http.Transport{}, nil
	case "https":
		return createHTTPSTransport(cfg)
	default:
		return nil, fmt.Errorf("unsupported scheme: %s", p.Scheme)
	}
}

func createHTTPSTransport(cfg *config.WebhookConfig) (*http.Transport, error) {
	var err error
	rootCertPool := x509.NewCertPool()
	if cfg.TLSRootCAFile != "" {
		rootCertPool, err = readCACert(cfg.TLSRootCAFile)
		if err != nil {
			return nil, fmt.Errorf("failed to read file %q: %w", cfg.TLSRootCAFile, err)
		}
	}

	tr := &http.Transport{
		TLSClientConfig: &tls.Config{
			RootCAs:            rootCertPool,
			InsecureSkipVerify: cfg.InsecureSkipVerify,
			MinVersion:         tls.VersionTLS13,
		},
	}

	if cfg.ClientAuthentication != nil {
		clientCert, err := ReadCertificate(cfg.ClientAuthentication.ClientCertificateFile,
			cfg.ClientAuthentication.ClientKeyFile)
		if err != nil {
			return nil, fmt.Errorf("failed to read certificate: %w", err)
		}
		tr.TLSClientConfig.Certificates = []tls.Certificate{*clientCert}
	}

	return tr, nil
}

func readCACert(caPath string) (*x509.CertPool, error) {
	caCertPool := x509.NewCertPool()
	// Load CA cert
	caCert, err := os.ReadFile(caPath)
	if err != nil {
		return nil, err
	}
	caCertPool.AppendCertsFromPEM(caCert)
	return caCertPool, nil
}

func ReadCertificate(certPath, keyPath string) (*tls.Certificate, error) {
	cer, err := tls.LoadX509KeyPair(certPath, keyPath)
	if err != nil {
		return nil, err
	}
	return &cer, nil
}
