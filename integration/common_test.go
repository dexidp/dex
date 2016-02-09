package integration

import (
	"fmt"
	"io/ioutil"
	"net/http"
	"net/http/httptest"
	"net/url"

	"github.com/coreos/go-oidc/key"
	"github.com/jonboulle/clockwork"

	"github.com/coreos/dex/connector"
	"github.com/coreos/dex/db"
	"github.com/coreos/dex/user"
	"github.com/coreos/dex/user/manager"
)

var (
	clock = clockwork.NewFakeClock()

	testIssuerURL        = url.URL{Scheme: "https", Host: "auth.example.com"}
	testClientID         = "XXX"
	testClientSecret     = "yyy"
	testRedirectURL      = url.URL{Scheme: "https", Host: "client.example.com", Path: "/redirect"}
	testResetPasswordURL = url.URL{Scheme: "https", Host: "auth.example.com", Path: "/resetPassword"}
	testPrivKey, _       = key.GeneratePrivateKey()
)

type tokenHandlerTransport struct {
	Handler http.Handler
	Token   string
}

func (t *tokenHandlerTransport) RoundTrip(r *http.Request) (*http.Response, error) {
	r.Header.Set("Authorization", fmt.Sprintf("Bearer %s", t.Token))
	w := httptest.NewRecorder()
	t.Handler.ServeHTTP(w, r)
	resp := http.Response{
		StatusCode: w.Code,
		Header:     w.Header(),
		Body:       ioutil.NopCloser(w.Body),
	}
	return &resp, nil
}

func makeUserObjects(users []user.UserWithRemoteIdentities, passwords []user.PasswordInfo) (user.UserRepo, user.PasswordInfoRepo, *manager.UserManager) {
	dbMap := db.NewMemDB()
	ur := func() user.UserRepo {
		repo, err := db.NewUserRepoFromUsers(dbMap, users)
		if err != nil {
			panic("Failed to create user repo: " + err.Error())
		}
		return repo
	}()
	pwr := user.NewPasswordInfoRepoFromPasswordInfos(passwords)

	ccr := connector.NewConnectorConfigRepoFromConfigs(
		[]connector.ConnectorConfig{&connector.LocalConnectorConfig{ID: "local"}},
	)
	um := manager.NewUserManager(ur, pwr, ccr, db.TransactionFactory(dbMap), manager.ManagerOptions{})
	um.Clock = clock
	return ur, pwr, um
}
