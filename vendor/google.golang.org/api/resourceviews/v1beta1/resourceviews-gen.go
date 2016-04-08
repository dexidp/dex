// Package resourceviews provides access to the Resource Views API.
//
// See https://developers.google.com/compute/
//
// Usage example:
//
//   import "google.golang.org/api/resourceviews/v1beta1"
//   ...
//   resourceviewsService, err := resourceviews.New(oauthHttpClient)
package resourceviews

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"google.golang.org/api/googleapi"
	"io"
	"net/http"
	"net/url"
	"strconv"
	"strings"
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

const apiId = "resourceviews:v1beta1"
const apiName = "resourceviews"
const apiVersion = "v1beta1"
const basePath = "https://www.googleapis.com/resourceviews/v1beta1/projects/"

// OAuth2 scopes used by this API.
const (
	// View and manage your data across Google Cloud Platform services
	CloudPlatformScope = "https://www.googleapis.com/auth/cloud-platform"

	// View and manage your Google Compute Engine resources
	ComputeScope = "https://www.googleapis.com/auth/compute"

	// View your Google Compute Engine resources
	ComputeReadonlyScope = "https://www.googleapis.com/auth/compute.readonly"

	// View and manage your Google Cloud Platform management resources and
	// deployment status information
	NdevCloudmanScope = "https://www.googleapis.com/auth/ndev.cloudman"

	// View your Google Cloud Platform management resources and deployment
	// status information
	NdevCloudmanReadonlyScope = "https://www.googleapis.com/auth/ndev.cloudman.readonly"
)

func New(client *http.Client) (*Service, error) {
	if client == nil {
		return nil, errors.New("client is nil")
	}
	s := &Service{client: client, BasePath: basePath}
	s.RegionViews = NewRegionViewsService(s)
	s.ZoneViews = NewZoneViewsService(s)
	return s, nil
}

type Service struct {
	client   *http.Client
	BasePath string // API endpoint base URL

	RegionViews *RegionViewsService

	ZoneViews *ZoneViewsService
}

func NewRegionViewsService(s *Service) *RegionViewsService {
	rs := &RegionViewsService{s: s}
	return rs
}

type RegionViewsService struct {
	s *Service
}

func NewZoneViewsService(s *Service) *ZoneViewsService {
	rs := &ZoneViewsService{s: s}
	return rs
}

type ZoneViewsService struct {
	s *Service
}

type Label struct {
	// Key: Key of the label.
	Key string `json:"key,omitempty"`

	// Value: Value of the label.
	Value string `json:"value,omitempty"`
}

type RegionViewsAddResourcesRequest struct {
	// Resources: The list of resources to be added.
	Resources []string `json:"resources,omitempty"`
}

type RegionViewsInsertResponse struct {
	// Resource: The resource view object inserted.
	Resource *ResourceView `json:"resource,omitempty"`
}

type RegionViewsListResourcesResponse struct {
	// Members: The resources in the view.
	Members []string `json:"members,omitempty"`

	// NextPageToken: A token used for pagination.
	NextPageToken string `json:"nextPageToken,omitempty"`
}

type RegionViewsListResponse struct {
	// NextPageToken: A token used for pagination.
	NextPageToken string `json:"nextPageToken,omitempty"`

	// ResourceViews: The list of resource views that meet the criteria.
	ResourceViews []*ResourceView `json:"resourceViews,omitempty"`
}

type RegionViewsRemoveResourcesRequest struct {
	// Resources: The list of resources to be removed.
	Resources []string `json:"resources,omitempty"`
}

type ResourceView struct {
	// CreationTime: The creation time of the resource view.
	CreationTime string `json:"creationTime,omitempty"`

	// Description: The detailed description of the resource view.
	Description string `json:"description,omitempty"`

	// Id: [Output Only] The ID of the resource view.
	Id string `json:"id,omitempty"`

	// Kind: Type of the resource.
	Kind string `json:"kind,omitempty"`

	// Labels: The labels for events.
	Labels []*Label `json:"labels,omitempty"`

	// LastModified: The last modified time of the view. Not supported yet.
	LastModified string `json:"lastModified,omitempty"`

	// Members: A list of all resources in the resource view.
	Members []string `json:"members,omitempty"`

	// Name: The name of the resource view.
	Name string `json:"name,omitempty"`

	// NumMembers: The total number of resources in the resource view.
	NumMembers int64 `json:"numMembers,omitempty"`

	// SelfLink: [Output Only] A self-link to the resource view.
	SelfLink string `json:"selfLink,omitempty"`
}

type ZoneViewsAddResourcesRequest struct {
	// Resources: The list of resources to be added.
	Resources []string `json:"resources,omitempty"`
}

type ZoneViewsInsertResponse struct {
	// Resource: The resource view object that has been inserted.
	Resource *ResourceView `json:"resource,omitempty"`
}

type ZoneViewsListResourcesResponse struct {
	// Members: The full URL of resources in the view.
	Members []string `json:"members,omitempty"`

	// NextPageToken: A token used for pagination.
	NextPageToken string `json:"nextPageToken,omitempty"`
}

type ZoneViewsListResponse struct {
	// NextPageToken: A token used for pagination.
	NextPageToken string `json:"nextPageToken,omitempty"`

	// ResourceViews: The result that contains all resource views that meet
	// the criteria.
	ResourceViews []*ResourceView `json:"resourceViews,omitempty"`
}

type ZoneViewsRemoveResourcesRequest struct {
	// Resources: The list of resources to be removed.
	Resources []string `json:"resources,omitempty"`
}

// method id "resourceviews.regionViews.addresources":

type RegionViewsAddresourcesCall struct {
	s                              *Service
	projectName                    string
	region                         string
	resourceViewName               string
	regionviewsaddresourcesrequest *RegionViewsAddResourcesRequest
	opt_                           map[string]interface{}
}

// Addresources: Add resources to the view.
func (r *RegionViewsService) Addresources(projectName string, region string, resourceViewName string, regionviewsaddresourcesrequest *RegionViewsAddResourcesRequest) *RegionViewsAddresourcesCall {
	c := &RegionViewsAddresourcesCall{s: r.s, opt_: make(map[string]interface{})}
	c.projectName = projectName
	c.region = region
	c.resourceViewName = resourceViewName
	c.regionviewsaddresourcesrequest = regionviewsaddresourcesrequest
	return c
}

// Fields allows partial responses to be retrieved.
// See https://developers.google.com/gdata/docs/2.0/basics#PartialResponse
// for more information.
func (c *RegionViewsAddresourcesCall) Fields(s ...googleapi.Field) *RegionViewsAddresourcesCall {
	c.opt_["fields"] = googleapi.CombineFields(s)
	return c
}

