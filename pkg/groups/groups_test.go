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
		"nil given":                     {given: nil, required: []string{"ops"}, expected: []string{}},
		"nil required":                  {given: []string{"foo"}, required: nil, expected: []string{}},
		"both nil":                      {given: nil, required: nil, expected: []string{}},
		"exactly one match":             {given: []string{"foo"}, required: []string{"foo"}, expected: []string{"foo"}},
		"no group of the required ones":  {given: []string{"foo", "bar"}, required: []string{"baz"}, expected: []string{}},
		"subset matching":               {given: []string{"foo", "bar", "baz"}, required: []string{"bar", "baz"}, expected: []string{"bar", "baz"}},
		"duplicate in required":         {given: []string{"a", "b"}, required: []string{"a", "a", "b"}, expected: []string{"a", "b"}},
		"duplicate in given":            {given: []string{"a", "a", "b"}, required: []string{"a"}, expected: []string{"a", "a"}},
		"order preserved from given":    {given: []string{"z", "x", "y"}, required: []string{"y", "z"}, expected: []string{"z", "y"}},
	}
	for name, tc := range cases {
		t.Run(name, func(t *testing.T) {
			actual := groups.Filter(tc.given, tc.required)
			assert.ElementsMatch(t, tc.expected, actual)
		})
	}
}
