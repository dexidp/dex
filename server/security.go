package server

import (
	"fmt"

	jose "gopkg.in/square/go-jose.v2"
)

// Encrypt encrypts and serializes object
func (s *Server) Encrypt(message string) (string, error) {
	keys, err := s.storage.GetKeys()
	if err != nil {
		return "", fmt.Errorf("failed to get keys: %v", err)
	}

	if keys.SigningKeyPub == nil {
		return "", fmt.Errorf("no public keys found")
	}

	// Instantiate an encrypter using RSA-OAEP with AES128-GCM. An error would
	// indicate that the selected algorithm(s) are not currently supported.
	encrypter, err := jose.NewEncrypter(jose.A128GCM, jose.Recipient{Algorithm: jose.RSA_OAEP, Key: keys.SigningKeyPub}, nil)
	// encrypter, err := jose.NewEncrypter(jose.RSA_OAEP, jose.A128GCM, &keys.SigningKeyPub)
	if err != nil {
		return "", fmt.Errorf("unable to create encrypter: %v", err)
	}

	// Encrypt the incoming string. Calling the encrypter returns an encrypted
	// JWE object, which can then be serialized for output afterwards. An error
	// would indicate a problem in an underlying cryptographic primitive.
	object, err := encrypter.Encrypt([]byte(message))
	if err != nil {
		return "", fmt.Errorf("unable to encrypt message: %v", err)
	}

	// Serialize the encrypted object using the full serialization format.
	// Alternatively you can also use the compact format here by calling
	// object.CompactSerialize() instead.
	serializedObject, err := object.CompactSerialize()
	if err != nil {
		return "", fmt.Errorf("unable to serialize encrypted message: %v", err)
	}

	return serializedObject, nil
}

// Decrypt decrupts and deserializes a string
func (s *Server) Decrypt(message string) (string, error) {
	keys, err := s.storage.GetKeys()
	if err != nil {
		return "", fmt.Errorf("failed to get keys: %v", err)
	}

	if keys.SigningKey == nil {
		return "", fmt.Errorf("no private keys found")
	}

	// Parse the serialized, encrypted JWE object. An error would indicate that
	// the given input did not represent a valid message.
	object, err := jose.ParseEncrypted(message)
	if err != nil {
		return "", fmt.Errorf("unable to parse encrypted message")
	}

	// Decrypt and get back the original message. An error here
	// would indicate the the message failed to decrypt, e.g. because the auth
	// tag was broken or the message was tampered with.
	decrypted, err := object.Decrypt(keys.SigningKey)
	if err != nil {
		return "", fmt.Errorf("unable to decrypt message: %v", err)
	}

	return string(decrypted), nil
}
