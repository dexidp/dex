package simplehttp

import (
	"bytes"
	"github.com/mbanzon/callbackenv"
	"io/ioutil"
	"testing"
)

const (
	FILE_ENV = "SIMPLEHTTP_TEST_FILE"
)

func TestFormDataPayloadPost(t *testing.T) {
	payload := NewFormDataPayload()
	payload.AddValue("key", "value")

	buf := &bytes.Buffer{}
	buf.Write([]byte("testing testing testing"))

	rc := ioutil.NopCloser(buf)
	payload.AddReadCloser("foo", "bar", rc)

	callbackenv.RequireEnv(FILE_ENV,
		func(file string) {
			payload.AddFile("file", file)
		}, nil)

	request := NewHTTPRequest(dummyurl)
	request.MakePostRequest(payload)
}

func TestUrlEncodedPayloadPost(t *testing.T) {
	payload := NewUrlEncodedPayload()
	payload.AddValue("key", "value")
	request := NewHTTPRequest(dummyurl)
	request.MakePostRequest(payload)
}
