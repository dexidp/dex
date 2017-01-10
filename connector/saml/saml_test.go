package saml

import (
	"crypto/x509"
	"encoding/pem"
	"errors"
	"io/ioutil"
	"testing"

	sdig "github.com/russellhaering/goxmldsig"
)

func loadCert(ca string) (*x509.Certificate, error) {
	data, err := ioutil.ReadFile(ca)
	if err != nil {
		return nil, err
	}
	block, _ := pem.Decode(data)
	if block == nil {
		return nil, errors.New("ca file didn't contain any PEM data")
	}
	return x509.ParseCertificate(block.Bytes)
}

func TestVerify(t *testing.T) {
	cert, err := loadCert("testdata/okta-ca.pem")
	if err != nil {
		t.Fatal(err)
	}
	s := certStore{[]*x509.Certificate{cert}}

	validator := sdig.NewDefaultValidationContext(s)

	data, err := ioutil.ReadFile("testdata/okta-resp.xml")
	if err != nil {
		t.Fatal(err)
	}

	if _, err := verify(validator, data); err != nil {
		t.Fatal(err)
	}
}