func (c *RegionViewsAddresourcesCall) Do() error {
	var body io.Reader = nil
	body, err := googleapi.WithoutDataWrapper.JSONReader(c.regionviewsaddresourcesrequest)
	if err != nil {
		return err
	}
	ctype := "application/json"
	params := make(url.Values)
	params.Set("alt", "json")
	if v, ok := c.opt_["fields"]; ok {
		params.Set("fields", fmt.Sprintf("%v", v))
	}
	urls := googleapi.ResolveRelative(c.s.BasePath, "{projectName}/regions/{region}/resourceViews/{resourceViewName}/addResources")
	urls += "?" + params.Encode()
	req, _ := http.NewRequest("POST", urls, body)
	googleapi.Expand(req.URL, map[string]string{
		"projectName":      c.projectName,
		"region":           c.region,
		"resourceViewName": c.resourceViewName,
	})
	req.Header.Set("Content-Type", ctype)
	req.Header.Set("User-Agent", "google-api-go-client/0.5")
	res, err := c.s.client.Do(req)
	if err != nil {
		return err
	}
	defer googleapi.CloseBody(res)
	if err := googleapi.CheckResponse(res); err != nil {
		return err
	}
	return nil
	// {
	//   "description": "Add resources to the view.",
	//   "httpMethod": "POST",
	//   "id": "resourceviews.regionViews.addresources",
	//   "parameterOrder": [
	//     "projectName",
	//     "region",
	//     "resourceViewName"
	//   ],
	//   "parameters": {
	//     "projectName": {
	//       "description": "The project name of the resource view.",
	//       "location": "path",
	//       "required": true,
	//       "type": "string"
	//     },
	//     "region": {
	//       "description": "The region name of the resource view.",
	//       "location": "path",
	//       "required": true,
	//       "type": "string"
	//     },
	//     "resourceViewName": {
	//       "description": "The name of the resource view.",
	//       "location": "path",
	//       "required": true,
	//       "type": "string"
	//     }
	//   },
	//   "path": "{projectName}/regions/{region}/resourceViews/{resourceViewName}/addResources",
	//   "request": {
	//     "$ref": "RegionViewsAddResourcesRequest"
	//   },
	//   "scopes": [
	//     "https://www.googleapis.com/auth/cloud-platform",
	//     "https://www.googleapis.com/auth/compute",
	//     "https://www.googleapis.com/auth/ndev.cloudman"
	//   ]
	// }

}

// method id "resourceviews.regionViews.delete":

type RegionViewsDeleteCall struct {
	s                *Service
	projectName      string
	region           string
	resourceViewName string
	opt_             map[string]interface{}
}

// Delete: Delete a resource view.
func (r *RegionViewsService) Delete(projectName string, region string, resourceViewName string) *RegionViewsDeleteCall {
	c := &RegionViewsDeleteCall{s: r.s, opt_: make(map[string]interface{})}
	c.projectName = projectName
	c.region = region
	c.resourceViewName = resourceViewName
	return c
}

// Fields allows partial responses to be retrieved.
// See https://developers.google.com/gdata/docs/2.0/basics#PartialResponse
// for more information.
func (c *RegionViewsDeleteCall) Fields(s ...googleapi.Field) *RegionViewsDeleteCall {
	c.opt_["fields"] = googleapi.CombineFields(s)
	return c
}

func (c *RegionViewsDeleteCall) Do() error {
	var body io.Reader = nil
	params := make(url.Values)
	params.Set("alt", "json")
	if v, ok := c.opt_["fields"]; ok {
		params.Set("fields", fmt.Sprintf("%v", v))
	}
	urls := googleapi.ResolveRelative(c.s.BasePath, "{projectName}/regions/{region}/resourceViews/{resourceViewName}")
	urls += "?" + params.Encode()
	req, _ := http.NewRequest("DELETE", urls, body)
	googleapi.Expand(req.URL, map[string]string{
		"projectName":      c.projectName,
		"region":           c.region,
		"resourceViewName": c.resourceViewName,
	})
	req.Header.Set("User-Agent", "google-api-go-client/0.5")
	res, err := c.s.client.Do(req)
	if err != nil {
		return err
	}
	defer googleapi.CloseBody(res)
	if err := googleapi.CheckResponse(res); err != nil {
		return err
	}
	return nil
	// {
	//   "description": "Delete a resource view.",
	//   "httpMethod": "DELETE",
	//   "id": "resourceviews.regionViews.delete",
	//   "parameterOrder": [
	//     "projectName",
	//     "region",
	//     "resourceViewName"
	//   ],
	//   "parameters": {
	//     "projectName": {
	//       "description": "The project name of the resource view.",
	//       "location": "path",
	//       "required": true,
	//       "type": "string"
	//     },
	//     "region": {
	//       "description": "The region name of the resource view.",
	//       "location": "path",
	//       "required": true,
	//       "type": "string"
	//     },
	//     "resourceViewName": {
	//       "description": "The name of the resource view.",
	//       "location": "path",
	//       "required": true,
	//       "type": "string"
	//     }
	//   },
	//   "path": "{projectName}/regions/{region}/resourceViews/{resourceViewName}",
	//   "scopes": [
	//     "https://www.googleapis.com/auth/cloud-platform",
	//     "https://www.googleapis.com/auth/compute",
	//     "https://www.googleapis.com/auth/ndev.cloudman"
	//   ]
	// }

}

// method id "resourceviews.regionViews.get":

type RegionViewsGetCall struct {
	s                *Service
	projectName      string
	region           string
	resourceViewName string
	opt_             map[string]interface{}
}

// Get: Get the information of a resource view.
func (r *RegionViewsService) Get(projectName string, region string, resourceViewName string) *RegionViewsGetCall {
	c := &RegionViewsGetCall{s: r.s, opt_: make(map[string]interface{})}
	c.projectName = projectName
	c.region = region
	c.resourceViewName = resourceViewName
	return c
}

// Fields allows partial responses to be retrieved.
// See https://developers.google.com/gdata/docs/2.0/basics#PartialResponse
// for more information.
func (c *RegionViewsGetCall) Fields(s ...googleapi.Field) *RegionViewsGetCall {
	c.opt_["fields"] = googleapi.CombineFields(s)
	return c
}

