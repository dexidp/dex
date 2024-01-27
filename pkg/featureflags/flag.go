package featureflags

import (
	"os"
	"strconv"
	"strings"
)

type flag struct {
	Name    string
	Default bool
}

func (f *flag) env() string {
	return "DEX_" + strings.ToUpper(f.Name)
}

func (f *flag) Enabled() bool {
	raw := os.Getenv(f.env())
	if raw == "" {
		return f.Default
	}

	res, err := strconv.ParseBool(raw)
	if err != nil {
		return f.Default
	}
	return res
}

func newFlag(s string, d bool) *flag {
	return &flag{Name: s, Default: d}
}
