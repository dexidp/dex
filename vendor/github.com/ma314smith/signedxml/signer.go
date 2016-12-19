package signedxml

import (
	"crypto"
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"encoding/base64"
	"errors"

	"github.com/beevik/etree"
)

var signingAlgorithms map[x509.SignatureAlgorithm]cryptoHash

func init() {
	signingAlgorithms = map[x509.SignatureAlgorithm]cryptoHash{
		// MD2 not supported
		// x509.MD2WithRSA: cryptoHash{algorithm: "rsa", hash: crypto.MD2},
		x509.MD5WithRSA:    cryptoHash{algorithm: "rsa", hash: crypto.MD5},
		x509.SHA1WithRSA:   cryptoHash{algorithm: "rsa", hash: crypto.SHA1},
		x509.SHA256WithRSA: cryptoHash{algorithm: "rsa", hash: crypto.SHA256},
		x509.SHA384WithRSA: cryptoHash{algorithm: "rsa", hash: crypto.SHA384},
		x509.SHA512WithRSA: cryptoHash{algorithm: "rsa", hash: crypto.SHA512},
		// DSA not supported
		// x509.DSAWithSHA1:  cryptoHash{algorithm: "dsa", hash: crypto.SHA1},
		// x509.DSAWithSHA256:cryptoHash{algorithm: "dsa", hash: crypto.SHA256},
		// Golang ECDSA support is lacking, can't seem to load private keys
		// x509.ECDSAWithSHA1:   cryptoHash{algorithm: "ecdsa", hash: crypto.SHA1},
		// x509.ECDSAWithSHA256: cryptoHash{algorithm: "ecdsa", hash: crypto.SHA256},
		// x509.ECDSAWithSHA384: cryptoHash{algorithm: "ecdsa", hash: crypto.SHA384},
		// x509.ECDSAWithSHA512: cryptoHash{algorithm: "ecdsa", hash: crypto.SHA512},
	}
}

type cryptoHash struct {
	algorithm string
	hash      crypto.Hash
}

// Signer provides options for signing an XML document
type Signer struct {
	signatureData
	privateKey interface{}
}

// NewSigner returns a *Signer for the XML provided
func NewSigner(xml string) (*Signer, error) {
	doc := etree.NewDocument()
	err := doc.ReadFromString(xml)
	if err != nil {
		return nil, err
	}
	s := &Signer{signatureData: signatureData{xml: doc}}
	return s, nil
}

// Sign populates the XML digest and signature based on the parameters present and privateKey given
func (s *Signer) Sign(privateKey interface{}) (string, error) {
	s.privateKey = privateKey

	if s.signature == nil {
		if err := s.parseEnvelopedSignature(); err != nil {
			return "", err
		}
	}
	if err := s.parseSignedInfo(); err != nil {
		return "", err
	}
	if err := s.parseSigAlgorithm(); err != nil {
		return "", err
	}
	if err := s.parseCanonAlgorithm(); err != nil {
		return "", err
	}
	if err := s.setDigest(); err != nil {
		return "", err
	}
	if err := s.setSignature(); err != nil {
		return "", err
	}

	xml, err := s.xml.WriteToString()
	if err != nil {
		return "", err
	}
	return xml, nil
}

func (s *Signer) setDigest() (err error) {
	references := s.signedInfo.FindElements("./Reference")
	for _, ref := range references {
		doc := s.xml.Copy()
		transforms := ref.SelectElement("Transforms")
		for _, transform := range transforms.SelectElements("Transform") {
			doc, err = processTransform(transform, doc)
			if err != nil {
				return err
			}
		}

		doc, err := getReferencedXML(ref, doc)
		if err != nil {
			return err
		}

		calculatedValue, err := calculateHash(ref, doc)
		if err != nil {
			return err
		}

		digestValueElement := ref.SelectElement("DigestValue")
		if digestValueElement == nil {
			return errors.New("signedxml: unable to find DigestValue")
		}
		digestValueElement.SetText(calculatedValue)
	}
	return nil
}

func (s *Signer) setSignature() error {
	doc := etree.NewDocument()
	doc.SetRoot(s.signedInfo.Copy())
	signedInfo, err := doc.WriteToString()
	if err != nil {
		return err
	}

	canonSignedInfo, err := s.canonAlgorithm.Process(signedInfo, "")
	if err != nil {
		return err
	}

	var hashed, signature []byte
	//var h1, h2 *big.Int
	signingAlgorithm, ok := signingAlgorithms[s.sigAlgorithm]
	if !ok {
		return errors.New("signedxml: unsupported algorithm")
	}

	hasher := signingAlgorithm.hash.New()
	hasher.Write([]byte(canonSignedInfo))
	hashed = hasher.Sum(nil)

	switch signingAlgorithm.algorithm {
	case "rsa":
		signature, err = rsa.SignPKCS1v15(rand.Reader, s.privateKey.(*rsa.PrivateKey), signingAlgorithm.hash, hashed)
		/*
			case "dsa":
				h1, h2, err = dsa.Sign(rand.Reader, s.privateKey.(*dsa.PrivateKey), hashed)
			case "ecdsa":
				h1, h2, err = ecdsa.Sign(rand.Reader, s.privateKey.(*ecdsa.PrivateKey), hashed)
		*/
	}
	if err != nil {
		return err
	}

	// DSA and ECDSA has not been validated
	/*
		if signature == nil && h1 != nil && h2 != nil {
			signature = append(h1.Bytes(), h2.Bytes()...)
		}
	*/

	b64 := base64.StdEncoding.EncodeToString(signature)
	sigValueElement := s.signature.SelectElement("SignatureValue")
	sigValueElement.SetText(b64)

	return nil
}
