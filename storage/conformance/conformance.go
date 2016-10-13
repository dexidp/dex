// +build go1.7

// Package conformance provides conformance tests for storage implementations.
package conformance

import (
	"reflect"
	"testing"
	"time"

	"golang.org/x/crypto/bcrypt"

	"github.com/coreos/dex/storage"

	"github.com/kylelemons/godebug/pretty"
)

// ensure that values being tested on never expire.
var neverExpire = time.Now().UTC().Add(time.Hour * 24 * 365 * 100)

// RunTests runs a set of conformance tests against a storage. newStorage should
// return an initialized but empty storage. The storage will be closed at the
// end of each test run.
func RunTests(t *testing.T, newStorage func() storage.Storage) {
	tests := []struct {
		name string
		run  func(t *testing.T, s storage.Storage)
	}{
		{"AuthCodeCRUD", testAuthCodeCRUD},
		{"AuthRequestCRUD", testAuthRequestCRUD},
		{"ClientCRUD", testClientCRUD},
		{"RefreshTokenCRUD", testRefreshTokenCRUD},
		{"PasswordCRUD", testPasswordCRUD},
		{"GarbageCollection", testGC},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			s := newStorage()
			test.run(t, s)
			s.Close()
		})
	}
}

func mustBeErrNotFound(t *testing.T, kind string, err error) {
	switch {
	case err == nil:
		t.Errorf("deleting non-existant %s should return an error", kind)
	case err != storage.ErrNotFound:
		t.Errorf("deleting %s expected storage.ErrNotFound, got %v", kind, err)
	}
}

func testAuthRequestCRUD(t *testing.T, s storage.Storage) {
	a := storage.AuthRequest{
		ID:                  storage.NewID(),
		ClientID:            "foobar",
		ResponseTypes:       []string{"code"},
		Scopes:              []string{"openid", "email"},
		RedirectURI:         "https://localhost:80/callback",
		Nonce:               "foo",
		State:               "bar",
		ForceApprovalPrompt: true,
		LoggedIn:            true,
		Expiry:              neverExpire,
		ConnectorID:         "ldap",
		ConnectorData:       []byte(`{"some":"data"}`),
		Claims: storage.Claims{
			UserID:        "1",
			Username:      "jane",
			Email:         "jane.doe@example.com",
			EmailVerified: true,
			Groups:        []string{"a", "b"},
		},
	}

	identity := storage.Claims{Email: "foobar"}

	if err := s.CreateAuthRequest(a); err != nil {
		t.Fatalf("failed creating auth request: %v", err)
	}
	if err := s.UpdateAuthRequest(a.ID, func(old storage.AuthRequest) (storage.AuthRequest, error) {
		old.Claims = identity
		old.ConnectorID = "connID"
		return old, nil
	}); err != nil {
		t.Fatalf("failed to update auth request: %v", err)
	}

	got, err := s.GetAuthRequest(a.ID)
	if err != nil {
		t.Fatalf("failed to get auth req: %v", err)
	}
	if !reflect.DeepEqual(got.Claims, identity) {
		t.Fatalf("update failed, wanted identity=%#v got %#v", identity, got.Claims)
	}
}

func testAuthCodeCRUD(t *testing.T, s storage.Storage) {
	a := storage.AuthCode{
		ID:            storage.NewID(),
		ClientID:      "foobar",
		RedirectURI:   "https://localhost:80/callback",
		Nonce:         "foobar",
		Scopes:        []string{"openid", "email"},
		Expiry:        neverExpire,
		ConnectorID:   "ldap",
		ConnectorData: []byte(`{"some":"data"}`),
		Claims: storage.Claims{
			UserID:        "1",
			Username:      "jane",
			Email:         "jane.doe@example.com",
			EmailVerified: true,
			Groups:        []string{"a", "b"},
		},
	}

	if err := s.CreateAuthCode(a); err != nil {
		t.Fatalf("failed creating auth code: %v", err)
	}

	got, err := s.GetAuthCode(a.ID)
	if err != nil {
		t.Fatalf("failed to get auth req: %v", err)
	}
	if a.Expiry.Unix() != got.Expiry.Unix() {
		t.Errorf("auth code expiry did not match want=%s vs got=%s", a.Expiry, got.Expiry)
	}
	got.Expiry = a.Expiry // time fields do not compare well
	if diff := pretty.Compare(a, got); diff != "" {
		t.Errorf("auth code retrieved from storage did not match: %s", diff)
	}

	if err := s.DeleteAuthCode(a.ID); err != nil {
		t.Fatalf("delete auth code: %v", err)
	}

	_, err = s.GetAuthCode(a.ID)
	mustBeErrNotFound(t, "auth code", err)
}

