// Package claims implements support for manipulating custom claims.
package claims

import (
	"context"
	"fmt"
	"regexp"
	"strings"

	"github.com/dexidp/dex/connector"
	"github.com/dexidp/dex/middleware"

	"github.com/dexidp/dex/pkg/log"
)

// Config holds the configuration parameters for the claims middleware.
// The claims middleware provides the ability to filter and prefix custom claims
// returned by things further up the chain.
//
// An example config:
//
//     type: claims
//     config:
//       actions:
//         - discard: "^foo\.example\.com/.*$"
//         - rename:
//             pattern: "^(.*)\.example\.com/(.*)$"
//             as: "example.com/$1/$2"
//         - stripPrefix: "foo/"
//         - addPrefix: "example.com/"
//       inject:
//         "example.com/foo": bar
//         "example.com/foobar": true
//
// Note that this middleware will *not* affect the standard claims made by
// Dex.  It will only manipulate custom claims.  We don't really anticipate
// much use of this middleware, because custom claims will more often be
// injected on the basis of external logic, which is more appropriately handled
// via gRPC.
//
type Config struct {
	// A list of actions to perform on each claim
	Actions []Action `json:"actions,omitempty"`

	// Additional claims to inject
	Inject map[string]interface{} `json:"inject,omitempty"`
}

// Renames claims matching the regular expression Pattern with As
type RenameAction struct {
	Pattern string `json:"pattern"`
	As      string `json:"as"`
}

// An action
type Action struct {
	// Discards claims whose names match a regexp
	Discard string `json:"discard,omitempty"`

	// Renames claims using a regexp
	Rename *RenameAction `json:"rename,omitempty"`

	// Remove a prefix string from a claim name
	StripPrefix string `json:"stripPrefix,omitempty"`

	// Add a prefix string to a claim name
	AddPrefix string `json:"addPrefix,omitempty"`
}

// The actual Middleware object uses these instead, so that we can pre-compile
// the regular expressions
type renameAction struct {
	Regexp *regexp.Regexp
	As     string
}

type action struct {
	Discard     *regexp.Regexp
	Rename      *renameAction
	StripPrefix string
	AddPrefix   string
}

// Open returns a claims Middleware
func (c *Config) Open(logger log.Logger) (middleware.Middleware, error) {
	// Compile the regular expressions
	actions := make([]action, len(c.Actions))
	for n, a := range c.Actions {
		actions[n] = action{
			StripPrefix: a.StripPrefix,
			AddPrefix:   a.AddPrefix,
		}

		if a.Discard != "" {
			re, err := regexp.Compile(a.Discard)
			if err != nil {
				return nil, fmt.Errorf("claims: unable to compile discard regexp %q: %v", a.Discard, err)
			}
			actions[n].Discard = re
		}

		if a.Rename != nil {
			re, err := regexp.Compile(a.Rename.Pattern)
			if err != nil {
				return nil, fmt.Errorf("claims: unable to compile rename regexp %q: %v", a.Rename.Pattern, err)
			}
			actions[n].Rename = &renameAction{
				Regexp: re,
				As:     a.Rename.As,
			}
		}
	}

	return &claimsMiddleware{Config: *c, CompiledActions: actions}, nil
}

type claimsMiddleware struct {
	Config

	CompiledActions []action
}

// Apply the actions to the claims in the incoming identity
func (c *claimsMiddleware) Process(ctx context.Context, identity connector.Identity) (connector.Identity, error) {
	newClaims := map[string]interface{}{}

	for claim, value := range identity.CustomClaims {
		discard := false

		for _, action := range c.CompiledActions {
			if action.Discard != nil {
				if action.Discard.MatchString(claim) {
					discard = true
					break
				}
			}

			if action.Rename != nil {
				claim = action.Rename.Regexp.ReplaceAllString(claim,
					action.Rename.As)
			}

			if action.StripPrefix != "" {
				claim = strings.TrimPrefix(claim, action.StripPrefix)
			}

			if action.AddPrefix != "" {
				claim = action.AddPrefix + claim
			}
		}

		if !discard {
			newClaims[claim] = value
		}
	}

	for claim, value := range c.Inject {
		newClaims[claim] = value
	}

	identity.CustomClaims = newClaims

	return identity, nil
}
