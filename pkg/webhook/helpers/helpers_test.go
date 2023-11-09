package helpers

import (
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/tls"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/pem"
	"math/big"
	"net"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/dexidp/dex/pkg/webhook/config"
)

func TestNewWebhookHTTPHelpers_InsecureSkip(t *testing.T) {
	h, err := NewWebhookHTTPHelpers(&config.WebhookConfig{
		URL:                "https://test.com",
		InsecureSkipVerify: true,
	})
	assert.NoError(t, err)
	assert.Equal(t, h.(*webhookHTTPHelpersImpl).url, "https://test.com")
	assert.Equal(t, h.(*webhookHTTPHelpersImpl).transport.TLSClientConfig.InsecureSkipVerify, true)
}

func TestNewWebhookHTTPHelpers_TLSRootCAFile(t *testing.T) {
	dir, err := os.MkdirTemp("", "prefix")
	require.NoError(t, err)
	defer os.RemoveAll(dir)
	caCertPem, rootCAPool, _, _, err := generateCA()
	require.NoError(t, err)
	require.NotNil(t, caCertPem)
	filePath := filepath.Join(dir, "ca.crt")

	err = os.WriteFile(filePath, caCertPem, 0o644)
	require.NoError(t, err)
	h, err := NewWebhookHTTPHelpers(&config.WebhookConfig{
		URL:                "https://test.com",
		InsecureSkipVerify: false,
		TLSRootCAFile:      filePath,
	})
	assert.NoError(t, err)
	assert.Equal(t, h.(*webhookHTTPHelpersImpl).url, "https://test.com")
	assert.Equal(t, h.(*webhookHTTPHelpersImpl).transport.TLSClientConfig.InsecureSkipVerify, false)
	assert.NotNil(t, h.(*webhookHTTPHelpersImpl).transport.TLSClientConfig.RootCAs)
	assert.True(t, h.(*webhookHTTPHelpersImpl).transport.TLSClientConfig.RootCAs.Equal(rootCAPool))
}

func TestNewWebhookHTTPHelpers_TLSClientCertFile(t *testing.T) {
	dir, err := os.MkdirTemp("", "prefix")
	require.NoError(t, err)
	defer os.RemoveAll(dir)
	caCertPem, rootCAPool, caTemplate, caPrivateKey, err := generateCA()
	require.NoError(t, err)
	require.NotNil(t, caCertPem)
	filePath := filepath.Join(dir, "ca.crt")
	err = os.WriteFile(filePath, caCertPem, 0o644)
	require.NoError(t, err)

	certPEM, cert, keyPEM, key, err := generateTestCertificates(true, *caTemplate, caPrivateKey)
	require.NoError(t, err)
	certFilePath := filepath.Join(dir, "cert.pem")
	keyFilePath := filepath.Join(dir, "key.pem")
	err = os.WriteFile(certFilePath, certPEM, 0o644)
	require.NoError(t, err)
	err = os.WriteFile(keyFilePath, keyPEM, 0o644)
	require.NoError(t, err)

	h, err := NewWebhookHTTPHelpers(&config.WebhookConfig{
		URL:                "https://test.com",
		InsecureSkipVerify: false,
		TLSRootCAFile:      filePath,
		ClientAuthentication: &config.ClientAuthentication{
			ClientCertificateFile: certFilePath,
			ClientKeyFile:         keyFilePath,
			ClientCAFile:          filePath,
		},
	})

	assert.NoError(t, err)
	assert.Equal(t, h.(*webhookHTTPHelpersImpl).url, "https://test.com")
	assert.Equal(t, h.(*webhookHTTPHelpersImpl).transport.TLSClientConfig.InsecureSkipVerify, false)
	assert.NotNil(t, h.(*webhookHTTPHelpersImpl).transport.TLSClientConfig.RootCAs)
	assert.True(t, h.(*webhookHTTPHelpersImpl).transport.TLSClientConfig.RootCAs.Equal(rootCAPool))
	assert.NotNil(t, h.(*webhookHTTPHelpersImpl).transport.TLSClientConfig.Certificates)
	assert.Equal(t, h.(*webhookHTTPHelpersImpl).transport.TLSClientConfig.Certificates[0], *cert)
	assert.NotNil(t, h.(*webhookHTTPHelpersImpl).transport.TLSClientConfig.Certificates[0].PrivateKey)
	assert.Equal(t, h.(*webhookHTTPHelpersImpl).transport.TLSClientConfig.Certificates[0].PrivateKey, key)
}

