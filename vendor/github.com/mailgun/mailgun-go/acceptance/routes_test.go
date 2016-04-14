// +build acceptance

package acceptance

import (
	"github.com/mailgun/mailgun-go"
	"testing"
)

func TestRouteCRUD(t *testing.T) {
	domain := reqEnv(t, "MG_DOMAIN")
	apiKey := reqEnv(t, "MG_API_KEY")
	mg := mailgun.NewMailgun(domain, apiKey, "")

	var countRoutes = func() int {
		count, _, err := mg.GetRoutes(mailgun.DefaultLimit, mailgun.DefaultSkip)
		if err != nil {
			t.Fatal(err)
		}
		return count
	}

	routeCount := countRoutes()

	newRoute, err := mg.CreateRoute(mailgun.Route{
		Priority:    1,
		Description: "Sample Route",
		Expression:  "match_recipient(\".*@samples.mailgun.org\")",
		Actions: []string{
			"forward(\"http://example.com/messages/\")",
			"stop()",
		},
	})
	if err != nil {
		t.Fatal(err)
	}
	if newRoute.ID == "" {
		t.Fatal("I expected the route created to have an ID associated with it.")
	}
	defer func() {
		err = mg.DeleteRoute(newRoute.ID)
		if err != nil {
			t.Fatal(err)
		}

		newCount := countRoutes()
		if newCount != routeCount {
			t.Fatalf("Expected %d routes defined; got %d", routeCount, newCount)
		}
	}()

	newCount := countRoutes()
	if newCount <= routeCount {
		t.Fatalf("Expected %d routes defined; got %d", routeCount+1, newCount)
	}

	theRoute, err := mg.GetRouteByID(newRoute.ID)
	if err != nil {
		t.Fatal(err)
	}
	if ((newRoute.Priority) != (theRoute.Priority)) ||
		((newRoute.Description) != (theRoute.Description)) ||
		((newRoute.Expression) != (theRoute.Expression)) ||
		(len(newRoute.Actions) != len(theRoute.Actions)) ||
		((newRoute.CreatedAt) != (theRoute.CreatedAt)) ||
		((newRoute.ID) != (theRoute.ID)) {
		t.Fatalf("Expected %#v, got %#v", newRoute, theRoute)
	}
	for i, action := range newRoute.Actions {
		if action != theRoute.Actions[i] {
			t.Fatalf("Expected %#v, got %#v", newRoute, theRoute)
		}
	}

	changedRoute, err := mg.UpdateRoute(newRoute.ID, mailgun.Route{
		Priority: 2,
	})
	if err != nil {
		t.Fatal(err)
	}
	if changedRoute.Priority != 2 {
		t.Fatalf("Expected a priority of 2; got %d", changedRoute.Priority)
	}
	if len(changedRoute.Actions) != 2 {
		t.Fatalf("Expected actions to not be touched; got %d entries now", len(changedRoute.Actions))
	}
}
