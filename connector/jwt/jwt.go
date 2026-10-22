// Package jwt implements a connector which authenticates users from a JWT
// bearer token passed in an HTTP header by an upstream proxy or gateway.
//
// Unlike the authproxy connector, which trusts plain headers, this connector
// cryptographically verifies the token against the upstream provider's JWKS
// and always enforces the configured issuer. Unlike the oidc connector, it
// performs no redirect to the upstream provider: the token is expected to
// already be present on the request.
package jwt

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"net/url"
	"regexp"
	"slices"
	"strings"

	"github.com/coreos/go-oidc/v3/oidc"

	"github.com/dexidp/dex/connector"
	groups_pkg "github.com/dexidp/dex/pkg/groups"
	"github.com/dexidp/dex/pkg/httpclient"
)

// Config holds the configuration parameters for the JWT bearer token connector.
type Config struct {
	// Issuer is the expected "iss" claim of incoming tokens. It is always
	// required and always enforced. When JWKSURL is empty, it is also used to
	// discover the JWKS endpoint via OIDC discovery.
	Issuer string `json:"issuer"`

	// JWKSURL overrides key retrieval only. Issuer validation is performed
	// regardless of this setting.
	JWKSURL string `json:"jwksURL"`

	// Audiences is the list of accepted "aud" values. A token is accepted when
	// at least one of its audiences matches. When empty, "aud" is not checked.
	Audiences []string `json:"audiences"`

	// SupportedSigningAlgs restricts the algorithms accepted for the token
	// signature. Defaults to RS256.
	SupportedSigningAlgs []string `json:"supportedSigningAlgs"`

	// Header is the HTTP header carrying the token. Defaults to Authorization.
	// An optional "Bearer " prefix is stripped case-insensitively.
	Header string `json:"header"`

	// Certificates for SSL validation when fetching the JWKS.
	RootCAs []string `json:"rootCAs"`

	// InsecureSkipVerify disables certificate verification when fetching the JWKS.
	InsecureSkipVerify bool `json:"insecureSkipVerify"`

	// UserIDKey is the claim used as the user ID. Defaults to "sub".
	UserIDKey string `json:"userIDKey"`

	// UserNameKey is the claim used as the username. Defaults to "name".
	UserNameKey string `json:"userNameKey"`

	// InsecureSkipEmailVerified overrides the "email_verified" claim with true.
	InsecureSkipEmailVerified bool `json:"insecureSkipEmailVerified"`

	// AllowedGroups restricts logins to users holding at least one of these
	// groups. The identity's groups are reduced to the matching set.
	AllowedGroups []string `json:"allowedGroups"`

	// ClaimMapping mirrors the oidc connector so configuration can be moved
	// between the two connectors.
	ClaimMapping struct {
		// Configurable key which contains the preferred username claim.
		PreferredUsernameKey string `json:"preferred_username"` // defaults to "preferred_username"

		// Configurable key which contains the email claim.
		EmailKey string `json:"email"` // defaults to "email"

		// Configurable key which contains the groups claim.
		GroupsKey string `json:"groups"` // defaults to "groups"
	} `json:"claimMapping"`

	// ClaimMutations holds all claim mutation options.
	ClaimMutations struct {
		NewGroupFromClaims []NewGroupFromClaims `json:"newGroupFromClaims"`
		FilterGroupClaims  FilterGroupClaims    `json:"filterGroupClaims"`
		ModifyGroupNames   ModifyGroupNames     `json:"modifyGroupNames"`
	} `json:"claimModifications"`

	// RoleMapping binds claim values to additional groups. The resulting group
	// names are meant to be used as-is by downstream authorization systems such
	// as Argo CD's RBAC, so they are added after filtering and after the
	// prefix/suffix of ModifyGroupNames have been applied.
	RoleMapping []RoleMapping `json:"roleMapping"`
}

// NewGroupFromClaims creates a new group from a list of claims and appends it to the list of existing groups.
type NewGroupFromClaims struct {
	// List of claim to join together
	Claims []string `json:"claims"`

	// String to separate the claims
	Delimiter string `json:"delimiter"`

	// Should Dex remove the Delimiter string from claim values
	// This is done to keep resulting claim structure in full control of the Dex operator
	ClearDelimiter bool `json:"clearDelimiter"`

	// String to place before the first claim
	Prefix string `json:"prefix"`
}

// FilterGroupClaims is a regex filter to keep only the matching groups.
type FilterGroupClaims struct {
	GroupsFilter string `json:"groupsFilter"`
}

// ModifyGroupNames allows to modify the group claims by adding a prefix and/or suffix to each group.
type ModifyGroupNames struct {
	Prefix string `json:"prefix"`
	Suffix string `json:"suffix"`
}

