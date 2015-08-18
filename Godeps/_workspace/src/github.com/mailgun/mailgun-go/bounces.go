package mailgun

import (
	"github.com/mbanzon/simplehttp"
	"strconv"
	"time"
)

// Bounce aggregates data relating to undeliverable messages to a specific intended recipient,
// identified by Address.
// Code provides the SMTP error code causing the bounce,
// while Error provides a human readable reason why.
// CreatedAt provides the time at which Mailgun detected the bounce.
type Bounce struct {
	CreatedAt string      `json:"created_at"`
	code      interface{} `json:"code"`
	Address   string      `json:"address"`
	Error     string      `json:"error"`
}

type bounceEnvelope struct {
	TotalCount int      `json:"total_count"`
	Items      []Bounce `json:"items"`
}

type singleBounceEnvelope struct {
	Bounce Bounce `json:"bounce"`
}

// GetCreatedAt parses the textual, RFC-822 timestamp into a standard Go-compatible
// Time structure.
func (i Bounce) GetCreatedAt() (t time.Time, err error) {
	return parseMailgunTime(i.CreatedAt)
}

// GetCode will return the bounce code for the message, regardless if it was
// returned as a string or as an integer.  This method overcomes a protocol
// bug in the Mailgun API.
func (b Bounce) GetCode() (int, error) {
	switch c := b.code.(type) {
	case int:
		return c, nil
	case string:
		return strconv.Atoi(c)
	default:
		return -1, strconv.ErrSyntax
	}
}

// GetBounces returns a complete set of bounces logged against the sender's domain, if any.
// The results include the total number of bounces (regardless of skip or limit settings),
// and the slice of bounces specified, if successful.
// Note that the length of the slice may be smaller than the total number of bounces.
func (m *MailgunImpl) GetBounces(limit, skip int) (int, []Bounce, error) {
	r := simplehttp.NewHTTPRequest(generateApiUrl(m, bouncesEndpoint))
	if limit != -1 {
		r.AddParameter("limit", strconv.Itoa(limit))
	}
	if skip != -1 {
		r.AddParameter("skip", strconv.Itoa(skip))
	}

	r.SetBasicAuth(basicAuthUser, m.ApiKey())

	var response bounceEnvelope
	err := getResponseFromJSON(r, &response)
	if err != nil {
		return -1, nil, err
	}

	return response.TotalCount, response.Items, nil
}

// GetSingleBounce retrieves a single bounce record, if any exist, for the given recipient address.
func (m *MailgunImpl) GetSingleBounce(address string) (Bounce, error) {
	r := simplehttp.NewHTTPRequest(generateApiUrl(m, bouncesEndpoint) + "/" + address)
	r.SetBasicAuth(basicAuthUser, m.ApiKey())

	var response singleBounceEnvelope
	err := getResponseFromJSON(r, &response)
	return response.Bounce, err
}

// AddBounce files a bounce report.
// Address identifies the intended recipient of the message that bounced.
// Code corresponds to the numeric response given by the e-mail server which rejected the message.
// Error providees the corresponding human readable reason for the problem.
// For example,
// here's how the these two fields relate.
// Suppose the SMTP server responds with an error, as below.
// Then, . . .
//
//      550  Requested action not taken: mailbox unavailable
//     \___/\_______________________________________________/
//       |                         |
//       `-- Code                  `-- Error
//
// Note that both code and error exist as strings, even though
// code will report as a number.
func (m *MailgunImpl) AddBounce(address, code, error string) error {
	r := simplehttp.NewHTTPRequest(generateApiUrl(m, bouncesEndpoint))
	r.SetBasicAuth(basicAuthUser, m.ApiKey())

	payload := simplehttp.NewUrlEncodedPayload()
	payload.AddValue("address", address)
	if code != "" {
		payload.AddValue("code", code)
	}
	if error != "" {
		payload.AddValue("error", error)
	}
	_, err := makePostRequest(r, payload)
	return err
}

// DeleteBounce removes all bounces associted with the provided e-mail address.
func (m *MailgunImpl) DeleteBounce(address string) error {
	r := simplehttp.NewHTTPRequest(generateApiUrl(m, bouncesEndpoint) + "/" + address)
	r.SetBasicAuth(basicAuthUser, m.ApiKey())
	_, err := makeDeleteRequest(r)
	return err
}
