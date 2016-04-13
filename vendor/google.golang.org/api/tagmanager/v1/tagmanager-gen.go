// Package tagmanager provides access to the Tag Manager API.
//
// See https://developers.google.com/tag-manager/api/v1/
//
// Usage example:
//
//   import "google.golang.org/api/tagmanager/v1"
//   ...
//   tagmanagerService, err := tagmanager.New(oauthHttpClient)
package tagmanager

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

const apiId = "tagmanager:v1"
const apiName = "tagmanager"
const apiVersion = "v1"
const basePath = "https://www.googleapis.com/tagmanager/v1/"

// OAuth2 scopes used by this API.
const (
	// Delete your Google Tag Manager containers
	TagmanagerDeleteContainersScope = "https://www.googleapis.com/auth/tagmanager.delete.containers"

	// Manage your Google Tag Manager containers
	TagmanagerEditContainersScope = "https://www.googleapis.com/auth/tagmanager.edit.containers"

	// Manage your Google Tag Manager container versions
	TagmanagerEditContainerversionsScope = "https://www.googleapis.com/auth/tagmanager.edit.containerversions"

	// Manage your Google Tag Manager accounts
	TagmanagerManageAccountsScope = "https://www.googleapis.com/auth/tagmanager.manage.accounts"

	// Manage user permissions of your Google Tag Manager data
	TagmanagerManageUsersScope = "https://www.googleapis.com/auth/tagmanager.manage.users"

	// Publish your Google Tag Manager containers
	TagmanagerPublishScope = "https://www.googleapis.com/auth/tagmanager.publish"

	// View your Google Tag Manager containers
	TagmanagerReadonlyScope = "https://www.googleapis.com/auth/tagmanager.readonly"
)

func New(client *http.Client) (*Service, error) {
	if client == nil {
		return nil, errors.New("client is nil")
	}
	s := &Service{client: client, BasePath: basePath}
	s.Accounts = NewAccountsService(s)
	return s, nil
}

type Service struct {
	client   *http.Client
	BasePath string // API endpoint base URL

	Accounts *AccountsService
}

func NewAccountsService(s *Service) *AccountsService {
	rs := &AccountsService{s: s}
	rs.Containers = NewAccountsContainersService(s)
	rs.Permissions = NewAccountsPermissionsService(s)
	return rs
}

type AccountsService struct {
	s *Service

	Containers *AccountsContainersService

	Permissions *AccountsPermissionsService
}

func NewAccountsContainersService(s *Service) *AccountsContainersService {
	rs := &AccountsContainersService{s: s}
	rs.Macros = NewAccountsContainersMacrosService(s)
	rs.Rules = NewAccountsContainersRulesService(s)
	rs.Tags = NewAccountsContainersTagsService(s)
	rs.Triggers = NewAccountsContainersTriggersService(s)
	rs.Variables = NewAccountsContainersVariablesService(s)
	rs.Versions = NewAccountsContainersVersionsService(s)
	return rs
}

type AccountsContainersService struct {
	s *Service

	Macros *AccountsContainersMacrosService

	Rules *AccountsContainersRulesService

	Tags *AccountsContainersTagsService

	Triggers *AccountsContainersTriggersService

	Variables *AccountsContainersVariablesService

	Versions *AccountsContainersVersionsService
}

func NewAccountsContainersMacrosService(s *Service) *AccountsContainersMacrosService {
	rs := &AccountsContainersMacrosService{s: s}
	return rs
}

type AccountsContainersMacrosService struct {
	s *Service
}

func NewAccountsContainersRulesService(s *Service) *AccountsContainersRulesService {
	rs := &AccountsContainersRulesService{s: s}
	return rs
}

type AccountsContainersRulesService struct {
	s *Service
}

func NewAccountsContainersTagsService(s *Service) *AccountsContainersTagsService {
	rs := &AccountsContainersTagsService{s: s}
	return rs
}

type AccountsContainersTagsService struct {
	s *Service
}

func NewAccountsContainersTriggersService(s *Service) *AccountsContainersTriggersService {
	rs := &AccountsContainersTriggersService{s: s}
	return rs
}

type AccountsContainersTriggersService struct {
	s *Service
}

func NewAccountsContainersVariablesService(s *Service) *AccountsContainersVariablesService {
	rs := &AccountsContainersVariablesService{s: s}
	return rs
}

type AccountsContainersVariablesService struct {
	s *Service
}

func NewAccountsContainersVersionsService(s *Service) *AccountsContainersVersionsService {
	rs := &AccountsContainersVersionsService{s: s}
	return rs
}

type AccountsContainersVersionsService struct {
	s *Service
}

func NewAccountsPermissionsService(s *Service) *AccountsPermissionsService {
	rs := &AccountsPermissionsService{s: s}
	return rs
}

type AccountsPermissionsService struct {
	s *Service
}

type Account struct {
	// AccountId: The Account ID uniquely identifies the GTM Account.
	AccountId string `json:"accountId,omitempty"`

	// Fingerprint: The fingerprint of the GTM Account as computed at
	// storage time. This value is recomputed whenever the account is
	// modified.
	Fingerprint string `json:"fingerprint,omitempty"`

	// Name: Account display name.
	Name string `json:"name,omitempty"`

	// ShareData: Whether the account shares data anonymously with Google
	// and others.
	ShareData bool `json:"shareData,omitempty"`
}

type AccountAccess struct {
	// Permission: List of Account permissions. Valid account permissions
	// are read and manage.
	Permission []string `json:"permission,omitempty"`
}

type Condition struct {
	// Parameter: A list of named parameters (key/value), depending on the
	// condition's type. Notes:
	// - For binary operators, include parameters
	// named arg0 and arg1 for specifying the left and right operands,
	// respectively.
	// - At this time, the left operand (arg0) must be a
	// reference to a macro.
	// - For case-insensitive Regex matching, include
	// a boolean parameter named ignore_case that is set to true. If not
	// specified or set to any other value, the matching will be case
	// sensitive.
	// - To negate an operator, include a boolean parameter
	// named negate boolean parameter that is set to true.
	Parameter []*Parameter `json:"parameter,omitempty"`

	// Type: The type of operator for this condition.
	Type string `json:"type,omitempty"`
}

type Container struct {
	// AccountId: GTM Account ID.
	AccountId string `json:"accountId,omitempty"`

	// ContainerId: The Container ID uniquely identifies the GTM Container.
	ContainerId string `json:"containerId,omitempty"`

	// DomainName: Optional list of domain names associated with the
	// Container.
	DomainName []string `json:"domainName,omitempty"`

	// Fingerprint: The fingerprint of the GTM Container as computed at
	// storage time. This value is recomputed whenever the account is
	// modified.
	Fingerprint string `json:"fingerprint,omitempty"`

	// Name: Container display name.
	Name string `json:"name,omitempty"`

	// Notes: Container Notes.
	Notes string `json:"notes,omitempty"`

	// PublicId: Container Public ID.
	PublicId string `json:"publicId,omitempty"`

	// TimeZoneCountryId: Container Country ID.
	TimeZoneCountryId string `json:"timeZoneCountryId,omitempty"`

	// TimeZoneId: Container Time Zone ID.
	TimeZoneId string `json:"timeZoneId,omitempty"`

	// UsageContext: List of Usage Contexts for the Container. Valid values
	// include: web, android, ios.
	UsageContext []string `json:"usageContext,omitempty"`
}

type ContainerAccess struct {
	// ContainerId: GTM Container ID.
	ContainerId string `json:"containerId,omitempty"`

	// Permission: List of Container permissions. Valid container
	// permissions are: read, edit, delete, publish.
	Permission []string `json:"permission,omitempty"`
}

type ContainerVersion struct {
	// AccountId: GTM Account ID.
	AccountId string `json:"accountId,omitempty"`

	// Container: The container that this version was taken from.
	Container *Container `json:"container,omitempty"`

	// ContainerId: GTM Container ID.
	ContainerId string `json:"containerId,omitempty"`

	// ContainerVersionId: The Container Version ID uniquely identifies the
	// GTM Container Version.
	ContainerVersionId string `json:"containerVersionId,omitempty"`

	// Deleted: A value of true indicates this container version has been
	// deleted.
	Deleted bool `json:"deleted,omitempty"`

	// Fingerprint: The fingerprint of the GTM Container Version as computed
	// at storage time. This value is recomputed whenever the container
	// version is modified.
	Fingerprint string `json:"fingerprint,omitempty"`

	// Macro: The macros in the container that this version was taken from.
	Macro []*Macro `json:"macro,omitempty"`

	// Name: Container version display name.
	Name string `json:"name,omitempty"`

	// Notes: User notes on how to apply this container version in the
	// container.
	Notes string `json:"notes,omitempty"`

	// Rule: The rules in the container that this version was taken from.
	Rule []*Rule `json:"rule,omitempty"`

	// Tag: The tags in the container that this version was taken from.
	Tag []*Tag `json:"tag,omitempty"`

	// Trigger: The triggers in the container that this version was taken
	// from.
	Trigger []*Trigger `json:"trigger,omitempty"`

	// Variable: The variables in the container that this version was taken
	// from.
	Variable []*Variable `json:"variable,omitempty"`
}

type ContainerVersionHeader struct {
	// AccountId: GTM Account ID.
	AccountId string `json:"accountId,omitempty"`

	// ContainerId: GTM Container ID.
	ContainerId string `json:"containerId,omitempty"`

	// ContainerVersionId: The Container Version ID uniquely identifies the
	// GTM Container Version.
	ContainerVersionId string `json:"containerVersionId,omitempty"`

	// Deleted: A value of true indicates this container version has been
	// deleted.
	Deleted bool `json:"deleted,omitempty"`

	// Name: Container version display name.
	Name string `json:"name,omitempty"`

	// NumMacros: Number of macros in the container version.
	NumMacros string `json:"numMacros,omitempty"`

	// NumRules: Number of rules in the container version.
	NumRules string `json:"numRules,omitempty"`

	// NumTags: Number of tags in the container version.
	NumTags string `json:"numTags,omitempty"`

	// NumTriggers: Number of triggers in the container version.
	NumTriggers string `json:"numTriggers,omitempty"`

	// NumVariables: Number of variables in the container version.
	NumVariables string `json:"numVariables,omitempty"`
}

type CreateContainerVersionRequestVersionOptions struct {
	// Name: The name of the container version to be created.
	Name string `json:"name,omitempty"`

	// Notes: The notes of the container version to be created.
	Notes string `json:"notes,omitempty"`

	// QuickPreview: The creation of this version may be for quick preview
	// and shouldn't be saved.
	QuickPreview bool `json:"quickPreview,omitempty"`
}

type CreateContainerVersionResponse struct {
	// CompilerError: Compiler errors or not.
	CompilerError bool `json:"compilerError,omitempty"`

	// ContainerVersion: The container version created.
	ContainerVersion *ContainerVersion `json:"containerVersion,omitempty"`
}

type ListAccountUsersResponse struct {
	// UserAccess: All GTM AccountUsers of a GTM Account.
	UserAccess []*UserAccess `json:"userAccess,omitempty"`
}

type ListAccountsResponse struct {
	// Accounts: List of GTM Accounts that a user has access to.
	Accounts []*Account `json:"accounts,omitempty"`
}

type ListContainerVersionsResponse struct {
	// ContainerVersion: All versions of a GTM Container.
	ContainerVersion []*ContainerVersion `json:"containerVersion,omitempty"`

	// ContainerVersionHeader: All container version headers of a GTM
	// Container.
	ContainerVersionHeader []*ContainerVersionHeader `json:"containerVersionHeader,omitempty"`
}

type ListContainersResponse struct {
	// Containers: All Containers of a GTM Account.
	Containers []*Container `json:"containers,omitempty"`
}

type ListMacrosResponse struct {
	// Macros: All GTM Macros of a GTM Container.
	Macros []*Macro `json:"macros,omitempty"`
}

type ListRulesResponse struct {
	// Rules: All GTM Rules of a GTM Container.
	Rules []*Rule `json:"rules,omitempty"`
}

type ListTagsResponse struct {
	// Tags: All GTM Tags of a GTM Container.
	Tags []*Tag `json:"tags,omitempty"`
}

type ListTriggersResponse struct {
	// Triggers: All GTM Triggers of a GTM Container.
	Triggers []*Trigger `json:"triggers,omitempty"`
}

type ListVariablesResponse struct {
	// Variables: All GTM Variables of a GTM Container.
	Variables []*Variable `json:"variables,omitempty"`
}

type Macro struct {
	// AccountId: GTM Account ID.
	AccountId string `json:"accountId,omitempty"`

	// ContainerId: GTM Container ID.
	ContainerId string `json:"containerId,omitempty"`

	// DisablingRuleId: For mobile containers only: A list of rule IDs for
	// disabling conditional macros; the macro is enabled if one of the
	// enabling rules is true while all the disabling rules are false.
	// Treated as an unordered set.
	DisablingRuleId []string `json:"disablingRuleId,omitempty"`

	// EnablingRuleId: For mobile containers only: A list of rule IDs for
	// enabling conditional macros; the macro is enabled if one of the
	// enabling rules is true while all the disabling rules are false.
	// Treated as an unordered set.
	EnablingRuleId []string `json:"enablingRuleId,omitempty"`

	// Fingerprint: The fingerprint of the GTM Macro as computed at storage
	// time. This value is recomputed whenever the macro is modified.
	Fingerprint string `json:"fingerprint,omitempty"`

	// MacroId: The Macro ID uniquely identifies the GTM Macro.
	MacroId string `json:"macroId,omitempty"`

	// Name: Macro display name.
	Name string `json:"name,omitempty"`

	// Notes: User notes on how to apply this macro in the container.
	Notes string `json:"notes,omitempty"`

	// Parameter: The macro's parameters.
	Parameter []*Parameter `json:"parameter,omitempty"`

	// ScheduleEndMs: The end timestamp in milliseconds to schedule a macro.
	ScheduleEndMs int64 `json:"scheduleEndMs,omitempty,string"`

	// ScheduleStartMs: The start timestamp in milliseconds to schedule a
	// macro.
	ScheduleStartMs int64 `json:"scheduleStartMs,omitempty,string"`

	// Type: GTM Macro Type.
	Type string `json:"type,omitempty"`
}

type Parameter struct {
	// Key: The named key that uniquely identifies a parameter. Required for
	// top-level parameters, as well as map values. Ignored for list values.
	Key string `json:"key,omitempty"`

	// List: This list parameter's parameters (keys will be ignored).
	List []*Parameter `json:"list,omitempty"`

	// Map: This map parameter's parameters (must have keys; keys must be
	// unique).
	Map []*Parameter `json:"map,omitempty"`

	// Type: The parameter type. Valid values are:
	// - boolean: The value
	// represents a boolean, represented as 'true' or 'false'
	// - integer:
	// The value represents a 64-bit signed integer value, in base 10
	// -
	// list: A list of parameters should be specified
	// - map: A map of
	// parameters should be specified
	// - template: The value represents any
	// text; this can include macro references (even macro references that
	// might return non-string types)
	Type string `json:"type,omitempty"`

	// Value: A parameter's value (may contain macro references such as
	// "{{myMacro}}") as appropriate to the specified type.
	Value string `json:"value,omitempty"`
}

type PublishContainerVersionResponse struct {
	// CompilerError: Compiler errors or not.
	CompilerError bool `json:"compilerError,omitempty"`

	// ContainerVersion: The container version created.
	ContainerVersion *ContainerVersion `json:"containerVersion,omitempty"`
}

type Rule struct {
	// AccountId: GTM Account ID.
	AccountId string `json:"accountId,omitempty"`

	// Condition: The list of conditions that make up this rule (implicit
	// AND between them).
	Condition []*Condition `json:"condition,omitempty"`

	// ContainerId: GTM Container ID.
	ContainerId string `json:"containerId,omitempty"`

	// Fingerprint: The fingerprint of the GTM Rule as computed at storage
	// time. This value is recomputed whenever the rule is modified.
	Fingerprint string `json:"fingerprint,omitempty"`

	// Name: Rule display name.
	Name string `json:"name,omitempty"`

	// Notes: User notes on how to apply this rule in the container.
	Notes string `json:"notes,omitempty"`

	// RuleId: The Rule ID uniquely identifies the GTM Rule.
	RuleId string `json:"ruleId,omitempty"`
}

type Tag struct {
	// AccountId: GTM Account ID.
	AccountId string `json:"accountId,omitempty"`

	// BlockingRuleId: Blocking rule IDs. If any of the listed rules
	// evaluate to true, the tag will not fire.
	BlockingRuleId []string `json:"blockingRuleId,omitempty"`

	// BlockingTriggerId: Blocking trigger IDs. If any of the listed
	// triggers evaluate to true, the tag will not fire.
	BlockingTriggerId []string `json:"blockingTriggerId,omitempty"`

	// ContainerId: GTM Container ID.
	ContainerId string `json:"containerId,omitempty"`

	// Dependencies: An optional list of tag names that this tag depends on
	// to fire. Execution of this tag will be prevented until the tags with
	// the given names complete their execution.
	Dependencies *Parameter `json:"dependencies,omitempty"`

	// Fingerprint: The fingerprint of the GTM Tag as computed at storage
	// time. This value is recomputed whenever the tag is modified.
	Fingerprint string `json:"fingerprint,omitempty"`

	// FiringRuleId: Firing rule IDs. A tag will fire when any of the listed
	// rules are true and all of its blockingRuleIds (if any specified) are
	// false.
	FiringRuleId []string `json:"firingRuleId,omitempty"`

	// FiringTriggerId: Firing trigger IDs. A tag will fire when any of the
	// listed triggers are true and all of its blockingTriggerIds (if any
	// specified) are false.
	FiringTriggerId []string `json:"firingTriggerId,omitempty"`

	// LiveOnly: If set to true, this tag will only fire in the live
	// environment (e.g. not in preview or debug mode).
	LiveOnly bool `json:"liveOnly,omitempty"`

	// Name: Tag display name.
	Name string `json:"name,omitempty"`

	// Notes: User notes on how to apply this tag in the container.
	Notes string `json:"notes,omitempty"`

	// Parameter: The tag's parameters.
	Parameter []*Parameter `json:"parameter,omitempty"`

	// Priority: User defined numeric priority of the tag. Tags are fired
	// asynchronously in order of priority. Tags with higher numeric value
	// fire first. A tag's priority can be a positive or negative value. The
	// default value is 0.
	Priority *Parameter `json:"priority,omitempty"`

	// ScheduleEndMs: The end timestamp in milliseconds to schedule a tag.
	ScheduleEndMs int64 `json:"scheduleEndMs,omitempty,string"`

	// ScheduleStartMs: The start timestamp in milliseconds to schedule a
	// tag.
	ScheduleStartMs int64 `json:"scheduleStartMs,omitempty,string"`

	// TagId: The Tag ID uniquely identifies the GTM Tag.
	TagId string `json:"tagId,omitempty"`

	// Type: GTM Tag Type.
	Type string `json:"type,omitempty"`
}

