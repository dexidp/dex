package server

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestParsePrompt(t *testing.T) {
	tests := []struct {
		name    string
		raw     string
		want    Prompt
		wantErr bool
	}{
		{name: "empty", raw: "", want: Prompt{}},
		{name: "none", raw: "none", want: Prompt{none: true}},
		{name: "login", raw: "login", want: Prompt{login: true}},
		{name: "consent", raw: "consent", want: Prompt{consent: true}},
		{name: "login consent", raw: "login consent", want: Prompt{login: true, consent: true}},
		{name: "consent login", raw: "consent login", want: Prompt{login: true, consent: true}},
		{name: "select_account", raw: "select_account", want: Prompt{selectAccount: true}},
		{name: "login select_account", raw: "login select_account", want: Prompt{login: true, selectAccount: true}},
		{name: "consent select_account", raw: "consent select_account", want: Prompt{consent: true, selectAccount: true}},
		{name: "duplicate values", raw: "login login", want: Prompt{login: true}},
		{name: "whitespace padding", raw: "  login  ", want: Prompt{login: true}},

		// Errors.
		{name: "none with login", raw: "none login", wantErr: true},
		{name: "none with consent", raw: "none consent", wantErr: true},
		{name: "none with select_account", raw: "none select_account", wantErr: true},
		{name: "unknown value", raw: "bogus", wantErr: true},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got, err := ParsePrompt(tc.raw)
			if tc.wantErr {
				require.Error(t, err)
				return
			}
			require.NoError(t, err)
			assert.Equal(t, tc.want, got)
		})
	}
}

func TestPromptString(t *testing.T) {
	tests := []struct {
		prompt Prompt
		want   string
	}{
		{Prompt{}, ""},
		{Prompt{none: true}, "none"},
		{Prompt{login: true}, "login"},
		{Prompt{consent: true}, "consent"},
		{Prompt{login: true, consent: true}, "login consent"},
		{Prompt{selectAccount: true}, "select_account"},
		{Prompt{login: true, selectAccount: true}, "login select_account"},
	}

	for _, tc := range tests {
		t.Run(tc.want, func(t *testing.T) {
			assert.Equal(t, tc.want, tc.prompt.String())
		})
	}
}
