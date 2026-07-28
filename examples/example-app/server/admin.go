package server

import (
	"context"
	"crypto/tls"
	"crypto/x509"
	"fmt"
	"net/http"
	"net/url"
	"os"
	"strings"
	"time"

	"golang.org/x/crypto/bcrypt"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials"
	"google.golang.org/grpc/credentials/insecure"

	api "github.com/dexidp/dex/api/v2"
)

// adminClient talks to dex's gRPC API — the interface that manages the
// provider itself rather than authenticating against it.
type adminClient struct {
	conn *grpc.ClientConn
	api  api.DexClient
}

// newAdminClient dials the gRPC API. Plaintext is allowed because that is how
// dex's own example config exposes it locally; anything reachable off the host
// should be given the certificates instead.
func newAdminClient(opts Options) (*adminClient, error) {
	creds := insecure.NewCredentials()

	if opts.GRPCCA != "" {
		pool := x509.NewCertPool()
		caCert, err := os.ReadFile(opts.GRPCCA)
		if err != nil {
			return nil, fmt.Errorf("read gRPC CA: %v", err)
		}
		if !pool.AppendCertsFromPEM(caCert) {
			return nil, fmt.Errorf("no certificates found in %q", opts.GRPCCA)
		}

		cfg := &tls.Config{RootCAs: pool}
		if opts.GRPCClientCert != "" || opts.GRPCClientKey != "" {
			cert, err := tls.LoadX509KeyPair(opts.GRPCClientCert, opts.GRPCClientKey)
			if err != nil {
				return nil, fmt.Errorf("load gRPC client key pair: %v", err)
			}
			cfg.Certificates = []tls.Certificate{cert}
		}
		creds = credentials.NewTLS(cfg)
	}

	conn, err := grpc.NewClient(opts.GRPCAddr, grpc.WithTransportCredentials(creds))
	if err != nil {
		return nil, fmt.Errorf("dial gRPC API at %q: %v", opts.GRPCAddr, err)
	}

	return &adminClient{conn: conn, api: api.NewDexClient(conn)}, nil
}

func (a *adminClient) close() {
	if a.conn != nil {
		a.conn.Close()
	}
}

// adminSections are the tabs of the API page. dex's API has more than twenty
// methods; putting every list and every form on one page is what made the last
// version unreadable.
// connectorTypes are the types dex's config parser knows. The list is short,
// fixed, and impossible to guess the spelling of, so the form offers it rather
// than asking you to type one.
var connectorTypes = []string{
	"atlassian-crowd", "authproxy", "bitbucket-cloud", "gitea", "github",
	"gitlab", "google", "keystone", "ldap", "linkedin", "microsoft",
	"mockCallback", "mockPassword", "oauth", "oidc", "openshift", "saml",
}

var adminSections = []AdminSection{
	{ID: "clients", Label: "Clients"},
	{ID: "passwords", Label: "Passwords"},
	{ID: "connectors", Label: "Connectors"},
	{ID: "identities", Label: "Users"},
	{ID: "sessions", Label: "Sessions"},
}

