// +build acceptance

package acceptance

import (
	"github.com/mailgun/mailgun-go"
	"testing"
)

func TestWebhookCRUD(t *testing.T) {
	domain := reqEnv(t, "MG_DOMAIN")
	apiKey := reqEnv(t, "MG_API_KEY")
	mg := mailgun.NewMailgun(domain, apiKey, "")

	var countHooks = func() int {
		hooks, err := mg.GetWebhooks()
		if err != nil {
			t.Fatal(err)
		}
		return len(hooks)
	}

	hookCount := countHooks()

	err := mg.CreateWebhook("deliver", "http://www.example.com")
	if err != nil {
		t.Fatal(err)
	}
	defer func() {
		err = mg.DeleteWebhook("deliver")
		if err != nil {
			t.Fatal(err)
		}

		newCount := countHooks()
		if newCount != hookCount {
			t.Fatalf("Expected %d routes defined; got %d", hookCount, newCount)
		}
	}()

	newCount := countHooks()
	if newCount <= hookCount {
		t.Fatalf("Expected %d routes defined; got %d", hookCount+1, newCount)
	}

	theURL, err := mg.GetWebhookByType("deliver")
	if err != nil {
		t.Fatal(err)
	}
	if theURL != "http://www.example.com" {
		t.Fatalf("Expected http://www.example.com, got %#v", theURL)
	}

	err = mg.UpdateWebhook("deliver", "http://api.example.com")
	if err != nil {
		t.Fatal(err)
	}

	hooks, err := mg.GetWebhooks()
	if err != nil {
		t.Fatal(err)
	}
	if hooks["deliver"] != "http://api.example.com" {
		t.Fatalf("Expected http://api.example.com, got %#v", hooks["deliver"])
	}
}
