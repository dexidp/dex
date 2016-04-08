package simplehttp

import (
	"encoding/json"
	"encoding/xml"
	"testing"
)

func TestParsingGetFromJson(t *testing.T) {
	tmp := testStruct{
		Value1: "1",
		Value2: "2",
		Value3: "3",
	}

	data, err := json.Marshal(tmp)
	if err != nil {
		t.Fail()
	}

	server.SetNextResponse(data)

	request := NewHTTPRequest(dummyurl)
	var retVal testStruct
	err = request.GetResponseFromJSON(&retVal)

	if err != nil {
		t.Fail()
	}

	if tmp.Value1 != retVal.Value1 {
		t.Fail()
	}

	if tmp.Value2 != retVal.Value2 {
		t.Fail()
	}

	if tmp.Value3 != retVal.Value3 {
		t.Fail()
	}
}

func TestFailingParsingGetFromJson(t *testing.T) {
	request := NewHTTPRequest(invalidurl)
	var retVal testStruct
	err := request.GetResponseFromJSON(&retVal)

	if err == nil {
		t.Fail()
	}
}

func TestParsingPostFromJson(t *testing.T) {
	tmp := testStruct{
		Value1: "1",
		Value2: "2",
		Value3: "3",
	}

	data, err := json.Marshal(tmp)
	if err != nil {
		t.Fail()
	}

	server.SetNextResponse(data)

	request := NewHTTPRequest(dummyurl)
	var retVal testStruct
	err = request.PostResponseFromJSON(nil, &retVal)

	if err != nil {
		t.Fail()
	}

	if tmp.Value1 != retVal.Value1 {
		t.Fail()
	}

	if tmp.Value2 != retVal.Value2 {
		t.Fail()
	}

	if tmp.Value3 != retVal.Value3 {
		t.Fail()
	}
}

func TestFailingParsingPostFromJson(t *testing.T) {
	request := NewHTTPRequest(invalidurl)
	var retVal testStruct
	err := request.PostResponseFromJSON(nil, &retVal)

	if err == nil {
		t.Fail()
	}
}

func TestParsingGetFromXml(t *testing.T) {
	tmp := testStruct{
		Value1: "1",
		Value2: "2",
		Value3: "3",
	}

	data, err := xml.Marshal(tmp)
	if err != nil {
		t.Fail()
	}

	server.SetNextResponse(data)

	request := NewHTTPRequest(dummyurl)
	var retVal testStruct
	response, err := request.MakeGetRequest()
	response.ParseFromXML(&retVal)

	if err != nil {
		t.Fail()
	}

	if tmp.Value1 != retVal.Value1 {
		t.Fail()
	}

	if tmp.Value2 != retVal.Value2 {
		t.Fail()
	}

	if tmp.Value3 != retVal.Value3 {
		t.Fail()
	}
}
