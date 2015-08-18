package server

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestWriteResponseWithBody(t *testing.T) {
	type Foo struct {
		Bar string `json:"bar"`
	}

	tests := []struct {
		code     int
		obj      interface{}
		wantCode int
		wantBody string
	}{
		// unserializable
		{
			code:     http.StatusTeapot,
			obj:      make(chan bool),
			wantCode: http.StatusInternalServerError,
		},
		// serializable
		{
			code:     http.StatusTeapot,
			obj:      Foo{"asdf"},
			wantCode: http.StatusTeapot,
			wantBody: `{"bar":"asdf"}`,
		},
	}

	for i, tt := range tests {
		w := httptest.NewRecorder()
		writeResponseWithBody(w, tt.code, tt.obj)

		if tt.wantCode != w.Code {
			t.Fatalf("case %d: incorrect status code: want=%d got=%d", i, tt.wantCode, w.Code)
		}

		body := w.Body.String()
		if tt.wantBody != body {
			t.Fatalf("case %d: incorrect body: want=%s got=%s", i, tt.wantBody, body)
		}
	}
}
