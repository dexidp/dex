/*-
 * Copyright 2016 Zbigniew Mandziejewicz
 * Copyright 2016 Square, Inc.
 *
 * Licensed under the Apache License, Version 2.0 (the "License");
 * you may not use this file except in compliance with the License.
 * You may obtain a copy of the License at
 *
 *     http://www.apache.org/licenses/LICENSE-2.0
 *
 * Unless required by applicable law or agreed to in writing, software
 * distributed under the License is distributed on an "AS IS" BASIS,
 * WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 * See the License for the specific language governing permissions and
 * limitations under the License.
 */

package jwt

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gopkg.in/square/go-jose.v2"
)

var encryptionKey = []byte("secret")
var rawToken = `eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9.eyJzdWIiOiJzdWJqZWN0IiwiaXNzIjoiaXNzdWVyIiwic2NvcGVzIjpbInMxIiwiczIiXX0.Y6_PfQHrzRJ_Vlxij5VI07-pgDIuJNN3Z_g5sSaGQ0c`

type customClaims struct {
	Claims
	Scopes []string `json:"scopes,omitempty"`
}

func TestDecodeToken(t *testing.T) {
	tok, err := ParseSigned(rawToken)
	assert.NoError(t, err)
	c := &Claims{}
	if assert.NoError(t, tok.Claims(c, encryptionKey)) {
		assert.Equal(t, c.Subject, "subject")
		assert.Equal(t, c.Issuer, "issuer")
	}

	c2 := &customClaims{}
	if assert.NoError(t, tok.Claims(c2, encryptionKey)) {
		assert.Equal(t, c2.Subject, "subject")
		assert.Equal(t, c2.Issuer, "issuer")
		assert.Equal(t, c2.Scopes, []string{"s1", "s2"})
	}
}

func TestEncodeToken(t *testing.T) {
	signer, err := jose.NewSigner(jose.SigningKey{Algorithm: jose.HS256, Key: encryptionKey}, &jose.SignerOptions{})
	require.NoError(t, err)

	c := &customClaims{
		Claims: Claims{
			Subject: "subject",
			Issuer:  "issuer",
		},
		Scopes: []string{"s1", "s2"},
	}

	raw, err := Signed(signer).Claims(c).CompactSerialize()
	require.NoError(t, err)

	tok, err := ParseSigned(raw)
	require.NoError(t, err)

	c2 := &customClaims{}
	if assert.NoError(t, tok.Claims(c2, encryptionKey)) {
		assert.Equal(t, c2.Subject, "subject")
		assert.Equal(t, c2.Issuer, "issuer")
		assert.Equal(t, c2.Scopes, []string{"s1", "s2"})
	}
}
