package mailgun

import (
	"fmt"
)

// The MailgunGoUserAgent identifies the client to the server, for logging purposes.
// In the event of problems requiring a human administrator's assistance,
// this user agent allows them to identify the client from human-generated activity.
const MailgunGoUserAgent = "mailgun-go/1.0.0"

// This error will be returned whenever a Mailgun API returns an error response.
// Your application can check the Actual field to see the actual HTTP response code returned.
// URL contains the base URL accessed, sans any query parameters.
type UnexpectedResponseError struct {
	Expected []int
	Actual   int
	URL      string
	Data     []byte
}

// String() converts the error into a human-readable, logfmt-compliant string.
// See http://godoc.org/github.com/kr/logfmt for details on logfmt formatting.
func (e *UnexpectedResponseError) String() string {
	return fmt.Sprintf("UnexpectedResponseError URL=%s ExpectedOneOf=%#v Got=%d Error: %s", e.URL, e.Expected, e.Actual, string(e.Data))
}

// Error() performs as String().
func (e *UnexpectedResponseError) Error() string {
	return e.String()
}

// newError creates a new error condition to be returned.
func newError(url string, expected []int, got *httpResponse) error {
	return &UnexpectedResponseError{
		URL:      url,
		Expected: expected,
		Actual:   got.Code,
		Data:     got.Data,
	}
}

// notGood searches a list of response codes (the haystack) for a matching entry (the needle).
// If found, the response code is considered good, and thus false is returned.
// Otherwise true is returned.
func notGood(needle int, haystack []int) bool {
	for _, i := range haystack {
		if needle == i {
			return false
		}
	}
	return true
}

// expected denotes the expected list of known-good HTTP response codes possible from the Mailgun API.
var expected = []int{200, 202, 204}

// makeRequest shim performs a generic request, checking for a positive outcome.
// See simplehttp.MakeRequest for more details.
func makeRequest(r *httpRequest, kind string, p payload) (*httpResponse, error) {
	r.addHeader("User-Agent", MailgunGoUserAgent)
	rsp, err := r.makeRequest(kind, p)
	if (err == nil) && notGood(rsp.Code, expected) {
		return rsp, newError(r.URL, expected, rsp)
	}
	return rsp, err
}

// getResponseFromJSON shim performs a GET request, checking for a positive outcome.
// See simplehttp.GetResponseFromJSON for more details.
func getResponseFromJSON(r *httpRequest, v interface{}) error {
	r.addHeader("User-Agent", MailgunGoUserAgent)
	response, err := r.makeGetRequest()
	if err != nil {
		return err
	}
	if notGood(response.Code, expected) {
		return newError(r.URL, expected, response)
	}
	return response.parseFromJSON(v)
}

// postResponseFromJSON shim performs a POST request, checking for a positive outcome.
// See simplehttp.PostResponseFromJSON for more details.
func postResponseFromJSON(r *httpRequest, p payload, v interface{}) error {
	r.addHeader("User-Agent", MailgunGoUserAgent)
	response, err := r.makePostRequest(p)
	if err != nil {
		return err
	}
	if notGood(response.Code, expected) {
		return newError(r.URL, expected, response)
	}
	return response.parseFromJSON(v)
}

// putResponseFromJSON shim performs a PUT request, checking for a positive outcome.
// See simplehttp.PutResponseFromJSON for more details.
func putResponseFromJSON(r *httpRequest, p payload, v interface{}) error {
	r.addHeader("User-Agent", MailgunGoUserAgent)
	response, err := r.makePutRequest(p)
	if err != nil {
		return err
	}
	if notGood(response.Code, expected) {
		return newError(r.URL, expected, response)
	}
	return response.parseFromJSON(v)
}

// makeGetRequest shim performs a GET request, checking for a positive outcome.
// See simplehttp.MakeGetRequest for more details.
func makeGetRequest(r *httpRequest) (*httpResponse, error) {
	r.addHeader("User-Agent", MailgunGoUserAgent)
	rsp, err := r.makeGetRequest()
	if (err == nil) && notGood(rsp.Code, expected) {
		return rsp, newError(r.URL, expected, rsp)
	}
	return rsp, err
}

// makePostRequest shim performs a POST request, checking for a positive outcome.
// See simplehttp.MakePostRequest for more details.
func makePostRequest(r *httpRequest, p payload) (*httpResponse, error) {
	r.addHeader("User-Agent", MailgunGoUserAgent)
	rsp, err := r.makePostRequest(p)
	if (err == nil) && notGood(rsp.Code, expected) {
		return rsp, newError(r.URL, expected, rsp)
	}
	return rsp, err
}

// makePutRequest shim performs a PUT request, checking for a positive outcome.
// See simplehttp.MakePutRequest for more details.
func makePutRequest(r *httpRequest, p payload) (*httpResponse, error) {
	r.addHeader("User-Agent", MailgunGoUserAgent)
	rsp, err := r.makePutRequest(p)
	if (err == nil) && notGood(rsp.Code, expected) {
		return rsp, newError(r.URL, expected, rsp)
	}
	return rsp, err
}

// makeDeleteRequest shim performs a DELETE request, checking for a positive outcome.
// See simplehttp.MakeDeleteRequest for more details.
func makeDeleteRequest(r *httpRequest) (*httpResponse, error) {
	r.addHeader("User-Agent", MailgunGoUserAgent)
	rsp, err := r.makeDeleteRequest()
	if (err == nil) && notGood(rsp.Code, expected) {
		return rsp, newError(r.URL, expected, rsp)
	}
	return rsp, err
}
