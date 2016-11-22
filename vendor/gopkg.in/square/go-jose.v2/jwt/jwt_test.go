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
)

var (
	hmacSignedToken              = `eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9.eyJzdWIiOiJzdWJqZWN0IiwiaXNzIjoiaXNzdWVyIiwic2NvcGVzIjpbInMxIiwiczIiXX0.Y6_PfQHrzRJ_Vlxij5VI07-pgDIuJNN3Z_g5sSaGQ0c`
	rsaSignedToken               = `eyJhbGciOiJSUzI1NiJ9.eyJpc3MiOiJpc3N1ZXIiLCJzY29wZXMiOlsiczEiLCJzMiJdLCJzdWIiOiJzdWJqZWN0In0.UDDtyK9gC9kyHltcP7E_XODsnqcJWZIiXeGmSAH7SE9YKy3N0KSfFIN85dCNjTfs6zvy4rkrCHzLB7uKAtzMearh3q7jL4nxbhUMhlUcs_9QDVoN4q_j58XmRqBqRnBk-RmDu9TgcV8RbErP4awpIhwWb5UU-hR__4_iNbHdKqwSUPDKYGlf5eicuiYrPxH8mxivk4LRD-vyRdBZZKBt0XIDnEU4TdcNCzAXojkftqcFWYsczwS8R4JHd1qYsMyiaWl4trdHZkO4QkeLe34z4ZAaPMt3wE-gcU-VoqYTGxz-K3Le2VaZ0r3j_z6bOInsv0yngC_cD1dCXMyQJWnWjQ`
	invalidPayloadSignedToken    = `eyJhbGciOiJIUzI1NiJ9.aW52YWxpZC1wYXlsb2Fk.ScBKKm18jcaMLGYDNRUqB5gVMRZl4DM6dh3ShcxeNgY`
	invalidPartsSignedToken      = `eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9.eyJzdWIiOiJzdWJqZWN0IiwiaXNzIjoiaXNzdWVyIiwic2NvcGVzIjpbInMxIiwiczIiXX0`
	hmacEncryptedToken           = `eyJhbGciOiJkaXIiLCJlbmMiOiJBMTI4R0NNIn0..NZrU98U4QNO0y-u6.HSq5CvlmkUT1BPqLGZ4.1-zuiZ4RbHrTTUoA8Dvfhg`
	rsaEncryptedToken            = `eyJhbGciOiJSU0ExXzUiLCJlbmMiOiJBMTI4Q0JDLUhTMjU2In0.IvkVHHiI8JwwavvTR80xGjYvkzubMrZ-TDDx8k8SNJMEylfFfNUc7F2rC3WAABF_xmJ3SW2A6on-S6EAG97k0RsjqHHNqZuaFpDvjeuLqZFfYKzI45aCtkGG4C2ij2GbeySqJ784CcvFJPUWJ-6VPN2Ho2nhefUSqig0jE2IvOKy1ywTj_VBVBxF_dyXFnXwxPKGUQr3apxrWeRJfDh2Cf8YPBlLiRznjfBfwgePB1jP7WCZNwItj10L7hsT_YWEx01XJcbxHaXFLwKyVzwWaDhreFyaWMRbGqEfqVuOT34zfmhLDhQlgLLwkXrvYqX90NsQ9Ftg0LLIfRMbsfdgug.BFy2Tj1RZN8yq2Lk-kMiZQ.9Z0eOyPiv5cEzmXh64RlAQ36Uvz0WpZgqRcc2_69zHTmUOv0Vnl1I6ks8sTraUEvukAilolNBjBj47s0b4b-Og.VM8-eJg5ZsqnTqs0LtGX_Q`
	invalidPayloadEncryptedToken = `eyJhbGciOiJkaXIiLCJlbmMiOiJBMTI4R0NNIn0..T4jCS4Yyw1GCH0aW.y4gFaMITdBs_QZM8RKrL.6MPyk1cMVaOJFoNGlEuaRQ`
	invalidPartsEncryptedToken   = `eyJhbGciOiJkaXIiLCJlbmMiOiJBMTI4R0NNIn0..NZrU98U4QNO0y-u6.HSq5CvlmkUT1BPqLGZ4`
)

