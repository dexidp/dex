package http

import (
	"encoding/json"
	"net/http"
	"net/url"
	"path"

	"github.com/coreos/dex/pkg/log"
)

func WriteError(w http.ResponseWriter, code int, msg string) {
	e := struct {
		Error string `json:"error"`
	}{
		Error: msg,
	}
	b, err := json.Marshal(e)
	if err != nil {
		log.Errorf("Failed marshaling %#v to JSON: %v", e, err)
	}
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(code)
	w.Write(b)
}

// MergeQuery appends additional query values to an existing URL.
func MergeQuery(u url.URL, q url.Values) url.URL {
	uv := u.Query()
	for k, vs := range q {
		for _, v := range vs {
			uv.Add(k, v)
		}
	}
	u.RawQuery = uv.Encode()
	return u
}

// NewResourceLocation appends a resource id to the end of the requested URL path.
func NewResourceLocation(reqURL *url.URL, id string) string {
	var u url.URL
	u = *reqURL
	u.Path = path.Join(u.Path, id)
	u.RawQuery = ""
	u.Fragment = ""
	return u.String()
}
