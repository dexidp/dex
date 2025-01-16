package cloudfoundry

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"reflect"
	"strings"
	"testing"

	"github.com/dexidp/dex/connector"
)

func TestOpen(t *testing.T) {
	testServer := testSetup()
	defer testServer.Close()

	conn := newConnector(t, testServer.URL)

	expectEqual(t, conn.clientID, "test-client")
	expectEqual(t, conn.clientSecret, "secret")
	expectEqual(t, conn.redirectURI, testServer.URL+"/callback")
}

func TestHandleCallback(t *testing.T) {
	testServer := testSetup()
	defer testServer.Close()

	cloudfoundryConn := &cloudfoundryConnector{
		tokenURL:         fmt.Sprintf("%s/token", testServer.URL),
		authorizationURL: fmt.Sprintf("%s/authorize", testServer.URL),
		userInfoURL:      fmt.Sprintf("%s/userinfo", testServer.URL),
		apiURL:           testServer.URL,
		clientSecret:     "secret",
		clientID:         "test-client",
		redirectURI:      "localhost:8080/sky/dex/callback",
		httpClient:       http.DefaultClient,
	}

	req, err := http.NewRequest("GET", testServer.URL, nil)
	expectEqual(t, err, nil)

	t.Run("CallbackWithGroupsScope", func(t *testing.T) {
		identity, err := cloudfoundryConn.HandleCallback(connector.Scopes{Groups: true}, req)
		expectEqual(t, err, nil)

		expectEqual(t, len(identity.Groups), 24)
		expectEqual(t, identity.Groups[0], "some-org-guid-1")
		expectEqual(t, identity.Groups[1], "some-org-guid-2")
		expectEqual(t, identity.Groups[2], "some-org-guid-3")
		expectEqual(t, identity.Groups[3], "some-org-guid-4")
		expectEqual(t, identity.Groups[4], "some-org-name-1")
		expectEqual(t, identity.Groups[5], "some-org-name-1:some-space-name-1")
		expectEqual(t, identity.Groups[6], "some-org-name-1:some-space-name-1:auditor")
		expectEqual(t, identity.Groups[7], "some-org-name-1:some-space-name-1:developer")
		expectEqual(t, identity.Groups[8], "some-org-name-1:some-space-name-1:manager")
		expectEqual(t, identity.Groups[9], "some-org-name-2")
		expectEqual(t, identity.Groups[10], "some-org-name-2:some-space-name-2")
		expectEqual(t, identity.Groups[11], "some-org-name-2:some-space-name-2:auditor")
		expectEqual(t, identity.Groups[12], "some-org-name-2:some-space-name-2:developer")
		expectEqual(t, identity.Groups[13], "some-org-name-2:some-space-name-2:manager")
		expectEqual(t, identity.Groups[14], "some-org-name-3")
		expectEqual(t, identity.Groups[15], "some-org-name-4")
		expectEqual(t, identity.Groups[16], "some-space-guid-1")
		expectEqual(t, identity.Groups[17], "some-space-guid-1:auditor")
		expectEqual(t, identity.Groups[18], "some-space-guid-1:developer")
		expectEqual(t, identity.Groups[19], "some-space-guid-1:manager")
		expectEqual(t, identity.Groups[20], "some-space-guid-2")
		expectEqual(t, identity.Groups[21], "some-space-guid-2:auditor")
		expectEqual(t, identity.Groups[22], "some-space-guid-2:developer")
		expectEqual(t, identity.Groups[23], "some-space-guid-2:manager")
	})

	t.Run("CallbackWithoutGroupsScope", func(t *testing.T) {
		identity, err := cloudfoundryConn.HandleCallback(connector.Scopes{}, req)

		expectEqual(t, err, nil)
		expectEqual(t, identity.UserID, "12345")
		expectEqual(t, identity.Username, "test-user")
	})

	t.Run("CallbackWithOfflineAccessScope", func(t *testing.T) {
		identity, err := cloudfoundryConn.HandleCallback(connector.Scopes{OfflineAccess: true}, req)

		expectEqual(t, err, nil)
		expectNotEqual(t, len(identity.ConnectorData), 0)

		cData := connectorData{}
		err = json.Unmarshal(identity.ConnectorData, &cData)

		expectEqual(t, err, nil)
		expectNotEqual(t, cData.AccessToken, "")
	})
}

