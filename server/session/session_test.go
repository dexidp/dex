package session

import (
	"context"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/dexidp/dex/server/reqctx"
)

// The address stored on a session is an audit record, so it has to be the
// client's address and nothing else — in particular not the ephemeral source
// port, which identifies the connection rather than the client.
func TestRemoteIP(t *testing.T) {
	tests := []struct {
		name       string
		remoteAddr string
		ctxIP      string
		want       string
	}{
		{
			name:       "resolver value wins",
			remoteAddr: "10.0.0.1:4711",
			ctxIP:      "203.0.113.7",
			want:       "203.0.113.7",
		},
		{
			name:       "empty resolver value falls through",
			remoteAddr: "10.0.0.1:4711",
			ctxIP:      "",
			want:       "10.0.0.1",
		},
		{
			name:       "ipv4 with port",
			remoteAddr: "192.168.1.10:54321",
			want:       "192.168.1.10",
		},
		{
			name:       "ipv6 with port",
			remoteAddr: "[2001:db8::1]:54321",
			want:       "2001:db8::1",
		},
		{
			name:       "ipv6 zone with port",
			remoteAddr: "[fe80::1%eth0]:8080",
			want:       "fe80::1%eth0",
		},
		{
			// Not something net/http produces, but the fallback must not turn an
			// unexpected value into an empty audit record.
			name:       "no port",
			remoteAddr: "192.168.1.10",
			want:       "192.168.1.10",
		},
		{
			name:       "unix socket",
			remoteAddr: "@",
			want:       "@",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			r := httptest.NewRequest("GET", "/", nil)
			r.RemoteAddr = tc.remoteAddr
			if tc.ctxIP != "" {
				r = r.WithContext(context.WithValue(r.Context(), reqctx.RequestKeyRemoteIP, tc.ctxIP))
			}

			assert.Equal(t, tc.want, remoteIP(r))
		})
	}
}
