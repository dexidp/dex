package signedxml

import (
	"crypto/x509"
	"encoding/pem"
	"fmt"
	"io/ioutil"
	"os"
	"testing"

	"github.com/beevik/etree"
	. "github.com/smartystreets/goconvey/convey"
)

func TestSign(t *testing.T) {
	Convey("Given an XML, certificate, and RSA key", t, func() {
		pemString, _ := ioutil.ReadFile("./testdata/rsa.crt")
		pemBlock, _ := pem.Decode([]byte(pemString))
		cert, _ := x509.ParseCertificate(pemBlock.Bytes)

		pemString, _ = ioutil.ReadFile("./testdata/rsa.key")
		pemBlock, _ = pem.Decode([]byte(pemString))
		key, _ := x509.ParsePKCS1PrivateKey(pemBlock.Bytes)

		xml, _ := ioutil.ReadFile("./testdata/nosignature.xml")

		Convey("When generating the signature", func() {
			signer, _ := NewSigner(string(xml))
			xmlStr, err := signer.Sign(key)
			Convey("Then no error occurs", func() {
				So(err, ShouldBeNil)
			})
			Convey("And the signature should be valid", func() {
				validator, _ := NewValidator(xmlStr)
				validator.Certificates = append(validator.Certificates, *cert)
				err := validator.Validate()
				So(err, ShouldBeNil)
			})
		})
	})
}

