package saml

import (
	"crypto/x509"
	"encoding/pem"
	"errors"
	"io/ioutil"

	"github.com/ma314smith/signedxml"
)

// Sign creates a signature for an XML document and returns it
func Sign(xml string, privateKeyPath string) (string, error) {
	pemString, err := ioutil.ReadFile(privateKeyPath)
	if err != nil {
		return "", err
	}

	pemBlock, _ := pem.Decode([]byte(pemString))
	if pemBlock == nil {
		return "", errors.New("Count not parse private key")
	}

	key, err := x509.ParsePKCS1PrivateKey(pemBlock.Bytes)
	if err != nil {
		return "", err
	}

	signer, err := signedxml.NewSigner(xml)
	if err != nil {
		return "", err
	}

	samlSignedRequestXML, err := signer.Sign(key)
	if err != nil {
		return "", err
	}

	return samlSignedRequestXML, nil
}

// Verify validates the signature of an XML document
func Verify(xml string, publicCertPath string) error {
	pemString, err := ioutil.ReadFile(publicCertPath)
	if err != nil {
		return err
	}

	pemBlock, _ := pem.Decode([]byte(pemString))
	if pemBlock == nil {
		return errors.New("Could not parse certificate")
	}

	cert, err := x509.ParseCertificate(pemBlock.Bytes)
	if err != nil {
		return err
	}

	validator, err := signedxml.NewValidator(xml)
	if err != nil {
		return err
	}

	validator.Certificates = append(validator.Certificates, *cert)

	err = validator.Validate()
	if err != nil {
		return err
	}
	return nil
}
