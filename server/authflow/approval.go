package authflow

// approval.go handles the consent/approval step and builds the authorization-code
// or token response back to the client.

import (
	"context"
	"net/http"
	"net/url"
	"slices"
	"strconv"
	"time"

	"github.com/dexidp/dex/pkg/featureflags"
	"github.com/dexidp/dex/server/internal"
	"github.com/dexidp/dex/server/oauth2"
	"github.com/dexidp/dex/server/tokens"
	"github.com/dexidp/dex/storage"
)

func (h *Handler) handleApproval(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	macEncoded := r.FormValue("hmac")
	if macEncoded == "" {
		h.RenderError(r, w, http.StatusUnauthorized, "Unauthorized request")
		return
	}
	authReq, err := h.storage.GetAuthRequest(ctx, r.FormValue("req"))
	if err != nil {
		if err == storage.ErrNotFound {
			h.RenderError(r, w, http.StatusBadRequest, "User session error.")
			return
		}
		h.logger.ErrorContext(r.Context(), "failed to get auth request", "err", err)
		h.RenderError(r, w, http.StatusInternalServerError, "Database error.")
		return
	}
	if !authReq.LoggedIn {
		h.logger.ErrorContext(r.Context(), "auth request does not have an identity for approval")
		h.RenderError(r, w, http.StatusInternalServerError, "Login process not yet finalized.")
		return
	}

	// Defensively re-check the next step: MFA may have been enabled after the flow
	// started, in which case the user is sent through it before consent.
	step, err := h.nextAuthStep(ctx, &authReq)
	if err != nil {
		h.logger.ErrorContext(ctx, "failed to determine next auth step", "err", err)
		h.RenderError(r, w, http.StatusInternalServerError, "Internal server error.")
		return
	}
	if mfa, ok := step.(mfaStep); ok {
		http.Redirect(w, r, h.mfa.BuildRedirectURL(authReq, mfa.authenticator), http.StatusSeeOther)
		return
	}

	if !internal.VerifyHMAC(authReq.HMACKey, macEncoded, authReq.ID, "") {
		h.RenderError(r, w, http.StatusUnauthorized, "Unauthorized request")
		return
	}

	switch r.Method {
	case http.MethodGet:
		// Skip the approval page and issue the code directly when consent isn't
		// needed. This also covers the MFA redirect case: after MFA completion the
		// user lands here via GET and shouldn't see the consent screen again.
		if _, ok := step.(issueStep); ok {
			h.sendCodeResponse(w, r, authReq)
			return
		}

		client, err := h.storage.GetClient(ctx, authReq.ClientID)
		if err != nil {
			h.logger.ErrorContext(r.Context(), "Failed to get client", "client_id", authReq.ClientID, "err", err)
			h.RenderError(r, w, http.StatusInternalServerError, "Failed to retrieve client.")
			return
		}
		if err := h.templates.Approval(r, w, authReq.ID, authReq.Claims.Username, client.Name, authReq.Scopes); err != nil {
			h.logger.ErrorContext(r.Context(), "server template error", "err", err)
		}
	case http.MethodPost:
		if r.FormValue("approval") != "approve" {
			h.RenderError(r, w, http.StatusInternalServerError, "Approval rejected.")
			return
		}
		// Persist user-approved scopes as consent for this client.
		if featureflags.SessionsEnabled.Enabled() {
			if err := h.storage.UpdateUserIdentity(ctx, authReq.Claims.UserID, authReq.ConnectorID, func(old storage.UserIdentity) (storage.UserIdentity, error) {
				if old.Consents == nil {
					old.Consents = make(map[string][]string)
				}
				old.Consents[authReq.ClientID] = authReq.Scopes
				return old, nil
			}); err != nil {
				h.logger.ErrorContext(ctx, "failed to update user identity consents", "err", err)
			}
		}
		h.sendCodeResponse(w, r, authReq)
	}
}