// RoleMapping binds the values of a single claim to a set of groups. Exactly
// one of Value or ValueRegex must be set.
type RoleMapping struct {
	// Claim is the name of the claim to inspect. String and array-of-string
	// claims are supported.
	Claim string `json:"claim"`

	// Value matches a claim value exactly.
	Value string `json:"value"`

	// ValueRegex matches a claim value against a regular expression.
	ValueRegex string `json:"valueRegex"`

	// Groups are appended to the identity when the claim matches.
	Groups []string `json:"groups"`
}

type roleMapping struct {
	claim  string
	value  string
	regex  *regexp.Regexp
	groups []string
}

// Open returns a connector which authenticates users from an upstream JWT.
func (c *Config) Open(id string, logger *slog.Logger) (connector.Connector, error) {
	if c.Issuer == "" {
		return nil, errors.New("jwt: issuer is required")
	}

	roleMappings, err := parseRoleMappings(c.RoleMapping)
	if err != nil {
		return nil, err
	}

	var groupsFilter *regexp.Regexp
	if c.ClaimMutations.FilterGroupClaims.GroupsFilter != "" {
		groupsFilter, err = regexp.Compile(c.ClaimMutations.FilterGroupClaims.GroupsFilter)
		if err != nil {
			return nil, fmt.Errorf("jwt: invalid groupsFilter regex %q: %v", c.ClaimMutations.FilterGroupClaims.GroupsFilter, err)
		}
	}

	httpClient, err := httpclient.NewHTTPClient(c.RootCAs, c.InsecureSkipVerify)
	if err != nil {
		return nil, err
	}

	bgctx, cancel := context.WithCancel(context.Background())
	ctx := oidc.ClientContext(bgctx, httpClient)

	keySet, err := newKeySet(ctx, c.Issuer, c.JWKSURL)
	if err != nil {
		cancel()
		return nil, err
	}

	verifier := oidc.NewVerifier(c.Issuer, keySet, &oidc.Config{
		// Audiences are verified against the full configured list below, which
		// the single-valued ClientID check cannot express.
		SkipClientIDCheck:    true,
		SupportedSigningAlgs: c.SupportedSigningAlgs,
	})

	header := c.Header
	if header == "" {
		header = "Authorization"
	}

	return &jwtConnector{
		verifier:                  verifier,
		cancel:                    cancel,
		header:                    header,
		audiences:                 c.Audiences,
		insecureSkipEmailVerified: c.InsecureSkipEmailVerified,
		allowedGroups:             c.AllowedGroups,
		userIDKey:                 c.UserIDKey,
		userNameKey:               c.UserNameKey,
		preferredUsernameKey:      c.ClaimMapping.PreferredUsernameKey,
		emailKey:                  c.ClaimMapping.EmailKey,
		groupsKey:                 c.ClaimMapping.GroupsKey,
		newGroupFromClaims:        c.ClaimMutations.NewGroupFromClaims,
		groupsFilter:              groupsFilter,
		groupsPrefix:              c.ClaimMutations.ModifyGroupNames.Prefix,
		groupsSuffix:              c.ClaimMutations.ModifyGroupNames.Suffix,
		roleMappings:              roleMappings,
		logger:                    logger.With(slog.Group("connector", "type", "jwt", "id", id)),
		pathSuffix:                "/" + id,
	}, nil
}

func parseRoleMappings(mappings []RoleMapping) ([]roleMapping, error) {
	parsed := make([]roleMapping, 0, len(mappings))
	for i, m := range mappings {
		if m.Claim == "" {
			return nil, fmt.Errorf("jwt: roleMapping[%d]: claim is required", i)
		}
		if m.Value == "" && m.ValueRegex == "" {
			return nil, fmt.Errorf("jwt: roleMapping[%d]: one of value or valueRegex is required", i)
		}
		if m.Value != "" && m.ValueRegex != "" {
			return nil, fmt.Errorf("jwt: roleMapping[%d]: value and valueRegex are mutually exclusive", i)
		}

		entry := roleMapping{claim: m.Claim, value: m.Value, groups: m.Groups}
		if m.ValueRegex != "" {
			re, err := regexp.Compile(m.ValueRegex)
			if err != nil {
				return nil, fmt.Errorf("jwt: roleMapping[%d]: invalid valueRegex %q: %v", i, m.ValueRegex, err)
			}
			entry.regex = re
		}
		parsed = append(parsed, entry)
	}
	return parsed, nil
}

