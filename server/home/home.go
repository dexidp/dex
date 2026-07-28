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
		data.SignedInISO, data.SignedInText = timeFields(session.CreatedAt)
		expiry, idle := sessionExpiry(session)
		data.SessionExpiresISO, data.SessionExpiresText = timeFields(expiry)
		data.SessionExpiryIsIdle = idle
		h.populateData(ctx, &data, session.UserID, session.ConnectorID)
	}

	if err := h.Templates.Home(r, w, data); err != nil {
		h.Logger.ErrorContext(ctx, "failed to render home template", "err", err)
		h.renderError(r, w, http.StatusInternalServerError, "Internal server error.")
	}
}

// timeFields renders a timestamp for the page: an ISO 8601 string for the <time>
// element's datetime attribute, and the UTC text shown to readers whose browser
// will not run the script that restates it in their own timezone. Both are empty
// for the zero time, which drops the row entirely.
func timeFields(t time.Time) (iso, text string) {
	if t.IsZero() {
		return "", ""
	}
	return t.UTC().Format(time.RFC3339), t.UTC().Format("2 Jan 2006, 15:04 UTC")
}

// sessionExpiry reports when the session ends: the earlier of its absolute
// lifetime and its idle timeout, since whichever comes first logs the user out.
// It returns the zero time when neither is set, which drops the row.
//
// The second return says the idle timeout won. That distinction has to reach
// the page: the absolute expiry is a fixed moment, while the idle one slides
// forward every time the session is used, so labeling both "session ends"
// would state a deadline that is not one.
func sessionExpiry(s *storage.AuthSession) (time.Time, bool) {
	expiry, idle := s.AbsoluteExpiry, false
	if expiry.IsZero() || (!s.IdleExpiry.IsZero() && s.IdleExpiry.Before(expiry)) {
		expiry, idle = s.IdleExpiry, true
	}
	if expiry.IsZero() {
		return time.Time{}, false
	}
	return expiry, idle
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
