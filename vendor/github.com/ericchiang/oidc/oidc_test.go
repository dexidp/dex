package oidc

import "testing"

func TestClientVerifier(t *testing.T) {
	tests := []struct {
		clientID string
		payload  string
		wantErr  bool
	}{
		{
			clientID: "1",
			payload:  `{"aud":"1"}`,
		},
		{
			clientID: "1",
			payload:  `{"aud":"2"}`,
			wantErr:  true,
		},
		{
			clientID: "1",
			payload:  `{"aud":["1"]}`,
		},
		{
			clientID: "1",
			payload:  `{"aud":["1", "2"]}`,
		},
		{
			clientID: "3",
			payload:  `{"aud":["1", "2"]}`,
			wantErr:  true,
		},
		{
			clientID: "3",
			payload:  `{"aud":}`, // invalid JSON
			wantErr:  true,
		},
		{
			clientID: "1",
			payload:  `{}`,
			wantErr:  true,
		},
	}

	for i, tc := range tests {
		err := (clientVerifier{tc.clientID}).verifyIDTokenPayload([]byte(tc.payload))
		if err != nil && !tc.wantErr {
			t.Errorf("case %d: %v", i)
		}
		if err == nil && tc.wantErr {
			t.Errorf("case %d: expected error")
		}
	}
}
