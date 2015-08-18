package simplehttp

import (
	"github.com/mbanzon/dummyserver"
	"log"
	"strconv"
	"testing"
)

var (
	server     *dummyserver.DummyServer
	dummyurl   string
	invalidurl string
)

type testStruct struct {
	Value1 string `json:"value1" xml:"value1"`
	Value2 string `json:"value2" xml:"value2"`
	Value3 string `json:"value3" xml:"value3"`
}

func init() {
	server = dummyserver.NewRandomServer()
	go func() {
		err := server.Start()
		log.Fatal(err)
	}()
	dummyurl = "http://localhost:" + strconv.Itoa(server.GetPort()) + "/"
	invalidurl = "invalid://invalid"
}

func TestAddParameters(t *testing.T) {
	request := NewHTTPRequest(dummyurl)
	request.AddParameter("p1", "v1")

	request.MakeGetRequest()
}