func TestValidate(t *testing.T) {
	Convey("Given valid signed XML", t, func() {
		cases := map[string]string{
			"(WSFed BBAuth Metadata)":    "./testdata/bbauth-metadata.xml",
			"(SAML External NS)":         "./testdata/saml-external-ns.xml",
			"(Signature w/Inclusive NS)": "./testdata/signature-with-inclusivenamespaces.xml",
			"(SAML)":                     "./testdata/valid-saml.xml",
			// this one doesn't appear to follow the spec... ( http://webservices20.blogspot.co.il/2013/06/validating-windows-mobile-app-store.html)
			//"(Windows Store Signature)":  "./testdata/windows-store-signature.xml",
			"(WSFed Generic Metadata)": "./testdata/wsfed-metadata.xml",
		}

		for description, test := range cases {
			Convey(fmt.Sprintf("When Validate is called %s", description), func() {
				xmlFile, err := os.Open(test)
				if err != nil {
					fmt.Println("Error opening file:", err)
				}
				defer xmlFile.Close()
				xmlBytes, _ := ioutil.ReadAll(xmlFile)
				validator, _ := NewValidator(string(xmlBytes))
				err = validator.Validate()
				Convey("Then no error occurs", func() {
					So(err, ShouldBeNil)
					So(validator.SigningCert().PublicKey, ShouldNotBeNil)
				})
			})
		}

		Convey("When Validate is called with an external Signature", func() {
			xmlFile, _ := os.Open("./testdata/bbauth-metadata.xml")
			sig := `<Signature xmlns="http://www.w3.org/2000/09/xmldsig#"><SignedInfo><CanonicalizationMethod Algorithm="http://www.w3.org/2001/10/xml-exc-c14n#"/><SignatureMethod Algorithm="http://www.w3.org/2001/04/xmldsig-more#rsa-sha256"/><Reference URI="#_69b42076-409e-4476-af41-339962e49427"><Transforms><Transform Algorithm="http://www.w3.org/2000/09/xmldsig#enveloped-signature"/><Transform Algorithm="http://www.w3.org/2001/10/xml-exc-c14n#"/></Transforms><DigestMethod Algorithm="http://www.w3.org/2001/04/xmlenc#sha256"/><DigestValue>LPHoiAkLmA/TGIuVbgpwlFLXL+ymEBc7TS0fC9/PTQU=</DigestValue></Reference></SignedInfo><SignatureValue>d2CXq9GEeDKvMxdpxtTRKQ8TGeSWhJOVPs8LMD0ObeE1t/YGiAm9keorMiki4laxbWqAuOmwHK3qNHogRFgkIYi3fnuFBzMrahXf0n3A5PRXXW1m768Z92GKV09pGuygKUXCtXzwq0seDi6PnzMCJFzFXGQWnum0paa8Oz+6425Sn0zO0fT3ttp3AXeXGyNXwYPYcX1iEMB7klUlyiz2hmn8ngCIbTkru7uIeyPmQ5WD4SS/qQaL4yb3FZibXoe/eRXrbkG1NAJCw9OWw0jsvWncE1rKFaqEMbz21fXSDhh3Ls+p9yVf+xbCrpkT0FMqjTHpNRvccMPZe/hDGrHV7Q==</SignatureValue><KeyInfo><X509Data><X509Certificate>MIIDNzCCAh+gAwIBAgIQQVK+d/vLK4ZNMDk15HGUoTANBgkqhkiG9w0BAQ0FADAoMSYwJAYDVQQDEx1CbGFja2JhdWQgQXV0aGVudGljYXRpb24gMjAyMjAeFw0wMDAxMDEwNDAwMDBaFw0yMjAxMDEwNDAwMDBaMCgxJjAkBgNVBAMTHUJsYWNrYmF1ZCBBdXRoZW50aWNhdGlvbiAyMDIyMIIBIjANBgkqhkiG9w0BAQEFAAOCAQ8AMIIBCgKCAQEArgByjSPVvP4DLf/l7QRz7G7Dhkdns0QjWslnWejHlFIezfkJ4NGPp0+5CRCFYBqAb7DhqyK77Ek5xdzmwgYb1X6GD6UDltWvN5BBFAw69I6/K0WjguFUxk19T7xdc8vTCNAMi+6Ys49O3EBNnI2fiqDoBdMjUTud1F04QY3N2rZWkjMrHV+CnzhoUwqsO/ABWrDbkPzBXdOOIbsKH0k0IP8q2+35pe1y2nxtB9f1fCyCmbUH2HINMHahDmxxanTW5Jy14yD/HSRTFQF9JMTeglomWq5q9VPx0NjsEJR+B5IkRCTf75LoYrrr/fvQm3aummmYPdHauXCBrcm0moX4ywIDAQABo10wWzBZBgNVHQEEUjBQgBDCHOfardZfhltQSbLqsukZoSowKDEmMCQGA1UEAxMdQmxhY2tiYXVkIEF1dGhlbnRpY2F0aW9uIDIwMjKCEEFSvnf7yyuGTTA5NeRxlKEwDQYJKoZIhvcNAQENBQADggEBADrOhfRiynRKGD7EHohpPrltFScJ9+QErYMhEvteqh3C48T99uKgDY8wTqv+PI08QUSZuhmmF2d+W7aRBo3t8ZZepIXCwDaKo/oUp2h5Y9O3vyGDguq5ptgDTmPNYDCwWtdt0TtQYeLtCQTJVbYByWL0eT+KdzQOkAi48cPEOObSc9Biga7LTCcbCVPeJlYzmHDQUhzBt2jcy5BGvmZloI5SsoZvve6ug74qNq8IJMyzJzUp3kRuB0ruKIioSDi1lc783LDT3LSXyIbOGw/vHBEBY4Ax7FK8CqXJ2TsYqVsyo8QypqXDnveLcgK+PNEAhezhxC9hyV8j1I8pfF72ABE=</X509Certificate></X509Data></KeyInfo></Signature>`
			defer xmlFile.Close()
			xmlBytes, _ := ioutil.ReadAll(xmlFile)
			validator := Validator{}
			validator.SetXML(string(xmlBytes))
			validator.SetSignature(sig)
			err := validator.Validate()
			Convey("Then no error occurs", func() {
				So(err, ShouldBeNil)
				So(validator.SigningCert().PublicKey, ShouldNotBeNil)
			})
		})

		Convey("When Validate is called with an external certificate and root xmlns", func() {
			xmlFile, _ := os.Open("./testdata/rootxmlns.xml")
			pemString, _ := ioutil.ReadFile("./testdata/rootxmlns.crt")
			pemBlock, _ := pem.Decode([]byte(pemString))
			cert, _ := x509.ParseCertificate(pemBlock.Bytes)
			defer xmlFile.Close()
			xmlBytes, _ := ioutil.ReadAll(xmlFile)
			validator := Validator{}
			validator.SetXML(string(xmlBytes))
			validator.Certificates = append(validator.Certificates, *cert)
			err := validator.Validate()
			Convey("Then no error occurs", func() {
				So(err, ShouldBeNil)
				So(validator.SigningCert().PublicKey, ShouldNotBeNil)
			})
		})
	})

	Convey("Given invalid signed XML", t, func() {
		cases := map[string]string{
			"(Changed Content)":        "./testdata/invalid-signature-changed content.xml",
			"(Non-existing Reference)": "./testdata/invalid-signature-non-existing-reference.xml",
			"(Wrong Sig Value)":        "./testdata/invalid-signature-signature-value.xml",
		}

		for description, test := range cases {
			Convey(fmt.Sprintf("When Validate is called %s", description), func() {
				xmlFile, err := os.Open(test)
				if err != nil {
					fmt.Println("Error opening file:", err)
				}
				defer xmlFile.Close()
				xmlBytes, _ := ioutil.ReadAll(xmlFile)
				validator, _ := NewValidator(string(xmlBytes))

				err = validator.Validate()
				Convey("Then an error occurs", func() {
					So(err, ShouldNotBeNil)
					So(err.Error(), ShouldContainSubstring, "signedxml:")
				})
			})
		}
	})
}

