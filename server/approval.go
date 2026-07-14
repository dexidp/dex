package server

// approval.go handles the consent/approval step and builds the authorization-code
// or token response back to the client.

import (
	"net/http"
	"net/url"
	"strconv"
	"time"

	"github.com/dexidp/dex/pkg/featureflags"
	"github.com/dexidp/dex/storage"
)

func (s *Server) handleApproval(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	macEncoded := r.FormValue("hmac")
	if macEncoded == "" {
		s.renderError(r, w, http.StatusUnauthorized, "Unauthorized request")
		return
	}
	authReq, err := s.storage.GetAuthRequest(ctx, r.FormValue("req"))
	if err != nil {
		if err == storage.ErrNotFound {
			s.renderError(r, w, http.StatusBadRequest, "User session error.")
			return
		}
		s.logger.ErrorContext(r.Context(), "failed to get auth request", "err", err)
		s.renderError(r, w, http.StatusInternalServerError, "Database error.")
		return
	}
	if !authReq.LoggedIn {
		s.logger.ErrorContext(r.Context(), "auth request does not have an identity for approval")
		s.renderError(r, w, http.StatusInternalServerError, "Login process not yet finalized.")
		return
	}

	if !authReq.MFAValidated {
		// Check if MFA is actually required — if so, redirect to TOTP instead of blocking.
		// This handles the case where MFA was enabled after the auth flow started.
		mfaChain, err := s.mfaChainForClient(ctx, authReq.ClientID, authReq.ConnectorID)
		if err != nil {
			s.logger.ErrorContext(ctx, "failed to get MFA chain", "err", err)
			s.renderError(r, w, http.StatusInternalServerError, "Internal server error.")
			return
		}
		if len(mfaChain) > 0 {
			http.Redirect(w, r, s.buildMFARedirectURL(authReq, mfaChain[0]), http.StatusSeeOther)
			return
		}
		// No MFA required but flag not set — allow through (backward compat).
	}

	if !verifyHMAC(authReq.HMACKey, macEncoded, authReq.ID, "") {
		s.renderError(r, w, http.StatusUnauthorized, "Unauthorized request")
		return
	}

	switch r.Method {
	case http.MethodGet:
		// Skip the approval page and issue the code directly if:
		// 1. The client didn't force the approval prompt, AND
		// 2. Either the server is configured to skip approval globally,
		//    or the user has already consented to all requested scopes for this client.
		// This handles the MFA redirect case: after MFA completion the user lands on
		// /approval via GET, and we don't want to show the consent screen again.
		if !authReq.ForceApprovalPrompt {
			if s.skipApproval {
				s.sendCodeResponse(w, r, authReq)
				return
			}
			ui, err := s.storage.GetUserIdentity(ctx, authReq.Claims.UserID, authReq.ConnectorID)
			if err == nil && scopesCoveredByConsent(ui.Consents[authReq.ClientID], authReq.Scopes) {
				s.sendCodeResponse(w, r, authReq)
				return
			}
		}

		client, err := s.storage.GetClient(ctx, authReq.ClientID)
		if err != nil {
			s.logger.ErrorContext(r.Context(), "Failed to get client", "client_id", authReq.ClientID, "err", err)
			s.renderError(r, w, http.StatusInternalServerError, "Failed to retrieve client.")
			return
		}
		if err := s.templates.approval(r, w, authReq.ID, authReq.Claims.Username, client.Name, authReq.Scopes); err != nil {
			s.logger.ErrorContext(r.Context(), "server template error", "err", err)
		}
	case http.MethodPost:
		if r.FormValue("approval") != "approve" {
			s.renderError(r, w, http.StatusInternalServerError, "Approval rejected.")
			return
		}
		// Persist user-approved scopes as consent for this client.
		if featureflags.SessionsEnabled.Enabled() {
			if err := s.storage.UpdateUserIdentity(ctx, authReq.Claims.UserID, authReq.ConnectorID, func(old storage.UserIdentity) (storage.UserIdentity, error) {
				if old.Consents == nil {
					old.Consents = make(map[string][]string)
				}
				old.Consents[authReq.ClientID] = authReq.Scopes
				return old, nil
			}); err != nil {
				s.logger.ErrorContext(ctx, "failed to update user identity consents", "err", err)
			}
		}
		s.sendCodeResponse(w, r, authReq)
	}
}

