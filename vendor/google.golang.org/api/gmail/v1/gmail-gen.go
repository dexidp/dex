// Package gmail provides access to the Gmail API.
//
// See https://developers.google.com/gmail/api/
//
// Usage example:
//
//   import "google.golang.org/api/gmail/v1"
//   ...
//   gmailService, err := gmail.New(oauthHttpClient)
package gmail

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

const apiId = "gmail:v1"
const apiName = "gmail"
const apiVersion = "v1"
const basePath = "https://www.googleapis.com/gmail/v1/users/"

// OAuth2 scopes used by this API.
const (
	// View and manage your mail
	MailGoogleComScope = "https://mail.google.com/"

	// Manage drafts and send emails
	GmailComposeScope = "https://www.googleapis.com/auth/gmail.compose"

	// View and modify but not delete your email
	GmailModifyScope = "https://www.googleapis.com/auth/gmail.modify"

	// View your emails messages and settings
	GmailReadonlyScope = "https://www.googleapis.com/auth/gmail.readonly"
)

func New(client *http.Client) (*Service, error) {
	if client == nil {
		return nil, errors.New("client is nil")
	}
	s := &Service{client: client, BasePath: basePath}
	s.Users = NewUsersService(s)
	return s, nil
}

type Service struct {
	client   *http.Client
	BasePath string // API endpoint base URL

	Users *UsersService
}

func NewUsersService(s *Service) *UsersService {
	rs := &UsersService{s: s}
	rs.Drafts = NewUsersDraftsService(s)
	rs.History = NewUsersHistoryService(s)
	rs.Labels = NewUsersLabelsService(s)
	rs.Messages = NewUsersMessagesService(s)
	rs.Threads = NewUsersThreadsService(s)
	return rs
}

type UsersService struct {
	s *Service

	Drafts *UsersDraftsService

	History *UsersHistoryService

	Labels *UsersLabelsService

	Messages *UsersMessagesService

	Threads *UsersThreadsService
}

func NewUsersDraftsService(s *Service) *UsersDraftsService {
	rs := &UsersDraftsService{s: s}
	return rs
}

type UsersDraftsService struct {
	s *Service
}

func NewUsersHistoryService(s *Service) *UsersHistoryService {
	rs := &UsersHistoryService{s: s}
	return rs
}

type UsersHistoryService struct {
	s *Service
}

func NewUsersLabelsService(s *Service) *UsersLabelsService {
	rs := &UsersLabelsService{s: s}
	return rs
}

type UsersLabelsService struct {
	s *Service
}

func NewUsersMessagesService(s *Service) *UsersMessagesService {
	rs := &UsersMessagesService{s: s}
	rs.Attachments = NewUsersMessagesAttachmentsService(s)
	return rs
}

type UsersMessagesService struct {
	s *Service

	Attachments *UsersMessagesAttachmentsService
}

func NewUsersMessagesAttachmentsService(s *Service) *UsersMessagesAttachmentsService {
	rs := &UsersMessagesAttachmentsService{s: s}
	return rs
}

type UsersMessagesAttachmentsService struct {
	s *Service
}

func NewUsersThreadsService(s *Service) *UsersThreadsService {
	rs := &UsersThreadsService{s: s}
	return rs
}

type UsersThreadsService struct {
	s *Service
}

type Draft struct {
	// Id: The immutable ID of the draft.
	Id string `json:"id,omitempty"`

	// Message: The message content of the draft.
	Message *Message `json:"message,omitempty"`
}

type History struct {
	// Id: The mailbox sequence ID.
	Id uint64 `json:"id,omitempty,string"`

	// Messages: The messages that changed in this history record.
	Messages []*Message `json:"messages,omitempty"`
}

type Label struct {
	// Id: The immutable ID of the label.
	Id string `json:"id,omitempty"`

	// LabelListVisibility: The visibility of the label in the label list in
	// the Gmail web interface.
	LabelListVisibility string `json:"labelListVisibility,omitempty"`

	// MessageListVisibility: The visibility of the label in the message
	// list in the Gmail web interface.
	MessageListVisibility string `json:"messageListVisibility,omitempty"`

	// MessagesTotal: The total number of messages with the label.
	MessagesTotal int64 `json:"messagesTotal,omitempty"`

	// MessagesUnread: The number of unread messages with the label.
	MessagesUnread int64 `json:"messagesUnread,omitempty"`

	// Name: The display name of the label.
	Name string `json:"name,omitempty"`

	// ThreadsTotal: The total number of threads with the label.
	ThreadsTotal int64 `json:"threadsTotal,omitempty"`

	// ThreadsUnread: The number of unread threads with the label.
	ThreadsUnread int64 `json:"threadsUnread,omitempty"`

	// Type: The owner type for the label. User labels are created by the
	// user and can be modified and deleted by the user and can be applied
	// to any message or thread. System labels are internally created and
	// cannot be added, modified, or deleted. System labels may be able to
	// be applied to or removed from messages and threads under some
	// circumstances but this is not guaranteed. For example, users can
	// apply and remove the INBOX and UNREAD labels from messages and
	// threads, but cannot apply or remove the DRAFTS or SENT labels from
	// messages or threads.
	Type string `json:"type,omitempty"`
}

type ListDraftsResponse struct {
	// Drafts: List of drafts.
	Drafts []*Draft `json:"drafts,omitempty"`

	// NextPageToken: Token to retrieve the next page of results in the
	// list.
	NextPageToken string `json:"nextPageToken,omitempty"`

	// ResultSizeEstimate: Estimated total number of results.
	ResultSizeEstimate int64 `json:"resultSizeEstimate,omitempty"`
}

type ListHistoryResponse struct {
	// History: List of history records.
	History []*History `json:"history,omitempty"`

	// HistoryId: The ID of the mailbox's current history record.
	HistoryId uint64 `json:"historyId,omitempty,string"`

	// NextPageToken: Page token to retrieve the next page of results in the
	// list.
	NextPageToken string `json:"nextPageToken,omitempty"`
}

type ListLabelsResponse struct {
	// Labels: List of labels.
	Labels []*Label `json:"labels,omitempty"`
}

type ListMessagesResponse struct {
	// Messages: List of messages.
	Messages []*Message `json:"messages,omitempty"`

	// NextPageToken: Token to retrieve the next page of results in the
	// list.
	NextPageToken string `json:"nextPageToken,omitempty"`

	// ResultSizeEstimate: Estimated total number of results.
	ResultSizeEstimate int64 `json:"resultSizeEstimate,omitempty"`
}

type ListThreadsResponse struct {
	// NextPageToken: Page token to retrieve the next page of results in the
	// list.
	NextPageToken string `json:"nextPageToken,omitempty"`

	// ResultSizeEstimate: Estimated total number of results.
	ResultSizeEstimate int64 `json:"resultSizeEstimate,omitempty"`

	// Threads: List of threads.
	Threads []*Thread `json:"threads,omitempty"`
}

type Message struct {
	// HistoryId: The ID of the last history record that modified this
	// message.
	HistoryId uint64 `json:"historyId,omitempty,string"`

	// Id: The immutable ID of the message.
	Id string `json:"id,omitempty"`

	// LabelIds: List of IDs of labels applied to this message.
	LabelIds []string `json:"labelIds,omitempty"`

	// Payload: The parsed email structure in the message parts.
	Payload *MessagePart `json:"payload,omitempty"`

	// Raw: The entire email message in an RFC 2822 formatted and URL-safe
	// base64 encoded string. Returned in messages.get and drafts.get
	// responses when the format=RAW parameter is supplied.
	Raw string `json:"raw,omitempty"`

	// SizeEstimate: Estimated size in bytes of the message.
	SizeEstimate int64 `json:"sizeEstimate,omitempty"`

	// Snippet: A short part of the message text.
	Snippet string `json:"snippet,omitempty"`

	// ThreadId: The ID of the thread the message belongs to. To add a
	// message or draft to a thread, the following criteria must be met:
	// -
	// The requested threadId must be specified on the Message or
	// Draft.Message you supply with your request.
	// - The References and
	// In-Reply-To headers must be set in compliance with the RFC 2822
	// standard.
	// - The Subject headers must match.
	ThreadId string `json:"threadId,omitempty"`
}

type MessagePart struct {
	// Body: The message part body for this part, which may be empty for
	// container MIME message parts.
	Body *MessagePartBody `json:"body,omitempty"`

	// Filename: The filename of the attachment. Only present if this
	// message part represents an attachment.
	Filename string `json:"filename,omitempty"`

	// Headers: List of headers on this message part. For the top-level
	// message part, representing the entire message payload, it will
	// contain the standard RFC 2822 email headers such as To, From, and
	// Subject.
	Headers []*MessagePartHeader `json:"headers,omitempty"`

	// MimeType: The MIME type of the message part.
	MimeType string `json:"mimeType,omitempty"`

	// PartId: The immutable ID of the message part.
	PartId string `json:"partId,omitempty"`

	// Parts: The child MIME message parts of this part. This only applies
	// to container MIME message parts, for example multipart/*. For non-
	// container MIME message part types, such as text/plain, this field is
	// empty. For more information, see RFC 1521.
	Parts []*MessagePart `json:"parts,omitempty"`
}

type MessagePartBody struct {
	// AttachmentId: When present, contains the ID of an external attachment
	// that can be retrieved in a separate messages.attachments.get request.
	// When not present, the entire content of the message part body is
	// contained in the data field.
	AttachmentId string `json:"attachmentId,omitempty"`

	// Data: The body data of a MIME message part. May be empty for MIME
	// container types that have no message body or when the body data is
	// sent as a separate attachment. An attachment ID is present if the
	// body data is contained in a separate attachment.
	Data string `json:"data,omitempty"`

	// Size: Total number of bytes in the body of the message part.
	Size int64 `json:"size,omitempty"`
}

type MessagePartHeader struct {
	// Name: The name of the header before the : separator. For example, To.
	Name string `json:"name,omitempty"`

	// Value: The value of the header after the : separator. For example,
	// someuser@example.com.
	Value string `json:"value,omitempty"`
}

type ModifyMessageRequest struct {
	// AddLabelIds: A list of IDs of labels to add to this message.
	AddLabelIds []string `json:"addLabelIds,omitempty"`

	// RemoveLabelIds: A list IDs of labels to remove from this message.
	RemoveLabelIds []string `json:"removeLabelIds,omitempty"`
}

type ModifyThreadRequest struct {
	// AddLabelIds: A list of IDs of labels to add to this thread.
	AddLabelIds []string `json:"addLabelIds,omitempty"`

	// RemoveLabelIds: A list of IDs of labels to remove from this thread.
	RemoveLabelIds []string `json:"removeLabelIds,omitempty"`
}

type Profile struct {
	// EmailAddress: The user's email address.
	EmailAddress string `json:"emailAddress,omitempty"`

	// HistoryId: The ID of the mailbox's current history record.
	HistoryId uint64 `json:"historyId,omitempty,string"`

	// MessagesTotal: The total number of messages in the mailbox.
	MessagesTotal int64 `json:"messagesTotal,omitempty"`

	// ThreadsTotal: The total number of threads in the mailbox.
	ThreadsTotal int64 `json:"threadsTotal,omitempty"`
}

type Thread struct {
	// HistoryId: The ID of the last history record that modified this
	// thread.
	HistoryId uint64 `json:"historyId,omitempty,string"`

	// Id: The unique ID of the thread.
	Id string `json:"id,omitempty"`

	// Messages: The list of messages in the thread.
	Messages []*Message `json:"messages,omitempty"`

	// Snippet: A short part of the message text.
	Snippet string `json:"snippet,omitempty"`
}

// method id "gmail.users.getProfile":

type UsersGetProfileCall struct {
	s      *Service
	userId string
	opt_   map[string]interface{}
}

// GetProfile: Gets the current user's Gmail profile.
func (r *UsersService) GetProfile(userId string) *UsersGetProfileCall {
	c := &UsersGetProfileCall{s: r.s, opt_: make(map[string]interface{})}
	c.userId = userId
	return c
}

// Fields allows partial responses to be retrieved.
// See https://developers.google.com/gdata/docs/2.0/basics#PartialResponse
// for more information.
func (c *UsersGetProfileCall) Fields(s ...googleapi.Field) *UsersGetProfileCall {
	c.opt_["fields"] = googleapi.CombineFields(s)
	return c
}