func testClientCRUD(t *testing.T, s storage.Storage) {
	id := storage.NewID()
	c := storage.Client{
		ID:           id,
		Secret:       "foobar",
		RedirectURIs: []string{"foo://bar.com/", "https://auth.example.com"},
		Name:         "dex client",
		LogoURL:      "https://goo.gl/JIyzIC",
	}
	err := s.DeleteClient(id)
	mustBeErrNotFound(t, "client", err)

	if err := s.CreateClient(c); err != nil {
		t.Fatalf("create client: %v", err)
	}

	getAndCompare := func(id string, want storage.Client) {
		gc, err := s.GetClient(id)
		if err != nil {
			t.Errorf("get client: %v", err)
			return
		}
		if diff := pretty.Compare(want, gc); diff != "" {
			t.Errorf("client retrieved from storage did not match: %s", diff)
		}
	}

	getAndCompare(id, c)

	newSecret := "barfoo"
	err = s.UpdateClient(id, func(old storage.Client) (storage.Client, error) {
		old.Secret = newSecret
		return old, nil
	})
	if err != nil {
		t.Errorf("update client: %v", err)
	}
	c.Secret = newSecret
	getAndCompare(id, c)

	if err := s.DeleteClient(id); err != nil {
		t.Fatalf("delete client: %v", err)
	}

	_, err = s.GetClient(id)
	mustBeErrNotFound(t, "client", err)
}

func testRefreshTokenCRUD(t *testing.T, s storage.Storage) {
	id := storage.NewID()
	refresh := storage.RefreshToken{
		RefreshToken: id,
		ClientID:     "client_id",
		ConnectorID:  "client_secret",
		Scopes:       []string{"openid", "email", "profile"},
		Claims: storage.Claims{
			UserID:        "1",
			Username:      "jane",
			Email:         "jane.doe@example.com",
			EmailVerified: true,
			Groups:        []string{"a", "b"},
		},
	}
	if err := s.CreateRefresh(refresh); err != nil {
		t.Fatalf("create refresh token: %v", err)
	}

	getAndCompare := func(id string, want storage.RefreshToken) {
		gr, err := s.GetRefresh(id)
		if err != nil {
			t.Errorf("get refresh: %v", err)
			return
		}
		if diff := pretty.Compare(want, gr); diff != "" {
			t.Errorf("refresh token retrieved from storage did not match: %s", diff)
		}
	}

	getAndCompare(id, refresh)

	if err := s.DeleteRefresh(id); err != nil {
		t.Fatalf("failed to delete refresh request: %v", err)
	}

	if _, err := s.GetRefresh(id); err != storage.ErrNotFound {
		t.Errorf("after deleting refresh expected storage.ErrNotFound, got %v", err)
	}
}