func testSpaceHandler(reqURL string) (result map[string]interface{}) {
	if strings.Contains(reqURL, "spaces?page=2&per_page=50") {
		result = map[string]interface{}{
			"pagination": map[string]interface{}{
				"next": map[string]interface{}{
					"href": nil,
				},
			},
			"resources": []map[string]interface{}{
				{
					"guid": "some-space-guid-2",
					"name": "some-space-name-2",
					"relationships": map[string]interface{}{
						"user": nil,
						"organization": map[string]interface{}{
							"data": map[string]interface{}{
								"guid": "some-org-guid-2",
							},
						},
						"space": nil,
					},
				},
			},
		}
	} else {
		nextURL := fmt.Sprintf("%s?page=2&per_page=50", reqURL)
		result = map[string]interface{}{
			"pagination": map[string]interface{}{
				"next": map[string]interface{}{
					"href": nextURL,
				},
			},
			"resources": []map[string]interface{}{
				{
					"guid": "some-space-guid-1",
					"name": "some-space-name-1",
					"relationships": map[string]interface{}{
						"user": nil,
						"organization": map[string]interface{}{
							"data": map[string]interface{}{
								"guid": "some-org-guid-1",
							},
						},
						"space": nil,
					},
				},
			},
		}
	}
	return result
}

func testOrgHandler(reqURL string) (result map[string]interface{}) {
	if strings.Contains(reqURL, "organizations?page=2&per_page=50") {
		result = map[string]interface{}{
			"pagination": map[string]interface{}{
				"next": map[string]interface{}{
					"href": nil,
				},
			},
			"resources": []map[string]interface{}{
				{
					"guid": "some-org-guid-3",
					"name": "some-org-name-3",
					"relationships": map[string]interface{}{
						"user":         nil,
						"organization": nil,
						"space":        nil,
					},
				},
				{
					"guid": "some-org-guid-4",
					"name": "some-org-name-4",
					"relationships": map[string]interface{}{
						"user":         nil,
						"organization": nil,
						"space":        nil,
					},
				},
			},
		}
	} else {
		nextURL := fmt.Sprintf("%s?page=2&per_page=50", reqURL)
		result = map[string]interface{}{
			"pagination": map[string]interface{}{
				"next": map[string]interface{}{
					"href": nextURL,
				},
			},
			"resources": []map[string]interface{}{
				{
					"guid": "some-org-guid-1",
					"name": "some-org-name-1",
					"relationships": map[string]interface{}{
						"user":         nil,
						"organization": nil,
						"space": map[string]interface{}{
							"data": map[string]interface{}{
								"guid": "some-space-guid-1",
							},
						},
					},
				},
				{
					"guid": "some-org-guid-2",
					"name": "some-org-name-2",
					"relationships": map[string]interface{}{
						"user":         nil,
						"organization": nil,
						"space": map[string]interface{}{
							"data": map[string]interface{}{
								"guid": "some-space-guid-2",
							},
						},
					},
				},
			},
		}
	}
	return result
}

