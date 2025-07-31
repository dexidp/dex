package server

import (
	"fmt"
	"net/http"
	"net/url"

	"github.com/gorilla/mux"

	"github.com/dexidp/dex/connector"
	"github.com/dexidp/dex/pkg/otel/traces"
)

func (s *Server) handleConnectorLogin(w http.ResponseWriter, r *http.Request) {
	ctx, span := traces.InstrumentHandler(r)
	defer span.End()

	authReq, err := s.parseAuthorizationRequest(r)
	if err != nil {
		s.logger.ErrorContext(ctx, "failed to parse authorization request", "err", err)
		switch authErr := err.(type) {
		case *redirectedAuthErr:
			authErr.Handler().ServeHTTP(w, r)
		case *displayedAuthErr:
			s.renderError(r, w, authErr.Status, err.Error())
		default:
			panic("unsupported error type")
		}

		return
	}

	connID, err := url.PathUnescape(mux.Vars(r)["connector"])
	if err != nil {
		s.logger.ErrorContext(ctx, "failed to parse connector", "err", err)
		s.renderError(r, w, http.StatusBadRequest, "Requested resource does not exist")
		return
	}

	conn, err := s.getConnector(ctx, connID)
	if err != nil {
		s.logger.ErrorContext(ctx, "Failed to get connector", "err", err)
		s.renderError(r, w, http.StatusBadRequest, "Connector failed to initialize")
		return
	}

	// Set the connector being used for the login.
	if authReq.ConnectorID != "" && authReq.ConnectorID != connID {
		s.logger.ErrorContext(ctx, "mismatched connector ID in auth request",
			"auth_request_connector_id", authReq.ConnectorID, "connector_id", connID)
		s.renderError(r, w, http.StatusBadRequest, "Bad connector ID")
		return
	}

	authReq.ConnectorID = connID

	// Actually create the auth request
	authReq.Expiry = s.now().Add(s.authRequestsValidFor)
	if err := s.storage.CreateAuthRequest(ctx, *authReq); err != nil {
		s.logger.ErrorContext(ctx, "failed to create authorization request", "err", err)
		s.renderError(r, w, http.StatusInternalServerError, "Failed to connect to the database.")
		return
	}

	scopes := parseScopes(authReq.Scopes)

	// Work out where the "Select another login method" link should go.
	backLink := ""
	if len(s.connectors) > 1 {
		backLinkURL := url.URL{
			Path:     s.absPath("/auth"),
			RawQuery: r.Form.Encode(),
		}
		backLink = backLinkURL.String()
	}

	switch r.Method {
	case http.MethodGet:
		switch conn := conn.Connector.(type) {
		case connector.CallbackConnector:

			// Use the auth request ID as the "state" token.
			//
			// TODO(ericchiang): Is this appropriate or should we also be using a nonce?
			callbackURL, err := conn.LoginURL(scopes, s.absURL("/callback"), authReq.ID)
			if err != nil {
				s.logger.ErrorContext(ctx, "connector returned error when creating callback", "connector_id", connID, "err", err)
				s.renderError(r, w, http.StatusInternalServerError, "Login error.")
				return
			}
			http.Redirect(w, r, callbackURL, http.StatusFound)
		case connector.PasswordConnector:

			loginURL := url.URL{
				Path: s.absPath("/auth", connID, "login"),
			}
			q := loginURL.Query()
			q.Set("state", authReq.ID)
			q.Set("back", backLink)
			loginURL.RawQuery = q.Encode()

			http.Redirect(w, r, loginURL.String(), http.StatusFound)
		case connector.SAMLConnector:

			action, value, err := conn.POSTData(scopes, authReq.ID)
			if err != nil {
				s.logger.ErrorContext(r.Context(), "creating SAML data", "err", err)
				s.renderError(r, w, http.StatusInternalServerError, "Connector Login Error")
				return
			}

			// TODO(ericchiang): Don't inline this.
			fmt.Fprintf(w, `<!DOCTYPE html>
			  <html lang="en">
			  <head>
			    <meta http-equiv="content-type" content="text/html; charset=utf-8">
			    <title>SAML login</title>
			  </head>
			  <body>
			    <form method="post" action="%s" >
				    <input type="hidden" name="SAMLRequest" value="%s" />
				    <input type="hidden" name="RelayState" value="%s" />
			    </form>
				<script>
				    document.forms[0].submit();
				</script>
			  </body>
			  </html>`, action, value, authReq.ID)
		default:
			s.renderError(r, w, http.StatusBadRequest, "Requested resource does not exist.")
		}
	default:
		s.renderError(r, w, http.StatusBadRequest, "Unsupported request method.")
	}
}
