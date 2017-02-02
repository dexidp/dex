package main

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"encoding/base64"
	"errors"
	"fmt"
	"io"
	"net/http"
	"sync"
)

// An encrypted store for cookie-ing users.
type cookieStore struct {
	// If omitted, will be generated on first use.
	key     *[32]byte
	keyOnce sync.Once

	insecureAllowHTTPCookies bool

	// TODO(ericchiang): allow multiple decryption keys.
}

func (c *cookieStore) setCookie(w http.ResponseWriter, name string, val []byte) error {
	encrypted, err := encrypt(val, c.getKey())
	if err != nil {
		return fmt.Errorf("encrypt cookie: %v", err)
	}

	http.SetCookie(w, &http.Cookie{
		Name:     name,
		Value:    base64.RawURLEncoding.EncodeToString(encrypted),
		Path:     "/",
		Secure:   !c.insecureAllowHTTPCookies,
		HttpOnly: true,
	})
	return nil
}

func (c *cookieStore) deleteCookie(w http.ResponseWriter, name string) {
	http.SetCookie(w, &http.Cookie{
		Name:     name,
		Path:     "/",
		Secure:   !c.insecureAllowHTTPCookies,
		HttpOnly: true,
		MaxAge:   -1,
	})
}

func (c *cookieStore) cookie(r *http.Request, name string) []byte {
	cookie, err := r.Cookie(name)
	if err != nil {
		return nil
	}
	val, err := base64.RawURLEncoding.DecodeString(cookie.Value)
	if err != nil {
		return nil
	}
	plainText, err := decrypt(val, c.getKey())
	if err != nil {
		return nil
	}
	return plainText
}

func (c *cookieStore) getKey() *[32]byte {
	c.keyOnce.Do(func() {
		if c.key != nil {
			return
		}

		var b [32]byte
		if _, err := rand.Read(b[:]); err != nil {
			panic(err)
		}
		c.key = &b
	})
	return c.key
}

// Copied from github.com/gtank/cryptopasta

// Encrypt encrypts data using 256-bit AES-GCM.  This both hides the content of
// the data and provides a check that it hasn't been altered. Output takes the
// form nonce|ciphertext|tag where '|' indicates concatenation.
func encrypt(plaintext []byte, key *[32]byte) (ciphertext []byte, err error) {
	block, err := aes.NewCipher(key[:])
	if err != nil {
		return nil, err
	}

	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return nil, err
	}

	nonce := make([]byte, gcm.NonceSize())
	_, err = io.ReadFull(rand.Reader, nonce)
	if err != nil {
		return nil, err
	}

	return gcm.Seal(nonce, nonce, plaintext, nil), nil
}

// Decrypt decrypts data using 256-bit AES-GCM.  This both hides the content of
// the data and provides a check that it hasn't been altered. Expects input
// form nonce|ciphertext|tag where '|' indicates concatenation.
func decrypt(ciphertext []byte, key *[32]byte) (plaintext []byte, err error) {
	block, err := aes.NewCipher(key[:])
	if err != nil {
		return nil, err
	}

	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return nil, err
	}

	if len(ciphertext) < gcm.NonceSize() {
		return nil, errors.New("malformed ciphertext")
	}

	return gcm.Open(nil,
		ciphertext[:gcm.NonceSize()],
		ciphertext[gcm.NonceSize():],
		nil,
	)
}
