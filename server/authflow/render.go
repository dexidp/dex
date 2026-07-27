package authflow

import (
	"net/http"

	"github.com/dexidp/dex/server/templates"
)

// renderError renders a user-facing HTML error page.
func (h *Handler) renderError(r *http.Request, w http.ResponseWriter, status int, description string) {
	templates.RenderError(h.Templates, h.Logger, r, w, status, description)
}
