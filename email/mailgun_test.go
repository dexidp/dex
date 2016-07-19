package email

import (
	"strings"
	"testing"

	"github.com/kylelemons/godebug/pretty"
)

func TestNewEmailConfigFromReader(t *testing.T) {
	tests := []struct {
		json        string
		want        MailgunEmailerConfig
		wantErr     bool
		wantInitErr bool // want error when calling Emailer() with no fromAddr
	}{
		{
			json: `{"type":"mailgun","id":"mg","privateAPIKey":"private","publicAPIKey":"public","domain":"example.com","from":"admin@example.com"}`,
			want: MailgunEmailerConfig{
				PrivateAPIKey: "private",
				PublicAPIKey:  "public",
				Domain:        "example.com",
				FromAddr:      "admin@example.com",
			},
		},
		{
			json:    `{"type":"mailgun","id":"mg","publicAPIKey":"public","domain":"example.com",""}`,
			wantErr: true,
		},
		{
			json:    `{"type":"mailgun","id":"mg","privateAPIKey":"private","publicAPIKey":"public"}`,
			wantErr: true,
		},
		{
			json:    `{"type":"mailgun","id":"mg","privateAPIKey":"private","domain":"example.com"}`,
			wantErr: true,
		},
		{
			json: `{"type":"mailgun","id":"mg","privateAPIKey":"private","publicAPIKey":"public","domain":"example.com"}`,
			want: MailgunEmailerConfig{
				PrivateAPIKey: "private",
				PublicAPIKey:  "public",
				Domain:        "example.com",
			},

			// No fromAddr email provided. Calling Emailer("") should error since fromAddr needs to be provided
			// in the config or as a command line argument.
			wantInitErr: true,
		},
	}

	for i, tt := range tests {
		r := strings.NewReader(tt.json)
		ec, err := newEmailerConfigFromReader(r)
		if tt.wantErr {
			if err == nil {
				t.Errorf("case %d: want non-nil err.", i)
			}
			continue
		}
		if err != nil {
			t.Errorf("case %d: want nil err: %v", i, err)
			continue
		}
		if diff := pretty.Compare(tt.want, ec); diff != "" {
			t.Errorf("case %d: Compare(want, got): %v", i, diff)
		}

		_, err = ec.Emailer("")
		if err != nil && !tt.wantInitErr {
			t.Errorf("case %d: failed to initialize emailer: %v", i, err)
		}
		if err == nil && tt.wantInitErr {
			t.Errorf("case %d: expected error initializing emailer", i)
		}
	}
}
