package simplehttp

import (
	"encoding/json"
	"encoding/xml"
)

// Parses the HTTPResponse as JSON to the given interface.
func (r *HTTPResponse) ParseFromJSON(v interface{}) error {
	return json.Unmarshal(r.Data, v)
}

func (r *HTTPResponse) ParseFromXML(v interface{}) error {
	return xml.Unmarshal(r.Data, v)
}
