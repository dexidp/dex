// +build acceptance

package acceptance

import (
	"crypto/rand"
	"fmt"
	"github.com/mailgun/mailgun-go"
	"testing"
)

func TestGetDomains(t *testing.T) {
	domain := reqEnv(t, "MG_DOMAIN")
	apiKey := reqEnv(t, "MG_API_KEY")
	mg := mailgun.NewMailgun(domain, apiKey, "")
	n, domains, err := mg.GetDomains(mailgun.DefaultLimit, mailgun.DefaultSkip)
	if err != nil {
		t.Fatal(err)
	}
	fmt.Printf("TestGetDomains: %d domains retrieved\n", n)
	for _, d := range domains {
		fmt.Printf("TestGetDomains: %#v\n", d)
	}
}

func TestGetSingleDomain(t *testing.T) {
	domain := reqEnv(t, "MG_DOMAIN")
	apiKey := reqEnv(t, "MG_API_KEY")
	mg := mailgun.NewMailgun(domain, apiKey, "")
	_, domains, err := mg.GetDomains(mailgun.DefaultLimit, mailgun.DefaultSkip)
	if err != nil {
		t.Fatal(err)
	}
	dr, rxDnsRecords, txDnsRecords, err := mg.GetSingleDomain(domains[0].Name)
	if err != nil {
		t.Fatal(err)
	}

	fmt.Printf("TestGetSingleDomain: %#v\n", dr)
	for _, rxd := range rxDnsRecords {
		fmt.Printf("TestGetSingleDomains:   %#v\n", rxd)
	}
	for _, txd := range txDnsRecords {
		fmt.Printf("TestGetSingleDomains:   %#v\n", txd)
	}
}

func TestGetSingleDomainNotExist(t *testing.T) {
	domain := reqEnv(t, "MG_DOMAIN")
	apiKey := reqEnv(t, "MG_API_KEY")
	mg := mailgun.NewMailgun(domain, apiKey, "")
	_, _, _, err := mg.GetSingleDomain(randomString(32, "com.edu.org.")+".com")
	if err == nil {
		t.Fatal("Did not expect a domain to exist")
	}
	ure, ok := err.(*mailgun.UnexpectedResponseError)
	if !ok {
		t.Fatal("Expected UnexpectedResponseError")
	}
	if ure.Actual != 404 {
		t.Fatalf("Expected 404 response code; got %d", ure.Actual)
	}
}

func TestAddDeleteDomain(t *testing.T) {
	// First, we need to add the domain.
	domain := reqEnv(t, "MG_DOMAIN")
	apiKey := reqEnv(t, "MG_API_KEY")
	mg := mailgun.NewMailgun(domain, apiKey, "")
	randomDomainName := randomString(16, "DOMAIN") + ".example.com"
	randomPassword := randomString(16, "PASSWD")
	err := mg.CreateDomain(randomDomainName, randomPassword, mailgun.Tag, false)
	if err != nil {
		t.Fatal(err)
	}

	// Next, we delete it.
	err = mg.DeleteDomain(randomDomainName)
	if err != nil {
		t.Fatal(err)
	}
}

// randomString generates a string of given length, but random content.
// All content will be within the ASCII graphic character set.
// (Implementation from Even Shaw's contribution on
// http://stackoverflow.com/questions/12771930/what-is-the-fastest-way-to-generate-a-long-random-string-in-go).
func randomString(n int, prefix string) string {
	const alphanum = "0123456789ABCDEFGHIJKLMNOPQRSTUVWXYZabcdefghijklmnopqrstuvwxyz"
	var bytes = make([]byte, n)
	rand.Read(bytes)
	for i, b := range bytes {
		bytes[i] = alphanum[b%byte(len(alphanum))]
	}
	return prefix + string(bytes)
}
