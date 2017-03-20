package main

import (
	"fmt"
	"strings"

	oidc "github.com/coreos/go-oidc"
)

// authorizer is a strategy for determining if a user is allowed to view the
// backend resources behind the proxy.
//
// Most strategies involve evalutating the email claim.
type authorizer interface {
	authorized(idToken *oidc.IDToken) error
}

type unionAuthorizer struct {
	authorizers []authorizer
}

type multiErr struct {
	errs []error
}

func (m *multiErr) Error() string {
	s := make([]string, len(m.errs))
	for i, e := range m.errs {
		s[i] = e.Error()
	}
	return strings.Join(s, ", ")
}

func (a unionAuthorizer) authorized(idToken *oidc.IDToken) error {
	var errs []error
	for _, authorizer := range a.authorizers {
		err := authorizer.authorized(idToken)
		if err != nil {
			errs = append(errs, err)
			continue
		}
		return nil
	}
	return &multiErr{errs: errs}
}

type allowAll struct{}

func (a *allowAll) authorized(idToken *oidc.IDToken) error {
	return nil
}

func emailFromClaims(idToken *oidc.IDToken) (string, error) {
	var c struct {
		Email    string `json:"email"`
		Verified bool   `json:"email_verified"`
	}
	if err := idToken.Claims(&c); err != nil {
		return "", fmt.Errorf("decode claims: %v", err)
	}
	if c.Email == "" {
		return "", fmt.Errorf("no email in claims")
	}
	if !c.Verified {
		return "", fmt.Errorf("email isn't verified")
	}
	return c.Email, nil
}

func sliceToMap(s []string) map[string]bool {
	m := make(map[string]bool, len(s))
	for _, v := range s {
		m[v] = true
	}
	return m
}

func allowEmailDomains(domains ...string) authorizer {
	return &emailDomainAuthorizer{sliceToMap(domains)}
}

type emailDomainAuthorizer struct {
	domains map[string]bool
}

func (e *emailDomainAuthorizer) authorized(idToken *oidc.IDToken) error {
	email, err := emailFromClaims(idToken)
	if err != nil {
		return fmt.Errorf("parsing email in claims: %v", err)
	}

	// Simple email parsing. Just grab everything after the first "@".
	i := strings.LastIndex(email, "@")
	if i < 0 || i == len(email) {
		return fmt.Errorf("email address has no domain: %s", email)
	}
	domain := email[i+1:]

	if e.domains[domain] {
		return nil
	}
	return fmt.Errorf("email not in allowed list of domains: %s", email)
}

func allowEmailWhitelist(emails ...string) authorizer {
	return &emailWhitelist{sliceToMap(emails)}
}

type emailWhitelist struct {
	emails map[string]bool
}

func (e *emailWhitelist) authorized(idToken *oidc.IDToken) error {
	email, err := emailFromClaims(idToken)
	if err != nil {
		return fmt.Errorf("parsing email in claims: %v", err)
	}
	if e.emails[email] {
		return nil
	}
	return fmt.Errorf("email not in whitelist: %s", email)
}
