// +build acceptance

package acceptance

import (
	"fmt"
	mailgun "github.com/mailgun/mailgun-go"
	"os"
	"testing"
	"text/tabwriter"
)

func TestGetCredentials(t *testing.T) {
	domain := reqEnv(t, "MG_DOMAIN")
	apiKey := reqEnv(t, "MG_API_KEY")
	mg := mailgun.NewMailgun(domain, apiKey, "")
	n, cs, err := mg.GetCredentials(mailgun.DefaultLimit, mailgun.DefaultSkip)
	if err != nil {
		t.Fatal(err)
	}
	tw := &tabwriter.Writer{}
	tw.Init(os.Stdout, 2, 8, 2, ' ', 0)
	fmt.Fprintf(tw, "Login\tCreated At\t\n")
	for _, c := range cs {
		fmt.Fprintf(tw, "%s\t%s\t\n", c.Login, c.CreatedAt)
	}
	tw.Flush()
	fmt.Printf("%d credentials listed out of %d\n", len(cs), n)
}

func TestCreateDeleteCredentials(t *testing.T) {
	domain := reqEnv(t, "MG_DOMAIN")
	apiKey := reqEnv(t, "MG_API_KEY")
	mg := mailgun.NewMailgun(domain, apiKey, "")
	randomPassword := randomString(16, "pw")
	randomID := randomString(16, "usr")
	randomLogin := fmt.Sprintf("%s@%s", randomID, domain)

	err := mg.CreateCredential(randomLogin, randomPassword)
	if err != nil {
		t.Fatal(err)
	}

	err = mg.ChangeCredentialPassword(randomID, randomString(16, "pw2"))
	if err != nil {
		t.Fatal(err)
	}

	err = mg.DeleteCredential(randomID)
	if err != nil {
		t.Fatal(err)
	}
}
