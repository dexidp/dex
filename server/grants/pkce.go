package grants

import (
	"net/http"

	"github.com/dexidp/dex/server/oauth2"
	"github.com/dexidp/dex/storage"
)

// verifyPKCE checks a code_verifier against a stored PKCE challenge (RFC 7636).
// It is the single PKCE check, shared by the authorization_code and device_code
// grants, which redeem a code challenge stored at /auth.
func verifyPKCE(codeVerifier string, pkce storage.PKCE) *oauth2.Error {
	switch {
	case codeVerifier != "" && pkce.CodeChallenge != "":
		calculated, err := oauth2.CalculateCodeChallenge(codeVerifier, pkce.CodeChallengeMethod)
		if err != nil {
			return &oauth2.Error{Type: oauth2.ServerError, Status: http.StatusInternalServerError}
		}
		if pkce.CodeChallenge != calculated {
			return &oauth2.Error{Type: oauth2.InvalidGrant, Description: "Invalid code_verifier.", Status: http.StatusBadRequest}
		}
	case codeVerifier != "":
		// No code_challenge on /auth, but a code_verifier on /token.
		return &oauth2.Error{Type: oauth2.InvalidRequest, Description: "No PKCE flow started. Cannot check code_verifier.", Status: http.StatusBadRequest}
	case pkce.CodeChallenge != "":
		// PKCE started on /auth, but no code_verifier on /token.
		return &oauth2.Error{Type: oauth2.InvalidGrant, Description: "Expecting parameter code_verifier in PKCE flow.", Status: http.StatusBadRequest}
	}
	return nil
}
