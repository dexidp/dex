package connector

import (
	"fmt"
	"io/ioutil"
	"net/http"
	"strings"
	"testing"

	"github.com/coreos/go-oidc/oidc"
	"github.com/kylelemons/godebug/pretty"
)

type response struct {
	statusCode int
	body       string
}

type oauth2IdentityTest struct {
	urlResps map[string]response
	want     oidc.Identity
	wantErr  error
}

type fakeClient func(*http.Request) (*http.Response, error)

// implement github.com/coreos/go-oidc/oauth2.Client
func (f fakeClient) Do(r *http.Request) (*http.Response, error) {
	return f(r)
}

func runOAuth2IdentityTests(t *testing.T, conn oauth2Connector, tests []oauth2IdentityTest) {
	for i, tt := range tests {
		f := func(req *http.Request) (*http.Response, error) {
			resp, ok := tt.urlResps[req.URL.String()]
			if !ok {
				return nil, fmt.Errorf("unexpected request URL: %s", req.URL.String())
			}
			return &http.Response{
				StatusCode: resp.statusCode,
				Body:       ioutil.NopCloser(strings.NewReader(resp.body)),
			}, nil
		}
		got, err := conn.Identity(fakeClient(f))
		if tt.wantErr == nil {
			if err != nil {
				t.Errorf("case %d: failed to get identity=%v", i, err)
				continue
			}
			if diff := pretty.Compare(tt.want, got); diff != "" {
				t.Errorf("case %d: Compare(want, got) = %v", i, diff)
			}
		} else {
			if err == nil {
				t.Errorf("case %d: want error=%v, got=<nil>", i, tt.wantErr)
				continue
			}
			if diff := pretty.Compare(tt.wantErr, err); diff != "" {
				t.Errorf("case %d: Compare(wantErr, gotErr) = %v", i, diff)
			}
		}
	}
}