type Trigger struct {
	// AccountId: GTM Account ID.
	AccountId string `json:"accountId,omitempty"`

	// AutoEventFilter: Used in the case of auto event tracking.
	AutoEventFilter []*Condition `json:"autoEventFilter,omitempty"`

	// CheckValidation: Whether or not we should only fire tags if the form
	// submit or link click event is not cancelled by some other event
	// handler (e.g. because of validation). Only valid for Form Submission
	// and Link Click triggers.
	CheckValidation *Parameter `json:"checkValidation,omitempty"`

	// ContainerId: GTM Container ID.
	ContainerId string `json:"containerId,omitempty"`

	// CustomEventFilter: Used in the case of custom event, which is fired
	// iff all Conditions are true.
	CustomEventFilter []*Condition `json:"customEventFilter,omitempty"`

	// EnableAllVideos: Reloads the videos in the page that don't already
	// have the YT API enabled. If false, only capture events from videos
	// that already have the API enabled. Only valid for YouTube triggers.
	EnableAllVideos *Parameter `json:"enableAllVideos,omitempty"`

	// EventName: Name of the GTM event that is fired. Only valid for Timer
	// triggers.
	EventName *Parameter `json:"eventName,omitempty"`

	// Filter: The trigger will only fire iff all Conditions are true.
	Filter []*Condition `json:"filter,omitempty"`

	// Fingerprint: The fingerprint of the GTM Trigger as computed at
	// storage time. This value is recomputed whenever the trigger is
	// modified.
	Fingerprint string `json:"fingerprint,omitempty"`

	// Interval: Time between triggering recurring Timer Events (in
	// milliseconds). Only valid for Timer triggers.
	Interval *Parameter `json:"interval,omitempty"`

	// Limit: Limit of the number of GTM events this Timer Trigger will
	// fire. If no limit is set, we will continue to fire GTM events until
	// the user leaves the page. Only valid for Timer triggers.
	Limit *Parameter `json:"limit,omitempty"`

	// Name: Trigger display name.
	Name string `json:"name,omitempty"`

	// TriggerId: The Trigger ID uniquely identifies the GTM Trigger.
	TriggerId string `json:"triggerId,omitempty"`

	// Type: Defines the data layer event that causes this trigger.
	Type string `json:"type,omitempty"`

	// UniqueTriggerId: Globally unique id of the trigger that
	// auto-generates this Form Submit or Link Click listeners if any. Used
	// to make incompatible auto-events work together with trigger filtering
	// based on trigger ids. This value is populated during output
	// generation since the tags implied by triggers don't exist until then.
	// Only valid for Form Submission and Link Click triggers.
	UniqueTriggerId *Parameter `json:"uniqueTriggerId,omitempty"`

	// VideoPercentageList: List of integer percentage values. The trigger
	// will fire as each percentage is reached in any instrumented videos.
	// Only valid for YouTube triggers.
	VideoPercentageList *Parameter `json:"videoPercentageList,omitempty"`

	// WaitForTags: Whether or not we should delay the form submissions or
	// link opening until all of the tags have fired (by preventing the
	// default action and later simulating the default action). Only valid
	// for Form Submission and Link Click triggers.
	WaitForTags *Parameter `json:"waitForTags,omitempty"`

	// WaitForTagsTimeout: How long to wait (in milliseconds) for tags to
	// fire when 'waits_for_tags' above evaluates to true. Only valid for
	// Form Submission and Link Click triggers.
	WaitForTagsTimeout *Parameter `json:"waitForTagsTimeout,omitempty"`
}

type UserAccess struct {
	// AccountAccess: GTM Account access permissions.
	AccountAccess *AccountAccess `json:"accountAccess,omitempty"`

	// AccountId: GTM Account ID.
	AccountId string `json:"accountId,omitempty"`

	// ContainerAccess: GTM Container access permissions.
	ContainerAccess []*ContainerAccess `json:"containerAccess,omitempty"`

	// EmailAddress: User's email address.
	EmailAddress string `json:"emailAddress,omitempty"`

	// PermissionId: Account Permission ID.
	PermissionId string `json:"permissionId,omitempty"`
}

type Variable struct {
	// AccountId: GTM Account ID.
	AccountId string `json:"accountId,omitempty"`

	// ContainerId: GTM Container ID.
	ContainerId string `json:"containerId,omitempty"`

	// DisablingTriggerId: For mobile containers only: A list of trigger IDs
	// for disabling conditional variables; the variable is enabled if one
	// of the enabling trigger is true while all the disabling trigger are
	// false. Treated as an unordered set.
	DisablingTriggerId []string `json:"disablingTriggerId,omitempty"`

	// EnablingTriggerId: For mobile containers only: A list of trigger IDs
	// for enabling conditional variables; the variable is enabled if one of
	// the enabling triggers is true while all the disabling triggers are
	// false. Treated as an unordered set.
	EnablingTriggerId []string `json:"enablingTriggerId,omitempty"`

	// Fingerprint: The fingerprint of the GTM Variable as computed at
	// storage time. This value is recomputed whenever the variable is
	// modified.
	Fingerprint string `json:"fingerprint,omitempty"`

	// Name: Variable display name.
	Name string `json:"name,omitempty"`

	// Notes: User notes on how to apply this variable in the container.
	Notes string `json:"notes,omitempty"`

	// Parameter: The variable's parameters.
	Parameter []*Parameter `json:"parameter,omitempty"`

	// ScheduleEndMs: The end timestamp in milliseconds to schedule a
	// variable.
	ScheduleEndMs int64 `json:"scheduleEndMs,omitempty,string"`

	// ScheduleStartMs: The start timestamp in milliseconds to schedule a
	// variable.
	ScheduleStartMs int64 `json:"scheduleStartMs,omitempty,string"`

	// Type: GTM Variable Type.
	Type string `json:"type,omitempty"`

	// VariableId: The Variable ID uniquely identifies the GTM Variable.
	VariableId string `json:"variableId,omitempty"`
}

// method id "tagmanager.accounts.get":

type AccountsGetCall struct {
	s         *Service
	accountId string
	opt_      map[string]interface{}
}

// Get: Gets a GTM Account.
func (r *AccountsService) Get(accountId string) *AccountsGetCall {
	c := &AccountsGetCall{s: r.s, opt_: make(map[string]interface{})}
	c.accountId = accountId
	return c
}

// Fields allows partial responses to be retrieved.
// See https://developers.google.com/gdata/docs/2.0/basics#PartialResponse
// for more information.
func (c *AccountsGetCall) Fields(s ...googleapi.Field) *AccountsGetCall {
	c.opt_["fields"] = googleapi.CombineFields(s)
	return c
}

