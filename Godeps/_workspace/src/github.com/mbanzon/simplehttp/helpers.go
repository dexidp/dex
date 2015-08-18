package simplehttp

func (r *HTTPRequest) GetResponseFromJSON(v interface{}) error {
	response, err := r.MakeGetRequest()
	if err != nil {
		return err
	}
	return response.ParseFromJSON(v)
}

func (r *HTTPRequest) PostResponseFromJSON(payload Payload, v interface{}) error {
	response, err := r.MakePostRequest(payload)
	if err != nil {
		return err
	}
	return response.ParseFromJSON(v)
}
