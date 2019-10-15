// +build go1.8

package gocbcore

import (
	"crypto/tls"
)

func cloneTLSConfig(c *tls.Config) *tls.Config {
	if c == nil {
		return &tls.Config{}
	}

	return c.Clone()
}