func (s *Server) sendCodeResponse(w http.ResponseWriter, r *http.Request, authReq storage.AuthRequest) {
	s.updateSessionTokenIssuedAt(r, authReq.ClientID)

	ctx := r.Context()
	if s.now().After(authReq.Expiry) {
		s.renderError(r, w, http.StatusBadRequest, "User session has expired.")
		return
	}

	if err := s.storage.DeleteAuthRequest(ctx, authReq.ID); err != nil {
		if err != storage.ErrNotFound {
			s.logger.ErrorContext(r.Context(), "Failed to delete authorization request", "err", err)
			s.renderError(r, w, http.StatusInternalServerError, "Internal server error.")
		} else {
			s.renderError(r, w, http.StatusBadRequest, "User session error.")
		}
		return
	}
	u, err := url.Parse(authReq.RedirectURI)
	if err != nil {
		s.renderError(r, w, http.StatusInternalServerError, "Invalid redirect URI.")
		return
	}

	var (
		// Was the initial request using the implicit or hybrid flow instead of
		// the "normal" code flow?
		implicitOrHybrid = false

		// Only present in hybrid or code flow. code.ID == "" if this is not set.
		code storage.AuthCode

		// ID token returned immediately if the response_type includes "id_token".
		// Only valid for implicit and hybrid flows.
		idToken       string
		idTokenExpiry time.Time

		// Access token
		accessToken string
	)

	for _, responseType := range authReq.ResponseTypes {
		switch responseType {
		case responseTypeCode:
			code = storage.AuthCode{
				ID:            storage.NewID(),
				ClientID:      authReq.ClientID,
				ConnectorID:   authReq.ConnectorID,
				Nonce:         authReq.Nonce,
				Scopes:        authReq.Scopes,
				Claims:        authReq.Claims,
				Expiry:        s.now().Add(time.Minute * 30),
				RedirectURI:   authReq.RedirectURI,
				ConnectorData: authReq.ConnectorData,
				PKCE:          authReq.PKCE,
				AuthTime:      authReq.AuthTime,
			}
			if err := s.storage.CreateAuthCode(ctx, code); err != nil {
				s.logger.ErrorContext(r.Context(), "Failed to create auth code", "err", err)
				s.renderError(r, w, http.StatusInternalServerError, "Internal server error.")
				return
			}

			// Implicit and hybrid flows that try to use the OOB redirect URI are
			// rejected earlier. If we got here we're using the code flow.
			if authReq.RedirectURI == redirectURIOOB {
				if err := s.templates.oob(r, w, code.ID); err != nil {
					s.logger.ErrorContext(r.Context(), "server template error", "err", err)
				}
				return
			}
		case responseTypeToken:
			implicitOrHybrid = true
			var err error

			accessToken, _, err = s.newAccessToken(r.Context(), authReq.ClientID, authReq.Claims, authReq.Scopes, authReq.Nonce, authReq.ConnectorID, authReq.AuthTime)
			if err != nil {
				s.logger.ErrorContext(r.Context(), "failed to create new access token", "err", err)
				s.tokenErrHelper(w, errServerError, "", http.StatusInternalServerError)
				return
			}
		case responseTypeIDToken:
			implicitOrHybrid = true
			var err error

			idToken, idTokenExpiry, err = s.newIDToken(r.Context(), authReq.ClientID, authReq.Claims, authReq.Scopes, authReq.Nonce, accessToken, code.ID, authReq.ConnectorID, authReq.AuthTime)
			if err != nil {
				s.logger.ErrorContext(r.Context(), "failed to create ID token", "err", err)
				s.tokenErrHelper(w, errServerError, "", http.StatusInternalServerError)
				return
			}
		}
	}

	if implicitOrHybrid {
		v := url.Values{}
		if accessToken != "" {
			v.Set("access_token", accessToken)
			v.Set("token_type", "bearer")
			// The hybrid flow with "code token" or "code id_token token" doesn't return an
			// "expires_in" value. If "code" wasn't provided, indicating the implicit flow,
			// don't add it.
			//
			// https://openid.net/specs/openid-connect-core-1_0.html#HybridAuthResponse
			if code.ID == "" {
				v.Set("expires_in", strconv.Itoa(int(idTokenExpiry.Sub(s.now()).Seconds())))
			}
		}
		v.Set("state", authReq.State)
		if idToken != "" {
			v.Set("id_token", idToken)
		}
		if code.ID != "" {
			v.Set("code", code.ID)
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
		q.Set("code", code.ID)
		q.Set("state", authReq.State)
		u.RawQuery = q.Encode()
	}

	http.Redirect(w, r, u.String(), http.StatusSeeOther)
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
		if scope == scopeOpenID {
			continue
		}
		if _, ok := approvedSet[scope]; !ok {
			return false
		}
	}

	return true
}
