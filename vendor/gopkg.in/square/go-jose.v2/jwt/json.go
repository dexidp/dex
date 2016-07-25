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

package jwt

import (
	"math"
	"reflect"
	"strconv"
	"time"

	"gopkg.in/square/go-jose.v2"
)

// NumericDate represents date and time as the number of seconds since the
// epoch, including leap seconds. Non-integer values can be represented
// in the serialized format, but we round to the nearest second.
type NumericDate int64

// TimeToNumericDate converts time.Time value into NumericDate.
func TimeToNumericDate(t time.Time) NumericDate {
	if t.IsZero() {
		return NumericDate(0)
	}

	// While RFC 7519 technically states that NumericDate values may be
	// non-integer values, we don't bother serializing timestamps in
	// claims with sub-second accurancy and just round to the nearest
	// second instead. Not convined sub-second accuracy is useful here.
	return NumericDate(t.Unix())
}

// MarshalJSON serializes the given NumericDate into its JSON representation.
func (n NumericDate) MarshalJSON() ([]byte, error) {
	return []byte(strconv.FormatInt(int64(n), 10)), nil
}

// UnmarshalJSON reads a date from its JSON representation.
func (n *NumericDate) UnmarshalJSON(b []byte) error {
	s := string(b)

	f, err := strconv.ParseFloat(s, 64)
	if err != nil {
		return ErrUnmarshalNumericDate
	}

	*n = NumericDate(f)
	return nil
}

// Time returns time.Time representation of NumericDate.
func (n NumericDate) Time() time.Time {
	i, f := math.Modf(float64(n))
	return time.Unix(int64(i), int64(f*float64(time.Second)))
}

type audience []string

func (s *audience) UnmarshalJSON(b []byte) error {
	var v interface{}
	if err := jose.UnmarshalJSON(b, &v); err != nil {
		return err
	}

	switch v := v.(type) {
	case string:
		*s = append(*s, v)
	case []interface{}:
		a := make([]string, len(v))
		for i, e := range v {
			s, ok := e.(string)
			if !ok {
				return ErrUnmarshalAudience
			}
			a[i] = s
		}
		*s = a
	default:
		return ErrUnmarshalAudience
	}

	return nil
}

var claimsType = reflect.TypeOf((*Claims)(nil)).Elem()

func publicClaims(cl interface{}) (*Claims, error) {
	v := reflect.ValueOf(cl)
	if v.IsNil() || v.Kind() != reflect.Ptr || v.Elem().Kind() != reflect.Struct {
		return nil, ErrInvalidClaims
	}

	v = v.Elem()
	f := v.FieldByName("Claims")
	if !f.IsValid() || f.Type() != claimsType {
		return nil, nil
	}

	c := f.Addr().Interface().(*Claims)
	return c, nil
}

func marshalClaims(cl interface{}) ([]byte, error) {
	switch cl := cl.(type) {
	case *Claims:
		return cl.marshalJSON()
	case map[string]interface{}:
		return jose.MarshalJSON(cl)
	}

	public, err := publicClaims(cl)
	if err != nil {
		return nil, err
	}
	// i doesn't contain nested jwt.Claims
	if public == nil {
		return jose.MarshalJSON(cl)
	}

	// marshal jwt.Claims
	b1, err := public.marshalJSON()
	if err != nil {
		return nil, err
	}

	// marshal private claims
	b2, err := jose.MarshalJSON(cl)
	if err != nil {
		return nil, err
	}

	// merge claims
	r := make([]byte, len(b1)+len(b2)-1)
	copy(r, b1)
	r[len(b1)-1] = ','
	copy(r[len(b1):], b2[1:])

	return r, nil
}

func unmarshalClaims(b []byte, cl interface{}) error {
	switch cl := cl.(type) {
	case *Claims:
		return cl.unmarshalJSON(b)
	case map[string]interface{}:
		return jose.UnmarshalJSON(b, cl)
	}

	if err := jose.UnmarshalJSON(b, cl); err != nil {
		return err
	}

	public, err := publicClaims(cl)
	if err != nil {
		return err
	}
	// unmarshal jwt.Claims
	if public != nil {
		if err := public.unmarshalJSON(b); err != nil {
			return err
		}
	}

	return nil
}