func testPasswordCRUD(t *testing.T, s storage.Storage) {
	// Use bcrypt.MinCost to keep the tests short.
	passwordHash, err := bcrypt.GenerateFromPassword([]byte("secret"), bcrypt.MinCost)
	if err != nil {
		t.Fatal(err)
	}

	password := storage.Password{
		Email:    "jane@example.com",
		Hash:     passwordHash,
		Username: "jane",
		UserID:   "foobar",
	}
	if err := s.CreatePassword(password); err != nil {
		t.Fatalf("create password token: %v", err)
	}

	getAndCompare := func(id string, want storage.Password) {
		gr, err := s.GetPassword(id)
		if err != nil {
			t.Errorf("get password %q: %v", id, err)
			return
		}
		if diff := pretty.Compare(want, gr); diff != "" {
			t.Errorf("password retrieved from storage did not match: %s", diff)
		}
	}

	getAndCompare("jane@example.com", password)
	getAndCompare("JANE@example.com", password) // Emails should be case insensitive

	if err := s.UpdatePassword(password.Email, func(old storage.Password) (storage.Password, error) {
		old.Username = "jane doe"
		return old, nil
	}); err != nil {
		t.Fatalf("failed to update auth request: %v", err)
	}

	password.Username = "jane doe"
	getAndCompare("jane@example.com", password)

	if err := s.DeletePassword(password.Email); err != nil {
		t.Fatalf("failed to delete password: %v", err)
	}

	if _, err := s.GetPassword(password.Email); err != storage.ErrNotFound {
		t.Errorf("after deleting password expected storage.ErrNotFound, got %v", err)
	}
}

func testGC(t *testing.T, s storage.Storage) {
	n := time.Now()
	c := storage.AuthCode{
		ID:            storage.NewID(),
		ClientID:      "foobar",
		RedirectURI:   "https://localhost:80/callback",
		Nonce:         "foobar",
		Scopes:        []string{"openid", "email"},
		Expiry:        n.Add(time.Second),
		ConnectorID:   "ldap",
		ConnectorData: []byte(`{"some":"data"}`),
		Claims: storage.Claims{
			UserID:        "1",
			Username:      "jane",
			Email:         "jane.doe@example.com",
			EmailVerified: true,
			Groups:        []string{"a", "b"},
		},
	}

	if err := s.CreateAuthCode(c); err != nil {
		t.Fatalf("failed creating auth code: %v", err)
	}

	if _, err := s.GarbageCollect(n); err != nil {
		t.Errorf("garbage collection failed: %v", err)
	}
	if _, err := s.GetAuthCode(c.ID); err != nil {
		t.Errorf("expected to be able to get auth code after GC: %v", err)
	}

	if r, err := s.GarbageCollect(n.Add(time.Minute)); err != nil {
		t.Errorf("garbage collection failed: %v", err)
	} else if r.AuthCodes != 1 {
		t.Errorf("expected to garbage collect 1 objects, got %d", r.AuthCodes)
	}

	if _, err := s.GetAuthCode(c.ID); err == nil {
		t.Errorf("expected auth code to be GC'd")
	} else if err != storage.ErrNotFound {
		t.Errorf("expected storage.ErrNotFound, got %v", err)
	}

	a := storage.AuthRequest{
		ID:                  storage.NewID(),
		ClientID:            "foobar",
		ResponseTypes:       []string{"code"},
		Scopes:              []string{"openid", "email"},
		RedirectURI:         "https://localhost:80/callback",
		Nonce:               "foo",
		State:               "bar",
		ForceApprovalPrompt: true,
		LoggedIn:            true,
		Expiry:              n,
		ConnectorID:         "ldap",
		ConnectorData:       []byte(`{"some":"data"}`),
		Claims: storage.Claims{
			UserID:        "1",
			Username:      "jane",
			Email:         "jane.doe@example.com",
			EmailVerified: true,
			Groups:        []string{"a", "b"},
		},
	}

	if err := s.CreateAuthRequest(a); err != nil {
		t.Fatalf("failed creating auth request: %v", err)
	}

	if _, err := s.GarbageCollect(n); err != nil {
		t.Errorf("garbage collection failed: %v", err)
	}
	if _, err := s.GetAuthRequest(a.ID); err != nil {
		t.Errorf("expected to be able to get auth code after GC: %v", err)
	}

	if r, err := s.GarbageCollect(n.Add(time.Minute)); err != nil {
		t.Errorf("garbage collection failed: %v", err)
	} else if r.AuthRequests != 1 {
		t.Errorf("expected to garbage collect 1 objects, got %d", r.AuthRequests)
	}

	if _, err := s.GetAuthRequest(a.ID); err == nil {
		t.Errorf("expected auth code to be GC'd")
	} else if err != storage.ErrNotFound {
		t.Errorf("expected storage.ErrNotFound, got %v", err)
	}
}
