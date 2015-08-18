package mailgun

import (
	"encoding/json"
	"errors"
	"io"
	"time"

	"github.com/mbanzon/simplehttp"
)

// MaxNumberOfRecipients represents the largest batch of recipients that Mailgun can support in a single API call.
// This figure includes To:, Cc:, Bcc:, etc. recipients.
const MaxNumberOfRecipients = 1000

// Message structures contain both the message text and the envelop for an e-mail message.
type Message struct {
	to                []string
	tags              []string
	campaigns         []string
	dkim              bool
	deliveryTime      *time.Time
	attachments       []string
	readerAttachments []ReaderAttachment
	inlines           []string

	testMode           bool
	tracking           bool
	trackingClicks     bool
	trackingOpens      bool
	headers            map[string]string
	variables          map[string]string
	recipientVariables map[string]map[string]interface{}

	dkimSet           bool
	trackingSet       bool
	trackingClicksSet bool
	trackingOpensSet  bool

	specific features
	mg       Mailgun
}

type ReaderAttachment struct {
	Filename   string
	ReadCloser io.ReadCloser
}

// StoredMessage structures contain the (parsed) message content for an email
// sent to a Mailgun account.
//
// The MessageHeaders field is special, in that it's formatted as a slice of pairs.
// Each pair consists of a name [0] and value [1].  Array notation is used instead of a map
// because that's how it's sent over the wire, and it's how encoding/json expects this field
// to be.
type StoredMessage struct {
	Recipients        string                 `json:"recipients"`
	Sender            string                 `json:"sender"`
	From              string                 `json:"from"`
	Subject           string                 `json:"subject"`
	BodyPlain         string                 `json:"body-plain"`
	StrippedText      string                 `json:"stripped-text"`
	StrippedSignature string                 `json:"stripped-signature"`
	BodyHtml          string                 `json:"body-html"`
	StrippedHtml      string                 `json:"stripped-html"`
	Attachments       []StoredAttachment     `json:"attachments"`
	MessageUrl        string                 `json:"message-url"`
	ContentIDMap      map[string]interface{} `json:"content-id-map"`
	MessageHeaders    [][]string             `json:"message-headers"`
}

// StoredAttachment structures contain information on an attachment associated with a stored message.
type StoredAttachment struct {
	Size        int    `json:"size"`
	Url         string `json:"url"`
	Name        string `json:"name"`
	ContentType string `json:"content-type"`
}

type StoredMessageRaw struct {
	Recipients string `json:"recipients"`
	Sender     string `json:"sender"`
	From       string `json:"from"`
	Subject    string `json:"subject"`
	BodyMime   string `json:"body-mime"`
}

// plainMessage contains fields relevant to plain API-synthesized messages.
// You're expected to use various setters to set most of these attributes,
// although from, subject, and text are set when the message is created with
// NewMessage.
type plainMessage struct {
	from    string
	cc      []string
	bcc     []string
	subject string
	text    string
	html    string
}

// mimeMessage contains fields relevant to pre-packaged MIME messages.
type mimeMessage struct {
	body io.ReadCloser
}

type sendMessageResponse struct {
	Message string `json:"message"`
	Id      string `json:"id"`
}

// features abstracts the common characteristics between regular and MIME messages.
// addCC, addBCC, recipientCount, and setHTML are invoked via the package-global AddCC, AddBCC,
// RecipientCount, and SetHtml calls, as these functions are ignored for MIME messages.
// Send() invokes addValues to add message-type-specific MIME headers for the API call
// to Mailgun.  isValid yeilds true if and only if the message is valid enough for sending
// through the API.  Finally, endpoint() tells Send() which endpoint to use to submit the API call.
type features interface {
	addCC(string)
	addBCC(string)
	setHtml(string)
	addValues(*simplehttp.FormDataPayload)
	isValid() bool
	endpoint() string
	recipientCount() int
}

// NewMessage returns a new e-mail message with the simplest envelop needed to send.
//
// DEPRECATED.
// The package will panic if you use AddRecipient(), AddBcc(), AddCc(), et. al.
// on a message already equipped with MaxNumberOfRecipients recipients.
// Use Mailgun.NewMessage() instead.
// It works similarly to this function, but supports larger lists of recipients.
func NewMessage(from string, subject string, text string, to ...string) *Message {
	return &Message{
		specific: &plainMessage{
			from:    from,
			subject: subject,
			text:    text,
		},
		to: to,
	}
}