func (c *UsersGetProfileCall) Do() (*Profile, error) {
	var body io.Reader = nil
	params := make(url.Values)
	params.Set("alt", "json")
	if v, ok := c.opt_["fields"]; ok {
		params.Set("fields", fmt.Sprintf("%v", v))
	}
	urls := googleapi.ResolveRelative(c.s.BasePath, "{userId}/profile")
	urls += "?" + params.Encode()
	req, _ := http.NewRequest("GET", urls, body)
	googleapi.Expand(req.URL, map[string]string{
		"userId": c.userId,
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
	var ret *Profile
	if err := json.NewDecoder(res.Body).Decode(&ret); err != nil {
		return nil, err
	}
	return ret, nil
	// {
	//   "description": "Gets the current user's Gmail profile.",
	//   "httpMethod": "GET",
	//   "id": "gmail.users.getProfile",
	//   "parameterOrder": [
	//     "userId"
	//   ],
	//   "parameters": {
	//     "userId": {
	//       "default": "me",
	//       "description": "The user's email address. The special value me can be used to indicate the authenticated user.",
	//       "location": "path",
	//       "required": true,
	//       "type": "string"
	//     }
	//   },
	//   "path": "{userId}/profile",
	//   "response": {
	//     "$ref": "Profile"
	//   },
	//   "scopes": [
	//     "https://mail.google.com/",
	//     "https://www.googleapis.com/auth/gmail.compose",
	//     "https://www.googleapis.com/auth/gmail.modify",
	//     "https://www.googleapis.com/auth/gmail.readonly"
	//   ]
	// }

}

// method id "gmail.users.drafts.create":

type UsersDraftsCreateCall struct {
	s      *Service
	userId string
	draft  *Draft
	opt_   map[string]interface{}
	media_ io.Reader
}

// Create: Creates a new draft with the DRAFT label.
func (r *UsersDraftsService) Create(userId string, draft *Draft) *UsersDraftsCreateCall {
	c := &UsersDraftsCreateCall{s: r.s, opt_: make(map[string]interface{})}
	c.userId = userId
	c.draft = draft
	return c
}
func (c *UsersDraftsCreateCall) Media(r io.Reader) *UsersDraftsCreateCall {
	c.media_ = r
	return c
}

// Fields allows partial responses to be retrieved.
// See https://developers.google.com/gdata/docs/2.0/basics#PartialResponse
// for more information.
func (c *UsersDraftsCreateCall) Fields(s ...googleapi.Field) *UsersDraftsCreateCall {
	c.opt_["fields"] = googleapi.CombineFields(s)
	return c
}

func (c *UsersDraftsCreateCall) Do() (*Draft, error) {
	var body io.Reader = nil
	body, err := googleapi.WithoutDataWrapper.JSONReader(c.draft)
	if err != nil {
		return nil, err
	}
	ctype := "application/json"
	params := make(url.Values)
	params.Set("alt", "json")
	if v, ok := c.opt_["fields"]; ok {
		params.Set("fields", fmt.Sprintf("%v", v))
	}
	urls := googleapi.ResolveRelative(c.s.BasePath, "{userId}/drafts")
	if c.media_ != nil {
		urls = strings.Replace(urls, "https://www.googleapis.com/", "https://www.googleapis.com/upload/", 1)
		params.Set("uploadType", "multipart")
	}
	urls += "?" + params.Encode()
	contentLength_, hasMedia_ := googleapi.ConditionallyIncludeMedia(c.media_, &body, &ctype)
	req, _ := http.NewRequest("POST", urls, body)
	googleapi.Expand(req.URL, map[string]string{
		"userId": c.userId,
	})
	if hasMedia_ {
		req.ContentLength = contentLength_
	}
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
	var ret *Draft
	if err := json.NewDecoder(res.Body).Decode(&ret); err != nil {
		return nil, err
	}
	return ret, nil
	// {
	//   "description": "Creates a new draft with the DRAFT label.",
	//   "httpMethod": "POST",
	//   "id": "gmail.users.drafts.create",
	//   "mediaUpload": {
	//     "accept": [
	//       "message/rfc822"
	//     ],
	//     "maxSize": "35MB",
	//     "protocols": {
	//       "resumable": {
	//         "multipart": true,
	//         "path": "/resumable/upload/gmail/v1/users/{userId}/drafts"
	//       },
	//       "simple": {
	//         "multipart": true,
	//         "path": "/upload/gmail/v1/users/{userId}/drafts"
	//       }
	//     }
	//   },
	//   "parameterOrder": [
	//     "userId"
	//   ],
	//   "parameters": {
	//     "userId": {
	//       "default": "me",
	//       "description": "The user's email address. The special value me can be used to indicate the authenticated user.",
	//       "location": "path",
	//       "required": true,
	//       "type": "string"
	//     }
	//   },
	//   "path": "{userId}/drafts",
	//   "request": {
	//     "$ref": "Draft"
	//   },
	//   "response": {
	//     "$ref": "Draft"
	//   },
	//   "scopes": [
	//     "https://mail.google.com/",
	//     "https://www.googleapis.com/auth/gmail.compose",
	//     "https://www.googleapis.com/auth/gmail.modify"
	//   ],
	//   "supportsMediaUpload": true
	// }

}

// method id "gmail.users.drafts.delete":

type UsersDraftsDeleteCall struct {
	s      *Service
	userId string
	id     string
	opt_   map[string]interface{}
}

// Delete: Immediately and permanently deletes the specified draft. Does
// not simply trash it.
func (r *UsersDraftsService) Delete(userId string, id string) *UsersDraftsDeleteCall {
	c := &UsersDraftsDeleteCall{s: r.s, opt_: make(map[string]interface{})}
	c.userId = userId
	c.id = id
	return c
}

// Fields allows partial responses to be retrieved.
// See https://developers.google.com/gdata/docs/2.0/basics#PartialResponse
// for more information.
func (c *UsersDraftsDeleteCall) Fields(s ...googleapi.Field) *UsersDraftsDeleteCall {
	c.opt_["fields"] = googleapi.CombineFields(s)
	return c
}

func (c *UsersDraftsDeleteCall) Do() error {
	var body io.Reader = nil
	params := make(url.Values)
	params.Set("alt", "json")
	if v, ok := c.opt_["fields"]; ok {
		params.Set("fields", fmt.Sprintf("%v", v))
	}
	urls := googleapi.ResolveRelative(c.s.BasePath, "{userId}/drafts/{id}")
	urls += "?" + params.Encode()
	req, _ := http.NewRequest("DELETE", urls, body)
	googleapi.Expand(req.URL, map[string]string{
		"userId": c.userId,
		"id":     c.id,
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
	//   "description": "Immediately and permanently deletes the specified draft. Does not simply trash it.",
	//   "httpMethod": "DELETE",
	//   "id": "gmail.users.drafts.delete",
	//   "parameterOrder": [
	//     "userId",
	//     "id"
	//   ],
	//   "parameters": {
	//     "id": {
	//       "description": "The ID of the draft to delete.",
	//       "location": "path",
	//       "required": true,
	//       "type": "string"
	//     },
	//     "userId": {
	//       "default": "me",
	//       "description": "The user's email address. The special value me can be used to indicate the authenticated user.",
	//       "location": "path",
	//       "required": true,
	//       "type": "string"
	//     }
	//   },
	//   "path": "{userId}/drafts/{id}",
	//   "scopes": [
	//     "https://mail.google.com/",
	//     "https://www.googleapis.com/auth/gmail.compose",
	//     "https://www.googleapis.com/auth/gmail.modify"
	//   ]
	// }

}

// method id "gmail.users.drafts.get":

type UsersDraftsGetCall struct {
	s      *Service
	userId string
	id     string
	opt_   map[string]interface{}
}

// Get: Gets the specified draft.
func (r *UsersDraftsService) Get(userId string, id string) *UsersDraftsGetCall {
	c := &UsersDraftsGetCall{s: r.s, opt_: make(map[string]interface{})}
	c.userId = userId
	c.id = id
	return c
}

// Format sets the optional parameter "format": The format to return the
// draft in.
func (c *UsersDraftsGetCall) Format(format string) *UsersDraftsGetCall {
	c.opt_["format"] = format
	return c
}

// Fields allows partial responses to be retrieved.
// See https://developers.google.com/gdata/docs/2.0/basics#PartialResponse
// for more information.
func (c *UsersDraftsGetCall) Fields(s ...googleapi.Field) *UsersDraftsGetCall {
	c.opt_["fields"] = googleapi.CombineFields(s)
	return c
}

func (c *UsersDraftsGetCall) Do() (*Draft, error) {
	var body io.Reader = nil
	params := make(url.Values)
	params.Set("alt", "json")
	if v, ok := c.opt_["format"]; ok {
		params.Set("format", fmt.Sprintf("%v", v))
	}
	if v, ok := c.opt_["fields"]; ok {
		params.Set("fields", fmt.Sprintf("%v", v))
	}
	urls := googleapi.ResolveRelative(c.s.BasePath, "{userId}/drafts/{id}")
	urls += "?" + params.Encode()
	req, _ := http.NewRequest("GET", urls, body)
	googleapi.Expand(req.URL, map[string]string{
		"userId": c.userId,
		"id":     c.id,
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
	var ret *Draft
	if err := json.NewDecoder(res.Body).Decode(&ret); err != nil {
		return nil, err
	}
	return ret, nil
	// {
	//   "description": "Gets the specified draft.",
	//   "httpMethod": "GET",
	//   "id": "gmail.users.drafts.get",
	//   "parameterOrder": [
	//     "userId",
	//     "id"
	//   ],
	//   "parameters": {
	//     "format": {
	//       "default": "full",
	//       "description": "The format to return the draft in.",
	//       "enum": [
	//         "full",
	//         "metadata",
	//         "minimal",
	//         "raw"
	//       ],
	//       "enumDescriptions": [
	//         "",
	//         "",
	//         "",
	//         ""
	//       ],
	//       "location": "query",
	//       "type": "string"
	//     },
	//     "id": {
	//       "description": "The ID of the draft to retrieve.",
	//       "location": "path",
	//       "required": true,
	//       "type": "string"
	//     },
	//     "userId": {
	//       "default": "me",
	//       "description": "The user's email address. The special value me can be used to indicate the authenticated user.",
	//       "location": "path",
	//       "required": true,
	//       "type": "string"
	//     }
	//   },
	//   "path": "{userId}/drafts/{id}",
	//   "response": {
	//     "$ref": "Draft"
	//   },
	//   "scopes": [
	//     "https://mail.google.com/",
	//     "https://www.googleapis.com/auth/gmail.compose",
	//     "https://www.googleapis.com/auth/gmail.modify",
	//     "https://www.googleapis.com/auth/gmail.readonly"
	//   ]
	// }

}

// method id "gmail.users.drafts.list":

type UsersDraftsListCall struct {
	s      *Service
	userId string
	opt_   map[string]interface{}
}

// List: Lists the drafts in the user's mailbox.
func (r *UsersDraftsService) List(userId string) *UsersDraftsListCall {
	c := &UsersDraftsListCall{s: r.s, opt_: make(map[string]interface{})}
	c.userId = userId
	return c
}

// MaxResults sets the optional parameter "maxResults": Maximum number
// of drafts to return.
func (c *UsersDraftsListCall) MaxResults(maxResults int64) *UsersDraftsListCall {
	c.opt_["maxResults"] = maxResults
	return c
}

// PageToken sets the optional parameter "pageToken": Page token to
// retrieve a specific page of results in the list.
func (c *UsersDraftsListCall) PageToken(pageToken string) *UsersDraftsListCall {
	c.opt_["pageToken"] = pageToken
	return c
}

// Fields allows partial responses to be retrieved.
// See https://developers.google.com/gdata/docs/2.0/basics#PartialResponse
// for more information.
func (c *UsersDraftsListCall) Fields(s ...googleapi.Field) *UsersDraftsListCall {
	c.opt_["fields"] = googleapi.CombineFields(s)
	return c
}

func (c *UsersDraftsListCall) Do() (*ListDraftsResponse, error) {
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
	urls := googleapi.ResolveRelative(c.s.BasePath, "{userId}/drafts")
	urls += "?" + params.Encode()
	req, _ := http.NewRequest("GET", urls, body)
	googleapi.Expand(req.URL, map[string]string{
		"userId": c.userId,
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
	var ret *ListDraftsResponse
	if err := json.NewDecoder(res.Body).Decode(&ret); err != nil {
		return nil, err
	}
	return ret, nil
	// {
	//   "description": "Lists the drafts in the user's mailbox.",
	//   "httpMethod": "GET",
	//   "id": "gmail.users.drafts.list",
	//   "parameterOrder": [
	//     "userId"
	//   ],
	//   "parameters": {
	//     "maxResults": {
	//       "default": "100",
	//       "description": "Maximum number of drafts to return.",
	//       "format": "uint32",
	//       "location": "query",
	//       "type": "integer"
	//     },
	//     "pageToken": {
	//       "description": "Page token to retrieve a specific page of results in the list.",
	//       "location": "query",
	//       "type": "string"
	//     },
	//     "userId": {
	//       "default": "me",
	//       "description": "The user's email address. The special value me can be used to indicate the authenticated user.",
	//       "location": "path",
	//       "required": true,
	//       "type": "string"
	//     }
	//   },
	//   "path": "{userId}/drafts",
	//   "response": {
	//     "$ref": "ListDraftsResponse"
	//   },
	//   "scopes": [
	//     "https://mail.google.com/",
	//     "https://www.googleapis.com/auth/gmail.compose",
	//     "https://www.googleapis.com/auth/gmail.modify",
	//     "https://www.googleapis.com/auth/gmail.readonly"
	//   ]
	// }

}

// method id "gmail.users.drafts.send":

type UsersDraftsSendCall struct {
	s      *Service
	userId string
	draft  *Draft
	opt_   map[string]interface{}
	media_ io.Reader
}

// Send: Sends the specified, existing draft to the recipients in the
// To, Cc, and Bcc headers.
func (r *UsersDraftsService) Send(userId string, draft *Draft) *UsersDraftsSendCall {
	c := &UsersDraftsSendCall{s: r.s, opt_: make(map[string]interface{})}
	c.userId = userId
	c.draft = draft
	return c
}
func (c *UsersDraftsSendCall) Media(r io.Reader) *UsersDraftsSendCall {
	c.media_ = r
	return c
}

// Fields allows partial responses to be retrieved.
// See https://developers.google.com/gdata/docs/2.0/basics#PartialResponse
// for more information.
func (c *UsersDraftsSendCall) Fields(s ...googleapi.Field) *UsersDraftsSendCall {
	c.opt_["fields"] = googleapi.CombineFields(s)
	return c
}

func (c *UsersDraftsSendCall) Do() (*Message, error) {
	var body io.Reader = nil
	body, err := googleapi.WithoutDataWrapper.JSONReader(c.draft)
	if err != nil {
		return nil, err
	}
	ctype := "application/json"
	params := make(url.Values)
	params.Set("alt", "json")
	if v, ok := c.opt_["fields"]; ok {
		params.Set("fields", fmt.Sprintf("%v", v))
	}
	urls := googleapi.ResolveRelative(c.s.BasePath, "{userId}/drafts/send")
	if c.media_ != nil {
		urls = strings.Replace(urls, "https://www.googleapis.com/", "https://www.googleapis.com/upload/", 1)
		params.Set("uploadType", "multipart")
	}
	urls += "?" + params.Encode()
	contentLength_, hasMedia_ := googleapi.ConditionallyIncludeMedia(c.media_, &body, &ctype)
	req, _ := http.NewRequest("POST", urls, body)
	googleapi.Expand(req.URL, map[string]string{
		"userId": c.userId,
	})
	if hasMedia_ {
		req.ContentLength = contentLength_
	}
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
	var ret *Message
	if err := json.NewDecoder(res.Body).Decode(&ret); err != nil {
		return nil, err
	}
	return ret, nil
	// {
	//   "description": "Sends the specified, existing draft to the recipients in the To, Cc, and Bcc headers.",
	//   "httpMethod": "POST",
	//   "id": "gmail.users.drafts.send",
	//   "mediaUpload": {
	//     "accept": [
	//       "message/rfc822"
	//     ],
	//     "maxSize": "35MB",
	//     "protocols": {
	//       "resumable": {
	//         "multipart": true,
	//         "path": "/resumable/upload/gmail/v1/users/{userId}/drafts/send"
	//       },
	//       "simple": {
	//         "multipart": true,
	//         "path": "/upload/gmail/v1/users/{userId}/drafts/send"
	//       }
	//     }
	//   },
	//   "parameterOrder": [
	//     "userId"
	//   ],
	//   "parameters": {
	//     "userId": {
	//       "default": "me",
	//       "description": "The user's email address. The special value me can be used to indicate the authenticated user.",
	//       "location": "path",
	//       "required": true,
	//       "type": "string"
	//     }
	//   },
	//   "path": "{userId}/drafts/send",
	//   "request": {
	//     "$ref": "Draft"
	//   },
	//   "response": {
	//     "$ref": "Message"
	//   },
	//   "scopes": [
	//     "https://mail.google.com/",
	//     "https://www.googleapis.com/auth/gmail.compose",
	//     "https://www.googleapis.com/auth/gmail.modify"
	//   ],
	//   "supportsMediaUpload": true
	// }

}

// method id "gmail.users.drafts.update":

type UsersDraftsUpdateCall struct {
	s      *Service
	userId string
	id     string
	draft  *Draft
	opt_   map[string]interface{}
	media_ io.Reader
}

// Update: Replaces a draft's content.
func (r *UsersDraftsService) Update(userId string, id string, draft *Draft) *UsersDraftsUpdateCall {
	c := &UsersDraftsUpdateCall{s: r.s, opt_: make(map[string]interface{})}
	c.userId = userId
	c.id = id
	c.draft = draft
	return c
}
func (c *UsersDraftsUpdateCall) Media(r io.Reader) *UsersDraftsUpdateCall {
	c.media_ = r
	return c
}

// Fields allows partial responses to be retrieved.
// See https://developers.google.com/gdata/docs/2.0/basics#PartialResponse
// for more information.
func (c *UsersDraftsUpdateCall) Fields(s ...googleapi.Field) *UsersDraftsUpdateCall {
	c.opt_["fields"] = googleapi.CombineFields(s)
	return c
}

func (c *UsersDraftsUpdateCall) Do() (*Draft, error) {
	var body io.Reader = nil
	body, err := googleapi.WithoutDataWrapper.JSONReader(c.draft)
	if err != nil {
		return nil, err
	}
	ctype := "application/json"
	params := make(url.Values)
	params.Set("alt", "json")
	if v, ok := c.opt_["fields"]; ok {
		params.Set("fields", fmt.Sprintf("%v", v))
	}
	urls := googleapi.ResolveRelative(c.s.BasePath, "{userId}/drafts/{id}")
	if c.media_ != nil {
		urls = strings.Replace(urls, "https://www.googleapis.com/", "https://www.googleapis.com/upload/", 1)
		params.Set("uploadType", "multipart")
	}
	urls += "?" + params.Encode()
	contentLength_, hasMedia_ := googleapi.ConditionallyIncludeMedia(c.media_, &body, &ctype)
	req, _ := http.NewRequest("PUT", urls, body)
	googleapi.Expand(req.URL, map[string]string{
		"userId": c.userId,
		"id":     c.id,
	})
	if hasMedia_ {
		req.ContentLength = contentLength_
	}
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
	var ret *Draft
	if err := json.NewDecoder(res.Body).Decode(&ret); err != nil {
		return nil, err
	}
	return ret, nil
	// {
	//   "description": "Replaces a draft's content.",
	//   "httpMethod": "PUT",
	//   "id": "gmail.users.drafts.update",
	//   "mediaUpload": {
	//     "accept": [
	//       "message/rfc822"
	//     ],
	//     "maxSize": "35MB",
	//     "protocols": {
	//       "resumable": {
	//         "multipart": true,
	//         "path": "/resumable/upload/gmail/v1/users/{userId}/drafts/{id}"
	//       },
	//       "simple": {
	//         "multipart": true,
	//         "path": "/upload/gmail/v1/users/{userId}/drafts/{id}"
	//       }
	//     }
	//   },
	//   "parameterOrder": [
	//     "userId",
	//     "id"
	//   ],
	//   "parameters": {
	//     "id": {
	//       "description": "The ID of the draft to update.",
	//       "location": "path",
	//       "required": true,
	//       "type": "string"
	//     },
	//     "userId": {
	//       "default": "me",
	//       "description": "The user's email address. The special value me can be used to indicate the authenticated user.",
	//       "location": "path",
	//       "required": true,
	//       "type": "string"
	//     }
	//   },
	//   "path": "{userId}/drafts/{id}",
	//   "request": {
	//     "$ref": "Draft"
	//   },
	//   "response": {
	//     "$ref": "Draft"
	//   },
	//   "scopes": [
	//     "https://mail.google.com/",
	//     "https://www.googleapis.com/auth/gmail.compose",
	//     "https://www.googleapis.com/auth/gmail.modify"
	//   ],
	//   "supportsMediaUpload": true
	// }

}

// method id "gmail.users.history.list":

type UsersHistoryListCall struct {
	s      *Service
	userId string
	opt_   map[string]interface{}
}

// List: Lists the history of all changes to the given mailbox. History
// results are returned in chronological order (increasing historyId).
func (r *UsersHistoryService) List(userId string) *UsersHistoryListCall {
	c := &UsersHistoryListCall{s: r.s, opt_: make(map[string]interface{})}
	c.userId = userId
	return c
}

// LabelId sets the optional parameter "labelId": Only return messages
// with a label matching the ID.
func (c *UsersHistoryListCall) LabelId(labelId string) *UsersHistoryListCall {
	c.opt_["labelId"] = labelId
	return c
}

// MaxResults sets the optional parameter "maxResults": The maximum
// number of history records to return.
func (c *UsersHistoryListCall) MaxResults(maxResults int64) *UsersHistoryListCall {
	c.opt_["maxResults"] = maxResults
	return c
}

// PageToken sets the optional parameter "pageToken": Page token to
// retrieve a specific page of results in the list.
func (c *UsersHistoryListCall) PageToken(pageToken string) *UsersHistoryListCall {
	c.opt_["pageToken"] = pageToken
	return c
}

// StartHistoryId sets the optional parameter "startHistoryId":
// Required. Returns history records after the specified startHistoryId.
// The supplied startHistoryId should be obtained from the historyId of
// a message, thread, or previous list response. History IDs increase
// chronologically but are not contiguous with random gaps in between
// valid IDs. Supplying an invalid or out of date startHistoryId
// typically returns an HTTP 404 error code. A historyId is typically
// valid for at least a week, but in some circumstances may be valid for
// only a few hours. If you receive an HTTP 404 error response, your
// application should perform a full sync. If you receive no
// nextPageToken in the response, there are no updates to retrieve and
// you can store the returned historyId for a future request.
func (c *UsersHistoryListCall) StartHistoryId(startHistoryId uint64) *UsersHistoryListCall {
	c.opt_["startHistoryId"] = startHistoryId
	return c
}

// Fields allows partial responses to be retrieved.
// See https://developers.google.com/gdata/docs/2.0/basics#PartialResponse
// for more information.
func (c *UsersHistoryListCall) Fields(s ...googleapi.Field) *UsersHistoryListCall {
	c.opt_["fields"] = googleapi.CombineFields(s)
	return c
}

func (c *UsersHistoryListCall) Do() (*ListHistoryResponse, error) {
	var body io.Reader = nil
	params := make(url.Values)
	params.Set("alt", "json")
	if v, ok := c.opt_["labelId"]; ok {
		params.Set("labelId", fmt.Sprintf("%v", v))
	}
	if v, ok := c.opt_["maxResults"]; ok {
		params.Set("maxResults", fmt.Sprintf("%v", v))
	}
	if v, ok := c.opt_["pageToken"]; ok {
		params.Set("pageToken", fmt.Sprintf("%v", v))
	}
	if v, ok := c.opt_["startHistoryId"]; ok {
		params.Set("startHistoryId", fmt.Sprintf("%v", v))
	}
	if v, ok := c.opt_["fields"]; ok {
		params.Set("fields", fmt.Sprintf("%v", v))
	}
	urls := googleapi.ResolveRelative(c.s.BasePath, "{userId}/history")
	urls += "?" + params.Encode()
	req, _ := http.NewRequest("GET", urls, body)
	googleapi.Expand(req.URL, map[string]string{
		"userId": c.userId,
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
	var ret *ListHistoryResponse
	if err := json.NewDecoder(res.Body).Decode(&ret); err != nil {
		return nil, err
	}
	return ret, nil
	// {
	//   "description": "Lists the history of all changes to the given mailbox. History results are returned in chronological order (increasing historyId).",
	//   "httpMethod": "GET",
	//   "id": "gmail.users.history.list",
	//   "parameterOrder": [
	//     "userId"
	//   ],
	//   "parameters": {
	//     "labelId": {
	//       "description": "Only return messages with a label matching the ID.",
	//       "location": "query",
	//       "type": "string"
	//     },
	//     "maxResults": {
	//       "default": "100",
	//       "description": "The maximum number of history records to return.",
	//       "format": "uint32",
	//       "location": "query",
	//       "type": "integer"
	//     },
	//     "pageToken": {
	//       "description": "Page token to retrieve a specific page of results in the list.",
	//       "location": "query",
	//       "type": "string"
	//     },
	//     "startHistoryId": {
	//       "description": "Required. Returns history records after the specified startHistoryId. The supplied startHistoryId should be obtained from the historyId of a message, thread, or previous list response. History IDs increase chronologically but are not contiguous with random gaps in between valid IDs. Supplying an invalid or out of date startHistoryId typically returns an HTTP 404 error code. A historyId is typically valid for at least a week, but in some circumstances may be valid for only a few hours. If you receive an HTTP 404 error response, your application should perform a full sync. If you receive no nextPageToken in the response, there are no updates to retrieve and you can store the returned historyId for a future request.",
	//       "format": "uint64",
	//       "location": "query",
	//       "type": "string"
	//     },
	//     "userId": {
	//       "default": "me",
	//       "description": "The user's email address. The special value me can be used to indicate the authenticated user.",
	//       "location": "path",
	//       "required": true,
	//       "type": "string"
	//     }
	//   },
	//   "path": "{userId}/history",
	//   "response": {
	//     "$ref": "ListHistoryResponse"
	//   },
	//   "scopes": [
	//     "https://mail.google.com/",
	//     "https://www.googleapis.com/auth/gmail.modify",
	//     "https://www.googleapis.com/auth/gmail.readonly"
	//   ]
	// }

}

// method id "gmail.users.labels.create":

type UsersLabelsCreateCall struct {
	s      *Service
	userId string
	label  *Label
	opt_   map[string]interface{}
}

// Create: Creates a new label.
func (r *UsersLabelsService) Create(userId string, label *Label) *UsersLabelsCreateCall {
	c := &UsersLabelsCreateCall{s: r.s, opt_: make(map[string]interface{})}
	c.userId = userId
	c.label = label
	return c
}

// Fields allows partial responses to be retrieved.
// See https://developers.google.com/gdata/docs/2.0/basics#PartialResponse
// for more information.
func (c *UsersLabelsCreateCall) Fields(s ...googleapi.Field) *UsersLabelsCreateCall {
	c.opt_["fields"] = googleapi.CombineFields(s)
	return c
}

func (c *UsersLabelsCreateCall) Do() (*Label, error) {
	var body io.Reader = nil
	body, err := googleapi.WithoutDataWrapper.JSONReader(c.label)
	if err != nil {
		return nil, err
	}
	ctype := "application/json"
	params := make(url.Values)
	params.Set("alt", "json")
	if v, ok := c.opt_["fields"]; ok {
		params.Set("fields", fmt.Sprintf("%v", v))
	}
	urls := googleapi.ResolveRelative(c.s.BasePath, "{userId}/labels")
	urls += "?" + params.Encode()
	req, _ := http.NewRequest("POST", urls, body)
	googleapi.Expand(req.URL, map[string]string{
		"userId": c.userId,
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
	var ret *Label
	if err := json.NewDecoder(res.Body).Decode(&ret); err != nil {
		return nil, err
	}
	return ret, nil
	// {
	//   "description": "Creates a new label.",
	//   "httpMethod": "POST",
	//   "id": "gmail.users.labels.create",
	//   "parameterOrder": [
	//     "userId"
	//   ],
	//   "parameters": {
	//     "userId": {
	//       "default": "me",
	//       "description": "The user's email address. The special value me can be used to indicate the authenticated user.",
	//       "location": "path",
	//       "required": true,
	//       "type": "string"
	//     }
	//   },
	//   "path": "{userId}/labels",
	//   "request": {
	//     "$ref": "Label"
	//   },
	//   "response": {
	//     "$ref": "Label"
	//   },
	//   "scopes": [
	//     "https://mail.google.com/",
	//     "https://www.googleapis.com/auth/gmail.modify"
	//   ]
	// }

}

// method id "gmail.users.labels.delete":

type UsersLabelsDeleteCall struct {
	s      *Service
	userId string
	id     string
	opt_   map[string]interface{}
}

// Delete: Immediately and permanently deletes the specified label and
// removes it from any messages and threads that it is applied to.
func (r *UsersLabelsService) Delete(userId string, id string) *UsersLabelsDeleteCall {
	c := &UsersLabelsDeleteCall{s: r.s, opt_: make(map[string]interface{})}
	c.userId = userId
	c.id = id
	return c
}

// Fields allows partial responses to be retrieved.
// See https://developers.google.com/gdata/docs/2.0/basics#PartialResponse
// for more information.
func (c *UsersLabelsDeleteCall) Fields(s ...googleapi.Field) *UsersLabelsDeleteCall {
	c.opt_["fields"] = googleapi.CombineFields(s)
	return c
}

func (c *UsersLabelsDeleteCall) Do() error {
	var body io.Reader = nil
	params := make(url.Values)
	params.Set("alt", "json")
	if v, ok := c.opt_["fields"]; ok {
		params.Set("fields", fmt.Sprintf("%v", v))
	}
	urls := googleapi.ResolveRelative(c.s.BasePath, "{userId}/labels/{id}")
	urls += "?" + params.Encode()
	req, _ := http.NewRequest("DELETE", urls, body)
	googleapi.Expand(req.URL, map[string]string{
		"userId": c.userId,
		"id":     c.id,
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
	//   "description": "Immediately and permanently deletes the specified label and removes it from any messages and threads that it is applied to.",
	//   "httpMethod": "DELETE",
	//   "id": "gmail.users.labels.delete",
	//   "parameterOrder": [
	//     "userId",
	//     "id"
	//   ],
	//   "parameters": {
	//     "id": {
	//       "description": "The ID of the label to delete.",
	//       "location": "path",
	//       "required": true,
	//       "type": "string"
	//     },
	//     "userId": {
	//       "default": "me",
	//       "description": "The user's email address. The special value me can be used to indicate the authenticated user.",
	//       "location": "path",
	//       "required": true,
	//       "type": "string"
	//     }
	//   },
	//   "path": "{userId}/labels/{id}",
	//   "scopes": [
	//     "https://mail.google.com/",
	//     "https://www.googleapis.com/auth/gmail.modify"
	//   ]
	// }

}

// method id "gmail.users.labels.get":

type UsersLabelsGetCall struct {
	s      *Service
	userId string
	id     string
	opt_   map[string]interface{}
}

// Get: Gets the specified label.
func (r *UsersLabelsService) Get(userId string, id string) *UsersLabelsGetCall {
	c := &UsersLabelsGetCall{s: r.s, opt_: make(map[string]interface{})}
	c.userId = userId
	c.id = id
	return c
}

// Fields allows partial responses to be retrieved.
// See https://developers.google.com/gdata/docs/2.0/basics#PartialResponse
// for more information.
func (c *UsersLabelsGetCall) Fields(s ...googleapi.Field) *UsersLabelsGetCall {
	c.opt_["fields"] = googleapi.CombineFields(s)
	return c
}

func (c *UsersLabelsGetCall) Do() (*Label, error) {
	var body io.Reader = nil
	params := make(url.Values)
	params.Set("alt", "json")
	if v, ok := c.opt_["fields"]; ok {
		params.Set("fields", fmt.Sprintf("%v", v))
	}
	urls := googleapi.ResolveRelative(c.s.BasePath, "{userId}/labels/{id}")
	urls += "?" + params.Encode()
	req, _ := http.NewRequest("GET", urls, body)
	googleapi.Expand(req.URL, map[string]string{
		"userId": c.userId,
		"id":     c.id,
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
	var ret *Label
	if err := json.NewDecoder(res.Body).Decode(&ret); err != nil {
		return nil, err
	}
	return ret, nil
	// {
	//   "description": "Gets the specified label.",
	//   "httpMethod": "GET",
	//   "id": "gmail.users.labels.get",
	//   "parameterOrder": [
	//     "userId",
	//     "id"
	//   ],
	//   "parameters": {
	//     "id": {
	//       "description": "The ID of the label to retrieve.",
	//       "location": "path",
	//       "required": true,
	//       "type": "string"
	//     },
	//     "userId": {
	//       "default": "me",
	//       "description": "The user's email address. The special value me can be used to indicate the authenticated user.",
	//       "location": "path",
	//       "required": true,
	//       "type": "string"
	//     }
	//   },
	//   "path": "{userId}/labels/{id}",
	//   "response": {
	//     "$ref": "Label"
	//   },
	//   "scopes": [
	//     "https://mail.google.com/",
	//     "https://www.googleapis.com/auth/gmail.modify",
	//     "https://www.googleapis.com/auth/gmail.readonly"
	//   ]
	// }

}

// method id "gmail.users.labels.list":

type UsersLabelsListCall struct {
	s      *Service
	userId string
	opt_   map[string]interface{}
}

// List: Lists all labels in the user's mailbox.
func (r *UsersLabelsService) List(userId string) *UsersLabelsListCall {
	c := &UsersLabelsListCall{s: r.s, opt_: make(map[string]interface{})}
	c.userId = userId
	return c
}

// Fields allows partial responses to be retrieved.
// See https://developers.google.com/gdata/docs/2.0/basics#PartialResponse
// for more information.
func (c *UsersLabelsListCall) Fields(s ...googleapi.Field) *UsersLabelsListCall {
	c.opt_["fields"] = googleapi.CombineFields(s)
	return c
}

func (c *UsersLabelsListCall) Do() (*ListLabelsResponse, error) {
	var body io.Reader = nil
	params := make(url.Values)
	params.Set("alt", "json")
	if v, ok := c.opt_["fields"]; ok {
		params.Set("fields", fmt.Sprintf("%v", v))
	}
	urls := googleapi.ResolveRelative(c.s.BasePath, "{userId}/labels")
	urls += "?" + params.Encode()
	req, _ := http.NewRequest("GET", urls, body)
	googleapi.Expand(req.URL, map[string]string{
		"userId": c.userId,
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
	var ret *ListLabelsResponse
	if err := json.NewDecoder(res.Body).Decode(&ret); err != nil {
		return nil, err
	}
	return ret, nil
	// {
	//   "description": "Lists all labels in the user's mailbox.",
	//   "httpMethod": "GET",
	//   "id": "gmail.users.labels.list",
	//   "parameterOrder": [
	//     "userId"
	//   ],
	//   "parameters": {
	//     "userId": {
	//       "default": "me",
	//       "description": "The user's email address. The special value me can be used to indicate the authenticated user.",
	//       "location": "path",
	//       "required": true,
	//       "type": "string"
	//     }
	//   },
	//   "path": "{userId}/labels",
	//   "response": {
	//     "$ref": "ListLabelsResponse"
	//   },
	//   "scopes": [
	//     "https://mail.google.com/",
	//     "https://www.googleapis.com/auth/gmail.modify",
	//     "https://www.googleapis.com/auth/gmail.readonly"
	//   ]
	// }

}

// method id "gmail.users.labels.patch":

type UsersLabelsPatchCall struct {
	s      *Service
	userId string
	id     string
	label  *Label
	opt_   map[string]interface{}
}

// Patch: Updates the specified label. This method supports patch
// semantics.
func (r *UsersLabelsService) Patch(userId string, id string, label *Label) *UsersLabelsPatchCall {
	c := &UsersLabelsPatchCall{s: r.s, opt_: make(map[string]interface{})}
	c.userId = userId
	c.id = id
	c.label = label
	return c
}

// Fields allows partial responses to be retrieved.
// See https://developers.google.com/gdata/docs/2.0/basics#PartialResponse
// for more information.
func (c *UsersLabelsPatchCall) Fields(s ...googleapi.Field) *UsersLabelsPatchCall {
	c.opt_["fields"] = googleapi.CombineFields(s)
	return c
}

func (c *UsersLabelsPatchCall) Do() (*Label, error) {
	var body io.Reader = nil
	body, err := googleapi.WithoutDataWrapper.JSONReader(c.label)
	if err != nil {
		return nil, err
	}
	ctype := "application/json"
	params := make(url.Values)
	params.Set("alt", "json")
	if v, ok := c.opt_["fields"]; ok {
		params.Set("fields", fmt.Sprintf("%v", v))
	}
	urls := googleapi.ResolveRelative(c.s.BasePath, "{userId}/labels/{id}")
	urls += "?" + params.Encode()
	req, _ := http.NewRequest("PATCH", urls, body)
	googleapi.Expand(req.URL, map[string]string{
		"userId": c.userId,
		"id":     c.id,
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
	var ret *Label
	if err := json.NewDecoder(res.Body).Decode(&ret); err != nil {
		return nil, err
	}
	return ret, nil
	// {
	//   "description": "Updates the specified label. This method supports patch semantics.",
	//   "httpMethod": "PATCH",
	//   "id": "gmail.users.labels.patch",
	//   "parameterOrder": [
	//     "userId",
	//     "id"
	//   ],
	//   "parameters": {
	//     "id": {
	//       "description": "The ID of the label to update.",
	//       "location": "path",
	//       "required": true,
	//       "type": "string"
	//     },
	//     "userId": {
	//       "default": "me",
	//       "description": "The user's email address. The special value me can be used to indicate the authenticated user.",
	//       "location": "path",
	//       "required": true,
	//       "type": "string"
	//     }
	//   },
	//   "path": "{userId}/labels/{id}",
	//   "request": {
	//     "$ref": "Label"
	//   },
	//   "response": {
	//     "$ref": "Label"
	//   },
	//   "scopes": [
	//     "https://mail.google.com/",
	//     "https://www.googleapis.com/auth/gmail.modify"
	//   ]
	// }

}

// method id "gmail.users.labels.update":

type UsersLabelsUpdateCall struct {
	s      *Service
	userId string
	id     string
	label  *Label
	opt_   map[string]interface{}
}

// Update: Updates the specified label.
func (r *UsersLabelsService) Update(userId string, id string, label *Label) *UsersLabelsUpdateCall {
	c := &UsersLabelsUpdateCall{s: r.s, opt_: make(map[string]interface{})}
	c.userId = userId
	c.id = id
	c.label = label
	return c
}

// Fields allows partial responses to be retrieved.
// See https://developers.google.com/gdata/docs/2.0/basics#PartialResponse
// for more information.
func (c *UsersLabelsUpdateCall) Fields(s ...googleapi.Field) *UsersLabelsUpdateCall {
	c.opt_["fields"] = googleapi.CombineFields(s)
	return c
}

func (c *UsersLabelsUpdateCall) Do() (*Label, error) {
	var body io.Reader = nil
	body, err := googleapi.WithoutDataWrapper.JSONReader(c.label)
	if err != nil {
		return nil, err
	}
	ctype := "application/json"
	params := make(url.Values)
	params.Set("alt", "json")
	if v, ok := c.opt_["fields"]; ok {
		params.Set("fields", fmt.Sprintf("%v", v))
	}
	urls := googleapi.ResolveRelative(c.s.BasePath, "{userId}/labels/{id}")
	urls += "?" + params.Encode()
	req, _ := http.NewRequest("PUT", urls, body)
	googleapi.Expand(req.URL, map[string]string{
		"userId": c.userId,
		"id":     c.id,
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
	var ret *Label
	if err := json.NewDecoder(res.Body).Decode(&ret); err != nil {
		return nil, err
	}
	return ret, nil
	// {
	//   "description": "Updates the specified label.",
	//   "httpMethod": "PUT",
	//   "id": "gmail.users.labels.update",
	//   "parameterOrder": [
	//     "userId",
	//     "id"
	//   ],
	//   "parameters": {
	//     "id": {
	//       "description": "The ID of the label to update.",
	//       "location": "path",
	//       "required": true,
	//       "type": "string"
	//     },
	//     "userId": {
	//       "default": "me",
	//       "description": "The user's email address. The special value me can be used to indicate the authenticated user.",
	//       "location": "path",
	//       "required": true,
	//       "type": "string"
	//     }
	//   },
	//   "path": "{userId}/labels/{id}",
	//   "request": {
	//     "$ref": "Label"
	//   },
	//   "response": {
	//     "$ref": "Label"
	//   },
	//   "scopes": [
	//     "https://mail.google.com/",
	//     "https://www.googleapis.com/auth/gmail.modify"
	//   ]
	// }

}

// method id "gmail.users.messages.delete":

type UsersMessagesDeleteCall struct {
	s      *Service
	userId string
	id     string
	opt_   map[string]interface{}
}

// Delete: Immediately and permanently deletes the specified message.
// This operation cannot be undone. Prefer messages.trash instead.
func (r *UsersMessagesService) Delete(userId string, id string) *UsersMessagesDeleteCall {
	c := &UsersMessagesDeleteCall{s: r.s, opt_: make(map[string]interface{})}
	c.userId = userId
	c.id = id
	return c
}

// Fields allows partial responses to be retrieved.
// See https://developers.google.com/gdata/docs/2.0/basics#PartialResponse
// for more information.
func (c *UsersMessagesDeleteCall) Fields(s ...googleapi.Field) *UsersMessagesDeleteCall {
	c.opt_["fields"] = googleapi.CombineFields(s)
	return c
}

func (c *UsersMessagesDeleteCall) Do() error {
	var body io.Reader = nil
	params := make(url.Values)
	params.Set("alt", "json")
	if v, ok := c.opt_["fields"]; ok {
		params.Set("fields", fmt.Sprintf("%v", v))
	}
	urls := googleapi.ResolveRelative(c.s.BasePath, "{userId}/messages/{id}")
	urls += "?" + params.Encode()
	req, _ := http.NewRequest("DELETE", urls, body)
	googleapi.Expand(req.URL, map[string]string{
		"userId": c.userId,
		"id":     c.id,
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
	//   "description": "Immediately and permanently deletes the specified message. This operation cannot be undone. Prefer messages.trash instead.",
	//   "httpMethod": "DELETE",
	//   "id": "gmail.users.messages.delete",
	//   "parameterOrder": [
	//     "userId",
	//     "id"
	//   ],
	//   "parameters": {
	//     "id": {
	//       "description": "The ID of the message to delete.",
	//       "location": "path",
	//       "required": true,
	//       "type": "string"
	//     },
	//     "userId": {
	//       "default": "me",
	//       "description": "The user's email address. The special value me can be used to indicate the authenticated user.",
	//       "location": "path",
	//       "required": true,
	//       "type": "string"
	//     }
	//   },
	//   "path": "{userId}/messages/{id}",
	//   "scopes": [
	//     "https://mail.google.com/"
	//   ]
	// }

}

// method id "gmail.users.messages.get":

type UsersMessagesGetCall struct {
	s      *Service
	userId string
	id     string
	opt_   map[string]interface{}
}

// Get: Gets the specified message.
func (r *UsersMessagesService) Get(userId string, id string) *UsersMessagesGetCall {
	c := &UsersMessagesGetCall{s: r.s, opt_: make(map[string]interface{})}
	c.userId = userId
	c.id = id
	return c
}

// Format sets the optional parameter "format": The format to return the
// message in.
func (c *UsersMessagesGetCall) Format(format string) *UsersMessagesGetCall {
	c.opt_["format"] = format
	return c
}

// MetadataHeaders sets the optional parameter "metadataHeaders": When
// given and format is METADATA, only include headers specified.
func (c *UsersMessagesGetCall) MetadataHeaders(metadataHeaders string) *UsersMessagesGetCall {
	c.opt_["metadataHeaders"] = metadataHeaders
	return c
}

// Fields allows partial responses to be retrieved.
// See https://developers.google.com/gdata/docs/2.0/basics#PartialResponse
// for more information.
func (c *UsersMessagesGetCall) Fields(s ...googleapi.Field) *UsersMessagesGetCall {
	c.opt_["fields"] = googleapi.CombineFields(s)
	return c
}

func (c *UsersMessagesGetCall) Do() (*Message, error) {
	var body io.Reader = nil
	params := make(url.Values)
	params.Set("alt", "json")
	if v, ok := c.opt_["format"]; ok {
		params.Set("format", fmt.Sprintf("%v", v))
	}
	if v, ok := c.opt_["metadataHeaders"]; ok {
		params.Set("metadataHeaders", fmt.Sprintf("%v", v))
	}
	if v, ok := c.opt_["fields"]; ok {
		params.Set("fields", fmt.Sprintf("%v", v))
	}
	urls := googleapi.ResolveRelative(c.s.BasePath, "{userId}/messages/{id}")
	urls += "?" + params.Encode()
	req, _ := http.NewRequest("GET", urls, body)
	googleapi.Expand(req.URL, map[string]string{
		"userId": c.userId,
		"id":     c.id,
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
	var ret *Message
	if err := json.NewDecoder(res.Body).Decode(&ret); err != nil {
		return nil, err
	}
	return ret, nil
	// {
	//   "description": "Gets the specified message.",
	//   "httpMethod": "GET",
	//   "id": "gmail.users.messages.get",
	//   "parameterOrder": [
	//     "userId",
	//     "id"
	//   ],
	//   "parameters": {
	//     "format": {
	//       "default": "full",
	//       "description": "The format to return the message in.",
	//       "enum": [
	//         "full",
	//         "metadata",
	//         "minimal",
	//         "raw"
	//       ],
	//       "enumDescriptions": [
	//         "",
	//         "",
	//         "",
	//         ""
	//       ],
	//       "location": "query",
	//       "type": "string"
	//     },
	//     "id": {
	//       "description": "The ID of the message to retrieve.",
	//       "location": "path",
	//       "required": true,
	//       "type": "string"
	//     },
	//     "metadataHeaders": {
	//       "description": "When given and format is METADATA, only include headers specified.",
	//       "location": "query",
	//       "repeated": true,
	//       "type": "string"
	//     },
	//     "userId": {
	//       "default": "me",
	//       "description": "The user's email address. The special value me can be used to indicate the authenticated user.",
	//       "location": "path",
	//       "required": true,
	//       "type": "string"
	//     }
	//   },
	//   "path": "{userId}/messages/{id}",
	//   "response": {
	//     "$ref": "Message"
	//   },
	//   "scopes": [
	//     "https://mail.google.com/",
	//     "https://www.googleapis.com/auth/gmail.modify",
	//     "https://www.googleapis.com/auth/gmail.readonly"
	//   ]
	// }

}

// method id "gmail.users.messages.import":

type UsersMessagesImportCall struct {
	s       *Service
	userId  string
	message *Message
	opt_    map[string]interface{}
	media_  io.Reader
}

// Import: Imports a message into only this user's mailbox, with
// standard email delivery scanning and classification similar to
// receiving via SMTP. Does not send a message.
func (r *UsersMessagesService) Import(userId string, message *Message) *UsersMessagesImportCall {
	c := &UsersMessagesImportCall{s: r.s, opt_: make(map[string]interface{})}
	c.userId = userId
	c.message = message
	return c
}

// InternalDateSource sets the optional parameter "internalDateSource":
// Source for Gmail's internal date of the message.
func (c *UsersMessagesImportCall) InternalDateSource(internalDateSource string) *UsersMessagesImportCall {
	c.opt_["internalDateSource"] = internalDateSource
	return c
}
func (c *UsersMessagesImportCall) Media(r io.Reader) *UsersMessagesImportCall {
	c.media_ = r
	return c
}

// Fields allows partial responses to be retrieved.
// See https://developers.google.com/gdata/docs/2.0/basics#PartialResponse
// for more information.
func (c *UsersMessagesImportCall) Fields(s ...googleapi.Field) *UsersMessagesImportCall {
	c.opt_["fields"] = googleapi.CombineFields(s)
	return c
}

func (c *UsersMessagesImportCall) Do() (*Message, error) {
	var body io.Reader = nil
	body, err := googleapi.WithoutDataWrapper.JSONReader(c.message)
	if err != nil {
		return nil, err
	}
	ctype := "application/json"
	params := make(url.Values)
	params.Set("alt", "json")
	if v, ok := c.opt_["internalDateSource"]; ok {
		params.Set("internalDateSource", fmt.Sprintf("%v", v))
	}
	if v, ok := c.opt_["fields"]; ok {
		params.Set("fields", fmt.Sprintf("%v", v))
	}
	urls := googleapi.ResolveRelative(c.s.BasePath, "{userId}/messages/import")
	if c.media_ != nil {
		urls = strings.Replace(urls, "https://www.googleapis.com/", "https://www.googleapis.com/upload/", 1)
		params.Set("uploadType", "multipart")
	}
	urls += "?" + params.Encode()
	contentLength_, hasMedia_ := googleapi.ConditionallyIncludeMedia(c.media_, &body, &ctype)
	req, _ := http.NewRequest("POST", urls, body)
	googleapi.Expand(req.URL, map[string]string{
		"userId": c.userId,
	})
	if hasMedia_ {
		req.ContentLength = contentLength_
	}
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
	var ret *Message
	if err := json.NewDecoder(res.Body).Decode(&ret); err != nil {
		return nil, err
	}
	return ret, nil
	// {
	//   "description": "Imports a message into only this user's mailbox, with standard email delivery scanning and classification similar to receiving via SMTP. Does not send a message.",
	//   "httpMethod": "POST",
	//   "id": "gmail.users.messages.import",
	//   "mediaUpload": {
	//     "accept": [
	//       "message/rfc822"
	//     ],
	//     "maxSize": "35MB",
	//     "protocols": {
	//       "resumable": {
	//         "multipart": true,
	//         "path": "/resumable/upload/gmail/v1/users/{userId}/messages/import"
	//       },
	//       "simple": {
	//         "multipart": true,
	//         "path": "/upload/gmail/v1/users/{userId}/messages/import"
	//       }
	//     }
	//   },
	//   "parameterOrder": [
	//     "userId"
	//   ],
	//   "parameters": {
	//     "internalDateSource": {
	//       "default": "dateHeader",
	//       "description": "Source for Gmail's internal date of the message.",
	//       "enum": [
	//         "dateHeader",
	//         "receivedTime"
	//       ],
	//       "enumDescriptions": [
	//         "",
	//         ""
	//       ],
	//       "location": "query",
	//       "type": "string"
	//     },
	//     "userId": {
	//       "default": "me",
	//       "description": "The user's email address. The special value me can be used to indicate the authenticated user.",
	//       "location": "path",
	//       "required": true,
	//       "type": "string"
	//     }
	//   },
	//   "path": "{userId}/messages/import",
	//   "request": {
	//     "$ref": "Message"
	//   },
	//   "response": {
	//     "$ref": "Message"
	//   },
	//   "scopes": [
	//     "https://mail.google.com/",
	//     "https://www.googleapis.com/auth/gmail.modify"
	//   ],
	//   "supportsMediaUpload": true
	// }

}

// method id "gmail.users.messages.insert":

type UsersMessagesInsertCall struct {
	s       *Service
	userId  string
	message *Message
	opt_    map[string]interface{}
	media_  io.Reader
}

// Insert: Directly inserts a message into only this user's mailbox
// similar to IMAP APPEND, bypassing most scanning and classification.
// Does not send a message.
func (r *UsersMessagesService) Insert(userId string, message *Message) *UsersMessagesInsertCall {
	c := &UsersMessagesInsertCall{s: r.s, opt_: make(map[string]interface{})}
	c.userId = userId
	c.message = message
	return c
}

// InternalDateSource sets the optional parameter "internalDateSource":
// Source for Gmail's internal date of the message.
func (c *UsersMessagesInsertCall) InternalDateSource(internalDateSource string) *UsersMessagesInsertCall {
	c.opt_["internalDateSource"] = internalDateSource
	return c
}
func (c *UsersMessagesInsertCall) Media(r io.Reader) *UsersMessagesInsertCall {
	c.media_ = r
	return c
}

// Fields allows partial responses to be retrieved.
// See https://developers.google.com/gdata/docs/2.0/basics#PartialResponse
// for more information.
func (c *UsersMessagesInsertCall) Fields(s ...googleapi.Field) *UsersMessagesInsertCall {
	c.opt_["fields"] = googleapi.CombineFields(s)
	return c
}

func (c *UsersMessagesInsertCall) Do() (*Message, error) {
	var body io.Reader = nil
	body, err := googleapi.WithoutDataWrapper.JSONReader(c.message)
	if err != nil {
		return nil, err
	}
	ctype := "application/json"
	params := make(url.Values)
	params.Set("alt", "json")
	if v, ok := c.opt_["internalDateSource"]; ok {
		params.Set("internalDateSource", fmt.Sprintf("%v", v))
	}
	if v, ok := c.opt_["fields"]; ok {
		params.Set("fields", fmt.Sprintf("%v", v))
	}
	urls := googleapi.ResolveRelative(c.s.BasePath, "{userId}/messages")
	if c.media_ != nil {
		urls = strings.Replace(urls, "https://www.googleapis.com/", "https://www.googleapis.com/upload/", 1)
		params.Set("uploadType", "multipart")
	}
	urls += "?" + params.Encode()
	contentLength_, hasMedia_ := googleapi.ConditionallyIncludeMedia(c.media_, &body, &ctype)
	req, _ := http.NewRequest("POST", urls, body)
	googleapi.Expand(req.URL, map[string]string{
		"userId": c.userId,
	})
	if hasMedia_ {
		req.ContentLength = contentLength_
	}
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
	var ret *Message
	if err := json.NewDecoder(res.Body).Decode(&ret); err != nil {
		return nil, err
	}
	return ret, nil
	// {
	//   "description": "Directly inserts a message into only this user's mailbox similar to IMAP APPEND, bypassing most scanning and classification. Does not send a message.",
	//   "httpMethod": "POST",
	//   "id": "gmail.users.messages.insert",
	//   "mediaUpload": {
	//     "accept": [
	//       "message/rfc822"
	//     ],
	//     "maxSize": "35MB",
	//     "protocols": {
	//       "resumable": {
	//         "multipart": true,
	//         "path": "/resumable/upload/gmail/v1/users/{userId}/messages"
	//       },
	//       "simple": {
	//         "multipart": true,
	//         "path": "/upload/gmail/v1/users/{userId}/messages"
	//       }
	//     }
	//   },
	//   "parameterOrder": [
	//     "userId"
	//   ],
	//   "parameters": {
	//     "internalDateSource": {
	//       "default": "receivedTime",
	//       "description": "Source for Gmail's internal date of the message.",
	//       "enum": [
	//         "dateHeader",
	//         "receivedTime"
	//       ],
	//       "enumDescriptions": [
	//         "",
	//         ""
	//       ],
	//       "location": "query",
	//       "type": "string"
	//     },
	//     "userId": {
	//       "default": "me",
	//       "description": "The user's email address. The special value me can be used to indicate the authenticated user.",
	//       "location": "path",
	//       "required": true,
	//       "type": "string"
	//     }
	//   },
	//   "path": "{userId}/messages",
	//   "request": {
	//     "$ref": "Message"
	//   },
	//   "response": {
	//     "$ref": "Message"
	//   },
	//   "scopes": [
	//     "https://mail.google.com/",
	//     "https://www.googleapis.com/auth/gmail.modify"
	//   ],
	//   "supportsMediaUpload": true
	// }

}

// method id "gmail.users.messages.list":

type UsersMessagesListCall struct {
	s      *Service
	userId string
	opt_   map[string]interface{}
}

// List: Lists the messages in the user's mailbox.
func (r *UsersMessagesService) List(userId string) *UsersMessagesListCall {
	c := &UsersMessagesListCall{s: r.s, opt_: make(map[string]interface{})}
	c.userId = userId
	return c
}

// IncludeSpamTrash sets the optional parameter "includeSpamTrash":
// Include messages from SPAM and TRASH in the results.
func (c *UsersMessagesListCall) IncludeSpamTrash(includeSpamTrash bool) *UsersMessagesListCall {
	c.opt_["includeSpamTrash"] = includeSpamTrash
	return c
}

// LabelIds sets the optional parameter "labelIds": Only return messages
// with labels that match all of the specified label IDs.
func (c *UsersMessagesListCall) LabelIds(labelIds string) *UsersMessagesListCall {
	c.opt_["labelIds"] = labelIds
	return c
}

// MaxResults sets the optional parameter "maxResults": Maximum number
// of messages to return.
func (c *UsersMessagesListCall) MaxResults(maxResults int64) *UsersMessagesListCall {
	c.opt_["maxResults"] = maxResults
	return c
}

// PageToken sets the optional parameter "pageToken": Page token to
// retrieve a specific page of results in the list.
func (c *UsersMessagesListCall) PageToken(pageToken string) *UsersMessagesListCall {
	c.opt_["pageToken"] = pageToken
	return c
}

// Q sets the optional parameter "q": Only return messages matching the
// specified query. Supports the same query format as the Gmail search
// box. For example, "from:someuser@example.com rfc822msgid: is:unread".
func (c *UsersMessagesListCall) Q(q string) *UsersMessagesListCall {
	c.opt_["q"] = q
	return c
}

// Fields allows partial responses to be retrieved.
// See https://developers.google.com/gdata/docs/2.0/basics#PartialResponse
// for more information.
func (c *UsersMessagesListCall) Fields(s ...googleapi.Field) *UsersMessagesListCall {
	c.opt_["fields"] = googleapi.CombineFields(s)
	return c
}

func (c *UsersMessagesListCall) Do() (*ListMessagesResponse, error) {
	var body io.Reader = nil
	params := make(url.Values)
	params.Set("alt", "json")
	if v, ok := c.opt_["includeSpamTrash"]; ok {
		params.Set("includeSpamTrash", fmt.Sprintf("%v", v))
	}
	if v, ok := c.opt_["labelIds"]; ok {
		params.Set("labelIds", fmt.Sprintf("%v", v))
	}
	if v, ok := c.opt_["maxResults"]; ok {
		params.Set("maxResults", fmt.Sprintf("%v", v))
	}
	if v, ok := c.opt_["pageToken"]; ok {
		params.Set("pageToken", fmt.Sprintf("%v", v))
	}
	if v, ok := c.opt_["q"]; ok {
		params.Set("q", fmt.Sprintf("%v", v))
	}
	if v, ok := c.opt_["fields"]; ok {
		params.Set("fields", fmt.Sprintf("%v", v))
	}
	urls := googleapi.ResolveRelative(c.s.BasePath, "{userId}/messages")
	urls += "?" + params.Encode()
	req, _ := http.NewRequest("GET", urls, body)
	googleapi.Expand(req.URL, map[string]string{
		"userId": c.userId,
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
	var ret *ListMessagesResponse
	if err := json.NewDecoder(res.Body).Decode(&ret); err != nil {
		return nil, err
	}
	return ret, nil
	// {
	//   "description": "Lists the messages in the user's mailbox.",
	//   "httpMethod": "GET",
	//   "id": "gmail.users.messages.list",
	//   "parameterOrder": [
	//     "userId"
	//   ],
	//   "parameters": {
	//     "includeSpamTrash": {
	//       "default": "false",
	//       "description": "Include messages from SPAM and TRASH in the results.",
	//       "location": "query",
	//       "type": "boolean"
	//     },
	//     "labelIds": {
	//       "description": "Only return messages with labels that match all of the specified label IDs.",
	//       "location": "query",
	//       "repeated": true,
	//       "type": "string"
	//     },
	//     "maxResults": {
	//       "default": "100",
	//       "description": "Maximum number of messages to return.",
	//       "format": "uint32",
	//       "location": "query",
	//       "type": "integer"
	//     },
	//     "pageToken": {
	//       "description": "Page token to retrieve a specific page of results in the list.",
	//       "location": "query",
	//       "type": "string"
	//     },
	//     "q": {
	//       "description": "Only return messages matching the specified query. Supports the same query format as the Gmail search box. For example, \"from:someuser@example.com rfc822msgid: is:unread\".",
	//       "location": "query",
	//       "type": "string"
	//     },
	//     "userId": {
	//       "default": "me",
	//       "description": "The user's email address. The special value me can be used to indicate the authenticated user.",
	//       "location": "path",
	//       "required": true,
	//       "type": "string"
	//     }
	//   },
	//   "path": "{userId}/messages",
	//   "response": {
	//     "$ref": "ListMessagesResponse"
	//   },
	//   "scopes": [
	//     "https://mail.google.com/",
	//     "https://www.googleapis.com/auth/gmail.modify",
	//     "https://www.googleapis.com/auth/gmail.readonly"
	//   ]
	// }

}

// method id "gmail.users.messages.modify":

type UsersMessagesModifyCall struct {
	s                    *Service
	userId               string
	id                   string
	modifymessagerequest *ModifyMessageRequest
	opt_                 map[string]interface{}
}

// Modify: Modifies the labels on the specified message.
func (r *UsersMessagesService) Modify(userId string, id string, modifymessagerequest *ModifyMessageRequest) *UsersMessagesModifyCall {
	c := &UsersMessagesModifyCall{s: r.s, opt_: make(map[string]interface{})}
	c.userId = userId
	c.id = id
	c.modifymessagerequest = modifymessagerequest
	return c
}

// Fields allows partial responses to be retrieved.
// See https://developers.google.com/gdata/docs/2.0/basics#PartialResponse
// for more information.
func (c *UsersMessagesModifyCall) Fields(s ...googleapi.Field) *UsersMessagesModifyCall {
	c.opt_["fields"] = googleapi.CombineFields(s)
	return c
}

func (c *UsersMessagesModifyCall) Do() (*Message, error) {
	var body io.Reader = nil
	body, err := googleapi.WithoutDataWrapper.JSONReader(c.modifymessagerequest)
	if err != nil {
		return nil, err
	}
	ctype := "application/json"
	params := make(url.Values)
	params.Set("alt", "json")
	if v, ok := c.opt_["fields"]; ok {
		params.Set("fields", fmt.Sprintf("%v", v))
	}
	urls := googleapi.ResolveRelative(c.s.BasePath, "{userId}/messages/{id}/modify")
	urls += "?" + params.Encode()
	req, _ := http.NewRequest("POST", urls, body)
	googleapi.Expand(req.URL, map[string]string{
		"userId": c.userId,
		"id":     c.id,
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
	var ret *Message
	if err := json.NewDecoder(res.Body).Decode(&ret); err != nil {
		return nil, err
	}
	return ret, nil
	// {
	//   "description": "Modifies the labels on the specified message.",
	//   "httpMethod": "POST",
	//   "id": "gmail.users.messages.modify",
	//   "parameterOrder": [
	//     "userId",
	//     "id"
	//   ],
	//   "parameters": {
	//     "id": {
	//       "description": "The ID of the message to modify.",
	//       "location": "path",
	//       "required": true,
	//       "type": "string"
	//     },
	//     "userId": {
	//       "default": "me",
	//       "description": "The user's email address. The special value me can be used to indicate the authenticated user.",
	//       "location": "path",
	//       "required": true,
	//       "type": "string"
	//     }
	//   },
	//   "path": "{userId}/messages/{id}/modify",
	//   "request": {
	//     "$ref": "ModifyMessageRequest"
	//   },
	//   "response": {
	//     "$ref": "Message"
	//   },
	//   "scopes": [
	//     "https://mail.google.com/",
	//     "https://www.googleapis.com/auth/gmail.modify"
	//   ]
	// }

}

// method id "gmail.users.messages.send":

type UsersMessagesSendCall struct {
	s       *Service
	userId  string
	message *Message
	opt_    map[string]interface{}
	media_  io.Reader
}

// Send: Sends the specified message to the recipients in the To, Cc,
// and Bcc headers.
func (r *UsersMessagesService) Send(userId string, message *Message) *UsersMessagesSendCall {
	c := &UsersMessagesSendCall{s: r.s, opt_: make(map[string]interface{})}
	c.userId = userId
	c.message = message
	return c
}
func (c *UsersMessagesSendCall) Media(r io.Reader) *UsersMessagesSendCall {
	c.media_ = r
	return c
}

// Fields allows partial responses to be retrieved.
// See https://developers.google.com/gdata/docs/2.0/basics#PartialResponse
// for more information.
func (c *UsersMessagesSendCall) Fields(s ...googleapi.Field) *UsersMessagesSendCall {
	c.opt_["fields"] = googleapi.CombineFields(s)
	return c
}

func (c *UsersMessagesSendCall) Do() (*Message, error) {
	var body io.Reader = nil
	body, err := googleapi.WithoutDataWrapper.JSONReader(c.message)
	if err != nil {
		return nil, err
	}
	ctype := "application/json"
	params := make(url.Values)
	params.Set("alt", "json")
	if v, ok := c.opt_["fields"]; ok {
		params.Set("fields", fmt.Sprintf("%v", v))
	}
	urls := googleapi.ResolveRelative(c.s.BasePath, "{userId}/messages/send")
	if c.media_ != nil {
		urls = strings.Replace(urls, "https://www.googleapis.com/", "https://www.googleapis.com/upload/", 1)
		params.Set("uploadType", "multipart")
	}
	urls += "?" + params.Encode()
	contentLength_, hasMedia_ := googleapi.ConditionallyIncludeMedia(c.media_, &body, &ctype)
	req, _ := http.NewRequest("POST", urls, body)
	googleapi.Expand(req.URL, map[string]string{
		"userId": c.userId,
	})
	if hasMedia_ {
		req.ContentLength = contentLength_
	}
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
	var ret *Message
	if err := json.NewDecoder(res.Body).Decode(&ret); err != nil {
		return nil, err
	}
	return ret, nil
	// {
	//   "description": "Sends the specified message to the recipients in the To, Cc, and Bcc headers.",
	//   "httpMethod": "POST",
	//   "id": "gmail.users.messages.send",
	//   "mediaUpload": {
	//     "accept": [
	//       "message/rfc822"
	//     ],
	//     "maxSize": "35MB",
	//     "protocols": {
	//       "resumable": {
	//         "multipart": true,
	//         "path": "/resumable/upload/gmail/v1/users/{userId}/messages/send"
	//       },
	//       "simple": {
	//         "multipart": true,
	//         "path": "/upload/gmail/v1/users/{userId}/messages/send"
	//       }
	//     }
	//   },
	//   "parameterOrder": [
	//     "userId"
	//   ],
	//   "parameters": {
	//     "userId": {
	//       "default": "me",
	//       "description": "The user's email address. The special value me can be used to indicate the authenticated user.",
	//       "location": "path",
	//       "required": true,
	//       "type": "string"
	//     }
	//   },
	//   "path": "{userId}/messages/send",
	//   "request": {
	//     "$ref": "Message"
	//   },
	//   "response": {
	//     "$ref": "Message"
	//   },
	//   "scopes": [
	//     "https://mail.google.com/",
	//     "https://www.googleapis.com/auth/gmail.compose",
	//     "https://www.googleapis.com/auth/gmail.modify"
	//   ],
	//   "supportsMediaUpload": true
	// }

}

// method id "gmail.users.messages.trash":

type UsersMessagesTrashCall struct {
	s      *Service
	userId string
	id     string
	opt_   map[string]interface{}
}

// Trash: Moves the specified message to the trash.
func (r *UsersMessagesService) Trash(userId string, id string) *UsersMessagesTrashCall {
	c := &UsersMessagesTrashCall{s: r.s, opt_: make(map[string]interface{})}
	c.userId = userId
	c.id = id
	return c
}

// Fields allows partial responses to be retrieved.
// See https://developers.google.com/gdata/docs/2.0/basics#PartialResponse
// for more information.
func (c *UsersMessagesTrashCall) Fields(s ...googleapi.Field) *UsersMessagesTrashCall {
	c.opt_["fields"] = googleapi.CombineFields(s)
	return c
}

func (c *UsersMessagesTrashCall) Do() (*Message, error) {
	var body io.Reader = nil
	params := make(url.Values)
	params.Set("alt", "json")
	if v, ok := c.opt_["fields"]; ok {
		params.Set("fields", fmt.Sprintf("%v", v))
	}
	urls := googleapi.ResolveRelative(c.s.BasePath, "{userId}/messages/{id}/trash")
	urls += "?" + params.Encode()
	req, _ := http.NewRequest("POST", urls, body)
	googleapi.Expand(req.URL, map[string]string{
		"userId": c.userId,
		"id":     c.id,
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
	var ret *Message
	if err := json.NewDecoder(res.Body).Decode(&ret); err != nil {
		return nil, err
	}
	return ret, nil
	// {
	//   "description": "Moves the specified message to the trash.",
	//   "httpMethod": "POST",
	//   "id": "gmail.users.messages.trash",
	//   "parameterOrder": [
	//     "userId",
	//     "id"
	//   ],
	//   "parameters": {
	//     "id": {
	//       "description": "The ID of the message to Trash.",
	//       "location": "path",
	//       "required": true,
	//       "type": "string"
	//     },
	//     "userId": {
	//       "default": "me",
	//       "description": "The user's email address. The special value me can be used to indicate the authenticated user.",
	//       "location": "path",
	//       "required": true,
	//       "type": "string"
	//     }
	//   },
	//   "path": "{userId}/messages/{id}/trash",
	//   "response": {
	//     "$ref": "Message"
	//   },
	//   "scopes": [
	//     "https://mail.google.com/",
	//     "https://www.googleapis.com/auth/gmail.modify"
	//   ]
	// }

}

// method id "gmail.users.messages.untrash":

type UsersMessagesUntrashCall struct {
	s      *Service
	userId string
	id     string
	opt_   map[string]interface{}
}

// Untrash: Removes the specified message from the trash.
func (r *UsersMessagesService) Untrash(userId string, id string) *UsersMessagesUntrashCall {
	c := &UsersMessagesUntrashCall{s: r.s, opt_: make(map[string]interface{})}
	c.userId = userId
	c.id = id
	return c
}

// Fields allows partial responses to be retrieved.
// See https://developers.google.com/gdata/docs/2.0/basics#PartialResponse
// for more information.
func (c *UsersMessagesUntrashCall) Fields(s ...googleapi.Field) *UsersMessagesUntrashCall {
	c.opt_["fields"] = googleapi.CombineFields(s)
	return c
}

func (c *UsersMessagesUntrashCall) Do() (*Message, error) {
	var body io.Reader = nil
	params := make(url.Values)
	params.Set("alt", "json")
	if v, ok := c.opt_["fields"]; ok {
		params.Set("fields", fmt.Sprintf("%v", v))
	}
	urls := googleapi.ResolveRelative(c.s.BasePath, "{userId}/messages/{id}/untrash")
	urls += "?" + params.Encode()
	req, _ := http.NewRequest("POST", urls, body)
	googleapi.Expand(req.URL, map[string]string{
		"userId": c.userId,
		"id":     c.id,
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
	var ret *Message
	if err := json.NewDecoder(res.Body).Decode(&ret); err != nil {
		return nil, err
	}
	return ret, nil
	// {
	//   "description": "Removes the specified message from the trash.",
	//   "httpMethod": "POST",
	//   "id": "gmail.users.messages.untrash",
	//   "parameterOrder": [
	//     "userId",
	//     "id"
	//   ],
	//   "parameters": {
	//     "id": {
	//       "description": "The ID of the message to remove from Trash.",
	//       "location": "path",
	//       "required": true,
	//       "type": "string"
	//     },
	//     "userId": {
	//       "default": "me",
	//       "description": "The user's email address. The special value me can be used to indicate the authenticated user.",
	//       "location": "path",
	//       "required": true,
	//       "type": "string"
	//     }
	//   },
	//   "path": "{userId}/messages/{id}/untrash",
	//   "response": {
	//     "$ref": "Message"
	//   },
	//   "scopes": [
	//     "https://mail.google.com/",
	//     "https://www.googleapis.com/auth/gmail.modify"
	//   ]
	// }

}

// method id "gmail.users.messages.attachments.get":

type UsersMessagesAttachmentsGetCall struct {
	s         *Service
	userId    string
	messageId string
	id        string
	opt_      map[string]interface{}
}

// Get: Gets the specified message attachment.
func (r *UsersMessagesAttachmentsService) Get(userId string, messageId string, id string) *UsersMessagesAttachmentsGetCall {
	c := &UsersMessagesAttachmentsGetCall{s: r.s, opt_: make(map[string]interface{})}
	c.userId = userId
	c.messageId = messageId
	c.id = id
	return c
}

// Fields allows partial responses to be retrieved.
// See https://developers.google.com/gdata/docs/2.0/basics#PartialResponse
// for more information.
func (c *UsersMessagesAttachmentsGetCall) Fields(s ...googleapi.Field) *UsersMessagesAttachmentsGetCall {
	c.opt_["fields"] = googleapi.CombineFields(s)
	return c
}

func (c *UsersMessagesAttachmentsGetCall) Do() (*MessagePartBody, error) {
	var body io.Reader = nil
	params := make(url.Values)
	params.Set("alt", "json")
	if v, ok := c.opt_["fields"]; ok {
		params.Set("fields", fmt.Sprintf("%v", v))
	}
	urls := googleapi.ResolveRelative(c.s.BasePath, "{userId}/messages/{messageId}/attachments/{id}")
	urls += "?" + params.Encode()
	req, _ := http.NewRequest("GET", urls, body)
	googleapi.Expand(req.URL, map[string]string{
		"userId":    c.userId,
		"messageId": c.messageId,
		"id":        c.id,
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
	var ret *MessagePartBody
	if err := json.NewDecoder(res.Body).Decode(&ret); err != nil {
		return nil, err
	}
	return ret, nil
	// {
	//   "description": "Gets the specified message attachment.",
	//   "httpMethod": "GET",
	//   "id": "gmail.users.messages.attachments.get",
	//   "parameterOrder": [
	//     "userId",
	//     "messageId",
	//     "id"
	//   ],
	//   "parameters": {
	//     "id": {
	//       "description": "The ID of the attachment.",
	//       "location": "path",
	//       "required": true,
	//       "type": "string"
	//     },
	//     "messageId": {
	//       "description": "The ID of the message containing the attachment.",
	//       "location": "path",
	//       "required": true,
	//       "type": "string"
	//     },
	//     "userId": {
	//       "default": "me",
	//       "description": "The user's email address. The special value me can be used to indicate the authenticated user.",
	//       "location": "path",
	//       "required": true,
	//       "type": "string"
	//     }
	//   },
	//   "path": "{userId}/messages/{messageId}/attachments/{id}",
	//   "response": {
	//     "$ref": "MessagePartBody"
	//   },
	//   "scopes": [
	//     "https://mail.google.com/",
	//     "https://www.googleapis.com/auth/gmail.modify",
	//     "https://www.googleapis.com/auth/gmail.readonly"
	//   ]
	// }

}

// method id "gmail.users.threads.delete":

type UsersThreadsDeleteCall struct {
	s      *Service
	userId string
	id     string
	opt_   map[string]interface{}
}

// Delete: Immediately and permanently deletes the specified thread.
// This operation cannot be undone. Prefer threads.trash instead.
func (r *UsersThreadsService) Delete(userId string, id string) *UsersThreadsDeleteCall {
	c := &UsersThreadsDeleteCall{s: r.s, opt_: make(map[string]interface{})}
	c.userId = userId
	c.id = id
	return c
}

// Fields allows partial responses to be retrieved.
// See https://developers.google.com/gdata/docs/2.0/basics#PartialResponse
// for more information.
func (c *UsersThreadsDeleteCall) Fields(s ...googleapi.Field) *UsersThreadsDeleteCall {
	c.opt_["fields"] = googleapi.CombineFields(s)
	return c
}

func (c *UsersThreadsDeleteCall) Do() error {
	var body io.Reader = nil
	params := make(url.Values)
	params.Set("alt", "json")
	if v, ok := c.opt_["fields"]; ok {
		params.Set("fields", fmt.Sprintf("%v", v))
	}
	urls := googleapi.ResolveRelative(c.s.BasePath, "{userId}/threads/{id}")
	urls += "?" + params.Encode()
	req, _ := http.NewRequest("DELETE", urls, body)
	googleapi.Expand(req.URL, map[string]string{
		"userId": c.userId,
		"id":     c.id,
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
	//   "description": "Immediately and permanently deletes the specified thread. This operation cannot be undone. Prefer threads.trash instead.",
	//   "httpMethod": "DELETE",
	//   "id": "gmail.users.threads.delete",
	//   "parameterOrder": [
	//     "userId",
	//     "id"
	//   ],
	//   "parameters": {
	//     "id": {
	//       "description": "ID of the Thread to delete.",
	//       "location": "path",
	//       "required": true,
	//       "type": "string"
	//     },
	//     "userId": {
	//       "default": "me",
	//       "description": "The user's email address. The special value me can be used to indicate the authenticated user.",
	//       "location": "path",
	//       "required": true,
	//       "type": "string"
	//     }
	//   },
	//   "path": "{userId}/threads/{id}",
	//   "scopes": [
	//     "https://mail.google.com/"
	//   ]
	// }

}

// method id "gmail.users.threads.get":

type UsersThreadsGetCall struct {
	s      *Service
	userId string
	id     string
	opt_   map[string]interface{}
}

// Get: Gets the specified thread.
func (r *UsersThreadsService) Get(userId string, id string) *UsersThreadsGetCall {
	c := &UsersThreadsGetCall{s: r.s, opt_: make(map[string]interface{})}
	c.userId = userId
	c.id = id
	return c
}

// Format sets the optional parameter "format": The format to return the
// messages in.
func (c *UsersThreadsGetCall) Format(format string) *UsersThreadsGetCall {
	c.opt_["format"] = format
	return c
}

// MetadataHeaders sets the optional parameter "metadataHeaders": When
// given and format is METADATA, only include headers specified.
func (c *UsersThreadsGetCall) MetadataHeaders(metadataHeaders string) *UsersThreadsGetCall {
	c.opt_["metadataHeaders"] = metadataHeaders
	return c
}

// Fields allows partial responses to be retrieved.
// See https://developers.google.com/gdata/docs/2.0/basics#PartialResponse
// for more information.
func (c *UsersThreadsGetCall) Fields(s ...googleapi.Field) *UsersThreadsGetCall {
	c.opt_["fields"] = googleapi.CombineFields(s)
	return c
}

func (c *UsersThreadsGetCall) Do() (*Thread, error) {
	var body io.Reader = nil
	params := make(url.Values)
	params.Set("alt", "json")
	if v, ok := c.opt_["format"]; ok {
		params.Set("format", fmt.Sprintf("%v", v))
	}
	if v, ok := c.opt_["metadataHeaders"]; ok {
		params.Set("metadataHeaders", fmt.Sprintf("%v", v))
	}
	if v, ok := c.opt_["fields"]; ok {
		params.Set("fields", fmt.Sprintf("%v", v))
	}
	urls := googleapi.ResolveRelative(c.s.BasePath, "{userId}/threads/{id}")
	urls += "?" + params.Encode()
	req, _ := http.NewRequest("GET", urls, body)
	googleapi.Expand(req.URL, map[string]string{
		"userId": c.userId,
		"id":     c.id,
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
	var ret *Thread
	if err := json.NewDecoder(res.Body).Decode(&ret); err != nil {
		return nil, err
	}
	return ret, nil
	// {
	//   "description": "Gets the specified thread.",
	//   "httpMethod": "GET",
	//   "id": "gmail.users.threads.get",
	//   "parameterOrder": [
	//     "userId",
	//     "id"
	//   ],
	//   "parameters": {
	//     "format": {
	//       "default": "full",
	//       "description": "The format to return the messages in.",
	//       "enum": [
	//         "full",
	//         "metadata",
	//         "minimal"
	//       ],
	//       "enumDescriptions": [
	//         "",
	//         "",
	//         ""
	//       ],
	//       "location": "query",
	//       "type": "string"
	//     },
	//     "id": {
	//       "description": "The ID of the thread to retrieve.",
	//       "location": "path",
	//       "required": true,
	//       "type": "string"
	//     },
	//     "metadataHeaders": {
	//       "description": "When given and format is METADATA, only include headers specified.",
	//       "location": "query",
	//       "repeated": true,
	//       "type": "string"
	//     },
	//     "userId": {
	//       "default": "me",
	//       "description": "The user's email address. The special value me can be used to indicate the authenticated user.",
	//       "location": "path",
	//       "required": true,
	//       "type": "string"
	//     }
	//   },
	//   "path": "{userId}/threads/{id}",
	//   "response": {
	//     "$ref": "Thread"
	//   },
	//   "scopes": [
	//     "https://mail.google.com/",
	//     "https://www.googleapis.com/auth/gmail.modify",
	//     "https://www.googleapis.com/auth/gmail.readonly"
	//   ]
	// }

}

// method id "gmail.users.threads.list":

type UsersThreadsListCall struct {
	s      *Service
	userId string
	opt_   map[string]interface{}
}

// List: Lists the threads in the user's mailbox.
func (r *UsersThreadsService) List(userId string) *UsersThreadsListCall {
	c := &UsersThreadsListCall{s: r.s, opt_: make(map[string]interface{})}
	c.userId = userId
	return c
}

// IncludeSpamTrash sets the optional parameter "includeSpamTrash":
// Include threads from SPAM and TRASH in the results.
func (c *UsersThreadsListCall) IncludeSpamTrash(includeSpamTrash bool) *UsersThreadsListCall {
	c.opt_["includeSpamTrash"] = includeSpamTrash
	return c
}

// LabelIds sets the optional parameter "labelIds": Only return threads
// with labels that match all of the specified label IDs.
func (c *UsersThreadsListCall) LabelIds(labelIds string) *UsersThreadsListCall {
	c.opt_["labelIds"] = labelIds
	return c
}

// MaxResults sets the optional parameter "maxResults": Maximum number
// of threads to return.
func (c *UsersThreadsListCall) MaxResults(maxResults int64) *UsersThreadsListCall {
	c.opt_["maxResults"] = maxResults
	return c
}

// PageToken sets the optional parameter "pageToken": Page token to
// retrieve a specific page of results in the list.
func (c *UsersThreadsListCall) PageToken(pageToken string) *UsersThreadsListCall {
	c.opt_["pageToken"] = pageToken
	return c
}

// Q sets the optional parameter "q": Only return threads matching the
// specified query. Supports the same query format as the Gmail search
// box. For example, "from:someuser@example.com rfc822msgid: is:unread".
func (c *UsersThreadsListCall) Q(q string) *UsersThreadsListCall {
	c.opt_["q"] = q
	return c
}

// Fields allows partial responses to be retrieved.
// See https://developers.google.com/gdata/docs/2.0/basics#PartialResponse
// for more information.
func (c *UsersThreadsListCall) Fields(s ...googleapi.Field) *UsersThreadsListCall {
	c.opt_["fields"] = googleapi.CombineFields(s)
	return c
}

func (c *UsersThreadsListCall) Do() (*ListThreadsResponse, error) {
	var body io.Reader = nil
	params := make(url.Values)
	params.Set("alt", "json")
	if v, ok := c.opt_["includeSpamTrash"]; ok {
		params.Set("includeSpamTrash", fmt.Sprintf("%v", v))
	}
	if v, ok := c.opt_["labelIds"]; ok {
		params.Set("labelIds", fmt.Sprintf("%v", v))
	}
	if v, ok := c.opt_["maxResults"]; ok {
		params.Set("maxResults", fmt.Sprintf("%v", v))
	}
	if v, ok := c.opt_["pageToken"]; ok {
		params.Set("pageToken", fmt.Sprintf("%v", v))
	}
	if v, ok := c.opt_["q"]; ok {
		params.Set("q", fmt.Sprintf("%v", v))
	}
	if v, ok := c.opt_["fields"]; ok {
		params.Set("fields", fmt.Sprintf("%v", v))
	}
	urls := googleapi.ResolveRelative(c.s.BasePath, "{userId}/threads")
	urls += "?" + params.Encode()
	req, _ := http.NewRequest("GET", urls, body)
	googleapi.Expand(req.URL, map[string]string{
		"userId": c.userId,
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
	var ret *ListThreadsResponse
	if err := json.NewDecoder(res.Body).Decode(&ret); err != nil {
		return nil, err
	}
	return ret, nil
	// {
	//   "description": "Lists the threads in the user's mailbox.",
	//   "httpMethod": "GET",
	//   "id": "gmail.users.threads.list",
	//   "parameterOrder": [
	//     "userId"
	//   ],
	//   "parameters": {
	//     "includeSpamTrash": {
	//       "default": "false",
	//       "description": "Include threads from SPAM and TRASH in the results.",
	//       "location": "query",
	//       "type": "boolean"
	//     },
	//     "labelIds": {
	//       "description": "Only return threads with labels that match all of the specified label IDs.",
	//       "location": "query",
	//       "repeated": true,
	//       "type": "string"
	//     },
	//     "maxResults": {
	//       "default": "100",
	//       "description": "Maximum number of threads to return.",
	//       "format": "uint32",
	//       "location": "query",
	//       "type": "integer"
	//     },
	//     "pageToken": {
	//       "description": "Page token to retrieve a specific page of results in the list.",
	//       "location": "query",
	//       "type": "string"
	//     },
	//     "q": {
	//       "description": "Only return threads matching the specified query. Supports the same query format as the Gmail search box. For example, \"from:someuser@example.com rfc822msgid: is:unread\".",
	//       "location": "query",
	//       "type": "string"
	//     },
	//     "userId": {
	//       "default": "me",
	//       "description": "The user's email address. The special value me can be used to indicate the authenticated user.",
	//       "location": "path",
	//       "required": true,
	//       "type": "string"
	//     }
	//   },
	//   "path": "{userId}/threads",
	//   "response": {
	//     "$ref": "ListThreadsResponse"
	//   },
	//   "scopes": [
	//     "https://mail.google.com/",
	//     "https://www.googleapis.com/auth/gmail.modify",
	//     "https://www.googleapis.com/auth/gmail.readonly"
	//   ]
	// }

}

// method id "gmail.users.threads.modify":

type UsersThreadsModifyCall struct {
	s                   *Service
	userId              string
	id                  string
	modifythreadrequest *ModifyThreadRequest
	opt_                map[string]interface{}
}

// Modify: Modifies the labels applied to the thread. This applies to
// all messages in the thread.
func (r *UsersThreadsService) Modify(userId string, id string, modifythreadrequest *ModifyThreadRequest) *UsersThreadsModifyCall {
	c := &UsersThreadsModifyCall{s: r.s, opt_: make(map[string]interface{})}
	c.userId = userId
	c.id = id
	c.modifythreadrequest = modifythreadrequest
	return c
}

// Fields allows partial responses to be retrieved.
// See https://developers.google.com/gdata/docs/2.0/basics#PartialResponse
// for more information.
func (c *UsersThreadsModifyCall) Fields(s ...googleapi.Field) *UsersThreadsModifyCall {
	c.opt_["fields"] = googleapi.CombineFields(s)
	return c
}

func (c *UsersThreadsModifyCall) Do() (*Thread, error) {
	var body io.Reader = nil
	body, err := googleapi.WithoutDataWrapper.JSONReader(c.modifythreadrequest)
	if err != nil {
		return nil, err
	}
	ctype := "application/json"
	params := make(url.Values)
	params.Set("alt", "json")
	if v, ok := c.opt_["fields"]; ok {
		params.Set("fields", fmt.Sprintf("%v", v))
	}
	urls := googleapi.ResolveRelative(c.s.BasePath, "{userId}/threads/{id}/modify")
	urls += "?" + params.Encode()
	req, _ := http.NewRequest("POST", urls, body)
	googleapi.Expand(req.URL, map[string]string{
		"userId": c.userId,
		"id":     c.id,
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
	var ret *Thread
	if err := json.NewDecoder(res.Body).Decode(&ret); err != nil {
		return nil, err
	}
	return ret, nil
	// {
	//   "description": "Modifies the labels applied to the thread. This applies to all messages in the thread.",
	//   "httpMethod": "POST",
	//   "id": "gmail.users.threads.modify",
	//   "parameterOrder": [
	//     "userId",
	//     "id"
	//   ],
	//   "parameters": {
	//     "id": {
	//       "description": "The ID of the thread to modify.",
	//       "location": "path",
	//       "required": true,
	//       "type": "string"
	//     },
	//     "userId": {
	//       "default": "me",
	//       "description": "The user's email address. The special value me can be used to indicate the authenticated user.",
	//       "location": "path",
	//       "required": true,
	//       "type": "string"
	//     }
	//   },
	//   "path": "{userId}/threads/{id}/modify",
	//   "request": {
	//     "$ref": "ModifyThreadRequest"
	//   },
	//   "response": {
	//     "$ref": "Thread"
	//   },
	//   "scopes": [
	//     "https://mail.google.com/",
	//     "https://www.googleapis.com/auth/gmail.modify"
	//   ]
	// }

}

// method id "gmail.users.threads.trash":

type UsersThreadsTrashCall struct {
	s      *Service
	userId string
	id     string
	opt_   map[string]interface{}
}

// Trash: Moves the specified thread to the trash.
func (r *UsersThreadsService) Trash(userId string, id string) *UsersThreadsTrashCall {
	c := &UsersThreadsTrashCall{s: r.s, opt_: make(map[string]interface{})}
	c.userId = userId
	c.id = id
	return c
}

// Fields allows partial responses to be retrieved.
// See https://developers.google.com/gdata/docs/2.0/basics#PartialResponse
// for more information.
func (c *UsersThreadsTrashCall) Fields(s ...googleapi.Field) *UsersThreadsTrashCall {
	c.opt_["fields"] = googleapi.CombineFields(s)
	return c
}

func (c *UsersThreadsTrashCall) Do() (*Thread, error) {
	var body io.Reader = nil
	params := make(url.Values)
	params.Set("alt", "json")
	if v, ok := c.opt_["fields"]; ok {
		params.Set("fields", fmt.Sprintf("%v", v))
	}
	urls := googleapi.ResolveRelative(c.s.BasePath, "{userId}/threads/{id}/trash")
	urls += "?" + params.Encode()
	req, _ := http.NewRequest("POST", urls, body)
	googleapi.Expand(req.URL, map[string]string{
		"userId": c.userId,
		"id":     c.id,
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
	var ret *Thread
	if err := json.NewDecoder(res.Body).Decode(&ret); err != nil {
		return nil, err
	}
	return ret, nil
	// {
	//   "description": "Moves the specified thread to the trash.",
	//   "httpMethod": "POST",
	//   "id": "gmail.users.threads.trash",
	//   "parameterOrder": [
	//     "userId",
	//     "id"
	//   ],
	//   "parameters": {
	//     "id": {
	//       "description": "The ID of the thread to Trash.",
	//       "location": "path",
	//       "required": true,
	//       "type": "string"
	//     },
	//     "userId": {
	//       "default": "me",
	//       "description": "The user's email address. The special value me can be used to indicate the authenticated user.",
	//       "location": "path",
	//       "required": true,
	//       "type": "string"
	//     }
	//   },
	//   "path": "{userId}/threads/{id}/trash",
	//   "response": {
	//     "$ref": "Thread"
	//   },
	//   "scopes": [
	//     "https://mail.google.com/",
	//     "https://www.googleapis.com/auth/gmail.modify"
	//   ]
	// }

}

// method id "gmail.users.threads.untrash":

type UsersThreadsUntrashCall struct {
	s      *Service
	userId string
	id     string
	opt_   map[string]interface{}
}

// Untrash: Removes the specified thread from the trash.
func (r *UsersThreadsService) Untrash(userId string, id string) *UsersThreadsUntrashCall {
	c := &UsersThreadsUntrashCall{s: r.s, opt_: make(map[string]interface{})}
	c.userId = userId
	c.id = id
	return c
}

// Fields allows partial responses to be retrieved.
// See https://developers.google.com/gdata/docs/2.0/basics#PartialResponse
// for more information.
func (c *UsersThreadsUntrashCall) Fields(s ...googleapi.Field) *UsersThreadsUntrashCall {
	c.opt_["fields"] = googleapi.CombineFields(s)
	return c
}

func (c *UsersThreadsUntrashCall) Do() (*Thread, error) {
	var body io.Reader = nil
	params := make(url.Values)
	params.Set("alt", "json")
	if v, ok := c.opt_["fields"]; ok {
		params.Set("fields", fmt.Sprintf("%v", v))
	}
	urls := googleapi.ResolveRelative(c.s.BasePath, "{userId}/threads/{id}/untrash")
	urls += "?" + params.Encode()
	req, _ := http.NewRequest("POST", urls, body)
	googleapi.Expand(req.URL, map[string]string{
		"userId": c.userId,
		"id":     c.id,
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
	var ret *Thread
	if err := json.NewDecoder(res.Body).Decode(&ret); err != nil {
		return nil, err
	}
	return ret, nil
	// {
	//   "description": "Removes the specified thread from the trash.",
	//   "httpMethod": "POST",
	//   "id": "gmail.users.threads.untrash",
	//   "parameterOrder": [
	//     "userId",
	//     "id"
	//   ],
	//   "parameters": {
	//     "id": {
	//       "description": "The ID of the thread to remove from Trash.",
	//       "location": "path",
	//       "required": true,
	//       "type": "string"
	//     },
	//     "userId": {
	//       "default": "me",
	//       "description": "The user's email address. The special value me can be used to indicate the authenticated user.",
	//       "location": "path",
	//       "required": true,
	//       "type": "string"
	//     }
	//   },
	//   "path": "{userId}/threads/{id}/untrash",
	//   "response": {
	//     "$ref": "Thread"
	//   },
	//   "scopes": [
	//     "https://mail.google.com/",
	//     "https://www.googleapis.com/auth/gmail.modify"
	//   ]
	// }

}
