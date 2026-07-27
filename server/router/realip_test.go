package router

import (
	"net/http"
	"net/http/httptest"
	"net/netip"
	"testing"

	"github.com/stretchr/testify/require"
)

func prefixes(t *testing.T, cidrs ...string) []netip.Prefix {
	t.Helper()
	out := make([]netip.Prefix, 0, len(cidrs))
	for _, c := range cidrs {
		p, err := netip.ParsePrefix(c)
		require.NoError(t, err)
		out = append(out, p)
	}
	return out
}

func TestParseRealIP(t *testing.T) {
	const header = "X-Forwarded-For"

	tests := []struct {
		name       string
		trusted    []string
		remoteAddr string
		headerVal  string
		want       string
	}{
		{
			name:       "trusted proxy, header honoured",
			trusted:    []string{"10.0.0.0/8"},
			remoteAddr: "10.1.2.3:1234",
			headerVal:  "203.0.113.9",
			want:       "203.0.113.9",
		},
		{
			// The resolver used to return on the first prefix that did not match,
			// so only the first entry of trustedProxies ever took effect.
			name:       "trusted proxy in a later prefix, header honoured",
			trusted:    []string{"10.0.0.0/8", "192.168.0.0/16"},
			remoteAddr: "192.168.5.5:1234",
			headerVal:  "203.0.113.9",
			want:       "203.0.113.9",
		},
		{
			name:       "untrusted peer, header ignored",
			trusted:    []string{"10.0.0.0/8"},
			remoteAddr: "203.0.113.1:1234",
			headerVal:  "198.51.100.7",
			want:       "203.0.113.1",
		},
		{
			// Nothing is trusted, so the header is spoofable by any client.
			name:       "no trusted proxies, header ignored",
			trusted:    nil,
			remoteAddr: "203.0.113.1:1234",
			headerVal:  "198.51.100.7",
			want:       "203.0.113.1",
		},
		{
			name:       "trusted proxy, no header",
			trusted:    []string{"10.0.0.0/8"},
			remoteAddr: "10.1.2.3:1234",
			want:       "10.1.2.3",
		},
		{
			name:       "trusted proxy, unparseable header",
			trusted:    []string{"10.0.0.0/8"},
			remoteAddr: "10.1.2.3:1234",
			headerVal:  "not-an-ip",
			want:       "10.1.2.3",
		},
		{
			name:       "IPv6 trusted proxy",
			trusted:    []string{"2001:db8::/32"},
			remoteAddr: "[2001:db8::1]:1234",
			headerVal:  "203.0.113.9",
			want:       "203.0.113.9",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			resolve := ParseRealIP(header, prefixes(t, tc.trusted...))

			r := httptest.NewRequest(http.MethodGet, "/auth", nil)
			r.RemoteAddr = tc.remoteAddr
			if tc.headerVal != "" {
				r.Header.Set(header, tc.headerVal)
			}

			got, err := resolve(r)
			require.NoError(t, err)
			require.Equal(t, tc.want, got)
		})
	}
}

func TestParseRealIPMalformedRemoteAddr(t *testing.T) {
	resolve := ParseRealIP("X-Forwarded-For", prefixes(t, "10.0.0.0/8"))

	r := httptest.NewRequest(http.MethodGet, "/auth", nil)
	r.RemoteAddr = "no-port"

	_, err := resolve(r)
	require.Error(t, err)
}