// handleAdmin renders one section of the API.
func (s *Server) handleAdmin(w http.ResponseWriter, r *http.Request) {
	section := r.URL.Query().Get("section")
	if section == "" {
		section = "clients"
	}

	data := AdminPageData{
		LogoURI:      dexLogoDataURI,
		AdminEnabled: true,
		Configured:   s.admin != nil,
		Section:      section,
		Notice:       r.URL.Query().Get("notice"),
		Error:        r.URL.Query().Get("error"),
		UserID:       r.URL.Query().Get("user_id"),
		ConnectorID:  r.URL.Query().Get("connector_id"),
		Mode:         r.URL.Query().Get("mode"),

		ConnectorTypes: connectorTypes,
		ConnectorGrantTypes: []string{
			grantAuthorizationCode, grantRefreshToken, grantDeviceCode,
			grantTokenExchange, grantClientCredentials, grantPassword,
		},
	}
	for _, sec := range adminSections {
		sec.Current = sec.ID == section
		data.Sections = append(data.Sections, sec)
	}

	// Without --grpc-addr the page explains itself rather than 404ing, which is
	// how you find out the feature exists at all.
	if s.admin == nil {
		s.renderer.RenderAdminPage(w, data)
		return
	}

	ctx, cancel := context.WithTimeout(r.Context(), 10*time.Second)
	defer cancel()

	fail := func(err error) {
		if data.Error == "" {
			data.Error = err.Error()
		}
	}

	if version, err := s.admin.api.GetVersion(ctx, &api.VersionReq{}); err == nil {
		data.Version = fmt.Sprintf("%s (API %d)", version.Server, version.Api)
	} else {
		fail(err)
	}
	if disco, err := s.admin.api.GetDiscovery(ctx, &api.DiscoveryReq{}); err == nil {
		data.Issuer = disco.Issuer
	}

	switch section {
	case "clients":
		if resp, err := s.admin.api.ListClients(ctx, &api.ListClientReq{}); err == nil {
			for _, c := range resp.Clients {
				data.Clients = append(data.Clients, AdminClient{
					ID:                c.Id,
					Name:              c.Name,
					RedirectURIs:      c.RedirectUris,
					TrustedPeers:      c.TrustedPeers,
					Public:            c.Public,
					LogoURL:           c.LogoUrl,
					AllowedConnectors: c.AllowedConnectors,
					SSOSharedWith:     c.SsoSharedWith,
				})
			}
		} else {
			fail(err)
		}

	case "passwords":
		if resp, err := s.admin.api.ListPasswords(ctx, &api.ListPasswordReq{}); err == nil {
			for _, p := range resp.Passwords {
				data.Passwords = append(data.Passwords, AdminPassword{
					Email:    p.Email,
					Username: p.Username,
					UserID:   p.UserId,
				})
			}
		} else {
			fail(err)
		}

	case "connectors":
		if resp, err := s.admin.api.ListConnectors(ctx, &api.ListConnectorReq{}); err == nil {
			for _, c := range resp.Connectors {
				data.Connectors = append(data.Connectors, AdminConnector{
					ID: c.Id, Type: c.Type, Name: c.Name, GrantTypes: c.GrantTypes,
				})
			}
		} else {
			fail(err)
		}

	case "identities":
		if resp, err := s.admin.api.ListUserIdentities(ctx, &api.ListUserIdentitiesReq{}); err == nil {
			for _, u := range resp.Identities {
				identity := AdminIdentity{
					UserID:        u.UserId,
					ConnectorID:   u.ConnectorId,
					Email:         u.Email,
					EmailVerified: u.EmailVerified,
					Username:      u.Username,
					Groups:        u.Groups,
				}
				for _, d := range u.MfaDevices {
					identity.MFADevices = append(identity.MFADevices, AdminMFADevice{AuthenticatorID: d.AuthenticatorId})
				}
				data.Identities = append(data.Identities, identity)
			}
		} else {
			fail(err)
		}

	case "sessions":
		// The two listings are keyed differently, which is a property of the API
		// rather than a choice here: sessions are stored under the ID the
		// connector gave the user, refresh tokens under the encoded sub claim
		// that ends up in tokens.
		if data.UserID != "" {
			if resp, err := s.admin.api.ListAuthSessions(ctx, &api.ListAuthSessionsReq{UserId: data.UserID}); err == nil {
				for _, sess := range resp.Sessions {
					data.Sessions = append(data.Sessions, AdminSession{
						UserID:      sess.UserId,
						ConnectorID: sess.ConnectorId,
						IPAddress:   sess.IpAddress,
						Created:     epochText(sess.CreatedAt),
						Expires:     epochText(sess.AbsoluteExpiry),
					})
				}
			} else {
				fail(err)
			}

		}

		// Refresh tokens are keyed by the sub claim, which is the user and the
		// connector encoded together.
		if data.UserID != "" && data.ConnectorID != "" {
			subject := idTokenSubject(data.UserID, data.ConnectorID)
			if resp, err := s.admin.api.ListRefresh(ctx, &api.ListRefreshReq{UserId: subject}); err == nil {
				for _, t := range resp.RefreshTokens {
					data.RefreshTokens = append(data.RefreshTokens, AdminRefreshToken{
						ID:       t.Id,
						ClientID: t.ClientId,
						Created:  epochText(t.CreatedAt),
						LastUsed: epochText(t.LastUsed),
					})
				}
			} else {
				fail(err)
			}
		}
	}

	// An edit starts from a row, so the form comes back filled in.
	if id := r.URL.Query().Get("edit"); id != "" && section == "clients" {
		if resp, err := s.admin.api.GetClient(ctx, &api.GetClientReq{Id: id}); err == nil && resp.Client != nil {
			c := resp.Client
			data.EditClient = &AdminClient{
				ID:                c.Id,
				Name:              c.Name,
				RedirectURIs:      c.RedirectUris,
				TrustedPeers:      c.TrustedPeers,
				Public:            c.Public,
				LogoURL:           c.LogoUrl,
				AllowedConnectors: c.AllowedConnectors,
				SSOSharedWith:     c.SsoSharedWith,
			}
		} else if err != nil {
			fail(err)
		}
	}
	if email := r.URL.Query().Get("edit"); email != "" && section == "passwords" {
		for _, p := range data.Passwords {
			if p.Email == email {
				entry := p
				data.EditPassword = &entry
				break
			}
		}
	}

	s.renderer.RenderAdminPage(w, data)
}

