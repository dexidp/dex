// +build acceptance

package acceptance

import (
	"fmt"
	mailgun "github.com/mailgun/mailgun-go"
	"os"
	"testing"
	"text/tabwriter"
)

func TestGetStats(t *testing.T) {
	domain := reqEnv(t, "MG_DOMAIN")
	apiKey := reqEnv(t, "MG_API_KEY")
	mg := mailgun.NewMailgun(domain, apiKey, "")

	totalCount, stats, err := mg.GetStats(-1, -1, nil, "sent", "opened")
	if err != nil {
		t.Fatal(err)
	}

	fmt.Printf("Total Count: %d\n", totalCount)
	tw := tabwriter.NewWriter(os.Stdout, 2, 8, 2, ' ', tabwriter.AlignRight)
	fmt.Fprintf(tw, "Id\tEvent\tCreatedAt\tTotalCount\t\n")
	for _, stat := range stats {
		fmt.Fprintf(tw, "%s\t%s\t%s\t%d\t\n", stat.Id, stat.Event, stat.CreatedAt, stat.TotalCount)
	}
	tw.Flush()
}

func TestDeleteTag(t *testing.T) {
	domain := reqEnv(t, "MG_DOMAIN")
	apiKey := reqEnv(t, "MG_API_KEY")
	mg := mailgun.NewMailgun(domain, apiKey, "")
	err := mg.DeleteTag("newsletter")
	if err != nil {
		t.Fatal(err)
	}
}
