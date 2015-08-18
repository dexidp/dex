// TODO(sfalvo):
// Document how to run acceptance tests.

// The mailgun package provides methods for interacting with the Mailgun API.
// It automates the HTTP request/response cycle, encodings, and other details needed by the API.
// This SDK lets you do everything the API lets you, in a more Go-friendly way.
//
// For further information please see the Mailgun documentation at
// http://documentation.mailgun.com/
//
//  Original Author: Michael Banzon
//  Contributions:   Samuel A. Falvo II <sam.falvo %at% rackspace.com>
//  Version:         0.99.0
//
// Examples
//
// This document includes a number of examples which illustrates some aspects of the GUI which might be misleading or confusing.
// All examples included are derived from an acceptance test.
// Note that every SDK function has a corresponding acceptance test, so
// if you don't find an example for a function you'd like to know more about,
// please check the acceptance sub-package for a corresponding test.
// Of course, contributions to the documentation are always welcome as well.
// Feel free to submit a pull request or open a Github issue if you cannot find an example to suit your needs.
//
// Limit and Skip Settings
//
// Many SDK functions consume a pair of parameters called limit and skip.
// These help control how much data Mailgun sends over the wire.
// Limit, as you'd expect, gives a count of the number of records you want to receive.
// Note that, at present, Mailgun imposes its own cap of 100, for all API endpoints.
// Skip indicates where in the data set you want to start receiving from.
// Mailgun defaults to the very beginning of the dataset if not specified explicitly.
//
// If you don't particularly care how much data you receive, you may specify DefaultLimit.
// If you similarly don't care about where the data starts, you may specify DefaultSkip.
//
// Functions that Return Totals
//
// Functions which accept a limit and skip setting, in general,
// will also return a total count of the items returned.
// Note that this total count is not the total in the bundle returned by the call.
// You can determine that easily enough with Go's len() function.
// The total that you receive actually refers to the complete set of data on the server.
// This total may well exceed the size returned from the API.
//
// If this happens, you may find yourself needing to iterate over the dataset of interest.
// For example:
//
//		// Get total amount of stuff we have to work with.
// 		mg := NewMailgun("example.com", "my_api_key", "")
// 		n, _, err := mg.GetStats(1, 0, nil, "sent", "opened")
// 		if err != nil {
// 			t.Fatal(err)
// 		}
//		// Loop over it all.
//		for sk := 0; sk < n; sk += limit {
//			_, stats, err := mg.GetStats(limit, sk, nil, "sent", "opened")
//		 	if err != nil {
//		 		t.Fatal(err)
//		 	}
//			doSomethingWith(stats)
//		}
//
// License
//
// Copyright (c) 2013-2014, Michael Banzon.
// All rights reserved.
//
// Redistribution and use in source and binary forms, with or without modification,
// are permitted provided that the following conditions are met:
//
// * Redistributions of source code must retain the above copyright notice, this
// list of conditions and the following disclaimer.
//
// * Redistributions in binary form must reproduce the above copyright notice, this
// list of conditions and the following disclaimer in the documentation and/or
// other materials provided with the distribution.
//
// * Neither the names of Mailgun, Michael Banzon, nor the names of their
// contributors may be used to endorse or promote products derived from
// this software without specific prior written permission.
//
// THIS SOFTWARE IS PROVIDED BY THE COPYRIGHT HOLDERS AND CONTRIBUTORS "AS IS" AND
// ANY EXPRESS OR IMPLIED WARRANTIES, INCLUDING, BUT NOT LIMITED TO, THE IMPLIED
// WARRANTIES OF MERCHANTABILITY AND FITNESS FOR A PARTICULAR PURPOSE ARE
// DISCLAIMED. IN NO EVENT SHALL THE COPYRIGHT HOLDER OR CONTRIBUTORS BE LIABLE FOR
// ANY DIRECT, INDIRECT, INCIDENTAL, SPECIAL, EXEMPLARY, OR CONSEQUENTIAL DAMAGES
// (INCLUDING, BUT NOT LIMITED TO, PROCUREMENT OF SUBSTITUTE GOODS OR SERVICES;
// LOSS OF USE, DATA, OR PROFITS; OR BUSINESS INTERRUPTION) HOWEVER CAUSED AND ON
// ANY THEORY OF LIABILITY, WHETHER IN CONTRACT, STRICT LIABILITY, OR TORT
// (INCLUDING NEGLIGENCE OR OTHERWISE) ARISING IN ANY WAY OUT OF THE USE OF THIS
// SOFTWARE, EVEN IF ADVISED OF THE POSSIBILITY OF SUCH DAMAGE.
package mailgun

