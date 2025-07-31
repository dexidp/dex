package server

import (
	"html/template"
	"net/http"
	"net/url"

	"github.com/dexidp/dex/pkg/otel/traces"
)

// handleAuthorization handles the OAuth2 auth endpoint.
func (s *Server) handleAuthorization(w http.ResponseWriter, r *http.Request) {
	ctx, span := traces.InstrumentHandler(r)
	defer span.End()

	// Extract the arguments
	if err := r.ParseForm(); err != nil {
		s.logger.ErrorContext(ctx, "failed to parse arguments", "err", err)
		s.renderError(r, w, http.StatusBadRequest, err.Error())
		return
	}

	connectorID := r.Form.Get("connector_id")
	connectors, err := s.storage.ListConnectors(ctx)
	if err != nil {
		s.logger.ErrorContext(r.Context(), "failed to get list of connectors", "err", err)
		s.renderError(r, w, http.StatusInternalServerError, "Failed to retrieve connector list.")
		return
	}

	// We don't need connector_id any more
	r.Form.Del("connector_id")

	// Construct a URL with all of the arguments in its query
	connURL := url.URL{
		RawQuery: r.Form.Encode(),
	}

	// Redirect if a client chooses a specific connector_id
	if connectorID != "" {
		for _, c := range connectors {
			if c.ID == connectorID {
				connURL.Path = s.absPath("/auth", url.PathEscape(c.ID))

				http.Redirect(w, r, connURL.String(), http.StatusFound)
				return
			}
		}
		s.renderError(r, w, http.StatusBadRequest, "Connector ID does not match a valid Connector")
		return
	}

	if len(connectors) == 1 && !s.alwaysShowLogin {
		connURL.Path = s.absPath("/auth", url.PathEscape(connectors[0].ID))

		http.Redirect(w, r, connURL.String(), http.StatusFound)
		return
	}

	connectorInfos := make([]connectorInfo, len(connectors))
	for index, conn := range connectors {
		connURL.Path = s.absPath("/auth", url.PathEscape(conn.ID))

		connectorInfos[index] = connectorInfo{
			ID:   conn.ID,
			Name: conn.Name,
			Type: conn.Type,
			URL:  template.URL(connURL.String()),
		}
	}

	if err := s.templates.login(r, w, connectorInfos); err != nil {
		s.logger.ErrorContext(r.Context(), "server template error", "err", err)
	}
}
