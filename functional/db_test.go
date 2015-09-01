package functional

import (
	"encoding/base64"
	"fmt"
	"net/url"
	"os"
	"testing"
	"time"

	"github.com/coreos/go-oidc/key"
	"github.com/coreos/go-oidc/oidc"
	"github.com/go-gorp/gorp"
	"github.com/kylelemons/godebug/pretty"

	"github.com/coreos/dex/client"
	"github.com/coreos/dex/db"
	"github.com/coreos/dex/refresh"
	"github.com/coreos/dex/session"
)

var (
	dsn string
)

func init() {
	dsn = os.Getenv("DEX_TEST_DSN")
	if dsn == "" {
		fmt.Println("Unable to proceed with empty env var DEX_TEST_DSN")
		os.Exit(1)
	}
}

func connect(t *testing.T) *gorp.DbMap {
	c, err := db.NewConnection(db.Config{DSN: dsn})
	if err != nil {
		t.Fatalf("Unable to connect to database: %v", err)
	}

	if err = c.DropTablesIfExists(); err != nil {
		t.Fatalf("Unable to drop database tables: %v", err)
	}

	if err = db.DropMigrationsTable(c); err != nil {
		panic(fmt.Sprintf("Unable to drop migration table: %v", err))
	}

	db.MigrateToLatest(c)

	return c
}

func TestDBSessionKeyRepoPushPop(t *testing.T) {
	r := db.NewSessionKeyRepo(connect(t))

	key := "123"
	sessionID := "456"

	r.Push(session.SessionKey{Key: key, SessionID: sessionID}, time.Second)

	got, err := r.Pop(key)
	if err != nil {
		t.Fatalf("Expected nil error: %v", err)
	}
	if got != sessionID {
		t.Fatalf("Incorrect sessionID: want=%s got=%s", sessionID, got)
	}

	// attempting to Pop a second time must fail
	if _, err := r.Pop(key); err == nil {
		t.Fatalf("Second call to Pop succeeded, expected non-nil error")
	}
}

func TestDBSessionRepoCreateUpdate(t *testing.T) {
	r := db.NewSessionRepo(connect(t))

	// postgres stores its time type with a lower precision
	// than we generate here. Stripping off nanoseconds gives
	// us a predictable value to use in comparisions.
	now := time.Now().Round(time.Second).UTC()

	ses := session.Session{
		ID:          "AAA",
		State:       session.SessionStateIdentified,
		CreatedAt:   now,
		ExpiresAt:   now.Add(time.Minute),
		ClientID:    "ZZZ",
		ClientState: "foo",
		RedirectURL: url.URL{
			Scheme: "http",
			Host:   "example.com",
			Path:   "/callback",
		},
		Identity: oidc.Identity{
			ID:        "YYY",
			Name:      "Elroy",
			Email:     "elroy@example.com",
			ExpiresAt: now.Add(time.Minute),
		},
	}

	if err := r.Create(ses); err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}

	got, err := r.Get(ses.ID)
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}

	if diff := pretty.Compare(ses, got); diff != "" {
		t.Fatalf("Retrieved incorrect Session: Compare(want,got): %v", diff)
	}
}

func TestDBPrivateKeySetRepoSetGet(t *testing.T) {
	s1 := []byte("xxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxx")
	s2 := []byte("oooooooooooooooooooooooooooooooo")
	s3 := []byte("wwwwwwwwwwwwwwwwwwwwwwwwwwwwwwww")

	keys := []*key.PrivateKey{}
	for i := 0; i < 2; i++ {
		k, err := key.GeneratePrivateKey()
		if err != nil {
			t.Fatalf("Unable to generate RSA key: %v", err)
		}
		keys = append(keys, k)
	}

	ks := key.NewPrivateKeySet(
		[]*key.PrivateKey{keys[0], keys[1]}, time.Now().Add(time.Minute))

	tests := []struct {
		setSecrets [][]byte
		getSecrets [][]byte
		wantErr    bool
	}{
		{
			// same secrets used to encrypt, decrypt
			setSecrets: [][]byte{s1, s2},
			getSecrets: [][]byte{s1, s2},
		},
		{
			// setSecrets got rotated, but getSecrets didn't yet.
			setSecrets: [][]byte{s2, s3},
			getSecrets: [][]byte{s1, s2},
		},
		{
			// getSecrets doesn't have s3
			setSecrets: [][]byte{s3},
			getSecrets: [][]byte{s1, s2},
			wantErr:    true,
		},
	}

	for i, tt := range tests {
		setRepo, err := db.NewPrivateKeySetRepo(connect(t), tt.setSecrets...)
		if err != nil {
			t.Fatalf(err.Error())
		}

		getRepo, err := db.NewPrivateKeySetRepo(connect(t), tt.getSecrets...)
		if err != nil {
			t.Fatalf(err.Error())
		}

		if err := setRepo.Set(ks); err != nil {
			t.Fatalf("case %d: Unexpected error: %v", i, err)
		}

		got, err := getRepo.Get()
		if tt.wantErr {
			if err == nil {
				t.Errorf("case %d: want err, got nil", i)
			}
			continue
		}
		if err != nil {
			t.Fatalf("case %d: Unexpected error: %v", i, err)
		}

		if diff := pretty.Compare(ks, got); diff != "" {
			t.Fatalf("case %d:Retrieved incorrect KeySet: Compare(want,got): %v", i, diff)
		}

	}
}

