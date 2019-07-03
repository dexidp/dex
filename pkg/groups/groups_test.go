package groups_test

import (
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/dexidp/dex/pkg/groups"
)

func TestFilter(t *testing.T) {
	cases := map[string]struct {
		given, required, expected []string
	}{
		"nothing given":                 {given: []string{}, required: []string{"ops"}, expected: []string{}},
		"exactly one match":             {given: []string{"foo"}, required: []string{"foo"}, expected: []string{"foo"}},
		"no group of the required ones": {given: []string{"foo", "bar"}, required: []string{"baz"}, expected: []string{}},
		"subset matching":               {given: []string{"foo", "bar", "baz"}, required: []string{"bar", "baz"}, expected: []string{"bar", "baz"}},
	}
	for name, tc := range cases {
		t.Run(name, func(t *testing.T) {
			actual := groups.Filter(tc.given, tc.required)
			assert.ElementsMatch(t, tc.expected, actual)
		})
	}
}