func TestEnvelopedSignatureProcess(t *testing.T) {
	Convey("Given a document without a Signature elemement", t, func() {
		doc := "<doc></doc>"
		Convey("When Process is called", func() {
			envSig := EnvelopedSignature{}
			_, err := envSig.Process(doc, "")
			Convey("Then an error occurs", func() {
				So(err, ShouldNotBeNil)
				So(err.Error(), ShouldContainSubstring, "signedxml:")
			})
		})
	})

	Convey("Given an unparented signature element", t, func() {
		doc := "<Signatrue></Signature>"
		Convey("When Process is called", func() {
			envSig := EnvelopedSignature{}
			_, err := envSig.Process(doc, "")
			Convey("Then an error occurs", func() {
				So(err, ShouldNotBeNil)
				So(err.Error(), ShouldContainSubstring, "signedxml:")
			})
		})
	})
}

func TestSignatureDataParsing(t *testing.T) {
	Convey("Given a document without a Signature elemement", t, func() {
		doc := etree.NewDocument()
		doc.CreateElement("root")
		Convey("When parseEnvelopedSignature is called", func() {
			sigData := signatureData{xml: doc}
			err := sigData.parseEnvelopedSignature()
			Convey("Then an error occurs", func() {
				So(err, ShouldNotBeNil)
				So(err.Error(), ShouldContainSubstring, "signedxml:")
			})
		})
	})

	Convey("Given a document without a SignedInfo elemement", t, func() {
		doc := etree.NewDocument()
		doc.CreateElement("root").CreateElement("Signature")
		sigData := signatureData{xml: doc}
		sigData.parseEnvelopedSignature()
		Convey("When parseSignedInfo is called", func() {
			err := sigData.parseSignedInfo()
			Convey("Then an error occurs", func() {
				So(err, ShouldNotBeNil)
				So(err.Error(), ShouldContainSubstring, "signedxml:")
			})
		})
	})

	Convey("Given a document without a SignatureValue elemement", t, func() {
		doc := etree.NewDocument()
		doc.CreateElement("root").CreateElement("Signature")
		sigData := signatureData{xml: doc}
		sigData.parseEnvelopedSignature()
		Convey("When parseSigValue is called", func() {
			err := sigData.parseSigValue()
			Convey("Then an error occurs", func() {
				So(err, ShouldNotBeNil)
				So(err.Error(), ShouldContainSubstring, "signedxml:")
			})
		})
	})

	Convey("Given a document without a SignatureMethod elemement", t, func() {
		doc := etree.NewDocument()
		doc.CreateElement("root").CreateElement("Signature").CreateElement("SignedInfo")
		sigData := signatureData{xml: doc}
		sigData.parseEnvelopedSignature()
		sigData.parseSignedInfo()
		Convey("When parseSigAlgorithm is called", func() {
			err := sigData.parseSigAlgorithm()
			Convey("Then an error occurs", func() {
				So(err, ShouldNotBeNil)
				So(err.Error(), ShouldContainSubstring, "signedxml:")
			})
		})
	})

	Convey("Given a document without a SignatureMethod Algorithm element", t, func() {
		doc := etree.NewDocument()
		doc.CreateElement("root").CreateElement("Signature").CreateElement("SignedInfo").CreateElement("SignatureMethod")
		sigData := signatureData{xml: doc}
		sigData.parseEnvelopedSignature()
		sigData.parseSignedInfo()
		Convey("When parseSigAlgorithm is called", func() {
			err := sigData.parseSigAlgorithm()
			Convey("Then an error occurs", func() {
				So(err, ShouldNotBeNil)
				So(err.Error(), ShouldContainSubstring, "signedxml:")
			})
		})
	})

	Convey("Given a document without a CanonicalizationMethod elemement", t, func() {
		doc := etree.NewDocument()
		doc.CreateElement("root").CreateElement("Signature").CreateElement("SignedInfo")
		sigData := signatureData{xml: doc}
		sigData.parseEnvelopedSignature()
		sigData.parseSignedInfo()
		Convey("When parseCanonAlgorithm is called", func() {
			err := sigData.parseCanonAlgorithm()
			Convey("Then an error occurs", func() {
				So(err, ShouldNotBeNil)
				So(err.Error(), ShouldContainSubstring, "signedxml:")
			})
		})
	})

	Convey("Given a document without a CanonicalizationMethod Algorithm element", t, func() {
		doc := etree.NewDocument()
		doc.CreateElement("root").CreateElement("Signature").CreateElement("SignedInfo").CreateElement("CanonicalizationMethod")
		sigData := signatureData{xml: doc}
		sigData.parseEnvelopedSignature()
		sigData.parseSignedInfo()
		Convey("When parseCanonAlgorithm is called", func() {
			err := sigData.parseCanonAlgorithm()
			Convey("Then an error occurs", func() {
				So(err, ShouldNotBeNil)
				So(err.Error(), ShouldContainSubstring, "signedxml:")
			})
		})
	})
}

