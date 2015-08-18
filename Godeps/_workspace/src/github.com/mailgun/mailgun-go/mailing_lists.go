package mailgun

import (
	"encoding/json"
	"fmt"
	"github.com/mbanzon/simplehttp"
	"strconv"
)

// A mailing list may have one of three membership modes.
// ReadOnly specifies that nobody, including Members,
// may send messages to the mailing list.
// Messages distributed on such lists come from list administrator accounts only.
// Members specifies that only those who subscribe to the mailing list may send messages.
// Everyone specifies that anyone and everyone may both read and submit messages
// to the mailing list, including non-subscribers.
const (
	ReadOnly = "readonly"
	Members  = "members"
	Everyone = "everyone"
)

// Mailing list members have an attribute that determines if they've subscribed to the mailing list or not.
// This attribute may be used to filter the results returned by GetSubscribers().
// All, Subscribed, and Unsubscribed provides a convenient and readable syntax for specifying the scope of the search.
var (
	All          *bool = nil
	Subscribed   *bool = &yes
	Unsubscribed *bool = &no
)

// yes and no are variables which provide us the ability to take their addresses.
// Subscribed and Unsubscribed are pointers to these booleans.
//
// We use a pointer to boolean as a kind of trinary data type:
// if nil, the relevant data type remains unspecified.
// Otherwise, its value is either true or false.
var (
	yes bool = true
	no  bool = false
)

// A List structure provides information for a mailing list.
//
// AccessLevel may be one of ReadOnly, Members, or Everyone.
type List struct {
	Address      string `json:"address",omitempty"`
	Name         string `json:"name",omitempty"`
	Description  string `json:"description",omitempty"`
	AccessLevel  string `json:"access_level",omitempty"`
	CreatedAt    string `json:"created_at",omitempty"`
	MembersCount int    `json:"members_count",omitempty"`
}

// A Member structure represents a member of the mailing list.
// The Vars field can represent any JSON-encodable data.
type Member struct {
	Address    string                 `json:"address,omitempty"`
	Name       string                 `json:"name,omitempty"`
	Subscribed *bool                  `json:"subscribed,omitempty"`
	Vars       map[string]interface{} `json:"vars,omitempty"`
}

// GetLists returns the specified set of mailing lists administered by your account.
func (mg *MailgunImpl) GetLists(limit, skip int, filter string) (int, []List, error) {
	r := simplehttp.NewHTTPRequest(generatePublicApiUrl(listsEndpoint))
	r.SetBasicAuth(basicAuthUser, mg.ApiKey())
	p := simplehttp.NewUrlEncodedPayload()
	if limit != DefaultLimit {
		p.AddValue("limit", strconv.Itoa(limit))
	}
	if skip != DefaultSkip {
		p.AddValue("skip", strconv.Itoa(skip))
	}
	if filter != "" {
		p.AddValue("address", filter)
	}
	var envelope struct {
		Items      []List `json:"items"`
		TotalCount int    `json:"total_count"`
	}
	response, err := makeRequest(r, "GET", p)
	if err != nil {
		return -1, nil, err
	}
	err = response.ParseFromJSON(&envelope)
	return envelope.TotalCount, envelope.Items, err
}

// CreateList creates a new mailing list under your Mailgun account.
// You need specify only the Address and Name members of the prototype;
// Description, and AccessLevel are optional.
// If unspecified, Description remains blank,
// while AccessLevel defaults to Everyone.
func (mg *MailgunImpl) CreateList(prototype List) (List, error) {
	r := simplehttp.NewHTTPRequest(generatePublicApiUrl(listsEndpoint))
	r.SetBasicAuth(basicAuthUser, mg.ApiKey())
	p := simplehttp.NewUrlEncodedPayload()
	if prototype.Address != "" {
		p.AddValue("address", prototype.Address)
	}
	if prototype.Name != "" {
		p.AddValue("name", prototype.Name)
	}
	if prototype.Description != "" {
		p.AddValue("description", prototype.Description)
	}
	if prototype.AccessLevel != "" {
		p.AddValue("access_level", prototype.AccessLevel)
	}
	response, err := makePostRequest(r, p)
	if err != nil {
		return List{}, err
	}
	var l List
	err = response.ParseFromJSON(&l)
	return l, err
}

