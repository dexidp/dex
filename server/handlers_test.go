package server

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"golang.org/x/net/context"
)

func TestHandleHealth(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	httpServer, server := newTestServer(ctx, t, nil)
	defer httpServer.Close()

	rr := httptest.NewRecorder()
	server.handleHealth(rr, httptest.NewRequest("GET", "/healthz", nil))
	if rr.Code != http.StatusOK {
		t.Errorf("expected 200 got %d", rr.Code)
	}

}

var discoveryHandlerCORSTests = []struct {
	DiscoveryAllowedOrigins []string
	Origin                  string
	ResponseAllowOrigin     string //The expected response: same as Origin in case of valid CORS flow
}{
	{nil, "http://foo.example", ""}, //Default behavior: cross origin requests not allowed
	{[]string{}, "http://foo.example", ""},
	{[]string{"http://foo.example"}, "http://foo.example", "http://foo.example"},
	{[]string{"http://bar.example", "http://foo.example"}, "http://foo.example", "http://foo.example"},
	{[]string{"*"}, "http://foo.example", "http://foo.example"},
	{[]string{"http://bar.example"}, "http://foo.example", ""},
}

func TestDiscoveryHandlerCORS(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	for _, testcase := range discoveryHandlerCORSTests {

		httpServer, server := newTestServer(ctx, t, func(c *Config) {
			c.DiscoveryAllowedOrigins = testcase.DiscoveryAllowedOrigins
		})
		defer httpServer.Close()

		discoveryHandler, err := server.discoveryHandler()
		if err != nil {
			t.Fatalf("failed to get discovery handler: %v", err)
		}

		//Perform preflight request
		rrPreflight := httptest.NewRecorder()
		reqPreflight := httptest.NewRequest("OPTIONS", "/.well-kown/openid-configuration", nil)
		reqPreflight.Header.Set("Origin", testcase.Origin)
		reqPreflight.Header.Set("Access-Control-Request-Method", "GET")
		discoveryHandler.ServeHTTP(rrPreflight, reqPreflight)
		if rrPreflight.Code != http.StatusOK {
			t.Errorf("expected 200 got %d", rrPreflight.Code)
		}
		headerAccessControlPreflight := rrPreflight.HeaderMap.Get("Access-Control-Allow-Origin")
		if headerAccessControlPreflight != testcase.ResponseAllowOrigin {
			t.Errorf("expected '%s' got '%s'", testcase.ResponseAllowOrigin, headerAccessControlPreflight)
		}

		//Perform request
		rr := httptest.NewRecorder()
		req := httptest.NewRequest("GET", "/.well-kown/openid-configuration", nil)
		req.Header.Set("Origin", testcase.Origin)
		discoveryHandler.ServeHTTP(rr, req)
		if rr.Code != http.StatusOK {
			t.Errorf("expected 200 got %d", rr.Code)
		}
		headerAccessControl := rr.HeaderMap.Get("Access-Control-Allow-Origin")
		if headerAccessControl != testcase.ResponseAllowOrigin {
			t.Errorf("expected '%s' got '%s'", testcase.ResponseAllowOrigin, headerAccessControl)
		}
	}
}
