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
	"fmt"
	"reflect"
)

// Encrypter represents an encrypter which produces an encrypted JWE object.
type Encrypter interface {
	Encrypt(plaintext []byte) (*JSONWebEncryption, error)
	EncryptWithAuthData(plaintext []byte, aad []byte) (*JSONWebEncryption, error)
}

// A generic content cipher
type contentCipher interface {
	keySize() int
	encrypt(cek []byte, aad, plaintext []byte) (*aeadParts, error)
	decrypt(cek []byte, aad []byte, parts *aeadParts) ([]byte, error)
}

// A key generator (for generating/getting a CEK)
type keyGenerator interface {
	keySize() int
	genKey() ([]byte, rawHeader, error)
}

// A generic key encrypter
type keyEncrypter interface {
	encryptKey(cek []byte, alg KeyAlgorithm) (recipientInfo, error) // Encrypt a key
}

// A generic key decrypter
type keyDecrypter interface {
	decryptKey(headers rawHeader, recipient *recipientInfo, generator keyGenerator) ([]byte, error) // Decrypt a key
}

// A generic encrypter based on the given key encrypter and content cipher.
type genericEncrypter struct {
	contentAlg     ContentEncryption
	compressionAlg CompressionAlgorithm
	cipher         contentCipher
	recipients     []recipientKeyInfo
	keyGenerator   keyGenerator
}

type recipientKeyInfo struct {
	keyID        string
	keyAlg       KeyAlgorithm
	keyEncrypter keyEncrypter
}

// EncrypterOptions represents options that can be set on new encrypters.
type EncrypterOptions struct {
	Compression CompressionAlgorithm
}

// Recipient represents an algorithm/key to encrypt messages to.
type Recipient struct {
	Algorithm KeyAlgorithm
	Key       interface{}
	KeyID     string
}

// NewEncrypter creates an appropriate encrypter based on the key type
func NewEncrypter(enc ContentEncryption, rcpt Recipient, opts *EncrypterOptions) (Encrypter, error) {
	encrypter := &genericEncrypter{
		contentAlg: enc,
		recipients: []recipientKeyInfo{},
		cipher:     getContentCipher(enc),
	}
	if opts != nil {
		encrypter.compressionAlg = opts.Compression
	}

	if encrypter.cipher == nil {
		return nil, ErrUnsupportedAlgorithm
	}

	var keyID string
	var rawKey interface{}
	switch encryptionKey := rcpt.Key.(type) {
	case *JSONWebKey:
		keyID = encryptionKey.KeyID
		rawKey = encryptionKey.Key
	default:
		rawKey = encryptionKey
	}

	switch rcpt.Algorithm {
	case DIRECT:
		// Direct encryption mode must be treated differently
		if reflect.TypeOf(rawKey) != reflect.TypeOf([]byte{}) {
			return nil, ErrUnsupportedKeyType
		}
		encrypter.keyGenerator = staticKeyGenerator{
			key: rawKey.([]byte),
		}
		recipientInfo, _ := newSymmetricRecipient(rcpt.Algorithm, rawKey.([]byte))
		recipientInfo.keyID = keyID
		if rcpt.KeyID != "" {
			recipientInfo.keyID = rcpt.KeyID
		}
		encrypter.recipients = []recipientKeyInfo{recipientInfo}
		return encrypter, nil
	case ECDH_ES:
		// ECDH-ES (w/o key wrapping) is similar to DIRECT mode
		typeOf := reflect.TypeOf(rawKey)
		if typeOf != reflect.TypeOf(&ecdsa.PublicKey{}) {
			return nil, ErrUnsupportedKeyType
		}
		encrypter.keyGenerator = ecKeyGenerator{
			size:      encrypter.cipher.keySize(),
			algID:     string(enc),
			publicKey: rawKey.(*ecdsa.PublicKey),
		}
		recipientInfo, _ := newECDHRecipient(rcpt.Algorithm, rawKey.(*ecdsa.PublicKey))
		recipientInfo.keyID = keyID
		if rcpt.KeyID != "" {
			recipientInfo.keyID = rcpt.KeyID
		}
		encrypter.recipients = []recipientKeyInfo{recipientInfo}
		return encrypter, nil
	default:
		// Can just add a standard recipient
		encrypter.keyGenerator = randomKeyGenerator{
			size: encrypter.cipher.keySize(),
		}
		err := encrypter.addRecipient(rcpt)
		return encrypter, err
	}
}

// NewMultiEncrypter creates a multi-encrypter based on the given parameters
func NewMultiEncrypter(enc ContentEncryption, rcpts []Recipient, opts *EncrypterOptions) (Encrypter, error) {
	cipher := getContentCipher(enc)

	if cipher == nil {
		return nil, ErrUnsupportedAlgorithm
	}
	if rcpts == nil || len(rcpts) == 0 {
		return nil, fmt.Errorf("square/go-jose: recipients is nil or empty")
	}

	encrypter := &genericEncrypter{
		contentAlg: enc,
		recipients: []recipientKeyInfo{},
		cipher:     cipher,
		keyGenerator: randomKeyGenerator{
			size: cipher.keySize(),
		},
	}

	if opts != nil {
		encrypter.compressionAlg = opts.Compression
	}

	for _, recipient := range rcpts {
		err := encrypter.addRecipient(recipient)
		if err != nil {
			return nil, err
		}
	}

	return encrypter, nil
}

