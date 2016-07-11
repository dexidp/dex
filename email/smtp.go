package email

import (
	"encoding/json"
	"errors"
	"fmt"
	"net"
	"strconv"

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
	Username string `json:"username"`
	Password string `json:"password"`
	FromAddr string `json:"from"`

	// OPTIONAL: If empty and host is of form "host:port" just use that. For backward
	// compatibility do not change this.
	Port int `json:"port"`

	// DEPRICATED: If "username" and "password" are provided, use them.
	Auth string `json:"auth"`
}

func (cfg SmtpEmailerConfig) EmailerType() string {
	return SmtpEmailerType
}

func (cfg SmtpEmailerConfig) EmailerID() string {
	return SmtpEmailerType
}

func (cfg SmtpEmailerConfig) Emailer(fromAddr string) (Emailer, error) {
	from := cfg.FromAddr
	if from == "" {
		from = fromAddr
	}
	if from == "" {
		return nil, errors.New(`missing "from" field in email config`)
	}

	host, port := cfg.Host, cfg.Port

	// If port hasn't been supplied, check the "host" field.
	if port == 0 {
		hostStr, portStr, err := net.SplitHostPort(cfg.Host)
		if err != nil {
			return nil, fmt.Errorf(`"host" must be in format of "host:port" %v`, err)
		}
		host = hostStr
		if port, err = strconv.Atoi(portStr); err != nil {
			return nil, fmt.Errorf(`failed to parse %q as "host:port" %v`, cfg.Host, err)
		}
	}

	if (cfg.Username == "") != (cfg.Password == "") {
		return nil, errors.New(`must provide both "username" and "password"`)
	}

	var dialer *gomail.Dialer
	if cfg.Username == "" {
		// NOTE(ericchiang): Guess SSL using the same logic as gomail. We should
		// eventually allow this to be set explicitly.
		dialer = &gomail.Dialer{Host: host, Port: port, SSL: port == 465}
	} else {
		dialer = gomail.NewPlainDialer(host, port, cfg.Username, cfg.Password)
	}

	return &smtpEmailer{dialer: dialer, from: from}, nil
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
	from   string
}

func (emailer *smtpEmailer) SendMail(subject, text, html string, to ...string) error {
	msg := gomail.NewMessage()
	msg.SetHeader("From", emailer.from)
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