// newKeySet returns a key set backed by the explicit JWKS URL when provided, or
// by the endpoint advertised through OIDC discovery on the issuer otherwise.
func newKeySet(ctx context.Context, issuer, jwksURL string) (oidc.KeySet, error) {
	if jwksURL == "" {
		provider, err := oidc.NewProvider(ctx, issuer)
		if err != nil {
			return nil, fmt.Errorf("jwt: failed to get provider: %v", err)
		}
		var metadata struct {
			JWKSURL string `json:"jwks_uri"`
		}
		if err := provider.Claims(&metadata); err != nil {
			return nil, fmt.Errorf("jwt: failed to extract provider discovery claims: %v", err)
		}
		if metadata.JWKSURL == "" {
			return nil, fmt.Errorf("jwt: provider %q does not advertise a jwks_uri, set jwksURL explicitly", issuer)
		}
		jwksURL = metadata.JWKSURL
	}

	if _, err := url.Parse(jwksURL); err != nil {
		return nil, fmt.Errorf("jwt: invalid jwksURL %q: %v", jwksURL, err)
	}
	return oidc.NewRemoteKeySet(ctx, jwksURL), nil
}

var _ connector.CallbackConnector = (*jwtConnector)(nil)

type jwtConnector struct {
	verifier                  *oidc.IDTokenVerifier
	cancel                    context.CancelFunc
	header                    string
	audiences                 []string
	insecureSkipEmailVerified bool
	allowedGroups             []string
	userIDKey                 string
	userNameKey               string
	preferredUsernameKey      string
	emailKey                  string
	groupsKey                 string
	newGroupFromClaims        []NewGroupFromClaims
	groupsFilter              *regexp.Regexp
	groupsPrefix              string
	groupsSuffix              string
	roleMappings              []roleMapping
	logger                    *slog.Logger
	pathSuffix                string
}

func (c *jwtConnector) Close() error {
	c.cancel()
	return nil
}

// LoginURL returns the connector's own callback URL. No upstream redirect is
// performed: the token is expected on the request itself.
func (c *jwtConnector) LoginURL(_ connector.Scopes, callbackURL, state string) (string, []byte, error) {
	u, err := url.Parse(callbackURL)
	if err != nil {
		return "", nil, fmt.Errorf("failed to parse callbackURL %q: %v", callbackURL, err)
	}
	u.Path += c.pathSuffix
	v := u.Query()
	v.Set("state", state)
	u.RawQuery = v.Encode()
	return u.String(), nil, nil
}

// HandleCallback verifies the upstream token and maps its claims to an identity.
func (c *jwtConnector) HandleCallback(_ connector.Scopes, _ []byte, r *http.Request) (connector.Identity, error) {
	rawToken := extractToken(r.Header.Get(c.header))
	if rawToken == "" {
		return connector.Identity{}, fmt.Errorf("jwt: required HTTP header %s is not set", c.header)
	}

	idToken, err := c.verifier.Verify(r.Context(), rawToken)
	if err != nil {
		return connector.Identity{}, fmt.Errorf("jwt: failed to verify token: %v", err)
	}

	if err := c.verifyAudience(idToken.Audience); err != nil {
		return connector.Identity{}, err
	}

	var claims map[string]any
	if err := idToken.Claims(&claims); err != nil {
		return connector.Identity{}, fmt.Errorf("jwt: failed to decode claims: %v", err)
	}

	return c.createIdentity(claims)
}

// extractToken strips an optional bearer prefix from the header value.
func extractToken(value string) string {
	value = strings.TrimSpace(value)
	if len(value) >= 7 && strings.EqualFold(value[:7], "bearer ") {
		value = strings.TrimSpace(value[7:])
	}
	return value
}

func (c *jwtConnector) verifyAudience(audiences []string) error {
	if len(c.audiences) == 0 {
		return nil
	}
	for _, aud := range audiences {
		if slices.Contains(c.audiences, aud) {
			return nil
		}
	}
	return fmt.Errorf("jwt: token audience %v does not match any of the expected audiences %v", audiences, c.audiences)
}

func (c *jwtConnector) createIdentity(claims map[string]any) (connector.Identity, error) {
	userIDKey := "sub"
	if c.userIDKey != "" {
		userIDKey = c.userIDKey
	}
	userID, found := claims[userIDKey].(string)
	if !found {
		return connector.Identity{}, fmt.Errorf("jwt: missing %q claim", userIDKey)
	}

	userNameKey := "name"
	if c.userNameKey != "" {
		userNameKey = c.userNameKey
	}
	username, found := claims[userNameKey].(string)
	if !found {
		return connector.Identity{}, fmt.Errorf("jwt: missing %q claim", userNameKey)
	}

	preferredUsernameKey := "preferred_username"
	if c.preferredUsernameKey != "" {
		preferredUsernameKey = c.preferredUsernameKey
	}
	preferredUsername, _ := claims[preferredUsernameKey].(string)

	emailKey := "email"
	if c.emailKey != "" {
		emailKey = c.emailKey
	}
	email, _ := claims[emailKey].(string)

	emailVerified, found := claims["email_verified"].(bool)
	switch {
	case c.insecureSkipEmailVerified:
		emailVerified = true
	case !found && email == "":
		// Nothing to verify when the token carries no email at all.
		emailVerified = false
	case !found:
		return connector.Identity{}, errors.New("jwt: missing \"email_verified\" claim")
	}

	groups, err := c.extractGroups(claims, userID)
	if err != nil {
		return connector.Identity{}, err
	}

	return connector.Identity{
		UserID:            userID,
		Username:          username,
		PreferredUsername: preferredUsername,
		Email:             email,
		EmailVerified:     emailVerified,
		Groups:            groups,
	}, nil
}

