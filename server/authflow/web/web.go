// Package web is the shared browser-facing infrastructure for the interactive
// auth flow: HTML error rendering and issuer-relative URL building. The flow's
// Handler and its MFA component embed a *UI so they render errors and build
// URLs the same way, without duplicating the logic.
package web

import (
	"log/slog"
	"net/http"
	"net/url"
	"path"

	"github.com/dexidp/dex/server/internal"
	"github.com/dexidp/dex/server/templates"
	"github.com/dexidp/dex/storage"
)

// UI renders error pages and builds issuer-relative URLs.
type UI struct {
	templates *templates.Templates
	issuerURL url.URL
	logger    *slog.Logger
}

// New builds a UI.
func New(templates *templates.Templates, issuerURL url.URL, logger *slog.Logger) *UI {
	return &UI{templates: templates, issuerURL: issuerURL, logger: logger}
}

// RenderError renders a user-facing error page.
func (u *UI) RenderError(r *http.Request, rw http.ResponseWriter, status int, description string) {
	if err := u.templates.Err(r, rw, status, description); err != nil {
		u.logger.ErrorContext(r.Context(), "server template error", "err", err)
	}
}

// AbsPath returns the issuer path joined with the given path items.
func (u *UI) AbsPath(pathItems ...string) string {
	paths := make([]string, len(pathItems)+1)
	paths[0] = u.issuerURL.Path
	copy(paths[1:], pathItems)
	return path.Join(paths...)
}

// AbsURL returns the absolute issuer URL for the given path items.
func (u *UI) AbsURL(pathItems ...string) string {
	v := u.issuerURL
	v.Path = u.AbsPath(pathItems...)
	return v.String()
}

// BuildApprovalURL builds an HMAC-protected approval URL.
func (u *UI) BuildApprovalURL(authReq storage.AuthRequest) string {
	v := url.Values{}
	v.Set("req", authReq.ID)
	v.Set("hmac", internal.ComputeHMAC(authReq.HMACKey, authReq.ID, ""))
	return u.AbsPath("/approval") + "?" + v.Encode()
}
