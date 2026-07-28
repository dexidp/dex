package home

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"time"

	"github.com/dexidp/dex/server/oauth2"
	"github.com/dexidp/dex/server/router"
	"github.com/dexidp/dex/server/session"
	"github.com/dexidp/dex/server/templates"
	"github.com/dexidp/dex/storage"
)

// Handler serves the landing page. When sessions are enabled and a home template
// is available it renders the rich page (with logged-in details); otherwise it
// falls back to a minimal inline page.
type Handler struct {
	IssuerURL oauth2.IssuerURL
	Storage   storage.Storage
	Templates *templates.Templates
	Logger    *slog.Logger
	// Sessions is the shared session manager; nil (or with a nil Config) when
	// sessions are disabled.
	Sessions *session.Manager
}

// Mount registers the landing-page route.
func (h *Handler) Mount(m router.Mux) {
	m.HandleCORS("/", h.handle)
}

func (h *Handler) renderError(r *http.Request, w http.ResponseWriter, status int, description string) {
	templates.RenderError(h.Templates, h.Logger, r, w, status, description)
}

func (h *Handler) handle(w http.ResponseWriter, r *http.Request) {
	if h.Sessions == nil || h.Sessions.Config == nil || !h.Templates.HasHome() {
		h.handleInline(w, r)
		return
	}

	ctx := r.Context()

	data := templates.HomeData{
		DiscoveryURL: h.IssuerURL.JoinPath(".well-known", "openid-configuration").String(),
		LogoutURL:    h.IssuerURL.AbsURL("/logout"),
	}

	// ValidSession enforces the nonce AND absolute/idle expiry (clearing an
	// expired session), so an expired-but-not-yet-purged cookie no longer renders
	// a logged-in page.
	if session := h.Sessions.ValidSession(ctx, w, r); session != nil {
		data.LoggedIn = true
		data.IPAddress = session.IPAddress
		data.UserAgent = session.UserAgent
		if !session.CreatedAt.IsZero() {
			data.SignedInEpoch = session.CreatedAt.Unix()
		}
		data.SessionExpiresEpoch, data.SessionExpiryIsIdle = sessionExpiry(session)
		h.populateData(ctx, &data, session.UserID, session.ConnectorID)
		data.SignedInText = utcText(data.SignedInEpoch)
		data.SessionExpiresText = utcText(data.SessionExpiresEpoch)
	}

	if err := h.Templates.Home(r, w, data); err != nil {
		h.Logger.ErrorContext(ctx, "failed to render home template", "err", err)
		h.renderError(r, w, http.StatusInternalServerError, "Internal server error.")
	}
}

// utcText renders an epoch for readers whose browser will not run the page's
// script, which otherwise restates these times in the local timezone. Without
// it those rows are an em dash. Empty for 0, which drops the row entirely.
func utcText(epoch int64) string {
	if epoch == 0 {
		return ""
	}
	return time.Unix(epoch, 0).UTC().Format("2 Jan 2006, 15:04 UTC")
}

// sessionExpiry reports when the session ends: the earlier of its absolute
// lifetime and its idle timeout, since whichever comes first logs the user out.
// It returns 0 when neither is set, which drops the row from the page.
//
// The second return says the idle timeout won. That distinction has to reach
// the page: the absolute expiry is a fixed moment, while the idle one slides
// forward every time the session is used, so labeling both "session ends"
// would state a deadline that is not one.
func sessionExpiry(s *storage.AuthSession) (int64, bool) {
	expiry, idle := s.AbsoluteExpiry, false
	if expiry.IsZero() || (!s.IdleExpiry.IsZero() && s.IdleExpiry.Before(expiry)) {
		expiry, idle = s.IdleExpiry, true
	}
	if expiry.IsZero() {
		return 0, false
	}
	return expiry.Unix(), idle
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

	conn, err := h.Storage.GetConnector(ctx, connectorID)
	if err == nil {
		data.ConnectorName = conn.Name
	}
}
