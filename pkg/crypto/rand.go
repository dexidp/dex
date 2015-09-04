package crypto

import (
	"crypto/rand"
	"errors"
)

func RandBytes(n int) ([]byte, error) {
	b := make([]byte, n)
	got, err := rand.Read(b)
	if err != nil {
		return nil, err
	}
	if n != got {
		return nil, errors.New("unable to generate enough random data")
	}
	return b, nil
}
