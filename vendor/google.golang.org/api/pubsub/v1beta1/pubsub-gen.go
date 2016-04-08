// Package pubsub provides access to the Cloud Pub/Sub API.
//
// See https://developers.google.com/pubsub/v1beta1
//
// Usage example:
//
//   import "google.golang.org/api/pubsub/v1beta1"
//   ...
//   pubsubService, err := pubsub.New(oauthHttpClient)
package pubsub

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

const apiId = "pubsub:v1beta1"
const apiName = "pubsub"
const apiVersion = "v1beta1"
const basePath = "https://www.googleapis.com/pubsub/v1beta1/"

// OAuth2 scopes used by this API.
const (
	// View and manage your data across Google Cloud Platform services
	CloudPlatformScope = "https://www.googleapis.com/auth/cloud-platform"

	// View and manage Pub/Sub topics and subscriptions
	PubsubScope = "https://www.googleapis.com/auth/pubsub"
)

func New(client *http.Client) (*Service, error) {
	if client == nil {
		return nil, errors.New("client is nil")
	}
	s := &Service{client: client, BasePath: basePath}
	s.Subscriptions = NewSubscriptionsService(s)
	s.Topics = NewTopicsService(s)
	return s, nil
}

type Service struct {
	client   *http.Client
	BasePath string // API endpoint base URL

	Subscriptions *SubscriptionsService

	Topics *TopicsService
}

func NewSubscriptionsService(s *Service) *SubscriptionsService {
	rs := &SubscriptionsService{s: s}
	return rs
}

type SubscriptionsService struct {
	s *Service
}

func NewTopicsService(s *Service) *TopicsService {
	rs := &TopicsService{s: s}
	return rs
}

type TopicsService struct {
	s *Service
}

type AcknowledgeRequest struct {
	// AckId: The Ack ID for the message being acknowledged. This was
	// returned by the Pub/Sub system in the Pull response.
	AckId []string `json:"ackId,omitempty"`

	// Subscription: The subscription whose message is being acknowledged.
	Subscription string `json:"subscription,omitempty"`
}

type Label struct {
	// Key: The key of a label is a syntactically valid URL (as per RFC
	// 1738) with the "scheme" and initial slashes omitted and with the
	// additional restrictions noted below. Each key should be globally
	// unique. The "host" portion is called the "namespace" and is not
	// necessarily resolvable to a network endpoint. Instead, the namespace
	// indicates what system or entity defines the semantics of the label.
	// Namespaces do not restrict the set of objects to which a label may be
	// associated.
	//
	// Keys are defined by the following grammar:
	//
	// key =
	// hostname "/" kpath kpath = ksegment *[ "/" ksegment ] ksegment =
	// alphadigit | *[ alphadigit | "-" | "_" | "." ]
	//
	// where "hostname" and
	// "alphadigit" are defined as in RFC 1738.
	//
	// Example key:
	// spanner.google.com/universe
	Key string `json:"key,omitempty"`

	// NumValue: An integer value.
	NumValue int64 `json:"numValue,omitempty,string"`

	// StrValue: A string value.
	StrValue string `json:"strValue,omitempty"`
}

type ListSubscriptionsResponse struct {
	// NextPageToken: If not empty, indicates that there are more
	// subscriptions that match the request and this value should be passed
	// to the next ListSubscriptionsRequest to continue.
	NextPageToken string `json:"nextPageToken,omitempty"`

	// Subscription: The subscriptions that match the request.
	Subscription []*Subscription `json:"subscription,omitempty"`
}

type ListTopicsResponse struct {
	// NextPageToken: If not empty, indicates that there are more topics
	// that match the request, and this value should be passed to the next
	// ListTopicsRequest to continue.
	NextPageToken string `json:"nextPageToken,omitempty"`

	// Topic: The resulting topics.
	Topic []*Topic `json:"topic,omitempty"`
}

type ModifyAckDeadlineRequest struct {
	// AckDeadlineSeconds: The new Ack deadline. Must be >= 0.
	AckDeadlineSeconds int64 `json:"ackDeadlineSeconds,omitempty"`

	// AckId: The Ack ID.
	AckId string `json:"ackId,omitempty"`

	// Subscription: The name of the subscription from which messages are
	// being pulled.
	Subscription string `json:"subscription,omitempty"`
}

type ModifyPushConfigRequest struct {
	// PushConfig: An empty push_config indicates that the Pub/Sub system
	// should pause pushing messages from the given subscription.
	PushConfig *PushConfig `json:"pushConfig,omitempty"`

	// Subscription: The name of the subscription.
	Subscription string `json:"subscription,omitempty"`
}

type PublishBatchRequest struct {
	// Messages: The messages to publish.
	Messages []*PubsubMessage `json:"messages,omitempty"`

	// Topic: The messages in the request will be published on this topic.
	Topic string `json:"topic,omitempty"`
}

type PublishBatchResponse struct {
	// MessageIds: The server-assigned ID of each published message, in the
	// same order as the messages in the request. IDs are guaranteed to be
	// unique within the topic.
	MessageIds []string `json:"messageIds,omitempty"`
}

type PublishRequest struct {
	// Message: The message to publish.
	Message *PubsubMessage `json:"message,omitempty"`

	// Topic: The message in the request will be published on this topic.
	Topic string `json:"topic,omitempty"`
}

type PubsubEvent struct {
	// Deleted: Indicates that this subscription has been deleted. (Note
	// that pull subscribers will always receive NOT_FOUND in response in
	// their pull request on the subscription, rather than seeing this
	// boolean.)
	Deleted bool `json:"deleted,omitempty"`

	// Message: A received message.
	Message *PubsubMessage `json:"message,omitempty"`

	// Subscription: The subscription that received the event.
	Subscription string `json:"subscription,omitempty"`

	// Truncated: Indicates that this subscription has been truncated.
	Truncated bool `json:"truncated,omitempty"`
}

