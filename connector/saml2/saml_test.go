package saml2

import (
	"io"
	"net/http"
	"net/http/httptest"
	"net/url"
	"os"
	"path/filepath"
	"testing"

	"github.com/amdonov/lite-idp/sp"
	"github.com/sirupsen/logrus"

	"github.com/coreos/dex/connector"
)

func TestCallbackSuccess(t *testing.T) {
	ts := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		f, _ := os.Open(filepath.Join("testdata", "response.xml"))
		defer f.Close()
		io.Copy(w, f)
	}))
	defer ts.Close()
	c := &Config{
		Certificate:                 filepath.Join("testdata", "certificate.pem"),
		Key:                         filepath.Join("testdata", "key.pem"),
		EntityID:                    "test",
		AssertionConsumerServiceURL: "http://test",
		IDPArtifactEndpoint:         "http://test",
		IDPRedirectEndpoint:         "http://test",
		EmailAttr:                   "email",
		NameAttr:                    "name",
	}
	conn, err := c.Open("saml2", logrus.New())
	if err != nil {
		t.Fatal(err)
	}
	sc := conn.(*samlConnector)
	tlsConfigClient, _ := configureTLS(c)
	serviceProvider, err := sp.New(sp.Configuration{
		EntityID:                    c.EntityID,
		AssertionConsumerServiceURL: "http://test",
		Client:                      ts.Client(),
		IDPArtifactEndpoint:         ts.URL,
		IDPRedirectEndpoint:         "http://test",
		TLSConfig:                   tlsConfigClient,
	})
	sc.serviceProvider = serviceProvider

	req, _ := http.NewRequest(http.MethodGet, "test", nil)
	q := url.Values{}
	q.Add("RelayState", "12345")
	q.Add("SAMLart", "ABCDEF")

	req.URL.RawQuery = q.Encode()
	i, err := sc.HandleCallback(connector.Scopes{
		OfflineAccess: true,
		Groups:        true}, req)
	if err != nil {
		t.Fatal(err)
	}
	if i.UserID != "user@mail.example.org" {
		t.Fatal("unexpected UserID, ", i.UserID)
	}
}
