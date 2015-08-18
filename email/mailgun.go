package email

import (
	"encoding/json"
	"errors"
	"expvar"

	"github.com/coreos/dex/pkg/log"
	mailgun "github.com/mailgun/mailgun-go"
)

const (
	MailgunEmailerType = "mailgun"
)

var (
	counterEmailSendErr = expvar.NewInt("mailgun.send.err")
)

func init() {
	RegisterEmailerConfigType(MailgunEmailerType, func() EmailerConfig { return &MailgunEmailerConfig{} })
}

type MailgunEmailerConfig struct {
	ID            string `json:"id"`
	PrivateAPIKey string `json:"privateAPIKey"`
	PublicAPIKey  string `json:"publicAPIKey"`
	Domain        string `json:"domain"`
}

func (cfg MailgunEmailerConfig) EmailerType() string {
	return MailgunEmailerType
}

func (cfg MailgunEmailerConfig) EmailerID() string {
	return cfg.ID
}

func (cfg MailgunEmailerConfig) Emailer() (Emailer, error) {
	mg := mailgun.NewMailgun(cfg.Domain, cfg.PrivateAPIKey, cfg.PublicAPIKey)
	return &mailgunEmailer{
		mg: mg,
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
	mg mailgun.Mailgun
}

func (m *mailgunEmailer) SendMail(from, subject, text, html string, to ...string) error {
	msg := m.mg.NewMessage(from, subject, text, to...)
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
