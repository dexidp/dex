package server

// token.go implements the /token endpoint: grant dispatch plus shared
// token-response and error helpers. Each grant lives in its own grant_*.go.

import (
	"crypto/sha256"
	"crypto/subtle"
	"encoding/base64"
	"fmt"
	"net/http"
	"net/url"
	"time"

	"github.com/dexidp/dex/server/oauth2"
	"github.com/dexidp/dex/server/tokens"
	"github.com/dexidp/dex/storage"
)

func (s *Server) withClientFromStorage(w http.ResponseWriter, r *http.Request, handler func(http.ResponseWriter, *http.Request, storage.Client)) {
	ctx := r.Context()
	clientID, clientSecret, ok := r.BasicAuth()
	if ok {
		var err error
		if clientID, err = url.QueryUnescape(clientID); err != nil {
			s.tokenErrHelper(w, oauth2.InvalidRequest, "client_id improperly encoded", http.StatusBadRequest)
			return
		}
		if clientSecret, err = url.QueryUnescape(clientSecret); err != nil {
			s.tokenErrHelper(w, oauth2.InvalidRequest, "client_secret improperly encoded", http.StatusBadRequest)
			return
		}
	} else {
		clientID = r.PostFormValue("client_id")
		clientSecret = r.PostFormValue("client_secret")
	}

	client, err := s.storage.GetClient(ctx, clientID)
	if err != nil {
		if err != storage.ErrNotFound {
			s.logger.ErrorContext(r.Context(), "failed to get client", "err", err)
			s.tokenErrHelper(w, oauth2.ServerError, "", http.StatusInternalServerError)
		} else {
			s.tokenErrHelper(w, oauth2.InvalidClient, "Invalid client credentials.", http.StatusUnauthorized)
		}
		return
	}

	if subtle.ConstantTimeCompare([]byte(client.Secret), []byte(clientSecret)) != 1 {
		if clientSecret == "" {
			s.logger.InfoContext(r.Context(), "missing client_secret on token request", "client_id", client.ID)
		} else {
			s.logger.InfoContext(r.Context(), "invalid client_secret on token request", "client_id", client.ID)
		}
		s.tokenErrHelper(w, oauth2.InvalidClient, "Invalid client credentials.", http.StatusUnauthorized)
		return
	}

	handler(w, r, client)
}

func (s *Server) handleToken(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	if r.Method != http.MethodPost {
		s.tokenErrHelper(w, oauth2.InvalidRequest, "method not allowed", http.StatusBadRequest)
		return
	}

	err := r.ParseForm()
	if err != nil {
		s.logger.ErrorContext(r.Context(), "could not parse request body", "err", err)
		s.tokenErrHelper(w, oauth2.InvalidRequest, "", http.StatusBadRequest)
		return
	}

	grantType := r.PostFormValue("grant_type")
	if !contains(s.supportedGrantTypes, grantType) {
		s.logger.ErrorContext(r.Context(), "unsupported grant type", "grant_type", grantType)
		s.tokenErrHelper(w, oauth2.UnsupportedGrantType, "", http.StatusBadRequest)
		return
	}
	switch grantType {
	case oauth2.GrantTypeDeviceCode:
		s.newDeviceHandler().HandleToken(w, r)
	case oauth2.GrantTypeAuthorizationCode:
		s.withClientFromStorage(w, r, s.handleAuthCode)
	case oauth2.GrantTypeRefreshToken:
		s.withClientFromStorage(w, r, s.handleRefreshToken)
	case oauth2.GrantTypePassword:
		s.withClientFromStorage(w, r, s.handlePasswordGrant)
	case oauth2.GrantTypeTokenExchange:
		s.withClientFromStorage(w, r, s.handleTokenExchange)
	case oauth2.GrantTypeClientCredentials:
		s.withClientFromStorage(w, r, s.handleClientCredentialsGrant)
	default:
		s.tokenErrHelper(w, oauth2.UnsupportedGrantType, "", http.StatusBadRequest)
	}
}

func (s *Server) calculateCodeChallenge(codeVerifier, codeChallengeMethod string) (string, error) {
	switch codeChallengeMethod {
	case oauth2.PKCEMethodPlain:
		return codeVerifier, nil
	case oauth2.PKCEMethodS256:
		shaSum := sha256.Sum256([]byte(codeVerifier))
		return base64.RawURLEncoding.EncodeToString(shaSum[:]), nil
	default:
		return "", fmt.Errorf("unknown challenge method (%v)", codeChallengeMethod)
	}
}

func (s *Server) toAccessTokenResponse(idToken, accessToken, refreshToken string, expiry time.Time) tokens.Response {
	ts := tokens.TokenSet{AccessToken: accessToken, IDToken: idToken, RefreshToken: refreshToken, Expiry: expiry}
	return ts.Response(s.now())
}

func (s *Server) writeAccessToken(w http.ResponseWriter, resp tokens.Response) {
	if err := resp.Write(w); err != nil {
		// TODO(nabokihms): error with context
		s.logger.Error("failed to write access token response", "err", err)
		s.tokenErrHelper(w, oauth2.ServerError, "", http.StatusInternalServerError)
	}
}

func (s *Server) tokenErrHelper(w http.ResponseWriter, typ string, description string, statusCode int) {
	if err := oauth2.WriteError(w, typ, description, statusCode); err != nil {
		// TODO(nabokihms): error with context
		s.logger.Error("token error response", "err", err)
	}
}

// writeTokenResponse writes a tokens.TokenSet as an OAuth2 token response.
func writeTokenResponse(w http.ResponseWriter, ts tokens.TokenSet, now time.Time) error {
	return ts.Response(now).Write(w)
}
