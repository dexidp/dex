package simplehttp

import (
	"testing"
)

func TestShorthandFailingPayload(t *testing.T) {
	Request{
		Url:  dummyurl,
		Data: nil,
	}.Post()
}

func TestShorthandGet(t *testing.T) {
	code, _, err := Request{
		Url:       dummyurl,
		UserAgent: "simplehttp go test",
	}.Get()

	if code == -1 || err != nil {
		t.Fail()
	}
}

func TestShorthandPost(t *testing.T) {
	code, _, err := Request{
		Url:       dummyurl,
		Data:      []byte("foobar"),
		UserAgent: "simplehttp go test",
		Authentication: BasicAuthentication{
			User:     "test",
			Password: "test",
		},
	}.Post()

	if code == -1 || err != nil {
		t.Fail()
	}
}

func TestShorthandPut(t *testing.T) {
	code, _, err := Request{
		Url:       dummyurl,
		Data:      []byte("foobar"),
		UserAgent: "simplehttp go test",
	}.Put()

	if code == -1 || err != nil {
		t.Fail()
	}
}

func TestShorthandDelete(t *testing.T) {
	code, _, err := Request{
		Url:       dummyurl,
		UserAgent: "simplehttp go test",
	}.Delete()

	if code == -1 || err != nil {
		t.Fail()
	}
}

func TestFailingShorthandGet(t *testing.T) {
	code, _, err := Request{
		Url: invalidurl,
	}.Get()

	if code != -1 || err == nil {
		t.Fail()
	}
}

func TestFailingShorthandPost(t *testing.T) {
	code, _, err := Request{
		Url:  invalidurl,
		Data: []byte("foobar"),
	}.Post()

	if code != -1 || err == nil {
		t.Fail()
	}
}

func TestFailingShorthandPut(t *testing.T) {
	code, _, err := Request{
		Url:  invalidurl,
		Data: []byte("foobar"),
	}.Put()

	if code != -1 || err == nil {
		t.Fail()
	}
}

func TestFailingShorthandDelete(t *testing.T) {
	code, _, err := Request{
		Url: invalidurl,
	}.Delete()

	if code != -1 || err == nil {
		t.Fail()
	}
}
