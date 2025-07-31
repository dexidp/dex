package server

import (
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"crypto/subtle"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"path"
	"sort"
	"strconv"
	"time"

	"github.com/go-jose/go-jose/v4"
	"go.opentelemetry.io/otel/codes"

	"github.com/dexidp/dex/connector"
	"github.com/dexidp/dex/pkg/otel/traces"
	"github.com/dexidp/dex/server/internal"
	"github.com/dexidp/dex/storage"
)

const (
	codeChallengeMethodPlain = "plain"
	codeChallengeMethodS256  = "S256"
)

type discovery struct {
	Issuer            string   `json:"issuer"`
	Auth              string   `json:"authorization_endpoint"`
	Token             string   `json:"token_endpoint"`
	Keys              string   `json:"jwks_uri"`
	UserInfo          string   `json:"userinfo_endpoint"`
	DeviceEndpoint    string   `json:"device_authorization_endpoint"`
	Introspect        string   `json:"introspection_endpoint"`
	GrantTypes        []string `json:"grant_types_supported"`
	ResponseTypes     []string `json:"response_types_supported"`
	Subjects          []string `json:"subject_types_supported"`
	IDTokenAlgs       []string `json:"id_token_signing_alg_values_supported"`
	CodeChallengeAlgs []string `json:"code_challenge_methods_supported"`
	Scopes            []string `json:"scopes_supported"`
	AuthMethods       []string `json:"token_endpoint_auth_methods_supported"`
	Claims            []string `json:"claims_supported"`
}

func (s *Server) constructDiscovery() discovery {
	d := discovery{
		Issuer:            s.issuerURL.String(),
		Auth:              s.absURL("/auth"),
		Token:             s.absURL("/token"),
		Keys:              s.absURL("/keys"),
		UserInfo:          s.absURL("/userinfo"),
		DeviceEndpoint:    s.absURL("/device/code"),
		Introspect:        s.absURL("/token/introspect"),
		Subjects:          []string{"public"},
		IDTokenAlgs:       []string{string(jose.RS256)},
		CodeChallengeAlgs: []string{codeChallengeMethodS256, codeChallengeMethodPlain},
		Scopes:            []string{"openid", "email", "groups", "profile", "offline_access"},
		AuthMethods:       []string{"client_secret_basic", "client_secret_post"},
		Claims: []string{
			"iss", "sub", "aud", "iat", "exp", "email", "email_verified",
			"locale", "name", "preferred_username", "at_hash",
		},
	}

	for responseType := range s.supportedResponseTypes {
		d.ResponseTypes = append(d.ResponseTypes, responseType)
	}
	sort.Strings(d.ResponseTypes)

	d.GrantTypes = s.supportedGrantTypes
	return d
}

