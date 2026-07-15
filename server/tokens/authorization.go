package tokens

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"
	"time"

	"github.com/dexidp/dex/storage"
)

// Authorization is the result of any grant: an authenticated subject and the
// permission it grants a client. The issuer turns an Authorization into a TokenSet.
type Authorization struct {
	Client        storage.Client
	Claims        storage.Claims
	Scopes        []string
	ConnectorID   string
	Nonce         string
	AuthTime      time.Time
	ConnectorData []byte
}

// TokenSet is what the /token endpoint returns for an Authorization.
type TokenSet struct {
	AccessToken  string
	IDToken      string
	RefreshToken string
	Expiry       time.Time
}

type Response struct {
	AccessToken     string `json:"access_token"`
	IssuedTokenType string `json:"issued_token_type,omitempty"`
	TokenType       string `json:"token_type"`
	ExpiresIn       int    `json:"expires_in,omitempty"`
	RefreshToken    string `json:"refresh_token,omitempty"`
	IDToken         string `json:"id_token,omitempty"`
	Scope           string `json:"scope,omitempty"`
}

// Response renders the token set as an OAuth2 token response as of now, filling
// in the bearer token type and the relative expiry. Grant-specific fields
// (IssuedTokenType, Scope) are left to the caller.
func (ts TokenSet) Response(now time.Time) Response {
	return Response{
		AccessToken:  ts.AccessToken,
		TokenType:    "bearer",
		ExpiresIn:    int(ts.Expiry.Sub(now).Seconds()),
		RefreshToken: ts.RefreshToken,
		IDToken:      ts.IDToken,
	}
}

// Write marshals the response and writes it with the required OAuth2 cache
// headers (https://tools.ietf.org/html/rfc6749#section-5.1).
func (r Response) Write(w http.ResponseWriter) error {
	data, err := json.Marshal(r)
	if err != nil {
		return fmt.Errorf("marshal token response: %w", err)
	}

	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Content-Length", strconv.Itoa(len(data)))
	w.Header().Set("Cache-Control", "no-store")
	w.Header().Set("Pragma", "no-cache")
	w.Write(data)
	return nil
}
