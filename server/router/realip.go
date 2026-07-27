package router

import (
	"net"
	"net/http"
	"net/netip"
	"slices"
)

// ParseRealIP returns the resolver that Config.RealIP is set to: it reports the
// client's IP for a request, reading it from the named header when — and only
// when — the request reached dex through a trusted proxy.
//
// A request's peer address is trusted if it falls inside any one of the trusted
// prefixes. The header is attacker-controlled for everyone else, so an untrusted
// peer (including every peer when trusted is empty) is reported by its own
// address. Callers that set a header without declaring trusted proxies get the
// peer address, never the header.
func ParseRealIP(header string, trusted []netip.Prefix) func(*http.Request) (string, error) {
	return func(r *http.Request) (string, error) {
		remoteAddr, _, err := net.SplitHostPort(r.RemoteAddr)
		if err != nil {
			return "", err
		}

		remoteIP, err := netip.ParseAddr(remoteAddr)
		if err != nil {
			return "", err
		}

		if !slices.ContainsFunc(trusted, func(n netip.Prefix) bool { return n.Contains(remoteIP) }) {
			return remoteAddr, nil
		}

		if ip, err := netip.ParseAddr(r.Header.Get(header)); err == nil {
			return ip.String(), nil
		}

		return remoteAddr, nil
	}
}
