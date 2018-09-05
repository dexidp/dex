// Package xmlsig supports add XML Digital Signatures to Go structs marshalled to XML.
package xmlsig

import (
	"crypto"
	"crypto/rand"
	"errors"
	// import supported crypto hash function
	_ "crypto/sha1"
	_ "crypto/sha256"
	"crypto/tls"
	"crypto/x509"
	"encoding/base64"
)

// Signer is used to create a Signature for the provided object.
type Signer interface {
	Sign([]byte) (string, error)
	CreateSignature(interface{}) (*Signature, error)
	Algorithm() string
}

type signer struct {
	cert      string
	sigAlg    *algorithm
	digestAlg *algorithm
	key       crypto.Signer
}

type algorithm struct {
	name string
	hash crypto.Hash
}

type SignerOptions struct {
	SignatureAlgorithm string
	DigestAlgorithm    string
}

func pickSignatureAlgorithm(certType x509.PublicKeyAlgorithm, alg string) (*algorithm, error) {
	var hash crypto.Hash
	switch certType {
	case x509.RSA:
		switch alg {
		case "":
			alg = "http://www.w3.org/2000/09/xmldsig#rsa-sha1"
			hash = crypto.SHA1
		case "http://www.w3.org/2000/09/xmldsig#rsa-sha1":
			hash = crypto.SHA1
		case "http://www.w3.org/2001/04/xmldsig-more#rsa-sha256":
			hash = crypto.SHA256
		default:
			return nil, errors.New("xmlsig does not currently the specfied algorithm for RSA certificates")
		}
	case x509.DSA:
		switch alg {
		case "":
			alg = "http://www.w3.org/2000/09/xmldsig#dsa-sha1"
			hash = crypto.SHA1
		case "http://www.w3.org/2000/09/xmldsig#dsa-sha1":
			hash = crypto.SHA1
		case "http://www.w3.org/2009/xmldsig11#dsa-sha256":
			hash = crypto.SHA256
		default:
			return nil, errors.New("xmlsig does not currently the specfied algorithm for DSA certificates")
		}
	case x509.ECDSA:
		return nil, errors.New("xmlsig does not currently support ECDSA certificates")
	default:
		return nil, errors.New("xmlsig needs some work to support your certificate")
	}
	return &algorithm{alg, hash}, nil
}

func pickDigestAlgorithm(alg string) (*algorithm, error) {
	switch alg {
	case "":
		fallthrough
	case "http://www.w3.org/2000/09/xmldsig#sha1":
		return &algorithm{"http://www.w3.org/2000/09/xmldsig#sha1", crypto.SHA1}, nil
	case "http://www.w3.org/2001/04/xmlenc#sha256":
		return &algorithm{"http://www.w3.org/2001/04/xmlenc#sha256", crypto.SHA256}, nil
	}
	return nil, errors.New("xmlsig does not support the specified digest algorithm")
}

// NewSigner creates a new Signer with the certificate.
func NewSigner(cert tls.Certificate) (Signer, error) {
	return NewSignerWithOptions(cert, SignerOptions{})
}

// NewSigner creates a new Signer with the certificate and options
func NewSignerWithOptions(cert tls.Certificate, options SignerOptions) (Signer, error) {
	c := cert.Certificate[0]
	parsedCert, err := x509.ParseCertificate(c)
	if err != nil {
		return nil, err
	}
	sigAlg, err := pickSignatureAlgorithm(parsedCert.PublicKeyAlgorithm, options.SignatureAlgorithm)
	if err != nil {
		return nil, err
	}
	digestAlg, err := pickDigestAlgorithm(options.DigestAlgorithm)
	if err != nil {
		return nil, err
	}
	k := cert.PrivateKey.(crypto.Signer)
	return &signer{base64.StdEncoding.EncodeToString(c), sigAlg, digestAlg, k}, nil
}

func (s *signer) Algorithm() string {
	return s.sigAlg.name
}

func (s *signer) CreateSignature(data interface{}) (*Signature, error) {
	signature := newSignature()
	signature.SignedInfo.SignatureMethod.Algorithm = s.sigAlg.name
	signature.SignedInfo.Reference.DigestMethod.Algorithm = s.digestAlg.name
	// canonicalize the Item
	canonData, id, err := canonicalize(data)
	if err != nil {
		return nil, err
	}
	if id != "" {
		signature.SignedInfo.Reference.URI = "#" + id
	}
	// calculate the digest
	digest := s.digest(canonData)
	signature.SignedInfo.Reference.DigestValue = digest
	// canonicalize the SignedInfo
	canonData, _, err = canonicalize(signature.SignedInfo)
	if err != nil {
		return nil, err
	}
	sig, err := s.Sign(canonData)
	if err != nil {
		return nil, err
	}
	signature.SignatureValue = sig
	x509Data := &X509Data{X509Certificate: s.cert}
	signature.KeyInfo.X509Data = x509Data
	return signature, nil
}

func (s *signer) Sign(data []byte) (string, error) {
	h := s.sigAlg.hash.New()
	h.Write(data)
	sum := h.Sum(nil)
	sig, err := s.key.Sign(rand.Reader, sum, s.sigAlg.hash)
	if err != nil {
		return "", err
	}
	return base64.StdEncoding.EncodeToString(sig), nil
}

func newSignature() *Signature {
	signature := &Signature{}
	signature.SignedInfo.CanonicalizationMethod.Algorithm =
		"http://www.w3.org/2001/10/xml-exc-c14n#"
	transforms := &signature.SignedInfo.Reference.Transforms.Transform
	*transforms = append(*transforms, Algorithm{"http://www.w3.org/2000/09/xmldsig#enveloped-signature"})
	*transforms = append(*transforms, Algorithm{"http://www.w3.org/2001/10/xml-exc-c14n#"})
	return signature
}

func (s *signer) digest(data []byte) string {
	h := s.digestAlg.hash.New()
	h.Write(data)
	sum := h.Sum(nil)
	return base64.StdEncoding.EncodeToString(sum)
}
