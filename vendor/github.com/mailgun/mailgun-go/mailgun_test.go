package mailgun

import (
	"net/http"
	"strconv"
	"testing"
)

const domain = "valid-mailgun-domain"
const apiKey = "valid-mailgun-api-key"
const publicApiKey = "valid-mailgun-public-api-key"

func TestMailgun(t *testing.T) {
	m := NewMailgun(domain, apiKey, publicApiKey)

	if domain != m.Domain() {
		t.Fatal("Domain not equal!")
	}

	if apiKey != m.ApiKey() {
		t.Fatal("ApiKey not equal!")
	}

	if publicApiKey != m.PublicApiKey() {
		t.Fatal("PublicApiKey not equal!")
	}

	if http.DefaultClient != m.Client() {
		t.Fatal("HTTP client not default!")
	}
	client := new(http.Client)
	m.SetClient(client)
	if client != m.Client() {
		t.Fatal("HTTP client not equal!")
	}
}

func TestBounceGetCode(t *testing.T) {
	b1 := &Bounce{
		CreatedAt: "blah",
		code:      123,
		Address:   "blort",
		Error:     "bletch",
	}
	c, err := b1.GetCode()
	if err != nil {
		t.Fatal(err)
	}
	if c != 123 {
		t.Fatal("Expected 123; got ", c)
	}

	b2 := &Bounce{
		CreatedAt: "blah",
		code:      "456",
		Address:   "blort",
		Error:     "Bletch",
	}
	c, err = b2.GetCode()
	if err != nil {
		t.Fatal(err)
	}
	if c != 456 {
		t.Fatal("Expected 456; got ", c)
	}

	b3 := &Bounce{
		CreatedAt: "blah",
		code:      "456H",
		Address:   "blort",
		Error:     "Bletch",
	}
	c, err = b3.GetCode()
	e, ok := err.(*strconv.NumError)
	if !ok && e != nil {
		t.Fatal("Expected a syntax error in numeric conversion: got ", err)
	}
}
