package integration

import (
	"encoding/base64"
	"fmt"
	"io/ioutil"
	"net/http"
	"net/http/httptest"
	"net/url"

	"github.com/coreos/go-oidc/key"
	"github.com/go-gorp/gorp"
	"github.com/jonboulle/clockwork"

	"github.com/coreos/dex/client"
	clientmanager "github.com/coreos/dex/client/manager"
	"github.com/coreos/dex/connector"
	"github.com/coreos/dex/db"
	"github.com/coreos/dex/user"
	"github.com/coreos/dex/user/manager"
)

var (
	clock = clockwork.NewFakeClock()

	testIssuerURL        = url.URL{Scheme: "https", Host: "auth.example.com"}
	testClientID         = "client.example.com"
	testClientSecret     = base64.URLEncoding.EncodeToString([]byte("secret"))
	testRedirectURL      = url.URL{Scheme: "https", Host: "client.example.com", Path: "/redirect"}
	testBadRedirectURL   = url.URL{Scheme: "https", Host: "bad.example.com", Path: "/redirect"}
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

// TODO(ericchiang): Replace DbMap with storage interface. See #278

func makeUserObjects(
	users []user.UserWithRemoteIdentities,
	passwords []user.PasswordInfo,
	organizations []user.Organization) (*gorp.DbMap, user.UserRepo, user.PasswordInfoRepo, user.OrganizationRepo, *manager.UserManager) {
	dbMap := db.NewMemDB()
	ur := func() user.UserRepo {
		repo, err := db.NewUserRepoFromUsers(dbMap, users)
		if err != nil {
			panic("Failed to create user repo: " + err.Error())
		}
		return repo
	}()
	pwr := func() user.PasswordInfoRepo {
		repo, err := db.NewPasswordInfoRepoFromPasswordInfos(dbMap, passwords)
		if err != nil {
			panic("Failed to create password info repo: " + err.Error())
		}
		return repo
	}()

	orgr := func() user.OrganizationRepo {
		repo, err := db.NewOrganizationRepoFromOrganizations(dbMap, organizations)
		if err != nil {
			panic("Failed to create password info repo: " + err.Error())
		}
		return repo
	}()

	ccr := func() connector.ConnectorConfigRepo {
		repo := db.NewConnectorConfigRepo(dbMap)
		c := []connector.ConnectorConfig{&connector.LocalConnectorConfig{ID: "local"}}
		if err := repo.Set(c); err != nil {
			panic(err)
		}
		return repo
	}()

	um := manager.NewUserManager(ur, pwr, orgr, ccr, db.TransactionFactory(dbMap), manager.ManagerOptions{})
	um.Clock = clock
	return dbMap, ur, pwr, orgr, um
}

func makeClientRepoAndManager(dbMap *gorp.DbMap, clients []client.LoadableClient) (client.ClientRepo, *clientmanager.ClientManager, error) {
	clientIDGenerator := func(hostport string) (string, error) {
		return hostport, nil
	}
	secGen := func() ([]byte, error) {
		return []byte("secret"), nil
	}
	clientRepo, err := db.NewClientRepoFromClients(dbMap, clients)
	if err != nil {
		return nil, nil, err
	}
	clientManager := clientmanager.NewClientManager(clientRepo, db.TransactionFactory(dbMap), clientmanager.ManagerOptions{ClientIDGenerator: clientIDGenerator, SecretGenerator: secGen})
	return clientRepo, clientManager, nil

}