var example31Input = `<?xml version="1.0"?>

<?xml-stylesheet   href="doc.xsl" type="text/xsl"   ?>

<!DOCTYPE doc SYSTEM "doc.dtd">

<doc>Hello, world!<!-- Comment 1 --></doc>

<?pi-without-data     ?>

<!-- Comment 2 -->

<!-- Comment 3 -->`

var example31Output = `<?xml-stylesheet href="doc.xsl" type="text/xsl"   ?>
<doc>Hello, world!</doc>
<?pi-without-data?>`

var example31OutputWithComments = `<?xml-stylesheet href="doc.xsl" type="text/xsl"   ?>
<doc>Hello, world!<!-- Comment 1 --></doc>
<?pi-without-data?>
<!-- Comment 2 -->
<!-- Comment 3 -->`

var example32Input = `<doc>
   <clean>   </clean>
   <dirty>   A   B   </dirty>
   <mixed>
      A
      <clean>   </clean>
      B
      <dirty>   A   B   </dirty>
      C
   </mixed>
</doc>`

var example32Output = `<doc>
   <clean>   </clean>
   <dirty>   A   B   </dirty>
   <mixed>
      A
      <clean>   </clean>
      B
      <dirty>   A   B   </dirty>
      C
   </mixed>
</doc>`

var example33Input = `<!DOCTYPE doc [<!ATTLIST e9 attr CDATA "default">]>
<doc>
   <e1   />
   <e2   ></e2>
   <e3   name = "elem3"   id="elem3"   />
   <e4   name="elem4"   id="elem4"   ></e4>
   <e5 a:attr="out" b:attr="sorted" attr2="all" attr="I'm" xmlns:b="http://www.ietf.org" xmlns:a="http://www.w3.org" xmlns="http://example.org"/>
   <e6 xmlns="" xmlns:a="http://www.w3.org">
      <e7 xmlns="http://www.ietf.org">
         <e8 xmlns="" xmlns:a="http://www.w3.org">
            <e9 xmlns="" xmlns:a="http://www.ietf.org"/>
         </e8>
      </e7>
   </e6>
</doc>`

var example33Output = `<doc>
   <e1></e1>
   <e2></e2>
   <e3 id="elem3" name="elem3"></e3>
   <e4 id="elem4" name="elem4"></e4>
   <e5 xmlns="http://example.org" xmlns:a="http://www.w3.org" xmlns:b="http://www.ietf.org" attr="I'm" attr2="all" b:attr="sorted" a:attr="out"></e5>
   <e6 xmlns:a="http://www.w3.org">
      <e7 xmlns="http://www.ietf.org">
         <e8 xmlns="">
            <e9 xmlns:a="http://www.ietf.org" attr="default"></e9>
         </e8>
      </e7>
   </e6>
</doc>`

var example34Input = `<!DOCTYPE doc [
<!ATTLIST normId id ID #IMPLIED>
<!ATTLIST normNames attr NMTOKENS #IMPLIED>
]>
<doc>
   <text>First line&#x0d;&#10;Second line</text>
   <value>&#x32;</value>
   <compute><![CDATA[value>"0" && value<"10" ?"valid":"error"]]></compute>
   <compute expr='value>"0" &amp;&amp; value&lt;"10" ?"valid":"error"'>valid</compute>
   <norm attr=' &apos;   &#x20;&#13;&#xa;&#9;   &apos; '/>
   <normNames attr='   A   &#x20;&#13;&#xa;&#9;   B   '/>
</doc>`

