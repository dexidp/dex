// Package signedxml transforms and validates signedxml documents
package signedxml

import (
	"crypto"
	"crypto/x509"
	"encoding/base64"
	"encoding/pem"
	"errors"
	"fmt"
	"log"
	"os"
	"strings"

	"github.com/beevik/etree"
)

var logger = log.New(os.Stdout, "DEBUG-SIGNEDXML: ", log.Ldate|log.Ltime|log.Lshortfile)

func init() {
	hashAlgorithms = map[string]crypto.Hash{
		"http://www.w3.org/2001/04/xmldsig-more#md5":    crypto.MD5,
		"http://www.w3.org/2000/09/xmldsig#sha1":        crypto.SHA1,
		"http://www.w3.org/2001/04/xmldsig-more#sha224": crypto.SHA224,
		"http://www.w3.org/2001/04/xmlenc#sha256":       crypto.SHA256,
		"http://www.w3.org/2001/04/xmldsig-more#sha384": crypto.SHA384,
		"http://www.w3.org/2001/04/xmlenc#sha512":       crypto.SHA512,
		"http://www.w3.org/2001/04/xmlenc#ripemd160":    crypto.RIPEMD160,
	}

	signatureAlgorithms = map[string]x509.SignatureAlgorithm{
		"http://www.w3.org/2001/04/xmldsig-more#rsa-md2":      x509.MD2WithRSA,
		"http://www.w3.org/2001/04/xmldsig-more#rsa-md5":      x509.MD5WithRSA,
		"http://www.w3.org/2000/09/xmldsig#rsa-sha1":          x509.SHA1WithRSA,
		"http://www.w3.org/2001/04/xmldsig-more#rsa-sha256":   x509.SHA256WithRSA,
		"http://www.w3.org/2001/04/xmldsig-more#rsa-sha384":   x509.SHA384WithRSA,
		"http://www.w3.org/2001/04/xmldsig-more#rsa-sha512":   x509.SHA512WithRSA,
		"http://www.w3.org/2000/09/xmldsig#dsa-sha1":          x509.DSAWithSHA1,
		"http://www.w3.org/2000/09/xmldsig#dsa-sha256":        x509.DSAWithSHA256,
		"http://www.w3.org/2001/04/xmldsig-more#ecdsa-sha1":   x509.ECDSAWithSHA1,
		"http://www.w3.org/2001/04/xmldsig-more#ecdsa-sha256": x509.ECDSAWithSHA256,
		"http://www.w3.org/2001/04/xmldsig-more#ecdsa-sha384": x509.ECDSAWithSHA384,
		"http://www.w3.org/2001/04/xmldsig-more#ecdsa-sha512": x509.ECDSAWithSHA512,
	}

	CanonicalizationAlgorithms = map[string]CanonicalizationAlgorithm{
		"http://www.w3.org/2000/09/xmldsig#enveloped-signature": EnvelopedSignature{},
		"http://www.w3.org/2001/10/xml-exc-c14n#":               ExclusiveCanonicalization{},
		"http://www.w3.org/2001/10/xml-exc-c14n#WithComments":   ExclusiveCanonicalization{WithComments: true},
	}
}

// CanonicalizationAlgorithm defines an interface for processing an XML
// document into a standard format.
//
// If any child elements are in the Transform node, the entire transform node
// will be passed to the Process method through the transformXML parameter as an
// XML string. This is necessary for transforms that need additional processing
// data, like XPath (http://www.w3.org/TR/xmldsig-core/#sec-XPath). If there are
// no child elements in Transform (or CanonicalizationMethod), then an empty
// string will be passed through.
type CanonicalizationAlgorithm interface {
	Process(inputXML string, transformXML string) (outputXML string, err error)
}

// CanonicalizationAlgorithms maps the CanonicalizationMethod or
// Transform Algorithm URIs to a type that implements the
// CanonicalizationAlgorithm interface.
//
// Implementations are provided for the following transforms:
//  http://www.w3.org/2001/10/xml-exc-c14n# (ExclusiveCanonicalization)
//  http://www.w3.org/2001/10/xml-exc-c14n#WithComments (ExclusiveCanonicalizationWithComments)
//  http://www.w3.org/2000/09/xmldsig#enveloped-signature (EnvelopedSignature)
//
// Custom implementations can be added to the map
var CanonicalizationAlgorithms map[string]CanonicalizationAlgorithm
var hashAlgorithms map[string]crypto.Hash
var signatureAlgorithms map[string]x509.SignatureAlgorithm

