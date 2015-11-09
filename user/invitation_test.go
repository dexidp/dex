package user

import (
	"net/url"
	"testing"
	"time"

	"github.com/kylelemons/godebug/pretty"

	"github.com/coreos/go-oidc/jose"
	"github.com/coreos/go-oidc/key"
)

func TestInvitationParseAndVerify(t *testing.T) {
	issuer, _ := url.Parse("http://example.com")
	notIssuer, _ := url.Parse("http://other.com")
	client := "myclient"
	user := User{ID: "1234", Email: "user@example.com"}
	callback, _ := url.Parse("http://client.example.com")
	expires := time.Hour * 3
	password := Password("Halloween is the best holiday")
	privKey, _ := key.GeneratePrivateKey()
	signer := privKey.Signer()
	publicKeys := []key.PublicKey{*key.NewPublicKey(privKey.JWK())}

	tests := []struct {
		invite  Invitation
		wantErr bool
		signer  jose.Signer
	}{
		{
			invite:  NewInvitation(user, password, *issuer, client, *callback, expires),
			signer:  signer,
			wantErr: false,
		},
		{
			invite:  NewInvitation(user, password, *issuer, client, *callback, expires),
			signer:  signer,
			wantErr: false,
		},
		{
			invite:  NewInvitation(user, password, *issuer, client, *callback, -expires),
			signer:  signer,
			wantErr: true,
		},
		{
			invite:  NewInvitation(user, password, *notIssuer, client, *callback, expires),
			signer:  signer,
			wantErr: true,
		},
		{
			invite:  NewInvitation(User{Email: "noid@noid.com"}, password, *issuer, client, *callback, expires),
			signer:  signer,
			wantErr: true,
		},
		{
			invite:  NewInvitation(User{ID: "JONNY_NO_EMAIL"}, password, *issuer, client, *callback, expires),
			signer:  signer,
			wantErr: true,
		},
		{
			invite:  NewInvitation(user, Password(""), *issuer, client, *callback, expires),
			signer:  signer,
			wantErr: true,
		},
		{
			invite:  NewInvitation(user, password, *issuer, "", *callback, expires),
			signer:  signer,
			wantErr: true,
		},
		{
			invite:  NewInvitation(user, password, *issuer, "", url.URL{}, expires),
			signer:  signer,
			wantErr: true,
		},
	}

	for i, tt := range tests {
		jwt, err := jose.NewSignedJWT(tt.invite.Claims, tt.signer)
		if err != nil {
			t.Fatalf("case %d: failed to generate JWT, error: %v", i, err)
		}
		token := jwt.Encode()

		parsed, err := ParseAndVerifyInvitationToken(token, *issuer, publicKeys)

		if tt.wantErr {
			if err == nil {
				t.Errorf("case %d: want no-nil error, got nil", i)
			}
			continue
		}

		if err != nil {
			t.Errorf("case %d: unexpected error: %v", i, err)
			continue
		}

		if diff := pretty.Compare(tt.invite, parsed); diff != "" {
			t.Errorf("case %d: Compare(want, got): %v", i, diff)
		}
	}
}
