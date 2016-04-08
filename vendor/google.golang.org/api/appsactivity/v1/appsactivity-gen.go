// Package appsactivity provides access to the Google Apps Activity API.
//
// See https://developers.google.com/google-apps/activity/
//
// Usage example:
//
//   import "google.golang.org/api/appsactivity/v1"
//   ...
//   appsactivityService, err := appsactivity.New(oauthHttpClient)
package appsactivity

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

const apiId = "appsactivity:v1"
const apiName = "appsactivity"
const apiVersion = "v1"
const basePath = "https://www.googleapis.com/appsactivity/v1/"

// OAuth2 scopes used by this API.
const (
	// View the activity history of your Google Apps
	ActivityScope = "https://www.googleapis.com/auth/activity"

	// View and manage the files and documents in your Google Drive
	DriveScope = "https://www.googleapis.com/auth/drive"

	// View metadata for files and documents in your Google Drive
	DriveMetadataReadonlyScope = "https://www.googleapis.com/auth/drive.metadata.readonly"

	// View the files and documents in your Google Drive
	DriveReadonlyScope = "https://www.googleapis.com/auth/drive.readonly"
)

func New(client *http.Client) (*Service, error) {
	if client == nil {
		return nil, errors.New("client is nil")
	}
	s := &Service{client: client, BasePath: basePath}
	s.Activities = NewActivitiesService(s)
	return s, nil
}

type Service struct {
	client   *http.Client
	BasePath string // API endpoint base URL

	Activities *ActivitiesService
}

func NewActivitiesService(s *Service) *ActivitiesService {
	rs := &ActivitiesService{s: s}
	return rs
}

type ActivitiesService struct {
	s *Service
}

type Activity struct {
	// CombinedEvent: The fields common to all of the singleEvents that make
	// up the Activity.
	CombinedEvent *Event `json:"combinedEvent,omitempty"`

	// SingleEvents: A list of all the Events that make up the Activity.
	SingleEvents []*Event `json:"singleEvents,omitempty"`
}

type Event struct {
	// AdditionalEventTypes: Additional event types. Some events may have
	// multiple types when multiple actions are part of a single event. For
	// example, creating a document, renaming it, and sharing it may be part
	// of a single file-creation event.
	AdditionalEventTypes []string `json:"additionalEventTypes,omitempty"`

	// EventTimeMillis: The time at which the event occurred formatted as
	// Unix time in milliseconds.
	EventTimeMillis uint64 `json:"eventTimeMillis,omitempty,string"`

	// FromUserDeletion: Whether this event is caused by a user being
	// deleted.
	FromUserDeletion bool `json:"fromUserDeletion,omitempty"`

	// Move: Extra information for move type events, such as changes in an
	// object's parents.
	Move *Move `json:"move,omitempty"`

	// PermissionChanges: Extra information for permissionChange type
	// events, such as the user or group the new permission applies to.
	PermissionChanges []*PermissionChange `json:"permissionChanges,omitempty"`

	// PrimaryEventType: The main type of event that occurred.
	PrimaryEventType string `json:"primaryEventType,omitempty"`

	// Rename: Extra information for rename type events, such as the old and
	// new names.
	Rename *Rename `json:"rename,omitempty"`

	// Target: Information specific to the Target object modified by the
	// event.
	Target *Target `json:"target,omitempty"`

	// User: Represents the user responsible for the event.
	User *User `json:"user,omitempty"`
}

type ListActivitiesResponse struct {
	// Activities: List of activities.
	Activities []*Activity `json:"activities,omitempty"`

	// NextPageToken: Token for the next page of results.
	NextPageToken string `json:"nextPageToken,omitempty"`
}

type Move struct {
	// AddedParents: The added parent(s).
	AddedParents []*Parent `json:"addedParents,omitempty"`

	// RemovedParents: The removed parent(s).
	RemovedParents []*Parent `json:"removedParents,omitempty"`
}

type Parent struct {
	// Id: The parent's ID.
	Id string `json:"id,omitempty"`

	// IsRoot: Whether this is the root folder.
	IsRoot bool `json:"isRoot,omitempty"`

	// Title: The parent's title.
	Title string `json:"title,omitempty"`
}

