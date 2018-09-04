package xmlsig

import (
	"encoding/xml"
	"testing"
)

type Root struct {
	XMLName xml.Name `xml:"tns root"`
	B       string   `xml:"b,attr"`
	A       string   `xml:"http://someotherns/for/attr a,attr"`
	C       string   `xml:"anotherns/be a,attr"`
	Child   Child
}

type Child struct {
	XMLName xml.Name `xml:"tns child"`
	Data    string   `xml:",chardata"`
}

func TestCanonicalization(t *testing.T) {
	element := &Root{B: "1", A: "2", C: "3", Child: Child{Data: "data"}}
	// Go's default encoder would produce the following
	// <root xmlns="tns" b="1" xmlns:attr="http://someotherns/for/attr" attr:a="2" xmlns:be="anotherns/be" be:a="3"><child xmlns="tns">data</child></root>
	// It should produce
	// <root xmlns="tns" xmlns:attr="http://someotherns/for/attr" xmlns:be="anotherns/be" b="1" be:a="3"  attr:a="2"><child>data</child></root>
	data, _, err := canonicalize(element)
	if err != nil {
		t.Fatal(err)
	}
	actual := string(data)
	expected := `<root xmlns="tns" xmlns:attr="http://someotherns/for/attr" xmlns:be="anotherns/be" b="1" be:a="3" attr:a="2"><child>data</child></root>`
	if actual != expected {
		t.Fatalf("expected output of %s but got %s", expected, actual)
	}
}