// epochText renders one of the API's Unix timestamps, which are zero when the
// field was never set.
func epochText(epoch int64) string {
	if epoch == 0 {
		return ""
	}
	return time.Unix(epoch, 0).UTC().Format("2 Jan 2006, 15:04 UTC")
}

// handleAdminCreateClient registers an OAuth2 client.
func (s *Server) handleAdminCreateClient(w http.ResponseWriter, r *http.Request) {
	if err := r.ParseForm(); err != nil {
		s.adminRedirect(w, r, "", err.Error())
		return
	}

	ctx, cancel := context.WithTimeout(r.Context(), 10*time.Second)
	defer cancel()

	req := &api.CreateClientReq{
		Client: &api.Client{
			Id:                r.FormValue("id"),
			Name:              r.FormValue("name"),
			Secret:            r.FormValue("secret"),
			LogoUrl:           r.FormValue("logo_url"),
			RedirectUris:      splitLines(r.FormValue("redirect_uris")),
			TrustedPeers:      splitLines(r.FormValue("trusted_peers")),
			AllowedConnectors: splitLines(r.FormValue("allowed_connectors")),
			SsoSharedWith:     splitLines(r.FormValue("sso_shared_with")),
			Public:            r.FormValue("public") != "",
		},
	}

	resp, err := s.admin.api.CreateClient(ctx, req)
	switch {
	case err != nil:
		s.adminRedirect(w, r, "", err.Error())
	case resp.AlreadyExists:
		s.adminRedirect(w, r, "", fmt.Sprintf("client %q already exists", req.Client.Id))
	default:
		s.adminRedirect(w, r, fmt.Sprintf("created client %q", req.Client.Id), "")
	}
}

// handleAdminDeleteClient removes an OAuth2 client.
func (s *Server) handleAdminDeleteClient(w http.ResponseWriter, r *http.Request) {
	if err := r.ParseForm(); err != nil {
		s.adminRedirect(w, r, "", err.Error())
		return
	}

	ctx, cancel := context.WithTimeout(r.Context(), 10*time.Second)
	defer cancel()

	id := r.FormValue("id")
	resp, err := s.admin.api.DeleteClient(ctx, &api.DeleteClientReq{Id: id})
	switch {
	case err != nil:
		s.adminRedirect(w, r, "", err.Error())
	case resp.NotFound:
		s.adminRedirect(w, r, "", fmt.Sprintf("client %q not found", id))
	default:
		s.adminRedirect(w, r, fmt.Sprintf("deleted client %q", id), "")
	}
}