// finalizeLogin associates the user's identity with the current AuthRequest, then returns
// the approval page's path.
func (s *Server) finalizeLogin(ctx context.Context, identity connector.Identity, authReq storage.AuthRequest, conn connector.Connector) (string, bool, error) {
	ctx, span := traces.InstrumentationTracer(ctx, "dex.server.finalizeLogin")
	defer span.End()
	claims := storage.Claims{
		UserID:            identity.UserID,
		Username:          identity.Username,
		PreferredUsername: identity.PreferredUsername,
		Email:             identity.Email,
		EmailVerified:     identity.EmailVerified,
		Groups:            identity.Groups,
	}
	updater := func(a storage.AuthRequest) (storage.AuthRequest, error) {
		a.LoggedIn = true
		a.Claims = claims
		a.ConnectorData = identity.ConnectorData
		return a, nil
	}
	if err := s.storage.UpdateAuthRequest(ctx, authReq.ID, updater); err != nil {
		return "", false, fmt.Errorf("failed to update auth request: %v", err)
	}

	email := claims.Email
	if !claims.EmailVerified {
		email += " (unverified)"
	}

	s.logger.InfoContext(ctx, "login successful",
		"connector_id", authReq.ConnectorID, "username", claims.Username,
		"preferred_username", claims.PreferredUsername, "email", email, "groups", claims.Groups)

	offlineAccessRequested := false
	for _, scope := range authReq.Scopes {
		if scope == scopeOfflineAccess {
			offlineAccessRequested = true
			break
		}
	}
	_, canRefresh := conn.(connector.RefreshConnector)

	if offlineAccessRequested && canRefresh {
		// Try to retrieve an existing OfflineSession object for the corresponding user.
		session, err := s.storage.GetOfflineSessions(ctx, identity.UserID, authReq.ConnectorID)
		switch {
		case err != nil && err == storage.ErrNotFound:
			offlineSessions := storage.OfflineSessions{
				UserID:        identity.UserID,
				ConnID:        authReq.ConnectorID,
				Refresh:       make(map[string]*storage.RefreshTokenRef),
				ConnectorData: identity.ConnectorData,
			}

			// Create a new OfflineSession object for the user and add a reference object for
			// the newly received refreshtoken.
			if err := s.storage.CreateOfflineSessions(ctx, offlineSessions); err != nil {
				s.logger.ErrorContext(ctx, "failed to create offline session", "err", err)
				return "", false, err
			}
		case err == nil:
			// Update existing OfflineSession obj with new RefreshTokenRef.
			if err := s.storage.UpdateOfflineSessions(ctx, session.UserID, session.ConnID, func(old storage.OfflineSessions) (storage.OfflineSessions, error) {
				if len(identity.ConnectorData) > 0 {
					old.ConnectorData = identity.ConnectorData
				}
				return old, nil
			}); err != nil {
				s.logger.ErrorContext(ctx, "failed to update offline session", "err", err)
				return "", false, err
			}
		default:
			s.logger.ErrorContext(ctx, "failed to get offline session", "err", err)
			return "", false, err
		}
	}

	// we can skip the redirect to /approval and go ahead and send code if it's not required
	if s.skipApproval && !authReq.ForceApprovalPrompt {
		return "", true, nil
	}

	// an HMAC is used here to ensure that the request ID is unpredictable, ensuring that an attacker who intercepted the original
	// flow would be unable to poll for the result at the /approval endpoint
	h := hmac.New(sha256.New, authReq.HMACKey)
	h.Write([]byte(authReq.ID))
	mac := h.Sum(nil)

	returnURL := path.Join(s.issuerURL.Path, "/approval") + "?req=" + authReq.ID + "&hmac=" + base64.RawURLEncoding.EncodeToString(mac)
	return returnURL, false, nil
}

