package email

import (
	"encoding/json"
	"errors"

	"gopkg.in/gomail.v2"
)

const (
	SmtpEmailerType = "smtp"
)

func init() {
	RegisterEmailerConfigType(SmtpEmailerType, func() EmailerConfig { return &SmtpEmailerConfig{} })
}

type SmtpEmailerConfig struct {
	Host     string `json:"host"`
	Port     int    `json:"port"`
	Auth     string `json:"auth"`
	Username string `json:"username"`
	Password string `json:"password"`
}

func (cfg SmtpEmailerConfig) EmailerType() string {
	return SmtpEmailerType
}

func (cfg SmtpEmailerConfig) EmailerID() string {
	return SmtpEmailerType
}

func (cfg SmtpEmailerConfig) Emailer() (Emailer, error) {
	var dialer *gomail.Dialer
	if cfg.Auth == "plain" {
		dialer = gomail.NewPlainDialer(cfg.Host, cfg.Port, cfg.Username, cfg.Password)
	} else {
		dialer = &gomail.Dialer{
			Host: cfg.Host,
			Port: cfg.Port,
		}
	}
	return &smtpEmailer{
		dialer: dialer,
	}, nil
}

type smtpEmailerConfig SmtpEmailerConfig

func (cfg *SmtpEmailerConfig) UnmarshalJSON(data []byte) error {
	smtpCfg := smtpEmailerConfig{}
	err := json.Unmarshal(data, &smtpCfg)
	if err != nil {
		return err
	}
	if smtpCfg.Host == "" {
		return errors.New("must set SMTP host")
	}
	if smtpCfg.Port == 0 {
		return errors.New("must set SMTP port")
	}
	*cfg = SmtpEmailerConfig(smtpCfg)
	return nil
}

type smtpEmailer struct {
	dialer *gomail.Dialer
}

func (emailer *smtpEmailer) SendMail(from, subject, text, html string, to ...string) error {
	msg := gomail.NewMessage()
	msg.SetHeader("From", from)
	msg.SetHeader("To", to...)
	msg.SetHeader("Subject", subject)
	msg.SetBody("text/plain", text)
	msg.SetBody("text/html", html)
	err := emailer.dialer.DialAndSend(msg)
	if err != nil {
		counterEmailSendErr.Add(1)
		return err
	}
	return nil
}
