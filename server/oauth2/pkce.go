package oauth2

import (
	"crypto/sha256"
	"encoding/base64"
	"fmt"
)

// CalculateCodeChallenge derives the PKCE code challenge from a code verifier for
// the given method (RFC 7636). It is shared by every grant that verifies PKCE.
func CalculateCodeChallenge(codeVerifier, codeChallengeMethod string) (string, error) {
	switch codeChallengeMethod {
	case PKCEMethodPlain:
		return codeVerifier, nil
	case PKCEMethodS256:
		shaSum := sha256.Sum256([]byte(codeVerifier))
		return base64.RawURLEncoding.EncodeToString(shaSum[:]), nil
	default:
		return "", fmt.Errorf("unknown challenge method (%v)", codeChallengeMethod)
	}
}
