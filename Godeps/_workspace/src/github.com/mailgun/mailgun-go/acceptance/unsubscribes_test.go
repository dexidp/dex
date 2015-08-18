// +build acceptance

package acceptance

import (
	"fmt"
	mailgun "github.com/mailgun/mailgun-go"
	"os"
	"testing"
	"text/tabwriter"
)

func TestGetUnsubscribes(t *testing.T) {
	domain := reqEnv(t, "MG_DOMAIN")
	apiKey := reqEnv(t, "MG_API_KEY")
	mg := mailgun.NewMailgun(domain, apiKey, "")
	n, us, err := mg.GetUnsubscribes(mailgun.DefaultLimit, mailgun.DefaultSkip)
	if err != nil {
		t.Fatal(err)
	}
	fmt.Printf("Received %d out of %d unsubscribe records.\n", len(us), n)
	if len(us) > 0 {
		tw := &tabwriter.Writer{}
		tw.Init(os.Stdout, 2, 8, 2, ' ', 0)
		fmt.Fprintln(tw, "ID\tAddress\tCreated At\tTag\t")
		for _, u := range us {
			fmt.Fprintf(tw, "%s\t%s\t%s\t%s\t\n", u.ID, u.Address, u.CreatedAt, u.Tag)
		}
		tw.Flush()
	}
}

func TestGetUnsubscriptionByAddress(t *testing.T) {
	domain := reqEnv(t, "MG_DOMAIN")
	apiKey := reqEnv(t, "MG_API_KEY")
	email := reqEnv(t, "MG_EMAIL_ADDR")
	mg := mailgun.NewMailgun(domain, apiKey, "")
	n, us, err := mg.GetUnsubscribesByAddress(email)
	if err != nil {
		t.Fatal(err)
	}
	fmt.Printf("Received %d out of %d unsubscribe records.\n", len(us), n)
	if len(us) > 0 {
		tw := &tabwriter.Writer{}
		tw.Init(os.Stdout, 2, 8, 2, ' ', 0)
		fmt.Fprintln(tw, "ID\tAddress\tCreated At\tTag\t")
		for _, u := range us {
			fmt.Fprintf(tw, "%s\t%s\t%s\t%s\t\n", u.ID, u.Address, u.CreatedAt, u.Tag)
		}
		tw.Flush()
	}
}

func TestCreateDestroyUnsubscription(t *testing.T) {
	domain := reqEnv(t, "MG_DOMAIN")
	apiKey := reqEnv(t, "MG_API_KEY")
	email := reqEnv(t, "MG_EMAIL_ADDR")
	mg := mailgun.NewMailgun(domain, apiKey, "")

	// Create unsubscription record
	err := mg.Unsubscribe(email, "*")
	if err != nil {
		t.Fatal(err)
	}

	// Destroy the unsubscription record
	err = mg.RemoveUnsubscribe(email)
	if err != nil {
		t.Fatal(err)
	}
}