import (
	"fmt"
	"github.com/mbanzon/simplehttp"
	"io"
	"time"
)

const (
	apiBase                 = "https://api.mailgun.net/v2"
	messagesEndpoint        = "messages"
	mimeMessagesEndpoint    = "messages.mime"
	addressValidateEndpoint = "address/validate"
	addressParseEndpoint    = "address/parse"
	bouncesEndpoint         = "bounces"
	statsEndpoint           = "stats"
	domainsEndpoint         = "domains"
	deleteTagEndpoint       = "tags"
	campaignsEndpoint       = "campaigns"
	eventsEndpoint          = "events"
	credentialsEndpoint     = "credentials"
	unsubscribesEndpoint    = "unsubscribes"
	routesEndpoint          = "routes"
	webhooksEndpoint        = "webhooks"
	listsEndpoint           = "lists"
	basicAuthUser           = "api"
)

// Mailgun defines the supported subset of the Mailgun API.
// The Mailgun API may contain additional features which have been deprecated since writing this SDK.
// This SDK only covers currently supported interface endpoints.
//
// Note that Mailgun reserves the right to deprecate endpoints.
// Some endpoints listed in this interface may, at any time, become obsolete.
// Always double-check with the Mailgun API Documentation to
// determine the currently supported feature set.
type Mailgun interface {
	Domain() string
	ApiKey() string
	PublicApiKey() string
	Send(m *Message) (string, string, error)
	ValidateEmail(email string) (EmailVerification, error)
	ParseAddresses(addresses ...string) ([]string, []string, error)
	GetBounces(limit, skip int) (int, []Bounce, error)
	GetSingleBounce(address string) (Bounce, error)
	AddBounce(address, code, error string) error
	DeleteBounce(address string) error
	GetStats(limit int, skip int, startDate *time.Time, event ...string) (int, []Stat, error)
	DeleteTag(tag string) error
	GetDomains(limit, skip int) (int, []Domain, error)
	GetSingleDomain(domain string) (Domain, []DNSRecord, []DNSRecord, error)
	CreateDomain(name string, smtpPassword string, spamAction string, wildcard bool) error
	DeleteDomain(name string) error
	GetCampaigns() (int, []Campaign, error)
	CreateCampaign(name, id string) error
	UpdateCampaign(oldId, name, newId string) error
	DeleteCampaign(id string) error
	GetComplaints(limit, skip int) (int, []Complaint, error)
	GetSingleComplaint(address string) (Complaint, error)
	GetStoredMessage(id string) (StoredMessage, error)
	GetStoredMessageRaw(id string) (StoredMessageRaw, error)
	DeleteStoredMessage(id string) error
	GetCredentials(limit, skip int) (int, []Credential, error)
	CreateCredential(login, password string) error
	ChangeCredentialPassword(id, password string) error
	DeleteCredential(id string) error
	GetUnsubscribes(limit, skip int) (int, []Unsubscription, error)
	GetUnsubscribesByAddress(string) (int, []Unsubscription, error)
	Unsubscribe(address, tag string) error
	RemoveUnsubscribe(string) error
	CreateComplaint(string) error
	DeleteComplaint(string) error
	GetRoutes(limit, skip int) (int, []Route, error)
	GetRouteByID(string) (Route, error)
	CreateRoute(Route) (Route, error)
	DeleteRoute(string) error
	UpdateRoute(string, Route) (Route, error)
	GetWebhooks() (map[string]string, error)
	CreateWebhook(kind, url string) error
	DeleteWebhook(kind string) error
	GetWebhookByType(kind string) (string, error)
	UpdateWebhook(kind, url string) error
	GetLists(limit, skip int, filter string) (int, []List, error)
	CreateList(List) (List, error)
	DeleteList(string) error
	GetListByAddress(string) (List, error)
	UpdateList(string, List) (List, error)
	GetMembers(limit, skip int, subfilter *bool, address string) (int, []Member, error)
	GetMemberByAddress(MemberAddr, listAddr string) (Member, error)
	CreateMember(merge bool, addr string, prototype Member) error
	CreateMemberList(subscribed *bool, addr string, newMembers []interface{}) error
	UpdateMember(Member, list string, prototype Member) (Member, error)
	DeleteMember(Member, list string) error
	NewMessage(from, subject, text string, to ...string) *Message
	NewMIMEMessage(body io.ReadCloser, to ...string) *Message
	NewEventIterator() *EventIterator
}

