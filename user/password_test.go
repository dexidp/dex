package user

import (
	"net/url"
	"strings"
	"testing"
	"time"

	"github.com/jonboulle/clockwork"
	"github.com/kylelemons/godebug/pretty"
	"golang.org/x/crypto/bcrypt"

	"github.com/coreos/go-oidc/jose"
	"github.com/coreos/go-oidc/key"
)

func TestNewPasswordInfosFromReader(t *testing.T) {
	PasswordHasher = func(plaintext string) ([]byte, error) {
		return []byte(strings.ToUpper(plaintext)), nil
	}
	defer func() {
		PasswordHasher = DefaultPasswordHasher
	}()

	tests := []struct {
		json string
		want []PasswordInfo
	}{
		{
			json: `[{"userId":"12345","passwordPlaintext":"password"},{"userId":"78901","passwordHash":"WORDPASS", "passwordExpires":"2006-01-01T15:04:05Z"}]`,
			want: []PasswordInfo{
				{
					UserID:   "12345",
					Password: []byte("PASSWORD"),
				},
				{
					UserID:   "78901",
					Password: []byte("WORDPASS"),
					PasswordExpires: time.Date(2006,
						1, 1, 15, 4, 5, 0, time.UTC),
				},
			},
		},
	}

	for i, tt := range tests {
		r := strings.NewReader(tt.json)
		us, err := newPasswordInfosFromReader(r)
		if err != nil {
			t.Errorf("case %d: want nil err: %v", i, err)
			continue
		}
		if diff := pretty.Compare(tt.want, us); diff != "" {
			t.Errorf("case %d: Compare(want, got): %v", i, diff)
		}
	}
}

func TestNewPasswordFromHash(t *testing.T) {
	tests := []string{
		"test",
		"1",
	}

	for i, plaintext := range tests {
		p, err := NewPasswordFromPlaintext(plaintext)
		if err != nil {
			t.Errorf("case %d: unexpected error: %q", i, err)
			continue
		}
		if err = bcrypt.CompareHashAndPassword([]byte(p), []byte(plaintext)); err != nil {
			t.Errorf("case %d: err comparing hash and plaintext: %q", i, err)
		}
	}
}

func TestNewPasswordReset(t *testing.T) {
	clock = clockwork.NewFakeClock()
	defer func() {
		clock = clockwork.NewRealClock()
	}()

	now := clock.Now()

	issuer, _ := url.Parse("http://example.com")
	clientID := "myclient"
	usr := User{ID: "123456", Email: "user@example.com"}
	callback := "http://client.example.com/callback"
	expires := time.Hour * 3
	password := Password("passy")

	tests := []struct {
		user     User
		password Password
		issuer   url.URL
		clientID string
		callback string
		expires  time.Duration
		want     jose.Claims
	}{
		{
			issuer:   *issuer,
			clientID: clientID,
			user:     usr,
			callback: callback,
			expires:  expires,
			password: password,
			want: map[string]interface{}{
				"iss": issuer.String(),
				"aud": clientID,
				ClaimPasswordResetCallback: callback,
				ClaimPasswordResetPassword: string(password),
				"exp": float64(now.Add(expires).Unix()),
				"sub": usr.ID,
				"iat": float64(now.Unix()),
			},
		},
	}

	for i, tt := range tests {
		cbURL, err := url.Parse(tt.callback)
		if err != nil {
			t.Fatalf("case %d: non-nil err: %q", i, err)
		}
		ev := NewPasswordReset(tt.user.ID, tt.password, tt.issuer, tt.clientID, *cbURL, tt.expires)

		if diff := pretty.Compare(tt.want, ev.Claims); diff != "" {
			t.Errorf("case %d: Compare(want, got): %v", i, diff)
		}

		if diff := pretty.Compare(ev.Password(), password); diff != "" {
			t.Errorf("case %d: Compare(want, got): %v", i, diff)
		}
	}
}

func TestPasswordResetParseAndVerify(t *testing.T) {

	issuer, _ := url.Parse("http://example.com")
	otherIssuer, _ := url.Parse("http://bad.example.com")
	client := "myclient"
	user := User{ID: "1234", Email: "user@example.com"}
	callback, _ := url.Parse("http://client.example.com")
	expires := time.Hour * 3
	password := Password("passy")
	userID := user.ID

	goodPR := NewPasswordReset(userID, password, *issuer, client, *callback, expires)
	goodPRNoCB := NewPasswordReset(userID, password, *issuer, client, url.URL{}, expires)
	expiredPR := NewPasswordReset(userID, password, *issuer, client, *callback, -expires)
	wrongIssuerPR := NewPasswordReset(userID, password, *otherIssuer, client, *callback, expires)
	noSubPR := NewPasswordReset("", password, *issuer, client, *callback, expires)
	noPWPR := NewPasswordReset(userID, Password(""), *issuer, client, *callback, expires)
	noClientPR := NewPasswordReset(userID, password, *issuer, "", *callback, expires)
	noClientNoCBPR := NewPasswordReset(userID, password, *issuer, "", url.URL{}, expires)

	privKey, err := key.GeneratePrivateKey()
	if err != nil {
		t.Fatalf("Failed to generate private key, error=%v", err)
	}
	signer := privKey.Signer()

	privKey2, err := key.GeneratePrivateKey()
	if err != nil {
		t.Fatalf("Failed to generate private key, error=%v", err)
	}
	otherSigner := privKey2.Signer()

	tests := []struct {
		ev      PasswordReset
		wantErr bool
		signer  jose.Signer
	}{

		{
			ev:      goodPR,
			signer:  signer,
			wantErr: false,
		},
		{
			ev:      goodPRNoCB,
			signer:  signer,
			wantErr: false,
		},

		{
			ev:      expiredPR,
			signer:  signer,
			wantErr: true,
		},
		{
			ev:      wrongIssuerPR,
			signer:  signer,
			wantErr: true,
		},
		{
			ev:      goodPR,
			signer:  otherSigner,
			wantErr: true,
		},
		{
			ev:      noSubPR,
			signer:  signer,
			wantErr: true,
		},
		{
			ev:      noPWPR,
			signer:  signer,
			wantErr: true,
		},
		{
			ev:      noClientPR,
			signer:  signer,
			wantErr: true,
		},
		{
			ev:      noClientNoCBPR,
			signer:  signer,
			wantErr: true,
		},
	}

	for i, tt := range tests {

		jwt, err := jose.NewSignedJWT(tt.ev.Claims, tt.signer)
		if err != nil {
			t.Fatalf("Failed to generate JWT, error=%v", err)
		}
		token := jwt.Encode()

		ev, err := ParseAndVerifyPasswordResetToken(token, *issuer,
			[]key.PublicKey{*key.NewPublicKey(privKey.JWK())})

		if tt.wantErr {
			t.Logf("err: %v", err)
			if err == nil {
				t.Errorf("case %d: want non-nil err, got nil", i)
			}
			continue
		}

		if err != nil {
			t.Errorf("case %d: non-nil err: %q", i, err)

		}

		if diff := pretty.Compare(tt.ev.Claims, ev.Claims); diff != "" {
			t.Errorf("case %d: Compare(want, got): %v", i, diff)
		}
	}
}