// handleAdminCreatePassword adds a user to dex's local password database. The
// API takes a bcrypt hash rather than a password, so the hashing happens here.
func (s *Server) handleAdminCreatePassword(w http.ResponseWriter, r *http.Request) {
	if err := r.ParseForm(); err != nil {
		s.adminRedirect(w, r, "", err.Error())
		return
	}

	password := r.FormValue("password")
	if password == "" {
		s.adminRedirect(w, r, "", "password is required")
		return
	}

	hash, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		s.adminRedirect(w, r, "", err.Error())
		return
	}

	ctx, cancel := context.WithTimeout(r.Context(), 10*time.Second)
	defer cancel()

	email := r.FormValue("email")
	req := &api.CreatePasswordReq{
		Password: &api.Password{
			Email:    email,
			Username: r.FormValue("username"),
			UserId:   r.FormValue("user_id"),
			Hash:     hash,
		},
	}

	resp, err := s.admin.api.CreatePassword(ctx, req)
	switch {
	case err != nil:
		s.adminRedirect(w, r, "", err.Error())
	case resp.AlreadyExists:
		s.adminRedirect(w, r, "", fmt.Sprintf("password for %q already exists", email))
	default:
		s.adminRedirect(w, r, fmt.Sprintf("created password for %q", email), "")
	}
}

// handleAdminDeletePassword removes a user from the local password database.
func (s *Server) handleAdminDeletePassword(w http.ResponseWriter, r *http.Request) {
	if err := r.ParseForm(); err != nil {
		s.adminRedirect(w, r, "", err.Error())
		return
	}

	ctx, cancel := context.WithTimeout(r.Context(), 10*time.Second)
	defer cancel()

	email := r.FormValue("email")
	resp, err := s.admin.api.DeletePassword(ctx, &api.DeletePasswordReq{Email: email})
	switch {
	case err != nil:
		s.adminRedirect(w, r, "", err.Error())
	case resp.NotFound:
		s.adminRedirect(w, r, "", fmt.Sprintf("password for %q not found", email))
	default:
		s.adminRedirect(w, r, fmt.Sprintf("deleted password for %q", email), "")
	}
}

// handleAdminRevokeRefresh revokes a user's refresh token for one client. This
// is the other half of the session story: it is what makes a sign-in stop
// working before the token expires on its own.
func (s *Server) handleAdminRevokeRefresh(w http.ResponseWriter, r *http.Request) {
	if err := r.ParseForm(); err != nil {
		s.adminRedirect(w, r, "", err.Error())
		return
	}

	ctx, cancel := context.WithTimeout(r.Context(), 10*time.Second)
	defer cancel()

	userID, clientID := r.FormValue("user_id"), r.FormValue("client_id")
	resp, err := s.admin.api.RevokeRefresh(ctx, &api.RevokeRefreshReq{
		UserId:   userID,
		ClientId: clientID,
	})
	switch {
	case err != nil:
		s.adminRedirect(w, r, "", err.Error())
	case resp.NotFound:
		s.adminRedirect(w, r, "", fmt.Sprintf("no refresh token for user %q and client %q", userID, clientID))
	default:
		s.adminRedirect(w, r, fmt.Sprintf("revoked refresh token for user %q", userID), "")
	}
}

// adminRedirect returns to the admin page carrying the outcome, so a reload
// does not repeat the action.
func (s *Server) adminRedirect(w http.ResponseWriter, r *http.Request, notice, errMsg string) {
	q := url.Values{}
	if section := r.FormValue("section"); section != "" {
		q.Set("section", section)
	}
	if userID := r.FormValue("list_user_id"); userID != "" {
		q.Set("user_id", userID)
	}
	if connectorID := r.FormValue("list_connector_id"); connectorID != "" {
		q.Set("connector_id", connectorID)
	}
	switch {
	case errMsg != "":
		q.Set("error", errMsg)
	case notice != "":
		q.Set("notice", notice)
	}

	u := "/admin"
	if len(q) > 0 {
		u += "?" + q.Encode()
	}
	http.Redirect(w, r, u, http.StatusSeeOther)
}

