// Package home serves the dex landing page at "/".
package home

import (
	"context"
	"crypto/subtle"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"net/url"
	"path"

	"github.com/dexidp/dex/server/internal"
	"github.com/dexidp/dex/server/router"
	"github.com/dexidp/dex/server/templates"
	"github.com/dexidp/dex/storage"
)

// Handler serves the landing page. When sessions are enabled and a home template
// is available it renders the rich page (with logged-in details); otherwise it
// falls back to a minimal inline page.
type Handler struct {
	IssuerURL           url.URL
	Storage             storage.Storage
	Templates           *templates.Templates
	Logger              *slog.Logger
	SessionsEnabled     bool
	CookieName          string
	CookieEncryptionKey []byte
}

// Mount registers the landing-page route.
func (h *Handler) Mount(m router.Mux) {
	m.HandleCORS("/", h.handle)
}

func (h *Handler) renderError(r *http.Request, w http.ResponseWriter, status int, description string) {
	if err := h.Templates.Err(r, w, status, description); err != nil {
		h.Logger.ErrorContext(r.Context(), "server template error", "err", err)
	}
}

func (h *Handler) handle(w http.ResponseWriter, r *http.Request) {
	if !h.SessionsEnabled || !h.Templates.HasHome() {
		h.handleInline(w, r)
		return
	}

	ctx := r.Context()

	logoutURL := h.IssuerURL
	logoutURL.Path = path.Join(logoutURL.Path, "/logout")

	data := templates.HomeData{
		DiscoveryURL: h.IssuerURL.JoinPath(".well-known", "openid-configuration").String(),
		LogoutURL:    logoutURL.String(),
	}

	if cookie, err := r.Cookie(h.CookieName); err == nil && cookie.Value != "" {
		if userID, connectorID, nonce, err := internal.ParseSessionCookie(cookie.Value, h.CookieEncryptionKey); err == nil {
			session, err := h.Storage.GetAuthSession(ctx, userID, connectorID)
			if err == nil && subtle.ConstantTimeCompare([]byte(session.Nonce), []byte(nonce)) == 1 {
				data.LoggedIn = true
				data.IPAddress = session.IPAddress
				data.UserAgent = session.UserAgent
				h.populateData(ctx, &data, userID, connectorID)
			} else if err != nil && !errors.Is(err, storage.ErrNotFound) {
				h.Logger.ErrorContext(ctx, "home: failed to get auth session", "err", err)
			}
		}
	}

	if err := h.Templates.Home(r, w, data); err != nil {
		h.Logger.ErrorContext(ctx, "failed to render home template", "err", err)
		h.renderError(r, w, http.StatusInternalServerError, "Internal server error.")
	}
}

func (h *Handler) handleInline(w http.ResponseWriter, r *http.Request) {
	_, err := fmt.Fprintf(w, `<!DOCTYPE html>
		<title>Dex</title>
		<h1>Dex IdP</h1>
		<h3>A Federated OpenID Connect Provider</h3>
		<p><a href=%q>Discovery</a></p>`,
		h.IssuerURL.JoinPath(".well-known", "openid-configuration").String())
	if err != nil {
		h.Logger.Error("failed to write response", "err", err)
		h.renderError(r, w, http.StatusInternalServerError, "Handling the / path error.")
	}
}

func (h *Handler) populateData(ctx context.Context, data *templates.HomeData, userID, connectorID string) {
	ui, err := h.Storage.GetUserIdentity(ctx, userID, connectorID)
	if err != nil {
		if !errors.Is(err, storage.ErrNotFound) {
			h.Logger.ErrorContext(ctx, "home: failed to get user identity", "err", err)
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

	conn, err := h.Storage.GetConnector(ctx, connectorID)
	if err == nil {
		data.ConnectorName = conn.Name
	}
}