type PubsubMessage struct {
	// Data: The message payload.
	Data string `json:"data,omitempty"`

	// Label: Optional list of labels for this message. Keys in this
	// collection must be unique.
	Label []*Label `json:"label,omitempty"`

	// MessageId: ID of this message assigned by the server at publication
	// time. Guaranteed to be unique within the topic. This value may be
	// read by a subscriber that receives a PubsubMessage via a Pull call or
	// a push delivery. It must not be populated by a publisher in a Publish
	// call.
	MessageId string `json:"messageId,omitempty"`
}

type PullBatchRequest struct {
	// MaxEvents: The maximum number of PubsubEvents returned for this
	// request. The Pub/Sub system may return fewer than the number of
	// events specified.
	MaxEvents int64 `json:"maxEvents,omitempty"`

	// ReturnImmediately: If this is specified as true the system will
	// respond immediately even if it is not able to return a message in the
	// Pull response. Otherwise the system is allowed to wait until at least
	// one message is available rather than returning no messages. The
	// client may cancel the request if it does not wish to wait any longer
	// for the response.
	ReturnImmediately bool `json:"returnImmediately,omitempty"`

	// Subscription: The subscription from which messages should be pulled.
	Subscription string `json:"subscription,omitempty"`
}

type PullBatchResponse struct {
	// PullResponses: Received Pub/Sub messages or status events. The
	// Pub/Sub system will return zero messages if there are no more
	// messages available in the backlog. The Pub/Sub system may return
	// fewer than the max_events requested even if there are more messages
	// available in the backlog.
	PullResponses []*PullResponse `json:"pullResponses,omitempty"`
}

type PullRequest struct {
	// ReturnImmediately: If this is specified as true the system will
	// respond immediately even if it is not able to return a message in the
	// Pull response. Otherwise the system is allowed to wait until at least
	// one message is available rather than returning FAILED_PRECONDITION.
	// The client may cancel the request if it does not wish to wait any
	// longer for the response.
	ReturnImmediately bool `json:"returnImmediately,omitempty"`

	// Subscription: The subscription from which a message should be pulled.
	Subscription string `json:"subscription,omitempty"`
}

type PullResponse struct {
	// AckId: This ID must be used to acknowledge the received event or
	// message.
	AckId string `json:"ackId,omitempty"`

	// PubsubEvent: A pubsub message or truncation event.
	PubsubEvent *PubsubEvent `json:"pubsubEvent,omitempty"`
}

type PushConfig struct {
	// PushEndpoint: A URL locating the endpoint to which messages should be
	// pushed. For example, a Webhook endpoint might use
	// "https://example.com/push".
	PushEndpoint string `json:"pushEndpoint,omitempty"`
}

type Subscription struct {
	// AckDeadlineSeconds: For either push or pull delivery, the value is
	// the maximum time after a subscriber receives a message before the
	// subscriber should acknowledge or Nack the message. If the Ack
	// deadline for a message passes without an Ack or a Nack, the Pub/Sub
	// system will eventually redeliver the message. If a subscriber
	// acknowledges after the deadline, the Pub/Sub system may accept the
	// Ack, but it is possible that the message has been already delivered
	// again. Multiple Acks to the message are allowed and will
	// succeed.
	//
	// For push delivery, this value is used to set the request
	// timeout for the call to the push endpoint.
	//
	// For pull delivery, this
	// value is used as the initial value for the Ack deadline. It may be
	// overridden for a specific pull request (message) with
	// ModifyAckDeadline. While a message is outstanding (i.e. it has been
	// delivered to a pull subscriber and the subscriber has not yet Acked
	// or Nacked), the Pub/Sub system will not deliver that message to
	// another pull subscriber (on a best-effort basis).
	AckDeadlineSeconds int64 `json:"ackDeadlineSeconds,omitempty"`

	// Name: Name of the subscription.
	Name string `json:"name,omitempty"`

	// PushConfig: If push delivery is used with this subscription, this
	// field is used to configure it.
	PushConfig *PushConfig `json:"pushConfig,omitempty"`

	// Topic: The name of the topic from which this subscription is
	// receiving messages.
	Topic string `json:"topic,omitempty"`
}

type Topic struct {
	// Name: Name of the topic.
	Name string `json:"name,omitempty"`
}

// method id "pubsub.subscriptions.acknowledge":

type SubscriptionsAcknowledgeCall struct {
	s                  *Service
	acknowledgerequest *AcknowledgeRequest
	opt_               map[string]interface{}
}

// Acknowledge: Acknowledges a particular received message: the Pub/Sub
// system can remove the given message from the subscription.
// Acknowledging a message whose Ack deadline has expired may succeed,
// but the message could have been already redelivered. Acknowledging a
// message more than once will not result in an error. This is only used
// for messages received via pull.
func (r *SubscriptionsService) Acknowledge(acknowledgerequest *AcknowledgeRequest) *SubscriptionsAcknowledgeCall {
	c := &SubscriptionsAcknowledgeCall{s: r.s, opt_: make(map[string]interface{})}
	c.acknowledgerequest = acknowledgerequest
	return c
}

// Fields allows partial responses to be retrieved.
// See https://developers.google.com/gdata/docs/2.0/basics#PartialResponse
// for more information.
func (c *SubscriptionsAcknowledgeCall) Fields(s ...googleapi.Field) *SubscriptionsAcknowledgeCall {
	c.opt_["fields"] = googleapi.CombineFields(s)
	return c
}