// splitLines reads a textarea into a list, dropping blanks.
func splitLines(raw string) []string {
	var out []string
	for _, line := range strings.Split(raw, "\n") {
		if line = strings.TrimSpace(line); line != "" {
			out = append(out, line)
		}
	}
	return out
}

// handleAdminUpdateClient changes a client. Empty fields are left alone, since
// the API's update takes only what it should overwrite.
func (s *Server) handleAdminUpdateClient(w http.ResponseWriter, r *http.Request) {
	if err := r.ParseForm(); err != nil {
		s.adminRedirect(w, r, "", err.Error())
		return
	}

	ctx, cancel := context.WithTimeout(r.Context(), 10*time.Second)
	defer cancel()

	id := r.FormValue("id")
	resp, err := s.admin.api.UpdateClient(ctx, &api.UpdateClientReq{
		Id:                id,
		Name:              r.FormValue("name"),
		LogoUrl:           r.FormValue("logo_url"),
		RedirectUris:      splitLines(r.FormValue("redirect_uris")),
		TrustedPeers:      splitLines(r.FormValue("trusted_peers")),
		AllowedConnectors: splitLines(r.FormValue("allowed_connectors")),
		SsoSharedWith:     splitLines(r.FormValue("sso_shared_with")),
	})
	switch {
	case err != nil:
		s.adminRedirect(w, r, "", err.Error())
	case resp.NotFound:
		s.adminRedirect(w, r, "", fmt.Sprintf("client %q not found", id))
	default:
		s.adminRedirect(w, r, fmt.Sprintf("updated client %q", id), "")
	}
}

// handleAdminUpdatePassword changes a user's password or username.
func (s *Server) handleAdminUpdatePassword(w http.ResponseWriter, r *http.Request) {
	if err := r.ParseForm(); err != nil {
		s.adminRedirect(w, r, "", err.Error())
		return
	}

	req := &api.UpdatePasswordReq{
		Email:       r.FormValue("email"),
		NewUsername: r.FormValue("username"),
	}
	if password := r.FormValue("password"); password != "" {
		hash, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
		if err != nil {
			s.adminRedirect(w, r, "", err.Error())
			return
		}
		req.NewHash = hash
	}

	ctx, cancel := context.WithTimeout(r.Context(), 10*time.Second)
	defer cancel()

	resp, err := s.admin.api.UpdatePassword(ctx, req)
	switch {
	case err != nil:
		s.adminRedirect(w, r, "", err.Error())
	case resp.NotFound:
		s.adminRedirect(w, r, "", fmt.Sprintf("password for %q not found", req.Email))
	default:
		s.adminRedirect(w, r, fmt.Sprintf("updated password for %q", req.Email), "")
	}
}

// handleAdminVerifyPassword checks a password without issuing anything, which
// is how a service that owns its own login screen would use dex's user store.
func (s *Server) handleAdminVerifyPassword(w http.ResponseWriter, r *http.Request) {
	if err := r.ParseForm(); err != nil {
		s.adminRedirect(w, r, "", err.Error())
		return
	}

	ctx, cancel := context.WithTimeout(r.Context(), 10*time.Second)
	defer cancel()

	email := r.FormValue("email")
	resp, err := s.admin.api.VerifyPassword(ctx, &api.VerifyPasswordReq{
		Email:    email,
		Password: r.FormValue("password"),
	})
	switch {
	case err != nil:
		s.adminRedirect(w, r, "", err.Error())
	case resp.NotFound:
		s.adminRedirect(w, r, "", fmt.Sprintf("no password for %q", email))
	case !resp.Verified:
		s.adminRedirect(w, r, "", fmt.Sprintf("password for %q does not match", email))
	default:
		s.adminRedirect(w, r, fmt.Sprintf("password for %q verified", email), "")
	}
}

