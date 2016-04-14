// Package replicapoolupdater provides access to the Google Compute Engine Instance Group Updater API.
//
// See https://developers.google.com/compute/docs/instance-groups/manager/v1beta2
//
// Usage example:
//
//   import "google.golang.org/api/replicapoolupdater/v1beta1"
//   ...
//   replicapoolupdaterService, err := replicapoolupdater.New(oauthHttpClient)
package replicapoolupdater

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

const apiId = "replicapoolupdater:v1beta1"
const apiName = "replicapoolupdater"
const apiVersion = "v1beta1"
const basePath = "https://www.googleapis.com/replicapoolupdater/v1beta1/projects/"

// OAuth2 scopes used by this API.
const (
	// View and manage your data across Google Cloud Platform services
	CloudPlatformScope = "https://www.googleapis.com/auth/cloud-platform"

	// View and manage replica pools
	ReplicapoolScope = "https://www.googleapis.com/auth/replicapool"

	// View replica pools
	ReplicapoolReadonlyScope = "https://www.googleapis.com/auth/replicapool.readonly"
)

func New(client *http.Client) (*Service, error) {
	if client == nil {
		return nil, errors.New("client is nil")
	}
	s := &Service{client: client, BasePath: basePath}
	s.Updates = NewUpdatesService(s)
	return s, nil
}

type Service struct {
	client   *http.Client
	BasePath string // API endpoint base URL

	Updates *UpdatesService
}

func NewUpdatesService(s *Service) *UpdatesService {
	rs := &UpdatesService{s: s}
	return rs
}

type UpdatesService struct {
	s *Service
}

type InsertResponse struct {
	// Update: Unique (in the context of a group) handle of an update.
	Update string `json:"update,omitempty"`
}

type InstanceUpdate struct {
	// InstanceName: Name of an instance.
	InstanceName string `json:"instanceName,omitempty"`

	// State: State of an instance update.
	State string `json:"state,omitempty"`
}

type Update struct {
	// Details: [Output Only] Human-readable description of an update
	// progress.
	Details string `json:"details,omitempty"`

	// Handle: [Output Only] Unique (in the context of a group) handle
	// assigned to this update.
	Handle string `json:"handle,omitempty"`

	// InstanceTemplate: Url of an instance template to be applied.
	InstanceTemplate string `json:"instanceTemplate,omitempty"`

	// InstanceUpdates: [Output Only] Collection of instance updates.
	InstanceUpdates []*InstanceUpdate `json:"instanceUpdates,omitempty"`

	// Kind: [Output only] The resource type. Always
	// replicapoolupdater#update.
	Kind string `json:"kind,omitempty"`

	// Policy: Parameters of an update process.
	Policy *UpdatePolicy `json:"policy,omitempty"`

	// SelfLink: [Output only] The fully qualified URL for this resource.
	SelfLink string `json:"selfLink,omitempty"`

	// State: [Output Only] Current state of an update.
	State string `json:"state,omitempty"`

	// TargetState: [Output Only] Requested state of an update. This is the
	// state that the updater is moving towards. Acceptable values are:
	// -
	// "ROLLED_OUT": The user has requested the update to go forward.
	// -
	// "ROLLED_BACK": The user has requested the update to be rolled back.
	//
	// - "PAUSED": The user has requested the update to be paused.
	//
	// -
	// "CANCELLED": The user has requested the update to be cancelled. The
	// updater service is in the process of canceling the update.
	TargetState string `json:"targetState,omitempty"`
}

type UpdateList struct {
	// Items: Collection of requested updates.
	Items []*Update `json:"items,omitempty"`

	// NextPageToken: A token used to continue a truncated list request.
	NextPageToken string `json:"nextPageToken,omitempty"`
}

