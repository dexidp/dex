package net

import (
	"net/url"
	"strings"
)

// URLEqual checks two urls for equality using only the host and path portions.
func URLEqual(url1, url2 string) bool {
	u1, err := url.Parse(url1)
	if err != nil {
		return false
	}
	u2, err := url.Parse(url2)
	if err != nil {
		return false
	}

	return strings.ToLower(u1.Host+u1.Path) == strings.ToLower(u2.Host+u2.Path)
}
