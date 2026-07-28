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

// handleAdmin renders what the API can tell us about this dex.
func (s *Server) handleAdmin(w http.ResponseWriter, r *http.Request) {
	data := AdminPageData{
		LogoURI:      dexLogoDataURI,
		AdminEnabled: true,
		Configured:   s.admin != nil,
		Notice:       r.URL.Query().Get("notice"),
		Error:        r.URL.Query().Get("error"),
	}

	// Without --grpc-addr the page explains itself rather than 404ing, which is
	// how you find out the feature exists at all.
	if s.admin == nil {
		s.renderer.RenderAdminPage(w, data)
		return
	}

	ctx, cancel := context.WithTimeout(r.Context(), 10*time.Second)
	defer cancel()

	if version, err := s.admin.api.GetVersion(ctx, &api.VersionReq{}); err == nil {
		data.Version = fmt.Sprintf("%s (API %d)", version.Server, version.Api)
	} else {
		data.Error = err.Error()
	}

	if clients, err := s.admin.api.ListClients(ctx, &api.ListClientReq{}); err == nil {
		for _, c := range clients.Clients {
			data.Clients = append(data.Clients, AdminClient{
				ID:           c.Id,
				Name:         c.Name,
				RedirectURIs: c.RedirectUris,
				Public:       c.Public,
			})
		}
	} else if data.Error == "" {
		data.Error = err.Error()
	}

	if passwords, err := s.admin.api.ListPasswords(ctx, &api.ListPasswordReq{}); err == nil {
		for _, p := range passwords.Passwords {
			data.Passwords = append(data.Passwords, AdminPassword{
				Email:    p.Email,
				Username: p.Username,
				UserID:   p.UserId,
			})
		}
	}

	s.renderer.RenderAdminPage(w, data)
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
			Id:           r.FormValue("id"),
			Name:         r.FormValue("name"),
			Secret:       r.FormValue("secret"),
			RedirectUris: splitLines(r.FormValue("redirect_uris")),
			Public:       r.FormValue("public") != "",
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
	u := "/admin"
	switch {
	case errMsg != "":
		u += "?error=" + url.QueryEscape(errMsg)
	case notice != "":
		u += "?notice=" + url.QueryEscape(notice)
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
