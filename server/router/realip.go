package router

import (
	"net"
	"net/http"
	"net/netip"
	"slices"
	"strings"
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
//
// The header may carry a chain of hops (X-Forwarded-For appends one per proxy,
// left to right, so the rightmost is the closest). Only the hops added by the
// trusted proxies can be believed: the client controls everything to the left of
// its first trusted proxy and can prepend whatever it likes. So the chain is
// walked from the right, skipping trusted hops, and the first untrusted one is
// the client. A single-valued header (X-Real-IP) is a chain of one and needs no
// special case.
func ParseRealIP(header string, trusted []netip.Prefix) func(*http.Request) (string, error) {
	isTrusted := func(ip netip.Addr) bool {
		return slices.ContainsFunc(trusted, func(n netip.Prefix) bool { return n.Contains(ip) })
	}

	return func(r *http.Request) (string, error) {
		remoteAddr, _, err := net.SplitHostPort(r.RemoteAddr)
		if err != nil {
			return "", err
		}

		remoteIP, err := netip.ParseAddr(remoteAddr)
		if err != nil {
			return "", err
		}

		if !isTrusted(remoteIP) {
			return remoteAddr, nil
		}

		hops := strings.Split(r.Header.Get(header), ",")
		for _, hop := range slices.Backward(hops) {
			ip, err := netip.ParseAddr(strings.TrimSpace(hop))
			if err != nil {
				// A malformed hop breaks the chain: nothing to its left can be
				// attributed to a trusted proxy, so stop and fall back.
				break
			}
			if !isTrusted(ip) {
				return ip.String(), nil
			}
		}

		// Every hop was a trusted proxy, or the header was absent or malformed.
		return remoteAddr, nil
	}
}
