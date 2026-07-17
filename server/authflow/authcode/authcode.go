// Package authcode issues the authorization response once the flow is complete:
// it mints the auth code and, for implicit/hybrid flows, the access and ID
// tokens, then redirects (or renders the OOB page). One self-selecting handler
// per response_type populates a shared authResponse — fosite's
// AuthorizeEndpointHandler model.
package authcode

import (
	"context"
	"log/slog"
	"net/http"
	"net/url"
	"slices"
	"strconv"
	"time"

	"github.com/dexidp/dex/server/authflow/session"
	"github.com/dexidp/dex/server/authflow/web"
	"github.com/dexidp/dex/server/oauth2"
	"github.com/dexidp/dex/server/templates"
	"github.com/dexidp/dex/server/tokens"
	"github.com/dexidp/dex/storage"
)

// Issuer issues authorization-code and token responses.
type Issuer struct {
	*web.UI
	storage     storage.Storage
	tokenIssuer *tokens.Issuer
	sessions    *session.Manager
	templates   *templates.Templates
	now         func() time.Time
	logger      *slog.Logger
}

// New builds an Issuer.
func New(ui *web.UI, store storage.Storage, tokenIssuer *tokens.Issuer, sessions *session.Manager, tmpls *templates.Templates, now func() time.Time, logger *slog.Logger) *Issuer {
	return &Issuer{UI: ui, storage: store, tokenIssuer: tokenIssuer, sessions: sessions, templates: tmpls, now: now, logger: logger}
}

func (i *Issuer) Send(w http.ResponseWriter, r *http.Request, authReq storage.AuthRequest) {
	i.sessions.UpdateTokenIssuedAt(r, authReq.ClientID)

	ctx := r.Context()
	if i.now().After(authReq.Expiry) {
		i.RenderError(r, w, http.StatusBadRequest, "User session has expired.")
		return
	}

	if err := i.storage.DeleteAuthRequest(ctx, authReq.ID); err != nil {
		if err != storage.ErrNotFound {
			i.logger.ErrorContext(r.Context(), "Failed to delete authorization request", "err", err)
			i.RenderError(r, w, http.StatusInternalServerError, "Internal server error.")
		} else {
			i.RenderError(r, w, http.StatusBadRequest, "User session error.")
		}
		return
	}
	u, err := url.Parse(authReq.RedirectURI)
	if err != nil {
		i.RenderError(r, w, http.StatusInternalServerError, "Invalid redirect URI.")
		return
	}

	// Each response-type handler self-selects on authReq.ResponseTypes and
	// contributes its artifact to resp (fosite's AuthorizeEndpointHandler model).
	// Order matters: the code and access token feed the id_token signature.
	resp := &authResponse{}
	for _, handle := range []responseTypeHandler{
		i.issueCode,
		i.issueAccessToken,
		i.issueIDToken,
	} {
		if !handle(ctx, w, r, authReq, resp) {
			return // the handler already wrote the response (error or OOB)
		}
	}

	if resp.implicitOrHybrid {
		v := url.Values{}
		if resp.accessToken != "" {
			v.Set("access_token", resp.accessToken)
			v.Set("token_type", "bearer")
			// The hybrid flow with "code token" or "code id_token token" doesn't return an
			// "expires_in" value. If "code" wasn't provided, indicating the implicit flow,
			// don't add it.
			//
			// https://openid.net/specs/openid-connect-core-1_0.html#HybridAuthResponse
			if resp.code.ID == "" {
				v.Set("expires_in", strconv.Itoa(int(resp.idTokenExpiry.Sub(i.now()).Seconds())))
			}
		}
		v.Set("state", authReq.State)
		if resp.idToken != "" {
			v.Set("id_token", resp.idToken)
		}
		if resp.code.ID != "" {
			v.Set("code", resp.code.ID)
		}

		// Implicit and hybrid flows return their values as part of the fragment.
		//
		//   HTTP/1.1 303 See Other
		//   Location: https://client.example.org/cb#
		//     access_token=SlAV32hkKG
		//     &token_type=bearer
		//     &id_token=eyJ0 ... NiJ9.eyJ1c ... I6IjIifX0.DeWt4Qu ... ZXso
		//     &expires_in=3600
		//     &state=af0ifjsldkj
		//
		u.Fragment = v.Encode()
	} else {
		// The code flow add values to the URL query.
		//
		//   HTTP/1.1 303 See Other
		//   Location: https://client.example.org/cb?
		//     code=SplxlOBeZQQYbYS6WxSbIA
		//     &state=af0ifjsldkj
		//
		q := u.Query()
		q.Set("code", resp.code.ID)
		q.Set("state", authReq.State)
		u.RawQuery = q.Encode()
	}

	http.Redirect(w, r, u.String(), http.StatusSeeOther)
}