type Permission struct {
	// Name: The name of the user or group the permission applies to.
	Name string `json:"name,omitempty"`

	// PermissionId: The ID for this permission. Corresponds to the Drive
	// API's permission ID returned as part of the Drive Permissions
	// resource.
	PermissionId string `json:"permissionId,omitempty"`

	// Role: Indicates the Google Drive permissions role. The role
	// determines a user's ability to read, write, or comment on the file.
	Role string `json:"role,omitempty"`

	// Type: Indicates how widely permissions are granted.
	Type string `json:"type,omitempty"`

	// User: The user's information if the type is USER.
	User *User `json:"user,omitempty"`

	// WithLink: Whether the permission requires a link to the file.
	WithLink bool `json:"withLink,omitempty"`
}

type PermissionChange struct {
	// AddedPermissions: Lists all Permission objects added.
	AddedPermissions []*Permission `json:"addedPermissions,omitempty"`

	// RemovedPermissions: Lists all Permission objects removed.
	RemovedPermissions []*Permission `json:"removedPermissions,omitempty"`
}

type Photo struct {
	// Url: The URL of the photo.
	Url string `json:"url,omitempty"`
}

type Rename struct {
	// NewTitle: The new title.
	NewTitle string `json:"newTitle,omitempty"`

	// OldTitle: The old title.
	OldTitle string `json:"oldTitle,omitempty"`
}

type Target struct {
	// Id: The ID of the target. For example, in Google Drive, this is the
	// file or folder ID.
	Id string `json:"id,omitempty"`

	// MimeType: The MIME type of the target.
	MimeType string `json:"mimeType,omitempty"`

	// Name: The name of the target. For example, in Google Drive, this is
	// the title of the file.
	Name string `json:"name,omitempty"`
}

type User struct {
	// Name: The displayable name of the user.
	Name string `json:"name,omitempty"`

	// Photo: The profile photo of the user.
	Photo *Photo `json:"photo,omitempty"`
}

// method id "appsactivity.activities.list":

type ActivitiesListCall struct {
	s    *Service
	opt_ map[string]interface{}
}

// List: Returns a list of activities visible to the current logged in
// user. Visible activities are determined by the visiblity settings of
// the object that was acted on, e.g. Drive files a user can see. An
// activity is a record of past events. Multiple events may be merged if
// they are similar. A request is scoped to activities from a given
// Google service using the source parameter.
func (r *ActivitiesService) List() *ActivitiesListCall {
	c := &ActivitiesListCall{s: r.s, opt_: make(map[string]interface{})}
	return c
}

// DriveAncestorId sets the optional parameter "drive.ancestorId":
// Identifies the Drive folder containing the items for which to return
// activities.
func (c *ActivitiesListCall) DriveAncestorId(driveAncestorId string) *ActivitiesListCall {
	c.opt_["drive.ancestorId"] = driveAncestorId
	return c
}

// DriveFileId sets the optional parameter "drive.fileId": Identifies
// the Drive item to return activities for.
func (c *ActivitiesListCall) DriveFileId(driveFileId string) *ActivitiesListCall {
	c.opt_["drive.fileId"] = driveFileId
	return c
}

// GroupingStrategy sets the optional parameter "groupingStrategy":
// Indicates the strategy to use when grouping singleEvents items in the
// associated combinedEvent object.
func (c *ActivitiesListCall) GroupingStrategy(groupingStrategy string) *ActivitiesListCall {
	c.opt_["groupingStrategy"] = groupingStrategy
	return c
}

// PageSize sets the optional parameter "pageSize": The maximum number
// of events to return on a page. The response includes a continuation
// token if there are more events.
func (c *ActivitiesListCall) PageSize(pageSize int64) *ActivitiesListCall {
	c.opt_["pageSize"] = pageSize
	return c
}

// PageToken sets the optional parameter "pageToken": A token to
// retrieve a specific page of results.
func (c *ActivitiesListCall) PageToken(pageToken string) *ActivitiesListCall {
	c.opt_["pageToken"] = pageToken
	return c
}

// Source sets the optional parameter "source": The Google service from
// which to return activities. Possible values of source are:
// -
// drive.google.com
func (c *ActivitiesListCall) Source(source string) *ActivitiesListCall {
	c.opt_["source"] = source
	return c
}

// UserId sets the optional parameter "userId": Indicates the user to
// return activity for. Use the special value me to indicate the
// currently authenticated user.
func (c *ActivitiesListCall) UserId(userId string) *ActivitiesListCall {
	c.opt_["userId"] = userId
	return c
}

