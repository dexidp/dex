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
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestFieldsMatch(t *testing.T) {
	c := Claims{
		Issuer:   "issuer",
		Subject:  "subject",
		Audience: []string{"a1", "a2"},
		ID:       "42",
	}

	assert.NoError(t, c.Validate(Expected{Issuer: "issuer"}))
	err := c.Validate(Expected{Issuer: "invalid-issuer"})
	if assert.Error(t, err) {
		assert.Equal(t, err, ErrInvalidIssuer)
	}

	assert.NoError(t, c.Validate(Expected{Subject: "subject"}))
	err = c.Validate(Expected{Subject: "invalid-subject"})
	if assert.Error(t, err) {
		assert.Equal(t, err, ErrInvalidSubject)
	}

	assert.NoError(t, c.Validate(Expected{Audience: []string{"a1", "a2"}}))
	err = c.Validate(Expected{Audience: []string{"invalid-audience"}})
	if assert.Error(t, err) {
		assert.Equal(t, err, ErrInvalidAudience)
	}

	assert.NoError(t, c.Validate(Expected{ID: "42"}))
	err = c.Validate(Expected{ID: "invalid-id"})
	if assert.Error(t, err) {
		assert.Equal(t, err, ErrInvalidID)
	}
}

func TestExpiryAndNotBefore(t *testing.T) {
	now := time.Date(2016, 1, 1, 12, 0, 0, 0, time.UTC)
	twelveHoursAgo := now.Add(-12 * time.Hour)

	c := Claims{
		IssuedAt:  twelveHoursAgo,
		NotBefore: twelveHoursAgo,
		Expiry:    now,
	}

	// expired - default leeway (1 minute)
	assert.NoError(t, c.Validate(Expected{Time: now}))
	err := c.Validate(Expected{Time: now.Add(2 * DefaultLeeway)})
	if assert.Error(t, err) {
		assert.Equal(t, err, ErrExpired)
	}

	// expired - no leeway
	assert.NoError(t, c.ValidateWithLeeway(Expected{Time: now}, 0))
	err = c.ValidateWithLeeway(Expected{Time: now.Add(1 * time.Second)}, 0)
	if assert.Error(t, err) {
		assert.Equal(t, err, ErrExpired)
	}

	// not before - default leeway (1 minute)
	assert.NoError(t, c.Validate(Expected{Time: twelveHoursAgo}))
	err = c.Validate(Expected{Time: twelveHoursAgo.Add(-2 * DefaultLeeway)})
	if assert.Error(t, err) {
		assert.Equal(t, err, ErrNotValidYet)
	}
}
