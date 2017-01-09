package dsig

import (
	"bytes"
	"crypto/rsa"
	"crypto/x509"
	"encoding/base64"
	"encoding/pem"
	"errors"
	"fmt"
	"regexp"

	"github.com/beevik/etree"
)

var uriRegexp = regexp.MustCompile("^#[a-zA-Z_][\\w.-]*$")

type ValidationContext struct {
	CertificateStore X509CertificateStore
	IdAttribute      string
	Clock            *Clock
}

func NewDefaultValidationContext(certificateStore X509CertificateStore) *ValidationContext {
	return &ValidationContext{
		CertificateStore: certificateStore,
		IdAttribute:      DefaultIdAttr,
	}
}

// TODO(russell_h): More flexible namespace support. This might barely work.
func inNamespace(el *etree.Element, ns string) bool {
	for _, attr := range el.Attr {
		if attr.Value == ns {
			if attr.Space == "" && attr.Key == "xmlns" {
				return el.Space == ""
			} else if attr.Space == "xmlns" {
				return el.Space == attr.Key
			}
		}
	}

	return false
}

func childPath(space, tag string) string {
	if space == "" {
		return "./" + tag
	} else {
		return "./" + space + ":" + tag
	}
}

// The RemoveElement method on etree.Element isn't recursive...
func recursivelyRemoveElement(tree, el *etree.Element) bool {
	if tree.RemoveChild(el) != nil {
		return true
	}

	for _, child := range tree.Child {
		if childElement, ok := child.(*etree.Element); ok {
			if recursivelyRemoveElement(childElement, el) {
				return true
			}
		}
	}

	return false
}

// transform applies the passed set of transforms to the specified root element.
//
// The functionality of transform is currently very limited and purpose-specific.
//
// NOTE(russell_h): Ideally this wouldn't mutate the root passed to it, and would
// instead return a copy. Unfortunately copying the tree makes it difficult to
// correctly locate the signature. I'm opting, for now, to simply mutate the root
// parameter.
func (ctx *ValidationContext) transform(root, sig *etree.Element, transforms []*etree.Element) (*etree.Element, Canonicalizer, error) {
	if len(transforms) != 2 {
		return nil, nil, errors.New("Expected Enveloped and C14N transforms")
	}

	var canonicalizer Canonicalizer

	for _, transform := range transforms {
		algo := transform.SelectAttr(AlgorithmAttr)
		if algo == nil {
			return nil, nil, errors.New("Missing Algorithm attribute")
		}

		switch AlgorithmID(algo.Value) {
		case EnvelopedSignatureAltorithmId:
			if !recursivelyRemoveElement(root, sig) {
				return nil, nil, errors.New("Error applying canonicalization transform: Signature not found")
			}

		case CanonicalXML10ExclusiveAlgorithmId:
			var prefixList string
			ins := transform.FindElement(childPath("", InclusiveNamespacesTag))
			if ins != nil {
				prefixListEl := ins.SelectAttr(PrefixListAttr)
				if prefixListEl != nil {
					prefixList = prefixListEl.Value
				}
			}

			canonicalizer = MakeC14N10ExclusiveCanonicalizerWithPrefixList(prefixList)

		case CanonicalXML11AlgorithmId:
			canonicalizer = MakeC14N11Canonicalizer()

		default:
			return nil, nil, errors.New("Unknown Transform Algorithm: " + algo.Value)
		}
	}

	if canonicalizer == nil {
		return nil, nil, errors.New("Expected canonicalization transform")
	}

	return root, canonicalizer, nil
}

func (ctx *ValidationContext) digest(el *etree.Element, digestAlgorithmId string, canonicalizer Canonicalizer) ([]byte, error) {
	data, err := canonicalizer.Canonicalize(el)
	if err != nil {
		return nil, err
	}

	digestAlgorithm, ok := digestAlgorithmsByIdentifier[digestAlgorithmId]
	if !ok {
		return nil, errors.New("Unknown digest algorithm: " + digestAlgorithmId)
	}

	hash := digestAlgorithm.New()
	_, err = hash.Write(data)
	if err != nil {
		return nil, err
	}

	return hash.Sum(nil), nil
}