// signatureData provides options for verifying a signed XML document
type signatureData struct {
	xml            *etree.Document
	signature      *etree.Element
	signedInfo     *etree.Element
	sigValue       string
	sigAlgorithm   x509.SignatureAlgorithm
	canonAlgorithm CanonicalizationAlgorithm
}

// SetSignature can be used to assign an external signature for the XML doc
// that Validator will verify
func (s *signatureData) SetSignature(sig string) error {
	doc := etree.NewDocument()
	err := doc.ReadFromString(sig)
	s.signature = doc.Root()
	return err
}

func (s *signatureData) parseEnvelopedSignature() error {
	sig := s.xml.FindElement(".//Signature")
	if sig != nil {
		s.signature = sig
	} else {
		return errors.New("signedxml: Unable to find a unique signature element " +
			"in the xml document. The signature must either be enveloped in the " +
			"xml doc or externally assigned to Validator.SetSignature")
	}
	return nil
}

func (s *signatureData) parseSignedInfo() error {
	s.signedInfo = nil
	s.signedInfo = s.signature.SelectElement("SignedInfo")
	if s.signedInfo == nil {
		return errors.New("signedxml: unable to find SignedInfo element")
	}

	// move the Signature level namespace down to SignedInfo so that the signature
	// value will match up
	if s.signedInfo.Space != "" {
		attr := s.signature.SelectAttr(s.signedInfo.Space)
		if attr != nil {
			s.signedInfo.Attr = []etree.Attr{*attr}
		}
	} else {
		attr := s.signature.SelectAttr("xmlns")
		if attr != nil {
			s.signedInfo.Attr = []etree.Attr{*attr}
		}
	}

	// Copy SignedInfo xmlns: into itself if it does not exist and is defined as a root attribute
	root := s.xml.Root()

	if root != nil {
		sigNS := root.SelectAttr("xmlns:" + s.signedInfo.Space)
		if sigNS != nil {
			if s.signedInfo.SelectAttr("xmlns:"+s.signedInfo.Space) == nil {
				s.signedInfo.CreateAttr("xmlns:"+s.signedInfo.Space, sigNS.Value)
			}
		}
	}

	return nil
}

func (s *signatureData) parseSigValue() error {
	s.sigValue = ""
	sigValueElement := s.signature.SelectElement("SignatureValue")
	if sigValueElement != nil {
		s.sigValue = sigValueElement.Text()
		return nil
	}
	return errors.New("signedxml: unable to find SignatureValue")
}

func (s *signatureData) parseSigAlgorithm() error {
	s.sigAlgorithm = x509.UnknownSignatureAlgorithm
	sigMethod := s.signedInfo.SelectElement("SignatureMethod")

	var sigAlgoURI string
	if sigMethod == nil {
		return errors.New("signedxml: Unable to find SignatureMethod element")
	}

	sigAlgoURI = sigMethod.SelectAttrValue("Algorithm", "")
	sigAlgo, ok := signatureAlgorithms[sigAlgoURI]
	if ok {
		s.sigAlgorithm = sigAlgo
		return nil
	}

	return errors.New("signedxml: Unable to find Algorithm in SignatureMethod element")
}

func (s *signatureData) parseCanonAlgorithm() error {
	s.canonAlgorithm = nil
	canonMethod := s.signedInfo.SelectElement("CanonicalizationMethod")

	var canonAlgoURI string
	if canonMethod == nil {
		return errors.New("signedxml: Unable to find CanonicalizationMethod element")
	}

	canonAlgoURI = canonMethod.SelectAttrValue("Algorithm", "")
	canonAlgo, ok := CanonicalizationAlgorithms[canonAlgoURI]
	if ok {
		s.canonAlgorithm = canonAlgo
		return nil
	}

	return errors.New("signedxml: Unable to find Algorithm in " +
		"CanonicalizationMethod element")
}

