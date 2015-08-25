package flag

import (
	"encoding/base64"
	"fmt"
	"strings"
)

// Base64 implements flag.Value, and is used to populate []byte values from baes64 encoded strings.
type Base64 struct {
	val []byte
	len int
}

// NewBase64 returns a Base64 which accepts values which decode to len byte strings.
func NewBase64(len int) *Base64 {
	return &Base64{
		len: len,
	}
}

func (f *Base64) String() string {
	return base64.StdEncoding.EncodeToString(f.val)
}

// Set will set the []byte value of the Base64 to the base64 decoded values of the string, returning an error if it cannot be decoded or is of the wrong length.
func (f *Base64) Set(s string) error {
	b, err := base64.StdEncoding.DecodeString(s)
	if err != nil {
		return err
	}

	if len(b) != f.len {
		return fmt.Errorf("expected %d-byte secret", f.len)
	}

	f.val = b
	return nil
}

// Bytes returns the set []byte value.
// If no value has been set, a nil []byte is returned.
func (f *Base64) Bytes() []byte {
	return f.val
}

// NewBase64List returns a Base64List which accepts a comma-separated list of strings which must decode to len byte strings.
func NewBase64List(len int) *Base64List {
	return &Base64List{
		len: len,
	}
}

// Base64List implements flag.Value and is used to populate [][]byte values from a comma-separated list of base64 encoded strings.
type Base64List struct {
	val [][]byte
	len int
}

// Set will set the [][]byte value of the Base64List to the base64 decoded values of the comma-separated strings, returning an error on the first error it encounters.
func (f *Base64List) Set(ss string) error {
	if ss == "" {
		return nil
	}
	for i, s := range strings.Split(ss, ",") {
		b64 := NewBase64(f.len)
		err := b64.Set(s)
		if err != nil {
			return fmt.Errorf("error decoding string %d: %q", i, err)
		}
		f.val = append(f.val, b64.Bytes())
	}
	return nil
}

func (f *Base64List) String() string {
	ss := []string{}
	for _, b := range f.val {
		ss = append(ss, base64.StdEncoding.EncodeToString(b))
	}
	return strings.Join(ss, ",")
}

func (f *Base64List) BytesSlice() [][]byte {
	return f.val
}
