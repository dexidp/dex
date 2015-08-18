package mailgun

import (
	"github.com/mbanzon/simplehttp"
	"strings"
)

// The EmailVerificationParts structure breaks out the basic elements of an email address.
// LocalPart includes everything up to the '@' in an e-mail address.
// Domain includes everything after the '@'.
// DisplayName is no longer used, and will appear as "".
type EmailVerificationParts struct {
	LocalPart   string `json:"local_part"`
	Domain      string `json:"domain"`
	DisplayName string `json:"display_name"`
}

// EmailVerification records basic facts about a validated e-mail address.
// See the ValidateEmail method and example for more details.
//
// IsValid indicates whether an email address conforms to IETF RFC standards.
// Parts records the different subfields of an email address.
// Address echoes the address provided.
// DidYouMean provides a simple recommendation in case the address is invalid or
// Mailgun thinks you might have a typo.
// DidYouMean may be empty (""), in which case Mailgun has no recommendation to give.
// The existence of DidYouMean does NOT imply the email provided has anything wrong with it.
type EmailVerification struct {
	IsValid    bool                   `json:"is_valid"`
	Parts      EmailVerificationParts `json:"parts"`
	Address    string                 `json:"address"`
	DidYouMean string                 `json:"did_you_mean"`
}

type addressParseResult struct {
	Parsed      []string `json:"parsed"`
	Unparseable []string `json:"unparseable"`
}

// ValidateEmail performs various checks on the email address provided to ensure it's correctly formatted.
// It may also be used to break an email address into its sub-components.  (See example.)
// NOTE: Use of this function requires a proper public API key.  The private API key will not work.
func (m *MailgunImpl) ValidateEmail(email string) (EmailVerification, error) {
	r := simplehttp.NewHTTPRequest(generatePublicApiUrl(addressValidateEndpoint))
	r.AddParameter("address", email)
	r.SetBasicAuth(basicAuthUser, m.PublicApiKey())

	var response EmailVerification
	err := getResponseFromJSON(r, &response)
	if err != nil {
		return EmailVerification{}, err
	}

	return response, nil
}

// ParseAddresses takes a list of addresses and sorts them into valid and invalid address categories.
// NOTE: Use of this function requires a proper public API key.  The private API key will not work.
func (m *MailgunImpl) ParseAddresses(addresses ...string) ([]string, []string, error) {
	r := simplehttp.NewHTTPRequest(generatePublicApiUrl(addressParseEndpoint))
	r.AddParameter("addresses", strings.Join(addresses, ","))
	r.SetBasicAuth(basicAuthUser, m.PublicApiKey())

	var response addressParseResult
	err := getResponseFromJSON(r, &response)
	if err != nil {
		return nil, nil, err
	}

	return response.Parsed, response.Unparseable, nil
}
