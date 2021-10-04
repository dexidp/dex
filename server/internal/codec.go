package internal

import (
	"encoding/base64"

	"google.golang.org/protobuf/proto"
)

// Marshal converts a protobuf message to a URL legal string.
func Marshal(message proto.Message) (string, error) {
	data, err := proto.Marshal(message)
	if err != nil {
		return "", err
	}
	return base64.RawURLEncoding.EncodeToString(data), nil
}

// Unmarshal decodes a protobuf message.
func Unmarshal(s string, message proto.Message) error {
	data, err := base64.RawURLEncoding.DecodeString(s)
	if err != nil {
		return err
	}
	return proto.Unmarshal(data, message)
}
