package signer

import (
	"context"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestKeySetVerifySignatureTyp(t *testing.T) {
	ctx := context.Background()
	s, err := (&MockConfig{}).Open(ctx)
	require.NoError(t, err)
	ks := &KeySet{Signer: s}
	payload := []byte(`{"sub":"test-user"}`)

	for _, tc := range []struct {
		typ     string
		wantErr bool
	}{
		{typ: ""},
		{typ: "JWT"},
		{typ: "jwt"},
		{typ: "oauth-id-jag+jwt", wantErr: true},
		{typ: "at+jwt", wantErr: true},
	} {
		t.Run(tc.typ, func(t *testing.T) {
			token, err := s.SignWithType(ctx, payload, tc.typ)
			require.NoError(t, err)
			got, err := ks.VerifySignature(ctx, token)
			if tc.wantErr {
				require.Error(t, err)
				return
			}
			require.NoError(t, err)
			require.Equal(t, payload, got)
		})
	}
}
