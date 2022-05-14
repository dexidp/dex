package google

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"

	"github.com/sirupsen/logrus"
	"github.com/stretchr/testify/assert"
	"google.golang.org/api/option"
)

func testSetup(t *testing.T) *httptest.Server {
	mux := http.NewServeMux()
	// TODO: mock calls
	// mux.HandleFunc("/admin/directory/v1/groups", func(w http.ResponseWriter, r *http.Request) {
	// 	w.Header().Add("Content-Type", "application/json")
	// 	json.NewEncoder(w).Encode(&admin.Groups{
	// 		Groups: []*admin.Group{},
	// 	})
	// })
	return httptest.NewServer(mux)
}

func newConnector(config *Config, serverURL string) (*googleConnector, error) {
	log := logrus.New()
	conn, err := config.Open("id", log, option.WithEndpoint(serverURL))
	if err != nil {
		return nil, err
	}

	googleConn, ok := conn.(*googleConnector)
	if !ok {
		return nil, fmt.Errorf("failed to convert to googleConnector")
	}
	return googleConn, nil
}

func tempServiceAccountKey() (string, error) {
	fd, err := os.CreateTemp("", "google_service_account_key")
	if err != nil {
		return "", err
	}
	defer fd.Close()
	err = json.NewEncoder(fd).Encode(map[string]string{
		"type":                 "service_account",
		"project_id":           "abc",
		"private_key_id":       "abc",
		"private_key":          "-----BEGIN PRIVATE KEY-----\nabc\n-----END PRIVATE KEY-----\n",
		"client_id":            "abc",
		"client_x509_cert_url": "localhost",
	})
	return fd.Name(), err
}

func TestOpen(t *testing.T) {
	ts := testSetup(t)
	defer ts.Close()

	serviceAccountFilePath, err := tempServiceAccountKey()
	assert.Nil(t, err)

	for name, expected := range map[string]struct {
		serviceAccount bool
		config         *Config
		connector      *googleConnector

		// TODO: switch from fmt.Errorf to error wrapping for better
		// error checking in tests
		err string
	}{
		"missing_admin_email": {
			config: &Config{
				ClientID:     "testClient",
				ClientSecret: "testSecret",
				RedirectURI:  ts.URL + "/callback",
				Scopes:       []string{"openid", "groups"},
			},
			err: "requires adminEmail",
		},
		"workload_identity": {
			config: &Config{
				ClientID:     "testClient",
				ClientSecret: "testSecret",
				RedirectURI:  ts.URL + "/callback",
				Scopes:       []string{"openid", "groups"},
				AdminEmail:   "foo@bar.com",
			},
			err: "",
		},
		"service_account_key_not_found": {
			config: &Config{
				ClientID:               "testClient",
				ClientSecret:           "testSecret",
				RedirectURI:            ts.URL + "/callback",
				Scopes:                 []string{"openid", "groups"},
				AdminEmail:             "foo@bar.com",
				ServiceAccountFilePath: "not_found.json",
			},
			err: "error reading credentials",
		},
		"service_account_key_valid": {
			config: &Config{
				ClientID:               "testClient",
				ClientSecret:           "testSecret",
				RedirectURI:            ts.URL + "/callback",
				Scopes:                 []string{"openid", "groups"},
				AdminEmail:             "foo@bar.com",
				ServiceAccountFilePath: serviceAccountFilePath,
			},
			err: "",
		},
	} {
		expected := expected
		t.Run(name, func(t *testing.T) {
			assert := assert.New(t)
			conn, err := newConnector(expected.config, ts.URL)

			if expected.err == "" {
				assert.Nil(err)
				assert.NotNil(conn)
			} else {
				assert.ErrorContains(err, expected.err)
			}
		})
	}
}
