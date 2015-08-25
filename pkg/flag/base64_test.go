package flag

import (
	"encoding/base64"
	"strings"
	"testing"

	"github.com/kylelemons/godebug/pretty"
)

func TestBase64(t *testing.T) {
	toB64 := func(b []byte) string {
		return base64.StdEncoding.EncodeToString(b)
	}

	tests := []struct {
		s         string
		l         int
		b         []byte
		wantError bool
	}{
		{
			s: toB64([]byte("123456")),
			l: 6,
			b: []byte("123456"),
		},
		{
			s:         toB64([]byte("123456")),
			l:         5,
			wantError: true,
		},
		{
			s:         "not base64",
			l:         5,
			wantError: true,
		},
	}

	for i, tt := range tests {
		b64 := NewBase64(tt.l)
		err := b64.Set(tt.s)
		if tt.wantError {
			if err == nil {
				t.Errorf("case %d: want err, got nil", i)
			}
			continue
		}

		if err != nil {
			t.Errorf("case %d: unexpected error %q", i, err)
		}

		if diff := pretty.Compare(tt.b, b64.Bytes()); diff != "" {
			t.Errorf("case %d: Compare(want, got) = %v", i,
				diff)
		}

		if b64.String() != tt.s {
			t.Errorf("case %d: want=%q, got=%q", i, b64.String(), tt.s)
		}
	}
}

func TestBase64List(t *testing.T) {
	// toCSB64 == to comma separated base 64
	toCSB64 := func(bb ...[]byte) string {
		ss := []string{}
		for _, b := range bb {
			ss = append(ss, base64.StdEncoding.EncodeToString(b))
		}
		return strings.Join(ss, ",")
	}

	b123 := []byte("123456")
	b567 := []byte("567890")
	bShort := []byte("1234")

	tests := []struct {
		s         string
		l         int
		bb        [][]byte
		wantError bool
	}{
		{
			s:  toCSB64(b123, b567),
			l:  6,
			bb: [][]byte{b123, b567},
		},
		{
			s:  toCSB64(b123),
			l:  6,
			bb: [][]byte{b123},
		},
		{
			s:  "",
			l:  6,
			bb: [][]byte{},
		},
		{
			s:         toCSB64(b123, bShort),
			l:         6,
			wantError: true,
		},
		{
			s:         toCSB64(bShort, b123),
			l:         6,
			wantError: true,
		},
	}

	for i, tt := range tests {
		b64 := NewBase64List(tt.l)
		err := b64.Set(tt.s)
		if tt.wantError {
			if err == nil {
				t.Errorf("case %d: want err, got nil", i)
			}
			continue
		}

		if err != nil {
			t.Errorf("case %d: unexpected error %q", i, err)
		}

		if diff := pretty.Compare(tt.bb, b64.BytesSlice()); diff != "" {
			t.Errorf("case %d: Compare(want, got) = %v", i,
				diff)
		}

		if b64.String() != tt.s {
			t.Errorf("case %d: want=%q, got=%q", i, b64.String(), tt.s)
		}
	}
}