func (c *SubscriptionsAcknowledgeCall) Do() error {
	var body io.Reader = nil
	body, err := googleapi.WithoutDataWrapper.JSONReader(c.acknowledgerequest)
	if err != nil {
		return err
	}
	ctype := "application/json"
	params := make(url.Values)
	params.Set("alt", "json")
	if v, ok := c.opt_["fields"]; ok {
		params.Set("fields", fmt.Sprintf("%v", v))
	}
	urls := googleapi.ResolveRelative(c.s.BasePath, "subscriptions/acknowledge")
	urls += "?" + params.Encode()
	req, _ := http.NewRequest("POST", urls, body)
	googleapi.SetOpaque(req.URL)
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
	//   "description": "Acknowledges a particular received message: the Pub/Sub system can remove the given message from the subscription. Acknowledging a message whose Ack deadline has expired may succeed, but the message could have been already redelivered. Acknowledging a message more than once will not result in an error. This is only used for messages received via pull.",
	//   "httpMethod": "POST",
	//   "id": "pubsub.subscriptions.acknowledge",
	//   "path": "subscriptions/acknowledge",
	//   "request": {
	//     "$ref": "AcknowledgeRequest"
	//   },
	//   "scopes": [
	//     "https://www.googleapis.com/auth/cloud-platform",
	//     "https://www.googleapis.com/auth/pubsub"
	//   ]
	// }

}

// method id "pubsub.subscriptions.create":

type SubscriptionsCreateCall struct {
	s            *Service
	subscription *Subscription
	opt_         map[string]interface{}
}

// Create: Creates a subscription on a given topic for a given
// subscriber. If the subscription already exists, returns
// ALREADY_EXISTS. If the corresponding topic doesn't exist, returns
// NOT_FOUND.
//
// If the name is not provided in the request, the server
// will assign a random name for this subscription on the same project
// as the topic.
func (r *SubscriptionsService) Create(subscription *Subscription) *SubscriptionsCreateCall {
	c := &SubscriptionsCreateCall{s: r.s, opt_: make(map[string]interface{})}
	c.subscription = subscription
	return c
}

// Fields allows partial responses to be retrieved.
// See https://developers.google.com/gdata/docs/2.0/basics#PartialResponse
// for more information.
func (c *SubscriptionsCreateCall) Fields(s ...googleapi.Field) *SubscriptionsCreateCall {
	c.opt_["fields"] = googleapi.CombineFields(s)
	return c
}

func (c *SubscriptionsCreateCall) Do() (*Subscription, error) {
	var body io.Reader = nil
	body, err := googleapi.WithoutDataWrapper.JSONReader(c.subscription)
	if err != nil {
		return nil, err
	}
	ctype := "application/json"
	params := make(url.Values)
	params.Set("alt", "json")
	if v, ok := c.opt_["fields"]; ok {
		params.Set("fields", fmt.Sprintf("%v", v))
	}
	urls := googleapi.ResolveRelative(c.s.BasePath, "subscriptions")
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
	var ret *Subscription
	if err := json.NewDecoder(res.Body).Decode(&ret); err != nil {
		return nil, err
	}
	return ret, nil
	// {
	//   "description": "Creates a subscription on a given topic for a given subscriber. If the subscription already exists, returns ALREADY_EXISTS. If the corresponding topic doesn't exist, returns NOT_FOUND.\n\nIf the name is not provided in the request, the server will assign a random name for this subscription on the same project as the topic.",
	//   "httpMethod": "POST",
	//   "id": "pubsub.subscriptions.create",
	//   "path": "subscriptions",
	//   "request": {
	//     "$ref": "Subscription"
	//   },
	//   "response": {
	//     "$ref": "Subscription"
	//   },
	//   "scopes": [
	//     "https://www.googleapis.com/auth/cloud-platform",
	//     "https://www.googleapis.com/auth/pubsub"
	//   ]
	// }

}

// method id "pubsub.subscriptions.delete":

type SubscriptionsDeleteCall struct {
	s            *Service
	subscription string
	opt_         map[string]interface{}
}

// Delete: Deletes an existing subscription. All pending messages in the
// subscription are immediately dropped. Calls to Pull after deletion
// will return NOT_FOUND.
func (r *SubscriptionsService) Delete(subscription string) *SubscriptionsDeleteCall {
	c := &SubscriptionsDeleteCall{s: r.s, opt_: make(map[string]interface{})}
	c.subscription = subscription
	return c
}

// Fields allows partial responses to be retrieved.
// See https://developers.google.com/gdata/docs/2.0/basics#PartialResponse
// for more information.
func (c *SubscriptionsDeleteCall) Fields(s ...googleapi.Field) *SubscriptionsDeleteCall {
	c.opt_["fields"] = googleapi.CombineFields(s)
	return c
}

