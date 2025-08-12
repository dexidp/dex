package httpclient_test

import (
	"crypto/tls"
	"encoding/base64"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/dexidp/dex/pkg/httpclient"
)

func TestRootCAs(t *testing.T) {
	ts, err := NewLocalHTTPSTestServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprint(w, "Hello, client")
	}))
	assert.Nil(t, err)
	defer ts.Close()

	runTest := func(name string, certs []string) {
		t.Run(name, func(t *testing.T) {
			rootCAs := certs
			testClient, err := httpclient.NewHTTPClient(rootCAs, false)
			assert.Nil(t, err)

			res, err := testClient.Get(ts.URL)
			assert.Nil(t, err)

			greeting, err := io.ReadAll(res.Body)
			res.Body.Close()
			assert.Nil(t, err)

			assert.Equal(t, "Hello, client", string(greeting))
		})
	}

	runTest("From file", []string{"testdata/rootCA.pem"})

	content, err := os.ReadFile("testdata/rootCA.pem")
	assert.NoError(t, err)
	runTest("From string", []string{string(content)})

	contentStr := base64.StdEncoding.EncodeToString(content)
	runTest("From bytes", []string{contentStr})
}

func TestInsecureSkipVerify(t *testing.T) {
	ts, err := NewLocalHTTPSTestServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprint(w, "Hello, client")
	}))
	assert.Nil(t, err)
	defer ts.Close()

	insecureSkipVerify := true

	testClient, err := httpclient.NewHTTPClient(nil, insecureSkipVerify)
	assert.Nil(t, err)

	res, err := testClient.Get(ts.URL)
	assert.Nil(t, err)

	greeting, err := io.ReadAll(res.Body)
	res.Body.Close()
	assert.Nil(t, err)

	assert.Equal(t, "Hello, client", string(greeting))
}

func NewLocalHTTPSTestServer(handler http.Handler) (*httptest.Server, error) {
	ts := httptest.NewUnstartedServer(handler)
	cert, err := tls.LoadX509KeyPair("testdata/server.crt", "testdata/server.key")
	if err != nil {
		return nil, err
	}
	ts.TLS = &tls.Config{Certificates: []tls.Certificate{cert}}
	ts.StartTLS()
	return ts, nil
}