// extractGroups returns the identity's groups. userID is the identity's user
// ID (from userIDKey), reported when allowedGroups rejects the login.
func (c *jwtConnector) extractGroups(claims map[string]any, userID string) ([]string, error) {
	groupsKey := "groups"
	if c.groupsKey != "" {
		groupsKey = c.groupsKey
	}

	groups, err := claimValues(claims, groupsKey)
	if err != nil {
		return nil, fmt.Errorf("jwt: malformed %q claim", groupsKey)
	}

	if c.groupsFilter != nil {
		filtered := groups[:0]
		for _, group := range groups {
			if c.groupsFilter.MatchString(group) {
				filtered = append(filtered, group)
			}
		}
		groups = filtered
	}

	if len(c.allowedGroups) > 0 {
		groups = groups_pkg.Filter(groups, c.allowedGroups)
		if len(groups) == 0 {
			return nil, &connector.UserNotInRequiredGroupsError{
				UserID: userID,
				Groups: c.allowedGroups,
			}
		}
	}

	if c.groupsPrefix != "" || c.groupsSuffix != "" {
		for i, group := range groups {
			groups[i] = c.groupsPrefix + group + c.groupsSuffix
		}
	}

	groups = append(groups, newGroupsFromClaims(claims, c.newGroupFromClaims)...)
	groups = append(groups, c.mappedRoles(claims)...)

	return dedupe(groups), nil
}

func newGroupsFromClaims(claims map[string]any, configs []NewGroupFromClaims) []string {
	var groups []string
	for _, config := range configs {
		newGroupSegments := []string{config.Prefix}
		for _, claimName := range config.Claims {
			claimValue, ok := claims[claimName].(string)
			if !ok { // Non string claim values are ignored, concatenating them doesn't really make any sense.
				continue
			}

			if config.ClearDelimiter {
				// Removing the delimiter string from the concatenated claim ensures the
				// resulting claim structure is in full control of the Dex operator.
				claimValue = strings.ReplaceAll(claimValue, config.Delimiter, "")
			}

			newGroupSegments = append(newGroupSegments, claimValue)
		}

		if len(newGroupSegments) > 1 {
			groups = append(groups, strings.Join(newGroupSegments, config.Delimiter))
		}
	}
	return groups
}

// mappedRoles returns the groups bound to matching claim values by roleMapping.
func (c *jwtConnector) mappedRoles(claims map[string]any) []string {
	var groups []string
	for _, mapping := range c.roleMappings {
		values, err := claimValues(claims, mapping.claim)
		if err != nil {
			c.logger.Warn("ignoring malformed claim in role mapping", "claim", mapping.claim)
			continue
		}
		for _, value := range values {
			if mapping.regex != nil {
				if !mapping.regex.MatchString(value) {
					continue
				}
			} else if value != mapping.value {
				continue
			}
			groups = append(groups, mapping.groups...)
			break
		}
	}
	return groups
}

// claimValues reads a claim that may hold either a single string or a list of
// strings. A missing claim yields no values.
func claimValues(claims map[string]any, key string) ([]string, error) {
	switch value := claims[key].(type) {
	case nil:
		return nil, nil
	case string:
		return []string{value}, nil
	case []any:
		values := make([]string, 0, len(value))
		for _, item := range value {
			switch item := item.(type) {
			case string:
				values = append(values, item)
			case map[string]any:
				name, ok := item["name"].(string)
				if !ok {
					return nil, fmt.Errorf("malformed claim %q", key)
				}
				values = append(values, name)
			default:
				return nil, fmt.Errorf("malformed claim %q", key)
			}
		}
		return values, nil
	default:
		return nil, fmt.Errorf("malformed claim %q", key)
	}
}

func dedupe(groups []string) []string {
	if len(groups) == 0 {
		return nil
	}
	seen := make(map[string]struct{}, len(groups))
	result := make([]string, 0, len(groups))
	for _, group := range groups {
		if _, ok := seen[group]; ok {
			continue
		}
		seen[group] = struct{}{}
		result = append(result, group)
	}
	return result
}