type UpdatePolicy struct {
	// Canary: Parameters of a canary phase. If absent, canary will NOT be
	// performed.
	Canary *UpdatePolicyCanary `json:"canary,omitempty"`

	// MaxNumConcurrentInstances: Maximum number of instances that can be
	// updated simultaneously (concurrently). An update of an instance
	// starts when the instance is about to be restarted and finishes after
	// the instance has been restarted and the sleep period (defined by
	// sleep_after_instance_restart_sec) has passed.
	MaxNumConcurrentInstances int64 `json:"maxNumConcurrentInstances,omitempty"`

	// SleepAfterInstanceRestartSec: Time period after the instance has been
	// restarted but before marking the update of this instance as done.
	SleepAfterInstanceRestartSec int64 `json:"sleepAfterInstanceRestartSec,omitempty"`
}

type UpdatePolicyCanary struct {
	// NumInstances: Number of instances updated as a part of canary phase.
	// If absent, the default number of instances will be used.
	NumInstances int64 `json:"numInstances,omitempty"`
}

// method id "replicapoolupdater.updates.cancel":

type UpdatesCancelCall struct {
	s                    *Service
	project              string
	zone                 string
	instanceGroupManager string
	update               string
	opt_                 map[string]interface{}
}

// Cancel: Called on the particular Update endpoint. Cancels the update
// in state PAUSED. No-op if invoked in state CANCELLED.
func (r *UpdatesService) Cancel(project string, zone string, instanceGroupManager string, update string) *UpdatesCancelCall {
	c := &UpdatesCancelCall{s: r.s, opt_: make(map[string]interface{})}
	c.project = project
	c.zone = zone
	c.instanceGroupManager = instanceGroupManager
	c.update = update
	return c
}

// Fields allows partial responses to be retrieved.
// See https://developers.google.com/gdata/docs/2.0/basics#PartialResponse
// for more information.
func (c *UpdatesCancelCall) Fields(s ...googleapi.Field) *UpdatesCancelCall {
	c.opt_["fields"] = googleapi.CombineFields(s)
	return c
}

