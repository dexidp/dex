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

import "time"

const (
	// DefaultLeeway defines the default leeway for matching NotBefore/Expiry claims.
	DefaultLeeway = 1.0 * time.Minute
)

// Expected defines values used for claims validation.
type Expected struct {
	Issuer   string
	Subject  string
	Audience []string
	ID       string
	Time     time.Time
}

// WithTime copies expectations with new time.
func (e Expected) WithTime(t time.Time) Expected {
	e.Time = t
	return e
}

// Validate checks claims in a token against expected values.
// A default leeway value of one minute is used to compare time values.
func (c Claims) Validate(e Expected) error {
	return c.ValidateWithLeeway(e, DefaultLeeway)
}

// ValidateWithLeeway checks claims in a token against expected values. A
// custom leeway may be specified for comparing time values. You may pass a
// zero value to check time values with no leeway, but you should not that
// numeric date values are rounded to the nearest second and sub-second
// precision is not supported.
func (c Claims) ValidateWithLeeway(e Expected, leeway time.Duration) error {
	if e.Issuer != "" && e.Issuer != c.Issuer {
		return ErrInvalidIssuer
	}

	if e.Subject != "" && e.Subject != c.Subject {
		return ErrInvalidSubject
	}

	if e.ID != "" && e.ID != c.ID {
		return ErrInvalidID
	}

	if len(e.Audience) != 0 {
		if len(e.Audience) != len(c.Audience) {
			return ErrInvalidAudience
		}

		for i, a := range e.Audience {
			if a != c.Audience[i] {
				return ErrInvalidAudience
			}
		}
	}

	if !e.Time.IsZero() && e.Time.Add(leeway).Before(c.NotBefore) {
		return ErrNotValidYet
	}

	if !e.Time.IsZero() && e.Time.Add(-leeway).After(c.Expiry) {
		return ErrExpired
	}

	return nil
}
