package tokens

import (
	"crypto/sha256"
	"crypto/sha512"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"hash"
	"io"

	jose "github.com/go-jose/go-jose/v4"

	"github.com/dexidp/dex/server/internal"
)

var hashForSigAlg = map[jose.SignatureAlgorithm]func() hash.Hash{
	jose.RS256: sha256.New,
	jose.RS384: sha512.New384,
	jose.RS512: sha512.New,
	jose.ES256: sha256.New,
	jose.ES384: sha512.New384,
	jose.ES512: sha512.New,
}

// AccessTokenHash computes the at_hash/c_hash value for the given signing
// algorithm: the left-most half of the hash of the token, base64url-encoded.
func AccessTokenHash(alg jose.SignatureAlgorithm, accessToken string) (string, error) {
	newHash, ok := hashForSigAlg[alg]
	if !ok {
		return "", fmt.Errorf("unsupported signature algorithm: %s", alg)
	}

	hashFunc := newHash()
	if _, err := io.WriteString(hashFunc, accessToken); err != nil {
		return "", fmt.Errorf("computing hash: %v", err)
	}
	sum := hashFunc.Sum(nil)
	return base64.RawURLEncoding.EncodeToString(sum[:len(sum)/2]), nil
}

// Audience is an ID token's aud claim; it marshals to a single string when there
// is exactly one entry, otherwise to an array.
type Audience []string

func (a Audience) contains(aud string) bool {
	for _, e := range a {
		if aud == e {
			return true
		}
	}
	return false
}

func (a Audience) MarshalJSON() ([]byte, error) {
	if len(a) == 1 {
		return json.Marshal(a[0])
	}
	return json.Marshal([]string(a))
}

type IDTokenClaims struct {
	Issuer           string   `json:"iss"`
	Subject          string   `json:"sub"`
	Audience         Audience `json:"aud"`
	Expiry           int64    `json:"exp"`
	IssuedAt         int64    `json:"iat"`
	JWTID            string   `json:"jti,omitempty"`
	AuthorizingParty string   `json:"azp,omitempty"`
	Nonce            string   `json:"nonce,omitempty"`
	AuthTime         int64    `json:"auth_time,omitempty"`

	AccessTokenHash string `json:"at_hash,omitempty"`
	CodeHash        string `json:"c_hash,omitempty"`

	Email         string `json:"email,omitempty"`
	EmailVerified *bool  `json:"email_verified,omitempty"`

	Groups []string `json:"groups,omitempty"`

	Name              string `json:"name,omitempty"`
	PreferredUsername string `json:"preferred_username,omitempty"`

	FederatedIDClaims *FederatedIDClaims `json:"federated_claims,omitempty"`
}

// FederatedIDClaims carries the upstream connector's identity for the
// federated:id scope.
type FederatedIDClaims struct {
	ConnectorID string `json:"connector_id,omitempty"`
	UserID      string `json:"user_id,omitempty"`
}

// GetClientID resolves the requesting client from an audience and azp: the single
// audience entry, or the azp when the audience holds several clients.
func GetClientID(aud Audience, azp string) (string, error) {
	switch len(aud) {
	case 0:
		return "", fmt.Errorf("no audience is set, could not find ClientID")
	case 1:
		return aud[0], nil
	default:
		return azp, nil
	}
}

// GetAudience builds the aud claim for a client and its requested scopes,
// including any cross-client peers the scopes name.
func GetAudience(clientID string, scopes []string) Audience {
	var aud Audience

	for _, scope := range scopes {
		if peerID, ok := ParseCrossClientScope(scope); ok {
			aud = append(aud, peerID)
		}
	}

	if len(aud) == 0 {
		// Client didn't ask for cross client audience. Set the current
		// client as the audience.
		aud = Audience{clientID}
		// Client asked for cross client audience:
		// if the current client was not requested explicitly
	} else if !aud.contains(clientID) {
		// by default it becomes one of entries in Audience
		aud = append(aud, clientID)
	}

	return aud
}

// GenSubject encodes a (userID, connectorID) pair into an ID token subject.
func GenSubject(userID string, connID string) (string, error) {
	sub := &internal.IDTokenSubject{
		UserId: userID,
		ConnId: connID,
	}

	return internal.Marshal(sub)
}
