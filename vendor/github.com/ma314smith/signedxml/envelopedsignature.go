package signedxml

import (
	"errors"

	"github.com/beevik/etree"
)

// EnvelopedSignature implements the CanonicalizationAlgorithm
// interface and is used for processing the
// http://www.w3.org/2000/09/xmldsig#enveloped-signature transform
// algorithm
type EnvelopedSignature struct{}

// Process is called to transfrom the XML using the EnvelopedSignature
// algorithm
func (e EnvelopedSignature) Process(inputXML string,
	transformXML string) (outputXML string, err error) {

	doc := etree.NewDocument()
	doc.ReadFromString(inputXML)
	sig := doc.FindElement(".//Signature")
	if sig == nil {
		return "", errors.New("signedxml: unable to find Signature node")
	}

	sigParent := sig.Parent()
	elem := sigParent.RemoveChild(sig)
	if elem == nil {
		return "", errors.New("signedxml: unable to remove Signature element")
	}

	docString, err := doc.WriteToString()
	if err != nil {
		return "", err
	}
	//logger.Println(docString)
	return docString, nil
}