func TestDBClientIdentityRepoMetadata(t *testing.T) {
	r := db.NewClientIdentityRepo(connect(t))

	cm := oidc.ClientMetadata{
		RedirectURLs: []url.URL{
			url.URL{Scheme: "http", Host: "127.0.0.1:5556", Path: "/cb"},
			url.URL{Scheme: "https", Host: "example.com", Path: "/callback"},
		},
	}

	_, err := r.New("foo", cm)
	if err != nil {
		t.Fatalf(err.Error())
	}

	got, err := r.Metadata("foo")
	if err != nil {
		t.Fatalf(err.Error())
	}

	if diff := pretty.Compare(cm, *got); diff != "" {
		t.Fatalf("Retrieved incorrect ClientMetadata: Compare(want,got): %v", diff)
	}
}

func TestDBClientIdentityRepoMetadataNoExist(t *testing.T) {
	r := db.NewClientIdentityRepo(connect(t))

	got, err := r.Metadata("noexist")
	if err != client.ErrorNotFound {
		t.Errorf("want==%q, got==%q", client.ErrorNotFound, err)
	}
	if got != nil {
		t.Fatalf("Retrieved incorrect ClientMetadata: want=nil got=%#v", got)
	}
}

func TestDBClientIdentityRepoNewDuplicate(t *testing.T) {
	r := db.NewClientIdentityRepo(connect(t))

	meta1 := oidc.ClientMetadata{
		RedirectURLs: []url.URL{
			url.URL{Scheme: "http", Host: "foo.example.com"},
		},
	}

	if _, err := r.New("foo", meta1); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	meta2 := oidc.ClientMetadata{
		RedirectURLs: []url.URL{
			url.URL{Scheme: "http", Host: "bar.example.com"},
		},
	}

	if _, err := r.New("foo", meta2); err == nil {
		t.Fatalf("expected non-nil error")
	}
}

func TestDBClientIdentityRepoAuthenticate(t *testing.T) {
	r := db.NewClientIdentityRepo(connect(t))

	cm := oidc.ClientMetadata{
		RedirectURLs: []url.URL{
			url.URL{Scheme: "http", Host: "127.0.0.1:5556", Path: "/cb"},
		},
	}

	cc, err := r.New("baz", cm)
	if err != nil {
		t.Fatalf(err.Error())
	}

	if cc.ID != "baz" {
		t.Fatalf("Returned ClientCredentials has incorrect ID: want=baz got=%s", cc.ID)
	}

	ok, err := r.Authenticate(*cc)
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	} else if !ok {
		t.Fatalf("Authentication failed for good creds")
	}

	creds := []oidc.ClientCredentials{
		// completely made up
		oidc.ClientCredentials{ID: "foo", Secret: "bar"},

		// good client ID, bad secret
		oidc.ClientCredentials{ID: cc.ID, Secret: "bar"},

		// bad client ID, good secret
		oidc.ClientCredentials{ID: "foo", Secret: cc.Secret},

		// good client ID, secret with some fluff on the end
		oidc.ClientCredentials{ID: cc.ID, Secret: fmt.Sprintf("%sfluff", cc.Secret)},
	}
	for i, c := range creds {
		ok, err := r.Authenticate(c)
		if err != nil {
			t.Errorf("case %d: unexpected error: %v", i, err)
		} else if ok {
			t.Errorf("case %d: authentication succeeded for bad creds", i)
		}
	}
}

func TestDBClientIdentityAll(t *testing.T) {
	r := db.NewClientIdentityRepo(connect(t))

	cm := oidc.ClientMetadata{
		RedirectURLs: []url.URL{
			url.URL{Scheme: "http", Host: "127.0.0.1:5556", Path: "/cb"},
		},
	}

	_, err := r.New("foo", cm)
	if err != nil {
		t.Fatalf(err.Error())
	}

	got, err := r.All()
	if err != nil {
		t.Fatalf(err.Error())
	}
	count := len(got)
	if count != 1 {
		t.Fatalf("Retrieved incorrect number of ClientIdentities: want=1 got=%d", count)
	}

	if diff := pretty.Compare(cm, got[0].Metadata); diff != "" {
		t.Fatalf("Retrieved incorrect ClientMetadata: Compare(want,got): %v", diff)
	}

	cm = oidc.ClientMetadata{
		RedirectURLs: []url.URL{
			url.URL{Scheme: "http", Host: "foo.com", Path: "/cb"},
		},
	}
	_, err = r.New("bar", cm)
	if err != nil {
		t.Fatalf(err.Error())
	}

	got, err = r.All()
	if err != nil {
		t.Fatalf(err.Error())
	}
	count = len(got)
	if count != 2 {
		t.Fatalf("Retrieved incorrect number of ClientIdentities: want=2 got=%d", count)
	}
}

