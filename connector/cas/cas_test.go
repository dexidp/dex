package cas

import (
	"fmt"
	"log/slog"
	"math/rand"
	"net/http"
	"net/url"
	"os"
	"reflect"
	"testing"
	"time"

	"github.com/dexidp/dex/connector"
	"github.com/pkg/errors"
	"gopkg.in/yaml.v3"
)

type tcase struct {
	xml     string
	mapping map[string]string
	id      connector.Identity
	err     string
}

func TestOpen(t *testing.T) {
	configSection := `
portal: https://example.org/cas
mapping:
  username: name
  preferred_username: username
  email: email
  groups: affiliation
`

	var config Config
	if err := yaml.Unmarshal([]byte(configSection), &config); err != nil {
		t.Errorf("parse config: %v", err)
		return
	}

	conn, err := config.Open("cas", slog.Default())
	if err != nil {
		t.Errorf("open connector: %v", err)
		return
	}

	casConnector, _ := conn.(*casConnector)
	if casConnector.portal.String() != config.Portal {
		t.Errorf("expected portal %q, got %q", config.Portal, casConnector.portal.String())
		return
	}
	if !reflect.DeepEqual(casConnector.mapping, config.Mapping) {
		t.Errorf("expected mapping %v, got %v", config.Mapping, casConnector.mapping)
		return
	}
}

func TestCAS(t *testing.T) {
	callback := "https://dex.example.org/dex/callback"
	casURL, _ := url.Parse("https://example.org/cas")
	scope := connector.Scopes{Groups: true}

	cases := []tcase{{
		xml: "testdata/cas_success.xml",
		mapping: map[string]string{
			"username":           "name",
			"preferred_username": "username",
			"email":              "email",
		},
		id: connector.Identity{
			UserID:            "123456",
			Username:          "jdoe",
			PreferredUsername: "jdoe",
			Email:             "jdoe@example.org",
			EmailVerified:     true,
			Groups:            []string{"A", "B"},
			ConnectorData:     nil,
		},
		err: "",
	}, {
		xml: "testdata/cas_success.xml",
		mapping: map[string]string{
			"username":           "name",
			"preferred_username": "username",
			"email":              "email",
			"groups":             "affiliation",
		},
		id: connector.Identity{
			UserID:            "123456",
			Username:          "jdoe",
			PreferredUsername: "jdoe",
			Email:             "jdoe@example.org",
			EmailVerified:     true,
			Groups:            []string{"staff", "faculty"},
			ConnectorData:     nil,
		},
		err: "",
	}, {
		xml:     "testdata/cas_failure.xml",
		mapping: map[string]string{},
		id:      connector.Identity{},
		err:     "INVALID_TICKET: Ticket ST-1856339-aA5Yuvrxzpv8Tau1cYQ7 not recognized",
	}}

	seed := rand.NewSource(time.Now().UnixNano())
	for _, tc := range cases {
		ticket := fmt.Sprintf("ST-%d", seed.Int63())
		state := fmt.Sprintf("%d", seed.Int63())

		conn := &casConnector{
			portal:     casURL,
			mapping:    tc.mapping,
			logger:     slog.Default(),
			pathSuffix: "/cas",
			client: &http.Client{
				Transport: &mockTransport{
					ticket: ticket,
					file:   tc.xml,
				},
			},
		}

		// login
		login, err := conn.LoginURL(scope, callback, state)
		if err != nil {
			t.Errorf("get login url: %v", err)
			return
		}
		loginURL, err := url.Parse(login)
		if err != nil {
			t.Errorf("parse login url: %v", err)
			return
		}

		// cas server
		queryService := loginURL.Query().Get("service")
		serviceURL, err := url.Parse(queryService)
		if err != nil {
			t.Errorf("parse service url: %v", err)
			return
		}
		serviceQueryState := serviceURL.Query().Get("state")
		if serviceQueryState != state {
			t.Errorf("state: expected %#v, got %#v", state, serviceQueryState)
			return
		}
		req, _ := http.NewRequest(http.MethodGet, queryService, nil)
		q := req.URL.Query()
		q.Set("ticket", ticket)
		req.URL.RawQuery = q.Encode()

		// validate
		id, err := conn.HandleCallback(scope, req)
		if err != nil {
			if c := errors.Cause(err); c != nil && tc.err != "" && c.Error() == tc.err {
				continue
			}
			t.Errorf("handle callback: %v", err)
			return
		}
		if !reflect.DeepEqual(id, tc.id) {
			t.Errorf("identity: expected %#v, got %#v", tc.id, id)
			return
		}
	}
}

type mockTransport struct {
	ticket string
	file   string
}

func (f *mockTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	file, err := os.Open(f.file)
	if err != nil {
		return nil, err
	}

	if ticket := req.URL.Query().Get("ticket"); ticket != f.ticket {
		return nil, fmt.Errorf("ticket: expected %#v, got %#v", f.ticket, ticket)
	}

	return &http.Response{
		StatusCode: http.StatusOK,
		Body:       file,
		Header: http.Header{
			"Content-Type": []string{"text/xml"},
		},
		Request: req,
	}, nil
}