// DeleteList removes all current members of the list, then removes the list itself.
// Attempts to send e-mail to the list will fail subsequent to this call.
func (mg *MailgunImpl) DeleteList(addr string) error {
	r := simplehttp.NewHTTPRequest(generatePublicApiUrl(listsEndpoint) + "/" + addr)
	r.SetBasicAuth(basicAuthUser, mg.ApiKey())
	_, err := makeDeleteRequest(r)
	return err
}

// GetListByAddress allows your application to recover the complete List structure
// representing a mailing list, so long as you have its e-mail address.
func (mg *MailgunImpl) GetListByAddress(addr string) (List, error) {
	r := simplehttp.NewHTTPRequest(generatePublicApiUrl(listsEndpoint) + "/" + addr)
	r.SetBasicAuth(basicAuthUser, mg.ApiKey())
	response, err := makeGetRequest(r)
	var envelope struct {
		List `json:"list"`
	}
	err = response.ParseFromJSON(&envelope)
	return envelope.List, err
}

// UpdateList allows you to change various attributes of a list.
// Address, Name, Description, and AccessLevel are all optional;
// only those fields which are set in the prototype will change.
//
// Be careful!  If changing the address of a mailing list,
// e-mail sent to the old address will not succeed.
// Make sure you account for the change accordingly.
func (mg *MailgunImpl) UpdateList(addr string, prototype List) (List, error) {
	r := simplehttp.NewHTTPRequest(generatePublicApiUrl(listsEndpoint) + "/" + addr)
	r.SetBasicAuth(basicAuthUser, mg.ApiKey())
	p := simplehttp.NewUrlEncodedPayload()
	if prototype.Address != "" {
		p.AddValue("address", prototype.Address)
	}
	if prototype.Name != "" {
		p.AddValue("name", prototype.Name)
	}
	if prototype.Description != "" {
		p.AddValue("description", prototype.Description)
	}
	if prototype.AccessLevel != "" {
		p.AddValue("access_level", prototype.AccessLevel)
	}
	var l List
	response, err := makePutRequest(r, p)
	if err != nil {
		return l, err
	}
	err = response.ParseFromJSON(&l)
	return l, err
}

// GetMembers returns the list of members belonging to the indicated mailing list.
// The s parameter can be set to one of three settings to help narrow the returned data set:
// All indicates that you want both Members and unsubscribed members alike, while
// Subscribed and Unsubscribed indicate you want only those eponymous subsets.
func (mg *MailgunImpl) GetMembers(limit, skip int, s *bool, addr string) (int, []Member, error) {
	r := simplehttp.NewHTTPRequest(generateMemberApiUrl(listsEndpoint, addr))
	r.SetBasicAuth(basicAuthUser, mg.ApiKey())
	p := simplehttp.NewUrlEncodedPayload()
	if limit != DefaultLimit {
		p.AddValue("limit", strconv.Itoa(limit))
	}
	if skip != DefaultSkip {
		p.AddValue("skip", strconv.Itoa(skip))
	}
	if s != nil {
		p.AddValue("subscribed", yesNo(*s))
	}
	var envelope struct {
		TotalCount int      `json:"total_count"`
		Items      []Member `json:"items"`
	}
	response, err := makeRequest(r, "GET", p)
	if err != nil {
		return -1, nil, err
	}
	err = response.ParseFromJSON(&envelope)
	return envelope.TotalCount, envelope.Items, err
}

// GetMemberByAddress returns a complete Member structure for a member of a mailing list,
// given only their subscription e-mail address.
func (mg *MailgunImpl) GetMemberByAddress(s, l string) (Member, error) {
	r := simplehttp.NewHTTPRequest(generateMemberApiUrl(listsEndpoint, l) + "/" + s)
	r.SetBasicAuth(basicAuthUser, mg.ApiKey())
	response, err := makeGetRequest(r)
	if err != nil {
		return Member{}, err
	}
	var envelope struct {
		Member Member `json:"member"`
	}
	err = response.ParseFromJSON(&envelope)
	return envelope.Member, err
}

