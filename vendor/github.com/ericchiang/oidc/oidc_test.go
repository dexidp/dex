package oidc

import (
	"encoding/json"
	"reflect"
	"testing"
)

func TestClientVerifier(t *testing.T) {
	tests := []struct {
		clientID string
		aud      []string
		wantErr  bool
	}{
		{
			clientID: "1",
			aud:      []string{"1"},
		},
		{
			clientID: "1",
			aud:      []string{"2"},
			wantErr:  true,
		},
		{
			clientID: "1",
			aud:      []string{"2", "1"},
		},
		{
			clientID: "3",
			aud:      []string{"1", "2"},
			wantErr:  true,
		},
	}

	for i, tc := range tests {
		token := IDToken{Audience: tc.aud}
		err := (clientVerifier{tc.clientID}).verifyIDToken(&token)
		if err != nil && !tc.wantErr {
			t.Errorf("case %d: %v", i)
		}
		if err == nil && tc.wantErr {
			t.Errorf("case %d: expected error")
		}
	}
}

func TestUnmarshalAudience(t *testing.T) {
	tests := []struct {
		data    string
		want    audience
		wantErr bool
	}{
		{`"foo"`, audience{"foo"}, false},
		{`["foo","bar"]`, audience{"foo", "bar"}, false},
		{"foo", nil, true}, // invalid JSON
	}

	for _, tc := range tests {
		var a audience
		if err := json.Unmarshal([]byte(tc.data), &a); err != nil {
			if !tc.wantErr {
				t.Errorf("failed to unmarshal %q: %v", tc.data, err)
			}
			continue
		}

		if tc.wantErr {
			t.Errorf("did not expected to be able to unmarshal %q", tc.data)
			continue
		}

		if !reflect.DeepEqual(tc.want, a) {
			t.Errorf("from %q expected %q got %q", tc.data, tc.want, a)
		}
	}
}