// handleAdminCreateConnector adds a connector at runtime.
func (s *Server) handleAdminCreateConnector(w http.ResponseWriter, r *http.Request) {
	if err := r.ParseForm(); err != nil {
		s.adminRedirect(w, r, "", err.Error())
		return
	}

	ctx, cancel := context.WithTimeout(r.Context(), 10*time.Second)
	defer cancel()

	id := r.FormValue("id")
	resp, err := s.admin.api.CreateConnector(ctx, &api.CreateConnectorReq{
		Connector: &api.Connector{
			Id:         id,
			Type:       r.FormValue("type"),
			Name:       r.FormValue("name"),
			Config:     []byte(r.FormValue("config")),
			GrantTypes: r.Form["grant_types"],
		},
	})
	switch {
	case err != nil:
		s.adminRedirect(w, r, "", err.Error())
	case resp.AlreadyExists:
		s.adminRedirect(w, r, "", fmt.Sprintf("connector %q already exists", id))
	default:
		s.adminRedirect(w, r, fmt.Sprintf("created connector %q", id), "")
	}
}

// handleAdminDeleteConnector removes a connector.
func (s *Server) handleAdminDeleteConnector(w http.ResponseWriter, r *http.Request) {
	if err := r.ParseForm(); err != nil {
		s.adminRedirect(w, r, "", err.Error())
		return
	}

	ctx, cancel := context.WithTimeout(r.Context(), 10*time.Second)
	defer cancel()

	id := r.FormValue("id")
	resp, err := s.admin.api.DeleteConnector(ctx, &api.DeleteConnectorReq{Id: id})
	switch {
	case err != nil:
		s.adminRedirect(w, r, "", err.Error())
	case resp.NotFound:
		s.adminRedirect(w, r, "", fmt.Sprintf("connector %q not found", id))
	default:
		s.adminRedirect(w, r, fmt.Sprintf("deleted connector %q", id), "")
	}
}

// handleAdminDeleteIdentity forgets what dex recorded about a user from one
// connector. The next sign-in records it again.
func (s *Server) handleAdminDeleteIdentity(w http.ResponseWriter, r *http.Request) {
	if err := r.ParseForm(); err != nil {
		s.adminRedirect(w, r, "", err.Error())
		return
	}

	ctx, cancel := context.WithTimeout(r.Context(), 10*time.Second)
	defer cancel()

	userID := r.FormValue("user_id")
	resp, err := s.admin.api.DeleteUserIdentity(ctx, &api.DeleteUserIdentityReq{
		UserId:      userID,
		ConnectorId: r.FormValue("connector_id"),
	})
	switch {
	case err != nil:
		s.adminRedirect(w, r, "", err.Error())
	case resp.NotFound:
		s.adminRedirect(w, r, "", fmt.Sprintf("no identity for user %q", userID))
	default:
		s.adminRedirect(w, r, fmt.Sprintf("deleted identity for user %q", userID), "")
	}
}

// handleAdminDeleteSession ends one of dex's sessions.
func (s *Server) handleAdminDeleteSession(w http.ResponseWriter, r *http.Request) {
	if err := r.ParseForm(); err != nil {
		s.adminRedirect(w, r, "", err.Error())
		return
	}

	ctx, cancel := context.WithTimeout(r.Context(), 10*time.Second)
	defer cancel()

	userID := r.FormValue("user_id")
	resp, err := s.admin.api.DeleteAuthSession(ctx, &api.DeleteAuthSessionReq{
		UserId:      userID,
		ConnectorId: r.FormValue("connector_id"),
	})
	switch {
	case err != nil:
		s.adminRedirect(w, r, "", err.Error())
	case resp.NotFound:
		s.adminRedirect(w, r, "", fmt.Sprintf("no session for user %q", userID))
	default:
		s.adminRedirect(w, r, fmt.Sprintf("deleted session for user %q", userID), "")
	}
}

