/*-
 * Copyright 2016 Zbigniew Mandziejewicz
 * Copyright 2016 Square, Inc.
 *
 * Licensed under the Apache License, Version 2.0 (the "License");
 * you may not use this file except in compliance with the License.
 * You may obtain a copy of the License at
 *
 *     http://www.apache.org/licenses/LICENSE-2.0
 *
 * Unless required by applicable law or agreed to in writing, software
 * distributed under the License is distributed on an "AS IS" BASIS,
 * WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 * See the License for the specific language governing permissions and
 * limitations under the License.
 */

package jwt_test

import (
	"fmt"
	"strings"
	"time"

	"gopkg.in/square/go-jose.v2"
	"gopkg.in/square/go-jose.v2/jwt"
)

var sharedKey = []byte("secret")
var sharedEncryptionKey = []byte("itsa16bytesecret")
var signer, _ = jose.NewSigner(jose.SigningKey{Algorithm: jose.HS256, Key: sharedKey}, &jose.SignerOptions{})

func ExampleParseSigned() {
	raw := `eyJhbGciOiJIUzI1NiJ9.eyJpc3MiOiJpc3N1ZXIiLCJzdWIiOiJzdWJqZWN0In0.gpHyA1B1H6X4a4Edm9wo7D3X2v3aLSDBDG2_5BzXYe0`
	tok, err := jwt.ParseSigned(raw)
	if err != nil {
		panic(err)
	}

	out := jwt.Claims{}
	if err := tok.Claims(sharedKey, &out); err != nil {
		panic(err)
	}
	fmt.Printf("iss: %s, sub: %s\n", out.Issuer, out.Subject)
	// Output: iss: issuer, sub: subject
}

func ExampleParseEncrypted() {
	key := []byte("itsa16bytesecret")
	raw := `eyJhbGciOiJkaXIiLCJlbmMiOiJBMTI4R0NNIn0..jg45D9nmr6-8awml.z-zglLlEw9MVkYHi-Znd9bSwc-oRGbqKzf9WjXqZxno.kqji2DiZHZmh-1bLF6ARPw`
	tok, err := jwt.ParseEncrypted(raw)
	if err != nil {
		panic(err)
	}

	out := jwt.Claims{}
	if err := tok.Claims(key, &out); err != nil {
		panic(err)
	}
	fmt.Printf("iss: %s, sub: %s\n", out.Issuer, out.Subject)
	//Output: iss: issuer, sub: subject
}

func ExampleClaims_Validate() {
	cl := jwt.Claims{
		Subject:   "subject",
		Issuer:    "issuer",
		NotBefore: jwt.NewNumericDate(time.Date(2016, 1, 1, 0, 0, 0, 0, time.UTC)),
		Expiry:    jwt.NewNumericDate(time.Date(2016, 1, 1, 0, 15, 0, 0, time.UTC)),
		Audience:  jwt.Audience{"leela", "fry"},
	}

	err := cl.Validate(jwt.Expected{
		Issuer: "issuer",
		Time:   time.Date(2016, 1, 1, 0, 10, 0, 0, time.UTC),
	})
	if err != nil {
		panic(err)
	}

	fmt.Printf("valid!")
	// Output: valid!
}

func ExampleClaims_Validate_withParse() {
	raw := `eyJhbGciOiJIUzI1NiJ9.eyJpc3MiOiJpc3N1ZXIiLCJzdWIiOiJzdWJqZWN0In0.gpHyA1B1H6X4a4Edm9wo7D3X2v3aLSDBDG2_5BzXYe0`
	tok, err := jwt.ParseSigned(raw)
	if err != nil {
		panic(err)
	}

	cl := jwt.Claims{}
	if err := tok.Claims(sharedKey, &cl); err != nil {
		panic(err)
	}

	err = cl.Validate(jwt.Expected{
		Issuer:  "issuer",
		Subject: "subject",
	})
	if err != nil {
		panic(err)
	}

	fmt.Printf("valid!")
	// Output: valid!
}

