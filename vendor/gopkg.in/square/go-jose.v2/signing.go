/*-
 * Copyright 2014 Square Inc.
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

package jose

import (
	"crypto/ecdsa"
	"crypto/rsa"
	"encoding/base64"
	"fmt"
)

// NonceSource represents a source of random nonces to go into JWS objects
type NonceSource interface {
	Nonce() (string, error)
}

// Signer represents a signer which takes a payload and produces a signed JWS object.
type Signer interface {
	Sign(payload []byte) (*JSONWebSignature, error)
}

// SigningKey represents an algorithm/key used to sign a message.
type SigningKey struct {
	Algorithm SignatureAlgorithm
	Key       interface{}
}

// SignerOptions represents options that can be set when creating signers.
type SignerOptions struct {
	NonceSource NonceSource
	EmbedJWK    bool
}

type payloadSigner interface {
	signPayload(payload []byte, alg SignatureAlgorithm) (Signature, error)
}

type payloadVerifier interface {
	verifyPayload(payload []byte, signature []byte, alg SignatureAlgorithm) error
}

type genericSigner struct {
	recipients  []recipientSigInfo
	nonceSource NonceSource
	embedJWK    bool
}

type recipientSigInfo struct {
	sigAlg    SignatureAlgorithm
	publicKey *JSONWebKey
	signer    payloadSigner
}

// NewSigner creates an appropriate signer based on the key type
func NewSigner(sig SigningKey, opts *SignerOptions) (Signer, error) {
	return NewMultiSigner([]SigningKey{sig}, opts)
}

// NewMultiSigner creates a signer for multiple recipients
func NewMultiSigner(sigs []SigningKey, opts *SignerOptions) (Signer, error) {
	signer := &genericSigner{recipients: []recipientSigInfo{}}

	if opts != nil {
		signer.nonceSource = opts.NonceSource
		signer.embedJWK = opts.EmbedJWK
	}

	for _, sig := range sigs {
		err := signer.addRecipient(sig.Algorithm, sig.Key)
		if err != nil {
			return nil, err
		}
	}

	return signer, nil
}

// newVerifier creates a verifier based on the key type
func newVerifier(verificationKey interface{}) (payloadVerifier, error) {
	switch verificationKey := verificationKey.(type) {
	case *rsa.PublicKey:
		return &rsaEncrypterVerifier{
			publicKey: verificationKey,
		}, nil
	case *ecdsa.PublicKey:
		return &ecEncrypterVerifier{
			publicKey: verificationKey,
		}, nil
	case []byte:
		return &symmetricMac{
			key: verificationKey,
		}, nil
	case *JSONWebKey:
		return newVerifier(verificationKey.Key)
	default:
		return nil, ErrUnsupportedKeyType
	}
}

func (ctx *genericSigner) addRecipient(alg SignatureAlgorithm, signingKey interface{}) error {
	recipient, err := makeJWSRecipient(alg, signingKey)
	if err != nil {
		return err
	}

	ctx.recipients = append(ctx.recipients, recipient)
	return nil
}

func makeJWSRecipient(alg SignatureAlgorithm, signingKey interface{}) (recipientSigInfo, error) {
	switch signingKey := signingKey.(type) {
	case *rsa.PrivateKey:
		return newRSASigner(alg, signingKey)
	case *ecdsa.PrivateKey:
		return newECDSASigner(alg, signingKey)
	case []byte:
		return newSymmetricSigner(alg, signingKey)
	case *JSONWebKey:
		recipient, err := makeJWSRecipient(alg, signingKey.Key)
		if err != nil {
			return recipientSigInfo{}, err
		}
		recipient.publicKey.KeyID = signingKey.KeyID
		return recipient, nil
	default:
		return recipientSigInfo{}, ErrUnsupportedKeyType
	}
}

func (ctx *genericSigner) Sign(payload []byte) (*JSONWebSignature, error) {
	obj := &JSONWebSignature{}
	obj.payload = payload
	obj.Signatures = make([]Signature, len(ctx.recipients))

	for i, recipient := range ctx.recipients {
		protected := &rawHeader{
			Alg: string(recipient.sigAlg),
		}

		if recipient.publicKey != nil {
			if ctx.embedJWK {
				protected.Jwk = recipient.publicKey
			}
			protected.Kid = recipient.publicKey.KeyID
		}

		if ctx.nonceSource != nil {
			nonce, err := ctx.nonceSource.Nonce()
			if err != nil {
				return nil, fmt.Errorf("square/go-jose: Error generating nonce: %v", err)
			}
			protected.Nonce = nonce
		}

		serializedProtected := mustSerializeJSON(protected)

		input := []byte(fmt.Sprintf("%s.%s",
			base64.RawURLEncoding.EncodeToString(serializedProtected),
			base64.RawURLEncoding.EncodeToString(payload)))

		signatureInfo, err := recipient.signer.signPayload(input, recipient.sigAlg)
		if err != nil {
			return nil, err
		}

		signatureInfo.protected = protected
		obj.Signatures[i] = signatureInfo
	}

	return obj, nil
}

// Verify validates the signature on the object and returns the payload.
func (obj JSONWebSignature) Verify(verificationKey interface{}) ([]byte, error) {
	verifier, err := newVerifier(verificationKey)
	if err != nil {
		return nil, err
	}

	for _, signature := range obj.Signatures {
		headers := signature.mergedHeaders()
		if len(headers.Crit) > 0 {
			// Unsupported crit header
			continue
		}

		input := obj.computeAuthData(&signature)
		alg := SignatureAlgorithm(headers.Alg)
		err := verifier.verifyPayload(input, signature.Signature, alg)
		if err == nil {
			return obj.payload, nil
		}
	}

	return nil, ErrCryptoFailure
}
