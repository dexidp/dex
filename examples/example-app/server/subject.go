package server

import "encoding/base64"

// idTokenSubject builds the sub claim dex issues for a user.
//
// dex's API is inconsistent about which identifier it takes: sessions are keyed
// by the ID the connector gave the user, refresh tokens by the sub claim. Rather
// than ask for a value nobody has to hand, the app derives one from the other.
//
// The sub is base64url of a two-field protobuf message — user_id then conn_id,
// both length-delimited strings — which is small enough to encode here and
// avoids the example depending on dex's internals for it.
func idTokenSubject(userID, connectorID string) string {
	var buf []byte
	buf = appendProtoString(buf, 1, userID)
	buf = appendProtoString(buf, 2, connectorID)
	return base64.RawURLEncoding.EncodeToString(buf)
}

// appendProtoString writes one length-delimited protobuf string field.
func appendProtoString(buf []byte, field int, value string) []byte {
	if value == "" {
		return buf
	}
	buf = appendVarint(buf, uint64(field)<<3|2) // wire type 2: length-delimited
	buf = appendVarint(buf, uint64(len(value)))
	return append(buf, value...)
}

func appendVarint(buf []byte, v uint64) []byte {
	for v >= 0x80 {
		buf = append(buf, byte(v)|0x80)
		v >>= 7
	}
	return append(buf, byte(v))
}