func (c *SubscriptionsDeleteCall) Do() error {
	var body io.Reader = nil
	params := make(url.Values)
	params.Set("alt", "json")
	if v, ok := c.opt_["fields"]; ok {
		params.Set("fields", fmt.Sprintf("%v", v))
	}
	urls := googleapi.ResolveRelative(c.s.BasePath, "subscriptions/{+subscription}")
	urls += "?" + params.Encode()
	req, _ := http.NewRequest("DELETE", urls, body)
	googleapi.Expand(req.URL, map[string]string{
		"subscription": c.subscription,
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
	//   "description": "Deletes an existing subscription. All pending messages in the subscription are immediately dropped. Calls to Pull after deletion will return NOT_FOUND.",
	//   "httpMethod": "DELETE",
	//   "id": "pubsub.subscriptions.delete",
	//   "parameterOrder": [
	//     "subscription"
	//   ],
	//   "parameters": {
	//     "subscription": {
	//       "description": "The subscription to delete.",
	//       "location": "path",
	//       "required": true,
	//       "type": "string"
	//     }
	//   },
	//   "path": "subscriptions/{+subscription}",
	//   "scopes": [
	//     "https://www.googleapis.com/auth/cloud-platform",
	//     "https://www.googleapis.com/auth/pubsub"
	//   ]
	// }

}

// method id "pubsub.subscriptions.get":

type SubscriptionsGetCall struct {
	s            *Service
	subscription string
	opt_         map[string]interface{}
}

// Get: Gets the configuration details of a subscription.
func (r *SubscriptionsService) Get(subscription string) *SubscriptionsGetCall {
	c := &SubscriptionsGetCall{s: r.s, opt_: make(map[string]interface{})}
	c.subscription = subscription
	return c
}

// Fields allows partial responses to be retrieved.
// See https://developers.google.com/gdata/docs/2.0/basics#PartialResponse
// for more information.
func (c *SubscriptionsGetCall) Fields(s ...googleapi.Field) *SubscriptionsGetCall {
	c.opt_["fields"] = googleapi.CombineFields(s)
	return c
}

func (c *SubscriptionsGetCall) Do() (*Subscription, error) {
	var body io.Reader = nil
	params := make(url.Values)
	params.Set("alt", "json")
	if v, ok := c.opt_["fields"]; ok {
		params.Set("fields", fmt.Sprintf("%v", v))
	}
	urls := googleapi.ResolveRelative(c.s.BasePath, "subscriptions/{+subscription}")
	urls += "?" + params.Encode()
	req, _ := http.NewRequest("GET", urls, body)
	googleapi.Expand(req.URL, map[string]string{
		"subscription": c.subscription,
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
	var ret *Subscription
	if err := json.NewDecoder(res.Body).Decode(&ret); err != nil {
		return nil, err
	}
	return ret, nil
	// {
	//   "description": "Gets the configuration details of a subscription.",
	//   "httpMethod": "GET",
	//   "id": "pubsub.subscriptions.get",
	//   "parameterOrder": [
	//     "subscription"
	//   ],
	//   "parameters": {
	//     "subscription": {
	//       "description": "The name of the subscription to get.",
	//       "location": "path",
	//       "required": true,
	//       "type": "string"
	//     }
	//   },
	//   "path": "subscriptions/{+subscription}",
	//   "response": {
	//     "$ref": "Subscription"
	//   },
	//   "scopes": [
	//     "https://www.googleapis.com/auth/cloud-platform",
	//     "https://www.googleapis.com/auth/pubsub"
	//   ]
	// }

}

// method id "pubsub.subscriptions.list":

type SubscriptionsListCall struct {
	s    *Service
	opt_ map[string]interface{}
}

// List: Lists matching subscriptions.
func (r *SubscriptionsService) List() *SubscriptionsListCall {
	c := &SubscriptionsListCall{s: r.s, opt_: make(map[string]interface{})}
	return c
}

// MaxResults sets the optional parameter "maxResults": Maximum number
// of subscriptions to return.
func (c *SubscriptionsListCall) MaxResults(maxResults int64) *SubscriptionsListCall {
	c.opt_["maxResults"] = maxResults
	return c
}

// PageToken sets the optional parameter "pageToken": The value obtained
// in the last ListSubscriptionsResponse for continuation.
func (c *SubscriptionsListCall) PageToken(pageToken string) *SubscriptionsListCall {
	c.opt_["pageToken"] = pageToken
	return c
}

// Query sets the optional parameter "query": A valid label query
// expression.
func (c *SubscriptionsListCall) Query(query string) *SubscriptionsListCall {
	c.opt_["query"] = query
	return c
}

// Fields allows partial responses to be retrieved.
// See https://developers.google.com/gdata/docs/2.0/basics#PartialResponse
// for more information.
func (c *SubscriptionsListCall) Fields(s ...googleapi.Field) *SubscriptionsListCall {
	c.opt_["fields"] = googleapi.CombineFields(s)
	return c
}

func (c *SubscriptionsListCall) Do() (*ListSubscriptionsResponse, error) {
	var body io.Reader = nil
	params := make(url.Values)
	params.Set("alt", "json")
	if v, ok := c.opt_["maxResults"]; ok {
		params.Set("maxResults", fmt.Sprintf("%v", v))
	}
	if v, ok := c.opt_["pageToken"]; ok {
		params.Set("pageToken", fmt.Sprintf("%v", v))
	}
	if v, ok := c.opt_["query"]; ok {
		params.Set("query", fmt.Sprintf("%v", v))
	}
	if v, ok := c.opt_["fields"]; ok {
		params.Set("fields", fmt.Sprintf("%v", v))
	}
	urls := googleapi.ResolveRelative(c.s.BasePath, "subscriptions")
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
	var ret *ListSubscriptionsResponse
	if err := json.NewDecoder(res.Body).Decode(&ret); err != nil {
		return nil, err
	}
	return ret, nil
	// {
	//   "description": "Lists matching subscriptions.",
	//   "httpMethod": "GET",
	//   "id": "pubsub.subscriptions.list",
	//   "parameters": {
	//     "maxResults": {
	//       "description": "Maximum number of subscriptions to return.",
	//       "format": "int32",
	//       "location": "query",
	//       "type": "integer"
	//     },
	//     "pageToken": {
	//       "description": "The value obtained in the last ListSubscriptionsResponse for continuation.",
	//       "location": "query",
	//       "type": "string"
	//     },
	//     "query": {
	//       "description": "A valid label query expression.",
	//       "location": "query",
	//       "type": "string"
	//     }
	//   },
	//   "path": "subscriptions",
	//   "response": {
	//     "$ref": "ListSubscriptionsResponse"
	//   },
	//   "scopes": [
	//     "https://www.googleapis.com/auth/cloud-platform",
	//     "https://www.googleapis.com/auth/pubsub"
	//   ]
	// }

}

// method id "pubsub.subscriptions.modifyAckDeadline":

type SubscriptionsModifyAckDeadlineCall struct {
	s                        *Service
	modifyackdeadlinerequest *ModifyAckDeadlineRequest
	opt_                     map[string]interface{}
}

// ModifyAckDeadline: Modifies the Ack deadline for a message received
// from a pull request.
func (r *SubscriptionsService) ModifyAckDeadline(modifyackdeadlinerequest *ModifyAckDeadlineRequest) *SubscriptionsModifyAckDeadlineCall {
	c := &SubscriptionsModifyAckDeadlineCall{s: r.s, opt_: make(map[string]interface{})}
	c.modifyackdeadlinerequest = modifyackdeadlinerequest
	return c
}

// Fields allows partial responses to be retrieved.
// See https://developers.google.com/gdata/docs/2.0/basics#PartialResponse
// for more information.
func (c *SubscriptionsModifyAckDeadlineCall) Fields(s ...googleapi.Field) *SubscriptionsModifyAckDeadlineCall {
	c.opt_["fields"] = googleapi.CombineFields(s)
	return c
}

func (c *SubscriptionsModifyAckDeadlineCall) Do() error {
	var body io.Reader = nil
	body, err := googleapi.WithoutDataWrapper.JSONReader(c.modifyackdeadlinerequest)
	if err != nil {
		return err
	}
	ctype := "application/json"
	params := make(url.Values)
	params.Set("alt", "json")
	if v, ok := c.opt_["fields"]; ok {
		params.Set("fields", fmt.Sprintf("%v", v))
	}
	urls := googleapi.ResolveRelative(c.s.BasePath, "subscriptions/modifyAckDeadline")
	urls += "?" + params.Encode()
	req, _ := http.NewRequest("POST", urls, body)
	googleapi.SetOpaque(req.URL)
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
	//   "description": "Modifies the Ack deadline for a message received from a pull request.",
	//   "httpMethod": "POST",
	//   "id": "pubsub.subscriptions.modifyAckDeadline",
	//   "path": "subscriptions/modifyAckDeadline",
	//   "request": {
	//     "$ref": "ModifyAckDeadlineRequest"
	//   },
	//   "scopes": [
	//     "https://www.googleapis.com/auth/cloud-platform",
	//     "https://www.googleapis.com/auth/pubsub"
	//   ]
	// }

}

// method id "pubsub.subscriptions.modifyPushConfig":

type SubscriptionsModifyPushConfigCall struct {
	s                       *Service
	modifypushconfigrequest *ModifyPushConfigRequest
	opt_                    map[string]interface{}
}

// ModifyPushConfig: Modifies the PushConfig for a specified
// subscription. This method can be used to suspend the flow of messages
// to an end point by clearing the PushConfig field in the request.
// Messages will be accumulated for delivery even if no push
// configuration is defined or while the configuration is modified.
func (r *SubscriptionsService) ModifyPushConfig(modifypushconfigrequest *ModifyPushConfigRequest) *SubscriptionsModifyPushConfigCall {
	c := &SubscriptionsModifyPushConfigCall{s: r.s, opt_: make(map[string]interface{})}
	c.modifypushconfigrequest = modifypushconfigrequest
	return c
}

// Fields allows partial responses to be retrieved.
// See https://developers.google.com/gdata/docs/2.0/basics#PartialResponse
// for more information.
func (c *SubscriptionsModifyPushConfigCall) Fields(s ...googleapi.Field) *SubscriptionsModifyPushConfigCall {
	c.opt_["fields"] = googleapi.CombineFields(s)
	return c
}

func (c *SubscriptionsModifyPushConfigCall) Do() error {
	var body io.Reader = nil
	body, err := googleapi.WithoutDataWrapper.JSONReader(c.modifypushconfigrequest)
	if err != nil {
		return err
	}
	ctype := "application/json"
	params := make(url.Values)
	params.Set("alt", "json")
	if v, ok := c.opt_["fields"]; ok {
		params.Set("fields", fmt.Sprintf("%v", v))
	}
	urls := googleapi.ResolveRelative(c.s.BasePath, "subscriptions/modifyPushConfig")
	urls += "?" + params.Encode()
	req, _ := http.NewRequest("POST", urls, body)
	googleapi.SetOpaque(req.URL)
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
	//   "description": "Modifies the PushConfig for a specified subscription. This method can be used to suspend the flow of messages to an end point by clearing the PushConfig field in the request. Messages will be accumulated for delivery even if no push configuration is defined or while the configuration is modified.",
	//   "httpMethod": "POST",
	//   "id": "pubsub.subscriptions.modifyPushConfig",
	//   "path": "subscriptions/modifyPushConfig",
	//   "request": {
	//     "$ref": "ModifyPushConfigRequest"
	//   },
	//   "scopes": [
	//     "https://www.googleapis.com/auth/cloud-platform",
	//     "https://www.googleapis.com/auth/pubsub"
	//   ]
	// }

}

// method id "pubsub.subscriptions.pull":

type SubscriptionsPullCall struct {
	s           *Service
	pullrequest *PullRequest
	opt_        map[string]interface{}
}

// Pull: Pulls a single message from the server. If return_immediately
// is true, and no messages are available in the subscription, this
// method returns FAILED_PRECONDITION. The system is free to return an
// UNAVAILABLE error if no messages are available in a reasonable amount
// of time (to reduce system load).
func (r *SubscriptionsService) Pull(pullrequest *PullRequest) *SubscriptionsPullCall {
	c := &SubscriptionsPullCall{s: r.s, opt_: make(map[string]interface{})}
	c.pullrequest = pullrequest
	return c
}

// Fields allows partial responses to be retrieved.
// See https://developers.google.com/gdata/docs/2.0/basics#PartialResponse
// for more information.
func (c *SubscriptionsPullCall) Fields(s ...googleapi.Field) *SubscriptionsPullCall {
	c.opt_["fields"] = googleapi.CombineFields(s)
	return c
}

func (c *SubscriptionsPullCall) Do() (*PullResponse, error) {
	var body io.Reader = nil
	body, err := googleapi.WithoutDataWrapper.JSONReader(c.pullrequest)
	if err != nil {
		return nil, err
	}
	ctype := "application/json"
	params := make(url.Values)
	params.Set("alt", "json")
	if v, ok := c.opt_["fields"]; ok {
		params.Set("fields", fmt.Sprintf("%v", v))
	}
	urls := googleapi.ResolveRelative(c.s.BasePath, "subscriptions/pull")
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
	var ret *PullResponse
	if err := json.NewDecoder(res.Body).Decode(&ret); err != nil {
		return nil, err
	}
	return ret, nil
	// {
	//   "description": "Pulls a single message from the server. If return_immediately is true, and no messages are available in the subscription, this method returns FAILED_PRECONDITION. The system is free to return an UNAVAILABLE error if no messages are available in a reasonable amount of time (to reduce system load).",
	//   "httpMethod": "POST",
	//   "id": "pubsub.subscriptions.pull",
	//   "path": "subscriptions/pull",
	//   "request": {
	//     "$ref": "PullRequest"
	//   },
	//   "response": {
	//     "$ref": "PullResponse"
	//   },
	//   "scopes": [
	//     "https://www.googleapis.com/auth/cloud-platform",
	//     "https://www.googleapis.com/auth/pubsub"
	//   ]
	// }

}

// method id "pubsub.subscriptions.pullBatch":

type SubscriptionsPullBatchCall struct {
	s                *Service
	pullbatchrequest *PullBatchRequest
	opt_             map[string]interface{}
}

// PullBatch: Pulls messages from the server. Returns an empty list if
// there are no messages available in the backlog. The system is free to
// return UNAVAILABLE if there too many pull requests outstanding for a
// given subscription.
func (r *SubscriptionsService) PullBatch(pullbatchrequest *PullBatchRequest) *SubscriptionsPullBatchCall {
	c := &SubscriptionsPullBatchCall{s: r.s, opt_: make(map[string]interface{})}
	c.pullbatchrequest = pullbatchrequest
	return c
}

// Fields allows partial responses to be retrieved.
// See https://developers.google.com/gdata/docs/2.0/basics#PartialResponse
// for more information.
func (c *SubscriptionsPullBatchCall) Fields(s ...googleapi.Field) *SubscriptionsPullBatchCall {
	c.opt_["fields"] = googleapi.CombineFields(s)
	return c
}

func (c *SubscriptionsPullBatchCall) Do() (*PullBatchResponse, error) {
	var body io.Reader = nil
	body, err := googleapi.WithoutDataWrapper.JSONReader(c.pullbatchrequest)
	if err != nil {
		return nil, err
	}
	ctype := "application/json"
	params := make(url.Values)
	params.Set("alt", "json")
	if v, ok := c.opt_["fields"]; ok {
		params.Set("fields", fmt.Sprintf("%v", v))
	}
	urls := googleapi.ResolveRelative(c.s.BasePath, "subscriptions/pullBatch")
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
	var ret *PullBatchResponse
	if err := json.NewDecoder(res.Body).Decode(&ret); err != nil {
		return nil, err
	}
	return ret, nil
	// {
	//   "description": "Pulls messages from the server. Returns an empty list if there are no messages available in the backlog. The system is free to return UNAVAILABLE if there too many pull requests outstanding for a given subscription.",
	//   "httpMethod": "POST",
	//   "id": "pubsub.subscriptions.pullBatch",
	//   "path": "subscriptions/pullBatch",
	//   "request": {
	//     "$ref": "PullBatchRequest"
	//   },
	//   "response": {
	//     "$ref": "PullBatchResponse"
	//   },
	//   "scopes": [
	//     "https://www.googleapis.com/auth/cloud-platform",
	//     "https://www.googleapis.com/auth/pubsub"
	//   ]
	// }

}

// method id "pubsub.topics.create":

type TopicsCreateCall struct {
	s     *Service
	topic *Topic
	opt_  map[string]interface{}
}

// Create: Creates the given topic with the given name.
func (r *TopicsService) Create(topic *Topic) *TopicsCreateCall {
	c := &TopicsCreateCall{s: r.s, opt_: make(map[string]interface{})}
	c.topic = topic
	return c
}

// Fields allows partial responses to be retrieved.
// See https://developers.google.com/gdata/docs/2.0/basics#PartialResponse
// for more information.
func (c *TopicsCreateCall) Fields(s ...googleapi.Field) *TopicsCreateCall {
	c.opt_["fields"] = googleapi.CombineFields(s)
	return c
}

func (c *TopicsCreateCall) Do() (*Topic, error) {
	var body io.Reader = nil
	body, err := googleapi.WithoutDataWrapper.JSONReader(c.topic)
	if err != nil {
		return nil, err
	}
	ctype := "application/json"
	params := make(url.Values)
	params.Set("alt", "json")
	if v, ok := c.opt_["fields"]; ok {
		params.Set("fields", fmt.Sprintf("%v", v))
	}
	urls := googleapi.ResolveRelative(c.s.BasePath, "topics")
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
	var ret *Topic
	if err := json.NewDecoder(res.Body).Decode(&ret); err != nil {
		return nil, err
	}
	return ret, nil
	// {
	//   "description": "Creates the given topic with the given name.",
	//   "httpMethod": "POST",
	//   "id": "pubsub.topics.create",
	//   "path": "topics",
	//   "request": {
	//     "$ref": "Topic"
	//   },
	//   "response": {
	//     "$ref": "Topic"
	//   },
	//   "scopes": [
	//     "https://www.googleapis.com/auth/cloud-platform",
	//     "https://www.googleapis.com/auth/pubsub"
	//   ]
	// }

}

// method id "pubsub.topics.delete":

type TopicsDeleteCall struct {
	s     *Service
	topic string
	opt_  map[string]interface{}
}

// Delete: Deletes the topic with the given name. All subscriptions to
// this topic are also deleted. Returns NOT_FOUND if the topic does not
// exist. After a topic is deleted, a new topic may be created with the
// same name.
func (r *TopicsService) Delete(topic string) *TopicsDeleteCall {
	c := &TopicsDeleteCall{s: r.s, opt_: make(map[string]interface{})}
	c.topic = topic
	return c
}

// Fields allows partial responses to be retrieved.
// See https://developers.google.com/gdata/docs/2.0/basics#PartialResponse
// for more information.
func (c *TopicsDeleteCall) Fields(s ...googleapi.Field) *TopicsDeleteCall {
	c.opt_["fields"] = googleapi.CombineFields(s)
	return c
}

func (c *TopicsDeleteCall) Do() error {
	var body io.Reader = nil
	params := make(url.Values)
	params.Set("alt", "json")
	if v, ok := c.opt_["fields"]; ok {
		params.Set("fields", fmt.Sprintf("%v", v))
	}
	urls := googleapi.ResolveRelative(c.s.BasePath, "topics/{+topic}")
	urls += "?" + params.Encode()
	req, _ := http.NewRequest("DELETE", urls, body)
	googleapi.Expand(req.URL, map[string]string{
		"topic": c.topic,
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
	//   "description": "Deletes the topic with the given name. All subscriptions to this topic are also deleted. Returns NOT_FOUND if the topic does not exist. After a topic is deleted, a new topic may be created with the same name.",
	//   "httpMethod": "DELETE",
	//   "id": "pubsub.topics.delete",
	//   "parameterOrder": [
	//     "topic"
	//   ],
	//   "parameters": {
	//     "topic": {
	//       "description": "Name of the topic to delete.",
	//       "location": "path",
	//       "required": true,
	//       "type": "string"
	//     }
	//   },
	//   "path": "topics/{+topic}",
	//   "scopes": [
	//     "https://www.googleapis.com/auth/cloud-platform",
	//     "https://www.googleapis.com/auth/pubsub"
	//   ]
	// }

}

// method id "pubsub.topics.get":

type TopicsGetCall struct {
	s     *Service
	topic string
	opt_  map[string]interface{}
}

// Get: Gets the configuration of a topic. Since the topic only has the
// name attribute, this method is only useful to check the existence of
// a topic. If other attributes are added in the future, they will be
// returned here.
func (r *TopicsService) Get(topic string) *TopicsGetCall {
	c := &TopicsGetCall{s: r.s, opt_: make(map[string]interface{})}
	c.topic = topic
	return c
}

// Fields allows partial responses to be retrieved.
// See https://developers.google.com/gdata/docs/2.0/basics#PartialResponse
// for more information.
func (c *TopicsGetCall) Fields(s ...googleapi.Field) *TopicsGetCall {
	c.opt_["fields"] = googleapi.CombineFields(s)
	return c
}

func (c *TopicsGetCall) Do() (*Topic, error) {
	var body io.Reader = nil
	params := make(url.Values)
	params.Set("alt", "json")
	if v, ok := c.opt_["fields"]; ok {
		params.Set("fields", fmt.Sprintf("%v", v))
	}
	urls := googleapi.ResolveRelative(c.s.BasePath, "topics/{+topic}")
	urls += "?" + params.Encode()
	req, _ := http.NewRequest("GET", urls, body)
	googleapi.Expand(req.URL, map[string]string{
		"topic": c.topic,
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
	var ret *Topic
	if err := json.NewDecoder(res.Body).Decode(&ret); err != nil {
		return nil, err
	}
	return ret, nil
	// {
	//   "description": "Gets the configuration of a topic. Since the topic only has the name attribute, this method is only useful to check the existence of a topic. If other attributes are added in the future, they will be returned here.",
	//   "httpMethod": "GET",
	//   "id": "pubsub.topics.get",
	//   "parameterOrder": [
	//     "topic"
	//   ],
	//   "parameters": {
	//     "topic": {
	//       "description": "The name of the topic to get.",
	//       "location": "path",
	//       "required": true,
	//       "type": "string"
	//     }
	//   },
	//   "path": "topics/{+topic}",
	//   "response": {
	//     "$ref": "Topic"
	//   },
	//   "scopes": [
	//     "https://www.googleapis.com/auth/cloud-platform",
	//     "https://www.googleapis.com/auth/pubsub"
	//   ]
	// }

}

// method id "pubsub.topics.list":

type TopicsListCall struct {
	s    *Service
	opt_ map[string]interface{}
}

// List: Lists matching topics.
func (r *TopicsService) List() *TopicsListCall {
	c := &TopicsListCall{s: r.s, opt_: make(map[string]interface{})}
	return c
}

// MaxResults sets the optional parameter "maxResults": Maximum number
// of topics to return.
func (c *TopicsListCall) MaxResults(maxResults int64) *TopicsListCall {
	c.opt_["maxResults"] = maxResults
	return c
}

// PageToken sets the optional parameter "pageToken": The value obtained
// in the last ListTopicsResponse for continuation.
func (c *TopicsListCall) PageToken(pageToken string) *TopicsListCall {
	c.opt_["pageToken"] = pageToken
	return c
}

// Query sets the optional parameter "query": A valid label query
// expression.
func (c *TopicsListCall) Query(query string) *TopicsListCall {
	c.opt_["query"] = query
	return c
}

// Fields allows partial responses to be retrieved.
// See https://developers.google.com/gdata/docs/2.0/basics#PartialResponse
// for more information.
func (c *TopicsListCall) Fields(s ...googleapi.Field) *TopicsListCall {
	c.opt_["fields"] = googleapi.CombineFields(s)
	return c
}

func (c *TopicsListCall) Do() (*ListTopicsResponse, error) {
	var body io.Reader = nil
	params := make(url.Values)
	params.Set("alt", "json")
	if v, ok := c.opt_["maxResults"]; ok {
		params.Set("maxResults", fmt.Sprintf("%v", v))
	}
	if v, ok := c.opt_["pageToken"]; ok {
		params.Set("pageToken", fmt.Sprintf("%v", v))
	}
	if v, ok := c.opt_["query"]; ok {
		params.Set("query", fmt.Sprintf("%v", v))
	}
	if v, ok := c.opt_["fields"]; ok {
		params.Set("fields", fmt.Sprintf("%v", v))
	}
	urls := googleapi.ResolveRelative(c.s.BasePath, "topics")
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
	var ret *ListTopicsResponse
	if err := json.NewDecoder(res.Body).Decode(&ret); err != nil {
		return nil, err
	}
	return ret, nil
	// {
	//   "description": "Lists matching topics.",
	//   "httpMethod": "GET",
	//   "id": "pubsub.topics.list",
	//   "parameters": {
	//     "maxResults": {
	//       "description": "Maximum number of topics to return.",
	//       "format": "int32",
	//       "location": "query",
	//       "type": "integer"
	//     },
	//     "pageToken": {
	//       "description": "The value obtained in the last ListTopicsResponse for continuation.",
	//       "location": "query",
	//       "type": "string"
	//     },
	//     "query": {
	//       "description": "A valid label query expression.",
	//       "location": "query",
	//       "type": "string"
	//     }
	//   },
	//   "path": "topics",
	//   "response": {
	//     "$ref": "ListTopicsResponse"
	//   },
	//   "scopes": [
	//     "https://www.googleapis.com/auth/cloud-platform",
	//     "https://www.googleapis.com/auth/pubsub"
	//   ]
	// }

}

// method id "pubsub.topics.publish":

type TopicsPublishCall struct {
	s              *Service
	publishrequest *PublishRequest
	opt_           map[string]interface{}
}

// Publish: Adds a message to the topic. Returns NOT_FOUND if the topic
// does not exist.
func (r *TopicsService) Publish(publishrequest *PublishRequest) *TopicsPublishCall {
	c := &TopicsPublishCall{s: r.s, opt_: make(map[string]interface{})}
	c.publishrequest = publishrequest
	return c
}

// Fields allows partial responses to be retrieved.
// See https://developers.google.com/gdata/docs/2.0/basics#PartialResponse
// for more information.
func (c *TopicsPublishCall) Fields(s ...googleapi.Field) *TopicsPublishCall {
	c.opt_["fields"] = googleapi.CombineFields(s)
	return c
}

func (c *TopicsPublishCall) Do() error {
	var body io.Reader = nil
	body, err := googleapi.WithoutDataWrapper.JSONReader(c.publishrequest)
	if err != nil {
		return err
	}
	ctype := "application/json"
	params := make(url.Values)
	params.Set("alt", "json")
	if v, ok := c.opt_["fields"]; ok {
		params.Set("fields", fmt.Sprintf("%v", v))
	}
	urls := googleapi.ResolveRelative(c.s.BasePath, "topics/publish")
	urls += "?" + params.Encode()
	req, _ := http.NewRequest("POST", urls, body)
	googleapi.SetOpaque(req.URL)
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
	//   "description": "Adds a message to the topic. Returns NOT_FOUND if the topic does not exist.",
	//   "httpMethod": "POST",
	//   "id": "pubsub.topics.publish",
	//   "path": "topics/publish",
	//   "request": {
	//     "$ref": "PublishRequest"
	//   },
	//   "scopes": [
	//     "https://www.googleapis.com/auth/cloud-platform",
	//     "https://www.googleapis.com/auth/pubsub"
	//   ]
	// }

}

// method id "pubsub.topics.publishBatch":

type TopicsPublishBatchCall struct {
	s                   *Service
	publishbatchrequest *PublishBatchRequest
	opt_                map[string]interface{}
}

// PublishBatch: Adds one or more messages to the topic. Returns
// NOT_FOUND if the topic does not exist.
func (r *TopicsService) PublishBatch(publishbatchrequest *PublishBatchRequest) *TopicsPublishBatchCall {
	c := &TopicsPublishBatchCall{s: r.s, opt_: make(map[string]interface{})}
	c.publishbatchrequest = publishbatchrequest
	return c
}

// Fields allows partial responses to be retrieved.
// See https://developers.google.com/gdata/docs/2.0/basics#PartialResponse
// for more information.
func (c *TopicsPublishBatchCall) Fields(s ...googleapi.Field) *TopicsPublishBatchCall {
	c.opt_["fields"] = googleapi.CombineFields(s)
	return c
}

func (c *TopicsPublishBatchCall) Do() (*PublishBatchResponse, error) {
	var body io.Reader = nil
	body, err := googleapi.WithoutDataWrapper.JSONReader(c.publishbatchrequest)
	if err != nil {
		return nil, err
	}
	ctype := "application/json"
	params := make(url.Values)
	params.Set("alt", "json")
	if v, ok := c.opt_["fields"]; ok {
		params.Set("fields", fmt.Sprintf("%v", v))
	}
	urls := googleapi.ResolveRelative(c.s.BasePath, "topics/publishBatch")
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
	var ret *PublishBatchResponse
	if err := json.NewDecoder(res.Body).Decode(&ret); err != nil {
		return nil, err
	}
	return ret, nil
	// {
	//   "description": "Adds one or more messages to the topic. Returns NOT_FOUND if the topic does not exist.",
	//   "httpMethod": "POST",
	//   "id": "pubsub.topics.publishBatch",
	//   "path": "topics/publishBatch",
	//   "request": {
	//     "$ref": "PublishBatchRequest"
	//   },
	//   "response": {
	//     "$ref": "PublishBatchResponse"
	//   },
	//   "scopes": [
	//     "https://www.googleapis.com/auth/cloud-platform",
	//     "https://www.googleapis.com/auth/pubsub"
	//   ]
	// }

}
