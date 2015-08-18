// Package simplehttp provides some simple methods and types to do
// HTTP queries with form values and parameters easily - especially
// if the returned result is expected to be JSON or XML.
//
// Author: Michael Banzon
package simplehttp

import (
	"io"
	"io/ioutil"
	"net/http"
	"net/url"
)

// Holds all information used to make a HTTP request.
type HTTPRequest struct {
	URL               string
	Parameters        map[string][]string
	Headers           map[string]string
	BasicAuthUser     string
	BasicAuthPassword string
	Client            *http.Client
}

type HTTPResponse struct {
	Code int
	Data []byte
}

// Creates a new HTTPRequest instance.
func NewHTTPRequest(url string) *HTTPRequest {
	return &HTTPRequest{URL: url, Client: http.DefaultClient}
}

// Adds a parameter to the generated query string.
func (r *HTTPRequest) AddParameter(name, value string) {
	if r.Parameters == nil {
		r.Parameters = make(map[string][]string)
	}
	r.Parameters[name] = append(r.Parameters[name], value)
}

// Adds a header that will be sent with the HTTP request.
func (r *HTTPRequest) AddHeader(name, value string) {
	// hej
	if r.Headers == nil {
		r.Headers = make(map[string]string)
	}
	r.Headers[name] = value
}

// Sets username and password for basic authentication.
func (r *HTTPRequest) SetBasicAuth(user, password string) {
	r.BasicAuthUser = user
	r.BasicAuthPassword = password
}

func (r *HTTPRequest) MakeGetRequest() (*HTTPResponse, error) {
	return r.MakeRequest("GET", nil)
}

func (r *HTTPRequest) MakePostRequest(payload Payload) (*HTTPResponse, error) {
	return r.MakeRequest("POST", payload)
}

func (r *HTTPRequest) MakePutRequest(payload Payload) (*HTTPResponse, error) {
	return r.MakeRequest("PUT", payload)
}

func (r *HTTPRequest) MakeDeleteRequest() (*HTTPResponse, error) {
	return r.MakeRequest("DELETE", nil)
}

func (r *HTTPRequest) MakeRequest(method string, payload Payload) (*HTTPResponse, error) {
	url, err := r.generateUrlWithParameters()
	if err != nil {
		return nil, err
	}

	var body io.Reader
	if payload != nil {
		if body, err = payload.GetPayloadBuffer(); err != nil {
			return nil, err
		}
	} else {
		body = nil
	}

	req, err := http.NewRequest(method, url, body)
	if err != nil {
		return nil, err
	}

	if payload != nil && payload.GetContentType() != "" {
		req.Header.Add("Content-Type", payload.GetContentType())
	}

	if r.BasicAuthUser != "" && r.BasicAuthPassword != "" {
		req.SetBasicAuth(r.BasicAuthUser, r.BasicAuthPassword)
	}

	for header, value := range r.Headers {
		req.Header.Add(header, value)
	}

	response := HTTPResponse{}

	resp, err := r.Client.Do(req)
	if resp != nil {
		response.Code = resp.StatusCode
	}
	if err != nil {
		return nil, err
	}

	defer resp.Body.Close()
	responseBody, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	response.Data = responseBody
	return &response, nil
}

// Generates the complete URL using GET parameters.
func (r *HTTPRequest) generateUrlWithParameters() (string, error) {
	url, err := url.Parse(r.URL)
	if err != nil {
		return "", err
	}
	q := url.Query()
	if r.Parameters != nil && len(r.Parameters) > 0 {
		for name, values := range r.Parameters {
			for _, value := range values {
				q.Add(name, value)
			}
		}
	}
	url.RawQuery = q.Encode()

	return url.String(), nil
}

func (r *HTTPRequest) SetClient(c *http.Client) {
	r.Client = c
}