func (c *AccountsGetCall) Do() (*Account, error) {
	var body io.Reader = nil
	params := make(url.Values)
	params.Set("alt", "json")
	if v, ok := c.opt_["fields"]; ok {
		params.Set("fields", fmt.Sprintf("%v", v))
	}
	urls := googleapi.ResolveRelative(c.s.BasePath, "accounts/{accountId}")
	urls += "?" + params.Encode()
	req, _ := http.NewRequest("GET", urls, body)
	googleapi.Expand(req.URL, map[string]string{
		"accountId": c.accountId,
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
	var ret *Account
	if err := json.NewDecoder(res.Body).Decode(&ret); err != nil {
		return nil, err
	}
	return ret, nil
	// {
	//   "description": "Gets a GTM Account.",
	//   "httpMethod": "GET",
	//   "id": "tagmanager.accounts.get",
	//   "parameterOrder": [
	//     "accountId"
	//   ],
	//   "parameters": {
	//     "accountId": {
	//       "description": "The GTM Account ID.",
	//       "location": "path",
	//       "required": true,
	//       "type": "string"
	//     }
	//   },
	//   "path": "accounts/{accountId}",
	//   "response": {
	//     "$ref": "Account"
	//   },
	//   "scopes": [
	//     "https://www.googleapis.com/auth/tagmanager.edit.containers",
	//     "https://www.googleapis.com/auth/tagmanager.manage.accounts",
	//     "https://www.googleapis.com/auth/tagmanager.readonly"
	//   ]
	// }

}

// method id "tagmanager.accounts.list":

type AccountsListCall struct {
	s    *Service
	opt_ map[string]interface{}
}

// List: Lists all GTM Accounts that a user has access to.
func (r *AccountsService) List() *AccountsListCall {
	c := &AccountsListCall{s: r.s, opt_: make(map[string]interface{})}
	return c
}

// Fields allows partial responses to be retrieved.
// See https://developers.google.com/gdata/docs/2.0/basics#PartialResponse
// for more information.
func (c *AccountsListCall) Fields(s ...googleapi.Field) *AccountsListCall {
	c.opt_["fields"] = googleapi.CombineFields(s)
	return c
}

func (c *AccountsListCall) Do() (*ListAccountsResponse, error) {
	var body io.Reader = nil
	params := make(url.Values)
	params.Set("alt", "json")
	if v, ok := c.opt_["fields"]; ok {
		params.Set("fields", fmt.Sprintf("%v", v))
	}
	urls := googleapi.ResolveRelative(c.s.BasePath, "accounts")
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
	var ret *ListAccountsResponse
	if err := json.NewDecoder(res.Body).Decode(&ret); err != nil {
		return nil, err
	}
	return ret, nil
	// {
	//   "description": "Lists all GTM Accounts that a user has access to.",
	//   "httpMethod": "GET",
	//   "id": "tagmanager.accounts.list",
	//   "path": "accounts",
	//   "response": {
	//     "$ref": "ListAccountsResponse"
	//   },
	//   "scopes": [
	//     "https://www.googleapis.com/auth/tagmanager.edit.containers",
	//     "https://www.googleapis.com/auth/tagmanager.manage.accounts",
	//     "https://www.googleapis.com/auth/tagmanager.readonly"
	//   ]
	// }

}

// method id "tagmanager.accounts.update":

type AccountsUpdateCall struct {
	s         *Service
	accountId string
	account   *Account
	opt_      map[string]interface{}
}

// Update: Updates a GTM Account.
func (r *AccountsService) Update(accountId string, account *Account) *AccountsUpdateCall {
	c := &AccountsUpdateCall{s: r.s, opt_: make(map[string]interface{})}
	c.accountId = accountId
	c.account = account
	return c
}

// Fingerprint sets the optional parameter "fingerprint": When provided,
// this fingerprint must match the fingerprint of the account in
// storage.
func (c *AccountsUpdateCall) Fingerprint(fingerprint string) *AccountsUpdateCall {
	c.opt_["fingerprint"] = fingerprint
	return c
}

// Fields allows partial responses to be retrieved.
// See https://developers.google.com/gdata/docs/2.0/basics#PartialResponse
// for more information.
func (c *AccountsUpdateCall) Fields(s ...googleapi.Field) *AccountsUpdateCall {
	c.opt_["fields"] = googleapi.CombineFields(s)
	return c
}

func (c *AccountsUpdateCall) Do() (*Account, error) {
	var body io.Reader = nil
	body, err := googleapi.WithoutDataWrapper.JSONReader(c.account)
	if err != nil {
		return nil, err
	}
	ctype := "application/json"
	params := make(url.Values)
	params.Set("alt", "json")
	if v, ok := c.opt_["fingerprint"]; ok {
		params.Set("fingerprint", fmt.Sprintf("%v", v))
	}
	if v, ok := c.opt_["fields"]; ok {
		params.Set("fields", fmt.Sprintf("%v", v))
	}
	urls := googleapi.ResolveRelative(c.s.BasePath, "accounts/{accountId}")
	urls += "?" + params.Encode()
	req, _ := http.NewRequest("PUT", urls, body)
	googleapi.Expand(req.URL, map[string]string{
		"accountId": c.accountId,
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
	var ret *Account
	if err := json.NewDecoder(res.Body).Decode(&ret); err != nil {
		return nil, err
	}
	return ret, nil
	// {
	//   "description": "Updates a GTM Account.",
	//   "httpMethod": "PUT",
	//   "id": "tagmanager.accounts.update",
	//   "parameterOrder": [
	//     "accountId"
	//   ],
	//   "parameters": {
	//     "accountId": {
	//       "description": "The GTM Account ID.",
	//       "location": "path",
	//       "required": true,
	//       "type": "string"
	//     },
	//     "fingerprint": {
	//       "description": "When provided, this fingerprint must match the fingerprint of the account in storage.",
	//       "location": "query",
	//       "type": "string"
	//     }
	//   },
	//   "path": "accounts/{accountId}",
	//   "request": {
	//     "$ref": "Account"
	//   },
	//   "response": {
	//     "$ref": "Account"
	//   },
	//   "scopes": [
	//     "https://www.googleapis.com/auth/tagmanager.manage.accounts"
	//   ]
	// }

}

// method id "tagmanager.accounts.containers.create":

type AccountsContainersCreateCall struct {
	s         *Service
	accountId string
	container *Container
	opt_      map[string]interface{}
}

// Create: Creates a Container.
func (r *AccountsContainersService) Create(accountId string, container *Container) *AccountsContainersCreateCall {
	c := &AccountsContainersCreateCall{s: r.s, opt_: make(map[string]interface{})}
	c.accountId = accountId
	c.container = container
	return c
}

// Fields allows partial responses to be retrieved.
// See https://developers.google.com/gdata/docs/2.0/basics#PartialResponse
// for more information.
func (c *AccountsContainersCreateCall) Fields(s ...googleapi.Field) *AccountsContainersCreateCall {
	c.opt_["fields"] = googleapi.CombineFields(s)
	return c
}

func (c *AccountsContainersCreateCall) Do() (*Container, error) {
	var body io.Reader = nil
	body, err := googleapi.WithoutDataWrapper.JSONReader(c.container)
	if err != nil {
		return nil, err
	}
	ctype := "application/json"
	params := make(url.Values)
	params.Set("alt", "json")
	if v, ok := c.opt_["fields"]; ok {
		params.Set("fields", fmt.Sprintf("%v", v))
	}
	urls := googleapi.ResolveRelative(c.s.BasePath, "accounts/{accountId}/containers")
	urls += "?" + params.Encode()
	req, _ := http.NewRequest("POST", urls, body)
	googleapi.Expand(req.URL, map[string]string{
		"accountId": c.accountId,
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
	var ret *Container
	if err := json.NewDecoder(res.Body).Decode(&ret); err != nil {
		return nil, err
	}
	return ret, nil
	// {
	//   "description": "Creates a Container.",
	//   "httpMethod": "POST",
	//   "id": "tagmanager.accounts.containers.create",
	//   "parameterOrder": [
	//     "accountId"
	//   ],
	//   "parameters": {
	//     "accountId": {
	//       "description": "The GTM Account ID.",
	//       "location": "path",
	//       "required": true,
	//       "type": "string"
	//     }
	//   },
	//   "path": "accounts/{accountId}/containers",
	//   "request": {
	//     "$ref": "Container"
	//   },
	//   "response": {
	//     "$ref": "Container"
	//   },
	//   "scopes": [
	//     "https://www.googleapis.com/auth/tagmanager.edit.containers"
	//   ]
	// }

}

// method id "tagmanager.accounts.containers.delete":

type AccountsContainersDeleteCall struct {
	s           *Service
	accountId   string
	containerId string
	opt_        map[string]interface{}
}

// Delete: Deletes a Container.
func (r *AccountsContainersService) Delete(accountId string, containerId string) *AccountsContainersDeleteCall {
	c := &AccountsContainersDeleteCall{s: r.s, opt_: make(map[string]interface{})}
	c.accountId = accountId
	c.containerId = containerId
	return c
}

// Fields allows partial responses to be retrieved.
// See https://developers.google.com/gdata/docs/2.0/basics#PartialResponse
// for more information.
func (c *AccountsContainersDeleteCall) Fields(s ...googleapi.Field) *AccountsContainersDeleteCall {
	c.opt_["fields"] = googleapi.CombineFields(s)
	return c
}

func (c *AccountsContainersDeleteCall) Do() error {
	var body io.Reader = nil
	params := make(url.Values)
	params.Set("alt", "json")
	if v, ok := c.opt_["fields"]; ok {
		params.Set("fields", fmt.Sprintf("%v", v))
	}
	urls := googleapi.ResolveRelative(c.s.BasePath, "accounts/{accountId}/containers/{containerId}")
	urls += "?" + params.Encode()
	req, _ := http.NewRequest("DELETE", urls, body)
	googleapi.Expand(req.URL, map[string]string{
		"accountId":   c.accountId,
		"containerId": c.containerId,
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
	//   "description": "Deletes a Container.",
	//   "httpMethod": "DELETE",
	//   "id": "tagmanager.accounts.containers.delete",
	//   "parameterOrder": [
	//     "accountId",
	//     "containerId"
	//   ],
	//   "parameters": {
	//     "accountId": {
	//       "description": "The GTM Account ID.",
	//       "location": "path",
	//       "required": true,
	//       "type": "string"
	//     },
	//     "containerId": {
	//       "description": "The GTM Container ID.",
	//       "location": "path",
	//       "required": true,
	//       "type": "string"
	//     }
	//   },
	//   "path": "accounts/{accountId}/containers/{containerId}",
	//   "scopes": [
	//     "https://www.googleapis.com/auth/tagmanager.delete.containers"
	//   ]
	// }

}

// method id "tagmanager.accounts.containers.get":

type AccountsContainersGetCall struct {
	s           *Service
	accountId   string
	containerId string
	opt_        map[string]interface{}
}

// Get: Gets a Container.
func (r *AccountsContainersService) Get(accountId string, containerId string) *AccountsContainersGetCall {
	c := &AccountsContainersGetCall{s: r.s, opt_: make(map[string]interface{})}
	c.accountId = accountId
	c.containerId = containerId
	return c
}

// Fields allows partial responses to be retrieved.
// See https://developers.google.com/gdata/docs/2.0/basics#PartialResponse
// for more information.
func (c *AccountsContainersGetCall) Fields(s ...googleapi.Field) *AccountsContainersGetCall {
	c.opt_["fields"] = googleapi.CombineFields(s)
	return c
}

func (c *AccountsContainersGetCall) Do() (*Container, error) {
	var body io.Reader = nil
	params := make(url.Values)
	params.Set("alt", "json")
	if v, ok := c.opt_["fields"]; ok {
		params.Set("fields", fmt.Sprintf("%v", v))
	}
	urls := googleapi.ResolveRelative(c.s.BasePath, "accounts/{accountId}/containers/{containerId}")
	urls += "?" + params.Encode()
	req, _ := http.NewRequest("GET", urls, body)
	googleapi.Expand(req.URL, map[string]string{
		"accountId":   c.accountId,
		"containerId": c.containerId,
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
	var ret *Container
	if err := json.NewDecoder(res.Body).Decode(&ret); err != nil {
		return nil, err
	}
	return ret, nil
	// {
	//   "description": "Gets a Container.",
	//   "httpMethod": "GET",
	//   "id": "tagmanager.accounts.containers.get",
	//   "parameterOrder": [
	//     "accountId",
	//     "containerId"
	//   ],
	//   "parameters": {
	//     "accountId": {
	//       "description": "The GTM Account ID.",
	//       "location": "path",
	//       "required": true,
	//       "type": "string"
	//     },
	//     "containerId": {
	//       "description": "The GTM Container ID.",
	//       "location": "path",
	//       "required": true,
	//       "type": "string"
	//     }
	//   },
	//   "path": "accounts/{accountId}/containers/{containerId}",
	//   "response": {
	//     "$ref": "Container"
	//   },
	//   "scopes": [
	//     "https://www.googleapis.com/auth/tagmanager.edit.containers",
	//     "https://www.googleapis.com/auth/tagmanager.readonly"
	//   ]
	// }

}

// method id "tagmanager.accounts.containers.list":

type AccountsContainersListCall struct {
	s         *Service
	accountId string
	opt_      map[string]interface{}
}

// List: Lists all Containers that belongs to a GTM Account.
func (r *AccountsContainersService) List(accountId string) *AccountsContainersListCall {
	c := &AccountsContainersListCall{s: r.s, opt_: make(map[string]interface{})}
	c.accountId = accountId
	return c
}

// Fields allows partial responses to be retrieved.
// See https://developers.google.com/gdata/docs/2.0/basics#PartialResponse
// for more information.
func (c *AccountsContainersListCall) Fields(s ...googleapi.Field) *AccountsContainersListCall {
	c.opt_["fields"] = googleapi.CombineFields(s)
	return c
}

func (c *AccountsContainersListCall) Do() (*ListContainersResponse, error) {
	var body io.Reader = nil
	params := make(url.Values)
	params.Set("alt", "json")
	if v, ok := c.opt_["fields"]; ok {
		params.Set("fields", fmt.Sprintf("%v", v))
	}
	urls := googleapi.ResolveRelative(c.s.BasePath, "accounts/{accountId}/containers")
	urls += "?" + params.Encode()
	req, _ := http.NewRequest("GET", urls, body)
	googleapi.Expand(req.URL, map[string]string{
		"accountId": c.accountId,
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
	var ret *ListContainersResponse
	if err := json.NewDecoder(res.Body).Decode(&ret); err != nil {
		return nil, err
	}
	return ret, nil
	// {
	//   "description": "Lists all Containers that belongs to a GTM Account.",
	//   "httpMethod": "GET",
	//   "id": "tagmanager.accounts.containers.list",
	//   "parameterOrder": [
	//     "accountId"
	//   ],
	//   "parameters": {
	//     "accountId": {
	//       "description": "The GTM Account ID.",
	//       "location": "path",
	//       "required": true,
	//       "type": "string"
	//     }
	//   },
	//   "path": "accounts/{accountId}/containers",
	//   "response": {
	//     "$ref": "ListContainersResponse"
	//   },
	//   "scopes": [
	//     "https://www.googleapis.com/auth/tagmanager.edit.containers",
	//     "https://www.googleapis.com/auth/tagmanager.readonly"
	//   ]
	// }

}

// method id "tagmanager.accounts.containers.update":

type AccountsContainersUpdateCall struct {
	s           *Service
	accountId   string
	containerId string
	container   *Container
	opt_        map[string]interface{}
}

// Update: Updates a Container.
func (r *AccountsContainersService) Update(accountId string, containerId string, container *Container) *AccountsContainersUpdateCall {
	c := &AccountsContainersUpdateCall{s: r.s, opt_: make(map[string]interface{})}
	c.accountId = accountId
	c.containerId = containerId
	c.container = container
	return c
}

// Fingerprint sets the optional parameter "fingerprint": When provided,
// this fingerprint must match the fingerprint of the container in
// storage.
func (c *AccountsContainersUpdateCall) Fingerprint(fingerprint string) *AccountsContainersUpdateCall {
	c.opt_["fingerprint"] = fingerprint
	return c
}

// Fields allows partial responses to be retrieved.
// See https://developers.google.com/gdata/docs/2.0/basics#PartialResponse
// for more information.
func (c *AccountsContainersUpdateCall) Fields(s ...googleapi.Field) *AccountsContainersUpdateCall {
	c.opt_["fields"] = googleapi.CombineFields(s)
	return c
}

func (c *AccountsContainersUpdateCall) Do() (*Container, error) {
	var body io.Reader = nil
	body, err := googleapi.WithoutDataWrapper.JSONReader(c.container)
	if err != nil {
		return nil, err
	}
	ctype := "application/json"
	params := make(url.Values)
	params.Set("alt", "json")
	if v, ok := c.opt_["fingerprint"]; ok {
		params.Set("fingerprint", fmt.Sprintf("%v", v))
	}
	if v, ok := c.opt_["fields"]; ok {
		params.Set("fields", fmt.Sprintf("%v", v))
	}
	urls := googleapi.ResolveRelative(c.s.BasePath, "accounts/{accountId}/containers/{containerId}")
	urls += "?" + params.Encode()
	req, _ := http.NewRequest("PUT", urls, body)
	googleapi.Expand(req.URL, map[string]string{
		"accountId":   c.accountId,
		"containerId": c.containerId,
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
	var ret *Container
	if err := json.NewDecoder(res.Body).Decode(&ret); err != nil {
		return nil, err
	}
	return ret, nil
	// {
	//   "description": "Updates a Container.",
	//   "httpMethod": "PUT",
	//   "id": "tagmanager.accounts.containers.update",
	//   "parameterOrder": [
	//     "accountId",
	//     "containerId"
	//   ],
	//   "parameters": {
	//     "accountId": {
	//       "description": "The GTM Account ID.",
	//       "location": "path",
	//       "required": true,
	//       "type": "string"
	//     },
	//     "containerId": {
	//       "description": "The GTM Container ID.",
	//       "location": "path",
	//       "required": true,
	//       "type": "string"
	//     },
	//     "fingerprint": {
	//       "description": "When provided, this fingerprint must match the fingerprint of the container in storage.",
	//       "location": "query",
	//       "type": "string"
	//     }
	//   },
	//   "path": "accounts/{accountId}/containers/{containerId}",
	//   "request": {
	//     "$ref": "Container"
	//   },
	//   "response": {
	//     "$ref": "Container"
	//   },
	//   "scopes": [
	//     "https://www.googleapis.com/auth/tagmanager.edit.containers"
	//   ]
	// }

}

// method id "tagmanager.accounts.containers.macros.create":

type AccountsContainersMacrosCreateCall struct {
	s           *Service
	accountId   string
	containerId string
	macro       *Macro
	opt_        map[string]interface{}
}

// Create: Creates a GTM Macro.
func (r *AccountsContainersMacrosService) Create(accountId string, containerId string, macro *Macro) *AccountsContainersMacrosCreateCall {
	c := &AccountsContainersMacrosCreateCall{s: r.s, opt_: make(map[string]interface{})}
	c.accountId = accountId
	c.containerId = containerId
	c.macro = macro
	return c
}

// Fields allows partial responses to be retrieved.
// See https://developers.google.com/gdata/docs/2.0/basics#PartialResponse
// for more information.
func (c *AccountsContainersMacrosCreateCall) Fields(s ...googleapi.Field) *AccountsContainersMacrosCreateCall {
	c.opt_["fields"] = googleapi.CombineFields(s)
	return c
}

func (c *AccountsContainersMacrosCreateCall) Do() (*Macro, error) {
	var body io.Reader = nil
	body, err := googleapi.WithoutDataWrapper.JSONReader(c.macro)
	if err != nil {
		return nil, err
	}
	ctype := "application/json"
	params := make(url.Values)
	params.Set("alt", "json")
	if v, ok := c.opt_["fields"]; ok {
		params.Set("fields", fmt.Sprintf("%v", v))
	}
	urls := googleapi.ResolveRelative(c.s.BasePath, "accounts/{accountId}/containers/{containerId}/macros")
	urls += "?" + params.Encode()
	req, _ := http.NewRequest("POST", urls, body)
	googleapi.Expand(req.URL, map[string]string{
		"accountId":   c.accountId,
		"containerId": c.containerId,
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
	var ret *Macro
	if err := json.NewDecoder(res.Body).Decode(&ret); err != nil {
		return nil, err
	}
	return ret, nil
	// {
	//   "description": "Creates a GTM Macro.",
	//   "httpMethod": "POST",
	//   "id": "tagmanager.accounts.containers.macros.create",
	//   "parameterOrder": [
	//     "accountId",
	//     "containerId"
	//   ],
	//   "parameters": {
	//     "accountId": {
	//       "description": "The GTM Account ID.",
	//       "location": "path",
	//       "required": true,
	//       "type": "string"
	//     },
	//     "containerId": {
	//       "description": "The GTM Container ID.",
	//       "location": "path",
	//       "required": true,
	//       "type": "string"
	//     }
	//   },
	//   "path": "accounts/{accountId}/containers/{containerId}/macros",
	//   "request": {
	//     "$ref": "Macro"
	//   },
	//   "response": {
	//     "$ref": "Macro"
	//   },
	//   "scopes": [
	//     "https://www.googleapis.com/auth/tagmanager.edit.containers"
	//   ]
	// }

}

// method id "tagmanager.accounts.containers.macros.delete":

type AccountsContainersMacrosDeleteCall struct {
	s           *Service
	accountId   string
	containerId string
	macroId     string
	opt_        map[string]interface{}
}

// Delete: Deletes a GTM Macro.
func (r *AccountsContainersMacrosService) Delete(accountId string, containerId string, macroId string) *AccountsContainersMacrosDeleteCall {
	c := &AccountsContainersMacrosDeleteCall{s: r.s, opt_: make(map[string]interface{})}
	c.accountId = accountId
	c.containerId = containerId
	c.macroId = macroId
	return c
}

// Fields allows partial responses to be retrieved.
// See https://developers.google.com/gdata/docs/2.0/basics#PartialResponse
// for more information.
func (c *AccountsContainersMacrosDeleteCall) Fields(s ...googleapi.Field) *AccountsContainersMacrosDeleteCall {
	c.opt_["fields"] = googleapi.CombineFields(s)
	return c
}

func (c *AccountsContainersMacrosDeleteCall) Do() error {
	var body io.Reader = nil
	params := make(url.Values)
	params.Set("alt", "json")
	if v, ok := c.opt_["fields"]; ok {
		params.Set("fields", fmt.Sprintf("%v", v))
	}
	urls := googleapi.ResolveRelative(c.s.BasePath, "accounts/{accountId}/containers/{containerId}/macros/{macroId}")
	urls += "?" + params.Encode()
	req, _ := http.NewRequest("DELETE", urls, body)
	googleapi.Expand(req.URL, map[string]string{
		"accountId":   c.accountId,
		"containerId": c.containerId,
		"macroId":     c.macroId,
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
	//   "description": "Deletes a GTM Macro.",
	//   "httpMethod": "DELETE",
	//   "id": "tagmanager.accounts.containers.macros.delete",
	//   "parameterOrder": [
	//     "accountId",
	//     "containerId",
	//     "macroId"
	//   ],
	//   "parameters": {
	//     "accountId": {
	//       "description": "The GTM Account ID.",
	//       "location": "path",
	//       "required": true,
	//       "type": "string"
	//     },
	//     "containerId": {
	//       "description": "The GTM Container ID.",
	//       "location": "path",
	//       "required": true,
	//       "type": "string"
	//     },
	//     "macroId": {
	//       "description": "The GTM Macro ID.",
	//       "location": "path",
	//       "required": true,
	//       "type": "string"
	//     }
	//   },
	//   "path": "accounts/{accountId}/containers/{containerId}/macros/{macroId}",
	//   "scopes": [
	//     "https://www.googleapis.com/auth/tagmanager.edit.containers"
	//   ]
	// }

}

// method id "tagmanager.accounts.containers.macros.get":

type AccountsContainersMacrosGetCall struct {
	s           *Service
	accountId   string
	containerId string
	macroId     string
	opt_        map[string]interface{}
}

// Get: Gets a GTM Macro.
func (r *AccountsContainersMacrosService) Get(accountId string, containerId string, macroId string) *AccountsContainersMacrosGetCall {
	c := &AccountsContainersMacrosGetCall{s: r.s, opt_: make(map[string]interface{})}
	c.accountId = accountId
	c.containerId = containerId
	c.macroId = macroId
	return c
}

// Fields allows partial responses to be retrieved.
// See https://developers.google.com/gdata/docs/2.0/basics#PartialResponse
// for more information.
func (c *AccountsContainersMacrosGetCall) Fields(s ...googleapi.Field) *AccountsContainersMacrosGetCall {
	c.opt_["fields"] = googleapi.CombineFields(s)
	return c
}

func (c *AccountsContainersMacrosGetCall) Do() (*Macro, error) {
	var body io.Reader = nil
	params := make(url.Values)
	params.Set("alt", "json")
	if v, ok := c.opt_["fields"]; ok {
		params.Set("fields", fmt.Sprintf("%v", v))
	}
	urls := googleapi.ResolveRelative(c.s.BasePath, "accounts/{accountId}/containers/{containerId}/macros/{macroId}")
	urls += "?" + params.Encode()
	req, _ := http.NewRequest("GET", urls, body)
	googleapi.Expand(req.URL, map[string]string{
		"accountId":   c.accountId,
		"containerId": c.containerId,
		"macroId":     c.macroId,
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
	var ret *Macro
	if err := json.NewDecoder(res.Body).Decode(&ret); err != nil {
		return nil, err
	}
	return ret, nil
	// {
	//   "description": "Gets a GTM Macro.",
	//   "httpMethod": "GET",
	//   "id": "tagmanager.accounts.containers.macros.get",
	//   "parameterOrder": [
	//     "accountId",
	//     "containerId",
	//     "macroId"
	//   ],
	//   "parameters": {
	//     "accountId": {
	//       "description": "The GTM Account ID.",
	//       "location": "path",
	//       "required": true,
	//       "type": "string"
	//     },
	//     "containerId": {
	//       "description": "The GTM Container ID.",
	//       "location": "path",
	//       "required": true,
	//       "type": "string"
	//     },
	//     "macroId": {
	//       "description": "The GTM Macro ID.",
	//       "location": "path",
	//       "required": true,
	//       "type": "string"
	//     }
	//   },
	//   "path": "accounts/{accountId}/containers/{containerId}/macros/{macroId}",
	//   "response": {
	//     "$ref": "Macro"
	//   },
	//   "scopes": [
	//     "https://www.googleapis.com/auth/tagmanager.edit.containers",
	//     "https://www.googleapis.com/auth/tagmanager.readonly"
	//   ]
	// }

}

// method id "tagmanager.accounts.containers.macros.list":

type AccountsContainersMacrosListCall struct {
	s           *Service
	accountId   string
	containerId string
	opt_        map[string]interface{}
}

// List: Lists all GTM Macros of a Container.
func (r *AccountsContainersMacrosService) List(accountId string, containerId string) *AccountsContainersMacrosListCall {
	c := &AccountsContainersMacrosListCall{s: r.s, opt_: make(map[string]interface{})}
	c.accountId = accountId
	c.containerId = containerId
	return c
}

// Fields allows partial responses to be retrieved.
// See https://developers.google.com/gdata/docs/2.0/basics#PartialResponse
// for more information.
func (c *AccountsContainersMacrosListCall) Fields(s ...googleapi.Field) *AccountsContainersMacrosListCall {
	c.opt_["fields"] = googleapi.CombineFields(s)
	return c
}

func (c *AccountsContainersMacrosListCall) Do() (*ListMacrosResponse, error) {
	var body io.Reader = nil
	params := make(url.Values)
	params.Set("alt", "json")
	if v, ok := c.opt_["fields"]; ok {
		params.Set("fields", fmt.Sprintf("%v", v))
	}
	urls := googleapi.ResolveRelative(c.s.BasePath, "accounts/{accountId}/containers/{containerId}/macros")
	urls += "?" + params.Encode()
	req, _ := http.NewRequest("GET", urls, body)
	googleapi.Expand(req.URL, map[string]string{
		"accountId":   c.accountId,
		"containerId": c.containerId,
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
	var ret *ListMacrosResponse
	if err := json.NewDecoder(res.Body).Decode(&ret); err != nil {
		return nil, err
	}
	return ret, nil
	// {
	//   "description": "Lists all GTM Macros of a Container.",
	//   "httpMethod": "GET",
	//   "id": "tagmanager.accounts.containers.macros.list",
	//   "parameterOrder": [
	//     "accountId",
	//     "containerId"
	//   ],
	//   "parameters": {
	//     "accountId": {
	//       "description": "The GTM Account ID.",
	//       "location": "path",
	//       "required": true,
	//       "type": "string"
	//     },
	//     "containerId": {
	//       "description": "The GTM Container ID.",
	//       "location": "path",
	//       "required": true,
	//       "type": "string"
	//     }
	//   },
	//   "path": "accounts/{accountId}/containers/{containerId}/macros",
	//   "response": {
	//     "$ref": "ListMacrosResponse"
	//   },
	//   "scopes": [
	//     "https://www.googleapis.com/auth/tagmanager.edit.containers",
	//     "https://www.googleapis.com/auth/tagmanager.readonly"
	//   ]
	// }

}

// method id "tagmanager.accounts.containers.macros.update":

type AccountsContainersMacrosUpdateCall struct {
	s           *Service
	accountId   string
	containerId string
	macroId     string
	macro       *Macro
	opt_        map[string]interface{}
}

// Update: Updates a GTM Macro.
func (r *AccountsContainersMacrosService) Update(accountId string, containerId string, macroId string, macro *Macro) *AccountsContainersMacrosUpdateCall {
	c := &AccountsContainersMacrosUpdateCall{s: r.s, opt_: make(map[string]interface{})}
	c.accountId = accountId
	c.containerId = containerId
	c.macroId = macroId
	c.macro = macro
	return c
}

// Fingerprint sets the optional parameter "fingerprint": When provided,
// this fingerprint must match the fingerprint of the macro in storage.
func (c *AccountsContainersMacrosUpdateCall) Fingerprint(fingerprint string) *AccountsContainersMacrosUpdateCall {
	c.opt_["fingerprint"] = fingerprint
	return c
}

// Fields allows partial responses to be retrieved.
// See https://developers.google.com/gdata/docs/2.0/basics#PartialResponse
// for more information.
func (c *AccountsContainersMacrosUpdateCall) Fields(s ...googleapi.Field) *AccountsContainersMacrosUpdateCall {
	c.opt_["fields"] = googleapi.CombineFields(s)
	return c
}

func (c *AccountsContainersMacrosUpdateCall) Do() (*Macro, error) {
	var body io.Reader = nil
	body, err := googleapi.WithoutDataWrapper.JSONReader(c.macro)
	if err != nil {
		return nil, err
	}
	ctype := "application/json"
	params := make(url.Values)
	params.Set("alt", "json")
	if v, ok := c.opt_["fingerprint"]; ok {
		params.Set("fingerprint", fmt.Sprintf("%v", v))
	}
	if v, ok := c.opt_["fields"]; ok {
		params.Set("fields", fmt.Sprintf("%v", v))
	}
	urls := googleapi.ResolveRelative(c.s.BasePath, "accounts/{accountId}/containers/{containerId}/macros/{macroId}")
	urls += "?" + params.Encode()
	req, _ := http.NewRequest("PUT", urls, body)
	googleapi.Expand(req.URL, map[string]string{
		"accountId":   c.accountId,
		"containerId": c.containerId,
		"macroId":     c.macroId,
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
	var ret *Macro
	if err := json.NewDecoder(res.Body).Decode(&ret); err != nil {
		return nil, err
	}
	return ret, nil
	// {
	//   "description": "Updates a GTM Macro.",
	//   "httpMethod": "PUT",
	//   "id": "tagmanager.accounts.containers.macros.update",
	//   "parameterOrder": [
	//     "accountId",
	//     "containerId",
	//     "macroId"
	//   ],
	//   "parameters": {
	//     "accountId": {
	//       "description": "The GTM Account ID.",
	//       "location": "path",
	//       "required": true,
	//       "type": "string"
	//     },
	//     "containerId": {
	//       "description": "The GTM Container ID.",
	//       "location": "path",
	//       "required": true,
	//       "type": "string"
	//     },
	//     "fingerprint": {
	//       "description": "When provided, this fingerprint must match the fingerprint of the macro in storage.",
	//       "location": "query",
	//       "type": "string"
	//     },
	//     "macroId": {
	//       "description": "The GTM Macro ID.",
	//       "location": "path",
	//       "required": true,
	//       "type": "string"
	//     }
	//   },
	//   "path": "accounts/{accountId}/containers/{containerId}/macros/{macroId}",
	//   "request": {
	//     "$ref": "Macro"
	//   },
	//   "response": {
	//     "$ref": "Macro"
	//   },
	//   "scopes": [
	//     "https://www.googleapis.com/auth/tagmanager.edit.containers"
	//   ]
	// }

}

// method id "tagmanager.accounts.containers.rules.create":

type AccountsContainersRulesCreateCall struct {
	s           *Service
	accountId   string
	containerId string
	rule        *Rule
	opt_        map[string]interface{}
}

// Create: Creates a GTM Rule.
func (r *AccountsContainersRulesService) Create(accountId string, containerId string, rule *Rule) *AccountsContainersRulesCreateCall {
	c := &AccountsContainersRulesCreateCall{s: r.s, opt_: make(map[string]interface{})}
	c.accountId = accountId
	c.containerId = containerId
	c.rule = rule
	return c
}

// Fields allows partial responses to be retrieved.
// See https://developers.google.com/gdata/docs/2.0/basics#PartialResponse
// for more information.
func (c *AccountsContainersRulesCreateCall) Fields(s ...googleapi.Field) *AccountsContainersRulesCreateCall {
	c.opt_["fields"] = googleapi.CombineFields(s)
	return c
}

func (c *AccountsContainersRulesCreateCall) Do() (*Rule, error) {
	var body io.Reader = nil
	body, err := googleapi.WithoutDataWrapper.JSONReader(c.rule)
	if err != nil {
		return nil, err
	}
	ctype := "application/json"
	params := make(url.Values)
	params.Set("alt", "json")
	if v, ok := c.opt_["fields"]; ok {
		params.Set("fields", fmt.Sprintf("%v", v))
	}
	urls := googleapi.ResolveRelative(c.s.BasePath, "accounts/{accountId}/containers/{containerId}/rules")
	urls += "?" + params.Encode()
	req, _ := http.NewRequest("POST", urls, body)
	googleapi.Expand(req.URL, map[string]string{
		"accountId":   c.accountId,
		"containerId": c.containerId,
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
	var ret *Rule
	if err := json.NewDecoder(res.Body).Decode(&ret); err != nil {
		return nil, err
	}
	return ret, nil
	// {
	//   "description": "Creates a GTM Rule.",
	//   "httpMethod": "POST",
	//   "id": "tagmanager.accounts.containers.rules.create",
	//   "parameterOrder": [
	//     "accountId",
	//     "containerId"
	//   ],
	//   "parameters": {
	//     "accountId": {
	//       "description": "The GTM Account ID.",
	//       "location": "path",
	//       "required": true,
	//       "type": "string"
	//     },
	//     "containerId": {
	//       "description": "The GTM Container ID.",
	//       "location": "path",
	//       "required": true,
	//       "type": "string"
	//     }
	//   },
	//   "path": "accounts/{accountId}/containers/{containerId}/rules",
	//   "request": {
	//     "$ref": "Rule"
	//   },
	//   "response": {
	//     "$ref": "Rule"
	//   },
	//   "scopes": [
	//     "https://www.googleapis.com/auth/tagmanager.edit.containers"
	//   ]
	// }

}

// method id "tagmanager.accounts.containers.rules.delete":

type AccountsContainersRulesDeleteCall struct {
	s           *Service
	accountId   string
	containerId string
	ruleId      string
	opt_        map[string]interface{}
}

// Delete: Deletes a GTM Rule.
func (r *AccountsContainersRulesService) Delete(accountId string, containerId string, ruleId string) *AccountsContainersRulesDeleteCall {
	c := &AccountsContainersRulesDeleteCall{s: r.s, opt_: make(map[string]interface{})}
	c.accountId = accountId
	c.containerId = containerId
	c.ruleId = ruleId
	return c
}

// Fields allows partial responses to be retrieved.
// See https://developers.google.com/gdata/docs/2.0/basics#PartialResponse
// for more information.
func (c *AccountsContainersRulesDeleteCall) Fields(s ...googleapi.Field) *AccountsContainersRulesDeleteCall {
	c.opt_["fields"] = googleapi.CombineFields(s)
	return c
}

func (c *AccountsContainersRulesDeleteCall) Do() error {
	var body io.Reader = nil
	params := make(url.Values)
	params.Set("alt", "json")
	if v, ok := c.opt_["fields"]; ok {
		params.Set("fields", fmt.Sprintf("%v", v))
	}
	urls := googleapi.ResolveRelative(c.s.BasePath, "accounts/{accountId}/containers/{containerId}/rules/{ruleId}")
	urls += "?" + params.Encode()
	req, _ := http.NewRequest("DELETE", urls, body)
	googleapi.Expand(req.URL, map[string]string{
		"accountId":   c.accountId,
		"containerId": c.containerId,
		"ruleId":      c.ruleId,
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
	//   "description": "Deletes a GTM Rule.",
	//   "httpMethod": "DELETE",
	//   "id": "tagmanager.accounts.containers.rules.delete",
	//   "parameterOrder": [
	//     "accountId",
	//     "containerId",
	//     "ruleId"
	//   ],
	//   "parameters": {
	//     "accountId": {
	//       "description": "The GTM Account ID.",
	//       "location": "path",
	//       "required": true,
	//       "type": "string"
	//     },
	//     "containerId": {
	//       "description": "The GTM Container ID.",
	//       "location": "path",
	//       "required": true,
	//       "type": "string"
	//     },
	//     "ruleId": {
	//       "description": "The GTM Rule ID.",
	//       "location": "path",
	//       "required": true,
	//       "type": "string"
	//     }
	//   },
	//   "path": "accounts/{accountId}/containers/{containerId}/rules/{ruleId}",
	//   "scopes": [
	//     "https://www.googleapis.com/auth/tagmanager.edit.containers"
	//   ]
	// }

}

// method id "tagmanager.accounts.containers.rules.get":

type AccountsContainersRulesGetCall struct {
	s           *Service
	accountId   string
	containerId string
	ruleId      string
	opt_        map[string]interface{}
}

// Get: Gets a GTM Rule.
func (r *AccountsContainersRulesService) Get(accountId string, containerId string, ruleId string) *AccountsContainersRulesGetCall {
	c := &AccountsContainersRulesGetCall{s: r.s, opt_: make(map[string]interface{})}
	c.accountId = accountId
	c.containerId = containerId
	c.ruleId = ruleId
	return c
}

// Fields allows partial responses to be retrieved.
// See https://developers.google.com/gdata/docs/2.0/basics#PartialResponse
// for more information.
func (c *AccountsContainersRulesGetCall) Fields(s ...googleapi.Field) *AccountsContainersRulesGetCall {
	c.opt_["fields"] = googleapi.CombineFields(s)
	return c
}

func (c *AccountsContainersRulesGetCall) Do() (*Rule, error) {
	var body io.Reader = nil
	params := make(url.Values)
	params.Set("alt", "json")
	if v, ok := c.opt_["fields"]; ok {
		params.Set("fields", fmt.Sprintf("%v", v))
	}
	urls := googleapi.ResolveRelative(c.s.BasePath, "accounts/{accountId}/containers/{containerId}/rules/{ruleId}")
	urls += "?" + params.Encode()
	req, _ := http.NewRequest("GET", urls, body)
	googleapi.Expand(req.URL, map[string]string{
		"accountId":   c.accountId,
		"containerId": c.containerId,
		"ruleId":      c.ruleId,
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
	var ret *Rule
	if err := json.NewDecoder(res.Body).Decode(&ret); err != nil {
		return nil, err
	}
	return ret, nil
	// {
	//   "description": "Gets a GTM Rule.",
	//   "httpMethod": "GET",
	//   "id": "tagmanager.accounts.containers.rules.get",
	//   "parameterOrder": [
	//     "accountId",
	//     "containerId",
	//     "ruleId"
	//   ],
	//   "parameters": {
	//     "accountId": {
	//       "description": "The GTM Account ID.",
	//       "location": "path",
	//       "required": true,
	//       "type": "string"
	//     },
	//     "containerId": {
	//       "description": "The GTM Container ID.",
	//       "location": "path",
	//       "required": true,
	//       "type": "string"
	//     },
	//     "ruleId": {
	//       "description": "The GTM Rule ID.",
	//       "location": "path",
	//       "required": true,
	//       "type": "string"
	//     }
	//   },
	//   "path": "accounts/{accountId}/containers/{containerId}/rules/{ruleId}",
	//   "response": {
	//     "$ref": "Rule"
	//   },
	//   "scopes": [
	//     "https://www.googleapis.com/auth/tagmanager.edit.containers",
	//     "https://www.googleapis.com/auth/tagmanager.readonly"
	//   ]
	// }

}

// method id "tagmanager.accounts.containers.rules.list":

type AccountsContainersRulesListCall struct {
	s           *Service
	accountId   string
	containerId string
	opt_        map[string]interface{}
}

// List: Lists all GTM Rules of a Container.
func (r *AccountsContainersRulesService) List(accountId string, containerId string) *AccountsContainersRulesListCall {
	c := &AccountsContainersRulesListCall{s: r.s, opt_: make(map[string]interface{})}
	c.accountId = accountId
	c.containerId = containerId
	return c
}

// Fields allows partial responses to be retrieved.
// See https://developers.google.com/gdata/docs/2.0/basics#PartialResponse
// for more information.
func (c *AccountsContainersRulesListCall) Fields(s ...googleapi.Field) *AccountsContainersRulesListCall {
	c.opt_["fields"] = googleapi.CombineFields(s)
	return c
}

func (c *AccountsContainersRulesListCall) Do() (*ListRulesResponse, error) {
	var body io.Reader = nil
	params := make(url.Values)
	params.Set("alt", "json")
	if v, ok := c.opt_["fields"]; ok {
		params.Set("fields", fmt.Sprintf("%v", v))
	}
	urls := googleapi.ResolveRelative(c.s.BasePath, "accounts/{accountId}/containers/{containerId}/rules")
	urls += "?" + params.Encode()
	req, _ := http.NewRequest("GET", urls, body)
	googleapi.Expand(req.URL, map[string]string{
		"accountId":   c.accountId,
		"containerId": c.containerId,
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
	var ret *ListRulesResponse
	if err := json.NewDecoder(res.Body).Decode(&ret); err != nil {
		return nil, err
	}
	return ret, nil
	// {
	//   "description": "Lists all GTM Rules of a Container.",
	//   "httpMethod": "GET",
	//   "id": "tagmanager.accounts.containers.rules.list",
	//   "parameterOrder": [
	//     "accountId",
	//     "containerId"
	//   ],
	//   "parameters": {
	//     "accountId": {
	//       "description": "The GTM Account ID.",
	//       "location": "path",
	//       "required": true,
	//       "type": "string"
	//     },
	//     "containerId": {
	//       "description": "The GTM Container ID.",
	//       "location": "path",
	//       "required": true,
	//       "type": "string"
	//     }
	//   },
	//   "path": "accounts/{accountId}/containers/{containerId}/rules",
	//   "response": {
	//     "$ref": "ListRulesResponse"
	//   },
	//   "scopes": [
	//     "https://www.googleapis.com/auth/tagmanager.edit.containers",
	//     "https://www.googleapis.com/auth/tagmanager.readonly"
	//   ]
	// }

}

// method id "tagmanager.accounts.containers.rules.update":

type AccountsContainersRulesUpdateCall struct {
	s           *Service
	accountId   string
	containerId string
	ruleId      string
	rule        *Rule
	opt_        map[string]interface{}
}

// Update: Updates a GTM Rule.
func (r *AccountsContainersRulesService) Update(accountId string, containerId string, ruleId string, rule *Rule) *AccountsContainersRulesUpdateCall {
	c := &AccountsContainersRulesUpdateCall{s: r.s, opt_: make(map[string]interface{})}
	c.accountId = accountId
	c.containerId = containerId
	c.ruleId = ruleId
	c.rule = rule
	return c
}

// Fingerprint sets the optional parameter "fingerprint": When provided,
// this fingerprint must match the fingerprint of the rule in storage.
func (c *AccountsContainersRulesUpdateCall) Fingerprint(fingerprint string) *AccountsContainersRulesUpdateCall {
	c.opt_["fingerprint"] = fingerprint
	return c
}

// Fields allows partial responses to be retrieved.
// See https://developers.google.com/gdata/docs/2.0/basics#PartialResponse
// for more information.
func (c *AccountsContainersRulesUpdateCall) Fields(s ...googleapi.Field) *AccountsContainersRulesUpdateCall {
	c.opt_["fields"] = googleapi.CombineFields(s)
	return c
}

func (c *AccountsContainersRulesUpdateCall) Do() (*Rule, error) {
	var body io.Reader = nil
	body, err := googleapi.WithoutDataWrapper.JSONReader(c.rule)
	if err != nil {
		return nil, err
	}
	ctype := "application/json"
	params := make(url.Values)
	params.Set("alt", "json")
	if v, ok := c.opt_["fingerprint"]; ok {
		params.Set("fingerprint", fmt.Sprintf("%v", v))
	}
	if v, ok := c.opt_["fields"]; ok {
		params.Set("fields", fmt.Sprintf("%v", v))
	}
	urls := googleapi.ResolveRelative(c.s.BasePath, "accounts/{accountId}/containers/{containerId}/rules/{ruleId}")
	urls += "?" + params.Encode()
	req, _ := http.NewRequest("PUT", urls, body)
	googleapi.Expand(req.URL, map[string]string{
		"accountId":   c.accountId,
		"containerId": c.containerId,
		"ruleId":      c.ruleId,
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
	var ret *Rule
	if err := json.NewDecoder(res.Body).Decode(&ret); err != nil {
		return nil, err
	}
	return ret, nil
	// {
	//   "description": "Updates a GTM Rule.",
	//   "httpMethod": "PUT",
	//   "id": "tagmanager.accounts.containers.rules.update",
	//   "parameterOrder": [
	//     "accountId",
	//     "containerId",
	//     "ruleId"
	//   ],
	//   "parameters": {
	//     "accountId": {
	//       "description": "The GTM Account ID.",
	//       "location": "path",
	//       "required": true,
	//       "type": "string"
	//     },
	//     "containerId": {
	//       "description": "The GTM Container ID.",
	//       "location": "path",
	//       "required": true,
	//       "type": "string"
	//     },
	//     "fingerprint": {
	//       "description": "When provided, this fingerprint must match the fingerprint of the rule in storage.",
	//       "location": "query",
	//       "type": "string"
	//     },
	//     "ruleId": {
	//       "description": "The GTM Rule ID.",
	//       "location": "path",
	//       "required": true,
	//       "type": "string"
	//     }
	//   },
	//   "path": "accounts/{accountId}/containers/{containerId}/rules/{ruleId}",
	//   "request": {
	//     "$ref": "Rule"
	//   },
	//   "response": {
	//     "$ref": "Rule"
	//   },
	//   "scopes": [
	//     "https://www.googleapis.com/auth/tagmanager.edit.containers"
	//   ]
	// }

}

// method id "tagmanager.accounts.containers.tags.create":

type AccountsContainersTagsCreateCall struct {
	s           *Service
	accountId   string
	containerId string
	tag         *Tag
	opt_        map[string]interface{}
}

// Create: Creates a GTM Tag.
func (r *AccountsContainersTagsService) Create(accountId string, containerId string, tag *Tag) *AccountsContainersTagsCreateCall {
	c := &AccountsContainersTagsCreateCall{s: r.s, opt_: make(map[string]interface{})}
	c.accountId = accountId
	c.containerId = containerId
	c.tag = tag
	return c
}

// Fields allows partial responses to be retrieved.
// See https://developers.google.com/gdata/docs/2.0/basics#PartialResponse
// for more information.
func (c *AccountsContainersTagsCreateCall) Fields(s ...googleapi.Field) *AccountsContainersTagsCreateCall {
	c.opt_["fields"] = googleapi.CombineFields(s)
	return c
}

func (c *AccountsContainersTagsCreateCall) Do() (*Tag, error) {
	var body io.Reader = nil
	body, err := googleapi.WithoutDataWrapper.JSONReader(c.tag)
	if err != nil {
		return nil, err
	}
	ctype := "application/json"
	params := make(url.Values)
	params.Set("alt", "json")
	if v, ok := c.opt_["fields"]; ok {
		params.Set("fields", fmt.Sprintf("%v", v))
	}
	urls := googleapi.ResolveRelative(c.s.BasePath, "accounts/{accountId}/containers/{containerId}/tags")
	urls += "?" + params.Encode()
	req, _ := http.NewRequest("POST", urls, body)
	googleapi.Expand(req.URL, map[string]string{
		"accountId":   c.accountId,
		"containerId": c.containerId,
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
	var ret *Tag
	if err := json.NewDecoder(res.Body).Decode(&ret); err != nil {
		return nil, err
	}
	return ret, nil
	// {
	//   "description": "Creates a GTM Tag.",
	//   "httpMethod": "POST",
	//   "id": "tagmanager.accounts.containers.tags.create",
	//   "parameterOrder": [
	//     "accountId",
	//     "containerId"
	//   ],
	//   "parameters": {
	//     "accountId": {
	//       "description": "The GTM Account ID.",
	//       "location": "path",
	//       "required": true,
	//       "type": "string"
	//     },
	//     "containerId": {
	//       "description": "The GTM Container ID.",
	//       "location": "path",
	//       "required": true,
	//       "type": "string"
	//     }
	//   },
	//   "path": "accounts/{accountId}/containers/{containerId}/tags",
	//   "request": {
	//     "$ref": "Tag"
	//   },
	//   "response": {
	//     "$ref": "Tag"
	//   },
	//   "scopes": [
	//     "https://www.googleapis.com/auth/tagmanager.edit.containers"
	//   ]
	// }

}

// method id "tagmanager.accounts.containers.tags.delete":

type AccountsContainersTagsDeleteCall struct {
	s           *Service
	accountId   string
	containerId string
	tagId       string
	opt_        map[string]interface{}
}

// Delete: Deletes a GTM Tag.
func (r *AccountsContainersTagsService) Delete(accountId string, containerId string, tagId string) *AccountsContainersTagsDeleteCall {
	c := &AccountsContainersTagsDeleteCall{s: r.s, opt_: make(map[string]interface{})}
	c.accountId = accountId
	c.containerId = containerId
	c.tagId = tagId
	return c
}

// Fields allows partial responses to be retrieved.
// See https://developers.google.com/gdata/docs/2.0/basics#PartialResponse
// for more information.
func (c *AccountsContainersTagsDeleteCall) Fields(s ...googleapi.Field) *AccountsContainersTagsDeleteCall {
	c.opt_["fields"] = googleapi.CombineFields(s)
	return c
}

func (c *AccountsContainersTagsDeleteCall) Do() error {
	var body io.Reader = nil
	params := make(url.Values)
	params.Set("alt", "json")
	if v, ok := c.opt_["fields"]; ok {
		params.Set("fields", fmt.Sprintf("%v", v))
	}
	urls := googleapi.ResolveRelative(c.s.BasePath, "accounts/{accountId}/containers/{containerId}/tags/{tagId}")
	urls += "?" + params.Encode()
	req, _ := http.NewRequest("DELETE", urls, body)
	googleapi.Expand(req.URL, map[string]string{
		"accountId":   c.accountId,
		"containerId": c.containerId,
		"tagId":       c.tagId,
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
	//   "description": "Deletes a GTM Tag.",
	//   "httpMethod": "DELETE",
	//   "id": "tagmanager.accounts.containers.tags.delete",
	//   "parameterOrder": [
	//     "accountId",
	//     "containerId",
	//     "tagId"
	//   ],
	//   "parameters": {
	//     "accountId": {
	//       "description": "The GTM Account ID.",
	//       "location": "path",
	//       "required": true,
	//       "type": "string"
	//     },
	//     "containerId": {
	//       "description": "The GTM Container ID.",
	//       "location": "path",
	//       "required": true,
	//       "type": "string"
	//     },
	//     "tagId": {
	//       "description": "The GTM Tag ID.",
	//       "location": "path",
	//       "required": true,
	//       "type": "string"
	//     }
	//   },
	//   "path": "accounts/{accountId}/containers/{containerId}/tags/{tagId}",
	//   "scopes": [
	//     "https://www.googleapis.com/auth/tagmanager.edit.containers"
	//   ]
	// }

}

// method id "tagmanager.accounts.containers.tags.get":

type AccountsContainersTagsGetCall struct {
	s           *Service
	accountId   string
	containerId string
	tagId       string
	opt_        map[string]interface{}
}

// Get: Gets a GTM Tag.
func (r *AccountsContainersTagsService) Get(accountId string, containerId string, tagId string) *AccountsContainersTagsGetCall {
	c := &AccountsContainersTagsGetCall{s: r.s, opt_: make(map[string]interface{})}
	c.accountId = accountId
	c.containerId = containerId
	c.tagId = tagId
	return c
}

// Fields allows partial responses to be retrieved.
// See https://developers.google.com/gdata/docs/2.0/basics#PartialResponse
// for more information.
func (c *AccountsContainersTagsGetCall) Fields(s ...googleapi.Field) *AccountsContainersTagsGetCall {
	c.opt_["fields"] = googleapi.CombineFields(s)
	return c
}

func (c *AccountsContainersTagsGetCall) Do() (*Tag, error) {
	var body io.Reader = nil
	params := make(url.Values)
	params.Set("alt", "json")
	if v, ok := c.opt_["fields"]; ok {
		params.Set("fields", fmt.Sprintf("%v", v))
	}
	urls := googleapi.ResolveRelative(c.s.BasePath, "accounts/{accountId}/containers/{containerId}/tags/{tagId}")
	urls += "?" + params.Encode()
	req, _ := http.NewRequest("GET", urls, body)
	googleapi.Expand(req.URL, map[string]string{
		"accountId":   c.accountId,
		"containerId": c.containerId,
		"tagId":       c.tagId,
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
	var ret *Tag
	if err := json.NewDecoder(res.Body).Decode(&ret); err != nil {
		return nil, err
	}
	return ret, nil
	// {
	//   "description": "Gets a GTM Tag.",
	//   "httpMethod": "GET",
	//   "id": "tagmanager.accounts.containers.tags.get",
	//   "parameterOrder": [
	//     "accountId",
	//     "containerId",
	//     "tagId"
	//   ],
	//   "parameters": {
	//     "accountId": {
	//       "description": "The GTM Account ID.",
	//       "location": "path",
	//       "required": true,
	//       "type": "string"
	//     },
	//     "containerId": {
	//       "description": "The GTM Container ID.",
	//       "location": "path",
	//       "required": true,
	//       "type": "string"
	//     },
	//     "tagId": {
	//       "description": "The GTM Tag ID.",
	//       "location": "path",
	//       "required": true,
	//       "type": "string"
	//     }
	//   },
	//   "path": "accounts/{accountId}/containers/{containerId}/tags/{tagId}",
	//   "response": {
	//     "$ref": "Tag"
	//   },
	//   "scopes": [
	//     "https://www.googleapis.com/auth/tagmanager.edit.containers",
	//     "https://www.googleapis.com/auth/tagmanager.readonly"
	//   ]
	// }

}

// method id "tagmanager.accounts.containers.tags.list":

type AccountsContainersTagsListCall struct {
	s           *Service
	accountId   string
	containerId string
	opt_        map[string]interface{}
}

// List: Lists all GTM Tags of a Container.
func (r *AccountsContainersTagsService) List(accountId string, containerId string) *AccountsContainersTagsListCall {
	c := &AccountsContainersTagsListCall{s: r.s, opt_: make(map[string]interface{})}
	c.accountId = accountId
	c.containerId = containerId
	return c
}

// Fields allows partial responses to be retrieved.
// See https://developers.google.com/gdata/docs/2.0/basics#PartialResponse
// for more information.
func (c *AccountsContainersTagsListCall) Fields(s ...googleapi.Field) *AccountsContainersTagsListCall {
	c.opt_["fields"] = googleapi.CombineFields(s)
	return c
}

func (c *AccountsContainersTagsListCall) Do() (*ListTagsResponse, error) {
	var body io.Reader = nil
	params := make(url.Values)
	params.Set("alt", "json")
	if v, ok := c.opt_["fields"]; ok {
		params.Set("fields", fmt.Sprintf("%v", v))
	}
	urls := googleapi.ResolveRelative(c.s.BasePath, "accounts/{accountId}/containers/{containerId}/tags")
	urls += "?" + params.Encode()
	req, _ := http.NewRequest("GET", urls, body)
	googleapi.Expand(req.URL, map[string]string{
		"accountId":   c.accountId,
		"containerId": c.containerId,
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
	var ret *ListTagsResponse
	if err := json.NewDecoder(res.Body).Decode(&ret); err != nil {
		return nil, err
	}
	return ret, nil
	// {
	//   "description": "Lists all GTM Tags of a Container.",
	//   "httpMethod": "GET",
	//   "id": "tagmanager.accounts.containers.tags.list",
	//   "parameterOrder": [
	//     "accountId",
	//     "containerId"
	//   ],
	//   "parameters": {
	//     "accountId": {
	//       "description": "The GTM Account ID.",
	//       "location": "path",
	//       "required": true,
	//       "type": "string"
	//     },
	//     "containerId": {
	//       "description": "The GTM Container ID.",
	//       "location": "path",
	//       "required": true,
	//       "type": "string"
	//     }
	//   },
	//   "path": "accounts/{accountId}/containers/{containerId}/tags",
	//   "response": {
	//     "$ref": "ListTagsResponse"
	//   },
	//   "scopes": [
	//     "https://www.googleapis.com/auth/tagmanager.edit.containers",
	//     "https://www.googleapis.com/auth/tagmanager.readonly"
	//   ]
	// }

}

// method id "tagmanager.accounts.containers.tags.update":

type AccountsContainersTagsUpdateCall struct {
	s           *Service
	accountId   string
	containerId string
	tagId       string
	tag         *Tag
	opt_        map[string]interface{}
}

// Update: Updates a GTM Tag.
func (r *AccountsContainersTagsService) Update(accountId string, containerId string, tagId string, tag *Tag) *AccountsContainersTagsUpdateCall {
	c := &AccountsContainersTagsUpdateCall{s: r.s, opt_: make(map[string]interface{})}
	c.accountId = accountId
	c.containerId = containerId
	c.tagId = tagId
	c.tag = tag
	return c
}

// Fingerprint sets the optional parameter "fingerprint": When provided,
// this fingerprint must match the fingerprint of the tag in storage.
func (c *AccountsContainersTagsUpdateCall) Fingerprint(fingerprint string) *AccountsContainersTagsUpdateCall {
	c.opt_["fingerprint"] = fingerprint
	return c
}

// Fields allows partial responses to be retrieved.
// See https://developers.google.com/gdata/docs/2.0/basics#PartialResponse
// for more information.
func (c *AccountsContainersTagsUpdateCall) Fields(s ...googleapi.Field) *AccountsContainersTagsUpdateCall {
	c.opt_["fields"] = googleapi.CombineFields(s)
	return c
}

func (c *AccountsContainersTagsUpdateCall) Do() (*Tag, error) {
	var body io.Reader = nil
	body, err := googleapi.WithoutDataWrapper.JSONReader(c.tag)
	if err != nil {
		return nil, err
	}
	ctype := "application/json"
	params := make(url.Values)
	params.Set("alt", "json")
	if v, ok := c.opt_["fingerprint"]; ok {
		params.Set("fingerprint", fmt.Sprintf("%v", v))
	}
	if v, ok := c.opt_["fields"]; ok {
		params.Set("fields", fmt.Sprintf("%v", v))
	}
	urls := googleapi.ResolveRelative(c.s.BasePath, "accounts/{accountId}/containers/{containerId}/tags/{tagId}")
	urls += "?" + params.Encode()
	req, _ := http.NewRequest("PUT", urls, body)
	googleapi.Expand(req.URL, map[string]string{
		"accountId":   c.accountId,
		"containerId": c.containerId,
		"tagId":       c.tagId,
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
	var ret *Tag
	if err := json.NewDecoder(res.Body).Decode(&ret); err != nil {
		return nil, err
	}
	return ret, nil
	// {
	//   "description": "Updates a GTM Tag.",
	//   "httpMethod": "PUT",
	//   "id": "tagmanager.accounts.containers.tags.update",
	//   "parameterOrder": [
	//     "accountId",
	//     "containerId",
	//     "tagId"
	//   ],
	//   "parameters": {
	//     "accountId": {
	//       "description": "The GTM Account ID.",
	//       "location": "path",
	//       "required": true,
	//       "type": "string"
	//     },
	//     "containerId": {
	//       "description": "The GTM Container ID.",
	//       "location": "path",
	//       "required": true,
	//       "type": "string"
	//     },
	//     "fingerprint": {
	//       "description": "When provided, this fingerprint must match the fingerprint of the tag in storage.",
	//       "location": "query",
	//       "type": "string"
	//     },
	//     "tagId": {
	//       "description": "The GTM Tag ID.",
	//       "location": "path",
	//       "required": true,
	//       "type": "string"
	//     }
	//   },
	//   "path": "accounts/{accountId}/containers/{containerId}/tags/{tagId}",
	//   "request": {
	//     "$ref": "Tag"
	//   },
	//   "response": {
	//     "$ref": "Tag"
	//   },
	//   "scopes": [
	//     "https://www.googleapis.com/auth/tagmanager.edit.containers"
	//   ]
	// }

}

// method id "tagmanager.accounts.containers.triggers.create":

type AccountsContainersTriggersCreateCall struct {
	s           *Service
	accountId   string
	containerId string
	trigger     *Trigger
	opt_        map[string]interface{}
}

// Create: Creates a GTM Trigger.
func (r *AccountsContainersTriggersService) Create(accountId string, containerId string, trigger *Trigger) *AccountsContainersTriggersCreateCall {
	c := &AccountsContainersTriggersCreateCall{s: r.s, opt_: make(map[string]interface{})}
	c.accountId = accountId
	c.containerId = containerId
	c.trigger = trigger
	return c
}

// Fields allows partial responses to be retrieved.
// See https://developers.google.com/gdata/docs/2.0/basics#PartialResponse
// for more information.
func (c *AccountsContainersTriggersCreateCall) Fields(s ...googleapi.Field) *AccountsContainersTriggersCreateCall {
	c.opt_["fields"] = googleapi.CombineFields(s)
	return c
}

func (c *AccountsContainersTriggersCreateCall) Do() (*Trigger, error) {
	var body io.Reader = nil
	body, err := googleapi.WithoutDataWrapper.JSONReader(c.trigger)
	if err != nil {
		return nil, err
	}
	ctype := "application/json"
	params := make(url.Values)
	params.Set("alt", "json")
	if v, ok := c.opt_["fields"]; ok {
		params.Set("fields", fmt.Sprintf("%v", v))
	}
	urls := googleapi.ResolveRelative(c.s.BasePath, "accounts/{accountId}/containers/{containerId}/triggers")
	urls += "?" + params.Encode()
	req, _ := http.NewRequest("POST", urls, body)
	googleapi.Expand(req.URL, map[string]string{
		"accountId":   c.accountId,
		"containerId": c.containerId,
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
	var ret *Trigger
	if err := json.NewDecoder(res.Body).Decode(&ret); err != nil {
		return nil, err
	}
	return ret, nil
	// {
	//   "description": "Creates a GTM Trigger.",
	//   "httpMethod": "POST",
	//   "id": "tagmanager.accounts.containers.triggers.create",
	//   "parameterOrder": [
	//     "accountId",
	//     "containerId"
	//   ],
	//   "parameters": {
	//     "accountId": {
	//       "description": "The GTM Account ID.",
	//       "location": "path",
	//       "required": true,
	//       "type": "string"
	//     },
	//     "containerId": {
	//       "description": "The GTM Container ID.",
	//       "location": "path",
	//       "required": true,
	//       "type": "string"
	//     }
	//   },
	//   "path": "accounts/{accountId}/containers/{containerId}/triggers",
	//   "request": {
	//     "$ref": "Trigger"
	//   },
	//   "response": {
	//     "$ref": "Trigger"
	//   },
	//   "scopes": [
	//     "https://www.googleapis.com/auth/tagmanager.edit.containers"
	//   ]
	// }

}

// method id "tagmanager.accounts.containers.triggers.delete":

type AccountsContainersTriggersDeleteCall struct {
	s           *Service
	accountId   string
	containerId string
	triggerId   string
	opt_        map[string]interface{}
}

// Delete: Deletes a GTM Trigger.
func (r *AccountsContainersTriggersService) Delete(accountId string, containerId string, triggerId string) *AccountsContainersTriggersDeleteCall {
	c := &AccountsContainersTriggersDeleteCall{s: r.s, opt_: make(map[string]interface{})}
	c.accountId = accountId
	c.containerId = containerId
	c.triggerId = triggerId
	return c
}

// Fields allows partial responses to be retrieved.
// See https://developers.google.com/gdata/docs/2.0/basics#PartialResponse
// for more information.
func (c *AccountsContainersTriggersDeleteCall) Fields(s ...googleapi.Field) *AccountsContainersTriggersDeleteCall {
	c.opt_["fields"] = googleapi.CombineFields(s)
	return c
}

func (c *AccountsContainersTriggersDeleteCall) Do() error {
	var body io.Reader = nil
	params := make(url.Values)
	params.Set("alt", "json")
	if v, ok := c.opt_["fields"]; ok {
		params.Set("fields", fmt.Sprintf("%v", v))
	}
	urls := googleapi.ResolveRelative(c.s.BasePath, "accounts/{accountId}/containers/{containerId}/triggers/{triggerId}")
	urls += "?" + params.Encode()
	req, _ := http.NewRequest("DELETE", urls, body)
	googleapi.Expand(req.URL, map[string]string{
		"accountId":   c.accountId,
		"containerId": c.containerId,
		"triggerId":   c.triggerId,
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
	//   "description": "Deletes a GTM Trigger.",
	//   "httpMethod": "DELETE",
	//   "id": "tagmanager.accounts.containers.triggers.delete",
	//   "parameterOrder": [
	//     "accountId",
	//     "containerId",
	//     "triggerId"
	//   ],
	//   "parameters": {
	//     "accountId": {
	//       "description": "The GTM Account ID.",
	//       "location": "path",
	//       "required": true,
	//       "type": "string"
	//     },
	//     "containerId": {
	//       "description": "The GTM Container ID.",
	//       "location": "path",
	//       "required": true,
	//       "type": "string"
	//     },
	//     "triggerId": {
	//       "description": "The GTM Trigger ID.",
	//       "location": "path",
	//       "required": true,
	//       "type": "string"
	//     }
	//   },
	//   "path": "accounts/{accountId}/containers/{containerId}/triggers/{triggerId}",
	//   "scopes": [
	//     "https://www.googleapis.com/auth/tagmanager.edit.containers"
	//   ]
	// }

}

// method id "tagmanager.accounts.containers.triggers.get":

type AccountsContainersTriggersGetCall struct {
	s           *Service
	accountId   string
	containerId string
	triggerId   string
	opt_        map[string]interface{}
}

// Get: Gets a GTM Trigger.
func (r *AccountsContainersTriggersService) Get(accountId string, containerId string, triggerId string) *AccountsContainersTriggersGetCall {
	c := &AccountsContainersTriggersGetCall{s: r.s, opt_: make(map[string]interface{})}
	c.accountId = accountId
	c.containerId = containerId
	c.triggerId = triggerId
	return c
}

// Fields allows partial responses to be retrieved.
// See https://developers.google.com/gdata/docs/2.0/basics#PartialResponse
// for more information.
func (c *AccountsContainersTriggersGetCall) Fields(s ...googleapi.Field) *AccountsContainersTriggersGetCall {
	c.opt_["fields"] = googleapi.CombineFields(s)
	return c
}

func (c *AccountsContainersTriggersGetCall) Do() (*Trigger, error) {
	var body io.Reader = nil
	params := make(url.Values)
	params.Set("alt", "json")
	if v, ok := c.opt_["fields"]; ok {
		params.Set("fields", fmt.Sprintf("%v", v))
	}
	urls := googleapi.ResolveRelative(c.s.BasePath, "accounts/{accountId}/containers/{containerId}/triggers/{triggerId}")
	urls += "?" + params.Encode()
	req, _ := http.NewRequest("GET", urls, body)
	googleapi.Expand(req.URL, map[string]string{
		"accountId":   c.accountId,
		"containerId": c.containerId,
		"triggerId":   c.triggerId,
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
	var ret *Trigger
	if err := json.NewDecoder(res.Body).Decode(&ret); err != nil {
		return nil, err
	}
	return ret, nil
	// {
	//   "description": "Gets a GTM Trigger.",
	//   "httpMethod": "GET",
	//   "id": "tagmanager.accounts.containers.triggers.get",
	//   "parameterOrder": [
	//     "accountId",
	//     "containerId",
	//     "triggerId"
	//   ],
	//   "parameters": {
	//     "accountId": {
	//       "description": "The GTM Account ID.",
	//       "location": "path",
	//       "required": true,
	//       "type": "string"
	//     },
	//     "containerId": {
	//       "description": "The GTM Container ID.",
	//       "location": "path",
	//       "required": true,
	//       "type": "string"
	//     },
	//     "triggerId": {
	//       "description": "The GTM Trigger ID.",
	//       "location": "path",
	//       "required": true,
	//       "type": "string"
	//     }
	//   },
	//   "path": "accounts/{accountId}/containers/{containerId}/triggers/{triggerId}",
	//   "response": {
	//     "$ref": "Trigger"
	//   },
	//   "scopes": [
	//     "https://www.googleapis.com/auth/tagmanager.edit.containers",
	//     "https://www.googleapis.com/auth/tagmanager.readonly"
	//   ]
	// }

}

// method id "tagmanager.accounts.containers.triggers.list":

type AccountsContainersTriggersListCall struct {
	s           *Service
	accountId   string
	containerId string
	opt_        map[string]interface{}
}

// List: Lists all GTM Triggers of a Container.
func (r *AccountsContainersTriggersService) List(accountId string, containerId string) *AccountsContainersTriggersListCall {
	c := &AccountsContainersTriggersListCall{s: r.s, opt_: make(map[string]interface{})}
	c.accountId = accountId
	c.containerId = containerId
	return c
}

// Fields allows partial responses to be retrieved.
// See https://developers.google.com/gdata/docs/2.0/basics#PartialResponse
// for more information.
func (c *AccountsContainersTriggersListCall) Fields(s ...googleapi.Field) *AccountsContainersTriggersListCall {
	c.opt_["fields"] = googleapi.CombineFields(s)
	return c
}

func (c *AccountsContainersTriggersListCall) Do() (*ListTriggersResponse, error) {
	var body io.Reader = nil
	params := make(url.Values)
	params.Set("alt", "json")
	if v, ok := c.opt_["fields"]; ok {
		params.Set("fields", fmt.Sprintf("%v", v))
	}
	urls := googleapi.ResolveRelative(c.s.BasePath, "accounts/{accountId}/containers/{containerId}/triggers")
	urls += "?" + params.Encode()
	req, _ := http.NewRequest("GET", urls, body)
	googleapi.Expand(req.URL, map[string]string{
		"accountId":   c.accountId,
		"containerId": c.containerId,
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
	var ret *ListTriggersResponse
	if err := json.NewDecoder(res.Body).Decode(&ret); err != nil {
		return nil, err
	}
	return ret, nil
	// {
	//   "description": "Lists all GTM Triggers of a Container.",
	//   "httpMethod": "GET",
	//   "id": "tagmanager.accounts.containers.triggers.list",
	//   "parameterOrder": [
	//     "accountId",
	//     "containerId"
	//   ],
	//   "parameters": {
	//     "accountId": {
	//       "description": "The GTM Account ID.",
	//       "location": "path",
	//       "required": true,
	//       "type": "string"
	//     },
	//     "containerId": {
	//       "description": "The GTM Container ID.",
	//       "location": "path",
	//       "required": true,
	//       "type": "string"
	//     }
	//   },
	//   "path": "accounts/{accountId}/containers/{containerId}/triggers",
	//   "response": {
	//     "$ref": "ListTriggersResponse"
	//   },
	//   "scopes": [
	//     "https://www.googleapis.com/auth/tagmanager.edit.containers",
	//     "https://www.googleapis.com/auth/tagmanager.readonly"
	//   ]
	// }

}

// method id "tagmanager.accounts.containers.triggers.update":

type AccountsContainersTriggersUpdateCall struct {
	s           *Service
	accountId   string
	containerId string
	triggerId   string
	trigger     *Trigger
	opt_        map[string]interface{}
}

// Update: Updates a GTM Trigger.
func (r *AccountsContainersTriggersService) Update(accountId string, containerId string, triggerId string, trigger *Trigger) *AccountsContainersTriggersUpdateCall {
	c := &AccountsContainersTriggersUpdateCall{s: r.s, opt_: make(map[string]interface{})}
	c.accountId = accountId
	c.containerId = containerId
	c.triggerId = triggerId
	c.trigger = trigger
	return c
}

// Fingerprint sets the optional parameter "fingerprint": When provided,
// this fingerprint must match the fingerprint of the trigger in
// storage.
func (c *AccountsContainersTriggersUpdateCall) Fingerprint(fingerprint string) *AccountsContainersTriggersUpdateCall {
	c.opt_["fingerprint"] = fingerprint
	return c
}

// Fields allows partial responses to be retrieved.
// See https://developers.google.com/gdata/docs/2.0/basics#PartialResponse
// for more information.
func (c *AccountsContainersTriggersUpdateCall) Fields(s ...googleapi.Field) *AccountsContainersTriggersUpdateCall {
	c.opt_["fields"] = googleapi.CombineFields(s)
	return c
}

func (c *AccountsContainersTriggersUpdateCall) Do() (*Trigger, error) {
	var body io.Reader = nil
	body, err := googleapi.WithoutDataWrapper.JSONReader(c.trigger)
	if err != nil {
		return nil, err
	}
	ctype := "application/json"
	params := make(url.Values)
	params.Set("alt", "json")
	if v, ok := c.opt_["fingerprint"]; ok {
		params.Set("fingerprint", fmt.Sprintf("%v", v))
	}
	if v, ok := c.opt_["fields"]; ok {
		params.Set("fields", fmt.Sprintf("%v", v))
	}
	urls := googleapi.ResolveRelative(c.s.BasePath, "accounts/{accountId}/containers/{containerId}/triggers/{triggerId}")
	urls += "?" + params.Encode()
	req, _ := http.NewRequest("PUT", urls, body)
	googleapi.Expand(req.URL, map[string]string{
		"accountId":   c.accountId,
		"containerId": c.containerId,
		"triggerId":   c.triggerId,
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
	var ret *Trigger
	if err := json.NewDecoder(res.Body).Decode(&ret); err != nil {
		return nil, err
	}
	return ret, nil
	// {
	//   "description": "Updates a GTM Trigger.",
	//   "httpMethod": "PUT",
	//   "id": "tagmanager.accounts.containers.triggers.update",
	//   "parameterOrder": [
	//     "accountId",
	//     "containerId",
	//     "triggerId"
	//   ],
	//   "parameters": {
	//     "accountId": {
	//       "description": "The GTM Account ID.",
	//       "location": "path",
	//       "required": true,
	//       "type": "string"
	//     },
	//     "containerId": {
	//       "description": "The GTM Container ID.",
	//       "location": "path",
	//       "required": true,
	//       "type": "string"
	//     },
	//     "fingerprint": {
	//       "description": "When provided, this fingerprint must match the fingerprint of the trigger in storage.",
	//       "location": "query",
	//       "type": "string"
	//     },
	//     "triggerId": {
	//       "description": "The GTM Trigger ID.",
	//       "location": "path",
	//       "required": true,
	//       "type": "string"
	//     }
	//   },
	//   "path": "accounts/{accountId}/containers/{containerId}/triggers/{triggerId}",
	//   "request": {
	//     "$ref": "Trigger"
	//   },
	//   "response": {
	//     "$ref": "Trigger"
	//   },
	//   "scopes": [
	//     "https://www.googleapis.com/auth/tagmanager.edit.containers"
	//   ]
	// }

}

// method id "tagmanager.accounts.containers.variables.create":

type AccountsContainersVariablesCreateCall struct {
	s           *Service
	accountId   string
	containerId string
	variable    *Variable
	opt_        map[string]interface{}
}

// Create: Creates a GTM Variable.
func (r *AccountsContainersVariablesService) Create(accountId string, containerId string, variable *Variable) *AccountsContainersVariablesCreateCall {
	c := &AccountsContainersVariablesCreateCall{s: r.s, opt_: make(map[string]interface{})}
	c.accountId = accountId
	c.containerId = containerId
	c.variable = variable
	return c
}

// Fields allows partial responses to be retrieved.
// See https://developers.google.com/gdata/docs/2.0/basics#PartialResponse
// for more information.
func (c *AccountsContainersVariablesCreateCall) Fields(s ...googleapi.Field) *AccountsContainersVariablesCreateCall {
	c.opt_["fields"] = googleapi.CombineFields(s)
	return c
}

func (c *AccountsContainersVariablesCreateCall) Do() (*Variable, error) {
	var body io.Reader = nil
	body, err := googleapi.WithoutDataWrapper.JSONReader(c.variable)
	if err != nil {
		return nil, err
	}
	ctype := "application/json"
	params := make(url.Values)
	params.Set("alt", "json")
	if v, ok := c.opt_["fields"]; ok {
		params.Set("fields", fmt.Sprintf("%v", v))
	}
	urls := googleapi.ResolveRelative(c.s.BasePath, "accounts/{accountId}/containers/{containerId}/variables")
	urls += "?" + params.Encode()
	req, _ := http.NewRequest("POST", urls, body)
	googleapi.Expand(req.URL, map[string]string{
		"accountId":   c.accountId,
		"containerId": c.containerId,
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
	var ret *Variable
	if err := json.NewDecoder(res.Body).Decode(&ret); err != nil {
		return nil, err
	}
	return ret, nil
	// {
	//   "description": "Creates a GTM Variable.",
	//   "httpMethod": "POST",
	//   "id": "tagmanager.accounts.containers.variables.create",
	//   "parameterOrder": [
	//     "accountId",
	//     "containerId"
	//   ],
	//   "parameters": {
	//     "accountId": {
	//       "description": "The GTM Account ID.",
	//       "location": "path",
	//       "required": true,
	//       "type": "string"
	//     },
	//     "containerId": {
	//       "description": "The GTM Container ID.",
	//       "location": "path",
	//       "required": true,
	//       "type": "string"
	//     }
	//   },
	//   "path": "accounts/{accountId}/containers/{containerId}/variables",
	//   "request": {
	//     "$ref": "Variable"
	//   },
	//   "response": {
	//     "$ref": "Variable"
	//   },
	//   "scopes": [
	//     "https://www.googleapis.com/auth/tagmanager.edit.containers"
	//   ]
	// }

}

// method id "tagmanager.accounts.containers.variables.delete":

type AccountsContainersVariablesDeleteCall struct {
	s           *Service
	accountId   string
	containerId string
	variableId  string
	opt_        map[string]interface{}
}

// Delete: Deletes a GTM Variable.
func (r *AccountsContainersVariablesService) Delete(accountId string, containerId string, variableId string) *AccountsContainersVariablesDeleteCall {
	c := &AccountsContainersVariablesDeleteCall{s: r.s, opt_: make(map[string]interface{})}
	c.accountId = accountId
	c.containerId = containerId
	c.variableId = variableId
	return c
}

// Fields allows partial responses to be retrieved.
// See https://developers.google.com/gdata/docs/2.0/basics#PartialResponse
// for more information.
func (c *AccountsContainersVariablesDeleteCall) Fields(s ...googleapi.Field) *AccountsContainersVariablesDeleteCall {
	c.opt_["fields"] = googleapi.CombineFields(s)
	return c
}

func (c *AccountsContainersVariablesDeleteCall) Do() error {
	var body io.Reader = nil
	params := make(url.Values)
	params.Set("alt", "json")
	if v, ok := c.opt_["fields"]; ok {
		params.Set("fields", fmt.Sprintf("%v", v))
	}
	urls := googleapi.ResolveRelative(c.s.BasePath, "accounts/{accountId}/containers/{containerId}/variables/{variableId}")
	urls += "?" + params.Encode()
	req, _ := http.NewRequest("DELETE", urls, body)
	googleapi.Expand(req.URL, map[string]string{
		"accountId":   c.accountId,
		"containerId": c.containerId,
		"variableId":  c.variableId,
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
	//   "description": "Deletes a GTM Variable.",
	//   "httpMethod": "DELETE",
	//   "id": "tagmanager.accounts.containers.variables.delete",
	//   "parameterOrder": [
	//     "accountId",
	//     "containerId",
	//     "variableId"
	//   ],
	//   "parameters": {
	//     "accountId": {
	//       "description": "The GTM Account ID.",
	//       "location": "path",
	//       "required": true,
	//       "type": "string"
	//     },
	//     "containerId": {
	//       "description": "The GTM Container ID.",
	//       "location": "path",
	//       "required": true,
	//       "type": "string"
	//     },
	//     "variableId": {
	//       "description": "The GTM Variable ID.",
	//       "location": "path",
	//       "required": true,
	//       "type": "string"
	//     }
	//   },
	//   "path": "accounts/{accountId}/containers/{containerId}/variables/{variableId}",
	//   "scopes": [
	//     "https://www.googleapis.com/auth/tagmanager.edit.containers"
	//   ]
	// }

}

// method id "tagmanager.accounts.containers.variables.get":

type AccountsContainersVariablesGetCall struct {
	s           *Service
	accountId   string
	containerId string
	variableId  string
	opt_        map[string]interface{}
}

// Get: Gets a GTM Variable.
func (r *AccountsContainersVariablesService) Get(accountId string, containerId string, variableId string) *AccountsContainersVariablesGetCall {
	c := &AccountsContainersVariablesGetCall{s: r.s, opt_: make(map[string]interface{})}
	c.accountId = accountId
	c.containerId = containerId
	c.variableId = variableId
	return c
}

// Fields allows partial responses to be retrieved.
// See https://developers.google.com/gdata/docs/2.0/basics#PartialResponse
// for more information.
func (c *AccountsContainersVariablesGetCall) Fields(s ...googleapi.Field) *AccountsContainersVariablesGetCall {
	c.opt_["fields"] = googleapi.CombineFields(s)
	return c
}

func (c *AccountsContainersVariablesGetCall) Do() (*Variable, error) {
	var body io.Reader = nil
	params := make(url.Values)
	params.Set("alt", "json")
	if v, ok := c.opt_["fields"]; ok {
		params.Set("fields", fmt.Sprintf("%v", v))
	}
	urls := googleapi.ResolveRelative(c.s.BasePath, "accounts/{accountId}/containers/{containerId}/variables/{variableId}")
	urls += "?" + params.Encode()
	req, _ := http.NewRequest("GET", urls, body)
	googleapi.Expand(req.URL, map[string]string{
		"accountId":   c.accountId,
		"containerId": c.containerId,
		"variableId":  c.variableId,
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
	var ret *Variable
	if err := json.NewDecoder(res.Body).Decode(&ret); err != nil {
		return nil, err
	}
	return ret, nil
	// {
	//   "description": "Gets a GTM Variable.",
	//   "httpMethod": "GET",
	//   "id": "tagmanager.accounts.containers.variables.get",
	//   "parameterOrder": [
	//     "accountId",
	//     "containerId",
	//     "variableId"
	//   ],
	//   "parameters": {
	//     "accountId": {
	//       "description": "The GTM Account ID.",
	//       "location": "path",
	//       "required": true,
	//       "type": "string"
	//     },
	//     "containerId": {
	//       "description": "The GTM Container ID.",
	//       "location": "path",
	//       "required": true,
	//       "type": "string"
	//     },
	//     "variableId": {
	//       "description": "The GTM Variable ID.",
	//       "location": "path",
	//       "required": true,
	//       "type": "string"
	//     }
	//   },
	//   "path": "accounts/{accountId}/containers/{containerId}/variables/{variableId}",
	//   "response": {
	//     "$ref": "Variable"
	//   },
	//   "scopes": [
	//     "https://www.googleapis.com/auth/tagmanager.edit.containers",
	//     "https://www.googleapis.com/auth/tagmanager.readonly"
	//   ]
	// }

}

// method id "tagmanager.accounts.containers.variables.list":

type AccountsContainersVariablesListCall struct {
	s           *Service
	accountId   string
	containerId string
	opt_        map[string]interface{}
}

// List: Lists all GTM Variables of a Container.
func (r *AccountsContainersVariablesService) List(accountId string, containerId string) *AccountsContainersVariablesListCall {
	c := &AccountsContainersVariablesListCall{s: r.s, opt_: make(map[string]interface{})}
	c.accountId = accountId
	c.containerId = containerId
	return c
}

// Fields allows partial responses to be retrieved.
// See https://developers.google.com/gdata/docs/2.0/basics#PartialResponse
// for more information.
func (c *AccountsContainersVariablesListCall) Fields(s ...googleapi.Field) *AccountsContainersVariablesListCall {
	c.opt_["fields"] = googleapi.CombineFields(s)
	return c
}

func (c *AccountsContainersVariablesListCall) Do() (*ListVariablesResponse, error) {
	var body io.Reader = nil
	params := make(url.Values)
	params.Set("alt", "json")
	if v, ok := c.opt_["fields"]; ok {
		params.Set("fields", fmt.Sprintf("%v", v))
	}
	urls := googleapi.ResolveRelative(c.s.BasePath, "accounts/{accountId}/containers/{containerId}/variables")
	urls += "?" + params.Encode()
	req, _ := http.NewRequest("GET", urls, body)
	googleapi.Expand(req.URL, map[string]string{
		"accountId":   c.accountId,
		"containerId": c.containerId,
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
	var ret *ListVariablesResponse
	if err := json.NewDecoder(res.Body).Decode(&ret); err != nil {
		return nil, err
	}
	return ret, nil
	// {
	//   "description": "Lists all GTM Variables of a Container.",
	//   "httpMethod": "GET",
	//   "id": "tagmanager.accounts.containers.variables.list",
	//   "parameterOrder": [
	//     "accountId",
	//     "containerId"
	//   ],
	//   "parameters": {
	//     "accountId": {
	//       "description": "The GTM Account ID.",
	//       "location": "path",
	//       "required": true,
	//       "type": "string"
	//     },
	//     "containerId": {
	//       "description": "The GTM Container ID.",
	//       "location": "path",
	//       "required": true,
	//       "type": "string"
	//     }
	//   },
	//   "path": "accounts/{accountId}/containers/{containerId}/variables",
	//   "response": {
	//     "$ref": "ListVariablesResponse"
	//   },
	//   "scopes": [
	//     "https://www.googleapis.com/auth/tagmanager.edit.containers",
	//     "https://www.googleapis.com/auth/tagmanager.readonly"
	//   ]
	// }

}

// method id "tagmanager.accounts.containers.variables.update":

type AccountsContainersVariablesUpdateCall struct {
	s           *Service
	accountId   string
	containerId string
	variableId  string
	variable    *Variable
	opt_        map[string]interface{}
}

// Update: Updates a GTM Variable.
func (r *AccountsContainersVariablesService) Update(accountId string, containerId string, variableId string, variable *Variable) *AccountsContainersVariablesUpdateCall {
	c := &AccountsContainersVariablesUpdateCall{s: r.s, opt_: make(map[string]interface{})}
	c.accountId = accountId
	c.containerId = containerId
	c.variableId = variableId
	c.variable = variable
	return c
}

// Fingerprint sets the optional parameter "fingerprint": When provided,
// this fingerprint must match the fingerprint of the variable in
// storage.
func (c *AccountsContainersVariablesUpdateCall) Fingerprint(fingerprint string) *AccountsContainersVariablesUpdateCall {
	c.opt_["fingerprint"] = fingerprint
	return c
}

// Fields allows partial responses to be retrieved.
// See https://developers.google.com/gdata/docs/2.0/basics#PartialResponse
// for more information.
func (c *AccountsContainersVariablesUpdateCall) Fields(s ...googleapi.Field) *AccountsContainersVariablesUpdateCall {
	c.opt_["fields"] = googleapi.CombineFields(s)
	return c
}

func (c *AccountsContainersVariablesUpdateCall) Do() (*Variable, error) {
	var body io.Reader = nil
	body, err := googleapi.WithoutDataWrapper.JSONReader(c.variable)
	if err != nil {
		return nil, err
	}
	ctype := "application/json"
	params := make(url.Values)
	params.Set("alt", "json")
	if v, ok := c.opt_["fingerprint"]; ok {
		params.Set("fingerprint", fmt.Sprintf("%v", v))
	}
	if v, ok := c.opt_["fields"]; ok {
		params.Set("fields", fmt.Sprintf("%v", v))
	}
	urls := googleapi.ResolveRelative(c.s.BasePath, "accounts/{accountId}/containers/{containerId}/variables/{variableId}")
	urls += "?" + params.Encode()
	req, _ := http.NewRequest("PUT", urls, body)
	googleapi.Expand(req.URL, map[string]string{
		"accountId":   c.accountId,
		"containerId": c.containerId,
		"variableId":  c.variableId,
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
	var ret *Variable
	if err := json.NewDecoder(res.Body).Decode(&ret); err != nil {
		return nil, err
	}
	return ret, nil
	// {
	//   "description": "Updates a GTM Variable.",
	//   "httpMethod": "PUT",
	//   "id": "tagmanager.accounts.containers.variables.update",
	//   "parameterOrder": [
	//     "accountId",
	//     "containerId",
	//     "variableId"
	//   ],
	//   "parameters": {
	//     "accountId": {
	//       "description": "The GTM Account ID.",
	//       "location": "path",
	//       "required": true,
	//       "type": "string"
	//     },
	//     "containerId": {
	//       "description": "The GTM Container ID.",
	//       "location": "path",
	//       "required": true,
	//       "type": "string"
	//     },
	//     "fingerprint": {
	//       "description": "When provided, this fingerprint must match the fingerprint of the variable in storage.",
	//       "location": "query",
	//       "type": "string"
	//     },
	//     "variableId": {
	//       "description": "The GTM Variable ID.",
	//       "location": "path",
	//       "required": true,
	//       "type": "string"
	//     }
	//   },
	//   "path": "accounts/{accountId}/containers/{containerId}/variables/{variableId}",
	//   "request": {
	//     "$ref": "Variable"
	//   },
	//   "response": {
	//     "$ref": "Variable"
	//   },
	//   "scopes": [
	//     "https://www.googleapis.com/auth/tagmanager.edit.containers"
	//   ]
	// }

}

// method id "tagmanager.accounts.containers.versions.create":

type AccountsContainersVersionsCreateCall struct {
	s                                           *Service
	accountId                                   string
	containerId                                 string
	createcontainerversionrequestversionoptions *CreateContainerVersionRequestVersionOptions
	opt_                                        map[string]interface{}
}

// Create: Creates a Container Version.
func (r *AccountsContainersVersionsService) Create(accountId string, containerId string, createcontainerversionrequestversionoptions *CreateContainerVersionRequestVersionOptions) *AccountsContainersVersionsCreateCall {
	c := &AccountsContainersVersionsCreateCall{s: r.s, opt_: make(map[string]interface{})}
	c.accountId = accountId
	c.containerId = containerId
	c.createcontainerversionrequestversionoptions = createcontainerversionrequestversionoptions
	return c
}

// Fields allows partial responses to be retrieved.
// See https://developers.google.com/gdata/docs/2.0/basics#PartialResponse
// for more information.
func (c *AccountsContainersVersionsCreateCall) Fields(s ...googleapi.Field) *AccountsContainersVersionsCreateCall {
	c.opt_["fields"] = googleapi.CombineFields(s)
	return c
}

func (c *AccountsContainersVersionsCreateCall) Do() (*CreateContainerVersionResponse, error) {
	var body io.Reader = nil
	body, err := googleapi.WithoutDataWrapper.JSONReader(c.createcontainerversionrequestversionoptions)
	if err != nil {
		return nil, err
	}
	ctype := "application/json"
	params := make(url.Values)
	params.Set("alt", "json")
	if v, ok := c.opt_["fields"]; ok {
		params.Set("fields", fmt.Sprintf("%v", v))
	}
	urls := googleapi.ResolveRelative(c.s.BasePath, "accounts/{accountId}/containers/{containerId}/versions")
	urls += "?" + params.Encode()
	req, _ := http.NewRequest("POST", urls, body)
	googleapi.Expand(req.URL, map[string]string{
		"accountId":   c.accountId,
		"containerId": c.containerId,
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
	var ret *CreateContainerVersionResponse
	if err := json.NewDecoder(res.Body).Decode(&ret); err != nil {
		return nil, err
	}
	return ret, nil
	// {
	//   "description": "Creates a Container Version.",
	//   "httpMethod": "POST",
	//   "id": "tagmanager.accounts.containers.versions.create",
	//   "parameterOrder": [
	//     "accountId",
	//     "containerId"
	//   ],
	//   "parameters": {
	//     "accountId": {
	//       "description": "The GTM Account ID.",
	//       "location": "path",
	//       "required": true,
	//       "type": "string"
	//     },
	//     "containerId": {
	//       "description": "The GTM Container ID.",
	//       "location": "path",
	//       "required": true,
	//       "type": "string"
	//     }
	//   },
	//   "path": "accounts/{accountId}/containers/{containerId}/versions",
	//   "request": {
	//     "$ref": "CreateContainerVersionRequestVersionOptions"
	//   },
	//   "response": {
	//     "$ref": "CreateContainerVersionResponse"
	//   },
	//   "scopes": [
	//     "https://www.googleapis.com/auth/tagmanager.edit.containerversions"
	//   ]
	// }

}

// method id "tagmanager.accounts.containers.versions.delete":

type AccountsContainersVersionsDeleteCall struct {
	s                  *Service
	accountId          string
	containerId        string
	containerVersionId string
	opt_               map[string]interface{}
}

// Delete: Deletes a Container Version.
func (r *AccountsContainersVersionsService) Delete(accountId string, containerId string, containerVersionId string) *AccountsContainersVersionsDeleteCall {
	c := &AccountsContainersVersionsDeleteCall{s: r.s, opt_: make(map[string]interface{})}
	c.accountId = accountId
	c.containerId = containerId
	c.containerVersionId = containerVersionId
	return c
}

// Fields allows partial responses to be retrieved.
// See https://developers.google.com/gdata/docs/2.0/basics#PartialResponse
// for more information.
func (c *AccountsContainersVersionsDeleteCall) Fields(s ...googleapi.Field) *AccountsContainersVersionsDeleteCall {
	c.opt_["fields"] = googleapi.CombineFields(s)
	return c
}

func (c *AccountsContainersVersionsDeleteCall) Do() error {
	var body io.Reader = nil
	params := make(url.Values)
	params.Set("alt", "json")
	if v, ok := c.opt_["fields"]; ok {
		params.Set("fields", fmt.Sprintf("%v", v))
	}
	urls := googleapi.ResolveRelative(c.s.BasePath, "accounts/{accountId}/containers/{containerId}/versions/{containerVersionId}")
	urls += "?" + params.Encode()
	req, _ := http.NewRequest("DELETE", urls, body)
	googleapi.Expand(req.URL, map[string]string{
		"accountId":          c.accountId,
		"containerId":        c.containerId,
		"containerVersionId": c.containerVersionId,
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
	//   "description": "Deletes a Container Version.",
	//   "httpMethod": "DELETE",
	//   "id": "tagmanager.accounts.containers.versions.delete",
	//   "parameterOrder": [
	//     "accountId",
	//     "containerId",
	//     "containerVersionId"
	//   ],
	//   "parameters": {
	//     "accountId": {
	//       "description": "The GTM Account ID.",
	//       "location": "path",
	//       "required": true,
	//       "type": "string"
	//     },
	//     "containerId": {
	//       "description": "The GTM Container ID.",
	//       "location": "path",
	//       "required": true,
	//       "type": "string"
	//     },
	//     "containerVersionId": {
	//       "description": "The GTM Container Version ID.",
	//       "location": "path",
	//       "required": true,
	//       "type": "string"
	//     }
	//   },
	//   "path": "accounts/{accountId}/containers/{containerId}/versions/{containerVersionId}",
	//   "scopes": [
	//     "https://www.googleapis.com/auth/tagmanager.edit.containerversions"
	//   ]
	// }

}

// method id "tagmanager.accounts.containers.versions.get":

type AccountsContainersVersionsGetCall struct {
	s                  *Service
	accountId          string
	containerId        string
	containerVersionId string
	opt_               map[string]interface{}
}

// Get: Gets a Container Version.
func (r *AccountsContainersVersionsService) Get(accountId string, containerId string, containerVersionId string) *AccountsContainersVersionsGetCall {
	c := &AccountsContainersVersionsGetCall{s: r.s, opt_: make(map[string]interface{})}
	c.accountId = accountId
	c.containerId = containerId
	c.containerVersionId = containerVersionId
	return c
}

// Fields allows partial responses to be retrieved.
// See https://developers.google.com/gdata/docs/2.0/basics#PartialResponse
// for more information.
func (c *AccountsContainersVersionsGetCall) Fields(s ...googleapi.Field) *AccountsContainersVersionsGetCall {
	c.opt_["fields"] = googleapi.CombineFields(s)
	return c
}

func (c *AccountsContainersVersionsGetCall) Do() (*ContainerVersion, error) {
	var body io.Reader = nil
	params := make(url.Values)
	params.Set("alt", "json")
	if v, ok := c.opt_["fields"]; ok {
		params.Set("fields", fmt.Sprintf("%v", v))
	}
	urls := googleapi.ResolveRelative(c.s.BasePath, "accounts/{accountId}/containers/{containerId}/versions/{containerVersionId}")
	urls += "?" + params.Encode()
	req, _ := http.NewRequest("GET", urls, body)
	googleapi.Expand(req.URL, map[string]string{
		"accountId":          c.accountId,
		"containerId":        c.containerId,
		"containerVersionId": c.containerVersionId,
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
	var ret *ContainerVersion
	if err := json.NewDecoder(res.Body).Decode(&ret); err != nil {
		return nil, err
	}
	return ret, nil
	// {
	//   "description": "Gets a Container Version.",
	//   "httpMethod": "GET",
	//   "id": "tagmanager.accounts.containers.versions.get",
	//   "parameterOrder": [
	//     "accountId",
	//     "containerId",
	//     "containerVersionId"
	//   ],
	//   "parameters": {
	//     "accountId": {
	//       "description": "The GTM Account ID.",
	//       "location": "path",
	//       "required": true,
	//       "type": "string"
	//     },
	//     "containerId": {
	//       "description": "The GTM Container ID.",
	//       "location": "path",
	//       "required": true,
	//       "type": "string"
	//     },
	//     "containerVersionId": {
	//       "description": "The GTM Container Version ID. Specify published to retrieve the currently published version.",
	//       "location": "path",
	//       "required": true,
	//       "type": "string"
	//     }
	//   },
	//   "path": "accounts/{accountId}/containers/{containerId}/versions/{containerVersionId}",
	//   "response": {
	//     "$ref": "ContainerVersion"
	//   },
	//   "scopes": [
	//     "https://www.googleapis.com/auth/tagmanager.edit.containers",
	//     "https://www.googleapis.com/auth/tagmanager.edit.containerversions",
	//     "https://www.googleapis.com/auth/tagmanager.readonly"
	//   ]
	// }

}

// method id "tagmanager.accounts.containers.versions.list":

type AccountsContainersVersionsListCall struct {
	s           *Service
	accountId   string
	containerId string
	opt_        map[string]interface{}
}

// List: Lists all Container Versions of a GTM Container.
func (r *AccountsContainersVersionsService) List(accountId string, containerId string) *AccountsContainersVersionsListCall {
	c := &AccountsContainersVersionsListCall{s: r.s, opt_: make(map[string]interface{})}
	c.accountId = accountId
	c.containerId = containerId
	return c
}

// Headers sets the optional parameter "headers": Retrieve headers only
// when true.
func (c *AccountsContainersVersionsListCall) Headers(headers bool) *AccountsContainersVersionsListCall {
	c.opt_["headers"] = headers
	return c
}

// Fields allows partial responses to be retrieved.
// See https://developers.google.com/gdata/docs/2.0/basics#PartialResponse
// for more information.
func (c *AccountsContainersVersionsListCall) Fields(s ...googleapi.Field) *AccountsContainersVersionsListCall {
	c.opt_["fields"] = googleapi.CombineFields(s)
	return c
}

func (c *AccountsContainersVersionsListCall) Do() (*ListContainerVersionsResponse, error) {
	var body io.Reader = nil
	params := make(url.Values)
	params.Set("alt", "json")
	if v, ok := c.opt_["headers"]; ok {
		params.Set("headers", fmt.Sprintf("%v", v))
	}
	if v, ok := c.opt_["fields"]; ok {
		params.Set("fields", fmt.Sprintf("%v", v))
	}
	urls := googleapi.ResolveRelative(c.s.BasePath, "accounts/{accountId}/containers/{containerId}/versions")
	urls += "?" + params.Encode()
	req, _ := http.NewRequest("GET", urls, body)
	googleapi.Expand(req.URL, map[string]string{
		"accountId":   c.accountId,
		"containerId": c.containerId,
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
	var ret *ListContainerVersionsResponse
	if err := json.NewDecoder(res.Body).Decode(&ret); err != nil {
		return nil, err
	}
	return ret, nil
	// {
	//   "description": "Lists all Container Versions of a GTM Container.",
	//   "httpMethod": "GET",
	//   "id": "tagmanager.accounts.containers.versions.list",
	//   "parameterOrder": [
	//     "accountId",
	//     "containerId"
	//   ],
	//   "parameters": {
	//     "accountId": {
	//       "description": "The GTM Account ID.",
	//       "location": "path",
	//       "required": true,
	//       "type": "string"
	//     },
	//     "containerId": {
	//       "description": "The GTM Container ID.",
	//       "location": "path",
	//       "required": true,
	//       "type": "string"
	//     },
	//     "headers": {
	//       "default": "false",
	//       "description": "Retrieve headers only when true.",
	//       "location": "query",
	//       "type": "boolean"
	//     }
	//   },
	//   "path": "accounts/{accountId}/containers/{containerId}/versions",
	//   "response": {
	//     "$ref": "ListContainerVersionsResponse"
	//   },
	//   "scopes": [
	//     "https://www.googleapis.com/auth/tagmanager.edit.containers",
	//     "https://www.googleapis.com/auth/tagmanager.edit.containerversions",
	//     "https://www.googleapis.com/auth/tagmanager.readonly"
	//   ]
	// }

}

// method id "tagmanager.accounts.containers.versions.publish":

type AccountsContainersVersionsPublishCall struct {
	s                  *Service
	accountId          string
	containerId        string
	containerVersionId string
	opt_               map[string]interface{}
}

// Publish: Publishes a Container Version.
func (r *AccountsContainersVersionsService) Publish(accountId string, containerId string, containerVersionId string) *AccountsContainersVersionsPublishCall {
	c := &AccountsContainersVersionsPublishCall{s: r.s, opt_: make(map[string]interface{})}
	c.accountId = accountId
	c.containerId = containerId
	c.containerVersionId = containerVersionId
	return c
}

// Fingerprint sets the optional parameter "fingerprint": When provided,
// this fingerprint must match the fingerprint of the container version
// in storage.
func (c *AccountsContainersVersionsPublishCall) Fingerprint(fingerprint string) *AccountsContainersVersionsPublishCall {
	c.opt_["fingerprint"] = fingerprint
	return c
}

// Fields allows partial responses to be retrieved.
// See https://developers.google.com/gdata/docs/2.0/basics#PartialResponse
// for more information.
func (c *AccountsContainersVersionsPublishCall) Fields(s ...googleapi.Field) *AccountsContainersVersionsPublishCall {
	c.opt_["fields"] = googleapi.CombineFields(s)
	return c
}

func (c *AccountsContainersVersionsPublishCall) Do() (*PublishContainerVersionResponse, error) {
	var body io.Reader = nil
	params := make(url.Values)
	params.Set("alt", "json")
	if v, ok := c.opt_["fingerprint"]; ok {
		params.Set("fingerprint", fmt.Sprintf("%v", v))
	}
	if v, ok := c.opt_["fields"]; ok {
		params.Set("fields", fmt.Sprintf("%v", v))
	}
	urls := googleapi.ResolveRelative(c.s.BasePath, "accounts/{accountId}/containers/{containerId}/versions/{containerVersionId}/publish")
	urls += "?" + params.Encode()
	req, _ := http.NewRequest("POST", urls, body)
	googleapi.Expand(req.URL, map[string]string{
		"accountId":          c.accountId,
		"containerId":        c.containerId,
		"containerVersionId": c.containerVersionId,
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
	var ret *PublishContainerVersionResponse
	if err := json.NewDecoder(res.Body).Decode(&ret); err != nil {
		return nil, err
	}
	return ret, nil
	// {
	//   "description": "Publishes a Container Version.",
	//   "httpMethod": "POST",
	//   "id": "tagmanager.accounts.containers.versions.publish",
	//   "parameterOrder": [
	//     "accountId",
	//     "containerId",
	//     "containerVersionId"
	//   ],
	//   "parameters": {
	//     "accountId": {
	//       "description": "The GTM Account ID.",
	//       "location": "path",
	//       "required": true,
	//       "type": "string"
	//     },
	//     "containerId": {
	//       "description": "The GTM Container ID.",
	//       "location": "path",
	//       "required": true,
	//       "type": "string"
	//     },
	//     "containerVersionId": {
	//       "description": "The GTM Container Version ID.",
	//       "location": "path",
	//       "required": true,
	//       "type": "string"
	//     },
	//     "fingerprint": {
	//       "description": "When provided, this fingerprint must match the fingerprint of the container version in storage.",
	//       "location": "query",
	//       "type": "string"
	//     }
	//   },
	//   "path": "accounts/{accountId}/containers/{containerId}/versions/{containerVersionId}/publish",
	//   "response": {
	//     "$ref": "PublishContainerVersionResponse"
	//   },
	//   "scopes": [
	//     "https://www.googleapis.com/auth/tagmanager.publish"
	//   ]
	// }

}

// method id "tagmanager.accounts.containers.versions.restore":

type AccountsContainersVersionsRestoreCall struct {
	s                  *Service
	accountId          string
	containerId        string
	containerVersionId string
	opt_               map[string]interface{}
}

// Restore: Restores a Container Version. This will overwrite the
// container's current configuration (including its macros, rules and
// tags). The operation will not have any effect on the version that is
// being served (i.e. the published version).
func (r *AccountsContainersVersionsService) Restore(accountId string, containerId string, containerVersionId string) *AccountsContainersVersionsRestoreCall {
	c := &AccountsContainersVersionsRestoreCall{s: r.s, opt_: make(map[string]interface{})}
	c.accountId = accountId
	c.containerId = containerId
	c.containerVersionId = containerVersionId
	return c
}

// Fields allows partial responses to be retrieved.
// See https://developers.google.com/gdata/docs/2.0/basics#PartialResponse
// for more information.
func (c *AccountsContainersVersionsRestoreCall) Fields(s ...googleapi.Field) *AccountsContainersVersionsRestoreCall {
	c.opt_["fields"] = googleapi.CombineFields(s)
	return c
}

func (c *AccountsContainersVersionsRestoreCall) Do() (*ContainerVersion, error) {
	var body io.Reader = nil
	params := make(url.Values)
	params.Set("alt", "json")
	if v, ok := c.opt_["fields"]; ok {
		params.Set("fields", fmt.Sprintf("%v", v))
	}
	urls := googleapi.ResolveRelative(c.s.BasePath, "accounts/{accountId}/containers/{containerId}/versions/{containerVersionId}/restore")
	urls += "?" + params.Encode()
	req, _ := http.NewRequest("POST", urls, body)
	googleapi.Expand(req.URL, map[string]string{
		"accountId":          c.accountId,
		"containerId":        c.containerId,
		"containerVersionId": c.containerVersionId,
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
	var ret *ContainerVersion
	if err := json.NewDecoder(res.Body).Decode(&ret); err != nil {
		return nil, err
	}
	return ret, nil
	// {
	//   "description": "Restores a Container Version. This will overwrite the container's current configuration (including its macros, rules and tags). The operation will not have any effect on the version that is being served (i.e. the published version).",
	//   "httpMethod": "POST",
	//   "id": "tagmanager.accounts.containers.versions.restore",
	//   "parameterOrder": [
	//     "accountId",
	//     "containerId",
	//     "containerVersionId"
	//   ],
	//   "parameters": {
	//     "accountId": {
	//       "description": "The GTM Account ID.",
	//       "location": "path",
	//       "required": true,
	//       "type": "string"
	//     },
	//     "containerId": {
	//       "description": "The GTM Container ID.",
	//       "location": "path",
	//       "required": true,
	//       "type": "string"
	//     },
	//     "containerVersionId": {
	//       "description": "The GTM Container Version ID.",
	//       "location": "path",
	//       "required": true,
	//       "type": "string"
	//     }
	//   },
	//   "path": "accounts/{accountId}/containers/{containerId}/versions/{containerVersionId}/restore",
	//   "response": {
	//     "$ref": "ContainerVersion"
	//   },
	//   "scopes": [
	//     "https://www.googleapis.com/auth/tagmanager.edit.containers"
	//   ]
	// }

}

// method id "tagmanager.accounts.containers.versions.undelete":

type AccountsContainersVersionsUndeleteCall struct {
	s                  *Service
	accountId          string
	containerId        string
	containerVersionId string
	opt_               map[string]interface{}
}

// Undelete: Undeletes a Container Version.
func (r *AccountsContainersVersionsService) Undelete(accountId string, containerId string, containerVersionId string) *AccountsContainersVersionsUndeleteCall {
	c := &AccountsContainersVersionsUndeleteCall{s: r.s, opt_: make(map[string]interface{})}
	c.accountId = accountId
	c.containerId = containerId
	c.containerVersionId = containerVersionId
	return c
}

// Fields allows partial responses to be retrieved.
// See https://developers.google.com/gdata/docs/2.0/basics#PartialResponse
// for more information.
func (c *AccountsContainersVersionsUndeleteCall) Fields(s ...googleapi.Field) *AccountsContainersVersionsUndeleteCall {
	c.opt_["fields"] = googleapi.CombineFields(s)
	return c
}

func (c *AccountsContainersVersionsUndeleteCall) Do() (*ContainerVersion, error) {
	var body io.Reader = nil
	params := make(url.Values)
	params.Set("alt", "json")
	if v, ok := c.opt_["fields"]; ok {
		params.Set("fields", fmt.Sprintf("%v", v))
	}
	urls := googleapi.ResolveRelative(c.s.BasePath, "accounts/{accountId}/containers/{containerId}/versions/{containerVersionId}/undelete")
	urls += "?" + params.Encode()
	req, _ := http.NewRequest("POST", urls, body)
	googleapi.Expand(req.URL, map[string]string{
		"accountId":          c.accountId,
		"containerId":        c.containerId,
		"containerVersionId": c.containerVersionId,
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
	var ret *ContainerVersion
	if err := json.NewDecoder(res.Body).Decode(&ret); err != nil {
		return nil, err
	}
	return ret, nil
	// {
	//   "description": "Undeletes a Container Version.",
	//   "httpMethod": "POST",
	//   "id": "tagmanager.accounts.containers.versions.undelete",
	//   "parameterOrder": [
	//     "accountId",
	//     "containerId",
	//     "containerVersionId"
	//   ],
	//   "parameters": {
	//     "accountId": {
	//       "description": "The GTM Account ID.",
	//       "location": "path",
	//       "required": true,
	//       "type": "string"
	//     },
	//     "containerId": {
	//       "description": "The GTM Container ID.",
	//       "location": "path",
	//       "required": true,
	//       "type": "string"
	//     },
	//     "containerVersionId": {
	//       "description": "The GTM Container Version ID.",
	//       "location": "path",
	//       "required": true,
	//       "type": "string"
	//     }
	//   },
	//   "path": "accounts/{accountId}/containers/{containerId}/versions/{containerVersionId}/undelete",
	//   "response": {
	//     "$ref": "ContainerVersion"
	//   },
	//   "scopes": [
	//     "https://www.googleapis.com/auth/tagmanager.edit.containerversions"
	//   ]
	// }

}

// method id "tagmanager.accounts.containers.versions.update":

type AccountsContainersVersionsUpdateCall struct {
	s                  *Service
	accountId          string
	containerId        string
	containerVersionId string
	containerversion   *ContainerVersion
	opt_               map[string]interface{}
}

// Update: Updates a Container Version.
func (r *AccountsContainersVersionsService) Update(accountId string, containerId string, containerVersionId string, containerversion *ContainerVersion) *AccountsContainersVersionsUpdateCall {
	c := &AccountsContainersVersionsUpdateCall{s: r.s, opt_: make(map[string]interface{})}
	c.accountId = accountId
	c.containerId = containerId
	c.containerVersionId = containerVersionId
	c.containerversion = containerversion
	return c
}

// Fingerprint sets the optional parameter "fingerprint": When provided,
// this fingerprint must match the fingerprint of the container version
// in storage.
func (c *AccountsContainersVersionsUpdateCall) Fingerprint(fingerprint string) *AccountsContainersVersionsUpdateCall {
	c.opt_["fingerprint"] = fingerprint
	return c
}

// Fields allows partial responses to be retrieved.
// See https://developers.google.com/gdata/docs/2.0/basics#PartialResponse
// for more information.
func (c *AccountsContainersVersionsUpdateCall) Fields(s ...googleapi.Field) *AccountsContainersVersionsUpdateCall {
	c.opt_["fields"] = googleapi.CombineFields(s)
	return c
}

func (c *AccountsContainersVersionsUpdateCall) Do() (*ContainerVersion, error) {
	var body io.Reader = nil
	body, err := googleapi.WithoutDataWrapper.JSONReader(c.containerversion)
	if err != nil {
		return nil, err
	}
	ctype := "application/json"
	params := make(url.Values)
	params.Set("alt", "json")
	if v, ok := c.opt_["fingerprint"]; ok {
		params.Set("fingerprint", fmt.Sprintf("%v", v))
	}
	if v, ok := c.opt_["fields"]; ok {
		params.Set("fields", fmt.Sprintf("%v", v))
	}
	urls := googleapi.ResolveRelative(c.s.BasePath, "accounts/{accountId}/containers/{containerId}/versions/{containerVersionId}")
	urls += "?" + params.Encode()
	req, _ := http.NewRequest("PUT", urls, body)
	googleapi.Expand(req.URL, map[string]string{
		"accountId":          c.accountId,
		"containerId":        c.containerId,
		"containerVersionId": c.containerVersionId,
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
	var ret *ContainerVersion
	if err := json.NewDecoder(res.Body).Decode(&ret); err != nil {
		return nil, err
	}
	return ret, nil
	// {
	//   "description": "Updates a Container Version.",
	//   "httpMethod": "PUT",
	//   "id": "tagmanager.accounts.containers.versions.update",
	//   "parameterOrder": [
	//     "accountId",
	//     "containerId",
	//     "containerVersionId"
	//   ],
	//   "parameters": {
	//     "accountId": {
	//       "description": "The GTM Account ID.",
	//       "location": "path",
	//       "required": true,
	//       "type": "string"
	//     },
	//     "containerId": {
	//       "description": "The GTM Container ID.",
	//       "location": "path",
	//       "required": true,
	//       "type": "string"
	//     },
	//     "containerVersionId": {
	//       "description": "The GTM Container Version ID.",
	//       "location": "path",
	//       "required": true,
	//       "type": "string"
	//     },
	//     "fingerprint": {
	//       "description": "When provided, this fingerprint must match the fingerprint of the container version in storage.",
	//       "location": "query",
	//       "type": "string"
	//     }
	//   },
	//   "path": "accounts/{accountId}/containers/{containerId}/versions/{containerVersionId}",
	//   "request": {
	//     "$ref": "ContainerVersion"
	//   },
	//   "response": {
	//     "$ref": "ContainerVersion"
	//   },
	//   "scopes": [
	//     "https://www.googleapis.com/auth/tagmanager.edit.containerversions"
	//   ]
	// }

}

// method id "tagmanager.accounts.permissions.create":

type AccountsPermissionsCreateCall struct {
	s          *Service
	accountId  string
	useraccess *UserAccess
	opt_       map[string]interface{}
}

// Create: Creates a user's Account & Container Permissions.
func (r *AccountsPermissionsService) Create(accountId string, useraccess *UserAccess) *AccountsPermissionsCreateCall {
	c := &AccountsPermissionsCreateCall{s: r.s, opt_: make(map[string]interface{})}
	c.accountId = accountId
	c.useraccess = useraccess
	return c
}

// Fields allows partial responses to be retrieved.
// See https://developers.google.com/gdata/docs/2.0/basics#PartialResponse
// for more information.
func (c *AccountsPermissionsCreateCall) Fields(s ...googleapi.Field) *AccountsPermissionsCreateCall {
	c.opt_["fields"] = googleapi.CombineFields(s)
	return c
}

func (c *AccountsPermissionsCreateCall) Do() (*UserAccess, error) {
	var body io.Reader = nil
	body, err := googleapi.WithoutDataWrapper.JSONReader(c.useraccess)
	if err != nil {
		return nil, err
	}
	ctype := "application/json"
	params := make(url.Values)
	params.Set("alt", "json")
	if v, ok := c.opt_["fields"]; ok {
		params.Set("fields", fmt.Sprintf("%v", v))
	}
	urls := googleapi.ResolveRelative(c.s.BasePath, "accounts/{accountId}/permissions")
	urls += "?" + params.Encode()
	req, _ := http.NewRequest("POST", urls, body)
	googleapi.Expand(req.URL, map[string]string{
		"accountId": c.accountId,
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
	var ret *UserAccess
	if err := json.NewDecoder(res.Body).Decode(&ret); err != nil {
		return nil, err
	}
	return ret, nil
	// {
	//   "description": "Creates a user's Account \u0026 Container Permissions.",
	//   "httpMethod": "POST",
	//   "id": "tagmanager.accounts.permissions.create",
	//   "parameterOrder": [
	//     "accountId"
	//   ],
	//   "parameters": {
	//     "accountId": {
	//       "description": "The GTM Account ID.",
	//       "location": "path",
	//       "required": true,
	//       "type": "string"
	//     }
	//   },
	//   "path": "accounts/{accountId}/permissions",
	//   "request": {
	//     "$ref": "UserAccess"
	//   },
	//   "response": {
	//     "$ref": "UserAccess"
	//   },
	//   "scopes": [
	//     "https://www.googleapis.com/auth/tagmanager.manage.users"
	//   ]
	// }

}

// method id "tagmanager.accounts.permissions.delete":

type AccountsPermissionsDeleteCall struct {
	s            *Service
	accountId    string
	permissionId string
	opt_         map[string]interface{}
}

// Delete: Removes a user from the account, revoking access to it and
// all of its containers.
func (r *AccountsPermissionsService) Delete(accountId string, permissionId string) *AccountsPermissionsDeleteCall {
	c := &AccountsPermissionsDeleteCall{s: r.s, opt_: make(map[string]interface{})}
	c.accountId = accountId
	c.permissionId = permissionId
	return c
}

// Fields allows partial responses to be retrieved.
// See https://developers.google.com/gdata/docs/2.0/basics#PartialResponse
// for more information.
func (c *AccountsPermissionsDeleteCall) Fields(s ...googleapi.Field) *AccountsPermissionsDeleteCall {
	c.opt_["fields"] = googleapi.CombineFields(s)
	return c
}

func (c *AccountsPermissionsDeleteCall) Do() error {
	var body io.Reader = nil
	params := make(url.Values)
	params.Set("alt", "json")
	if v, ok := c.opt_["fields"]; ok {
		params.Set("fields", fmt.Sprintf("%v", v))
	}
	urls := googleapi.ResolveRelative(c.s.BasePath, "accounts/{accountId}/permissions/{permissionId}")
	urls += "?" + params.Encode()
	req, _ := http.NewRequest("DELETE", urls, body)
	googleapi.Expand(req.URL, map[string]string{
		"accountId":    c.accountId,
		"permissionId": c.permissionId,
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
	//   "description": "Removes a user from the account, revoking access to it and all of its containers.",
	//   "httpMethod": "DELETE",
	//   "id": "tagmanager.accounts.permissions.delete",
	//   "parameterOrder": [
	//     "accountId",
	//     "permissionId"
	//   ],
	//   "parameters": {
	//     "accountId": {
	//       "description": "The GTM Account ID.",
	//       "location": "path",
	//       "required": true,
	//       "type": "string"
	//     },
	//     "permissionId": {
	//       "description": "The GTM User ID.",
	//       "location": "path",
	//       "required": true,
	//       "type": "string"
	//     }
	//   },
	//   "path": "accounts/{accountId}/permissions/{permissionId}",
	//   "scopes": [
	//     "https://www.googleapis.com/auth/tagmanager.manage.users"
	//   ]
	// }

}

// method id "tagmanager.accounts.permissions.get":

type AccountsPermissionsGetCall struct {
	s            *Service
	accountId    string
	permissionId string
	opt_         map[string]interface{}
}

// Get: Gets a user's Account & Container Permissions.
func (r *AccountsPermissionsService) Get(accountId string, permissionId string) *AccountsPermissionsGetCall {
	c := &AccountsPermissionsGetCall{s: r.s, opt_: make(map[string]interface{})}
	c.accountId = accountId
	c.permissionId = permissionId
	return c
}

// Fields allows partial responses to be retrieved.
// See https://developers.google.com/gdata/docs/2.0/basics#PartialResponse
// for more information.
func (c *AccountsPermissionsGetCall) Fields(s ...googleapi.Field) *AccountsPermissionsGetCall {
	c.opt_["fields"] = googleapi.CombineFields(s)
	return c
}

func (c *AccountsPermissionsGetCall) Do() (*UserAccess, error) {
	var body io.Reader = nil
	params := make(url.Values)
	params.Set("alt", "json")
	if v, ok := c.opt_["fields"]; ok {
		params.Set("fields", fmt.Sprintf("%v", v))
	}
	urls := googleapi.ResolveRelative(c.s.BasePath, "accounts/{accountId}/permissions/{permissionId}")
	urls += "?" + params.Encode()
	req, _ := http.NewRequest("GET", urls, body)
	googleapi.Expand(req.URL, map[string]string{
		"accountId":    c.accountId,
		"permissionId": c.permissionId,
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
	var ret *UserAccess
	if err := json.NewDecoder(res.Body).Decode(&ret); err != nil {
		return nil, err
	}
	return ret, nil
	// {
	//   "description": "Gets a user's Account \u0026 Container Permissions.",
	//   "httpMethod": "GET",
	//   "id": "tagmanager.accounts.permissions.get",
	//   "parameterOrder": [
	//     "accountId",
	//     "permissionId"
	//   ],
	//   "parameters": {
	//     "accountId": {
	//       "description": "The GTM Account ID.",
	//       "location": "path",
	//       "required": true,
	//       "type": "string"
	//     },
	//     "permissionId": {
	//       "description": "The GTM User ID.",
	//       "location": "path",
	//       "required": true,
	//       "type": "string"
	//     }
	//   },
	//   "path": "accounts/{accountId}/permissions/{permissionId}",
	//   "response": {
	//     "$ref": "UserAccess"
	//   },
	//   "scopes": [
	//     "https://www.googleapis.com/auth/tagmanager.manage.users"
	//   ]
	// }

}

// method id "tagmanager.accounts.permissions.list":

type AccountsPermissionsListCall struct {
	s         *Service
	accountId string
	opt_      map[string]interface{}
}

// List: List all users that have access to the account along with
// Account and Container Permissions granted to each of them.
func (r *AccountsPermissionsService) List(accountId string) *AccountsPermissionsListCall {
	c := &AccountsPermissionsListCall{s: r.s, opt_: make(map[string]interface{})}
	c.accountId = accountId
	return c
}

// Fields allows partial responses to be retrieved.
// See https://developers.google.com/gdata/docs/2.0/basics#PartialResponse
// for more information.
func (c *AccountsPermissionsListCall) Fields(s ...googleapi.Field) *AccountsPermissionsListCall {
	c.opt_["fields"] = googleapi.CombineFields(s)
	return c
}

func (c *AccountsPermissionsListCall) Do() (*ListAccountUsersResponse, error) {
	var body io.Reader = nil
	params := make(url.Values)
	params.Set("alt", "json")
	if v, ok := c.opt_["fields"]; ok {
		params.Set("fields", fmt.Sprintf("%v", v))
	}
	urls := googleapi.ResolveRelative(c.s.BasePath, "accounts/{accountId}/permissions")
	urls += "?" + params.Encode()
	req, _ := http.NewRequest("GET", urls, body)
	googleapi.Expand(req.URL, map[string]string{
		"accountId": c.accountId,
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
	var ret *ListAccountUsersResponse
	if err := json.NewDecoder(res.Body).Decode(&ret); err != nil {
		return nil, err
	}
	return ret, nil
	// {
	//   "description": "List all users that have access to the account along with Account and Container Permissions granted to each of them.",
	//   "httpMethod": "GET",
	//   "id": "tagmanager.accounts.permissions.list",
	//   "parameterOrder": [
	//     "accountId"
	//   ],
	//   "parameters": {
	//     "accountId": {
	//       "description": "The GTM Account ID. @required tagmanager.accounts.permissions.list",
	//       "location": "path",
	//       "required": true,
	//       "type": "string"
	//     }
	//   },
	//   "path": "accounts/{accountId}/permissions",
	//   "response": {
	//     "$ref": "ListAccountUsersResponse"
	//   },
	//   "scopes": [
	//     "https://www.googleapis.com/auth/tagmanager.manage.users"
	//   ]
	// }

}

// method id "tagmanager.accounts.permissions.update":

type AccountsPermissionsUpdateCall struct {
	s            *Service
	accountId    string
	permissionId string
	useraccess   *UserAccess
	opt_         map[string]interface{}
}

// Update: Updates a user's Account & Container Permissions.
func (r *AccountsPermissionsService) Update(accountId string, permissionId string, useraccess *UserAccess) *AccountsPermissionsUpdateCall {
	c := &AccountsPermissionsUpdateCall{s: r.s, opt_: make(map[string]interface{})}
	c.accountId = accountId
	c.permissionId = permissionId
	c.useraccess = useraccess
	return c
}

// Fields allows partial responses to be retrieved.
// See https://developers.google.com/gdata/docs/2.0/basics#PartialResponse
// for more information.
func (c *AccountsPermissionsUpdateCall) Fields(s ...googleapi.Field) *AccountsPermissionsUpdateCall {
	c.opt_["fields"] = googleapi.CombineFields(s)
	return c
}

func (c *AccountsPermissionsUpdateCall) Do() (*UserAccess, error) {
	var body io.Reader = nil
	body, err := googleapi.WithoutDataWrapper.JSONReader(c.useraccess)
	if err != nil {
		return nil, err
	}
	ctype := "application/json"
	params := make(url.Values)
	params.Set("alt", "json")
	if v, ok := c.opt_["fields"]; ok {
		params.Set("fields", fmt.Sprintf("%v", v))
	}
	urls := googleapi.ResolveRelative(c.s.BasePath, "accounts/{accountId}/permissions/{permissionId}")
	urls += "?" + params.Encode()
	req, _ := http.NewRequest("PUT", urls, body)
	googleapi.Expand(req.URL, map[string]string{
		"accountId":    c.accountId,
		"permissionId": c.permissionId,
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
	var ret *UserAccess
	if err := json.NewDecoder(res.Body).Decode(&ret); err != nil {
		return nil, err
	}
	return ret, nil
	// {
	//   "description": "Updates a user's Account \u0026 Container Permissions.",
	//   "httpMethod": "PUT",
	//   "id": "tagmanager.accounts.permissions.update",
	//   "parameterOrder": [
	//     "accountId",
	//     "permissionId"
	//   ],
	//   "parameters": {
	//     "accountId": {
	//       "description": "The GTM Account ID.",
	//       "location": "path",
	//       "required": true,
	//       "type": "string"
	//     },
	//     "permissionId": {
	//       "description": "The GTM User ID.",
	//       "location": "path",
	//       "required": true,
	//       "type": "string"
	//     }
	//   },
	//   "path": "accounts/{accountId}/permissions/{permissionId}",
	//   "request": {
	//     "$ref": "UserAccess"
	//   },
	//   "response": {
	//     "$ref": "UserAccess"
	//   },
	//   "scopes": [
	//     "https://www.googleapis.com/auth/tagmanager.manage.users"
	//   ]
	// }

}
