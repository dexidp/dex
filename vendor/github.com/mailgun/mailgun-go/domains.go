package mailgun

import (
	"strconv"
	"time"
)

// DefaultLimit and DefaultSkip instruct the SDK to rely on Mailgun's reasonable defaults for pagination settings.
const (
	DefaultLimit = -1
	DefaultSkip  = -1
)

// Disabled, Tag, and Delete indicate spam actions.
// Disabled prevents Mailgun from taking any action on what it perceives to be spam.
// Tag instruments the received message with headers providing a measure of its spamness.
// Delete instructs Mailgun to just block or delete the message all-together.
const (
	Tag      = "tag"
	Disabled = "disabled"
	Delete   = "delete"
)

// A Domain structure holds information about a domain used when sending mail.
// The SpamAction field must be one of Tag, Disabled, or Delete.
type Domain struct {
	CreatedAt    string `json:"created_at"`
	SMTPLogin    string `json:"smtp_login"`
	Name         string `json:"name"`
	SMTPPassword string `json:"smtp_password"`
	Wildcard     bool   `json:"wildcard"`
	SpamAction   string `json:"spam_action"`
}

// DNSRecord structures describe intended records to properly configure your domain for use with Mailgun.
// Note that Mailgun does not host DNS records.
type DNSRecord struct {
	Priority   string
	RecordType string `json:"record_type"`
	Valid      string
	Name       string
	Value      string
}

type domainsEnvelope struct {
	TotalCount int      `json:"total_count"`
	Items      []Domain `json:"items"`
}

type singleDomainEnvelope struct {
	Domain              Domain      `json:"domain"`
	ReceivingDNSRecords []DNSRecord `json:"receiving_dns_records"`
	SendingDNSRecords   []DNSRecord `json:"sending_dns_records"`
}

// GetCreatedAt returns the time the domain was created as a normal Go time.Time type.
func (d Domain) GetCreatedAt() (t time.Time, err error) {
	t, err = parseMailgunTime(d.CreatedAt)
	return
}

// GetDomains retrieves a set of domains from Mailgun.
//
// Assuming no error, both the number of items retrieved and a slice of Domain instances.
// The number of items returned may be less than the specified limit, if it's specified.
// Note that zero items and a zero-length slice do not necessarily imply an error occurred.
// Except for the error itself, all results are undefined in the event of an error.
func (m *MailgunImpl) GetDomains(limit, skip int) (int, []Domain, error) {
	r := newHTTPRequest(generatePublicApiUrl(domainsEndpoint))
	r.setClient(m.Client())
	if limit != DefaultLimit {
		r.addParameter("limit", strconv.Itoa(limit))
	}
	if skip != DefaultSkip {
		r.addParameter("skip", strconv.Itoa(skip))
	}
	r.setBasicAuth(basicAuthUser, m.ApiKey())

	var envelope domainsEnvelope
	err := getResponseFromJSON(r, &envelope)
	if err != nil {
		return -1, nil, err
	}
	return envelope.TotalCount, envelope.Items, nil
}

// Retrieve detailed information about the named domain.
func (m *MailgunImpl) GetSingleDomain(domain string) (Domain, []DNSRecord, []DNSRecord, error) {
	r := newHTTPRequest(generatePublicApiUrl(domainsEndpoint) + "/" + domain)
	r.setClient(m.Client())
	r.setBasicAuth(basicAuthUser, m.ApiKey())
	var envelope singleDomainEnvelope
	err := getResponseFromJSON(r, &envelope)
	return envelope.Domain, envelope.ReceivingDNSRecords, envelope.SendingDNSRecords, err
}

// CreateDomain instructs Mailgun to create a new domain for your account.
// The name parameter identifies the domain.
// The smtpPassword parameter provides an access credential for the domain.
// The spamAction domain must be one of Delete, Tag, or Disabled.
// The wildcard parameter instructs Mailgun to treat all subdomains of this domain uniformly if true,
// and as different domains if false.
func (m *MailgunImpl) CreateDomain(name string, smtpPassword string, spamAction string, wildcard bool) error {
	r := newHTTPRequest(generatePublicApiUrl(domainsEndpoint))
	r.setClient(m.Client())
	r.setBasicAuth(basicAuthUser, m.ApiKey())

	payload := newUrlEncodedPayload()
	payload.addValue("name", name)
	payload.addValue("smtp_password", smtpPassword)
	payload.addValue("spam_action", spamAction)
	payload.addValue("wildcard", strconv.FormatBool(wildcard))
	_, err := makePostRequest(r, payload)
	return err
}

// DeleteDomain instructs Mailgun to dispose of the named domain name.
func (m *MailgunImpl) DeleteDomain(name string) error {
	r := newHTTPRequest(generatePublicApiUrl(domainsEndpoint) + "/" + name)
	r.setClient(m.Client())
	r.setBasicAuth(basicAuthUser, m.ApiKey())
	_, err := makeDeleteRequest(r)
	return err
}
