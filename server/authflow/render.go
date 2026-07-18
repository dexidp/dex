package authflow

import (
	"net/http"
	"path"
)

// renderError renders a user-facing HTML error page.
func (h *Handler) renderError(r *http.Request, w http.ResponseWriter, status int, description string) {
	if err := h.Templates.Err(r, w, status, description); err != nil {
		h.Logger.ErrorContext(r.Context(), "server template error", "err", err)
	}
}

// absPath returns the issuer path joined with the given path items.
func (h *Handler) absPath(pathItems ...string) string {
	return path.Join(append([]string{h.IssuerURL.Path}, pathItems...)...)
}

// absURL returns the absolute issuer URL for the given path items.
func (h *Handler) absURL(pathItems ...string) string {
	u := h.IssuerURL
	u.Path = h.absPath(pathItems...)
	return u.String()
}
