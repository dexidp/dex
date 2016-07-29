package email

import (
	"encoding/json"
	"errors"

	"github.com/coreos/dex/pkg/log"
	mailgun "github.com/mailgun/mailgun-go"
)

const (
	MailgunEmailerType = "mailgun"
)

func init() {
	RegisterEmailerConfigType(MailgunEmailerType, func() EmailerConfig { return &MailgunEmailerConfig{} })
}

type MailgunEmailerConfig struct {
	FromAddr      string `json:"from"`
	PrivateAPIKey string `json:"privateAPIKey"`
	PublicAPIKey  string `json:"publicAPIKey"`
	Domain        string `json:"domain"`
}

func (cfg MailgunEmailerConfig) EmailerType() string {
	return MailgunEmailerType
}

func (cfg MailgunEmailerConfig) EmailerID() string {
	return MailgunEmailerType
}

func (cfg MailgunEmailerConfig) Emailer(fromAddr string) (Emailer, error) {
	from := cfg.FromAddr
	if from == "" {
		from = fromAddr
	}

	if from == "" {
		return nil, errors.New(`missing "from" field in email config`)
	}
	mg := mailgun.NewMailgun(cfg.Domain, cfg.PrivateAPIKey, cfg.PublicAPIKey)
	return &mailgunEmailer{
		mg:   mg,
		from: from,
	}, nil
}

// mailgunEmailerConfig exists to avoid recusion.
type mailgunEmailerConfig MailgunEmailerConfig

func (cfg *MailgunEmailerConfig) UnmarshalJSON(data []byte) error {
	mgtmp := mailgunEmailerConfig{}
	err := json.Unmarshal(data, &mgtmp)
	if err != nil {
		return err
	}

	if mgtmp.PrivateAPIKey == "" {
		return errors.New("must have a privateAPIKey set")
	}

	if mgtmp.PublicAPIKey == "" {
		return errors.New("must have a publicAPIKey set")
	}

	if mgtmp.Domain == "" {
		return errors.New("must have a domain set")
	}

	*cfg = MailgunEmailerConfig(mgtmp)
	return nil
}

type mailgunEmailer struct {
	mg   mailgun.Mailgun
	from string
}

func (m *mailgunEmailer) SendMail(subject, text, html string, to ...string) error {
	msg := m.mg.NewMessage(m.from, subject, text, to...)
	if html != "" {
		msg.SetHtml(html)
	}
	mes, id, err := m.mg.Send(msg)
	if err != nil {
		counterEmailSendErr.Add(1)
		return err
	}
	log.Infof("SendMail: msgID: %v: %q", id, mes)
	return nil
}
