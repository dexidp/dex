package server

import (
	"net/http"

	"github.com/dexidp/dex/pkg/otel/traces"
	"github.com/dexidp/dex/storage"
)

// handle an access token request https://tools.ietf.org/html/rfc6749#section-4.1.3
func (s *Server) handleAuthCode(w http.ResponseWriter, r *http.Request, client storage.Client) {
	ctx, span := traces.InstrumentationTracer(r.Context(), "dex.handleAuthCode")
	defer span.End()

	code := r.PostFormValue("code")
	redirectURI := r.PostFormValue("redirect_uri")

	if code == "" {
		s.tokenErrHelper(ctx, w, errInvalidRequest, `Required param: code.`, http.StatusBadRequest)
		return
	}

	authCode, err := s.storage.GetAuthCode(ctx, code)
	if err != nil || s.now().After(authCode.Expiry) || authCode.ClientID != client.ID {
		if err != storage.ErrNotFound {
			s.logger.ErrorContext(ctx, "failed to get auth code", "err", err)
			s.tokenErrHelper(ctx, w, errServerError, "", http.StatusInternalServerError)
		} else {
			s.tokenErrHelper(ctx, w, errInvalidGrant, "Invalid or expired code parameter.", http.StatusBadRequest)
		}
		return
	}

	// RFC 7636 (PKCE)
	codeChallengeFromStorage := authCode.PKCE.CodeChallenge
	providedCodeVerifier := r.PostFormValue("code_verifier")

	switch {
	case providedCodeVerifier != "" && codeChallengeFromStorage != "":
		calculatedCodeChallenge, err := s.calculateCodeChallenge(providedCodeVerifier, authCode.PKCE.CodeChallengeMethod)
		if err != nil {
			s.logger.ErrorContext(ctx, "failed to calculate code challenge", "err", err)
			s.tokenErrHelper(ctx, w, errServerError, "", http.StatusInternalServerError)
			return
		}
		if codeChallengeFromStorage != calculatedCodeChallenge {
			s.tokenErrHelper(ctx, w, errInvalidGrant, "Invalid code_verifier.", http.StatusBadRequest)
			return
		}
	case providedCodeVerifier != "":
		// Received no code_challenge on /auth, but a code_verifier on /token
		s.tokenErrHelper(ctx, w, errInvalidRequest, "No PKCE flow started. Cannot check code_verifier.", http.StatusBadRequest)
		return
	case codeChallengeFromStorage != "":
		// Received PKCE request on /auth, but no code_verifier on /token
		s.tokenErrHelper(ctx, w, errInvalidGrant, "Expecting parameter code_verifier in PKCE flow.", http.StatusBadRequest)
		return
	}

	if authCode.RedirectURI != redirectURI {
		s.tokenErrHelper(ctx, w, errInvalidRequest, "redirect_uri did not match URI from initial request.", http.StatusBadRequest)
		return
	}

	tokenResponse, err := s.exchangeAuthCode(ctx, w, authCode, client)
	if err != nil {
		s.tokenErrHelper(ctx, w, errServerError, "", http.StatusInternalServerError)
		return
	}
	s.writeAccessToken(ctx, w, tokenResponse)
}
