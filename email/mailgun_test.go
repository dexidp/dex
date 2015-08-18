package email

import (
	"strings"
	"testing"

	"github.com/kylelemons/godebug/pretty"
)

func TestNewEmailConfigFromReader(t *testing.T) {
	tests := []struct {
		json    string
		want    MailgunEmailerConfig
		wantErr bool
	}{
		{
			json: `{"type":"mailgun","id":"mg","privateAPIKey":"private","publicAPIKey":"public","domain":"example.com"}`,
			want: MailgunEmailerConfig{
				ID:            "mg",
				PrivateAPIKey: "private",
				PublicAPIKey:  "public",
				Domain:        "example.com",
			},
		},
		{
			json:    `{"type":"mailgun","id":"mg","publicAPIKey":"public","domain":"example.com"}`,
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
	}

	for i, tt := range tests {
		r := strings.NewReader(tt.json)
		ec, err := newEmailerConfigFromReader(r)
		if tt.wantErr {
			if err == nil {
				t.Errorf("case %d: want non-nil err.", i)
			}
			t.Logf("WHAT: %v", err)
			continue
		}
		if err != nil {
			t.Errorf("case %d: want nil err: %v", i, err)
			continue
		}
		if diff := pretty.Compare(tt.want, ec); diff != "" {
			t.Errorf("case %d: Compare(want, got): %v", i, diff)
		}
	}
}
