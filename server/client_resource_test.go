package server

import (
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"io/ioutil"
	"net/http"
	"net/http/httptest"
	"net/url"
	"reflect"
	"sort"
	"strings"
	"testing"

	"github.com/coreos/dex/db"
	schema "github.com/coreos/dex/schema/workerschema"
	"github.com/coreos/go-oidc/oidc"
	"github.com/kylelemons/godebug/pretty"
)

func makeBody(s string) io.ReadCloser {
	return ioutil.NopCloser(strings.NewReader(s))
}

func TestCreateInvalidRequest(t *testing.T) {
	u := &url.URL{Scheme: "http", Host: "example.com", Path: "clients"}
	h := http.Header{"Content-Type": []string{"application/json"}}
	repo := db.NewClientIdentityRepo(db.NewMemDB())
	res := &clientResource{repo: repo}
	tests := []struct {
		req      *http.Request
		wantCode int
		wantBody string
	}{
		// invalid content-type
		{
			req:      &http.Request{Method: "POST", URL: u, Header: http.Header{"Content-Type": []string{"application/xml"}}},
			wantCode: http.StatusBadRequest,
			wantBody: `{"error":"invalid_request","error_description":"unsupported content-type"}`,
		},
		// invalid method
		{
			req:      &http.Request{Method: "DELETE", URL: u, Header: h},
			wantCode: http.StatusMethodNotAllowed,
			wantBody: `{"error":"invalid_request","error_description":"HTTP DELETE method not supported for this resource"}`,
		},
		// invalid method
		{
			req:      &http.Request{Method: "PUT", URL: u, Header: h},
			wantCode: http.StatusMethodNotAllowed,
			wantBody: `{"error":"invalid_request","error_description":"HTTP PUT method not supported for this resource"}`,
		},
		// invalid method
		{
			req:      &http.Request{Method: "HEAD", URL: u, Header: h},
			wantCode: http.StatusMethodNotAllowed,
			wantBody: `{"error":"invalid_request","error_description":"HTTP HEAD method not supported for this resource"}`,
		},
		// unserializable body
		{
			req:      &http.Request{Method: "POST", URL: u, Header: h, Body: makeBody("asdf")},
			wantCode: http.StatusBadRequest,
			wantBody: `{"error":"invalid_request","error_description":"unable to decode request body"}`,
		},
		// empty body
		{
			req:      &http.Request{Method: "POST", URL: u, Header: h, Body: makeBody("")},
			wantCode: http.StatusBadRequest,
			wantBody: `{"error":"invalid_request","error_description":"unable to decode request body"}`,
		},
		// missing url field
		{
			req:      &http.Request{Method: "POST", URL: u, Header: h, Body: makeBody(`{"id":"foo"}`)},
			wantCode: http.StatusBadRequest,
			wantBody: `{"error":"invalid_client_metadata","error_description":"zero redirect URLs"}`,
		},
		// empty url array
		{
			req:      &http.Request{Method: "POST", URL: u, Header: h, Body: makeBody(`{"redirectURIs":[]}`)},
			wantCode: http.StatusBadRequest,
			wantBody: `{"error":"invalid_client_metadata","error_description":"zero redirect URLs"}`,
		},
		// array with empty string
		{
			req:      &http.Request{Method: "POST", URL: u, Header: h, Body: makeBody(`{"redirectURIs":[""]}`)},
			wantCode: http.StatusBadRequest,
			wantBody: `{"error":"invalid_client_metadata","error_description":"missing or invalid field: redirectURIs"}`,
		},
		// uri with unusable scheme
		{
			req:      &http.Request{Method: "POST", URL: u, Header: h, Body: makeBody(`{"redirectURIs":["asdf.com"]}`)},
			wantCode: http.StatusBadRequest,
			wantBody: `{"error":"invalid_client_metadata","error_description":"no host for uri field redirect_uris"}`,
		},
		// uri missing host
		{
			req:      &http.Request{Method: "POST", URL: u, Header: h, Body: makeBody(`{"redirectURIs":["http://"]}`)},
			wantCode: http.StatusBadRequest,
			wantBody: `{"error":"invalid_client_metadata","error_description":"no host for uri field redirect_uris"}`,
		},
	}

	for i, tt := range tests {
		w := httptest.NewRecorder()
		res.ServeHTTP(w, tt.req)

		if w.Code != tt.wantCode {
			t.Errorf("case %d: invalid response code, want=%d, got=%d", i, tt.wantCode, w.Code)
		}

		gotBody := w.Body.String()
		if gotBody != tt.wantBody {
			t.Errorf("case %d: invalid response body, want=%s, got=%s", i, tt.wantBody, gotBody)
		}
	}
}

