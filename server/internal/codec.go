package internal

import (
	"encoding/base64"
	"encoding/json"

	"github.com/golang/protobuf/proto"
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

// UnmarshalJSON unmarshals the subject claim's internal format
func (s *IDTokenSubject) UnmarshalJSON(src []byte) error {
	var sub string
	if err := json.Unmarshal(src, &sub); err != nil {
		return err
	}
	return Unmarshal(sub, s)
}
