package roles

import (
	"github.com/dexidp/dex/connector"
	"sort"
)

func ApplyRoles(groups []string, appliedRoles map[string][]string, newIdent *connector.Identity) {
	if groups != nil && appliedRoles != nil {
		var identRoles []string
		uniqueRoles := make(map[string]int)
		identRoles = make([]string, 0)
		for _, group := range groups {
			if rolesToAdd, ok := appliedRoles[group]; ok {
				for _, element := range rolesToAdd {
					uniqueRoles[element] = 1
				}
			}
		}
		for role := range uniqueRoles {
			identRoles = append(identRoles, role)
		}
		sort.Strings(identRoles)
		newIdent.Roles = identRoles
	}
}
