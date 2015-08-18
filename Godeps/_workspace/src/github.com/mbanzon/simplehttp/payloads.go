package simplehttp

import (
	"bytes"
	"io"
	"mime/multipart"
	"net/url"
	"os"
	"path"
)

type keyValuePair struct {
	key   string
	value string
}

type keyNameRC struct {
	key   string
	name  string
	value io.ReadCloser
}

type Payload interface {
	GetPayloadBuffer() (*bytes.Buffer, error)
	GetContentType() string
}

type RawPayload struct {
	Data []byte
}

type FormDataPayload struct {
	contentType string
	Values      []keyValuePair
	Files       []keyValuePair
	ReadClosers []keyNameRC
}

type UrlEncodedPayload struct {
	Values []keyValuePair
}

func NewRawPayload(data []byte) *RawPayload {
	return &RawPayload{Data: data}
}

func (r *RawPayload) GetPayloadBuffer() (*bytes.Buffer, error) {
	data := &bytes.Buffer{}
	c, err := data.Write(r.Data)
	if c != len(r.Data) || err != nil {
		return data, err
	}
	return data, nil
}

func (r *RawPayload) GetContentType() string {
	return ""
}

func NewFormDataPayload() *FormDataPayload {
	return &FormDataPayload{}
}

func (f *FormDataPayload) AddValue(key, value string) {
	f.Values = append(f.Values, keyValuePair{key: key, value: value})
}

func (f *FormDataPayload) AddFile(key, file string) {
	f.Files = append(f.Files, keyValuePair{key: key, value: file})
}

func (f *FormDataPayload) AddReadCloser(key, name string, rc io.ReadCloser) {
	f.ReadClosers = append(f.ReadClosers, keyNameRC{key: key, name: name, value: rc})
}

func (f *FormDataPayload) GetPayloadBuffer() (*bytes.Buffer, error) {
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

func (f *FormDataPayload) GetContentType() string {
	if f.contentType == "" {
		f.GetPayloadBuffer()
	}
	return f.contentType
}

func NewUrlEncodedPayload() *UrlEncodedPayload {
	return &UrlEncodedPayload{}
}

func (f *UrlEncodedPayload) AddValue(key, value string) {
	f.Values = append(f.Values, keyValuePair{key: key, value: value})
}

func (f *UrlEncodedPayload) GetPayloadBuffer() (*bytes.Buffer, error) {
	data := url.Values{}
	for _, keyVal := range f.Values {
		data.Add(keyVal.key, keyVal.value)
	}
	return bytes.NewBufferString(data.Encode()), nil
}

func (f *UrlEncodedPayload) GetContentType() string {
	return "application/x-www-form-urlencoded"
}