func ExampleSigned() {
	key := []byte("secret")
	sig, err := jose.NewSigner(jose.SigningKey{Algorithm: jose.HS256, Key: key}, &jose.SignerOptions{})
	if err != nil {
		panic(err)
	}

	cl := jwt.Claims{
		Subject:   "subject",
		Issuer:    "issuer",
		NotBefore: jwt.NewNumericDate(time.Date(2016, 1, 1, 0, 0, 0, 0, time.UTC)),
		Audience:  jwt.Audience{"leela", "fry"},
	}
	raw, err := jwt.Signed(sig).Claims(cl).CompactSerialize()
	if err != nil {
		panic(err)
	}

	fmt.Println(raw)
	// Output: eyJhbGciOiJIUzI1NiJ9.eyJhdWQiOlsibGVlbGEiLCJmcnkiXSwiaXNzIjoiaXNzdWVyIiwibmJmIjoxLjQ1MTYwNjRlKzA5LCJzdWIiOiJzdWJqZWN0In0.uazfxZNgnlLdNDK7JkuYj3LlT4jSyEDG8EWISBPUuME
}

func ExampleEncrypted() {
	enc, err := jose.NewEncrypter(jose.A128GCM, jose.Recipient{Algorithm: jose.DIRECT, Key: sharedEncryptionKey}, nil)
	if err != nil {
		panic(err)
	}

	cl := jwt.Claims{
		Subject: "subject",
		Issuer:  "issuer",
	}
	raw, err := jwt.Encrypted(enc).Claims(cl).CompactSerialize()
	if err != nil {
		panic(err)
	}

	fmt.Println(raw)
}

func ExampleSigned_multipleClaims() {
	c := &jwt.Claims{
		Subject: "subject",
		Issuer:  "issuer",
	}
	c2 := struct {
		Scopes []string
	}{
		[]string{"foo", "bar"},
	}
	raw, err := jwt.Signed(signer).Claims(c).Claims(c2).CompactSerialize()
	if err != nil {
		panic(err)
	}

	fmt.Println(raw)
	// Output: eyJhbGciOiJIUzI1NiJ9.eyJTY29wZXMiOlsiZm9vIiwiYmFyIl0sImlzcyI6Imlzc3VlciIsInN1YiI6InN1YmplY3QifQ.esKOIsmwkudr_gnfnB4SngxIr-7pspd5XzG3PImfQ6Y
}

func ExampleJSONWebToken_Claims_map() {
	raw := `eyJhbGciOiJIUzI1NiJ9.eyJpc3MiOiJpc3N1ZXIiLCJzdWIiOiJzdWJqZWN0In0.gpHyA1B1H6X4a4Edm9wo7D3X2v3aLSDBDG2_5BzXYe0`
	tok, err := jwt.ParseSigned(raw)
	if err != nil {
		panic(err)
	}

	out := make(map[string]interface{})
	if err := tok.Claims(sharedKey, &out); err != nil {
		panic(err)
	}

	fmt.Printf("iss: %s, sub: %s\n", out["iss"], out["sub"])
	// Output: iss: issuer, sub: subject
}

func ExampleJSONWebToken_Claims_multiple() {
	raw := `eyJhbGciOiJIUzI1NiJ9.eyJTY29wZXMiOlsiZm9vIiwiYmFyIl0sImlzcyI6Imlzc3VlciIsInN1YiI6InN1YmplY3QifQ.esKOIsmwkudr_gnfnB4SngxIr-7pspd5XzG3PImfQ6Y`
	tok, err := jwt.ParseSigned(raw)
	if err != nil {
		panic(err)
	}

	out := jwt.Claims{}
	out2 := struct {
		Scopes []string
	}{}
	if err := tok.Claims(sharedKey, &out, &out2); err != nil {
		panic(err)
	}
	fmt.Printf("iss: %s, sub: %s, scopes: %s\n", out.Issuer, out.Subject, strings.Join(out2.Scopes, ","))
	// Output: iss: issuer, sub: subject, scopes: foo,bar
}
