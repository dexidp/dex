package server

import (
	"errors"
	"net/http"
	"net/http/httptest"
	"net/url"
	"reflect"
	"testing"

	"github.com/coreos/go-oidc/oauth2"
)

func TestWriteAPIError(t *testing.T) {
	tests := []struct {
		err      error
		code     int
		wantCode int
		wantBody string
	}{
		// standard
		{
			err:      newAPIError(errorInvalidRequest, "foo"),
			code:     http.StatusBadRequest,
			wantCode: http.StatusBadRequest,
			wantBody: `{"error":"invalid_request","error_description":"foo"}`,
		},
		// no description
		{
			err:      newAPIError(errorInvalidRequest, ""),
			code:     http.StatusBadRequest,
			wantCode: http.StatusBadRequest,
			wantBody: `{"error":"invalid_request"}`,
		},
		// no type
		{
			err:      newAPIError("", ""),
			code:     http.StatusBadRequest,
			wantCode: http.StatusBadRequest,
			wantBody: `{"error":"server_error"}`,
		},
		// generic error
		{
			err:      errors.New("generic failure"),
			code:     http.StatusTeapot,
			wantCode: http.StatusTeapot,
			wantBody: `{"error":"server_error"}`,
		},
		// nil error
		{
			err:      nil,
			code:     http.StatusTeapot,
			wantCode: http.StatusTeapot,
			wantBody: `{"error":"server_error"}`,
		},
		// empty code
		{
			err:      nil,
			code:     0,
			wantCode: http.StatusInternalServerError,
			wantBody: `{"error":"server_error"}`,
		},
	}

	for i, tt := range tests {
		w := httptest.NewRecorder()
		writeAPIError(w, tt.code, tt.err)

		if tt.wantCode != w.Code {
			t.Errorf("case %d: incorrect HTTP status: want=%d got=%d", i, tt.wantCode, w.Code)
		}

		gotBody := w.Body.String()
		if tt.wantBody != gotBody {
			t.Errorf("case %d: incorrect HTTP body: want=%q got=%q", i, tt.wantBody, gotBody)
		}
	}
}

func TestWriteTokenError(t *testing.T) {
	tests := []struct {
		err        error
		state      string
		wantCode   int
		wantHeader http.Header
		wantBody   string
	}{
		{
			err:      oauth2.NewError(oauth2.ErrorInvalidRequest),
			state:    "bazinga",
			wantCode: http.StatusBadRequest,
			wantHeader: http.Header{
				"Content-Type": []string{"application/json"},
			},
			wantBody: `{"error":"invalid_request","state":"bazinga"}`,
		},
		{
			err:      oauth2.NewError(oauth2.ErrorInvalidRequest),
			wantCode: http.StatusBadRequest,
			wantHeader: http.Header{
				"Content-Type": []string{"application/json"},
			},
			wantBody: `{"error":"invalid_request"}`,
		},
		{
			err:      oauth2.NewError(oauth2.ErrorInvalidGrant),
			wantCode: http.StatusBadRequest,
			wantHeader: http.Header{
				"Content-Type": []string{"application/json"},
			},
			wantBody: `{"error":"invalid_grant"}`,
		},
		{
			err:      oauth2.NewError(oauth2.ErrorInvalidClient),
			wantCode: http.StatusUnauthorized,
			wantHeader: http.Header{
				"Content-Type":     []string{"application/json"},
				"Www-Authenticate": []string{"Basic"},
			},
			wantBody: `{"error":"invalid_client"}`,
		},
		{
			err:      oauth2.NewError(oauth2.ErrorServerError),
			wantCode: http.StatusBadRequest,
			wantHeader: http.Header{
				"Content-Type": []string{"application/json"},
			},
			wantBody: `{"error":"server_error"}`,
		},
		{
			err:      oauth2.NewError(oauth2.ErrorUnsupportedGrantType),
			wantCode: http.StatusBadRequest,
			wantHeader: http.Header{
				"Content-Type": []string{"application/json"},
			},
			wantBody: `{"error":"unsupported_grant_type"}`,
		},
		{
			err:      errors.New("generic failure"),
			wantCode: http.StatusBadRequest,
			wantHeader: http.Header{
				"Content-Type": []string{"application/json"},
			},
			wantBody: `{"error":"server_error"}`,
		},
	}

	for i, tt := range tests {
		w := httptest.NewRecorder()
		writeTokenError(w, tt.err, tt.state)

		if tt.wantCode != w.Code {
			t.Errorf("case %d: incorrect HTTP status: want=%d got=%d", i, tt.wantCode, w.Code)
		}

		gotHeader := w.Header()
		if !reflect.DeepEqual(tt.wantHeader, gotHeader) {
			t.Errorf("case %d: incorrect HTTP headers: want=%#v got=%#v", i, tt.wantHeader, gotHeader)
		}

		gotBody := w.Body.String()
		if tt.wantBody != gotBody {
			t.Errorf("case %d: incorrect HTTP body: want=%q got=%q", i, tt.wantBody, gotBody)
		}
	}
}