var example34Output = `<doc>
   <text>First line&#xD;
Second line</text>
   <value>2</value>
   <compute>value&gt;"0" &amp;&amp; value&lt;"10" ?"valid":"error"</compute>
   <compute expr="value>&quot;0&quot; &amp;&amp; value&lt;&quot;10&quot; ?&quot;valid&quot;:&quot;error&quot;">valid</compute>
   <norm attr=" '    &#xD;&#xA;&#x9;   ' "></norm>
   <normNames attr="A &#xD;&#xA;&#x9; B"></normNames>
</doc>`

// modified to not include DTD processing. still tests for whitespace treated as
// CDATA
var example34ModifiedOutput = `<doc>
   <text>First line&#xD;
Second line</text>
   <value>2</value>
   <compute>value&gt;"0" &amp;&amp; value&lt;"10" ?"valid":"error"</compute>
   <compute expr="value>&quot;0&quot; &amp;&amp; value&lt;&quot;10&quot; ?&quot;valid&quot;:&quot;error&quot;">valid</compute>
   <norm attr=" '    &#xD;&#xA;&#x9;   ' "></norm>
   <normNames attr="   A    &#xD;&#xA;&#x9;   B   "></normNames>
</doc>`

var example35Input = `<!DOCTYPE doc [
<!ATTLIST doc attrExtEnt ENTITY #IMPLIED>
<!ENTITY ent1 "Hello">
<!ENTITY ent2 SYSTEM "world.txt">
<!ENTITY entExt SYSTEM "earth.gif" NDATA gif>
<!NOTATION gif SYSTEM "viewgif.exe">
]>
<doc attrExtEnt="entExt">
   &ent1;, &ent2;!
</doc>

<!-- Let world.txt contain "world" (excluding the quotes) -->`

var example35Output = `<doc attrExtEnt="entExt">
   Hello, world!
</doc>`

var example36Input = `<?xml version="1.0" encoding="ISO-8859-1"?>
<doc>&#169;</doc>`

var example36Output = "<doc>\u00A9</doc>"

var example37Input = `<!DOCTYPE doc [
<!ATTLIST e2 xml:space (default|preserve) 'preserve'>
<!ATTLIST e3 id ID #IMPLIED>
]>
<doc xmlns="http://www.ietf.org" xmlns:w3c="http://www.w3.org">
   <e1>
      <e2 xmlns="">
         <e3 id="E3"/>
      </e2>
   </e1>
</doc>`

var example37SubsetExpression = `<!-- Evaluate with declaration xmlns:ietf="http://www.ietf.org" -->

(//. | //@* | //namespace::*)
[
   self::ietf:e1 or (parent::ietf:e1 and not(self::text() or self::e2))
   or
   count(id("E3")|ancestor-or-self::node()) = count(ancestor-or-self::node())
]`

var example37Output = `<e1 xmlns="http://www.ietf.org" xmlns:w3c="http://www.w3.org"><e3 xmlns="" id="E3" xml:space="preserve"></e3></e1>`

type exampleXML struct {
	input        string
	output       string
	withComments bool
	expression   string
}

// test examples from the spec (www.w3.org/TR/2001/REC-xml-c14n-20010315#Examples)
func TestCanonicalizationExamples(t *testing.T) {
	Convey("Given XML Input", t, func() {
		cases := map[string]exampleXML{
			"(Example 3.1 w/o Comments)": exampleXML{input: example31Input, output: example31Output},
			"(Example 3.1 w/Comments)":   exampleXML{input: example31Input, output: example31OutputWithComments, withComments: true},
			"(Example 3.2)":              exampleXML{input: example32Input, output: example32Output},
			// 3.3 is for Canonical NOT ExclusiveCanonical (one of the exceptions here: http://www.w3.org/TR/xml-exc-c14n/#sec-Specification)
			//"(Example 3.3)":              exampleXML{input: example33Input, output: example33Output},
			"(Example 3.4)": exampleXML{input: example34Input, output: example34ModifiedOutput},
			//"(Example 3.5)": exampleXML{input: example35Input, output: example35Output},
			// 3.6 will work, but requires a change to the etree package first:
			// http://stackoverflow.com/questions/6002619/unmarshal-an-iso-8859-1-xml-input-in-go
			//"(Example 3.6)": exampleXML{input: example36Input, output: example36Output},
			//"(Example 3.7)": exampleXML{input: example37Input, output: example37Output, expression: example37SubsetExpression},
		}
		for description, test := range cases {
			Convey(fmt.Sprintf("When transformed %s", description), func() {
				transform := ExclusiveCanonicalization{WithComments: test.withComments}
				resultXML, err := transform.Process(test.input, "")
				Convey("Then the resulting XML match the example output", func() {
					So(err, ShouldBeNil)
					So(resultXML, ShouldEqual, test.output)
				})
			})
		}
	})
}