func testUserOrgsSpacesHandler(reqURL string) (result map[string]interface{}) {
	if strings.Contains(reqURL, "page=2&per_page=50") {
		result = map[string]interface{}{
			"pagination": map[string]interface{}{
				"next": map[string]interface{}{
					"href": nil,
				},
			},
			"resources": []map[string]interface{}{
				{
					"guid": "some-type-guid-3",
					"type": "organization_user",
					"relationships": map[string]interface{}{
						"user": nil,
						"organization": map[string]interface{}{
							"data": map[string]interface{}{
								"guid": "some-org-guid-3",
							},
						},
						"space": nil,
					},
				},
				{
					"guid": "some-type-guid-4",
					"type": "organization_user",
					"relationships": map[string]interface{}{
						"user": nil,
						"organization": map[string]interface{}{
							"data": map[string]interface{}{
								"guid": "some-org-guid-4",
							},
						},
						"space": nil,
					},
				},
				{
					"guid": "some-type-guid-1",
					"type": "space_manager",
					"relationships": map[string]interface{}{
						"user": nil,
						"organization": map[string]interface{}{
							"data": map[string]interface{}{
								"guid": "some-org-guid-1",
							},
						},
						"space": map[string]interface{}{
							"data": map[string]interface{}{
								"guid": "some-space-guid-1",
							},
						},
					},
				},
				{
					"guid": "some-type-guid-2",
					"type": "space_developer",
					"relationships": map[string]interface{}{
						"user": nil,
						"organization": map[string]interface{}{
							"data": map[string]interface{}{
								"guid": "some-org-guid-2",
							},
						},
						"space": map[string]interface{}{
							"data": map[string]interface{}{
								"guid": "some-space-guid-2",
							},
						},
					},
				},
				{
					"guid": "some-type-guid-2",
					"type": "space_auditor",
					"relationships": map[string]interface{}{
						"user": nil,
						"organization": map[string]interface{}{
							"data": map[string]interface{}{
								"guid": "some-org-guid-2",
							},
						},
						"space": map[string]interface{}{
							"data": map[string]interface{}{
								"guid": "some-space-guid-2",
							},
						},
					},
				},
				{
					"guid": "some-type-guid-2",
					"type": "space_manager",
					"relationships": map[string]interface{}{
						"user": nil,
						"organization": map[string]interface{}{
							"data": map[string]interface{}{
								"guid": "some-org-guid-2",
							},
						},
						"space": map[string]interface{}{
							"data": map[string]interface{}{
								"guid": "some-space-guid-2",
							},
						},
					},
				},
			},
		}
	} else {
		nextURL := fmt.Sprintf("%s?page=2&per_page=50", reqURL)
		result = map[string]interface{}{
			"pagination": map[string]interface{}{
				"next": map[string]interface{}{
					"href": nextURL,
				},
			},
			"resources": []map[string]interface{}{
				{
					"guid": "some-type-guid-1",
					"type": "space_developer",
					"relationships": map[string]interface{}{
						"user": nil,
						"organization": map[string]interface{}{
							"data": map[string]interface{}{
								"guid": "some-org-guid-1",
							},
						},
						"space": map[string]interface{}{
							"data": map[string]interface{}{
								"guid": "some-space-guid-1",
							},
						},
					},
				},
				{
					"guid": "some-type-guid-1",
					"type": "space_auditor",
					"relationships": map[string]interface{}{
						"user": nil,
						"organization": map[string]interface{}{
							"data": map[string]interface{}{
								"guid": "some-org-guid-1",
							},
						},
						"space": map[string]interface{}{
							"data": map[string]interface{}{
								"guid": "some-space-guid-1",
							},
						},
					},
				},
				{
					"guid": "some-type-guid-1",
					"type": "space_manager",
					"relationships": map[string]interface{}{
						"user": nil,
						"organization": map[string]interface{}{
							"data": map[string]interface{}{
								"guid": "some-org-guid-1",
							},
						},
						"space": map[string]interface{}{
							"data": map[string]interface{}{
								"guid": "some-space-guid-1",
							},
						},
					},
				},
				{
					"guid": "some-type-guid-2",
					"type": "space_developer",
					"relationships": map[string]interface{}{
						"user": nil,
						"organization": map[string]interface{}{
							"data": map[string]interface{}{
								"guid": "some-org-guid-2",
							},
						},
						"space": map[string]interface{}{
							"data": map[string]interface{}{
								"guid": "some-space-guid-2",
							},
						},
					},
				},
				{
					"guid": "some-type-guid-2",
					"type": "space_auditor",
					"relationships": map[string]interface{}{
						"user": nil,
						"organization": map[string]interface{}{
							"data": map[string]interface{}{
								"guid": "some-org-guid-2",
							},
						},
						"space": map[string]interface{}{
							"data": map[string]interface{}{
								"guid": "some-space-guid-2",
							},
						},
					},
				},
				{
					"guid": "some-type-guid-2",
					"type": "space_manager",
					"relationships": map[string]interface{}{
						"user": nil,
						"organization": map[string]interface{}{
							"data": map[string]interface{}{
								"guid": "some-org-guid-2",
							},
						},
						"space": map[string]interface{}{
							"data": map[string]interface{}{
								"guid": "some-space-guid-2",
							},
						},
					},
				},
			},
		}
	}
	return result
}

