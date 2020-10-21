// Package groups implements support for manipulating groups claims.
package groups

import (
	"context"
	"fmt"
	"regexp"
	"sort"
	"strings"

	"github.com/dexidp/dex/connector"
	"github.com/dexidp/dex/middleware"

	"github.com/dexidp/dex/pkg/log"
)

// Config holds the configuration parameters for the groups middleware.
// The groups middleware provides the ability to filter and prefix groups
// returned by things further up the chain.
//
// An example config:
//
//     type: groups
//     config:
//       actions:
//         - discard: "^admin$"
//         - replace:
//             pattern: "\s+"
//             with:    " "
//         - stripPrefix: "foo/"
//         - addPrefix: "ldap/"
//       inject:
//         - DexUsers
//         - Users
//       sorted: true
//       unique: true
//
type Config struct {
	// A list of actions to perform on each group name
	Actions []Action `json:"actions,omitempty"`

	// Additional groups to inject
	Inject []string `json:"inject,omitempty"`

	// If true, sort the resulting list of groups
	Sorted bool `json:"sorted,omitempty"`

	// If true, ensure that each group is listed at most once
	Unique bool `json:"unique,omitempty"`
}

// Replaces matches of the regular expression Pattern with With
type ReplaceAction struct {
	Pattern string `json:"pattern"`
	With    string `json:"with"`
}

// An action
type Action struct {
	// Discards groups whose names match a regexp
	Discard string `json:"discard,omitempty"`

	// Replace regexp matches in a group name
	Replace *ReplaceAction `json:"replace,omitempty"`

	// Remove a prefix string from a group name
	StripPrefix string `json:"stripPrefix,omitempty"`

	// Add a prefix string to a group name
	AddPrefix string `json:"addPrefix,omitempty"`
}

// The actual Middleware object uses these instead, so that we can pre-compile
// the regular expressions
type replaceAction struct {
	Regexp *regexp.Regexp
	With   string
}

type action struct {
	Discard     *regexp.Regexp
	Replace     *replaceAction
	StripPrefix string
	AddPrefix   string
}

// Open returns a groups Middleware
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
				return nil, fmt.Errorf("groups: unable to compile discard regexp %q: %v", a.Discard, err)
			}
			actions[n].Discard = re
		}

		if a.Replace != nil {
			re, err := regexp.Compile(a.Replace.Pattern)
			if err != nil {
				return nil, fmt.Errorf("groups: unable to compile replace regexp %q: %v", a.Replace.Pattern, err)
			}
			actions[n].Replace = &replaceAction{
				Regexp: re,
				With:   a.Replace.With,
			}
		}
	}

	return &groupsMiddleware{Config: *c, CompiledActions: actions}, nil
}

type groupsMiddleware struct {
	Config

	CompiledActions []action
}

// Apply the actions to the groups in the incoming identity
func (g *groupsMiddleware) Process(ctx context.Context, identity connector.Identity) (connector.Identity, error) {
	groupSet := map[string]struct{}{}
	exists := struct{}{}
	newGroups := []string{}

	if g.Unique && g.Inject != nil {
		for _, group := range g.Inject {
			groupSet[group] = exists
		}
	}

	for _, group := range identity.Groups {
		discard := false

		for _, action := range g.CompiledActions {
			if action.Discard != nil {
				if action.Discard.MatchString(group) {
					discard = true
					break
				}
			}

			if action.Replace != nil {
				group = action.Replace.Regexp.ReplaceAllString(group,
					action.Replace.With)
			}

			if action.StripPrefix != "" {
				group = strings.TrimPrefix(group, action.StripPrefix)
			}

			if action.AddPrefix != "" {
				group = action.AddPrefix + group
			}
		}

		if !discard && g.Unique {
			_, discard = groupSet[group]
			if !discard {
				groupSet[group] = exists
			}
		}

		if !discard {
			newGroups = append(newGroups, group)
		}
	}

	if g.Inject != nil {
		newGroups = append(newGroups, g.Inject...)
	}

	if g.Sorted {
		sort.Strings(newGroups)
	}

	identity.Groups = newGroups

	return identity, nil
}