func (s *Server) sendCodeResponse(w http.ResponseWriter, r *http.Request, authReq storage.AuthRequest) {
	ctx, span := traces.InstrumentationTracer(r.Context(), "dex.server.send_code_response")
	defer span.End()

	if s.now().After(authReq.Expiry) {
		s.renderError(r, w, http.StatusBadRequest, "User session has expired.")
		return
	}

	if err := s.storage.DeleteAuthRequest(ctx, authReq.ID); err != nil {
		if err != storage.ErrNotFound {
			s.logger.ErrorContext(ctx, "Failed to delete authorization request", "err", err)
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
			}
			if err := s.storage.CreateAuthCode(ctx, code); err != nil {
				s.logger.ErrorContext(ctx, "Failed to create auth code", "err", err)
				s.renderError(r, w, http.StatusInternalServerError, "Internal server error.")
				return
			}

			// Implicit and hybrid flows that try to use the OOB redirect URI are
			// rejected earlier. If we got here we're using the code flow.
			if authReq.RedirectURI == redirectURIOOB {
				if err := s.templates.oob(r, w, code.ID); err != nil {
					s.logger.ErrorContext(ctx, "server template error", "err", err)
				}
				return
			}
		case responseTypeToken:
			implicitOrHybrid = true
			var err error

			accessToken, _, err = s.newAccessToken(ctx, authReq.ClientID, authReq.Claims, authReq.Scopes, authReq.Nonce, authReq.ConnectorID)
			if err != nil {
				s.logger.ErrorContext(ctx, "failed to create new access token", "err", err)
				s.tokenErrHelper(ctx, w, errServerError, "", http.StatusInternalServerError)
				return
			}
		case responseTypeIDToken:
			implicitOrHybrid = true
			var err error

			idToken, idTokenExpiry, err = s.newIDToken(ctx, authReq.ClientID, authReq.Claims, authReq.Scopes, authReq.Nonce, accessToken, code.ID, authReq.ConnectorID)
			if err != nil {
				s.logger.ErrorContext(ctx, "failed to create ID token", "err", err)
				s.tokenErrHelper(ctx, w, errServerError, "", http.StatusInternalServerError)
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

func (s *Server) withClientFromStorage(w http.ResponseWriter, r *http.Request, handler func(http.ResponseWriter, *http.Request, storage.Client)) {
	ctx, span := traces.InstrumentationTracer(r.Context(), "dex.server.with_client_from_storage")
	defer span.End()
	clientID, clientSecret, ok := r.BasicAuth()
	if ok {
		var err error
		if clientID, err = url.QueryUnescape(clientID); err != nil {
			s.tokenErrHelper(ctx, w, errInvalidRequest, "client_id improperly encoded", http.StatusBadRequest)
			return
		}
		if clientSecret, err = url.QueryUnescape(clientSecret); err != nil {
			s.tokenErrHelper(ctx, w, errInvalidRequest, "client_secret improperly encoded", http.StatusBadRequest)
			return
		}
	} else {
		clientID = r.PostFormValue("client_id")
		clientSecret = r.PostFormValue("client_secret")
	}

	client, err := s.storage.GetClient(ctx, clientID)
	if err != nil {
		if err != storage.ErrNotFound {
			s.logger.ErrorContext(ctx, "failed to get client", "err", err)
			s.tokenErrHelper(ctx, w, errServerError, "", http.StatusInternalServerError)
		} else {
			s.tokenErrHelper(ctx, w, errInvalidClient, "Invalid client credentials.", http.StatusUnauthorized)
		}
		return
	}

	if subtle.ConstantTimeCompare([]byte(client.Secret), []byte(clientSecret)) != 1 {
		if clientSecret == "" {
			s.logger.InfoContext(ctx, "missing client_secret on token request", "client_id", client.ID)
		} else {
			s.logger.InfoContext(ctx, "invalid client_secret on token request", "client_id", client.ID)
		}
		s.tokenErrHelper(ctx, w, errInvalidClient, "Invalid client credentials.", http.StatusUnauthorized)
		return
	}

	handler(w, r, client)
}

func (s *Server) calculateCodeChallenge(codeVerifier, codeChallengeMethod string) (string, error) {
	switch codeChallengeMethod {
	case codeChallengeMethodPlain:
		return codeVerifier, nil
	case codeChallengeMethodS256:
		shaSum := sha256.Sum256([]byte(codeVerifier))
		return base64.RawURLEncoding.EncodeToString(shaSum[:]), nil
	default:
		return "", fmt.Errorf("unknown challenge method (%v)", codeChallengeMethod)
	}
}

func (s *Server) exchangeAuthCode(ctx context.Context, w http.ResponseWriter, authCode storage.AuthCode, client storage.Client) (*accessTokenResponse, error) {
	ctx, span := traces.InstrumentationTracer(ctx, "dex.server.exchange_auth_code")
	defer span.End()
	accessToken, _, err := s.newAccessToken(ctx, client.ID, authCode.Claims, authCode.Scopes, authCode.Nonce, authCode.ConnectorID)
	if err != nil {
		s.logger.ErrorContext(ctx, "failed to create new access token", "err", err)
		s.tokenErrHelper(ctx, w, errServerError, "", http.StatusInternalServerError)
		return nil, err
	}

	idToken, expiry, err := s.newIDToken(ctx, client.ID, authCode.Claims, authCode.Scopes, authCode.Nonce, accessToken, authCode.ID, authCode.ConnectorID)
	if err != nil {
		s.logger.ErrorContext(ctx, "failed to create ID token", "err", err)
		s.tokenErrHelper(ctx, w, errServerError, "", http.StatusInternalServerError)
		return nil, err
	}

	if err := s.storage.DeleteAuthCode(ctx, authCode.ID); err != nil {
		s.logger.ErrorContext(ctx, "failed to delete auth code", "err", err)
		s.tokenErrHelper(ctx, w, errServerError, "", http.StatusInternalServerError)
		return nil, err
	}

	reqRefresh := func() bool {
		// Ensure the connector supports refresh tokens.
		//
		// Connectors like `saml` do not implement RefreshConnector.
		conn, err := s.getConnector(ctx, authCode.ConnectorID)
		if err != nil {
			s.logger.ErrorContext(ctx, "connector not found", "connector_id", authCode.ConnectorID, "err", err)
			s.tokenErrHelper(ctx, w, errServerError, "", http.StatusInternalServerError)
			return false
		}

		_, ok := conn.Connector.(connector.RefreshConnector)
		if !ok {
			return false
		}

		for _, scope := range authCode.Scopes {
			if scope == scopeOfflineAccess {
				return true
			}
		}
		return false
	}()
	var refreshToken string
	if reqRefresh {
		refresh := storage.RefreshToken{
			ID:            storage.NewID(),
			Token:         storage.NewID(),
			ClientID:      authCode.ClientID,
			ConnectorID:   authCode.ConnectorID,
			Scopes:        authCode.Scopes,
			Claims:        authCode.Claims,
			Nonce:         authCode.Nonce,
			ConnectorData: authCode.ConnectorData,
			CreatedAt:     s.now(),
			LastUsed:      s.now(),
		}
		token := &internal.RefreshToken{
			RefreshId: refresh.ID,
			Token:     refresh.Token,
		}
		if refreshToken, err = internal.Marshal(token); err != nil {
			s.logger.ErrorContext(ctx, "failed to marshal refresh token", "err", err)
			s.tokenErrHelper(ctx, w, errServerError, "", http.StatusInternalServerError)
			return nil, err
		}

		if err := s.storage.CreateRefresh(ctx, refresh); err != nil {
			s.logger.ErrorContext(ctx, "failed to create refresh token", "err", err)
			s.tokenErrHelper(ctx, w, errServerError, "", http.StatusInternalServerError)
			return nil, err
		}

		// deleteToken determines if we need to delete the newly created refresh token
		// due to a failure in updating/creating the OfflineSession object for the
		// corresponding user.
		var deleteToken bool
		defer func() {
			if deleteToken {
				// Delete newly created refresh token from storage.
				if err := s.storage.DeleteRefresh(ctx, refresh.ID); err != nil {
					s.logger.ErrorContext(ctx, "failed to delete refresh token", "err", err)
					s.tokenErrHelper(ctx, w, errServerError, "", http.StatusInternalServerError)
					return
				}
			}
		}()

		tokenRef := storage.RefreshTokenRef{
			ID:        refresh.ID,
			ClientID:  refresh.ClientID,
			CreatedAt: refresh.CreatedAt,
			LastUsed:  refresh.LastUsed,
		}

		// Try to retrieve an existing OfflineSession object for the corresponding user.
		if session, err := s.storage.GetOfflineSessions(ctx, refresh.Claims.UserID, refresh.ConnectorID); err != nil {
			if err != storage.ErrNotFound {
				s.logger.ErrorContext(ctx, "failed to get offline session", "err", err)
				s.tokenErrHelper(ctx, w, errServerError, "", http.StatusInternalServerError)
				deleteToken = true
				return nil, err
			}
			offlineSessions := storage.OfflineSessions{
				UserID:  refresh.Claims.UserID,
				ConnID:  refresh.ConnectorID,
				Refresh: make(map[string]*storage.RefreshTokenRef),
			}
			offlineSessions.Refresh[tokenRef.ClientID] = &tokenRef

			// Create a new OfflineSession object for the user and add a reference object for
			// the newly received refreshtoken.
			if err := s.storage.CreateOfflineSessions(ctx, offlineSessions); err != nil {
				s.logger.ErrorContext(ctx, "failed to create offline session", "err", err)
				s.tokenErrHelper(ctx, w, errServerError, "", http.StatusInternalServerError)
				deleteToken = true
				return nil, err
			}
		} else {
			if oldTokenRef, ok := session.Refresh[tokenRef.ClientID]; ok {
				// Delete old refresh token from storage.
				if err := s.storage.DeleteRefresh(ctx, oldTokenRef.ID); err != nil && err != storage.ErrNotFound {
					s.logger.ErrorContext(ctx, "failed to delete refresh token", "err", err)
					s.tokenErrHelper(ctx, w, errServerError, "", http.StatusInternalServerError)
					deleteToken = true
					return nil, err
				}
			}

			// Update existing OfflineSession obj with new RefreshTokenRef.
			if err := s.storage.UpdateOfflineSessions(ctx, session.UserID, session.ConnID, func(old storage.OfflineSessions) (storage.OfflineSessions, error) {
				old.Refresh[tokenRef.ClientID] = &tokenRef
				return old, nil
			}); err != nil {
				s.logger.ErrorContext(ctx, "failed to update offline session", "err", err)
				s.tokenErrHelper(ctx, w, errServerError, "", http.StatusInternalServerError)
				deleteToken = true
				return nil, err
			}
		}
	}
	return s.toAccessTokenResponse(idToken, accessToken, refreshToken, expiry), nil
}

type accessTokenResponse struct {
	AccessToken     string `json:"access_token"`
	IssuedTokenType string `json:"issued_token_type,omitempty"`
	TokenType       string `json:"token_type"`
	ExpiresIn       int    `json:"expires_in,omitempty"`
	RefreshToken    string `json:"refresh_token,omitempty"`
	IDToken         string `json:"id_token,omitempty"`
	Scope           string `json:"scope,omitempty"`
}

func (s *Server) toAccessTokenResponse(idToken, accessToken, refreshToken string, expiry time.Time) *accessTokenResponse {
	return &accessTokenResponse{
		AccessToken:  accessToken,
		TokenType:    "bearer",
		ExpiresIn:    int(expiry.Sub(s.now()).Seconds()),
		RefreshToken: refreshToken,
		IDToken:      idToken,
	}
}

func (s *Server) writeAccessToken(ctx context.Context, w http.ResponseWriter, resp *accessTokenResponse) {
	ctx, span := traces.InstrumentationTracer(ctx, "dex.server.write_access_token")
	defer span.End()
	data, err := json.Marshal(resp)
	if err != nil {
		s.logger.ErrorContext(ctx, "failed to marshal access token response", "err", err)
		s.tokenErrHelper(ctx, w, errServerError, "", http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Content-Length", strconv.Itoa(len(data)))

	// Token response must include cache headers https://tools.ietf.org/html/rfc6749#section-5.1
	w.Header().Set("Cache-Control", "no-store")
	w.Header().Set("Pragma", "no-cache")
	w.Write(data)
}

func (s *Server) renderError(r *http.Request, w http.ResponseWriter, status int, description string) {
	ctx, span := traces.InstrumentationTracer(r.Context(), "dex.server.render_error")
	defer span.End()
	if err := s.templates.err(r, w, status, description); err != nil {
		s.logger.ErrorContext(ctx, "server template error", "err", err)
	}
}

func (s *Server) tokenErrHelper(ctx context.Context, w http.ResponseWriter, typ string, description string, statusCode int) {
	ctx, span := traces.InstrumentationTracer(ctx, "dex.server.token_err_helper")
	defer span.End()
	if statusCode >= 500 {
		span.SetStatus(codes.Error, "failed to write token error response")
	}
	if err := tokenErr(w, typ, description, statusCode); err != nil {
		s.logger.ErrorContext(ctx, "token error response", "err", err)
	}
}

// Check for username prompt override from connector. Defaults to "Username".
func usernamePrompt(conn connector.PasswordConnector) string {
	if attr := conn.Prompt(); attr != "" {
		return attr
	}
	return "Username"
}
