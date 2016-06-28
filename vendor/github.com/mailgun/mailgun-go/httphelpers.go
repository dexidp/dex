package mailgun

import (
	"bytes"
	"encoding/json"
	"io"
	"io/ioutil"
	"mime/multipart"
	"net/http"
	"net/url"
	"os"
	"path"
)

type httpRequest struct {
	URL               string
	Parameters        map[string][]string
	Headers           map[string]string
	BasicAuthUser     string
	BasicAuthPassword string
	Client            *http.Client
}

type httpResponse struct {
	Code int
	Data []byte
}

type payload interface {
	getPayloadBuffer() (*bytes.Buffer, error)
	getContentType() string
}

type keyValuePair struct {
	key   string
	value string
}

type keyNameRC struct {
	key   string
	name  string
	value io.ReadCloser
}

type formDataPayload struct {
	contentType string
	Values      []keyValuePair
	Files       []keyValuePair
	ReadClosers []keyNameRC
}

type urlEncodedPayload struct {
	Values []keyValuePair
}

func newHTTPRequest(url string) *httpRequest {
	return &httpRequest{URL: url, Client: http.DefaultClient}
}

func (r *httpRequest) addParameter(name, value string) {
	if r.Parameters == nil {
		r.Parameters = make(map[string][]string)
	}
	r.Parameters[name] = append(r.Parameters[name], value)
}

func (r *httpRequest) setClient(c *http.Client) {
	r.Client = c
}

func (r *httpRequest) setBasicAuth(user, password string) {
	r.BasicAuthUser = user
	r.BasicAuthPassword = password
}

func newUrlEncodedPayload() *urlEncodedPayload {
	return &urlEncodedPayload{}
}

func (f *urlEncodedPayload) addValue(key, value string) {
	f.Values = append(f.Values, keyValuePair{key: key, value: value})
}

func (f *urlEncodedPayload) getPayloadBuffer() (*bytes.Buffer, error) {
	data := url.Values{}
	for _, keyVal := range f.Values {
		data.Add(keyVal.key, keyVal.value)
	}
	return bytes.NewBufferString(data.Encode()), nil
}

func (f *urlEncodedPayload) getContentType() string {
	return "application/x-www-form-urlencoded"
}

func (r *httpResponse) parseFromJSON(v interface{}) error {
	return json.Unmarshal(r.Data, v)
}

func newFormDataPayload() *formDataPayload {
	return &formDataPayload{}
}

func (f *formDataPayload) addValue(key, value string) {
	f.Values = append(f.Values, keyValuePair{key: key, value: value})
}

func (f *formDataPayload) addFile(key, file string) {
	f.Files = append(f.Files, keyValuePair{key: key, value: file})
}

func (f *formDataPayload) addReadCloser(key, name string, rc io.ReadCloser) {
	f.ReadClosers = append(f.ReadClosers, keyNameRC{key: key, name: name, value: rc})
}

func (f *formDataPayload) getPayloadBuffer() (*bytes.Buffer, error) {
	data := &bytes.Buffer{}
	writer := multipart.NewWriter(data)
	defer writer.Close()

	for _, keyVal := range f.Values {
		if tmp, err := writer.CreateFormField(keyVal.key); err == nil {
			tmp.Write([]byte(keyVal.value))
		} else {
			return nil, err
		}
	}

	for _, file := range f.Files {
		if tmp, err := writer.CreateFormFile(file.key, path.Base(file.value)); err == nil {
			if fp, err := os.Open(file.value); err == nil {
				defer fp.Close()
				io.Copy(tmp, fp)
			} else {
				return nil, err
			}
		} else {
			return nil, err
		}
	}

	for _, file := range f.ReadClosers {
		if tmp, err := writer.CreateFormFile(file.key, file.name); err == nil {
			defer file.value.Close()
			io.Copy(tmp, file.value)
		} else {
			return nil, err
		}
	}

	f.contentType = writer.FormDataContentType()

	return data, nil
}

func (f *formDataPayload) getContentType() string {
	if f.contentType == "" {
		f.getPayloadBuffer()
	}
	return f.contentType
}

func (r *httpRequest) addHeader(name, value string) {
	if r.Headers == nil {
		r.Headers = make(map[string]string)
	}
	r.Headers[name] = value
}

func (r *httpRequest) makeGetRequest() (*httpResponse, error) {
	return r.makeRequest("GET", nil)
}

func (r *httpRequest) makePostRequest(payload payload) (*httpResponse, error) {
	return r.makeRequest("POST", payload)
}

func (r *httpRequest) makePutRequest(payload payload) (*httpResponse, error) {
	return r.makeRequest("PUT", payload)
}

func (r *httpRequest) makeDeleteRequest() (*httpResponse, error) {
	return r.makeRequest("DELETE", nil)
}

func (r *httpRequest) makeRequest(method string, payload payload) (*httpResponse, error) {
	url, err := r.generateUrlWithParameters()
	if err != nil {
		return nil, err
	}

	var body io.Reader
	if payload != nil {
		if body, err = payload.getPayloadBuffer(); err != nil {
			return nil, err
		}
	} else {
		body = nil
	}

	req, err := http.NewRequest(method, url, body)
	if err != nil {
		return nil, err
	}

	if payload != nil && payload.getContentType() != "" {
		req.Header.Add("Content-Type", payload.getContentType())
	}

	if r.BasicAuthUser != "" && r.BasicAuthPassword != "" {
		req.SetBasicAuth(r.BasicAuthUser, r.BasicAuthPassword)
	}

	for header, value := range r.Headers {
		req.Header.Add(header, value)
	}

	response := httpResponse{}

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

func (r *httpRequest) generateUrlWithParameters() (string, error) {
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
