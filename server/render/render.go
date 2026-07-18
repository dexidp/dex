package render

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
	Templates *templates.Templates
	IssuerURL url.URL
	Logger    *slog.Logger
}

// RenderError renders a user-facing error page.
func (u *UI) RenderError(r *http.Request, rw http.ResponseWriter, status int, description string) {
	if err := u.Templates.Err(r, rw, status, description); err != nil {
		u.Logger.ErrorContext(r.Context(), "server template error", "err", err)
	}
}

// AbsPath returns the issuer path joined with the given path items.
func (u *UI) AbsPath(pathItems ...string) string {
	paths := make([]string, len(pathItems)+1)
	paths[0] = u.IssuerURL.Path
	copy(paths[1:], pathItems)
	return path.Join(paths...)
}

// AbsURL returns the absolute issuer URL for the given path items.
func (u *UI) AbsURL(pathItems ...string) string {
	v := u.IssuerURL
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

// BuildContinueURL builds an HMAC-protected URL for the flow dispatcher: the
// central point every step returns to so the next step (or issuance) is decided
// in one place.
func (u *UI) BuildContinueURL(authReq storage.AuthRequest) string {
	v := url.Values{}
	v.Set("req", authReq.ID)
	v.Set("hmac", internal.ComputeHMAC(authReq.HMACKey, authReq.ID, "continue"))
	return u.AbsPath("/continue") + "?" + v.Encode()
}

// RedirectAuthError redirects back to the client's redirect_uri with an OAuth2
// error. Steps use it when prompt=none forbids the interaction they would need.
func (u *UI) RedirectAuthError(w http.ResponseWriter, r *http.Request, authReq storage.AuthRequest, errType, description string) {
	v := url.Values{}
	v.Add("state", authReq.State)
	v.Add("error", errType)
	if description != "" {
		v.Add("error_description", description)
	}

	redirectURI, err := url.Parse(authReq.RedirectURI)
	if err != nil {
		http.Error(w, "Invalid redirect URI", http.StatusBadRequest)
		return
	}

	query := redirectURI.Query()
	for key, values := range v {
		for _, value := range values {
			query.Add(key, value)
		}
	}
	redirectURI.RawQuery = query.Encode()

	http.Redirect(w, r, redirectURI.String(), http.StatusSeeOther)
}