type customClaims struct {
	Scopes []string `json:"scopes,omitempty"`
}

func TestDecodeToken(t *testing.T) {
	tok, err := ParseSigned(hmacSignedToken)
	if assert.NoError(t, err, "Error parsing signed token.") {
		c := &Claims{}
		c2 := &customClaims{}
		if assert.NoError(t, tok.Claims(sharedKey, c, c2)) {
			assert.Equal(t, "subject", c.Subject)
			assert.Equal(t, "issuer", c.Issuer)
			assert.Equal(t, []string{"s1", "s2"}, c2.Scopes)
		}
	}
	assert.EqualError(t, tok.Claims([]byte("invalid-secret")), "square/go-jose: error in cryptographic primitive")

	tok2, err := ParseSigned(rsaSignedToken)
	if assert.NoError(t, err, "Error parsing encrypted token.") {
		c := make(map[string]interface{})
		if assert.NoError(t, tok2.Claims(&testPrivRSAKey1.PublicKey, &c)) {
			assert.Equal(t, map[string]interface{}{
				"sub":    "subject",
				"iss":    "issuer",
				"scopes": []interface{}{"s1", "s2"},
			}, c)
		}
	}
	assert.EqualError(t, tok.Claims(&testPrivRSAKey2.PublicKey), "square/go-jose: error in cryptographic primitive")

	tok3, err := ParseSigned(invalidPayloadSignedToken)
	if assert.NoError(t, err, "Error parsing signed token.") {
		assert.Error(t, tok3.Claims(sharedKey, &Claims{}), "Expected unmarshaling claims to fail.")
	}

	_, err = ParseSigned(invalidPartsSignedToken)
	assert.EqualError(t, err, "square/go-jose: compact JWS format must have three parts")

	tok4, err := ParseEncrypted(hmacEncryptedToken)
	if assert.NoError(t, err, "Error parsing encrypted token.") {
		c := Claims{}
		if assert.NoError(t, tok4.Claims(sharedEncryptionKey, &c)) {
			assert.Equal(t, "foo", c.Subject)
		}
	}
	assert.EqualError(t, tok4.Claims([]byte("invalid-secret-key")), "square/go-jose: error in cryptographic primitive")

	tok5, err := ParseEncrypted(rsaEncryptedToken)
	if assert.NoError(t, err, "Error parsing encrypted token.") {
		c := make(map[string]interface{})
		if assert.NoError(t, tok5.Claims(testPrivRSAKey1, &c)) {
			assert.Equal(t, map[string]interface{}{
				"sub":    "subject",
				"iss":    "issuer",
				"scopes": []interface{}{"s1", "s2"},
			}, c)
		}
	}
	assert.EqualError(t, tok5.Claims(testPrivRSAKey2), "square/go-jose: error in cryptographic primitive")

	tok6, err := ParseEncrypted(invalidPayloadEncryptedToken)
	if assert.NoError(t, err, "Error parsing encrypted token.") {
		assert.Error(t, tok6.Claims(sharedEncryptionKey, &Claims{}))
	}

	_, err = ParseEncrypted(invalidPartsEncryptedToken)
	assert.EqualError(t, err, "square/go-jose: compact JWE format must have five parts")
}

func BenchmarkDecodeSignedToken(b *testing.B) {
	for i := 0; i < b.N; i++ {
		ParseSigned(hmacSignedToken)
	}
}

func BenchmarkDecodeEncryptedHMACToken(b *testing.B) {
	for i := 0; i < b.N; i++ {
		ParseEncrypted(hmacEncryptedToken)
	}
}
