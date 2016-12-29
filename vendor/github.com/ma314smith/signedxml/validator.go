package signedxml

import (
	"crypto/x509"
	"encoding/base64"
	"errors"
	"fmt"
	"log"

	"github.com/beevik/etree"
)

// Validator provides options for verifying a signed XML document
type Validator struct {
	Certificates []x509.Certificate
	signingCert  x509.Certificate
	signatureData
}

// NewValidator returns a *Validator for the XML provided
func NewValidator(xml string) (*Validator, error) {
	doc := etree.NewDocument()
	err := doc.ReadFromString(xml)
	if err != nil {
		return nil, err
	}
	v := &Validator{signatureData: signatureData{xml: doc}}
	return v, nil
}

// SetXML is used to assign the XML document that the Validator will verify
func (v *Validator) SetXML(xml string) error {
	doc := etree.NewDocument()
	err := doc.ReadFromString(xml)
	v.xml = doc
	return err
}

// SigningCert returns the certificate, if any, that was used to successfully
// validate the signature of the XML document. This will be a zero value
// x509.Certificate before Validator.Validate is successfully called.
func (v *Validator) SigningCert() x509.Certificate {
	return v.signingCert
}

// Validate validates the Reference digest values, and the signature value
// over the SignedInfo.
//
// If the signature is enveloped in the Validator.XML, then it will be used.
// Otherwise, an external signature should be assigned using
// Validator.SetSignature.
func (v *Validator) Validate() error {
	if err := v.loadValuesFromXML(); err != nil {
		return err
	}

	if err := v.validateReferences(); err != nil {
		return err
	}

	err := v.validateSignature()
	return err
}

func (v *Validator) loadValuesFromXML() error {
	if v.signature == nil {
		if err := v.parseEnvelopedSignature(); err != nil {
			return err
		}
	}
	if err := v.parseSignedInfo(); err != nil {
		return err
	}
	if err := v.parseSigValue(); err != nil {
		return err
	}
	if err := v.parseSigAlgorithm(); err != nil {
		return err
	}
	if err := v.parseCanonAlgorithm(); err != nil {
		return err
	}
	if err := v.loadCertificates(); err != nil {
		return err
	}
	return nil
}

func (v *Validator) validateReferences() (err error) {
	references := v.signedInfo.FindElements("./Reference")
	for _, ref := range references {
		doc := v.xml.Copy()
		transforms := ref.SelectElement("Transforms")
		for _, transform := range transforms.SelectElements("Transform") {
			doc, err = processTransform(transform, doc)
			if err != nil {
				return err
			}
		}

		doc, err = getReferencedXML(ref, doc)
		if err != nil {
			return err
		}

		digestValueElement := ref.SelectElement("DigestValue")
		if digestValueElement == nil {
			return errors.New("signedxml: unable to find DigestValue")
		}
		digestValue := digestValueElement.Text()

		calculatedValue, err := calculateHash(ref, doc)
		if err != nil {
			return err
		}

		if calculatedValue != digestValue {
			return fmt.Errorf("signedxml: Calculated digest does not match the"+
				" expected digestvalue of %s", digestValue)
		}
	}
	return nil
}

func (v *Validator) validateSignature() error {
	doc := etree.NewDocument()
	doc.SetRoot(v.signedInfo.Copy())
	signedInfo, err := doc.WriteToString()
	if err != nil {
		return err
	}

	canonSignedInfo, err := v.canonAlgorithm.Process(signedInfo, "")
	if err != nil {
		return err
	}

	b64, err := base64.StdEncoding.DecodeString(v.sigValue)
	if err != nil {
		return err
	}
	sig := []byte(b64)

	v.signingCert = x509.Certificate{}
	for _, cert := range v.Certificates {
		err := cert.CheckSignature(v.sigAlgorithm, []byte(canonSignedInfo), sig)
		if err == nil {
			v.signingCert = cert
			return nil
		}
	}

	return errors.New("signedxml: Calculated signature does not match the " +
		"SignatureValue provided")
}

func (v *Validator) loadCertificates() error {
	// If v.Certificates is already populated, then the client has already set it
	// to the desired cert. Otherwise, let's pull the public keys from the XML
	if len(v.Certificates) < 1 {
		keydata := v.xml.FindElements(".//X509Certificate")
		for _, key := range keydata {
			cert, err := getCertFromPEMString(key.Text())
			if err != nil {
				log.Printf("signedxml: Unable to load certificate: (%s). "+
					"Looking for another cert.", err)
			} else {
				v.Certificates = append(v.Certificates, *cert)
			}
		}
	}

	if len(v.Certificates) < 1 {
		return errors.New("signedxml: a certificate is required, but was not found")
	}
	return nil
}
