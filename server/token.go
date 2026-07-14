package server

// token.go implements the /token endpoint: grant dispatch plus shared
// token-response and error helpers. Each grant lives in its own grant_*.go.

import (
	"crypto/sha256"
	"crypto/subtle"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"strconv"
	"time"

	"github.com/dexidp/dex/storage"
)

const (
	codeChallengeMethodPlain = "plain"
	codeChallengeMethodS256  = "S256"
)

func (s *Server) withClientFromStorage(w http.ResponseWriter, r *http.Request, handler func(http.ResponseWriter, *http.Request, storage.Client)) {
	ctx := r.Context()
	clientID, clientSecret, ok := r.BasicAuth()
	if ok {
		var err error
		if clientID, err = url.QueryUnescape(clientID); err != nil {
			s.tokenErrHelper(w, errInvalidRequest, "client_id improperly encoded", http.StatusBadRequest)
			return
		}
		if clientSecret, err = url.QueryUnescape(clientSecret); err != nil {
			s.tokenErrHelper(w, errInvalidRequest, "client_secret improperly encoded", http.StatusBadRequest)
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
			s.tokenErrHelper(w, errServerError, "", http.StatusInternalServerError)
		} else {
			s.tokenErrHelper(w, errInvalidClient, "Invalid client credentials.", http.StatusUnauthorized)
		}
		return
	}

	if subtle.ConstantTimeCompare([]byte(client.Secret), []byte(clientSecret)) != 1 {
		if clientSecret == "" {
			s.logger.InfoContext(r.Context(), "missing client_secret on token request", "client_id", client.ID)
		} else {
			s.logger.InfoContext(r.Context(), "invalid client_secret on token request", "client_id", client.ID)
		}
		s.tokenErrHelper(w, errInvalidClient, "Invalid client credentials.", http.StatusUnauthorized)
		return
	}

	handler(w, r, client)
}

func (s *Server) handleToken(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	if r.Method != http.MethodPost {
		s.tokenErrHelper(w, errInvalidRequest, "method not allowed", http.StatusBadRequest)
		return
	}

	err := r.ParseForm()
	if err != nil {
		s.logger.ErrorContext(r.Context(), "could not parse request body", "err", err)
		s.tokenErrHelper(w, errInvalidRequest, "", http.StatusBadRequest)
		return
	}

	grantType := r.PostFormValue("grant_type")
	if !contains(s.supportedGrantTypes, grantType) {
		s.logger.ErrorContext(r.Context(), "unsupported grant type", "grant_type", grantType)
		s.tokenErrHelper(w, errUnsupportedGrantType, "", http.StatusBadRequest)
		return
	}
	switch grantType {
	case grantTypeDeviceCode:
		s.handleDeviceToken(w, r)
	case grantTypeAuthorizationCode:
		s.withClientFromStorage(w, r, s.handleAuthCode)
	case grantTypeRefreshToken:
		s.withClientFromStorage(w, r, s.handleRefreshToken)
	case grantTypePassword:
		s.withClientFromStorage(w, r, s.handlePasswordGrant)
	case grantTypeTokenExchange:
		s.withClientFromStorage(w, r, s.handleTokenExchange)
	case grantTypeClientCredentials:
		s.withClientFromStorage(w, r, s.handleClientCredentialsGrant)
	default:
		s.tokenErrHelper(w, errUnsupportedGrantType, "", http.StatusBadRequest)
	}
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

func (s *Server) writeAccessToken(w http.ResponseWriter, resp *accessTokenResponse) {
	data, err := json.Marshal(resp)
	if err != nil {
		// TODO(nabokihms): error with context
		s.logger.Error("failed to marshal access token response", "err", err)
		s.tokenErrHelper(w, errServerError, "", http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Content-Length", strconv.Itoa(len(data)))

	// Token response must include cache headers https://tools.ietf.org/html/rfc6749#section-5.1
	w.Header().Set("Cache-Control", "no-store")
	w.Header().Set("Pragma", "no-cache")
	w.Write(data)
}

func (s *Server) tokenErrHelper(w http.ResponseWriter, typ string, description string, statusCode int) {
	if err := tokenErr(w, typ, description, statusCode); err != nil {
		// TODO(nabokihms): error with context
		s.logger.Error("token error response", "err", err)
	}
}