func (ctx *genericEncrypter) addRecipient(recipient Recipient) (err error) {
	var recipientInfo recipientKeyInfo

	switch recipient.Algorithm {
	case DIRECT, ECDH_ES:
		return fmt.Errorf("square/go-jose: key algorithm '%s' not supported in multi-recipient mode", recipient.Algorithm)
	}

	recipientInfo, err = makeJWERecipient(recipient.Algorithm, recipient.Key)
	if recipient.KeyID != "" {
		recipientInfo.keyID = recipient.KeyID
	}

	if err == nil {
		ctx.recipients = append(ctx.recipients, recipientInfo)
	}
	return err
}

func makeJWERecipient(alg KeyAlgorithm, encryptionKey interface{}) (recipientKeyInfo, error) {
	switch encryptionKey := encryptionKey.(type) {
	case *rsa.PublicKey:
		return newRSARecipient(alg, encryptionKey)
	case *ecdsa.PublicKey:
		return newECDHRecipient(alg, encryptionKey)
	case []byte:
		return newSymmetricRecipient(alg, encryptionKey)
	case *JSONWebKey:
		recipient, err := makeJWERecipient(alg, encryptionKey.Key)
		recipient.keyID = encryptionKey.KeyID
		return recipient, err
	default:
		return recipientKeyInfo{}, ErrUnsupportedKeyType
	}
}

// newDecrypter creates an appropriate decrypter based on the key type
func newDecrypter(decryptionKey interface{}) (keyDecrypter, error) {
	switch decryptionKey := decryptionKey.(type) {
	case *rsa.PrivateKey:
		return &rsaDecrypterSigner{
			privateKey: decryptionKey,
		}, nil
	case *ecdsa.PrivateKey:
		return &ecDecrypterSigner{
			privateKey: decryptionKey,
		}, nil
	case []byte:
		return &symmetricKeyCipher{
			key: decryptionKey,
		}, nil
	case *JSONWebKey:
		return newDecrypter(decryptionKey.Key)
	default:
		return nil, ErrUnsupportedKeyType
	}
}

// Implementation of encrypt method producing a JWE object.
func (ctx *genericEncrypter) Encrypt(plaintext []byte) (*JSONWebEncryption, error) {
	return ctx.EncryptWithAuthData(plaintext, nil)
}

// Implementation of encrypt method producing a JWE object.
func (ctx *genericEncrypter) EncryptWithAuthData(plaintext, aad []byte) (*JSONWebEncryption, error) {
	obj := &JSONWebEncryption{}
	obj.aad = aad

	obj.protected = &rawHeader{
		Enc: ctx.contentAlg,
	}
	obj.recipients = make([]recipientInfo, len(ctx.recipients))

	if len(ctx.recipients) == 0 {
		return nil, fmt.Errorf("square/go-jose: no recipients to encrypt to")
	}

	cek, headers, err := ctx.keyGenerator.genKey()
	if err != nil {
		return nil, err
	}

	obj.protected.merge(&headers)

	for i, info := range ctx.recipients {
		recipient, err := info.keyEncrypter.encryptKey(cek, info.keyAlg)
		if err != nil {
			return nil, err
		}

		recipient.header.Alg = string(info.keyAlg)
		if info.keyID != "" {
			recipient.header.Kid = info.keyID
		}
		obj.recipients[i] = recipient
	}

	if len(ctx.recipients) == 1 {
		// Move per-recipient headers into main protected header if there's
		// only a single recipient.
		obj.protected.merge(obj.recipients[0].header)
		obj.recipients[0].header = nil
	}

	if ctx.compressionAlg != NONE {
		plaintext, err = compress(ctx.compressionAlg, plaintext)
		if err != nil {
			return nil, err
		}

		obj.protected.Zip = ctx.compressionAlg
	}

	authData := obj.computeAuthData()
	parts, err := ctx.cipher.encrypt(cek, authData, plaintext)
	if err != nil {
		return nil, err
	}

	obj.iv = parts.iv
	obj.ciphertext = parts.ciphertext
	obj.tag = parts.tag

	return obj, nil
}

// Decrypt and validate the object and return the plaintext.
func (obj JSONWebEncryption) Decrypt(decryptionKey interface{}) ([]byte, error) {
	headers := obj.mergedHeaders(nil)

	if len(headers.Crit) > 0 {
		return nil, fmt.Errorf("square/go-jose: unsupported crit header")
	}

	decrypter, err := newDecrypter(decryptionKey)
	if err != nil {
		return nil, err
	}

	cipher := getContentCipher(headers.Enc)
	if cipher == nil {
		return nil, fmt.Errorf("square/go-jose: unsupported enc value '%s'", string(headers.Enc))
	}

	generator := randomKeyGenerator{
		size: cipher.keySize(),
	}

	parts := &aeadParts{
		iv:         obj.iv,
		ciphertext: obj.ciphertext,
		tag:        obj.tag,
	}

	authData := obj.computeAuthData()

	var plaintext []byte
	for _, recipient := range obj.recipients {
		recipientHeaders := obj.mergedHeaders(&recipient)

		cek, err := decrypter.decryptKey(recipientHeaders, &recipient, generator)
		if err == nil {
			// Found a valid CEK -- let's try to decrypt.
			plaintext, err = cipher.decrypt(cek, authData, parts)
			if err == nil {
				break
			}
		}
	}

	if plaintext == nil {
		return nil, ErrCryptoFailure
	}

	// The "zip" header parameter may only be present in the protected header.
	if obj.protected.Zip != "" {
		plaintext, err = decompress(obj.protected.Zip, plaintext)
	}

	return plaintext, err
}