// NewMessage returns a new e-mail message with the simplest envelop needed to send.
//
// Unlike the global function,
// this method supports arbitrary-sized recipient lists by
// automatically sending mail in batches of up to MaxNumberOfRecipients.
//
// To support batch sending, you don't want to provide a fixed To: header at this point.
// Pass nil as the to parameter to skip adding the To: header at this stage.
// You can do this explicitly, or implicitly, as follows:
//
//     // Note absence of To parameter(s)!
//     m := mg.NewMessage("me@example.com", "Help save our planet", "Hello world!")
//
// Note that you'll need to invoke the AddRecipientAndVariables or AddRecipient method
// before sending, though.
func (mg *MailgunImpl) NewMessage(from, subject, text string, to ...string) *Message {
	return &Message{
		specific: &plainMessage{
			from:    from,
			subject: subject,
			text:    text,
		},
		to: to,
		mg: mg,
	}
}

// NewMIMEMessage creates a new MIME message.  These messages are largely canned;
// you do not need to invoke setters to set message-related headers.
// However, you do still need to call setters for Mailgun-specific settings.
//
// DEPRECATED.
// The package will panic if you use AddRecipient(), AddBcc(), AddCc(), et. al.
// on a message already equipped with MaxNumberOfRecipients recipients.
// Use Mailgun.NewMIMEMessage() instead.
// It works similarly to this function, but supports larger lists of recipients.
func NewMIMEMessage(body io.ReadCloser, to ...string) *Message {
	return &Message{
		specific: &mimeMessage{
			body: body,
		},
		to: to,
	}
}

// NewMIMEMessage creates a new MIME message.  These messages are largely canned;
// you do not need to invoke setters to set message-related headers.
// However, you do still need to call setters for Mailgun-specific settings.
//
// Unlike the global function,
// this method supports arbitrary-sized recipient lists by
// automatically sending mail in batches of up to MaxNumberOfRecipients.
//
// To support batch sending, you don't want to provide a fixed To: header at this point.
// Pass nil as the to parameter to skip adding the To: header at this stage.
// You can do this explicitly, or implicitly, as follows:
//
//     // Note absence of To parameter(s)!
//     m := mg.NewMessage("me@example.com", "Help save our planet", "Hello world!")
//
// Note that you'll need to invoke the AddRecipientAndVariables or AddRecipient method
// before sending, though.
func (mg *MailgunImpl) NewMIMEMessage(body io.ReadCloser, to ...string) *Message {
	return &Message{
		specific: &mimeMessage{
			body: body,
		},
		to: to,
		mg: mg,
	}
}

// AddReaderAttachment arranges to send a file along with the e-mail message.
// File contents are read from a io.ReadCloser.
// The filename parameter is the resulting filename of the attachment.
// The readCloser parameter is the io.ReadCloser which reads the actual bytes to be used
// as the contents of the attached file.
func (m *Message) AddReaderAttachment(filename string, readCloser io.ReadCloser) {
	ra := ReaderAttachment{Filename: filename, ReadCloser: readCloser}
	m.readerAttachments = append(m.readerAttachments, ra)
}

// AddAttachment arranges to send a file from the filesystem along with the e-mail message.
// The attachment parameter is a filename, which must refer to a file which actually resides
// in the local filesystem.
func (m *Message) AddAttachment(attachment string) {
	m.attachments = append(m.attachments, attachment)
}

// AddInline arranges to send a file along with the e-mail message, but does so
// in a way that its data remains "inline" with the rest of the message.  This
// can be used to send image or font data along with an HTML-encoded message body.
// The attachment parameter is a filename, which must refer to a file which actually resides
// in the local filesystem.
func (m *Message) AddInline(inline string) {
	m.inlines = append(m.inlines, inline)
}

// AddRecipient appends a receiver to the To: header of a message.
//
// NOTE: Above a certain limit (currently 1000 recipients),
// this function will cause the message as it's currently defined to be sent.
// This allows you to support large mailing lists without running into Mailgun's API limitations.
func (m *Message) AddRecipient(recipient string) error {
	return m.AddRecipientAndVariables(recipient, nil)
}