func TestWriteAuthError(t *testing.T) {
	wantCode := http.StatusBadRequest
	wantHeader := http.Header{"Content-Type": []string{"application/json"}}
	tests := []struct {
		err      error
		state    string
		wantBody string
	}{
		{
			err:      errors.New("foobar"),
			state:    "bazinga",
			wantBody: `{"error":"server_error","state":"bazinga"}`,
		},
		{
			err:      oauth2.NewError(oauth2.ErrorInvalidRequest),
			state:    "foo",
			wantBody: `{"error":"invalid_request","state":"foo"}`,
		},
		{
			err:      oauth2.NewError(oauth2.ErrorUnsupportedResponseType),
			state:    "bar",
			wantBody: `{"error":"unsupported_response_type","state":"bar"}`,
		},
	}

	for i, tt := range tests {
		w := httptest.NewRecorder()
		writeAuthError(w, tt.err, tt.state)

		if wantCode != w.Code {
			t.Errorf("case %d: incorrect HTTP status: want=%d got=%d", i, wantCode, w.Code)
		}

		gotHeader := w.Header()
		if !reflect.DeepEqual(wantHeader, gotHeader) {
			t.Errorf("case %d: incorrect HTTP headers: want=%#v got=%#v", i, wantHeader, gotHeader)
		}

		gotBody := w.Body.String()
		if tt.wantBody != gotBody {
			t.Errorf("case %d: incorrect HTTP body: want=%q got=%q", i, tt.wantBody, gotBody)
		}
	}
}

func TestRedirectAuthError(t *testing.T) {
	wantCode := http.StatusFound

	tests := []struct {
		err         error
		state       string
		redirectURL url.URL
		wantLoc     string
	}{
		{
			err:         errors.New("foobar"),
			state:       "bazinga",
			redirectURL: url.URL{Scheme: "http", Host: "server.example.com"},
			wantLoc:     "http://server.example.com?error=server_error&state=bazinga",
		},
		{
			err:         oauth2.NewError(oauth2.ErrorInvalidRequest),
			state:       "foo",
			redirectURL: url.URL{Scheme: "http", Host: "server.example.com"},
			wantLoc:     "http://server.example.com?error=invalid_request&state=foo",
		},
		{
			err:         oauth2.NewError(oauth2.ErrorUnsupportedResponseType),
			state:       "bar",
			redirectURL: url.URL{Scheme: "http", Host: "server.example.com"},
			wantLoc:     "http://server.example.com?error=unsupported_response_type&state=bar",
		},
	}

	for i, tt := range tests {
		w := httptest.NewRecorder()
		redirectAuthError(w, tt.err, tt.state, tt.redirectURL)

		if wantCode != w.Code {
			t.Errorf("case %d: incorrect HTTP status: want=%d got=%d", i, wantCode, w.Code)
		}

		wantHeader := http.Header{"Location": []string{tt.wantLoc}}
		gotHeader := w.Header()
		if !reflect.DeepEqual(wantHeader, gotHeader) {
			t.Errorf("case %d: incorrect HTTP headers: want=%#v got=%#v", i, wantHeader, gotHeader)
		}

		gotBody := w.Body.String()
		if gotBody != "" {
			t.Errorf("case %d: incorrect empty HTTP body, got=%q", i, gotBody)
		}
	}
}