func (h *Handler) sendCodeResponse(w http.ResponseWriter, r *http.Request, authReq storage.AuthRequest) {
	h.sessions.UpdateTokenIssuedAt(r, authReq.ClientID)

	ctx := r.Context()
	if h.now().After(authReq.Expiry) {
		h.RenderError(r, w, http.StatusBadRequest, "User session has expired.")
		return
	}

	if err := h.storage.DeleteAuthRequest(ctx, authReq.ID); err != nil {
		if err != storage.ErrNotFound {
			h.logger.ErrorContext(r.Context(), "Failed to delete authorization request", "err", err)
			h.RenderError(r, w, http.StatusInternalServerError, "Internal server error.")
		} else {
			h.RenderError(r, w, http.StatusBadRequest, "User session error.")
		}
		return
	}
	u, err := url.Parse(authReq.RedirectURI)
	if err != nil {
		h.RenderError(r, w, http.StatusInternalServerError, "Invalid redirect URI.")
		return
	}

	// Each response-type handler self-selects on authReq.ResponseTypes and
	// contributes its artifact to resp (fosite's AuthorizeEndpointHandler model).
	// Order matters: the code and access token feed the id_token signature.
	resp := &authResponse{}
	for _, handle := range []responseTypeHandler{
		h.issueCode,
		h.issueAccessToken,
		h.issueIDToken,
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
				v.Set("expires_in", strconv.Itoa(int(resp.idTokenExpiry.Sub(h.now()).Seconds())))
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
func (h *Handler) issueCode(ctx context.Context, w http.ResponseWriter, r *http.Request, authReq storage.AuthRequest, resp *authResponse) bool {
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
		Expiry:        h.now().Add(time.Minute * 30),
		RedirectURI:   authReq.RedirectURI,
		ConnectorData: authReq.ConnectorData,
		PKCE:          authReq.PKCE,
		AuthTime:      authReq.AuthTime,
	}
	if err := h.storage.CreateAuthCode(ctx, resp.code); err != nil {
		h.logger.ErrorContext(r.Context(), "Failed to create auth code", "err", err)
		h.RenderError(r, w, http.StatusInternalServerError, "Internal server error.")
		return false
	}

	// Implicit and hybrid flows that try to use the OOB redirect URI are
	// rejected earlier. If we got here we're using the code flow.
	if authReq.RedirectURI == oauth2.RedirectURIOOB {
		if err := h.templates.OOB(r, w, resp.code.ID); err != nil {
			h.logger.ErrorContext(r.Context(), "server template error", "err", err)
		}
		return false // OOB fully rendered the response
	}
	return true
}

// issueAccessToken handles the "token" response_type: it signs an access token.
func (h *Handler) issueAccessToken(ctx context.Context, w http.ResponseWriter, r *http.Request, authReq storage.AuthRequest, resp *authResponse) bool {
	if !slices.Contains(authReq.ResponseTypes, oauth2.ResponseTypeToken) {
		return true
	}
	resp.implicitOrHybrid = true
	accessToken, _, err := h.issuer.SignAccessToken(ctx, tokens.Authorization{
		Client:      storage.Client{ID: authReq.ClientID},
		Claims:      authReq.Claims,
		Scopes:      authReq.Scopes,
		ConnectorID: authReq.ConnectorID,
		Nonce:       authReq.Nonce,
		AuthTime:    authReq.AuthTime,
	})
	if err != nil {
		h.logger.ErrorContext(r.Context(), "failed to create new access token", "err", err)
		h.tokenErrHelper(w, oauth2.ServerError, "", http.StatusInternalServerError)
		return false
	}
	resp.accessToken = accessToken
	return true
}

// issueIDToken handles the "id_token" response_type. It runs after issueCode and
// issueAccessToken because the id_token signature binds the code and access token.
func (h *Handler) issueIDToken(ctx context.Context, w http.ResponseWriter, r *http.Request, authReq storage.AuthRequest, resp *authResponse) bool {
	if !slices.Contains(authReq.ResponseTypes, oauth2.ResponseTypeIDToken) {
		return true
	}
	resp.implicitOrHybrid = true
	idToken, idTokenExpiry, err := h.issuer.SignIDToken(ctx, tokens.Authorization{
		Client:      storage.Client{ID: authReq.ClientID},
		Claims:      authReq.Claims,
		Scopes:      authReq.Scopes,
		ConnectorID: authReq.ConnectorID,
		Nonce:       authReq.Nonce,
		AuthTime:    authReq.AuthTime,
	}, resp.accessToken, resp.code.ID)
	if err != nil {
		h.logger.ErrorContext(r.Context(), "failed to create ID token", "err", err)
		h.tokenErrHelper(w, oauth2.ServerError, "", http.StatusInternalServerError)
		return false
	}
	resp.idToken = idToken
	resp.idTokenExpiry = idTokenExpiry
	return true
}

// scopesCoveredByConsent checks whether the approved scopes cover all requested scopes.
// The openid scope is excluded from the comparison as it is a technical scope
// that does not require user consent.
func scopesCoveredByConsent(approved, requested []string) bool {
	approvedSet := make(map[string]struct{}, len(approved))
	for _, s := range approved {
		approvedSet[s] = struct{}{}
	}

	for _, scope := range requested {
		if scope == tokens.ScopeOpenID {
			continue
		}
		if _, ok := approvedSet[scope]; !ok {
			return false
		}
	}

	return true
}

// tokenErrHelper writes a JSON OAuth2 error response.
func (h *Handler) tokenErrHelper(w http.ResponseWriter, typ string, description string, statusCode int) {
	if err := oauth2.WriteError(w, typ, description, statusCode); err != nil {
		h.logger.Error("token error response", "err", err)
	}
}