// AddRecipientAndVariables appends a receiver to the To: header of a message,
// and as well attaches a set of variables relevant for this recipient.
//
// NOTE: Above a certain limit (see MaxNumberOfRecipients),
// this function will cause the message as it's currently defined to be sent.
// This allows you to support large mailing lists without running into Mailgun's API limitations.
func (m *Message) AddRecipientAndVariables(r string, vars map[string]interface{}) error {
	if m.RecipientCount() >= MaxNumberOfRecipients {
		_, _, err := m.send()
		if err != nil {
			return err
		}
		m.to = make([]string, len(m.to))
		m.recipientVariables = make(map[string]map[string]interface{}, len(m.recipientVariables))
	}
	m.to = append(m.to, r)
	if vars != nil {
		if m.recipientVariables == nil {
			m.recipientVariables = make(map[string]map[string]interface{})
		}
		m.recipientVariables[r] = vars
	}
	return nil
}

// RecipientCount returns the total number of recipients for the message.
// This includes To:, Cc:, and Bcc: fields.
//
// NOTE: At present, this method is reliable only for non-MIME messages, as the
// Bcc: and Cc: fields are easily accessible.
// For MIME messages, only the To: field is considered.
// A fix for this issue is planned for a future release.
// For now, MIME messages are always assumed to have 10 recipients between Cc: and Bcc: fields.
// If your MIME messages have more than 10 non-To: field recipients,
// you may find that some recipients will not receive your e-mail.
// It's perfectly OK, of course, for a MIME message to not have any Cc: or Bcc: recipients.
func (m *Message) RecipientCount() int {
	return len(m.to) + m.specific.recipientCount()
}

func (pm *plainMessage) recipientCount() int {
	return len(pm.bcc) + len(pm.cc)
}

func (mm *mimeMessage) recipientCount() int {
	return 10
}

func (m *Message) send() (string, string, error) {
	return m.mg.Send(m)
}

// AddCC appends a receiver to the carbon-copy header of a message.
func (m *Message) AddCC(recipient string) {
	m.specific.addCC(recipient)
}

func (pm *plainMessage) addCC(r string) {
	pm.cc = append(pm.cc, r)
}

func (mm *mimeMessage) addCC(_ string) {}

// AddBCC appends a receiver to the blind-carbon-copy header of a message.
func (m *Message) AddBCC(recipient string) {
	m.specific.addBCC(recipient)
}

func (pm *plainMessage) addBCC(r string) {
	pm.bcc = append(pm.bcc, r)
}

func (mm *mimeMessage) addBCC(_ string) {}

// If you're sending a message that isn't already MIME encoded, SetHtml() will arrange to bundle
// an HTML representation of your message in addition to your plain-text body.
func (m *Message) SetHtml(html string) {
	m.specific.setHtml(html)
}

func (pm *plainMessage) setHtml(h string) {
	pm.html = h
}

func (mm *mimeMessage) setHtml(_ string) {}

// AddTag attaches a tag to the message.  Tags are useful for metrics gathering and event tracking purposes.
// Refer to the Mailgun documentation for further details.
func (m *Message) AddTag(tag string) {
	m.tags = append(m.tags, tag)
}

// This feature is deprecated for new software.
func (m *Message) AddCampaign(campaign string) {
	m.campaigns = append(m.campaigns, campaign)
}

// SetDKIM arranges to send the o:dkim header with the message, and sets its value accordingly.
// Refer to the Mailgun documentation for more information.
func (m *Message) SetDKIM(dkim bool) {
	m.dkim = dkim
	m.dkimSet = true
}

// EnableTestMode allows submittal of a message, such that it will be discarded by Mailgun.
// This facilitates testing client-side software without actually consuming e-mail resources.
func (m *Message) EnableTestMode() {
	m.testMode = true
}

// SetDeliveryTime schedules the message for transmission at the indicated time.
// Pass nil to remove any installed schedule.
// Refer to the Mailgun documentation for more information.
func (m *Message) SetDeliveryTime(dt time.Time) {
	pdt := new(time.Time)
	*pdt = dt
	m.deliveryTime = pdt
}