func generateTestCertificates(clientCert bool, caTemplate x509.Certificate,
	caPrivateKey *ecdsa.PrivateKey,
) ([]byte, *tls.Certificate, []byte, *ecdsa.PrivateKey, error) {
	// Generate a new private key (ECDSA P-256)
	privateKey, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	if err != nil {
		return nil, nil, nil, nil, err
	}

	keyUsage := []x509.ExtKeyUsage{x509.ExtKeyUsageServerAuth}
	if clientCert {
		keyUsage = []x509.ExtKeyUsage{x509.ExtKeyUsageClientAuth}
	}

	// Create a template for the certificate
	certTemplate := x509.Certificate{
		SerialNumber: big.NewInt(1),
		Subject: pkix.Name{
			CommonName: "localhost",
		},
		NotBefore:             time.Now(),
		NotAfter:              time.Now().Add(time.Hour * 24 * 365), // Valid for one year
		KeyUsage:              x509.KeyUsageDigitalSignature | x509.KeyUsageKeyEncipherment,
		BasicConstraintsValid: true,
		IsCA:                  true,
		IPAddresses:           []net.IP{net.ParseIP("127.0.0.1")},
		ExtKeyUsage:           keyUsage,
		DNSNames:              []string{"client.example.com", "alt.example.com"}, // SAN field for DNS names

	}

	// Generate the certificate signed by CA
	certDERBytes, err := x509.CreateCertificate(rand.Reader, &certTemplate, &caTemplate, &privateKey.PublicKey, caPrivateKey)
	if err != nil {
		return nil, nil, nil, nil, err
	}

	// PEM encode the certificate
	certPEM := pem.EncodeToMemory(&pem.Block{
		Type:  "CERTIFICATE",
		Bytes: certDERBytes,
	})

	// PEM encode the private key
	keyPEM, err := x509.MarshalECPrivateKey(privateKey)
	if err != nil {
		return nil, nil, nil, nil, err
	}
	keyPEMBlock := pem.EncodeToMemory(&pem.Block{
		Type:  "EC PRIVATE KEY",
		Bytes: keyPEM,
	})

	// Create a TLS certificate using the PEM-encoded certificate and private key
	tlsCert, err := tls.X509KeyPair(certPEM, keyPEMBlock)
	if err != nil {
		return nil, nil, nil, nil, err
	}

	return certPEM, &tlsCert, keyPEMBlock, privateKey, nil
}

func generateCA() ([]byte, *x509.CertPool, *x509.Certificate, *ecdsa.PrivateKey, error) {
	// Generate a new private key (ECDSA P-256)
	caPrivateKey, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	if err != nil {
		return nil, nil, nil, nil, err
	}

	// Create a template for the CA certificate
	caTemplate := x509.Certificate{
		DNSNames:     []string{"test.com"},
		SerialNumber: big.NewInt(1),
		Subject: pkix.Name{
			CommonName: "My CA",
		},
		NotBefore:             time.Now(),
		NotAfter:              time.Now().Add(time.Hour * 24 * 365 * 10), // Valid for 10 years
		KeyUsage:              x509.KeyUsageDigitalSignature | x509.KeyUsageCertSign,
		BasicConstraintsValid: true,
		IsCA:                  true,
	}

	// Generate the CA certificate
	caCertDERBytes, err := x509.CreateCertificate(rand.Reader, &caTemplate, &caTemplate, &caPrivateKey.PublicKey, caPrivateKey)
	if err != nil {
		return nil, nil, nil, nil, err
	}

	// PEM encode the CA certificate
	caCertPEM := pem.EncodeToMemory(&pem.Block{
		Type:  "CERTIFICATE",
		Bytes: caCertDERBytes,
	})

	caCertPool := x509.NewCertPool()
	if ok := caCertPool.AppendCertsFromPEM(caCertPEM); !ok {
		panic("Failed to append CA certificate to CertPool")
	}

	return caCertPEM, caCertPool, &caTemplate, caPrivateKey, nil
}
