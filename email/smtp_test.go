package email

import (
	"encoding/json"
	"strconv"
	"testing"

	"github.com/kylelemons/godebug/pretty"

	"gopkg.in/gomail.v2"
)

func TestNewSmtpEmailer(t *testing.T) {
	// If (and only if) this port is provided, gomail assumes SSL.
	gomailSSLPort := 465

	tests := []struct {
		config SmtpEmailerConfig

		// formAddr set by the dex-worker flag
		fromAddrFlag string

		wantEmailer Emailer
		wantErr     bool
	}{
		{
			config: SmtpEmailerConfig{
				Host:     "example.com:" + strconv.Itoa(gomailSSLPort),
				FromAddr: "foo@example.com",
			},
			wantEmailer: &smtpEmailer{
				from: "foo@example.com",
				dialer: &gomail.Dialer{
					Host: "example.com",
					Port: gomailSSLPort,
					SSL:  true,
				},
			},
		},
		{
			config: SmtpEmailerConfig{
				Host:     "example.com",
				Port:     gomailSSLPort,
				FromAddr: "foo@example.com",
			},
			wantEmailer: &smtpEmailer{
				from: "foo@example.com",
				dialer: &gomail.Dialer{
					Host: "example.com",
					Port: gomailSSLPort,
					SSL:  true,
				},
			},
		},
		{
			config: SmtpEmailerConfig{
				Host:     "example.com",
				Port:     80,
				FromAddr: "foo@example.com",
			},
			wantEmailer: &smtpEmailer{
				from: "foo@example.com",
				dialer: &gomail.Dialer{
					Host: "example.com",
					Port: 80,
				},
			},
		},
		{
			// No port provided.
			config: SmtpEmailerConfig{
				Host:     "example.com",
				FromAddr: "foo@example.com",
			},
			wantErr: true,
		},
		{
			config: SmtpEmailerConfig{
				Host:     "example.com",
				Port:     80,
				FromAddr: "foo@example.com",
			},
			fromAddrFlag: "bar@example.com",
			wantEmailer: &smtpEmailer{
				from: "foo@example.com", // config should override flag.
				dialer: &gomail.Dialer{
					Host: "example.com",
					Port: 80,
				},
			},
		},
		{
			// No fromAddr provided as a flag or in config.
			config:  SmtpEmailerConfig{Host: "example.com"},
			wantErr: true,
		},
		{
			config: SmtpEmailerConfig{
				Host:     "example.com",
				Port:     80,
				Username: "foo",
				Password: "bar",
				FromAddr: "foo@example.com",
			},
			wantEmailer: &smtpEmailer{
				from:   "foo@example.com", // config should override flag.
				dialer: gomail.NewPlainDialer("example.com", 80, "foo", "bar"),
			},
		},
		{
			// Password provided without username.
			config: SmtpEmailerConfig{
				Host:     "example.com",
				Port:     80,
				Password: "bar",
				FromAddr: "foo@example.com",
			},
			wantErr: true,
		},
	}

	for i, tt := range tests {
		testCase, err := json.MarshalIndent(tt.config, "", "  ")
		if err != nil {
			t.Fatal(err)
		}

		emailer, err := tt.config.Emailer(tt.fromAddrFlag)
		if err != nil {
			if !tt.wantErr {
				t.Errorf("case %d %s.Emailer(): %v", i, testCase, err)
			}
			continue
		}
		if tt.wantErr {
			t.Errorf("case %d %s.Emailer(): expected error creating emailer", i, testCase)
			continue
		}

		if diff := pretty.Compare(emailer, tt.wantEmailer); diff != "" {
			t.Errorf("case %d: unexpected emailer %s", i, diff)
		}
	}
}