func (c *RegionViewsGetCall) Do() (*ResourceView, error) {
	var body io.Reader = nil
	params := make(url.Values)
	params.Set("alt", "json")
	if v, ok := c.opt_["fields"]; ok {
		params.Set("fields", fmt.Sprintf("%v", v))
	}
	urls := googleapi.ResolveRelative(c.s.BasePath, "{projectName}/regions/{region}/resourceViews/{resourceViewName}")
	urls += "?" + params.Encode()
	req, _ := http.NewRequest("GET", urls, body)
	googleapi.Expand(req.URL, map[string]string{
		"projectName":      c.projectName,
		"region":           c.region,
		"resourceViewName": c.resourceViewName,
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
	var ret *ResourceView
	if err := json.NewDecoder(res.Body).Decode(&ret); err != nil {
		return nil, err
	}
	return ret, nil
	// {
	//   "description": "Get the information of a resource view.",
	//   "httpMethod": "GET",
	//   "id": "resourceviews.regionViews.get",
	//   "parameterOrder": [
	//     "projectName",
	//     "region",
	//     "resourceViewName"
	//   ],
	//   "parameters": {
	//     "projectName": {
	//       "description": "The project name of the resource view.",
	//       "location": "path",
	//       "required": true,
	//       "type": "string"
	//     },
	//     "region": {
	//       "description": "The region name of the resource view.",
	//       "location": "path",
	//       "required": true,
	//       "type": "string"
	//     },
	//     "resourceViewName": {
	//       "description": "The name of the resource view.",
	//       "location": "path",
	//       "required": true,
	//       "type": "string"
	//     }
	//   },
	//   "path": "{projectName}/regions/{region}/resourceViews/{resourceViewName}",
	//   "response": {
	//     "$ref": "ResourceView"
	//   },
	//   "scopes": [
	//     "https://www.googleapis.com/auth/cloud-platform",
	//     "https://www.googleapis.com/auth/compute",
	//     "https://www.googleapis.com/auth/compute.readonly",
	//     "https://www.googleapis.com/auth/ndev.cloudman",
	//     "https://www.googleapis.com/auth/ndev.cloudman.readonly"
	//   ]
	// }

}

// method id "resourceviews.regionViews.insert":

type RegionViewsInsertCall struct {
	s            *Service
	projectName  string
	region       string
	resourceview *ResourceView
	opt_         map[string]interface{}
}

// Insert: Create a resource view.
func (r *RegionViewsService) Insert(projectName string, region string, resourceview *ResourceView) *RegionViewsInsertCall {
	c := &RegionViewsInsertCall{s: r.s, opt_: make(map[string]interface{})}
	c.projectName = projectName
	c.region = region
	c.resourceview = resourceview
	return c
}

// Fields allows partial responses to be retrieved.
// See https://developers.google.com/gdata/docs/2.0/basics#PartialResponse
// for more information.
func (c *RegionViewsInsertCall) Fields(s ...googleapi.Field) *RegionViewsInsertCall {
	c.opt_["fields"] = googleapi.CombineFields(s)
	return c
}

func (c *RegionViewsInsertCall) Do() (*RegionViewsInsertResponse, error) {
	var body io.Reader = nil
	body, err := googleapi.WithoutDataWrapper.JSONReader(c.resourceview)
	if err != nil {
		return nil, err
	}
	ctype := "application/json"
	params := make(url.Values)
	params.Set("alt", "json")
	if v, ok := c.opt_["fields"]; ok {
		params.Set("fields", fmt.Sprintf("%v", v))
	}
	urls := googleapi.ResolveRelative(c.s.BasePath, "{projectName}/regions/{region}/resourceViews")
	urls += "?" + params.Encode()
	req, _ := http.NewRequest("POST", urls, body)
	googleapi.Expand(req.URL, map[string]string{
		"projectName": c.projectName,
		"region":      c.region,
	})
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
	var ret *RegionViewsInsertResponse
	if err := json.NewDecoder(res.Body).Decode(&ret); err != nil {
		return nil, err
	}
	return ret, nil
	// {
	//   "description": "Create a resource view.",
	//   "httpMethod": "POST",
	//   "id": "resourceviews.regionViews.insert",
	//   "parameterOrder": [
	//     "projectName",
	//     "region"
	//   ],
	//   "parameters": {
	//     "projectName": {
	//       "description": "The project name of the resource view.",
	//       "location": "path",
	//       "required": true,
	//       "type": "string"
	//     },
	//     "region": {
	//       "description": "The region name of the resource view.",
	//       "location": "path",
	//       "required": true,
	//       "type": "string"
	//     }
	//   },
	//   "path": "{projectName}/regions/{region}/resourceViews",
	//   "request": {
	//     "$ref": "ResourceView"
	//   },
	//   "response": {
	//     "$ref": "RegionViewsInsertResponse"
	//   },
	//   "scopes": [
	//     "https://www.googleapis.com/auth/cloud-platform",
	//     "https://www.googleapis.com/auth/compute",
	//     "https://www.googleapis.com/auth/ndev.cloudman"
	//   ]
	// }

}

// method id "resourceviews.regionViews.list":

type RegionViewsListCall struct {
	s           *Service
	projectName string
	region      string
	opt_        map[string]interface{}
}

// List: List resource views.
func (r *RegionViewsService) List(projectName string, region string) *RegionViewsListCall {
	c := &RegionViewsListCall{s: r.s, opt_: make(map[string]interface{})}
	c.projectName = projectName
	c.region = region
	return c
}

// MaxResults sets the optional parameter "maxResults": Maximum count of
// results to be returned. Acceptable values are 0 to 5000, inclusive.
// (Default: 5000)
func (c *RegionViewsListCall) MaxResults(maxResults int64) *RegionViewsListCall {
	c.opt_["maxResults"] = maxResults
	return c
}

// PageToken sets the optional parameter "pageToken": Specifies a
// nextPageToken returned by a previous list request. This token can be
// used to request the next page of results from a previous list
// request.
func (c *RegionViewsListCall) PageToken(pageToken string) *RegionViewsListCall {
	c.opt_["pageToken"] = pageToken
	return c
}

// Fields allows partial responses to be retrieved.
// See https://developers.google.com/gdata/docs/2.0/basics#PartialResponse
// for more information.
func (c *RegionViewsListCall) Fields(s ...googleapi.Field) *RegionViewsListCall {
	c.opt_["fields"] = googleapi.CombineFields(s)
	return c
}

func (c *RegionViewsListCall) Do() (*RegionViewsListResponse, error) {
	var body io.Reader = nil
	params := make(url.Values)
	params.Set("alt", "json")
	if v, ok := c.opt_["maxResults"]; ok {
		params.Set("maxResults", fmt.Sprintf("%v", v))
	}
	if v, ok := c.opt_["pageToken"]; ok {
		params.Set("pageToken", fmt.Sprintf("%v", v))
	}
	if v, ok := c.opt_["fields"]; ok {
		params.Set("fields", fmt.Sprintf("%v", v))
	}
	urls := googleapi.ResolveRelative(c.s.BasePath, "{projectName}/regions/{region}/resourceViews")
	urls += "?" + params.Encode()
	req, _ := http.NewRequest("GET", urls, body)
	googleapi.Expand(req.URL, map[string]string{
		"projectName": c.projectName,
		"region":      c.region,
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
	var ret *RegionViewsListResponse
	if err := json.NewDecoder(res.Body).Decode(&ret); err != nil {
		return nil, err
	}
	return ret, nil
	// {
	//   "description": "List resource views.",
	//   "httpMethod": "GET",
	//   "id": "resourceviews.regionViews.list",
	//   "parameterOrder": [
	//     "projectName",
	//     "region"
	//   ],
	//   "parameters": {
	//     "maxResults": {
	//       "default": "5000",
	//       "description": "Maximum count of results to be returned. Acceptable values are 0 to 5000, inclusive. (Default: 5000)",
	//       "format": "int32",
	//       "location": "query",
	//       "maximum": "5000",
	//       "minimum": "0",
	//       "type": "integer"
	//     },
	//     "pageToken": {
	//       "description": "Specifies a nextPageToken returned by a previous list request. This token can be used to request the next page of results from a previous list request.",
	//       "location": "query",
	//       "type": "string"
	//     },
	//     "projectName": {
	//       "description": "The project name of the resource view.",
	//       "location": "path",
	//       "required": true,
	//       "type": "string"
	//     },
	//     "region": {
	//       "description": "The region name of the resource view.",
	//       "location": "path",
	//       "required": true,
	//       "type": "string"
	//     }
	//   },
	//   "path": "{projectName}/regions/{region}/resourceViews",
	//   "response": {
	//     "$ref": "RegionViewsListResponse"
	//   },
	//   "scopes": [
	//     "https://www.googleapis.com/auth/cloud-platform",
	//     "https://www.googleapis.com/auth/compute",
	//     "https://www.googleapis.com/auth/compute.readonly",
	//     "https://www.googleapis.com/auth/ndev.cloudman",
	//     "https://www.googleapis.com/auth/ndev.cloudman.readonly"
	//   ]
	// }

}

// method id "resourceviews.regionViews.listresources":

type RegionViewsListresourcesCall struct {
	s                *Service
	projectName      string
	region           string
	resourceViewName string
	opt_             map[string]interface{}
}

// Listresources: List the resources in the view.
func (r *RegionViewsService) Listresources(projectName string, region string, resourceViewName string) *RegionViewsListresourcesCall {
	c := &RegionViewsListresourcesCall{s: r.s, opt_: make(map[string]interface{})}
	c.projectName = projectName
	c.region = region
	c.resourceViewName = resourceViewName
	return c
}

// MaxResults sets the optional parameter "maxResults": Maximum count of
// results to be returned. Acceptable values are 0 to 5000, inclusive.
// (Default: 5000)
func (c *RegionViewsListresourcesCall) MaxResults(maxResults int64) *RegionViewsListresourcesCall {
	c.opt_["maxResults"] = maxResults
	return c
}

// PageToken sets the optional parameter "pageToken": Specifies a
// nextPageToken returned by a previous list request. This token can be
// used to request the next page of results from a previous list
// request.
func (c *RegionViewsListresourcesCall) PageToken(pageToken string) *RegionViewsListresourcesCall {
	c.opt_["pageToken"] = pageToken
	return c
}

// Fields allows partial responses to be retrieved.
// See https://developers.google.com/gdata/docs/2.0/basics#PartialResponse
// for more information.
func (c *RegionViewsListresourcesCall) Fields(s ...googleapi.Field) *RegionViewsListresourcesCall {
	c.opt_["fields"] = googleapi.CombineFields(s)
	return c
}

func (c *RegionViewsListresourcesCall) Do() (*RegionViewsListResourcesResponse, error) {
	var body io.Reader = nil
	params := make(url.Values)
	params.Set("alt", "json")
	if v, ok := c.opt_["maxResults"]; ok {
		params.Set("maxResults", fmt.Sprintf("%v", v))
	}
	if v, ok := c.opt_["pageToken"]; ok {
		params.Set("pageToken", fmt.Sprintf("%v", v))
	}
	if v, ok := c.opt_["fields"]; ok {
		params.Set("fields", fmt.Sprintf("%v", v))
	}
	urls := googleapi.ResolveRelative(c.s.BasePath, "{projectName}/regions/{region}/resourceViews/{resourceViewName}/resources")
	urls += "?" + params.Encode()
	req, _ := http.NewRequest("POST", urls, body)
	googleapi.Expand(req.URL, map[string]string{
		"projectName":      c.projectName,
		"region":           c.region,
		"resourceViewName": c.resourceViewName,
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
	var ret *RegionViewsListResourcesResponse
	if err := json.NewDecoder(res.Body).Decode(&ret); err != nil {
		return nil, err
	}
	return ret, nil
	// {
	//   "description": "List the resources in the view.",
	//   "httpMethod": "POST",
	//   "id": "resourceviews.regionViews.listresources",
	//   "parameterOrder": [
	//     "projectName",
	//     "region",
	//     "resourceViewName"
	//   ],
	//   "parameters": {
	//     "maxResults": {
	//       "default": "5000",
	//       "description": "Maximum count of results to be returned. Acceptable values are 0 to 5000, inclusive. (Default: 5000)",
	//       "format": "int32",
	//       "location": "query",
	//       "maximum": "5000",
	//       "minimum": "0",
	//       "type": "integer"
	//     },
	//     "pageToken": {
	//       "description": "Specifies a nextPageToken returned by a previous list request. This token can be used to request the next page of results from a previous list request.",
	//       "location": "query",
	//       "type": "string"
	//     },
	//     "projectName": {
	//       "description": "The project name of the resource view.",
	//       "location": "path",
	//       "required": true,
	//       "type": "string"
	//     },
	//     "region": {
	//       "description": "The region name of the resource view.",
	//       "location": "path",
	//       "required": true,
	//       "type": "string"
	//     },
	//     "resourceViewName": {
	//       "description": "The name of the resource view.",
	//       "location": "path",
	//       "required": true,
	//       "type": "string"
	//     }
	//   },
	//   "path": "{projectName}/regions/{region}/resourceViews/{resourceViewName}/resources",
	//   "response": {
	//     "$ref": "RegionViewsListResourcesResponse"
	//   },
	//   "scopes": [
	//     "https://www.googleapis.com/auth/cloud-platform",
	//     "https://www.googleapis.com/auth/compute",
	//     "https://www.googleapis.com/auth/compute.readonly",
	//     "https://www.googleapis.com/auth/ndev.cloudman",
	//     "https://www.googleapis.com/auth/ndev.cloudman.readonly"
	//   ]
	// }

}

// method id "resourceviews.regionViews.removeresources":

type RegionViewsRemoveresourcesCall struct {
	s                                 *Service
	projectName                       string
	region                            string
	resourceViewName                  string
	regionviewsremoveresourcesrequest *RegionViewsRemoveResourcesRequest
	opt_                              map[string]interface{}
}

// Removeresources: Remove resources from the view.
func (r *RegionViewsService) Removeresources(projectName string, region string, resourceViewName string, regionviewsremoveresourcesrequest *RegionViewsRemoveResourcesRequest) *RegionViewsRemoveresourcesCall {
	c := &RegionViewsRemoveresourcesCall{s: r.s, opt_: make(map[string]interface{})}
	c.projectName = projectName
	c.region = region
	c.resourceViewName = resourceViewName
	c.regionviewsremoveresourcesrequest = regionviewsremoveresourcesrequest
	return c
}

// Fields allows partial responses to be retrieved.
// See https://developers.google.com/gdata/docs/2.0/basics#PartialResponse
// for more information.
func (c *RegionViewsRemoveresourcesCall) Fields(s ...googleapi.Field) *RegionViewsRemoveresourcesCall {
	c.opt_["fields"] = googleapi.CombineFields(s)
	return c
}

func (c *RegionViewsRemoveresourcesCall) Do() error {
	var body io.Reader = nil
	body, err := googleapi.WithoutDataWrapper.JSONReader(c.regionviewsremoveresourcesrequest)
	if err != nil {
		return err
	}
	ctype := "application/json"
	params := make(url.Values)
	params.Set("alt", "json")
	if v, ok := c.opt_["fields"]; ok {
		params.Set("fields", fmt.Sprintf("%v", v))
	}
	urls := googleapi.ResolveRelative(c.s.BasePath, "{projectName}/regions/{region}/resourceViews/{resourceViewName}/removeResources")
	urls += "?" + params.Encode()
	req, _ := http.NewRequest("POST", urls, body)
	googleapi.Expand(req.URL, map[string]string{
		"projectName":      c.projectName,
		"region":           c.region,
		"resourceViewName": c.resourceViewName,
	})
	req.Header.Set("Content-Type", ctype)
	req.Header.Set("User-Agent", "google-api-go-client/0.5")
	res, err := c.s.client.Do(req)
	if err != nil {
		return err
	}
	defer googleapi.CloseBody(res)
	if err := googleapi.CheckResponse(res); err != nil {
		return err
	}
	return nil
	// {
	//   "description": "Remove resources from the view.",
	//   "httpMethod": "POST",
	//   "id": "resourceviews.regionViews.removeresources",
	//   "parameterOrder": [
	//     "projectName",
	//     "region",
	//     "resourceViewName"
	//   ],
	//   "parameters": {
	//     "projectName": {
	//       "description": "The project name of the resource view.",
	//       "location": "path",
	//       "required": true,
	//       "type": "string"
	//     },
	//     "region": {
	//       "description": "The region name of the resource view.",
	//       "location": "path",
	//       "required": true,
	//       "type": "string"
	//     },
	//     "resourceViewName": {
	//       "description": "The name of the resource view.",
	//       "location": "path",
	//       "required": true,
	//       "type": "string"
	//     }
	//   },
	//   "path": "{projectName}/regions/{region}/resourceViews/{resourceViewName}/removeResources",
	//   "request": {
	//     "$ref": "RegionViewsRemoveResourcesRequest"
	//   },
	//   "scopes": [
	//     "https://www.googleapis.com/auth/cloud-platform",
	//     "https://www.googleapis.com/auth/compute",
	//     "https://www.googleapis.com/auth/ndev.cloudman"
	//   ]
	// }

}

// method id "resourceviews.zoneViews.addresources":

type ZoneViewsAddresourcesCall struct {
	s                            *Service
	projectName                  string
	zone                         string
	resourceViewName             string
	zoneviewsaddresourcesrequest *ZoneViewsAddResourcesRequest
	opt_                         map[string]interface{}
}

// Addresources: Add resources to the view.
func (r *ZoneViewsService) Addresources(projectName string, zone string, resourceViewName string, zoneviewsaddresourcesrequest *ZoneViewsAddResourcesRequest) *ZoneViewsAddresourcesCall {
	c := &ZoneViewsAddresourcesCall{s: r.s, opt_: make(map[string]interface{})}
	c.projectName = projectName
	c.zone = zone
	c.resourceViewName = resourceViewName
	c.zoneviewsaddresourcesrequest = zoneviewsaddresourcesrequest
	return c
}

// Fields allows partial responses to be retrieved.
// See https://developers.google.com/gdata/docs/2.0/basics#PartialResponse
// for more information.
func (c *ZoneViewsAddresourcesCall) Fields(s ...googleapi.Field) *ZoneViewsAddresourcesCall {
	c.opt_["fields"] = googleapi.CombineFields(s)
	return c
}

func (c *ZoneViewsAddresourcesCall) Do() error {
	var body io.Reader = nil
	body, err := googleapi.WithoutDataWrapper.JSONReader(c.zoneviewsaddresourcesrequest)
	if err != nil {
		return err
	}
	ctype := "application/json"
	params := make(url.Values)
	params.Set("alt", "json")
	if v, ok := c.opt_["fields"]; ok {
		params.Set("fields", fmt.Sprintf("%v", v))
	}
	urls := googleapi.ResolveRelative(c.s.BasePath, "{projectName}/zones/{zone}/resourceViews/{resourceViewName}/addResources")
	urls += "?" + params.Encode()
	req, _ := http.NewRequest("POST", urls, body)
	googleapi.Expand(req.URL, map[string]string{
		"projectName":      c.projectName,
		"zone":             c.zone,
		"resourceViewName": c.resourceViewName,
	})
	req.Header.Set("Content-Type", ctype)
	req.Header.Set("User-Agent", "google-api-go-client/0.5")
	res, err := c.s.client.Do(req)
	if err != nil {
		return err
	}
	defer googleapi.CloseBody(res)
	if err := googleapi.CheckResponse(res); err != nil {
		return err
	}
	return nil
	// {
	//   "description": "Add resources to the view.",
	//   "httpMethod": "POST",
	//   "id": "resourceviews.zoneViews.addresources",
	//   "parameterOrder": [
	//     "projectName",
	//     "zone",
	//     "resourceViewName"
	//   ],
	//   "parameters": {
	//     "projectName": {
	//       "description": "The project name of the resource view.",
	//       "location": "path",
	//       "required": true,
	//       "type": "string"
	//     },
	//     "resourceViewName": {
	//       "description": "The name of the resource view.",
	//       "location": "path",
	//       "required": true,
	//       "type": "string"
	//     },
	//     "zone": {
	//       "description": "The zone name of the resource view.",
	//       "location": "path",
	//       "required": true,
	//       "type": "string"
	//     }
	//   },
	//   "path": "{projectName}/zones/{zone}/resourceViews/{resourceViewName}/addResources",
	//   "request": {
	//     "$ref": "ZoneViewsAddResourcesRequest"
	//   },
	//   "scopes": [
	//     "https://www.googleapis.com/auth/cloud-platform",
	//     "https://www.googleapis.com/auth/compute",
	//     "https://www.googleapis.com/auth/ndev.cloudman"
	//   ]
	// }

}

// method id "resourceviews.zoneViews.delete":

type ZoneViewsDeleteCall struct {
	s                *Service
	projectName      string
	zone             string
	resourceViewName string
	opt_             map[string]interface{}
}

// Delete: Delete a resource view.
func (r *ZoneViewsService) Delete(projectName string, zone string, resourceViewName string) *ZoneViewsDeleteCall {
	c := &ZoneViewsDeleteCall{s: r.s, opt_: make(map[string]interface{})}
	c.projectName = projectName
	c.zone = zone
	c.resourceViewName = resourceViewName
	return c
}

// Fields allows partial responses to be retrieved.
// See https://developers.google.com/gdata/docs/2.0/basics#PartialResponse
// for more information.
func (c *ZoneViewsDeleteCall) Fields(s ...googleapi.Field) *ZoneViewsDeleteCall {
	c.opt_["fields"] = googleapi.CombineFields(s)
	return c
}

func (c *ZoneViewsDeleteCall) Do() error {
	var body io.Reader = nil
	params := make(url.Values)
	params.Set("alt", "json")
	if v, ok := c.opt_["fields"]; ok {
		params.Set("fields", fmt.Sprintf("%v", v))
	}
	urls := googleapi.ResolveRelative(c.s.BasePath, "{projectName}/zones/{zone}/resourceViews/{resourceViewName}")
	urls += "?" + params.Encode()
	req, _ := http.NewRequest("DELETE", urls, body)
	googleapi.Expand(req.URL, map[string]string{
		"projectName":      c.projectName,
		"zone":             c.zone,
		"resourceViewName": c.resourceViewName,
	})
	req.Header.Set("User-Agent", "google-api-go-client/0.5")
	res, err := c.s.client.Do(req)
	if err != nil {
		return err
	}
	defer googleapi.CloseBody(res)
	if err := googleapi.CheckResponse(res); err != nil {
		return err
	}
	return nil
	// {
	//   "description": "Delete a resource view.",
	//   "httpMethod": "DELETE",
	//   "id": "resourceviews.zoneViews.delete",
	//   "parameterOrder": [
	//     "projectName",
	//     "zone",
	//     "resourceViewName"
	//   ],
	//   "parameters": {
	//     "projectName": {
	//       "description": "The project name of the resource view.",
	//       "location": "path",
	//       "required": true,
	//       "type": "string"
	//     },
	//     "resourceViewName": {
	//       "description": "The name of the resource view.",
	//       "location": "path",
	//       "required": true,
	//       "type": "string"
	//     },
	//     "zone": {
	//       "description": "The zone name of the resource view.",
	//       "location": "path",
	//       "required": true,
	//       "type": "string"
	//     }
	//   },
	//   "path": "{projectName}/zones/{zone}/resourceViews/{resourceViewName}",
	//   "scopes": [
	//     "https://www.googleapis.com/auth/cloud-platform",
	//     "https://www.googleapis.com/auth/compute",
	//     "https://www.googleapis.com/auth/ndev.cloudman"
	//   ]
	// }

}

// method id "resourceviews.zoneViews.get":

type ZoneViewsGetCall struct {
	s                *Service
	projectName      string
	zone             string
	resourceViewName string
	opt_             map[string]interface{}
}

// Get: Get the information of a zonal resource view.
func (r *ZoneViewsService) Get(projectName string, zone string, resourceViewName string) *ZoneViewsGetCall {
	c := &ZoneViewsGetCall{s: r.s, opt_: make(map[string]interface{})}
	c.projectName = projectName
	c.zone = zone
	c.resourceViewName = resourceViewName
	return c
}

// Fields allows partial responses to be retrieved.
// See https://developers.google.com/gdata/docs/2.0/basics#PartialResponse
// for more information.
func (c *ZoneViewsGetCall) Fields(s ...googleapi.Field) *ZoneViewsGetCall {
	c.opt_["fields"] = googleapi.CombineFields(s)
	return c
}

func (c *ZoneViewsGetCall) Do() (*ResourceView, error) {
	var body io.Reader = nil
	params := make(url.Values)
	params.Set("alt", "json")
	if v, ok := c.opt_["fields"]; ok {
		params.Set("fields", fmt.Sprintf("%v", v))
	}
	urls := googleapi.ResolveRelative(c.s.BasePath, "{projectName}/zones/{zone}/resourceViews/{resourceViewName}")
	urls += "?" + params.Encode()
	req, _ := http.NewRequest("GET", urls, body)
	googleapi.Expand(req.URL, map[string]string{
		"projectName":      c.projectName,
		"zone":             c.zone,
		"resourceViewName": c.resourceViewName,
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
	var ret *ResourceView
	if err := json.NewDecoder(res.Body).Decode(&ret); err != nil {
		return nil, err
	}
	return ret, nil
	// {
	//   "description": "Get the information of a zonal resource view.",
	//   "httpMethod": "GET",
	//   "id": "resourceviews.zoneViews.get",
	//   "parameterOrder": [
	//     "projectName",
	//     "zone",
	//     "resourceViewName"
	//   ],
	//   "parameters": {
	//     "projectName": {
	//       "description": "The project name of the resource view.",
	//       "location": "path",
	//       "required": true,
	//       "type": "string"
	//     },
	//     "resourceViewName": {
	//       "description": "The name of the resource view.",
	//       "location": "path",
	//       "required": true,
	//       "type": "string"
	//     },
	//     "zone": {
	//       "description": "The zone name of the resource view.",
	//       "location": "path",
	//       "required": true,
	//       "type": "string"
	//     }
	//   },
	//   "path": "{projectName}/zones/{zone}/resourceViews/{resourceViewName}",
	//   "response": {
	//     "$ref": "ResourceView"
	//   },
	//   "scopes": [
	//     "https://www.googleapis.com/auth/cloud-platform",
	//     "https://www.googleapis.com/auth/compute",
	//     "https://www.googleapis.com/auth/compute.readonly",
	//     "https://www.googleapis.com/auth/ndev.cloudman",
	//     "https://www.googleapis.com/auth/ndev.cloudman.readonly"
	//   ]
	// }

}

// method id "resourceviews.zoneViews.insert":

type ZoneViewsInsertCall struct {
	s            *Service
	projectName  string
	zone         string
	resourceview *ResourceView
	opt_         map[string]interface{}
}

// Insert: Create a resource view.
func (r *ZoneViewsService) Insert(projectName string, zone string, resourceview *ResourceView) *ZoneViewsInsertCall {
	c := &ZoneViewsInsertCall{s: r.s, opt_: make(map[string]interface{})}
	c.projectName = projectName
	c.zone = zone
	c.resourceview = resourceview
	return c
}

// Fields allows partial responses to be retrieved.
// See https://developers.google.com/gdata/docs/2.0/basics#PartialResponse
// for more information.
func (c *ZoneViewsInsertCall) Fields(s ...googleapi.Field) *ZoneViewsInsertCall {
	c.opt_["fields"] = googleapi.CombineFields(s)
	return c
}

func (c *ZoneViewsInsertCall) Do() (*ZoneViewsInsertResponse, error) {
	var body io.Reader = nil
	body, err := googleapi.WithoutDataWrapper.JSONReader(c.resourceview)
	if err != nil {
		return nil, err
	}
	ctype := "application/json"
	params := make(url.Values)
	params.Set("alt", "json")
	if v, ok := c.opt_["fields"]; ok {
		params.Set("fields", fmt.Sprintf("%v", v))
	}
	urls := googleapi.ResolveRelative(c.s.BasePath, "{projectName}/zones/{zone}/resourceViews")
	urls += "?" + params.Encode()
	req, _ := http.NewRequest("POST", urls, body)
	googleapi.Expand(req.URL, map[string]string{
		"projectName": c.projectName,
		"zone":        c.zone,
	})
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
	var ret *ZoneViewsInsertResponse
	if err := json.NewDecoder(res.Body).Decode(&ret); err != nil {
		return nil, err
	}
	return ret, nil
	// {
	//   "description": "Create a resource view.",
	//   "httpMethod": "POST",
	//   "id": "resourceviews.zoneViews.insert",
	//   "parameterOrder": [
	//     "projectName",
	//     "zone"
	//   ],
	//   "parameters": {
	//     "projectName": {
	//       "description": "The project name of the resource view.",
	//       "location": "path",
	//       "required": true,
	//       "type": "string"
	//     },
	//     "zone": {
	//       "description": "The zone name of the resource view.",
	//       "location": "path",
	//       "required": true,
	//       "type": "string"
	//     }
	//   },
	//   "path": "{projectName}/zones/{zone}/resourceViews",
	//   "request": {
	//     "$ref": "ResourceView"
	//   },
	//   "response": {
	//     "$ref": "ZoneViewsInsertResponse"
	//   },
	//   "scopes": [
	//     "https://www.googleapis.com/auth/cloud-platform",
	//     "https://www.googleapis.com/auth/compute",
	//     "https://www.googleapis.com/auth/ndev.cloudman"
	//   ]
	// }

}

// method id "resourceviews.zoneViews.list":

type ZoneViewsListCall struct {
	s           *Service
	projectName string
	zone        string
	opt_        map[string]interface{}
}

// List: List resource views.
func (r *ZoneViewsService) List(projectName string, zone string) *ZoneViewsListCall {
	c := &ZoneViewsListCall{s: r.s, opt_: make(map[string]interface{})}
	c.projectName = projectName
	c.zone = zone
	return c
}

// MaxResults sets the optional parameter "maxResults": Maximum count of
// results to be returned. Acceptable values are 0 to 5000, inclusive.
// (Default: 5000)
func (c *ZoneViewsListCall) MaxResults(maxResults int64) *ZoneViewsListCall {
	c.opt_["maxResults"] = maxResults
	return c
}

// PageToken sets the optional parameter "pageToken": Specifies a
// nextPageToken returned by a previous list request. This token can be
// used to request the next page of results from a previous list
// request.
func (c *ZoneViewsListCall) PageToken(pageToken string) *ZoneViewsListCall {
	c.opt_["pageToken"] = pageToken
	return c
}

// Fields allows partial responses to be retrieved.
// See https://developers.google.com/gdata/docs/2.0/basics#PartialResponse
// for more information.
func (c *ZoneViewsListCall) Fields(s ...googleapi.Field) *ZoneViewsListCall {
	c.opt_["fields"] = googleapi.CombineFields(s)
	return c
}

func (c *ZoneViewsListCall) Do() (*ZoneViewsListResponse, error) {
	var body io.Reader = nil
	params := make(url.Values)
	params.Set("alt", "json")
	if v, ok := c.opt_["maxResults"]; ok {
		params.Set("maxResults", fmt.Sprintf("%v", v))
	}
	if v, ok := c.opt_["pageToken"]; ok {
		params.Set("pageToken", fmt.Sprintf("%v", v))
	}
	if v, ok := c.opt_["fields"]; ok {
		params.Set("fields", fmt.Sprintf("%v", v))
	}
	urls := googleapi.ResolveRelative(c.s.BasePath, "{projectName}/zones/{zone}/resourceViews")
	urls += "?" + params.Encode()
	req, _ := http.NewRequest("GET", urls, body)
	googleapi.Expand(req.URL, map[string]string{
		"projectName": c.projectName,
		"zone":        c.zone,
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
	var ret *ZoneViewsListResponse
	if err := json.NewDecoder(res.Body).Decode(&ret); err != nil {
		return nil, err
	}
	return ret, nil
	// {
	//   "description": "List resource views.",
	//   "httpMethod": "GET",
	//   "id": "resourceviews.zoneViews.list",
	//   "parameterOrder": [
	//     "projectName",
	//     "zone"
	//   ],
	//   "parameters": {
	//     "maxResults": {
	//       "default": "5000",
	//       "description": "Maximum count of results to be returned. Acceptable values are 0 to 5000, inclusive. (Default: 5000)",
	//       "format": "int32",
	//       "location": "query",
	//       "maximum": "5000",
	//       "minimum": "0",
	//       "type": "integer"
	//     },
	//     "pageToken": {
	//       "description": "Specifies a nextPageToken returned by a previous list request. This token can be used to request the next page of results from a previous list request.",
	//       "location": "query",
	//       "type": "string"
	//     },
	//     "projectName": {
	//       "description": "The project name of the resource view.",
	//       "location": "path",
	//       "required": true,
	//       "type": "string"
	//     },
	//     "zone": {
	//       "description": "The zone name of the resource view.",
	//       "location": "path",
	//       "required": true,
	//       "type": "string"
	//     }
	//   },
	//   "path": "{projectName}/zones/{zone}/resourceViews",
	//   "response": {
	//     "$ref": "ZoneViewsListResponse"
	//   },
	//   "scopes": [
	//     "https://www.googleapis.com/auth/cloud-platform",
	//     "https://www.googleapis.com/auth/compute",
	//     "https://www.googleapis.com/auth/compute.readonly",
	//     "https://www.googleapis.com/auth/ndev.cloudman",
	//     "https://www.googleapis.com/auth/ndev.cloudman.readonly"
	//   ]
	// }

}

// method id "resourceviews.zoneViews.listresources":

type ZoneViewsListresourcesCall struct {
	s                *Service
	projectName      string
	zone             string
	resourceViewName string
	opt_             map[string]interface{}
}

// Listresources: List the resources of the resource view.
func (r *ZoneViewsService) Listresources(projectName string, zone string, resourceViewName string) *ZoneViewsListresourcesCall {
	c := &ZoneViewsListresourcesCall{s: r.s, opt_: make(map[string]interface{})}
	c.projectName = projectName
	c.zone = zone
	c.resourceViewName = resourceViewName
	return c
}

// MaxResults sets the optional parameter "maxResults": Maximum count of
// results to be returned. Acceptable values are 0 to 5000, inclusive.
// (Default: 5000)
func (c *ZoneViewsListresourcesCall) MaxResults(maxResults int64) *ZoneViewsListresourcesCall {
	c.opt_["maxResults"] = maxResults
	return c
}

// PageToken sets the optional parameter "pageToken": Specifies a
// nextPageToken returned by a previous list request. This token can be
// used to request the next page of results from a previous list
// request.
func (c *ZoneViewsListresourcesCall) PageToken(pageToken string) *ZoneViewsListresourcesCall {
	c.opt_["pageToken"] = pageToken
	return c
}

// Fields allows partial responses to be retrieved.
// See https://developers.google.com/gdata/docs/2.0/basics#PartialResponse
// for more information.
func (c *ZoneViewsListresourcesCall) Fields(s ...googleapi.Field) *ZoneViewsListresourcesCall {
	c.opt_["fields"] = googleapi.CombineFields(s)
	return c
}

func (c *ZoneViewsListresourcesCall) Do() (*ZoneViewsListResourcesResponse, error) {
	var body io.Reader = nil
	params := make(url.Values)
	params.Set("alt", "json")
	if v, ok := c.opt_["maxResults"]; ok {
		params.Set("maxResults", fmt.Sprintf("%v", v))
	}
	if v, ok := c.opt_["pageToken"]; ok {
		params.Set("pageToken", fmt.Sprintf("%v", v))
	}
	if v, ok := c.opt_["fields"]; ok {
		params.Set("fields", fmt.Sprintf("%v", v))
	}
	urls := googleapi.ResolveRelative(c.s.BasePath, "{projectName}/zones/{zone}/resourceViews/{resourceViewName}/resources")
	urls += "?" + params.Encode()
	req, _ := http.NewRequest("POST", urls, body)
	googleapi.Expand(req.URL, map[string]string{
		"projectName":      c.projectName,
		"zone":             c.zone,
		"resourceViewName": c.resourceViewName,
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
	var ret *ZoneViewsListResourcesResponse
	if err := json.NewDecoder(res.Body).Decode(&ret); err != nil {
		return nil, err
	}
	return ret, nil
	// {
	//   "description": "List the resources of the resource view.",
	//   "httpMethod": "POST",
	//   "id": "resourceviews.zoneViews.listresources",
	//   "parameterOrder": [
	//     "projectName",
	//     "zone",
	//     "resourceViewName"
	//   ],
	//   "parameters": {
	//     "maxResults": {
	//       "default": "5000",
	//       "description": "Maximum count of results to be returned. Acceptable values are 0 to 5000, inclusive. (Default: 5000)",
	//       "format": "int32",
	//       "location": "query",
	//       "maximum": "5000",
	//       "minimum": "0",
	//       "type": "integer"
	//     },
	//     "pageToken": {
	//       "description": "Specifies a nextPageToken returned by a previous list request. This token can be used to request the next page of results from a previous list request.",
	//       "location": "query",
	//       "type": "string"
	//     },
	//     "projectName": {
	//       "description": "The project name of the resource view.",
	//       "location": "path",
	//       "required": true,
	//       "type": "string"
	//     },
	//     "resourceViewName": {
	//       "description": "The name of the resource view.",
	//       "location": "path",
	//       "required": true,
	//       "type": "string"
	//     },
	//     "zone": {
	//       "description": "The zone name of the resource view.",
	//       "location": "path",
	//       "required": true,
	//       "type": "string"
	//     }
	//   },
	//   "path": "{projectName}/zones/{zone}/resourceViews/{resourceViewName}/resources",
	//   "response": {
	//     "$ref": "ZoneViewsListResourcesResponse"
	//   },
	//   "scopes": [
	//     "https://www.googleapis.com/auth/cloud-platform",
	//     "https://www.googleapis.com/auth/compute",
	//     "https://www.googleapis.com/auth/compute.readonly",
	//     "https://www.googleapis.com/auth/ndev.cloudman",
	//     "https://www.googleapis.com/auth/ndev.cloudman.readonly"
	//   ]
	// }

}

// method id "resourceviews.zoneViews.removeresources":

type ZoneViewsRemoveresourcesCall struct {
	s                               *Service
	projectName                     string
	zone                            string
	resourceViewName                string
	zoneviewsremoveresourcesrequest *ZoneViewsRemoveResourcesRequest
	opt_                            map[string]interface{}
}

// Removeresources: Remove resources from the view.
func (r *ZoneViewsService) Removeresources(projectName string, zone string, resourceViewName string, zoneviewsremoveresourcesrequest *ZoneViewsRemoveResourcesRequest) *ZoneViewsRemoveresourcesCall {
	c := &ZoneViewsRemoveresourcesCall{s: r.s, opt_: make(map[string]interface{})}
	c.projectName = projectName
	c.zone = zone
	c.resourceViewName = resourceViewName
	c.zoneviewsremoveresourcesrequest = zoneviewsremoveresourcesrequest
	return c
}

// Fields allows partial responses to be retrieved.
// See https://developers.google.com/gdata/docs/2.0/basics#PartialResponse
// for more information.
func (c *ZoneViewsRemoveresourcesCall) Fields(s ...googleapi.Field) *ZoneViewsRemoveresourcesCall {
	c.opt_["fields"] = googleapi.CombineFields(s)
	return c
}

func (c *ZoneViewsRemoveresourcesCall) Do() error {
	var body io.Reader = nil
	body, err := googleapi.WithoutDataWrapper.JSONReader(c.zoneviewsremoveresourcesrequest)
	if err != nil {
		return err
	}
	ctype := "application/json"
	params := make(url.Values)
	params.Set("alt", "json")
	if v, ok := c.opt_["fields"]; ok {
		params.Set("fields", fmt.Sprintf("%v", v))
	}
	urls := googleapi.ResolveRelative(c.s.BasePath, "{projectName}/zones/{zone}/resourceViews/{resourceViewName}/removeResources")
	urls += "?" + params.Encode()
	req, _ := http.NewRequest("POST", urls, body)
	googleapi.Expand(req.URL, map[string]string{
		"projectName":      c.projectName,
		"zone":             c.zone,
		"resourceViewName": c.resourceViewName,
	})
	req.Header.Set("Content-Type", ctype)
	req.Header.Set("User-Agent", "google-api-go-client/0.5")
	res, err := c.s.client.Do(req)
	if err != nil {
		return err
	}
	defer googleapi.CloseBody(res)
	if err := googleapi.CheckResponse(res); err != nil {
		return err
	}
	return nil
	// {
	//   "description": "Remove resources from the view.",
	//   "httpMethod": "POST",
	//   "id": "resourceviews.zoneViews.removeresources",
	//   "parameterOrder": [
	//     "projectName",
	//     "zone",
	//     "resourceViewName"
	//   ],
	//   "parameters": {
	//     "projectName": {
	//       "description": "The project name of the resource view.",
	//       "location": "path",
	//       "required": true,
	//       "type": "string"
	//     },
	//     "resourceViewName": {
	//       "description": "The name of the resource view.",
	//       "location": "path",
	//       "required": true,
	//       "type": "string"
	//     },
	//     "zone": {
	//       "description": "The zone name of the resource view.",
	//       "location": "path",
	//       "required": true,
	//       "type": "string"
	//     }
	//   },
	//   "path": "{projectName}/zones/{zone}/resourceViews/{resourceViewName}/removeResources",
	//   "request": {
	//     "$ref": "ZoneViewsRemoveResourcesRequest"
	//   },
	//   "scopes": [
	//     "https://www.googleapis.com/auth/cloud-platform",
	//     "https://www.googleapis.com/auth/compute",
	//     "https://www.googleapis.com/auth/ndev.cloudman"
	//   ]
	// }

}
