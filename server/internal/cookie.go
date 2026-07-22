package internal

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"encoding/base64"
	"errors"
	"fmt"
	"io"
)

// SessionCookieValue marshals the session identity into a cookie value. If
// encryptionKey is provided, the value is encrypted.
func SessionCookieValue(userID, connectorID, nonce string, encryptionKey []byte) string {
	val, err := Marshal(&SessionCookie{
		UserId:      userID,
		ConnectorId: connectorID,
		Nonce:       nonce,
	})
	if err != nil {
		// Should never happen with valid string inputs.
		panic(fmt.Sprintf("marshal session cookie: %v", err))
	}
	if len(encryptionKey) > 0 {
		val, err = encryptCookieValue(val, encryptionKey)
		if err != nil {
			panic(fmt.Sprintf("encrypt session cookie: %v", err))
		}
	}
	return val
}

// ParseSessionCookie decodes a session cookie value. If encryptionKey is
// provided, the value is decrypted first.
func ParseSessionCookie(value string, encryptionKey []byte) (userID, connectorID, nonce string, err error) {
	if len(encryptionKey) > 0 {
		value, err = decryptCookieValue(value, encryptionKey)
		if err != nil {
			return "", "", "", fmt.Errorf("decrypt session cookie: %w", err)
		}
	}
	var cookie SessionCookie
	if err := Unmarshal(value, &cookie); err != nil {
		return "", "", "", fmt.Errorf("decode session cookie: %w", err)
	}
	return cookie.UserId, cookie.ConnectorId, cookie.Nonce, nil
}

// encryptCookieValue encrypts plaintext with AES-GCM and returns base64url-encoded ciphertext.
func encryptCookieValue(plaintext string, key []byte) (string, error) {
	block, err := aes.NewCipher(key)
	if err != nil {
		return "", err
	}
	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return "", err
	}
	nonce := make([]byte, gcm.NonceSize())
	if _, err := io.ReadFull(rand.Reader, nonce); err != nil {
		return "", err
	}
	ciphertext := gcm.Seal(nonce, nonce, []byte(plaintext), nil)
	return base64.RawURLEncoding.EncodeToString(ciphertext), nil
}

// decryptCookieValue decodes base64url and decrypts AES-GCM ciphertext.
func decryptCookieValue(encrypted string, key []byte) (string, error) {
	data, err := base64.RawURLEncoding.DecodeString(encrypted)
	if err != nil {
		return "", err
	}
	block, err := aes.NewCipher(key)
	if err != nil {
		return "", err
	}
	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return "", err
	}
	nonceSize := gcm.NonceSize()
	if len(data) < nonceSize {
		return "", errors.New("ciphertext too short")
	}
	plaintext, err := gcm.Open(nil, data[:nonceSize], data[nonceSize:], nil)
	if err != nil {
		return "", err
	}
	return string(plaintext), nil
}
