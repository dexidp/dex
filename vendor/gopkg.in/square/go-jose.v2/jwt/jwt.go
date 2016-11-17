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
	"gopkg.in/square/go-jose.v2"
	"gopkg.in/square/go-jose.v2/json"
)

// JSONWebToken represents a JSON Web Token (as specified in RFC7519).
type JSONWebToken struct {
	payload func(k interface{}) ([]byte, error)
	Headers []jose.Header
}

// Claims deserializes a JSONWebToken into dest using the provided key.
func (t *JSONWebToken) Claims(key interface{}, dest ...interface{}) error {
	b, err := t.payload(key)
	if err != nil {
		return err
	}

	for _, d := range dest {
		if err := json.Unmarshal(b, d); err != nil {
			return err
		}
	}

	return nil
}

// ParseSigned parses token from JWS form.
func ParseSigned(s string) (*JSONWebToken, error) {
	sig, err := jose.ParseSigned(s)
	if err != nil {
		return nil, err
	}
	headers := make([]jose.Header, len(sig.Signatures))
	for i, signature := range sig.Signatures {
		headers[i] = signature.Header
	}

	return &JSONWebToken{sig.Verify, headers}, nil
}

// ParseEncrypted parses token from JWE form.
func ParseEncrypted(s string) (*JSONWebToken, error) {
	enc, err := jose.ParseEncrypted(s)
	if err != nil {
		return nil, err
	}

	return &JSONWebToken{enc.Decrypt, []jose.Header{enc.Header}}, nil
}
