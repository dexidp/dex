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
	"crypto/sha256"
	"crypto/sha512"
	"encoding/hex"
	"fmt"
	"io/ioutil"
	"os"
	"testing"
)

func TestPasswordHashing(t *testing.T) {
	bcryptTests := []struct {
		plaintext []byte
		hash      []byte
	}{
		{
			plaintext: []byte("password"),
			hash:      []byte("$2a$14$uALAQb/Lwl59oHVbuUa5m.xEFmQBc9ME/IiSgJK/VHtNJJXASCDoS"),
		},
	}

	for _, tt := range bcryptTests {
		hashed, err := HashPassword(tt.plaintext)
		if err != nil {
			t.Error(err)
		}

		if err = CheckPasswordHash(hashed, tt.plaintext); err != nil {
			t.Error(err)
		}
	}
}

// Benchmarks SHA256 on 16K of random data.
func BenchmarkSHA256(b *testing.B) {
	data, err := ioutil.ReadFile("testdata/random")
	if err != nil {
		b.Fatal(err)
	}
	b.SetBytes(int64(len(data)))
	for i := 0; i < b.N; i++ {
		_ = sha256.Sum256(data)
	}
}

// Benchmarks SHA512/256 on 16K of random data.
func BenchmarkSHA512_256(b *testing.B) {
	data, err := ioutil.ReadFile("testdata/random")
	if err != nil {
		b.Fatal(err)
	}
	b.SetBytes(int64(len(data)))
	for i := 0; i < b.N; i++ {
		_ = sha512.Sum512_256(data)
	}
}

func BenchmarkBcrypt(b *testing.B) {
	for i := 0; i < b.N; i++ {
		_, err := HashPassword([]byte("thisisareallybadpassword"))
		if err != nil {
			b.Error(err)
			break
		}
	}
}

func ExampleHash() {
	tag := "hashing file for lookup key"
	contents, err := ioutil.ReadFile("testdata/random")
	if err != nil {
		fmt.Printf("could not read file: %v\n", err)
		os.Exit(1)
	}
	digest := Hash(tag, contents)
	fmt.Println(hex.EncodeToString(digest))
	// Output: 9f4c795d8ae5c207f19184ccebee6a606c1fdfe509c793614066d613580f03e1
}
