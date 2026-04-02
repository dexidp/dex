package server

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/base64"

	"google.golang.org/protobuf/proto"

	"github.com/dexidp/dex/server/internal"
)

// computeHMAC computes a SHA-256 HMAC over a protobuf-encoded payload
// and returns the result as a base64 raw-URL-encoded string.
func computeHMAC(key []byte, values ...string) string {
	msg := marshalHMACPayload(values)
	h := hmac.New(sha256.New, key)
	h.Write(msg)
	return base64.RawURLEncoding.EncodeToString(h.Sum(nil))
}

// verifyHMAC checks that encodedMAC (base64 raw-URL) matches the
// HMAC-SHA256 of the protobuf-encoded payload under key.
func verifyHMAC(key []byte, encodedMAC string, values ...string) bool {
	mac, err := base64.RawURLEncoding.DecodeString(encodedMAC)
	if err != nil {
		return false
	}
	msg := marshalHMACPayload(values)
	h := hmac.New(sha256.New, key)
	h.Write(msg)
	return hmac.Equal(mac, h.Sum(nil))
}

func marshalHMACPayload(values []string) []byte {
	payload := &internal.HMACPayload{Values: values}
	// proto.Marshal is deterministic for the same input in the Go implementation.
	data, _ := proto.Marshal(payload)
	return data
}
