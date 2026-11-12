package tokens

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/coreos/go-oidc/v3/oidc"
	"github.com/google/uuid"

	"github.com/dexidp/dex/server/oauth2"
	"github.com/dexidp/dex/server/signer"
)

// IDJAGClaims is the payload of an Identity Assertion JWT Authorization Grant
// (draft-ietf-oauth-identity-assertion-authz-grant §3.1). The draft allows
// exactly one audience, so aud is a string.
type IDJAGClaims struct {
	Issuer   string `json:"iss"`
	Subject  string `json:"sub"`
	Audience string `json:"aud"`
	ClientID string `json:"client_id"`
	JWTID    string `json:"jti"`
	Expiry   int64  `json:"exp"`
	IssuedAt int64  `json:"iat"`

	Resource string `json:"resource,omitempty"`
	Scope    string `json:"scope,omitempty"`
	Email    string `json:"email,omitempty"`
}

// SignIDJAG fills in iss, jti, iat and exp and signs the claims as an ID-JAG.
func (i *Issuer) SignIDJAG(ctx context.Context, claims IDJAGClaims, validFor time.Duration) (string, IDJAGClaims, error) {
	issuedAt := i.now()
	claims.Issuer = i.issuerURL.String()
	claims.JWTID = uuid.New().String()
	claims.IssuedAt = issuedAt.Unix()
	claims.Expiry = issuedAt.Add(validFor).Unix()

	payload, err := json.Marshal(claims)
	if err != nil {
		return "", claims, fmt.Errorf("could not serialize claims: %v", err)
	}
	token, err := i.signer.SignWithType(ctx, payload, oauth2.JWTTypeIDJAG)
	if err != nil {
		return "", claims, fmt.Errorf("failed to sign payload: %v", err)
	}
	return token, claims, nil
}

// VerifyIDToken verifies an ID token this issuer signed and issued to clientID.
// Dex access tokens are ID tokens with a random at_hash, so they pass here too.
func (i *Issuer) VerifyIDToken(ctx context.Context, rawIDToken, clientID string) (*oidc.IDToken, error) {
	verifier := oidc.NewVerifier(i.issuerURL.String(), &signer.KeySet{Signer: i.signer}, &oidc.Config{
		ClientID:             clientID,
		Now:                  i.now,
		SupportedSigningAlgs: []string{oidc.RS256, oidc.RS384, oidc.RS512, oidc.ES256, oidc.ES384, oidc.ES512},
	})
	idToken, err := verifier.Verify(ctx, rawIDToken)
	if err != nil {
		return nil, err
	}
	var claims struct {
		AuthorizingParty string          `json:"azp"`
		Events           json.RawMessage `json:"events"`
	}
	if err := idToken.Claims(&claims); err != nil {
		return nil, err
	}
	if claims.Events != nil {
		return nil, fmt.Errorf("token is a logout token")
	}
	// A token with several audiences was issued to its azp, not to every audience.
	if issuedTo, _ := GetClientID(idToken.Audience, claims.AuthorizingParty); issuedTo != clientID {
		return nil, fmt.Errorf("id token was issued to %q", issuedTo)
	}
	return idToken, nil
}