// SetTracking sets the o:tracking message parameter to adjust, on a message-by-message basis,
// whether or not Mailgun will rewrite URLs to facilitate event tracking.
// Events tracked includes opens, clicks, unsubscribes, etc.
// Note: simply calling this method ensures that the o:tracking header is passed in with the message.
// Its yes/no setting is determined by the call's parameter.
// Note that this header is not passed on to the final recipient(s).
// Refer to the Mailgun documentation for more information.
func (m *Message) SetTracking(tracking bool) {
	m.tracking = tracking
	m.trackingSet = true
}

// Refer to the Mailgun documentation for more information.
func (m *Message) SetTrackingClicks(trackingClicks bool) {
	m.trackingClicks = trackingClicks
	m.trackingClicksSet = true
}

// Refer to the Mailgun documentation for more information.
func (m *Message) SetTrackingOpens(trackingOpens bool) {
	m.trackingOpens = trackingOpens
	m.trackingOpensSet = true
}

// AddHeader allows you to send custom MIME headers with the message.
func (m *Message) AddHeader(header, value string) {
	if m.headers == nil {
		m.headers = make(map[string]string)
	}
	m.headers[header] = value
}

// AddVariable lets you associate a set of variables with messages you send,
// which Mailgun can use to, in essence, complete form-mail.
// Refer to the Mailgun documentation for more information.
func (m *Message) AddVariable(variable string, value interface{}) error {
	j, err := json.Marshal(value)
	if err != nil {
		return err
	}
	if m.variables == nil {
		m.variables = make(map[string]string)
	}
	m.variables[variable] = string(j)
	return nil
}

// Send attempts to queue a message (see Message, NewMessage, and its methods) for delivery.
// It returns the Mailgun server response, which consists of two components:
// a human-readable status message, and a message ID.  The status and message ID are set only
// if no error occurred.
func (m *MailgunImpl) Send(message *Message) (mes string, id string, err error) {
	if !isValid(message) {
		err = errors.New("Message not valid")
	} else {
		payload := simplehttp.NewFormDataPayload()

		message.specific.addValues(payload)
		for _, to := range message.to {
			payload.AddValue("to", to)
		}
		for _, tag := range message.tags {
			payload.AddValue("o:tag", tag)
		}
		for _, campaign := range message.campaigns {
			payload.AddValue("o:campaign", campaign)
		}
		if message.dkimSet {
			payload.AddValue("o:dkim", yesNo(message.dkim))
		}
		if message.deliveryTime != nil {
			payload.AddValue("o:deliverytime", formatMailgunTime(message.deliveryTime))
		}
		if message.testMode {
			payload.AddValue("o:testmode", "yes")
		}
		if message.trackingSet {
			payload.AddValue("o:tracking", yesNo(message.tracking))
		}
		if message.trackingClicksSet {
			payload.AddValue("o:tracking-clicks", yesNo(message.trackingClicks))
		}
		if message.trackingOpensSet {
			payload.AddValue("o:tracking-opens", yesNo(message.trackingOpens))
		}
		if message.headers != nil {
			for header, value := range message.headers {
				payload.AddValue("h:"+header, value)
			}
		}
		if message.variables != nil {
			for variable, value := range message.variables {
				payload.AddValue("v:"+variable, value)
			}
		}
		if message.recipientVariables != nil {
			j, err := json.Marshal(message.recipientVariables)
			if err != nil {
				return "", "", err
			}
			payload.AddValue("recipient-variables", string(j))
		}
		if message.attachments != nil {
			for _, attachment := range message.attachments {
				payload.AddFile("attachment", attachment)
			}
		}
		if message.readerAttachments != nil {
			for _, readerAttachment := range message.readerAttachments {
				payload.AddReadCloser("attachment", readerAttachment.Filename, readerAttachment.ReadCloser)
			}
		}
		if message.inlines != nil {
			for _, inline := range message.inlines {
				payload.AddFile("inline", inline)
			}
		}

		r := simplehttp.NewHTTPRequest(generateApiUrl(m, message.specific.endpoint()))
		r.SetBasicAuth(basicAuthUser, m.ApiKey())

		var response sendMessageResponse
		err = postResponseFromJSON(r, payload, &response)
		if err == nil {
			mes = response.Message
			id = response.Id
		}
	}

	return
}

