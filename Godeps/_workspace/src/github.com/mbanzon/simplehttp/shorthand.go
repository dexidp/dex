package simplehttp

// Type to encapsulate basic authentication for requests.
type BasicAuthentication struct {
	User     string
	Password string
}

// Type to wrap requests.
type Request struct {
	Url            string
	Authentication BasicAuthentication
	UserAgent      string
	Data           []byte
}

func createHttpRequest(req Request) *HTTPRequest {
	r := NewHTTPRequest(req.Url)
	if req.Authentication.User != "" {
		r.SetBasicAuth(req.Authentication.User, req.Authentication.Password)
	}
	if req.UserAgent != "" {
		r.AddHeader("User-Agent", req.UserAgent)
	}
	return r
}

func (r Request) Get() (int, []byte, error) {
	req := createHttpRequest(r)
	res, err := req.MakeGetRequest()
	if err == nil {
		return res.Code, res.Data, err
	} else {
		return -1, nil, err
	}
}

func (r Request) Post() (int, []byte, error) {
	req := createHttpRequest(r)
	var payload Payload = nil
	if r.Data != nil {
		payload = NewRawPayload(r.Data)
	}
	res, err := req.MakePostRequest(payload)
	if err == nil {
		return res.Code, res.Data, err
	} else {
		return -1, nil, err
	}
}

func (r Request) Put() (int, []byte, error) {
	req := createHttpRequest(r)
	var payload Payload = nil
	if r.Data != nil {
		payload = NewRawPayload(r.Data)
	}
	res, err := req.MakePutRequest(payload)
	if err == nil {
		return res.Code, res.Data, err
	} else {
		return -1, nil, err
	}
}

func (r Request) Delete() (int, []byte, error) {
	req := createHttpRequest(r)
	res, err := req.MakeDeleteRequest()
	if err == nil {
		return res.Code, res.Data, err
	} else {
		return -1, nil, err
	}
}