func testSetup() *httptest.Server {
	mux := http.NewServeMux()
	mux.HandleFunc("/token", func(w http.ResponseWriter, r *http.Request) {
		token := "eyJhbGciOiJSUzI1NiIsImtpZCI6ImtleS0xIiwidHlwIjoiSldUIn0.eyJqdGkiOiIxMjk4MTNhZjJiNGM0ZDNhYmYyNjljMzM4OTFkZjNiZCIsInN1YiI6ImNmMWFlODk4LWQ1ODctNDBhYS1hNWRiLTE5ZTY3MjI0N2I1NyIsInNjb3BlIjpbImNsb3VkX2NvbnRyb2xsZXIucmVhZCIsIm9wZW5pZCJdLCJjbGllbnRfaWQiOiJjb25jb3Vyc2UiLCJjaWQiOiJjb25jb3Vyc2UiLCJhenAiOiJjb25jb3Vyc2UiLCJncmFudF90eXBlIjoiYXV0aG9yaXphdGlvbl9jb2RlIiwidXNlcl9pZCI6ImNmMWFlODk4LWQ1ODctNDBhYS1hNWRiLTE5ZTY3MjI0N2I1NyIsIm9yaWdpbiI6InVhYSIsInVzZXJfbmFtZSI6ImFkbWluIiwiZW1haWwiOiJhZG1pbiIsImF1dGhfdGltZSI6MTUyMzM3NDIwNCwicmV2X3NpZyI6IjYxNWJjMTk0IiwiaWF0IjoxNTIzMzc3MTUyLCJleHAiOjE1MjM0MjAzNTIsImlzcyI6Imh0dHBzOi8vdWFhLnN0eXgucHVzaC5nY3AuY2YtYXBwLmNvbS9vYXV0aC90b2tlbiIsInppZCI6InVhYSIsImF1ZCI6WyJjbG91ZF9jb250cm9sbGVyIiwiY29uY291cnNlIiwib3BlbmlkIl19.FslbnwvW0WScVRNK8IWghRX0buXfl6qaI1K7z_dzjPUVrdEyMtaYa3kJI8srA-2G1PjSSEWa_3Vzs_BEnTc3iG0JQWU0XlcjdCdAFTvnmKiHSzffy1O_oGYyH47KXtnZOxHf3rdV_Xgw4XTqPrfKXQxnPemUAJyKf2tjgs3XToGaqqBw-D_2BQVY79kF0_GgksQsViqq1GW0Dur6m2CgBhtc2h1AQGO16izXl3uNbpW6ClhaW43NQXlE4wqtr7kfmxyOigHJb2MSQ3wwPc6pqYdUT6ka_TMqavqbxEJ4QcS6SoEcVsDTmEQ4c8dmWUgXM0AZjd0CaEGTB6FDHxH5sw"
		w.Header().Add("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]string{
			"access_token": token,
		})
	})

	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		url := fmt.Sprintf("http://%s", r.Host)

		json.NewEncoder(w).Encode(map[string]interface{}{
			"links": map[string]interface{}{
				"login": map[string]string{
					"href": url,
				},
			},
		})
	})

	mux.HandleFunc("/.well-known/openid-configuration", func(w http.ResponseWriter, r *http.Request) {
		url := fmt.Sprintf("http://%s", r.Host)

		json.NewEncoder(w).Encode(map[string]string{
			"token_endpoint":         url,
			"authorization_endpoint": url,
			"userinfo_endpoint":      url,
		})
	})

	mux.HandleFunc("/callback", func(w http.ResponseWriter, r *http.Request) {
	})

	mux.HandleFunc("/userinfo", func(w http.ResponseWriter, r *http.Request) {
		json.NewEncoder(w).Encode(map[string]string{
			"user_id":   "12345",
			"user_name": "test-user",
			"email":     "blah-email",
		})
	})

	mux.HandleFunc("/v3/organizations", func(w http.ResponseWriter, r *http.Request) {
		json.NewEncoder(w).Encode(testOrgHandler(r.URL.String()))
	})

	mux.HandleFunc("/v3/spaces", func(w http.ResponseWriter, r *http.Request) {
		json.NewEncoder(w).Encode(testSpaceHandler(r.URL.String()))
	})

	mux.HandleFunc("/v3/roles", func(w http.ResponseWriter, r *http.Request) {
		json.NewEncoder(w).Encode(testUserOrgsSpacesHandler(r.URL.String()))
	})

	return httptest.NewServer(mux)
}

func newConnector(t *testing.T, serverURL string) *cloudfoundryConnector {
	callBackURL := fmt.Sprintf("%s/callback", serverURL)

	testConfig := Config{
		APIURL:             serverURL,
		ClientID:           "test-client",
		ClientSecret:       "secret",
		RedirectURI:        callBackURL,
		InsecureSkipVerify: true,
	}

	log := slog.New(slog.NewTextHandler(io.Discard, &slog.HandlerOptions{}))

	conn, err := testConfig.Open("id", log)
	if err != nil {
		t.Fatal(err)
	}

	cloudfoundryConn, ok := conn.(*cloudfoundryConnector)
	if !ok {
		t.Fatal(errors.New("it is not a cloudfoundry conn"))
	}

	return cloudfoundryConn
}

func expectEqual(t *testing.T, a interface{}, b interface{}) {
	if !reflect.DeepEqual(a, b) {
		t.Fatalf("Expected %+v to equal %+v", a, b)
	}
}

func expectNotEqual(t *testing.T, a interface{}, b interface{}) {
	if reflect.DeepEqual(a, b) {
		t.Fatalf("Expected %+v to NOT equal %+v", a, b)
	}
}
