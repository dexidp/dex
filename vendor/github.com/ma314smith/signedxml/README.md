## signedxml

[![Build Status](https://travis-ci.org/ma314smith/signedxml.svg?branch=master)](https://travis-ci.org/ma314smith/signedxml)
[![GoDoc](https://godoc.org/github.com/ma314smith/signedxml?status.svg)](https://godoc.org/github.com/ma314smith/signedxml)

The signedxml package transforms and validates signed xml documents. The main use case is to support Single Sign On protocols like SAML and WS-Federation.

Other packages that provide similar functionality rely on C libraries, which makes them difficult to run across platforms without significant configuration.  `signedxml` is written in pure go, and can be easily used on any platform.

### Install

`go get github.com/ma314smith/signedxml`

### Included Algorithms

- Hashes
  - http://www.w3.org/2001/04/xmldsig-more#md5
  - http://www.w3.org/2000/09/xmldsig#sha1
  - http://www.w3.org/2001/04/xmldsig-more#sha224
  - http://www.w3.org/2001/04/xmlenc#sha256
  - http://www.w3.org/2001/04/xmldsig-more#sha384
  - http://www.w3.org/2001/04/xmlenc#sha512
  - http://www.w3.org/2001/04/xmlenc#ripemd160


- Signatures
  - http://www.w3.org/2001/04/xmldsig-more#rsa-md2
  - http://www.w3.org/2001/04/xmldsig-more#rsa-md5
  - http://www.w3.org/2000/09/xmldsig#rsa-sha1
  - http://www.w3.org/2001/04/xmldsig-more#rsa-sha256
  - http://www.w3.org/2001/04/xmldsig-more#rsa-sha384
  - http://www.w3.org/2001/04/xmldsig-more#rsa-sha512
  - http://www.w3.org/2000/09/xmldsig#dsa-sha1
  - http://www.w3.org/2000/09/xmldsig#dsa-sha256
  - http://www.w3.org/2001/04/xmldsig-more#ecdsa-sha1
  - http://www.w3.org/2001/04/xmldsig-more#ecdsa-sha256
  - http://www.w3.org/2001/04/xmldsig-more#ecdsa-sha384
  - http://www.w3.org/2001/04/xmldsig-more#ecdsa-sha512


- Canonicalization Methods/Transforms
  - http://www.w3.org/2000/09/xmldsig#enveloped-signature
  - http://www.w3.org/2001/10/xml-exc-c14n#
  - http://www.w3.org/2001/10/xml-exc-c14n#WithComments

### Examples

#### Validating signed XML
If your signed xml contains the signature and certificate, then you can just pass in the xml and call `Validate()`.
```go
validator, err := signedxml.NewValidator(`<YourXMLString></YourXMLString>`)
err = validator.Validate()
```
`Validate()` verifies the DigestValue and SignatureValue in the xml document. If the error value is `nil`, then the signed xml is valid.

The x509.Certificate that was successfully used to validate the xml will be available by calling:
```go
validator.SigningCert()
```
You can then verify that you trust the certificate. You can optionally supply your trusted certificates ahead of time by assigning them to the `Certificates` property of the `Validator` object, which is an x509.Certificate array.

#### Using an external Signature
If you need to specify an external Signature, you can use the `SetSignature()` function to assign it:
```go
validator.SetSignature(<`Signature></Signature>`)
```

#### Generating signed XML
It is expected that your XML contains the Signature element with all the parameters set (except DigestValue and SignatureValue).
```go
signer, err := signedxml.NewSigner(`<YourXMLString></YourXMLString`)
signedXML, err := signer.Sign(`*rsa.PrivateKey object`)
```
`Sign()` will generate the DigestValue and SignatureValue, populate it in the XML, and return the signed XML string.

#### Implementing custom transforms
Additional Transform algorithms can be included by adding to the CanonicalizationAlgorithms map.  This interface will need to be implemented:
```go
type CanonicalizationAlgorithm interface {
	Process(inputXML string, transformXML string) (outputXML string, err error)
}
```
Simple Example:
```go
type NoChangeCanonicalization struct{}

func (n NoChangeCanonicalization) Process(inputXML string,
	transformXML string) (outputXML string, err error) {
	return inputXML, nil
}

signedxml.CanonicalizationAlgorithms["http://myTranform"] = NoChangeCanonicalization{}
```

See `envelopedsignature.go` and `exclusivecanonicalization.go` for examples of actual implementations.

### Contributions
Contributions are welcome. Just fork the repo and send a pull request.