// MailgunImpl bundles data needed by a large number of methods in order to interact with the Mailgun API.
// Colloquially, we refer to instances of this structure as "clients."
type MailgunImpl struct {
	domain       string
	apiKey       string
	publicApiKey string
}

// NewMailGun creates a new client instance.
func NewMailgun(domain, apiKey, publicApiKey string) Mailgun {
	m := MailgunImpl{domain: domain, apiKey: apiKey, publicApiKey: publicApiKey}
	return &m
}

// Domain returns the domain configured for this client.
func (m *MailgunImpl) Domain() string {
	return m.domain
}

// ApiKey returns the API key configured for this client.
func (m *MailgunImpl) ApiKey() string {
	return m.apiKey
}

// PublicApiKey returns the public API key configured for this client.
func (m *MailgunImpl) PublicApiKey() string {
	return m.publicApiKey
}

// generateApiUrl renders a URL for an API endpoint using the domain and endpoint name.
func generateApiUrl(m Mailgun, endpoint string) string {
	return fmt.Sprintf("%s/%s/%s", apiBase, m.Domain(), endpoint)
}

// generateMemberApiUrl renders a URL relevant for specifying mailing list members.
// The address parameter refers to the mailing list in question.
func generateMemberApiUrl(endpoint, address string) string {
	return fmt.Sprintf("%s/%s/%s/members", apiBase, endpoint, address)
}

// generateApiUrlWithTarget works as generateApiUrl,
// but consumes an additional resource parameter called 'target'.
func generateApiUrlWithTarget(m Mailgun, endpoint, target string) string {
	tail := ""
	if target != "" {
		tail = fmt.Sprintf("/%s", target)
	}
	return fmt.Sprintf("%s%s", generateApiUrl(m, endpoint), tail)
}

// generateDomainApiUrl renders a URL as generateApiUrl, but
// addresses a family of functions which have a non-standard URL structure.
// Most URLs consume a domain in the 2nd position, but some endpoints
// require the word "domains" to be there instead.
func generateDomainApiUrl(m Mailgun, endpoint string) string {
	return fmt.Sprintf("%s/domains/%s/%s", apiBase, m.Domain(), endpoint)
}

// generateCredentialsUrl renders a URL as generateDomainApiUrl,
// but focuses on the SMTP credentials family of API functions.
func generateCredentialsUrl(m Mailgun, id string) string {
	tail := ""
	if id != "" {
		tail = fmt.Sprintf("/%s", id)
	}
	return generateDomainApiUrl(m, fmt.Sprintf("credentials%s", tail))
	// return fmt.Sprintf("%s/domains/%s/credentials%s", apiBase, m.Domain(), tail)
}

// generateStoredMessageUrl generates the URL needed to acquire a copy of a stored message.
func generateStoredMessageUrl(m Mailgun, endpoint, id string) string {
	return generateDomainApiUrl(m, fmt.Sprintf("%s/%s", endpoint, id))
	// return fmt.Sprintf("%s/domains/%s/%s/%s", apiBase, m.Domain(), endpoint, id)
}

// generatePublicApiUrl works as generateApiUrl, except that generatePublicApiUrl has no need for the domain.
func generatePublicApiUrl(endpoint string) string {
	return fmt.Sprintf("%s/%s", apiBase, endpoint)
}

// generateParameterizedUrl works as generateApiUrl, but supports query parameters.
func generateParameterizedUrl(m Mailgun, endpoint string, payload simplehttp.Payload) (string, error) {
	paramBuffer, err := payload.GetPayloadBuffer()
	if err != nil {
		return "", err
	}
	params := string(paramBuffer.Bytes())
	return fmt.Sprintf("%s?%s", generateApiUrl(m, eventsEndpoint), params), nil
}

// parseMailgunTime translates a timestamp as returned by Mailgun into a Go standard timestamp.
func parseMailgunTime(ts string) (t time.Time, err error) {
	t, err = time.Parse("Mon, 2 Jan 2006 15:04:05 MST", ts)
	return
}

// formatMailgunTime translates a timestamp into a human-readable form.
func formatMailgunTime(t *time.Time) string {
	return t.Format("Mon, 2 Jan 2006 15:04:05 -0700")
}