func getReferencedXML(reference *etree.Element, inputDoc *etree.Document) (outputDoc *etree.Document, err error) {
	uri := reference.SelectAttrValue("URI", "")
	uri = strings.Replace(uri, "#", "", 1)
	// populate doc with the referenced xml from the Reference URI
	if uri == "" {
		outputDoc = inputDoc
	} else {
		path := fmt.Sprintf(".//[@ID='%s']", uri)
		e := inputDoc.FindElement(path)
		if e != nil {
			outputDoc = etree.NewDocument()
			outputDoc.SetRoot(e.Copy())
		} else {
			// SAML v1.1 Assertions use AssertionID
			path := fmt.Sprintf(".//[@AssertionID='%s']", uri)
			e := inputDoc.FindElement(path)
			if e != nil {
				outputDoc = etree.NewDocument()
				outputDoc.SetRoot(e.Copy())
			}
		}
	}

	if outputDoc == nil {
		return nil, errors.New("signedxml: unable to find refereced xml")
	}

	return outputDoc, nil
}

func getCertFromPEMString(pemString string) (*x509.Certificate, error) {
	pubkey := fmt.Sprintf("-----BEGIN PUBLIC KEY-----\n%s\n-----END PUBLIC KEY-----",
		pemString)

	pemBlock, _ := pem.Decode([]byte(pubkey))
	if pemBlock == nil {
		return &x509.Certificate{}, errors.New("Could not parse Public Key PEM")
	}
	if pemBlock.Type != "PUBLIC KEY" {
		return &x509.Certificate{}, errors.New("Found wrong key type")
	}

	cert, err := x509.ParseCertificate(pemBlock.Bytes)
	return cert, err
}

func processTransform(transform *etree.Element,
	docIn *etree.Document) (docOut *etree.Document, err error) {

	transformAlgoURI := transform.SelectAttrValue("Algorithm", "")
	if transformAlgoURI == "" {
		return nil, errors.New("signedxml: unable to find Algorithm in Transform")
	}

	transformAlgo, ok := CanonicalizationAlgorithms[transformAlgoURI]
	if !ok {
		return nil, fmt.Errorf("signedxml: unable to find matching transform"+
			"algorithm for %s in CanonicalizationAlgorithms", transformAlgoURI)
	}

	var transformContent string

	if transform.ChildElements() != nil {
		tDoc := etree.NewDocument()
		tDoc.SetRoot(transform.Copy())
		transformContent, err = tDoc.WriteToString()
		if err != nil {
			return nil, err
		}
	}

	docString, err := docIn.WriteToString()
	if err != nil {
		return nil, err
	}

	docString, err = transformAlgo.Process(docString, transformContent)
	if err != nil {
		return nil, err
	}

	docOut = etree.NewDocument()
	docOut.ReadFromString(docString)

	return docOut, nil
}

func calculateHash(reference *etree.Element, doc *etree.Document) (string, error) {
	digestMethodElement := reference.SelectElement("DigestMethod")
	if digestMethodElement == nil {
		return "", errors.New("signedxml: unable to find DigestMethod")
	}

	digestMethodURI := digestMethodElement.SelectAttrValue("Algorithm", "")
	if digestMethodURI == "" {
		return "", errors.New("signedxml: unable to find Algorithm in DigestMethod")
	}

	digestAlgo, ok := hashAlgorithms[digestMethodURI]
	if !ok {
		return "", fmt.Errorf("signedxml: unable to find matching hash"+
			"algorithm for %s in hashAlgorithms", digestMethodURI)
	}

	doc.WriteSettings.CanonicalEndTags = true
	doc.WriteSettings.CanonicalText = true
	doc.WriteSettings.CanonicalAttrVal = true

	h := digestAlgo.New()
	docBytes, err := doc.WriteToBytes()
	if err != nil {
		return "", err
	}

	//ioutil.WriteFile("C:/Temp/SignedXML/Suspect.xml", docBytes, 0644)
	//s, _ := doc.WriteToString()
	//logger.Println(s)

	h.Write(docBytes)
	d := h.Sum(nil)
	calculatedValue := base64.StdEncoding.EncodeToString(d)

	return calculatedValue, nil
}
