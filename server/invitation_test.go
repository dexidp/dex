package server

import (
	"bytes"
	"fmt"
	"net/http"
	"net/http/httptest"
	"net/url"
	"testing"
	"time"

	"github.com/jonboulle/clockwork"

	"github.com/coreos/dex/user"
	"github.com/coreos/go-oidc/jose"
	"github.com/coreos/go-oidc/key"
)

var (
	clock = clockwork.NewRealClock()
)

func TestInvitationHandler(t *testing.T) {
	invUserID := "ID-1"
	invVerifiedID := "ID-Verified"
	invGoodSigner := key.NewPrivateKeySet([]*key.PrivateKey{testPrivKey},
		time.Now().Add(time.Minute)).Active().Signer()

	badKey, err := key.GeneratePrivateKey()
	if err != nil {
		panic(fmt.Sprintf("couldn't make new key: %q", err))
	}

	invBadSigner := key.NewPrivateKeySet([]*key.PrivateKey{badKey},
		time.Now().Add(time.Minute)).Active().Signer()

	makeInvitationToken := func(password, userID, clientID, email string, callback url.URL, expires time.Duration, signer jose.Signer) string {
		iv := user.NewInvitation(
			user.User{ID: userID, Email: email},
			user.Password(password),
			testIssuerURL,
			clientID,
			callback,
			expires)

		jwt, err := jose.NewSignedJWT(iv.Claims, signer)
		if err != nil {
			t.Fatalf("couldn't make token: %q", err)
		}
		token := jwt.Encode()
		return token
	}

	tests := []struct {
		userID            string
		query             url.Values
		signer            jose.Signer
		wantCode          int
		wantCallback      url.URL
		wantEmailVerified bool
	}{
		{ // Case 0 Happy Path
			userID: invUserID,
			query: url.Values{
				"token": []string{makeInvitationToken("password", invUserID, testClientID, "Email-1@example.com", testRedirectURL, time.Hour*1, invGoodSigner)},
			},
			signer:            invGoodSigner,
			wantCode:          http.StatusSeeOther,
			wantCallback:      testRedirectURL,
			wantEmailVerified: true,
		},
		{ // Case 1 user already verified
			userID: invVerifiedID,
			query: url.Values{
				"token": []string{makeInvitationToken("password", invVerifiedID, testClientID, "Email-Verified@example.com", testRedirectURL, time.Hour*1, invGoodSigner)},
			},
			signer:            invGoodSigner,
			wantCode:          http.StatusSeeOther,
			wantCallback:      testRedirectURL,
			wantEmailVerified: true,
		},
		{ // Case 2 bad email
			userID: invUserID,
			query: url.Values{
				"token": []string{makeInvitationToken("password", invVerifiedID, testClientID, "NOPE@NOPE.com", testRedirectURL, time.Hour*1, invGoodSigner)},
			},
			signer:            invGoodSigner,
			wantCode:          http.StatusBadRequest,
			wantCallback:      testRedirectURL,
			wantEmailVerified: false,
		},
		{ // Case 3 bad signer
			userID: invUserID,
			query: url.Values{
				"token": []string{makeInvitationToken("password", invUserID, testClientID, "Email-1@example.com", testRedirectURL, time.Hour*1, invBadSigner)},
			},
			signer:            invGoodSigner,
			wantCode:          http.StatusBadRequest,
			wantCallback:      testRedirectURL,
			wantEmailVerified: false,
		},
	}

	for i, tt := range tests {
		f, err := makeTestFixtures()
		if err != nil {
			t.Fatalf("case %d: could not make test fixtures: %v", i, err)
		}

		keys, err := f.srv.KeyManager.PublicKeys()
		if err != nil {
			t.Fatalf("case %d: test fixture key infrastructure is broken: %v", i, err)
		}

		tZero := clock.Now()
		handler := &InvitationHandler{
			passwordResetURL:       f.srv.absURL("RESETME"),
			issuerURL:              testIssuerURL,
			um:                     f.srv.UserManager,
			keysFunc:               f.srv.KeyManager.PublicKeys,
			signerFunc:             func() (jose.Signer, error) { return tt.signer, nil },
			redirectValidityWindow: 100 * time.Second,
		}

		w := httptest.NewRecorder()
		u := testIssuerURL
		u.RawQuery = tt.query.Encode()
		req, err := http.NewRequest("GET", u.String(), nil)
		if err != nil {
			t.Fatalf("case %d: impossible error: %v", i, err)
		}

		handler.ServeHTTP(w, req)

		if tt.wantCode != w.Code {
			t.Errorf("case %d: wantCode=%v, got=%v", i, tt.wantCode, w.Code)
			continue
		}

		usr, err := f.srv.UserManager.Get(tt.userID)
		if err != nil {
			t.Fatalf("case %d: unexpected error: %v", i, err)
		}

		if usr.EmailVerified != tt.wantEmailVerified {
			t.Errorf("case %d: wantEmailVerified=%v got=%v", i, tt.wantEmailVerified, usr.EmailVerified)
		}

		if w.Code == http.StatusSeeOther {
			locString := w.HeaderMap.Get("Location")
			loc, err := url.Parse(locString)
			if err != nil {
				t.Fatalf("case %d: redirect returned nonsense url: '%v', %v", i, locString, err)
			}

			pwrToken := loc.Query().Get("token")
			pwrReset, err := user.ParseAndVerifyPasswordResetToken(pwrToken, testIssuerURL, keys)
			if err != nil {
				t.Errorf("case %d: password token is invalid: %v", i, err)
			}

			expTime := pwrReset.Claims["exp"].(float64)
			if expTime > float64(tZero.Add(handler.redirectValidityWindow).Unix()) ||
				expTime < float64(tZero.Unix()) {
				t.Errorf("case %d: funny expiration time detected: %d", i, pwrReset.Claims["exp"])
			}

			if pwrReset.Claims["aud"] != testClientID {
				t.Errorf("case %d: wanted \"aud\"=%v got=%v", i, testClientID, pwrReset.Claims["aud"])
			}

			if pwrReset.Claims["iss"] != testIssuerURL.String() {
				t.Errorf("case %d: wanted \"iss\"=%v got=%v", i, testIssuerURL, pwrReset.Claims["iss"])
			}

			if pwrReset.UserID() != tt.userID {
				t.Errorf("case %d: wanted UserID=%v got=%v", i, tt.userID, pwrReset.UserID())
			}

			if bytes.Compare(pwrReset.Password(), user.Password("password")) != 0 {
				t.Errorf("case %d: wanted Password=%v got=%v", i, user.Password("password"), pwrReset.Password())
			}

			if *pwrReset.Callback() != testRedirectURL {
				t.Errorf("case %d: wanted callback=%v got=%v", i, testRedirectURL, pwrReset.Callback())
			}
		}
	}
}