func (pm *plainMessage) addValues(p *simplehttp.FormDataPayload) {
	p.AddValue("from", pm.from)
	p.AddValue("subject", pm.subject)
	p.AddValue("text", pm.text)
	for _, cc := range pm.cc {
		p.AddValue("cc", cc)
	}
	for _, bcc := range pm.bcc {
		p.AddValue("bcc", bcc)
	}
	if pm.html != "" {
		p.AddValue("html", pm.html)
	}
}

func (mm *mimeMessage) addValues(p *simplehttp.FormDataPayload) {
	p.AddReadCloser("message", "message.mime", mm.body)
}

func (pm *plainMessage) endpoint() string {
	return messagesEndpoint
}

func (mm *mimeMessage) endpoint() string {
	return mimeMessagesEndpoint
}

// yesNo translates a true/false boolean value into a yes/no setting suitable for the Mailgun API.
func yesNo(b bool) string {
	if b {
		return "yes"
	} else {
		return "no"
	}
}

// isValid returns true if, and only if,
// a Message instance is sufficiently initialized to send via the Mailgun interface.
func isValid(m *Message) bool {
	if m == nil {
		return false
	}

	if !m.specific.isValid() {
		return false
	}

	if !validateStringList(m.to, true) {
		return false
	}

	if !validateStringList(m.tags, false) {
		return false
	}

	if !validateStringList(m.campaigns, false) || len(m.campaigns) > 3 {
		return false
	}

	return true
}

func (pm *plainMessage) isValid() bool {
	if pm.from == "" {
		return false
	}

	if !validateStringList(pm.cc, false) {
		return false
	}

	if !validateStringList(pm.bcc, false) {
		return false
	}

	if pm.text == "" {
		return false
	}

	return true
}

func (mm *mimeMessage) isValid() bool {
	return mm.body != nil
}

// validateStringList returns true if, and only if,
// a slice of strings exists AND all of its elements exist,
// OR if the slice doesn't exist AND it's not required to exist.
// The requireOne parameter indicates whether the list is required to exist.
func validateStringList(list []string, requireOne bool) bool {
	hasOne := false

	if list == nil {
		return !requireOne
	} else {
		for _, a := range list {
			if a == "" {
				return false
			} else {
				hasOne = hasOne || true
			}
		}
	}

	return hasOne
}

// GetStoredMessage retrieves information about a received e-mail message.
// This provides visibility into, e.g., replies to a message sent to a mailing list.
func (mg *MailgunImpl) GetStoredMessage(id string) (StoredMessage, error) {
	url := generateStoredMessageUrl(mg, messagesEndpoint, id)
	r := simplehttp.NewHTTPRequest(url)
	r.SetBasicAuth(basicAuthUser, mg.ApiKey())

	var response StoredMessage
	err := getResponseFromJSON(r, &response)
	return response, err
}

// GetStoredMessageRaw retrieves the raw MIME body of a received e-mail message.
// Compared to GetStoredMessage, it gives access to the unparsed MIME body, and
// thus delegates to the caller the required parsing.
func (mg *MailgunImpl) GetStoredMessageRaw(id string) (StoredMessageRaw, error) {
	url := generateStoredMessageUrl(mg, messagesEndpoint, id)
	r := simplehttp.NewHTTPRequest(url)
	r.SetBasicAuth(basicAuthUser, mg.ApiKey())
	r.AddHeader("Accept", "message/rfc2822")

	var response StoredMessageRaw
	err := getResponseFromJSON(r, &response)
	return response, err

}

// DeleteStoredMessage removes a previously stored message.
// Note that Mailgun institutes a policy of automatically deleting messages after a set time.
// Consult the current Mailgun API documentation for more details.
func (mg *MailgunImpl) DeleteStoredMessage(id string) error {
	url := generateStoredMessageUrl(mg, messagesEndpoint, id)
	r := simplehttp.NewHTTPRequest(url)
	r.SetBasicAuth(basicAuthUser, mg.ApiKey())
	_, err := makeDeleteRequest(r)
	return err
}