func (c *UpdatesCancelCall) Do() error {
	var body io.Reader = nil
	params := make(url.Values)
	params.Set("alt", "json")
	if v, ok := c.opt_["fields"]; ok {
		params.Set("fields", fmt.Sprintf("%v", v))
	}
	urls := googleapi.ResolveRelative(c.s.BasePath, "{project}/zones/{zone}/instanceGroupManagers/{instanceGroupManager}/updates/{update}/cancel")
	urls += "?" + params.Encode()
	req, _ := http.NewRequest("POST", urls, body)
	googleapi.Expand(req.URL, map[string]string{
		"project":              c.project,
		"zone":                 c.zone,
		"instanceGroupManager": c.instanceGroupManager,
		"update":               c.update,
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
	//   "description": "Called on the particular Update endpoint. Cancels the update in state PAUSED. No-op if invoked in state CANCELLED.",
	//   "httpMethod": "POST",
	//   "id": "replicapoolupdater.updates.cancel",
	//   "parameterOrder": [
	//     "project",
	//     "zone",
	//     "instanceGroupManager",
	//     "update"
	//   ],
	//   "parameters": {
	//     "instanceGroupManager": {
	//       "description": "Name of the instance group manager for this request.",
	//       "location": "path",
	//       "required": true,
	//       "type": "string"
	//     },
	//     "project": {
	//       "description": "Project ID for this request.",
	//       "location": "path",
	//       "required": true,
	//       "type": "string"
	//     },
	//     "update": {
	//       "description": "Unique (in the context of a group) handle of an update.",
	//       "location": "path",
	//       "required": true,
	//       "type": "string"
	//     },
	//     "zone": {
	//       "description": "Zone for the instance group manager.",
	//       "location": "path",
	//       "required": true,
	//       "type": "string"
	//     }
	//   },
	//   "path": "{project}/zones/{zone}/instanceGroupManagers/{instanceGroupManager}/updates/{update}/cancel",
	//   "scopes": [
	//     "https://www.googleapis.com/auth/cloud-platform",
	//     "https://www.googleapis.com/auth/replicapool"
	//   ]
	// }

}

// method id "replicapoolupdater.updates.get":

type UpdatesGetCall struct {
	s                    *Service
	project              string
	zone                 string
	instanceGroupManager string
	update               string
	opt_                 map[string]interface{}
}

// Get: Called on the particular Update endpoint. Returns the Update
// resource.
func (r *UpdatesService) Get(project string, zone string, instanceGroupManager string, update string) *UpdatesGetCall {
	c := &UpdatesGetCall{s: r.s, opt_: make(map[string]interface{})}
	c.project = project
	c.zone = zone
	c.instanceGroupManager = instanceGroupManager
	c.update = update
	return c
}

// Fields allows partial responses to be retrieved.
// See https://developers.google.com/gdata/docs/2.0/basics#PartialResponse
// for more information.
func (c *UpdatesGetCall) Fields(s ...googleapi.Field) *UpdatesGetCall {
	c.opt_["fields"] = googleapi.CombineFields(s)
	return c
}

func (c *UpdatesGetCall) Do() (*Update, error) {
	var body io.Reader = nil
	params := make(url.Values)
	params.Set("alt", "json")
	if v, ok := c.opt_["fields"]; ok {
		params.Set("fields", fmt.Sprintf("%v", v))
	}
	urls := googleapi.ResolveRelative(c.s.BasePath, "{project}/zones/{zone}/instanceGroupManagers/{instanceGroupManager}/updates/{update}")
	urls += "?" + params.Encode()
	req, _ := http.NewRequest("GET", urls, body)
	googleapi.Expand(req.URL, map[string]string{
		"project":              c.project,
		"zone":                 c.zone,
		"instanceGroupManager": c.instanceGroupManager,
		"update":               c.update,
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
	var ret *Update
	if err := json.NewDecoder(res.Body).Decode(&ret); err != nil {
		return nil, err
	}
	return ret, nil
	// {
	//   "description": "Called on the particular Update endpoint. Returns the Update resource.",
	//   "httpMethod": "GET",
	//   "id": "replicapoolupdater.updates.get",
	//   "parameterOrder": [
	//     "project",
	//     "zone",
	//     "instanceGroupManager",
	//     "update"
	//   ],
	//   "parameters": {
	//     "instanceGroupManager": {
	//       "description": "Name of the instance group manager for this request.",
	//       "location": "path",
	//       "required": true,
	//       "type": "string"
	//     },
	//     "project": {
	//       "description": "Project ID for this request.",
	//       "location": "path",
	//       "required": true,
	//       "type": "string"
	//     },
	//     "update": {
	//       "description": "Unique (in the context of a group) handle of an update.",
	//       "location": "path",
	//       "required": true,
	//       "type": "string"
	//     },
	//     "zone": {
	//       "description": "Zone for the instance group manager.",
	//       "location": "path",
	//       "required": true,
	//       "type": "string"
	//     }
	//   },
	//   "path": "{project}/zones/{zone}/instanceGroupManagers/{instanceGroupManager}/updates/{update}",
	//   "response": {
	//     "$ref": "Update"
	//   },
	//   "scopes": [
	//     "https://www.googleapis.com/auth/cloud-platform",
	//     "https://www.googleapis.com/auth/replicapool",
	//     "https://www.googleapis.com/auth/replicapool.readonly"
	//   ]
	// }

}

// method id "replicapoolupdater.updates.insert":

type UpdatesInsertCall struct {
	s                    *Service
	project              string
	zone                 string
	instanceGroupManager string
	update               *Update
	opt_                 map[string]interface{}
}

// Insert: Called on the collection endpoint. Inserts the new Update
// resource and starts the update.
func (r *UpdatesService) Insert(project string, zone string, instanceGroupManager string, update *Update) *UpdatesInsertCall {
	c := &UpdatesInsertCall{s: r.s, opt_: make(map[string]interface{})}
	c.project = project
	c.zone = zone
	c.instanceGroupManager = instanceGroupManager
	c.update = update
	return c
}

// Fields allows partial responses to be retrieved.
// See https://developers.google.com/gdata/docs/2.0/basics#PartialResponse
// for more information.
func (c *UpdatesInsertCall) Fields(s ...googleapi.Field) *UpdatesInsertCall {
	c.opt_["fields"] = googleapi.CombineFields(s)
	return c
}

func (c *UpdatesInsertCall) Do() (*InsertResponse, error) {
	var body io.Reader = nil
	body, err := googleapi.WithoutDataWrapper.JSONReader(c.update)
	if err != nil {
		return nil, err
	}
	ctype := "application/json"
	params := make(url.Values)
	params.Set("alt", "json")
	if v, ok := c.opt_["fields"]; ok {
		params.Set("fields", fmt.Sprintf("%v", v))
	}
	urls := googleapi.ResolveRelative(c.s.BasePath, "{project}/zones/{zone}/instanceGroupManagers/{instanceGroupManager}/updates")
	urls += "?" + params.Encode()
	req, _ := http.NewRequest("POST", urls, body)
	googleapi.Expand(req.URL, map[string]string{
		"project":              c.project,
		"zone":                 c.zone,
		"instanceGroupManager": c.instanceGroupManager,
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
	var ret *InsertResponse
	if err := json.NewDecoder(res.Body).Decode(&ret); err != nil {
		return nil, err
	}
	return ret, nil
	// {
	//   "description": "Called on the collection endpoint. Inserts the new Update resource and starts the update.",
	//   "httpMethod": "POST",
	//   "id": "replicapoolupdater.updates.insert",
	//   "parameterOrder": [
	//     "project",
	//     "zone",
	//     "instanceGroupManager"
	//   ],
	//   "parameters": {
	//     "instanceGroupManager": {
	//       "description": "Name of the instance group manager for this request.",
	//       "location": "path",
	//       "required": true,
	//       "type": "string"
	//     },
	//     "project": {
	//       "description": "Project ID for this request.",
	//       "location": "path",
	//       "required": true,
	//       "type": "string"
	//     },
	//     "zone": {
	//       "description": "Zone for the instance group manager.",
	//       "location": "path",
	//       "required": true,
	//       "type": "string"
	//     }
	//   },
	//   "path": "{project}/zones/{zone}/instanceGroupManagers/{instanceGroupManager}/updates",
	//   "request": {
	//     "$ref": "Update"
	//   },
	//   "response": {
	//     "$ref": "InsertResponse"
	//   },
	//   "scopes": [
	//     "https://www.googleapis.com/auth/cloud-platform",
	//     "https://www.googleapis.com/auth/replicapool"
	//   ]
	// }

}

// method id "replicapoolupdater.updates.list":

type UpdatesListCall struct {
	s                    *Service
	project              string
	zone                 string
	instanceGroupManager string
	opt_                 map[string]interface{}
}

// List: Called on the collection endpoint. Lists updates for a given
// instance group, in reverse chronological order. Pagination is
// supported, see ListRequestHeader.
func (r *UpdatesService) List(project string, zone string, instanceGroupManager string) *UpdatesListCall {
	c := &UpdatesListCall{s: r.s, opt_: make(map[string]interface{})}
	c.project = project
	c.zone = zone
	c.instanceGroupManager = instanceGroupManager
	return c
}

// MaxResults sets the optional parameter "maxResults": Maximum count of
// results to be returned. Acceptable values are 1 to 100, inclusive.
// (Default: 50)
func (c *UpdatesListCall) MaxResults(maxResults int64) *UpdatesListCall {
	c.opt_["maxResults"] = maxResults
	return c
}

// PageToken sets the optional parameter "pageToken": Set this to the
// nextPageToken value returned by a previous list request to obtain the
// next page of results from the previous list request.
func (c *UpdatesListCall) PageToken(pageToken string) *UpdatesListCall {
	c.opt_["pageToken"] = pageToken
	return c
}

// Fields allows partial responses to be retrieved.
// See https://developers.google.com/gdata/docs/2.0/basics#PartialResponse
// for more information.
func (c *UpdatesListCall) Fields(s ...googleapi.Field) *UpdatesListCall {
	c.opt_["fields"] = googleapi.CombineFields(s)
	return c
}

func (c *UpdatesListCall) Do() (*UpdateList, error) {
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
	urls := googleapi.ResolveRelative(c.s.BasePath, "{project}/zones/{zone}/instanceGroupManagers/{instanceGroupManager}/updates")
	urls += "?" + params.Encode()
	req, _ := http.NewRequest("GET", urls, body)
	googleapi.Expand(req.URL, map[string]string{
		"project":              c.project,
		"zone":                 c.zone,
		"instanceGroupManager": c.instanceGroupManager,
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
	var ret *UpdateList
	if err := json.NewDecoder(res.Body).Decode(&ret); err != nil {
		return nil, err
	}
	return ret, nil
	// {
	//   "description": "Called on the collection endpoint. Lists updates for a given instance group, in reverse chronological order. Pagination is supported, see ListRequestHeader.",
	//   "httpMethod": "GET",
	//   "id": "replicapoolupdater.updates.list",
	//   "parameterOrder": [
	//     "project",
	//     "zone",
	//     "instanceGroupManager"
	//   ],
	//   "parameters": {
	//     "instanceGroupManager": {
	//       "description": "Name of the instance group manager for this request.",
	//       "location": "path",
	//       "required": true,
	//       "type": "string"
	//     },
	//     "maxResults": {
	//       "default": "50",
	//       "description": "Maximum count of results to be returned. Acceptable values are 1 to 100, inclusive. (Default: 50)",
	//       "format": "int32",
	//       "location": "query",
	//       "maximum": "100",
	//       "minimum": "1",
	//       "type": "integer"
	//     },
	//     "pageToken": {
	//       "description": "Set this to the nextPageToken value returned by a previous list request to obtain the next page of results from the previous list request.",
	//       "location": "query",
	//       "type": "string"
	//     },
	//     "project": {
	//       "description": "Project ID for this request.",
	//       "location": "path",
	//       "required": true,
	//       "type": "string"
	//     },
	//     "zone": {
	//       "description": "Zone for the instance group manager.",
	//       "location": "path",
	//       "required": true,
	//       "type": "string"
	//     }
	//   },
	//   "path": "{project}/zones/{zone}/instanceGroupManagers/{instanceGroupManager}/updates",
	//   "response": {
	//     "$ref": "UpdateList"
	//   },
	//   "scopes": [
	//     "https://www.googleapis.com/auth/cloud-platform",
	//     "https://www.googleapis.com/auth/replicapool",
	//     "https://www.googleapis.com/auth/replicapool.readonly"
	//   ]
	// }

}

// method id "replicapoolupdater.updates.pause":

type UpdatesPauseCall struct {
	s                    *Service
	project              string
	zone                 string
	instanceGroupManager string
	update               string
	opt_                 map[string]interface{}
}

// Pause: Called on the particular Update endpoint. Pauses the update in
// state from { ROLLING_FORWARD, ROLLING_BACK, PAUSED }. No-op if
// invoked in state PAUSED.
func (r *UpdatesService) Pause(project string, zone string, instanceGroupManager string, update string) *UpdatesPauseCall {
	c := &UpdatesPauseCall{s: r.s, opt_: make(map[string]interface{})}
	c.project = project
	c.zone = zone
	c.instanceGroupManager = instanceGroupManager
	c.update = update
	return c
}

// Fields allows partial responses to be retrieved.
// See https://developers.google.com/gdata/docs/2.0/basics#PartialResponse
// for more information.
func (c *UpdatesPauseCall) Fields(s ...googleapi.Field) *UpdatesPauseCall {
	c.opt_["fields"] = googleapi.CombineFields(s)
	return c
}

func (c *UpdatesPauseCall) Do() error {
	var body io.Reader = nil
	params := make(url.Values)
	params.Set("alt", "json")
	if v, ok := c.opt_["fields"]; ok {
		params.Set("fields", fmt.Sprintf("%v", v))
	}
	urls := googleapi.ResolveRelative(c.s.BasePath, "{project}/zones/{zone}/instanceGroupManagers/{instanceGroupManager}/updates/{update}/pause")
	urls += "?" + params.Encode()
	req, _ := http.NewRequest("POST", urls, body)
	googleapi.Expand(req.URL, map[string]string{
		"project":              c.project,
		"zone":                 c.zone,
		"instanceGroupManager": c.instanceGroupManager,
		"update":               c.update,
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
	//   "description": "Called on the particular Update endpoint. Pauses the update in state from { ROLLING_FORWARD, ROLLING_BACK, PAUSED }. No-op if invoked in state PAUSED.",
	//   "httpMethod": "POST",
	//   "id": "replicapoolupdater.updates.pause",
	//   "parameterOrder": [
	//     "project",
	//     "zone",
	//     "instanceGroupManager",
	//     "update"
	//   ],
	//   "parameters": {
	//     "instanceGroupManager": {
	//       "description": "Name of the instance group manager for this request.",
	//       "location": "path",
	//       "required": true,
	//       "type": "string"
	//     },
	//     "project": {
	//       "description": "Project ID for this request.",
	//       "location": "path",
	//       "required": true,
	//       "type": "string"
	//     },
	//     "update": {
	//       "description": "Unique (in the context of a group) handle of an update.",
	//       "location": "path",
	//       "required": true,
	//       "type": "string"
	//     },
	//     "zone": {
	//       "description": "Zone for the instance group manager.",
	//       "location": "path",
	//       "required": true,
	//       "type": "string"
	//     }
	//   },
	//   "path": "{project}/zones/{zone}/instanceGroupManagers/{instanceGroupManager}/updates/{update}/pause",
	//   "scopes": [
	//     "https://www.googleapis.com/auth/cloud-platform",
	//     "https://www.googleapis.com/auth/replicapool"
	//   ]
	// }

}

// method id "replicapoolupdater.updates.rollback":

type UpdatesRollbackCall struct {
	s                    *Service
	project              string
	zone                 string
	instanceGroupManager string
	update               string
	opt_                 map[string]interface{}
}

// Rollback: Called on the particular Update endpoint. Rolls back the
// update in state from { ROLLING_FORWARD, ROLLING_BACK, PAUSED }. No-op
// if invoked in state ROLLED_BACK.
func (r *UpdatesService) Rollback(project string, zone string, instanceGroupManager string, update string) *UpdatesRollbackCall {
	c := &UpdatesRollbackCall{s: r.s, opt_: make(map[string]interface{})}
	c.project = project
	c.zone = zone
	c.instanceGroupManager = instanceGroupManager
	c.update = update
	return c
}

// Fields allows partial responses to be retrieved.
// See https://developers.google.com/gdata/docs/2.0/basics#PartialResponse
// for more information.
func (c *UpdatesRollbackCall) Fields(s ...googleapi.Field) *UpdatesRollbackCall {
	c.opt_["fields"] = googleapi.CombineFields(s)
	return c
}

func (c *UpdatesRollbackCall) Do() error {
	var body io.Reader = nil
	params := make(url.Values)
	params.Set("alt", "json")
	if v, ok := c.opt_["fields"]; ok {
		params.Set("fields", fmt.Sprintf("%v", v))
	}
	urls := googleapi.ResolveRelative(c.s.BasePath, "{project}/zones/{zone}/instanceGroupManagers/{instanceGroupManager}/updates/{update}/rollback")
	urls += "?" + params.Encode()
	req, _ := http.NewRequest("POST", urls, body)
	googleapi.Expand(req.URL, map[string]string{
		"project":              c.project,
		"zone":                 c.zone,
		"instanceGroupManager": c.instanceGroupManager,
		"update":               c.update,
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
	//   "description": "Called on the particular Update endpoint. Rolls back the update in state from { ROLLING_FORWARD, ROLLING_BACK, PAUSED }. No-op if invoked in state ROLLED_BACK.",
	//   "httpMethod": "POST",
	//   "id": "replicapoolupdater.updates.rollback",
	//   "parameterOrder": [
	//     "project",
	//     "zone",
	//     "instanceGroupManager",
	//     "update"
	//   ],
	//   "parameters": {
	//     "instanceGroupManager": {
	//       "description": "Name of the instance group manager for this request.",
	//       "location": "path",
	//       "required": true,
	//       "type": "string"
	//     },
	//     "project": {
	//       "description": "Project ID for this request.",
	//       "location": "path",
	//       "required": true,
	//       "type": "string"
	//     },
	//     "update": {
	//       "description": "Unique (in the context of a group) handle of an update.",
	//       "location": "path",
	//       "required": true,
	//       "type": "string"
	//     },
	//     "zone": {
	//       "description": "Zone for the instance group manager.",
	//       "location": "path",
	//       "required": true,
	//       "type": "string"
	//     }
	//   },
	//   "path": "{project}/zones/{zone}/instanceGroupManagers/{instanceGroupManager}/updates/{update}/rollback",
	//   "scopes": [
	//     "https://www.googleapis.com/auth/cloud-platform",
	//     "https://www.googleapis.com/auth/replicapool"
	//   ]
	// }

}

// method id "replicapoolupdater.updates.rollforward":

type UpdatesRollforwardCall struct {
	s                    *Service
	project              string
	zone                 string
	instanceGroupManager string
	update               string
	opt_                 map[string]interface{}
}

// Rollforward: Called on the particular Update endpoint. Rolls forward
// the update in state from { ROLLING_FORWARD, ROLLING_BACK, PAUSED }.
// No-op if invoked in state ROLLED_OUT.
func (r *UpdatesService) Rollforward(project string, zone string, instanceGroupManager string, update string) *UpdatesRollforwardCall {
	c := &UpdatesRollforwardCall{s: r.s, opt_: make(map[string]interface{})}
	c.project = project
	c.zone = zone
	c.instanceGroupManager = instanceGroupManager
	c.update = update
	return c
}

// Fields allows partial responses to be retrieved.
// See https://developers.google.com/gdata/docs/2.0/basics#PartialResponse
// for more information.
func (c *UpdatesRollforwardCall) Fields(s ...googleapi.Field) *UpdatesRollforwardCall {
	c.opt_["fields"] = googleapi.CombineFields(s)
	return c
}

func (c *UpdatesRollforwardCall) Do() error {
	var body io.Reader = nil
	params := make(url.Values)
	params.Set("alt", "json")
	if v, ok := c.opt_["fields"]; ok {
		params.Set("fields", fmt.Sprintf("%v", v))
	}
	urls := googleapi.ResolveRelative(c.s.BasePath, "{project}/zones/{zone}/instanceGroupManagers/{instanceGroupManager}/updates/{update}/rollforward")
	urls += "?" + params.Encode()
	req, _ := http.NewRequest("POST", urls, body)
	googleapi.Expand(req.URL, map[string]string{
		"project":              c.project,
		"zone":                 c.zone,
		"instanceGroupManager": c.instanceGroupManager,
		"update":               c.update,
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
	//   "description": "Called on the particular Update endpoint. Rolls forward the update in state from { ROLLING_FORWARD, ROLLING_BACK, PAUSED }. No-op if invoked in state ROLLED_OUT.",
	//   "httpMethod": "POST",
	//   "id": "replicapoolupdater.updates.rollforward",
	//   "parameterOrder": [
	//     "project",
	//     "zone",
	//     "instanceGroupManager",
	//     "update"
	//   ],
	//   "parameters": {
	//     "instanceGroupManager": {
	//       "description": "Name of the instance group manager for this request.",
	//       "location": "path",
	//       "required": true,
	//       "type": "string"
	//     },
	//     "project": {
	//       "description": "Project ID for this request.",
	//       "location": "path",
	//       "required": true,
	//       "type": "string"
	//     },
	//     "update": {
	//       "description": "Unique (in the context of a group) handle of an update.",
	//       "location": "path",
	//       "required": true,
	//       "type": "string"
	//     },
	//     "zone": {
	//       "description": "Zone for the instance group manager.",
	//       "location": "path",
	//       "required": true,
	//       "type": "string"
	//     }
	//   },
	//   "path": "{project}/zones/{zone}/instanceGroupManagers/{instanceGroupManager}/updates/{update}/rollforward",
	//   "scopes": [
	//     "https://www.googleapis.com/auth/cloud-platform",
	//     "https://www.googleapis.com/auth/replicapool"
	//   ]
	// }

}
