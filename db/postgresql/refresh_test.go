package db

import (
	"bytes"
	"testing"
)

func TestBuildAndParseToken(t *testing.T) {
	tests := []struct {
		id      int64
		payload []byte
	}{
		{11111, []byte("may the force be with you")},
		{123213, []byte("If we can hit that bullseye the rest of the dominoes will fall like a house of cards, checkmate!")},
		{1, []byte{0xd3, 0x22, 0xa8, 0x44, 0x34, 0x94, 0xd8}},
	}

	for i, tt := range tests {
		id, payload, err := parseToken(buildToken(tt.id, tt.payload))
		if err != nil {
			t.Errorf("case %d: failed to parse token: %v", i, err)
			continue
		}
		if tt.id != id {
			t.Errorf("case %d: want id=%d, got id=%d", i, tt.id, id)
		}
		if bytes.Compare(tt.payload, payload) != 0 {
			t.Errorf("case %d: want payload=%x, got payload=%x", i, tt.payload, payload)
		}
	}
}
