// Package adminschema provides access to the Dex Admin API.
//
// See http://github.com/coreos/dex
//
// Usage example:
//
//   import "google.golang.org/api/adminschema/v1"
//   ...
//   adminschemaService, err := adminschema.New(oauthHttpClient)
package adminschema

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strconv"
	"strings"

	"google.golang.org/api/googleapi"
)

// Always reference these packages, just in case the auto-generated code
// below doesn't.
var _ = bytes.NewBuffer
var _ = strconv.Itoa
var _ = fmt.Sprintf
var _ = json.NewDecoder
var _ = io.Copy
var _ = url.Parse
var _ = googleapi.Version
var _ = errors.New
var _ = strings.Replace

const apiId = "dex:v1"
const apiName = "adminschema"
const apiVersion = "v1"
const basePath = "$ENDPOINT/api/v1/"

func New(client *http.Client) (*Service, error) {
	if client == nil {
		return nil, errors.New("client is nil")
	}
	s := &Service{client: client, BasePath: basePath}
	s.Admin = NewAdminService(s)
	s.State = NewStateService(s)
	return s, nil
}

type Service struct {
	client   *http.Client
	BasePath string // API endpoint base URL

	Admin *AdminService

	State *StateService
}

func NewAdminService(s *Service) *AdminService {
	rs := &AdminService{s: s}
	return rs
}

type AdminService struct {
	s *Service
}

func NewStateService(s *Service) *StateService {
	rs := &StateService{s: s}
	return rs
}

type StateService struct {
	s *Service
}

type Admin struct {
	Email string `json:"email,omitempty"`

	Id string `json:"id,omitempty"`

	Password string `json:"password,omitempty"`
}

type State struct {
	AdminUserCreated bool `json:"AdminUserCreated,omitempty"`
}

// method id "dex.admin.Admin.Create":

type AdminCreateCall struct {
	s     *Service
	admin *Admin
	opt_  map[string]interface{}
}

// Create: Create a new admin user.
func (r *AdminService) Create(admin *Admin) *AdminCreateCall {
	c := &AdminCreateCall{s: r.s, opt_: make(map[string]interface{})}
	c.admin = admin
	return c
}

// Fields allows partial responses to be retrieved.
// See https://developers.google.com/gdata/docs/2.0/basics#PartialResponse
// for more information.
func (c *AdminCreateCall) Fields(s ...googleapi.Field) *AdminCreateCall {
	c.opt_["fields"] = googleapi.CombineFields(s)
	return c
}

func (c *AdminCreateCall) Do() (*Admin, error) {
	var body io.Reader = nil
	body, err := googleapi.WithoutDataWrapper.JSONReader(c.admin)
	if err != nil {
		return nil, err
	}
	ctype := "application/json"
	params := make(url.Values)
	params.Set("alt", "json")
	if v, ok := c.opt_["fields"]; ok {
		params.Set("fields", fmt.Sprintf("%v", v))
	}
	urls := googleapi.ResolveRelative(c.s.BasePath, "admin")
	urls += "?" + params.Encode()
	req, _ := http.NewRequest("POST", urls, body)
	googleapi.SetOpaque(req.URL)
	req.Header.Set("Content-Type", ctype)
	req.Header.Set("User-Agent", "google-api-go-client/0.5")
	res, err := c.s.client.Do(req)
	if err != nil {
		return nil, err
	}
	defer googleapi.CloseBody(res)
	if err := googleapi.CheckResponse(res); err != nil {
		return nil, err
	}
	var ret *Admin
	if err := json.NewDecoder(res.Body).Decode(&ret); err != nil {
		return nil, err
	}
	return ret, nil
	// {
	//   "description": "Create a new admin user.",
	//   "httpMethod": "POST",
	//   "id": "dex.admin.Admin.Create",
	//   "path": "admin",
	//   "request": {
	//     "$ref": "Admin"
	//   },
	//   "response": {
	//     "$ref": "Admin"
	//   }
	// }

}