func (ctx *ValidationContext) verifySignedInfo(signatureElement *etree.Element, canonicalizer Canonicalizer, signatureMethodId string, cert *x509.Certificate, sig []byte) error {
	signedInfo := signatureElement.FindElement(childPath(signatureElement.Space, SignedInfoTag))
	if signedInfo == nil {
		return errors.New("Missing SignedInfo")
	}

	// Any attributes from the 'Signature' element must be pushed down into the 'SignedInfo' element before it is canonicalized
	for _, attr := range signatureElement.Attr {
		signedInfo.CreateAttr(attr.Space+":"+attr.Key, attr.Value)
	}

	// Canonicalize the xml
	canonical, err := canonicalizer.Canonicalize(signedInfo)
	if err != nil {
		return err
	}

	signatureAlgorithm, ok := signatureMethodsByIdentifier[signatureMethodId]
	if !ok {
		return errors.New("Unknown signature method: " + signatureMethodId)
	}

	hash := signatureAlgorithm.New()
	_, err = hash.Write(canonical)
	if err != nil {
		return err
	}

	hashed := hash.Sum(nil)

	pubKey, ok := cert.PublicKey.(*rsa.PublicKey)
	if !ok {
		return errors.New("Invalid public key")
	}

	// Verify that the private key matching the public key from the cert was what was used to sign the 'SignedInfo' and produce the 'SignatureValue'
	err = rsa.VerifyPKCS1v15(pubKey, signatureAlgorithm, hashed[:], sig)
	if err != nil {
		return err
	}

	return nil
}

func (ctx *ValidationContext) validateSignature(el *etree.Element, cert *x509.Certificate) (*etree.Element, error) {
	el = el.Copy()

	// Verify the document minus the signedInfo against the 'DigestValue'
	// Find the 'Signature' element
	sig := el.FindElement(SignatureTag)

	if sig == nil {
		return nil, errors.New("Missing Signature")
	}

	if !inNamespace(sig, Namespace) {
		return nil, errors.New("Signature element is in the wrong namespace")
	}

	// Get the 'SignedInfo' element
	signedInfo := sig.FindElement(childPath(sig.Space, SignedInfoTag))
	if signedInfo == nil {
		return nil, errors.New("Missing SignedInfo")
	}

	reference := signedInfo.FindElement(childPath(sig.Space, ReferenceTag))
	if reference == nil {
		return nil, errors.New("Missing Reference")
	}

	transforms := reference.FindElement(childPath(sig.Space, TransformsTag))
	if transforms == nil {
		return nil, errors.New("Missing Transforms")
	}

	uri := reference.SelectAttr("URI")
	if uri == nil {
		// TODO(russell_h): It is permissible to leave this out. We should be
		// able to fall back to finding the referenced element some other way.
		return nil, errors.New("Reference is missing URI attribute")
	}

	if !uriRegexp.MatchString(uri.Value) {
		return nil, errors.New("Invalid URI: " + uri.Value)
	}

	// Get the element referenced in the 'SignedInfo'
	referencedElement := el.FindElement(fmt.Sprintf("//[@%s='%s']", ctx.IdAttribute, uri.Value[1:]))
	if referencedElement == nil {
		return nil, errors.New("Unable to find referenced element: " + uri.Value)
	}

	// Perform all transformations listed in the 'SignedInfo'
	// Basically, this means removing the 'SignedInfo'
	transformed, canonicalizer, err := ctx.transform(referencedElement, sig, transforms.ChildElements())
	if err != nil {
		return nil, err
	}

	digestMethod := reference.FindElement(childPath(sig.Space, DigestMethodTag))
	if digestMethod == nil {
		return nil, errors.New("Missing DigestMethod")
	}

	digestValue := reference.FindElement(childPath(sig.Space, DigestValueTag))
	if digestValue == nil {
		return nil, errors.New("Missing DigestValue")
	}

	digestAlgorithmAttr := digestMethod.SelectAttr(AlgorithmAttr)
	if digestAlgorithmAttr == nil {
		return nil, errors.New("Missing DigestMethod Algorithm attribute")
	}

	// Digest the transformed XML and compare it to the 'DigestValue' from the 'SignedInfo'
	digest, err := ctx.digest(transformed, digestAlgorithmAttr.Value, canonicalizer)
	if err != nil {
		return nil, err
	}

	decodedDigestValue, err := base64.StdEncoding.DecodeString(digestValue.Text())
	if err != nil {
		return nil, err
	}

	if !bytes.Equal(digest, decodedDigestValue) {
		return nil, errors.New("Signature could not be verified")
	}

	//Verify the signed info
	signatureMethod := signedInfo.FindElement(childPath(sig.Space, SignatureMethodTag))
	if signatureMethod == nil {
		return nil, errors.New("Missing SignatureMethod")
	}

	signatureMethodAlgorithmAttr := signatureMethod.SelectAttr(AlgorithmAttr)
	if digestAlgorithmAttr == nil {
		return nil, errors.New("Missing SignatureMethod Algorithm attribute")
	}

	// Decode the 'SignatureValue' so we can compare against it
	signatureValue := sig.FindElement(childPath(sig.Space, SignatureValueTag))
	if signatureValue == nil {
		return nil, errors.New("Missing SignatureValue")
	}

	decodedSignature, err := base64.StdEncoding.DecodeString(signatureValue.Text())

	if err != nil {
		return nil, errors.New("Could not decode signature")
	}
	// Actually verify the 'SignedInfo' was signed by a trusted source
	err = ctx.verifySignedInfo(sig, canonicalizer, signatureMethodAlgorithmAttr.Value, cert, decodedSignature)
	if err != nil {
		return nil, err
	}

	return transformed, nil
}

