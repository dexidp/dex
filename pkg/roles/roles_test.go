package roles_test

import (
	"github.com/dexidp/dex/connector"
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/dexidp/dex/pkg/roles"
)

func TestFilter(t *testing.T) {
	cases := map[string]struct {
		groups   []string
		roles    map[string][]string
		ident    connector.Identity
		expected []string
	}{
		"nothing given": {groups: []string{}, roles: map[string][]string{},
			ident: connector.Identity{}, expected: []string{}},
		"no groups but roles given": {groups: []string{}, roles: map[string][]string{
			"admin":   []string{"admin-role", "admin-ui-role"},
			"user":    []string{"user-role", "user-ui-role"},
			"group":   []string{"group-role"},
			"unknown": []string{"unknown-role"},
		}, ident: connector.Identity{}, expected: []string{}},
		"groups but no roles given": {groups: []string{"admin", "user", "group"}, roles: map[string][]string{},
			ident: connector.Identity{}, expected: []string{}},
		"one_group for one role given": {groups: []string{"group"}, roles: map[string][]string{
			"admin":   []string{"admin-role", "admin-ui-role"},
			"user":    []string{"user-role", "user-ui-role"},
			"group":   []string{"group-role"},
			"unknown": []string{"unknown-role"},
		}, ident: connector.Identity{}, expected: []string{"group-role"}},
		"two_group for four role given": {groups: []string{"admin", "user"}, roles: map[string][]string{
			"admin":   []string{"admin-role", "admin-ui-role"},
			"user":    []string{"user-role", "user-ui-role"},
			"group":   []string{"group-role"},
			"unknown": []string{"unknown-role"},
		}, ident: connector.Identity{}, expected: []string{"admin-role", "admin-ui-role", "user-role", "user-ui-role"}},
		"all_group for all role given": {groups: []string{"group", "unknown", "admin", "user"}, roles: map[string][]string{
			"admin":   []string{"admin-role", "admin-ui-role"},
			"user":    []string{"user-role", "user-ui-role"},
			"group":   []string{"group-role"},
			"unknown": []string{"unknown-role"},
		}, ident: connector.Identity{}, expected: []string{"admin-role", "admin-ui-role", "group-role", "unknown-role",
			"user-role", "user-ui-role"}},
	}
	for name, tc := range cases {
		t.Run(name, func(t *testing.T) {
			roles.ApplyRoles(tc.groups, tc.roles, &tc.ident)
			actual := tc.ident.Roles
			assert.ElementsMatch(t, tc.expected, actual)
		})
	}
}
