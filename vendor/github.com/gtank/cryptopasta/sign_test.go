// cryptopasta - basic cryptography examples
//
// Written in 2015 by George Tankersley <george.tankersley@gmail.com>
//
// To the extent possible under law, the author(s) have dedicated all copyright
// and related and neighboring rights to this software to the public domain
// worldwide. This software is distributed without any warranty.
//
// You should have received a copy of the CC0 Public Domain Dedication along
// with this software. If not, see // <http://creativecommons.org/publicdomain/zero/1.0/>.

package cryptopasta

import (
	"bytes"
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"encoding/hex"
	"testing"
)

// https://groups.google.com/d/msg/sci.crypt/OolWgsgQD-8/jHciyWkaL0gJ
var hmacTests = []struct {
	key    string
	data   string
	digest string
}{
	{
		key:    "0b0b0b0b0b0b0b0b0b0b0b0b0b0b0b0b0b0b0b0b",
		data:   "4869205468657265", // "Hi There"
		digest: "9f9126c3d9c3c330d760425ca8a217e31feae31bfe70196ff81642b868402eab",
	},
	{
		key:    "4a656665",                                                 // "Jefe"
		data:   "7768617420646f2079612077616e7420666f72206e6f7468696e673f", // "what do ya want for nothing?"
		digest: "6df7b24630d5ccb2ee335407081a87188c221489768fa2020513b2d593359456",
	},
}

func TestHMAC(t *testing.T) {
	for idx, tt := range hmacTests {
		keySlice, _ := hex.DecodeString(tt.key)
		dataBytes, _ := hex.DecodeString(tt.data)
		expectedDigest, _ := hex.DecodeString(tt.digest)

		keyBytes := &[32]byte{}
		copy(keyBytes[:], keySlice)

		macDigest := GenerateHMAC(dataBytes, keyBytes)
		if !bytes.Equal(macDigest, expectedDigest) {
			t.Errorf("test %d generated unexpected mac", idx)
		}
	}
}

func TestSign(t *testing.T) {
	message := []byte("Hello, world!")

	key, err := NewSigningKey()
	if err != nil {
		t.Error(err)
		return
	}

	signature, err := Sign(message, key)
	if err != nil {
		t.Error(err)
		return
	}

	if !Verify(message, signature, &key.PublicKey) {
		t.Error("signature was not correct")
		return
	}

	message[0] ^= 0xff
	if Verify(message, signature, &key.PublicKey) {
		t.Error("signature was good for altered message")
	}
}

func TestSignWithP384(t *testing.T) {
	message := []byte("Hello, world!")

	key, err := ecdsa.GenerateKey(elliptic.P384(), rand.Reader)
	if err != nil {
		t.Error(err)
		return
	}

	signature, err := Sign(message, key)
	if err != nil {
		t.Error(err)
		return
	}

	if !Verify(message, signature, &key.PublicKey) {
		t.Error("signature was not correct")
		return
	}

	message[0] ^= 0xff
	if Verify(message, signature, &key.PublicKey) {
		t.Error("signature was good for altered message")
	}
}