// handleAdminTerminateSessions ends every session a user has.
func (s *Server) handleAdminTerminateSessions(w http.ResponseWriter, r *http.Request) {
	if err := r.ParseForm(); err != nil {
		s.adminRedirect(w, r, "", err.Error())
		return
	}

	ctx, cancel := context.WithTimeout(r.Context(), 10*time.Second)
	defer cancel()

	userID := r.FormValue("user_id")
	resp, err := s.admin.api.TerminateSessionsByUser(ctx, &api.TerminateSessionsByUserReq{UserId: userID})
	if err != nil {
		s.adminRedirect(w, r, "", err.Error())
		return
	}
	s.adminRedirect(w, r, fmt.Sprintf("terminated %d session(s) for user %q", resp.SessionsTerminated, userID), "")
}

// handleAdminResetMFA clears a user's second factors so they enrol again.
func (s *Server) handleAdminResetMFA(w http.ResponseWriter, r *http.Request) {
	if err := r.ParseForm(); err != nil {
		s.adminRedirect(w, r, "", err.Error())
		return
	}

	ctx, cancel := context.WithTimeout(r.Context(), 10*time.Second)
	defer cancel()

	userID := r.FormValue("user_id")
	resp, err := s.admin.api.ResetMFA(ctx, &api.ResetMFAReq{
		UserId:      userID,
		ConnectorId: r.FormValue("connector_id"),
	})
	switch {
	case err != nil:
		s.adminRedirect(w, r, "", err.Error())
	case resp.NotFound:
		s.adminRedirect(w, r, "", fmt.Sprintf("no MFA enrolment for user %q", userID))
	default:
		s.adminRedirect(w, r, fmt.Sprintf("reset MFA for user %q", userID), "")
	}
}

// handleAdminDeleteMFASecret removes one authenticator's secret, leaving the
// user's other factors alone.
func (s *Server) handleAdminDeleteMFASecret(w http.ResponseWriter, r *http.Request) {
	if err := r.ParseForm(); err != nil {
		s.adminRedirect(w, r, "", err.Error())
		return
	}

	ctx, cancel := context.WithTimeout(r.Context(), 10*time.Second)
	defer cancel()

	userID, authenticator := r.FormValue("user_id"), r.FormValue("authenticator_id")
	resp, err := s.admin.api.DeleteMFASecret(ctx, &api.DeleteMFASecretReq{
		UserId:          userID,
		ConnectorId:     r.FormValue("connector_id"),
		AuthenticatorId: authenticator,
	})
	switch {
	case err != nil:
		s.adminRedirect(w, r, "", err.Error())
	case resp.NotFound:
		s.adminRedirect(w, r, "", fmt.Sprintf("no %q secret for user %q", authenticator, userID))
	default:
		s.adminRedirect(w, r, fmt.Sprintf("deleted %q secret for user %q", authenticator, userID), "")
	}
}

// handleAdminTerminateByConnector ends every session that came through one
// connector — what you reach for when a connector is compromised or retired.
func (s *Server) handleAdminTerminateByConnector(w http.ResponseWriter, r *http.Request) {
	if err := r.ParseForm(); err != nil {
		s.adminRedirect(w, r, "", err.Error())
		return
	}

	ctx, cancel := context.WithTimeout(r.Context(), 10*time.Second)
	defer cancel()

	connectorID := r.FormValue("connector_id")
	resp, err := s.admin.api.TerminateSessionsByConnector(ctx, &api.TerminateSessionsByConnectorReq{ConnectorId: connectorID})
	if err != nil {
		s.adminRedirect(w, r, "", err.Error())
		return
	}
	s.detailRedirect(w, r, fmt.Sprintf("terminated %d session(s) from connector %q", resp.SessionsTerminated, connectorID), "")
}