func TestCreate(t *testing.T) {
	repo := db.NewClientIdentityRepo(db.NewMemDB())
	res := &clientResource{repo: repo}
	tests := [][]string{
		[]string{"http://example.com"},
		[]string{"https://example.com"},
		[]string{"http://example.com/foo"},
		[]string{"http://example.com/bar", "http://example.com/foo"},
	}
	endpoint := "http://example.com/clients"

	for i, tt := range tests {
		body := strings.NewReader(fmt.Sprintf(`{"redirectURIs":["%s"]}`, strings.Join(tt, `","`)))
		r, err := http.NewRequest("POST", endpoint, body)
		if err != nil {
			t.Fatalf("Failed creating http.Request: %v", err)
		}
		r.Header.Set("content-type", "application/json")
		w := httptest.NewRecorder()
		res.ServeHTTP(w, r)

		if w.Code != http.StatusCreated {
			t.Errorf("case %d: invalid response code, want=%d, got=%d", i, http.StatusCreated, w.Code)
		}

		var client schema.ClientWithSecret
		if err := json.Unmarshal(w.Body.Bytes(), &client); err != nil {
			t.Errorf("case %d: unexpected error=%v", i, err)
		}
		if len(client.RedirectURIs) != len(tt) {
			t.Errorf("case %d: unexpected number of redirect URIs, want=%d, got=%d", i, len(tt), len(client.RedirectURIs))
		}

		if !reflect.DeepEqual(tt, client.RedirectURIs) {
			t.Errorf("case %d: unexpected client redirect URIs: want=%v got=%v", i, tt, client.RedirectURIs)
		}

		if client.Id == "" {
			t.Errorf("case %d: empty client ID in response", i)
		}

		if client.Secret == "" {
			t.Errorf("case %d: empty client secret in response", i)
		}

		wantLoc := fmt.Sprintf("%s/%s", endpoint, client.Id)
		gotLoc := w.Header().Get("Location")
		if gotLoc != wantLoc {
			t.Errorf("case %d: invalid location header, want=%v, got=%v", i, wantLoc, gotLoc)
		}
	}
}

func TestList(t *testing.T) {

	b64Encode := func(s string) string {
		return base64.URLEncoding.EncodeToString([]byte(s))
	}

	tests := []struct {
		cs   []oidc.ClientIdentity
		want []*schema.Client
	}{
		// empty repo
		{
			cs:   nil,
			want: nil,
		},
		// single client
		{
			cs: []oidc.ClientIdentity{
				oidc.ClientIdentity{
					Credentials: oidc.ClientCredentials{ID: "foo", Secret: b64Encode("bar")},
					Metadata: oidc.ClientMetadata{
						RedirectURIs: []url.URL{
							url.URL{Scheme: "http", Host: "example.com"},
						},
					},
				},
			},
			want: []*schema.Client{
				&schema.Client{
					Id:           "foo",
					RedirectURIs: []string{"http://example.com"},
				},
			},
		},
		// multi client
		{
			cs: []oidc.ClientIdentity{
				oidc.ClientIdentity{
					Credentials: oidc.ClientCredentials{ID: "foo", Secret: b64Encode("bar")},
					Metadata: oidc.ClientMetadata{
						RedirectURIs: []url.URL{
							url.URL{Scheme: "http", Host: "example.com"},
						},
					},
				},
				oidc.ClientIdentity{
					Credentials: oidc.ClientCredentials{ID: "biz", Secret: b64Encode("bang")},
					Metadata: oidc.ClientMetadata{
						RedirectURIs: []url.URL{
							url.URL{Scheme: "https", Host: "example.com", Path: "one/two/three"},
						},
					},
				},
			},
			want: []*schema.Client{
				&schema.Client{
					Id:           "biz",
					RedirectURIs: []string{"https://example.com/one/two/three"},
				},
				&schema.Client{
					Id:           "foo",
					RedirectURIs: []string{"http://example.com"},
				},
			},
		},
	}

	for i, tt := range tests {
		repo, err := db.NewClientIdentityRepoFromClients(db.NewMemDB(), tt.cs)
		if err != nil {
			t.Errorf("case %d: failed to create client identity repo: %v", i, err)
			continue
		}
		res := &clientResource{repo: repo}

		r, err := http.NewRequest("GET", "http://example.com/clients", nil)
		if err != nil {
			t.Fatalf("Failed creating http.Request: %v", err)
		}
		w := httptest.NewRecorder()
		res.ServeHTTP(w, r)

		if w.Code != http.StatusOK {
			t.Errorf("case %d: invalid response code, want=%d, got=%d", i, http.StatusOK, w.Code)
		}

		var resp schema.ClientPage
		if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
			t.Errorf("case %d: unexpected error=%v", i, err)
		}
		sort.Sort(byClientId(tt.want))
		sort.Sort(byClientId(resp.Clients))

		if diff := pretty.Compare(tt.want, resp.Clients); diff != "" {
			t.Errorf("case %d: invalid response body: %s", i, diff)
		}
	}
}

type byClientId []*schema.Client

func (b byClientId) Len() int           { return len(b) }
func (b byClientId) Less(i, j int) bool { return b[i].Id < b[j].Id }
func (b byClientId) Swap(i, j int)      { b[i], b[j] = b[j], b[i] }
