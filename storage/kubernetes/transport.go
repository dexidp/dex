package kubernetes

import (
	"net/http"
)

type transport struct {
	updateReq func(r *http.Request)
	base      http.RoundTripper
}

func (t transport) RoundTrip(r *http.Request) (*http.Response, error) {
	// shallow copy of the struct
	r2 := new(http.Request)
	*r2 = *r
	// deep copy of the Header
	r2.Header = make(http.Header, len(r.Header))
	for k, s := range r.Header {
		r2.Header[k] = append([]string(nil), s...)
	}
	t.updateReq(r2)
	return t.base.RoundTrip(r2)
}
