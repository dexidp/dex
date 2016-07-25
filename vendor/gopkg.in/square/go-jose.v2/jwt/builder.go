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

import "gopkg.in/square/go-jose.v2"

// Builder is a utility for making JSON Web Tokens. Calls can be chained, and
// errors are accumulated until the final call to CompactSerialize/FullSerialize.
type Builder struct {
	transform  func([]byte) (serializer, payload, error)
	payload    payload
	serializer serializer
	err        error
}

type payload func(interface{}) ([]byte, error)

type serializer interface {
	FullSerialize() string
	CompactSerialize() (string, error)
}

// Signed creates builder for signed tokens.
func Signed(sig jose.Signer) *Builder {
	return &Builder{
		transform: func(b []byte) (serializer, payload, error) {
			s, err := sig.Sign(b)
			if err != nil {
				return nil, nil, err
			}
			return s, s.Verify, nil
		},
	}
}

// Encrypted creates builder for encrypted tokens.
func Encrypted(enc jose.Encrypter) *Builder {
	return &Builder{
		transform: func(b []byte) (serializer, payload, error) {
			e, err := enc.Encrypt(b)
			if err != nil {
				return nil, nil, err
			}
			return e, e.Decrypt, nil
		},
	}
}

// Claims encodes claims into the builder.
func (b *Builder) Claims(c interface{}) *Builder {
	if b.transform == nil {
		panic("Signer/Encrypter not set")
	}

	if b.payload != nil {
		panic("Claims already set")
	}

	raw, err := marshalClaims(c)
	if err != nil {
		return &Builder{
			err: err,
		}
	}

	ser, pl, err := b.transform(raw)
	return &Builder{
		transform:  b.transform,
		serializer: ser,
		payload:    pl,
		err:        err,
	}
}

// Token builds a JSONWebToken from provided data.
func (b *Builder) Token() (*JSONWebToken, error) {
	if b.err != nil {
		return nil, b.err
	}

	if b.payload == nil {
		return nil, ErrInvalidClaims
	}

	return &JSONWebToken{b.payload}, nil
}

// FullSerialize serializes a token using the full serialization format.
func (b *Builder) FullSerialize() (string, error) {
	if b.err != nil {
		return "", b.err
	}

	if b.serializer == nil {
		return "", ErrInvalidClaims
	}

	return b.serializer.FullSerialize(), nil
}

// CompactSerialize serializes a token using the compact serialization format.
func (b *Builder) CompactSerialize() (string, error) {
	if b.err != nil {
		return "", b.err
	}

	if b.serializer == nil {
		return "", ErrInvalidClaims
	}

	return b.serializer.CompactSerialize()
}
