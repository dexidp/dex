package server

import (
	"context"
	"errors"
	"fmt"
	"net/http"

	"github.com/dexidp/dex/storage"
)

func (s *Server) handleHome(w http.ResponseWriter, r *http.Request) {
	if s.sessionConfig == nil || s.templates.homeTmpl == nil {
		s.handleHomeInline(w, r)
		return
	}

	ctx := r.Context()
	data := homeData{
		DiscoveryURL: s.issuerURL.JoinPath(".well-known", "openid-configuration").String(),
		LogoutURL:    s.absURL("/logout"),
	}

	if cookie, err := r.Cookie(s.sessionConfig.CookieName); err == nil && cookie.Value != "" {
		if userID, connectorID, nonce, err := parseSessionCookie(cookie.Value); err == nil {
			session, err := s.storage.GetAuthSession(ctx, userID, connectorID)
			if err == nil && session.Nonce == nonce {
				data.LoggedIn = true
				data.IPAddress = session.IPAddress
				data.UserAgent = session.UserAgent
				s.populateHomeData(ctx, &data, userID, connectorID)
			} else if err != nil && !errors.Is(err, storage.ErrNotFound) {
				s.logger.ErrorContext(ctx, "home: failed to get auth session", "err", err)
			}
		}
	}

	if err := s.templates.home(r, w, data); err != nil {
		s.logger.ErrorContext(ctx, "failed to render home template", "err", err)
		s.renderError(r, w, http.StatusInternalServerError, "Internal server error.")
	}
}

func (s *Server) handleHomeInline(w http.ResponseWriter, r *http.Request) {
	_, err := fmt.Fprintf(w, `<!DOCTYPE html>
		<title>Dex</title>
		<h1>Dex IdP</h1>
		<h3>A Federated OpenID Connect Provider</h3>
		<p><a href=%q>Discovery</a></p>`,
		s.issuerURL.JoinPath(".well-known", "openid-configuration").String())
	if err != nil {
		s.logger.Error("failed to write response", "err", err)
		s.renderError(r, w, http.StatusInternalServerError, "Handling the / path error.")
	}
}

func (s *Server) populateHomeData(ctx context.Context, data *homeData, userID, connectorID string) {
	ui, err := s.storage.GetUserIdentity(ctx, userID, connectorID)
	if err != nil {
		if !errors.Is(err, storage.ErrNotFound) {
			s.logger.ErrorContext(ctx, "home: failed to get user identity", "err", err)
		}
		return
	}

	data.Username = ui.Claims.PreferredUsername
	if data.Username == "" {
		data.Username = ui.Claims.Username
	}
	data.Email = ui.Claims.Email
	data.EmailVerified = ui.Claims.EmailVerified
	data.Groups = ui.Claims.Groups
	if !ui.LastLogin.IsZero() {
		data.LastLoginEpoch = ui.LastLogin.Unix()
	}

	conn, err := s.storage.GetConnector(ctx, connectorID)
	if err == nil {
		data.ConnectorName = conn.Name
	}
}