// buildRefreshToken combines the token ID and token payload to create a new token.
// used in the tests to created a refresh token.
func buildRefreshToken(tokenID int64, tokenPayload []byte) string {
	return fmt.Sprintf("%d%s%s", tokenID, refresh.TokenDelimer, base64.URLEncoding.EncodeToString(tokenPayload))
}

func TestDBRefreshRepoCreate(t *testing.T) {
	r := db.NewRefreshTokenRepo(connect(t))

	tests := []struct {
		userID   string
		clientID string
		err      error
	}{
		{
			"",
			"client-foo",
			refresh.ErrorInvalidUserID,
		},
		{
			"user-foo",
			"",
			refresh.ErrorInvalidClientID,
		},
		{
			"user-foo",
			"client-foo",
			nil,
		},
	}

	for i, tt := range tests {
		_, err := r.Create(tt.userID, tt.clientID)
		if err != tt.err {
			t.Errorf("Case #%d: expected: %v, got: %v", i, tt.err, err)
		}
	}
}

func TestDBRefreshRepoVerify(t *testing.T) {
	r := db.NewRefreshTokenRepo(connect(t))

	token, err := r.Create("user-foo", "client-foo")
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}

	badTokenPayload, err := refresh.DefaultRefreshTokenGenerator()
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}
	tokenWithBadID := "404" + token[1:]
	tokenWithBadPayload := buildRefreshToken(1, badTokenPayload)

	tests := []struct {
		token    string
		creds    oidc.ClientCredentials
		err      error
		expected string
	}{
		{
			"invalid-token-format",
			oidc.ClientCredentials{ID: "client-foo", Secret: "secret-foo"},
			refresh.ErrorInvalidToken,
			"",
		},
		{
			"b/invalid-base64-encoded-format",
			oidc.ClientCredentials{ID: "client-foo", Secret: "secret-foo"},
			refresh.ErrorInvalidToken,
			"",
		},
		{
			"1/invalid-base64-encoded-format",
			oidc.ClientCredentials{ID: "client-foo", Secret: "secret-foo"},
			refresh.ErrorInvalidToken,
			"",
		},
		{
			token + "corrupted-token-payload",
			oidc.ClientCredentials{ID: "client-foo", Secret: "secret-foo"},
			refresh.ErrorInvalidToken,
			"",
		},
		{
			// The token's ID content is invalid.
			tokenWithBadID,
			oidc.ClientCredentials{ID: "client-foo", Secret: "secret-foo"},
			refresh.ErrorInvalidToken,
			"",
		},
		{
			// The token's payload content is invalid.
			tokenWithBadPayload,
			oidc.ClientCredentials{ID: "client-foo", Secret: "secret-foo"},
			refresh.ErrorInvalidToken,
			"",
		},
		{
			token,
			oidc.ClientCredentials{ID: "invalid-client", Secret: "secret-foo"},
			refresh.ErrorInvalidClientID,
			"",
		},
		{
			token,
			oidc.ClientCredentials{ID: "client-foo", Secret: "secret-foo"},
			nil,
			"user-foo",
		},
	}

	for i, tt := range tests {
		result, err := r.Verify(tt.creds.ID, tt.token)
		if err != tt.err {
			t.Errorf("Case #%d: expected: %v, got: %v", i, tt.err, err)
		}
		if result != tt.expected {
			t.Errorf("Case #%d: expected: %v, got: %v", i, tt.expected, result)
		}
	}
}

func TestDBRefreshRepoRevoke(t *testing.T) {
	r := db.NewRefreshTokenRepo(connect(t))

	token, err := r.Create("user-foo", "client-foo")
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}

	badTokenPayload, err := refresh.DefaultRefreshTokenGenerator()
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}
	tokenWithBadID := "404" + token[1:]
	tokenWithBadPayload := buildRefreshToken(1, badTokenPayload)

	tests := []struct {
		token  string
		userID string
		err    error
	}{
		{
			"invalid-token-format",
			"user-foo",
			refresh.ErrorInvalidToken,
		},
		{
			"1/invalid-base64-encoded-format",
			"user-foo",
			refresh.ErrorInvalidToken,
		},
		{
			token + "corrupted-token-payload",
			"user-foo",
			refresh.ErrorInvalidToken,
		},
		{
			// The token's ID is invalid.
			tokenWithBadID,
			"user-foo",
			refresh.ErrorInvalidToken,
		},
		{
			// The token's payload is invalid.
			tokenWithBadPayload,
			"user-foo",
			refresh.ErrorInvalidToken,
		},
		{
			token,
			"invalid-user",
			refresh.ErrorInvalidUserID,
		},
		{
			token,
			"user-foo",
			nil,
		},
	}

	for i, tt := range tests {
		if err := r.Revoke(tt.userID, tt.token); err != tt.err {
			t.Errorf("Case #%d: expected: %v, got: %v", i, tt.err, err)
		}
	}
}
