// +build acceptance

package acceptance

import (
	"fmt"
	mailgun "github.com/mailgun/mailgun-go"
	"testing"
)

func TestGetBounces(t *testing.T) {
	domain := reqEnv(t, "MG_DOMAIN")
	apiKey := reqEnv(t, "MG_API_KEY")
	mg := mailgun.NewMailgun(domain, apiKey, "")
	n, bounces, err := mg.GetBounces(-1, -1)
	if err != nil {
		t.Fatal(err)
	}
	if n > 0 {
		t.Fatal("Expected no bounces for what should be a clean domain.")
	}
	if n != len(bounces) {
		t.Fatalf("Expected length of bounces %d to equal returned length %d", len(bounces), n)
	}
}

func TestGetSingleBounce(t *testing.T) {
	domain := reqEnv(t, "MG_DOMAIN")
	apiKey := reqEnv(t, "MG_API_KEY")
	mg := mailgun.NewMailgun(domain, apiKey, "")
	exampleEmail := fmt.Sprintf("baz@%s", domain)
	_, err := mg.GetSingleBounce(exampleEmail)
	if err == nil {
		t.Fatal("Did not expect a bounce to exist")
	}
	ure, ok := err.(*mailgun.UnexpectedResponseError)
	if !ok {
		t.Fatal("Expected UnexpectedResponseError")
	}
	if ure.Actual != 404 {
		t.Fatalf("Expected 404 response code; got %d", ure.Actual)
	}
}

func TestAddDelBounces(t *testing.T) {
	domain := reqEnv(t, "MG_DOMAIN")
	apiKey := reqEnv(t, "MG_API_KEY")
	mg := mailgun.NewMailgun(domain, apiKey, "")

	// Compute an e-mail address for our domain.

	exampleEmail := fmt.Sprintf("baz@%s", domain)

	// First, basic sanity check.
	// Fail early if we have bounces for a fictitious e-mail address.

	n, _, err := mg.GetBounces(-1, -1)
	if err != nil {
		t.Fatal(err)
	}
	if n > 0 {
		t.Fatal("Expected no bounces for what should be a clean domain.")
	}

	bounce, err := mg.GetSingleBounce(exampleEmail)
	if err == nil {
		t.Fatalf("Expected no bounces for %s", exampleEmail)
	}

	// Add the bounce for our address.

	err = mg.AddBounce(exampleEmail, "550", "TestAddDelBounces-generated error")
	if err != nil {
		t.Fatal(err)
	}

	// We should now have one bounce listed when we query the API.

	n, bounces, err := mg.GetBounces(-1, -1)
	if err != nil {
		t.Fatal(err)
	}
	if n != 1 {
		t.Fatal("Expected one bounce for this domain.")
	}
	if bounces[0].Address != exampleEmail {
		t.Fatalf("Expected bounce for address %s; got %s", exampleEmail, bounces[0].Address)
	}

	bounce, err = mg.GetSingleBounce(exampleEmail)
	if err != nil {
		t.Fatal(err)
	}
	if bounce.CreatedAt == "" {
		t.Fatalf("Expected at least one bounce for %s", exampleEmail)
	}

	// Delete it.  This should put us back the way we were.

	err = mg.DeleteBounce(exampleEmail)
	if err != nil {
		t.Fatal(err)
	}

	// Make sure we're back to the way we were.

	n, _, err = mg.GetBounces(-1, -1)
	if err != nil {
		t.Fatal(err)
	}
	if n > 0 {
		t.Fatal("Expected no bounces for what should be a clean domain.")
	}

	_, err = mg.GetSingleBounce(exampleEmail)
	if err == nil {
		t.Fatalf("Expected no bounces for %s", exampleEmail)
	}
}