// CreateMember registers a new member of the indicated mailing list.
// If merge is set to true, then the registration may update an existing Member's settings.
// Otherwise, an error will occur if you attempt to add a member with a duplicate e-mail address.
func (mg *MailgunImpl) CreateMember(merge bool, addr string, prototype Member) error {
	vs, err := json.Marshal(prototype.Vars)
	if err != nil {
		return err
	}

	r := simplehttp.NewHTTPRequest(generateMemberApiUrl(listsEndpoint, addr))
	r.SetBasicAuth(basicAuthUser, mg.ApiKey())
	p := simplehttp.NewFormDataPayload()
	p.AddValue("upsert", yesNo(merge))
	p.AddValue("address", prototype.Address)
	p.AddValue("name", prototype.Name)
	p.AddValue("vars", string(vs))
	if prototype.Subscribed != nil {
		p.AddValue("subscribed", yesNo(*prototype.Subscribed))
	}
	_, err = makePostRequest(r, p)
	return err
}

// UpdateMember lets you change certain details about the indicated mailing list member.
// Address, Name, Vars, and Subscribed fields may be changed.
func (mg *MailgunImpl) UpdateMember(s, l string, prototype Member) (Member, error) {
	r := simplehttp.NewHTTPRequest(generateMemberApiUrl(listsEndpoint, l) + "/" + s)
	r.SetBasicAuth(basicAuthUser, mg.ApiKey())
	p := simplehttp.NewFormDataPayload()
	if prototype.Address != "" {
		p.AddValue("address", prototype.Address)
	}
	if prototype.Name != "" {
		p.AddValue("name", prototype.Name)
	}
	if prototype.Vars != nil {
		vs, err := json.Marshal(prototype.Vars)
		if err != nil {
			return Member{}, err
		}
		p.AddValue("vars", string(vs))
	}
	if prototype.Subscribed != nil {
		p.AddValue("subscribed", yesNo(*prototype.Subscribed))
	}
	response, err := makePutRequest(r, p)
	if err != nil {
		return Member{}, err
	}
	var envelope struct {
		Member Member `json:"member"`
	}
	err = response.ParseFromJSON(&envelope)
	return envelope.Member, err
}

// DeleteMember removes the member from the list.
func (mg *MailgunImpl) DeleteMember(member, addr string) error {
	r := simplehttp.NewHTTPRequest(generateMemberApiUrl(listsEndpoint, addr) + "/" + member)
	r.SetBasicAuth(basicAuthUser, mg.ApiKey())
	_, err := makeDeleteRequest(r)
	return err
}

// CreateMemberList registers multiple Members and non-Member members to a single mailing list
// in a single round-trip.
// s indicates the default subscribed status (Subscribed or Unsubscribed).
// Use All to elect not to provide a default.
// The newMembers list can take one of two JSON-encodable forms: an slice of strings, or
// a slice of Member structures.
// If a simple slice of strings is passed, each string refers to the member's e-mail address.
// Otherwise, each Member needs to have at least the Address field filled out.
// Other fields are optional, but may be set according to your needs.
func (mg *MailgunImpl) CreateMemberList(s *bool, addr string, newMembers []interface{}) error {
	r := simplehttp.NewHTTPRequest(generateMemberApiUrl(listsEndpoint, addr) + ".json")
	r.SetBasicAuth(basicAuthUser, mg.ApiKey())
	p := simplehttp.NewFormDataPayload()
	if s != nil {
		p.AddValue("subscribed", yesNo(*s))
	}
	bs, err := json.Marshal(newMembers)
	if err != nil {
		return err
	}
	fmt.Println(string(bs))
	p.AddValue("members", string(bs))
	_, err = makePostRequest(r, p)
	return err
}