// method id "dex.admin.Admin.Get":

type AdminGetCall struct {
	s    *Service
	id   string
	opt_ map[string]interface{}
}

// Get: Retrieve information about an admin user.
func (r *AdminService) Get(id string) *AdminGetCall {
	c := &AdminGetCall{s: r.s, opt_: make(map[string]interface{})}
	c.id = id
	return c
}

// Fields allows partial responses to be retrieved.
// See https://developers.google.com/gdata/docs/2.0/basics#PartialResponse
// for more information.
func (c *AdminGetCall) Fields(s ...googleapi.Field) *AdminGetCall {
	c.opt_["fields"] = googleapi.CombineFields(s)
	return c
}

func (c *AdminGetCall) Do() (*Admin, error) {
	var body io.Reader = nil
	params := make(url.Values)
	params.Set("alt", "json")
	if v, ok := c.opt_["fields"]; ok {
		params.Set("fields", fmt.Sprintf("%v", v))
	}
	urls := googleapi.ResolveRelative(c.s.BasePath, "admin/{id}")
	urls += "?" + params.Encode()
	req, _ := http.NewRequest("GET", urls, body)
	googleapi.Expand(req.URL, map[string]string{
		"id": c.id,
	})
	req.Header.Set("User-Agent", "google-api-go-client/0.5")
	res, err := c.s.client.Do(req)
	if err != nil {
		return nil, err
	}
	defer googleapi.CloseBody(res)
	if err := googleapi.CheckResponse(res); err != nil {
		return nil, err
	}
	var ret *Admin
	if err := json.NewDecoder(res.Body).Decode(&ret); err != nil {
		return nil, err
	}
	return ret, nil
	// {
	//   "description": "Retrieve information about an admin user.",
	//   "httpMethod": "GET",
	//   "id": "dex.admin.Admin.Get",
	//   "parameterOrder": [
	//     "id"
	//   ],
	//   "parameters": {
	//     "id": {
	//       "location": "path",
	//       "required": true,
	//       "type": "string"
	//     }
	//   },
	//   "path": "admin/{id}",
	//   "response": {
	//     "$ref": "Admin"
	//   }
	// }

}

// method id "dex.admin.State.Get":

type StateGetCall struct {
	s    *Service
	opt_ map[string]interface{}
}

// Get: Get the state of the Dex DB
func (r *StateService) Get() *StateGetCall {
	c := &StateGetCall{s: r.s, opt_: make(map[string]interface{})}
	return c
}

// Fields allows partial responses to be retrieved.
// See https://developers.google.com/gdata/docs/2.0/basics#PartialResponse
// for more information.
func (c *StateGetCall) Fields(s ...googleapi.Field) *StateGetCall {
	c.opt_["fields"] = googleapi.CombineFields(s)
	return c
}

func (c *StateGetCall) Do() (*State, error) {
	var body io.Reader = nil
	params := make(url.Values)
	params.Set("alt", "json")
	if v, ok := c.opt_["fields"]; ok {
		params.Set("fields", fmt.Sprintf("%v", v))
	}
	urls := googleapi.ResolveRelative(c.s.BasePath, "state")
	urls += "?" + params.Encode()
	req, _ := http.NewRequest("GET", urls, body)
	googleapi.SetOpaque(req.URL)
	req.Header.Set("User-Agent", "google-api-go-client/0.5")
	res, err := c.s.client.Do(req)
	if err != nil {
		return nil, err
	}
	defer googleapi.CloseBody(res)
	if err := googleapi.CheckResponse(res); err != nil {
		return nil, err
	}
	var ret *State
	if err := json.NewDecoder(res.Body).Decode(&ret); err != nil {
		return nil, err
	}
	return ret, nil
	// {
	//   "description": "Get the state of the Dex DB",
	//   "httpMethod": "GET",
	//   "id": "dex.admin.State.Get",
	//   "path": "state",
	//   "response": {
	//     "$ref": "State"
	//   }
	// }

}
