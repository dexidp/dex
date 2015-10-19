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

	goodInvitation := NewInvitation(user, password, *issuer, client, *callback, expires)
	goodNoCB := NewInvitation(user, password, *issuer, client, *callback, expires)
	expired := NewInvitation(user, password, *issuer, client, *callback, -expires)
	wrongIssuer := NewInvitation(user, password, *notIssuer, client, *callback, expires)
	noSub := NewInvitation(User{Email: "noid@noid.com"}, password, *issuer, client, *callback, expires)
	noEmail := NewInvitation(User{ID: "JONNY_NO_EMAIL"}, password, *issuer, client, *callback, expires)
	noPassword := NewInvitation(user, Password(""), *issuer, client, *callback, expires)
	noClient := NewInvitation(user, password, *issuer, "", *callback, expires)
	noClientNoCB := NewInvitation(user, password, *issuer, "", url.URL{}, expires)

	tests := []struct {
		invite  Invitation
		wantErr bool
		signer  jose.Signer
	}{
		{
			invite:  goodInvitation,
			signer:  signer,
			wantErr: false,
		},
		{
			invite:  goodNoCB,
			signer:  signer,
			wantErr: false,
		},
		{
			invite:  expired,
			signer:  signer,
			wantErr: true,
		},
		{
			invite:  wrongIssuer,
			signer:  signer,
			wantErr: true,
		},
		{
			invite:  noSub,
			signer:  signer,
			wantErr: true,
		},
		{
			invite:  noEmail,
			signer:  signer,
			wantErr: true,
		},
		{
			invite:  noPassword,
			signer:  signer,
			wantErr: true,
		},
		{
			invite:  noClient,
			signer:  signer,
			wantErr: true,
		},
		{
			invite:  noClientNoCB,
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
