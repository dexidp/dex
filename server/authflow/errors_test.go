package authflow

import (
	"io"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/require"
)

// TestRenderErrorSafeMessages tests that renderError uses safe messages.
func TestRenderErrorSafeMessages(t *testing.T) {
	tests := []struct {
		name           string
		statusCode     int
		message        string
		expectedInBody []string
		notInBody      []string
	}{
		{
			name:           "Login error message",
			statusCode:     http.StatusInternalServerError,
			message:        ErrMsgLoginError,
			expectedInBody: []string{"Login error", "administrator"},
			notInBody:      []string{"stack", "panic", ".go:"},
		},
		{
			name:           "Authentication failed message",
			statusCode:     http.StatusInternalServerError,
			message:        ErrMsgAuthenticationFailed,
			expectedInBody: []string{"Authentication failed", "administrator"},
			notInBody:      []string{"stack", "panic", ".go:"},
		},
		{
			name:           "Database error message",
			statusCode:     http.StatusInternalServerError,
			message:        ErrMsgDatabaseError,
			expectedInBody: []string{"database error"},
			notInBody:      []string{"sql:", "connection", "timeout"},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			_, s := newTestHandler(t, nil)

			rr := httptest.NewRecorder()
			req := httptest.NewRequest("GET", "/", nil)

			s.RenderError(req, rr, tc.statusCode, tc.message)

			resp := rr.Result()
			defer resp.Body.Close()

			body, _ := io.ReadAll(resp.Body)
			bodyStr := string(body)

			require.Equal(t, tc.statusCode, resp.StatusCode)

			for _, expected := range tc.expectedInBody {
				require.Contains(t, bodyStr, expected, "Response should contain: %s", expected)
			}
			for _, notExpected := range tc.notInBody {
				require.NotContains(t, bodyStr, notExpected, "Response should not contain: %s", notExpected)
			}
		})
	}
}
