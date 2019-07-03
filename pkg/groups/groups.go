// Package groups contains helper functions related to groups
package groups

// Filter filters out any groups of given that are not in required. Thus it may
// happen that the resulting slice is empty.
func Filter(given, required []string) []string {
	groups := []string{}
	groupFilter := make(map[string]struct{})
	for _, group := range required {
		groupFilter[group] = struct{}{}
	}
	for _, group := range given {
		if _, ok := groupFilter[group]; ok {
			groups = append(groups, group)
		}
	}
	return groups
}