// authResponse accumulates the artifacts each response-type handler produces
// for the authorization response.
type authResponse struct {
	// Was the initial request using the implicit or hybrid flow instead of the
	// "normal" code flow?
	implicitOrHybrid bool

	// Only present in hybrid or code flow. code.ID == "" if this is not set.
	code storage.AuthCode

	// Access token, present when response_type includes "token".
	accessToken string

	// ID token, present when response_type includes "id_token". Only valid for
	// implicit and hybrid flows.
	idToken       string
	idTokenExpiry time.Time
}

// responseTypeHandler produces the response for a single OAuth2 response_type.
// It self-selects on authReq.ResponseTypes, populates resp, and returns false
// (after writing an error or OOB page itself) to abort the response.
type responseTypeHandler func(ctx context.Context, w http.ResponseWriter, r *http.Request, authReq storage.AuthRequest, resp *authResponse) bool

// issueCode handles the "code" response_type: it mints and stores an auth code.
func (i *Issuer) issueCode(ctx context.Context, w http.ResponseWriter, r *http.Request, authReq storage.AuthRequest, resp *authResponse) bool {
	if !slices.Contains(authReq.ResponseTypes, oauth2.ResponseTypeCode) {
		return true
	}
	resp.code = storage.AuthCode{
		ID:            storage.NewID(),
		ClientID:      authReq.ClientID,
		ConnectorID:   authReq.ConnectorID,
		Nonce:         authReq.Nonce,
		Scopes:        authReq.Scopes,
		Claims:        authReq.Claims,
		Expiry:        i.now().Add(time.Minute * 30),
		RedirectURI:   authReq.RedirectURI,
		ConnectorData: authReq.ConnectorData,
		PKCE:          authReq.PKCE,
		AuthTime:      authReq.AuthTime,
	}
	if err := i.storage.CreateAuthCode(ctx, resp.code); err != nil {
		i.logger.ErrorContext(r.Context(), "Failed to create auth code", "err", err)
		i.RenderError(r, w, http.StatusInternalServerError, "Internal server error.")
		return false
	}

	// Implicit and hybrid flows that try to use the OOB redirect URI are
	// rejected earlier. If we got here we're using the code flow.
	if authReq.RedirectURI == oauth2.RedirectURIOOB {
		if err := i.templates.OOB(r, w, resp.code.ID); err != nil {
			i.logger.ErrorContext(r.Context(), "server template error", "err", err)
		}
		return false // OOB fully rendered the response
	}
	return true
}

// issueAccessToken handles the "token" response_type: it signs an access token.
func (i *Issuer) issueAccessToken(ctx context.Context, w http.ResponseWriter, r *http.Request, authReq storage.AuthRequest, resp *authResponse) bool {
	if !slices.Contains(authReq.ResponseTypes, oauth2.ResponseTypeToken) {
		return true
	}
	resp.implicitOrHybrid = true
	accessToken, _, err := i.tokenIssuer.SignAccessToken(ctx, tokens.Authorization{
		Client:      storage.Client{ID: authReq.ClientID},
		Claims:      authReq.Claims,
		Scopes:      authReq.Scopes,
		ConnectorID: authReq.ConnectorID,
		Nonce:       authReq.Nonce,
		AuthTime:    authReq.AuthTime,
	})
	if err != nil {
		i.logger.ErrorContext(r.Context(), "failed to create new access token", "err", err)
		i.tokenErrHelper(w, oauth2.ServerError, "", http.StatusInternalServerError)
		return false
	}
	resp.accessToken = accessToken
	return true
}

// issueIDToken handles the "id_token" response_type. It runs after issueCode and
// issueAccessToken because the id_token signature binds the code and access token.
func (i *Issuer) issueIDToken(ctx context.Context, w http.ResponseWriter, r *http.Request, authReq storage.AuthRequest, resp *authResponse) bool {
	if !slices.Contains(authReq.ResponseTypes, oauth2.ResponseTypeIDToken) {
		return true
	}
	resp.implicitOrHybrid = true
	idToken, idTokenExpiry, err := i.tokenIssuer.SignIDToken(ctx, tokens.Authorization{
		Client:      storage.Client{ID: authReq.ClientID},
		Claims:      authReq.Claims,
		Scopes:      authReq.Scopes,
		ConnectorID: authReq.ConnectorID,
		Nonce:       authReq.Nonce,
		AuthTime:    authReq.AuthTime,
	}, resp.accessToken, resp.code.ID)
	if err != nil {
		i.logger.ErrorContext(r.Context(), "failed to create ID token", "err", err)
		i.tokenErrHelper(w, oauth2.ServerError, "", http.StatusInternalServerError)
		return false
	}
	resp.idToken = idToken
	resp.idTokenExpiry = idTokenExpiry
	return true
}

// scopesCoveredByConsent checks whether the approved scopes cover all requested scopes.
// The openid scope is excluded from the comparison as it is a technical scope
// that does not require user consent.
func (i *Issuer) tokenErrHelper(w http.ResponseWriter, typ string, description string, statusCode int) {
	if err := oauth2.WriteError(w, typ, description, statusCode); err != nil {
		i.logger.Error("token error response", "err", err)
	}
}
