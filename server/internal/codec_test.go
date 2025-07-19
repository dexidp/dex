package internal

import (
	"encoding/base64"
	"testing"

	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/types/known/anypb"
)

func TestMarshal(t *testing.T) {
	tests := []struct {
		name      string
		input     proto.Message
		expected  string
		expectErr bool
	}{
		{
			name:      "ValidMessage",
			input:     &anypb.Any{TypeUrl: "example.com/type", Value: []byte("test")},
			expected:  "ChBleGFtcGxlLmNvbS90eXBlEgR0ZXN0",
			expectErr: false,
		},
		{
			name:      "NilMessage",
			input:     nil,
			expected:  "",
			expectErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := Marshal(tt.input)
			if tt.expectErr {
				if err == nil {
					t.Fatalf("expected an error, got none")
				}
				return
			}

			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}

			if result != tt.expected {
				t.Fatalf("unexpected result: got %s, want %s", result, tt.expected)
			}

			if _, decodeErr := base64.RawURLEncoding.DecodeString(result); decodeErr != nil {
				t.Fatalf("result is not a valid base64 string: %v", decodeErr)
			}
		})
	}
}

func TestUnmarshal(t *testing.T) {
	tests := []struct {
		name      string
		input     string
		output    proto.Message
		expectErr bool
	}{
		{
			name: "ValidBase64Input",
			input: func() string {
				data, _ := proto.Marshal(&anypb.Any{TypeUrl: "example.com/type", Value: []byte("test")})
				return base64.RawURLEncoding.EncodeToString(data)
			}(),
			output:    &anypb.Any{},
			expectErr: false,
		},
		{
			name:      "InvalidBase64Input",
			input:     "%%invalid-base64%%",
			output:    &anypb.Any{},
			expectErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := Unmarshal(tt.input, tt.output)
			if tt.expectErr {
				if err == nil {
					t.Fatalf("expected an error, got none")
				}
				return
			}

			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}

			if anyMsg, ok := tt.output.(*anypb.Any); ok {
				expectedMsg := &anypb.Any{TypeUrl: "example.com/type", Value: []byte("test")}
				if !proto.Equal(anyMsg, expectedMsg) {
					t.Fatalf("unexpected decoded message: got %v, want %v", anyMsg, expectedMsg)
				}
			}
		})
	}
}
