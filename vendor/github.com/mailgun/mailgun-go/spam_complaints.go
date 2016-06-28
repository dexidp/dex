package mailgun

import (
	"strconv"
)

const (
	complaintsEndpoint = "complaints"
)

// Complaint structures track how many times one of your emails have been marked as spam.
// CreatedAt indicates when the first report arrives from a given recipient, identified by Address.
// Count provides a running counter of how many times
// the recipient thought your messages were not solicited.
type Complaint struct {
	Count     int    `json:"count"`
	CreatedAt string `json:"created_at"`
	Address   string `json:"address"`
}

type complaintsEnvelope struct {
	TotalCount int         `json:"total_count"`
	Items      []Complaint `json:"items"`
}

// GetComplaints returns a set of spam complaints registered against your domain.
// Recipients of your messages can click on a link which sends feedback to Mailgun
// indicating that the message they received is, to them, spam.
func (m *MailgunImpl) GetComplaints(limit, skip int) (int, []Complaint, error) {
	r := newHTTPRequest(generateApiUrl(m, complaintsEndpoint))
	r.setClient(m.Client())
	r.setBasicAuth(basicAuthUser, m.ApiKey())

	if limit != -1 {
		r.addParameter("limit", strconv.Itoa(limit))
	}
	if skip != -1 {
		r.addParameter("skip", strconv.Itoa(skip))
	}

	var envelope complaintsEnvelope
	err := getResponseFromJSON(r, &envelope)
	if err != nil {
		return -1, nil, err
	}
	return envelope.TotalCount, envelope.Items, nil
}

// GetSingleComplaint returns a single complaint record filed by a recipient at the email address provided.
// If no complaint exists, the Complaint instance returned will be empty.
func (m *MailgunImpl) GetSingleComplaint(address string) (Complaint, error) {
	r := newHTTPRequest(generateApiUrl(m, complaintsEndpoint) + "/" + address)
	r.setClient(m.Client())
	r.setBasicAuth(basicAuthUser, m.ApiKey())

	var c Complaint
	err := getResponseFromJSON(r, &c)
	return c, err
}

// CreateComplaint registers the specified address as a recipient who has complained of receiving spam
// from your domain.
func (m *MailgunImpl) CreateComplaint(address string) error {
	r := newHTTPRequest(generateApiUrl(m, complaintsEndpoint))
	r.setClient(m.Client())
	r.setBasicAuth(basicAuthUser, m.ApiKey())
	p := newUrlEncodedPayload()
	p.addValue("address", address)
	_, err := makePostRequest(r, p)
	return err
}

// DeleteComplaint removes a previously registered e-mail address from the list of people who complained
// of receiving spam from your domain.
func (m *MailgunImpl) DeleteComplaint(address string) error {
	r := newHTTPRequest(generateApiUrl(m, complaintsEndpoint) + "/" + address)
	r.setClient(m.Client())
	r.setBasicAuth(basicAuthUser, m.ApiKey())
	_, err := makeDeleteRequest(r)
	return err
}