// Fields allows partial responses to be retrieved.
// See https://developers.google.com/gdata/docs/2.0/basics#PartialResponse
// for more information.
func (c *ActivitiesListCall) Fields(s ...googleapi.Field) *ActivitiesListCall {
	c.opt_["fields"] = googleapi.CombineFields(s)
	return c
}

func (c *ActivitiesListCall) Do() (*ListActivitiesResponse, error) {
	var body io.Reader = nil
	params := make(url.Values)
	params.Set("alt", "json")
	if v, ok := c.opt_["drive.ancestorId"]; ok {
		params.Set("drive.ancestorId", fmt.Sprintf("%v", v))
	}
	if v, ok := c.opt_["drive.fileId"]; ok {
		params.Set("drive.fileId", fmt.Sprintf("%v", v))
	}
	if v, ok := c.opt_["groupingStrategy"]; ok {
		params.Set("groupingStrategy", fmt.Sprintf("%v", v))
	}
	if v, ok := c.opt_["pageSize"]; ok {
		params.Set("pageSize", fmt.Sprintf("%v", v))
	}
	if v, ok := c.opt_["pageToken"]; ok {
		params.Set("pageToken", fmt.Sprintf("%v", v))
	}
	if v, ok := c.opt_["source"]; ok {
		params.Set("source", fmt.Sprintf("%v", v))
	}
	if v, ok := c.opt_["userId"]; ok {
		params.Set("userId", fmt.Sprintf("%v", v))
	}
	if v, ok := c.opt_["fields"]; ok {
		params.Set("fields", fmt.Sprintf("%v", v))
	}
	urls := googleapi.ResolveRelative(c.s.BasePath, "activities")
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
	var ret *ListActivitiesResponse
	if err := json.NewDecoder(res.Body).Decode(&ret); err != nil {
		return nil, err
	}
	return ret, nil
	// {
	//   "description": "Returns a list of activities visible to the current logged in user. Visible activities are determined by the visiblity settings of the object that was acted on, e.g. Drive files a user can see. An activity is a record of past events. Multiple events may be merged if they are similar. A request is scoped to activities from a given Google service using the source parameter.",
	//   "httpMethod": "GET",
	//   "id": "appsactivity.activities.list",
	//   "parameters": {
	//     "drive.ancestorId": {
	//       "description": "Identifies the Drive folder containing the items for which to return activities.",
	//       "location": "query",
	//       "type": "string"
	//     },
	//     "drive.fileId": {
	//       "description": "Identifies the Drive item to return activities for.",
	//       "location": "query",
	//       "type": "string"
	//     },
	//     "groupingStrategy": {
	//       "default": "driveUi",
	//       "description": "Indicates the strategy to use when grouping singleEvents items in the associated combinedEvent object.",
	//       "enum": [
	//         "driveUi",
	//         "none"
	//       ],
	//       "enumDescriptions": [
	//         "",
	//         ""
	//       ],
	//       "location": "query",
	//       "type": "string"
	//     },
	//     "pageSize": {
	//       "default": "50",
	//       "description": "The maximum number of events to return on a page. The response includes a continuation token if there are more events.",
	//       "format": "int32",
	//       "location": "query",
	//       "type": "integer"
	//     },
	//     "pageToken": {
	//       "description": "A token to retrieve a specific page of results.",
	//       "location": "query",
	//       "type": "string"
	//     },
	//     "source": {
	//       "description": "The Google service from which to return activities. Possible values of source are: \n- drive.google.com",
	//       "location": "query",
	//       "type": "string"
	//     },
	//     "userId": {
	//       "default": "me",
	//       "description": "Indicates the user to return activity for. Use the special value me to indicate the currently authenticated user.",
	//       "location": "query",
	//       "type": "string"
	//     }
	//   },
	//   "path": "activities",
	//   "response": {
	//     "$ref": "ListActivitiesResponse"
	//   },
	//   "scopes": [
	//     "https://www.googleapis.com/auth/activity",
	//     "https://www.googleapis.com/auth/drive",
	//     "https://www.googleapis.com/auth/drive.metadata.readonly",
	//     "https://www.googleapis.com/auth/drive.readonly"
	//   ]
	// }

}