func contains(roots []*x509.Certificate, cert *x509.Certificate) bool {
	for _, root := range roots {
		if root.Equal(cert) {
			return true
		}
	}
	return false
}

func (ctx *ValidationContext) verifyCertificate(el *etree.Element) (*x509.Certificate, error) {
	now := ctx.Clock.Now()
	el = el.Copy()

	idAttr := el.SelectAttr(DefaultIdAttr)
	if idAttr == nil || idAttr.Value == "" {
		return nil, errors.New("Missing ID attribute")
	}

	signatureElements := el.FindElements("//" + SignatureTag)
	var signatureElement *etree.Element

	// Find the Signature element that references the whole Response element
	for _, e := range signatureElements {
		e2 := e.Copy()

		signedInfo := e2.FindElement(childPath(e2.Space, SignedInfoTag))
		if signedInfo == nil {
			return nil, errors.New("Missing SignedInfo")
		}

		referenceElement := signedInfo.FindElement(childPath(e2.Space, ReferenceTag))
		if referenceElement == nil {
			return nil, errors.New("Missing Reference Element")
		}

		uriAttr := referenceElement.SelectAttr(URIAttr)
		if uriAttr == nil || uriAttr.Value == "" {
			return nil, errors.New("Missing URI attribute")
		}

		if uriAttr.Value[1:] == idAttr.Value {
			signatureElement = e
			break
		}
	}

	if signatureElement == nil {
		return nil, errors.New("Missing signature referencing the top-level element")
	}

	// Get the x509 element from the signature
	x509Element := signatureElement.FindElement("//" + childPath(signatureElement.Space, X509CertificateTag))
	if x509Element == nil {
		return nil, errors.New("Missing x509 Element")
	}

	x509Text := "-----BEGIN CERTIFICATE-----\n" + x509Element.Text() + "\n-----END CERTIFICATE-----"
	block, _ := pem.Decode([]byte(x509Text))
	if block == nil {
		return nil, errors.New("Failed to parse certificate PEM")
	}

	cert, err := x509.ParseCertificate(block.Bytes)
	if err != nil {
		return nil, err
	}

	roots, err := ctx.CertificateStore.Certificates()
	if err != nil {
		return nil, err
	}

	// Verify that the certificate is one we trust
	if !contains(roots, cert) {
		return nil, errors.New("Could not verify certificate against trusted certs")
	}

	if now.Before(cert.NotBefore) || now.After(cert.NotAfter) {
		return nil, errors.New("Cert is not valid at this time")
	}

	return cert, nil
}

func (ctx *ValidationContext) Validate(el *etree.Element) (*etree.Element, error) {
	cert, err := ctx.verifyCertificate(el)

	if err != nil {
		return nil, err
	}

	return ctx.validateSignature(el, cert)
}
