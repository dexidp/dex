package server

import (
	"fmt"
	"strings"
)

// Prompt represents the parsed OIDC "prompt" parameter (RFC 6749 / OpenID Connect Core 3.1.2.1).
// The parameter is space-separated and may contain: "none", "login", "consent", "select_account".
// "none" must not be combined with any other value.
type Prompt struct {
	none    bool
	login   bool
	consent bool
}

// ParsePrompt parses and validates the raw prompt query parameter.
// Returns an error suitable for returning as an OAuth2 invalid_request if the value is invalid.
func ParsePrompt(raw string) (Prompt, error) {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return Prompt{}, nil
	}

	var p Prompt
	seen := make(map[string]bool)

	for _, v := range strings.Fields(raw) {
		if seen[v] {
			continue
		}
		seen[v] = true

		switch v {
		case "none":
			p.none = true
		case "login":
			p.login = true
		case "consent":
			p.consent = true
		case "select_account":
			// Dex does not support account selection; ignore per spec recommendation.
		default:
			return Prompt{}, fmt.Errorf("invalid prompt value %q", v)
		}
	}

	if p.none && (p.login || p.consent) {
		return Prompt{}, fmt.Errorf("prompt=none must not be combined with other values")
	}

	return p, nil
}

// None returns true if the caller requested no interactive UI.
func (p Prompt) None() bool { return p.none }

// Login returns true if the caller requested forced re-authentication.
func (p Prompt) Login() bool { return p.login }

// Consent returns true if the caller requested forced consent screen.
func (p Prompt) Consent() bool { return p.consent }

// String returns the canonical space-separated representation stored in the database.
func (p Prompt) String() string {
	var parts []string
	if p.none {
		return "none"
	}
	if p.login {
		parts = append(parts, "login")
	}
	if p.consent {
		parts = append(parts, "consent")
	}
	return strings.Join(parts, " ")
}
