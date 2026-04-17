// Package gcloudiap implements a connector which validates Google Cloud
// Identity-Aware Proxy (IAP) JWTs and optionally retrieves Google Workspace
// group membership via the Admin Directory API using workload identity
// (Application Default Credentials) — no domain-wide delegation required.
package gcloudiap

import (
	"context"
	"fmt"
	"log/slog"
	"net/http"
	"net/url"
	"path"
	"strings"

	"github.com/coreos/go-oidc/v3/oidc"
	admin "google.golang.org/api/admin/directory/v1"

	"github.com/dexidp/dex/connector"
)

const (
	// defaultIAPIssuer is the issuer claim present in IAP-signed JWTs.
	defaultIAPIssuer = "https://cloud.google.com/iap"

	// defaultIAPJWKSUrl is the public JWKS endpoint for IAP JWT verification.
	defaultIAPJWKSUrl = "https://www.gstatic.com/iap/verify/public_key-jwk"

	// iapJWTHeader is the HTTP header that IAP sets on every proxied request.
	iapJWTHeader = "X-Goog-IAP-JWT-Assertion"
)

// Config holds the configuration parameters for the gcloud-iap connector.
type Config struct {
	// Audience is the IAP backend-service audience string.
	// Format: /projects/<project-number>/global/backendServices/<service-id>
	// This field is required.
	Audience string `json:"audience"`

	// IAPIssuer is the expected issuer of IAP JWTs.
	// Defaults to https://cloud.google.com/iap
	IAPIssuer string `json:"iapIssuer"`

	// IAPJWKSUrl is the URL of the IAP public JWKS endpoint used to verify JWT signatures.
	// Defaults to https://www.gstatic.com/iap/verify/public_key-jwk
	IAPJWKSUrl string `json:"iapJWKSUrl"`

	// Domain scopes the Admin Directory API group lookup to a single Google
	// Workspace primary domain (e.g. "example.com"). Use this when your
	// Workspace organisation has a single domain or when you only want groups
	// from one specific domain.
	//
	// Mutually exclusive with CustomerID. Exactly one of Domain or CustomerID
	// must be set when GroupsFilter is non-empty.
	Domain string `json:"domain"`

	// CustomerID scopes the Admin Directory API group lookup to all domains
	// belonging to a Google Workspace customer account. Use this for
	// multi-domain Workspace organisations. The value is the numeric Workspace
	// customer ID (e.g. "C01abc123"), visible in the Admin console under
	// Account → Account settings.
	//
	// Note: the "my_customer" alias is intentionally NOT supported here because
	// the workload-identity service account is not itself a Workspace member
	// and the alias does not resolve correctly in that context.
	//
	// Mutually exclusive with Domain. Exactly one of Domain or CustomerID
	// must be set when GroupsFilter is non-empty.
	CustomerID string `json:"customerID"`

	// GroupsFilter is an optional list of glob patterns used to select which of
	// the user's Workspace groups are included in the identity. When non-empty,
	// the connector fetches the user's group membership from the Admin Directory
	// API and returns only those groups whose email address matches at least one
	// pattern. If no group matches after filtering, the login is denied.
	//
	// Patterns use standard shell glob syntax where '*' matches any sequence of
	// non-separator characters. Because group email addresses never contain '/',
	// '*' effectively matches any substring. Matching is case-insensitive.
	// Examples:
	//
	//   "*"                   – include all groups (fetch everything, no restriction)
	//   "*@example.com"       – all groups in the example.com domain
	//   "group-*@example.com" – groups whose local part starts with "group-"
	//   "sre@example.com"     – exact match
	//
	// When GroupsFilter is empty, no Admin Directory API call is made and the
	// Groups field of the returned identity will always be empty, regardless of
	// whether the downstream client requested the groups scope.
	//
	// Requires either Domain or CustomerID to be set.
	GroupsFilter []string `json:"groupsFilter"`

	// FetchTransitiveGroupMembership controls whether to recursively resolve
	// transitive group memberships in addition to direct ones.
	FetchTransitiveGroupMembership bool `json:"fetchTransitiveGroupMembership"`
}

// Open returns a connector which validates IAP JWTs and optionally resolves
// group memberships.
func (c *Config) Open(id string, logger *slog.Logger) (connector.Connector, error) {
	if c.Audience == "" {
		return nil, fmt.Errorf("gcloud-iap: audience is required (format: /projects/<number>/global/backendServices/<id>)")
	}

	if len(c.GroupsFilter) > 0 {
		if c.Domain == "" && c.CustomerID == "" {
			return nil, fmt.Errorf("gcloud-iap: either domain or customerID must be set when groupsFilter is configured")
		}
		if c.Domain != "" && c.CustomerID != "" {
			return nil, fmt.Errorf("gcloud-iap: domain and customerID are mutually exclusive, set only one")
		}
		// Validate all patterns at startup so misconfiguration is caught early
		// rather than at login time.
		for _, pattern := range c.GroupsFilter {
			if _, err := path.Match(pattern, ""); err != nil {
				return nil, fmt.Errorf("gcloud-iap: invalid groupsFilter pattern %q: %v", pattern, err)
			}
		}
	}

	issuer := c.IAPIssuer
	if issuer == "" {
		issuer = defaultIAPIssuer
	}

	jwksURL := c.IAPJWKSUrl
	if jwksURL == "" {
		jwksURL = defaultIAPJWKSUrl
	}

	ctx, cancel := context.WithCancel(context.Background())

	keySet := oidc.NewRemoteKeySet(ctx, jwksURL)
	verifier := oidc.NewVerifier(issuer, keySet, &oidc.Config{
		ClientID:             c.Audience,
		SupportedSigningAlgs: []string{oidc.ES256},
	})

	conn := &iapConnector{
		verifier:                       verifier,
		domain:                         c.Domain,
		customerID:                     c.CustomerID,
		groupsFilter:                   c.GroupsFilter,
		fetchTransitiveGroupMembership: c.FetchTransitiveGroupMembership,
		logger:                         logger.With(slog.Group("connector", "type", "gcloud-iap", "id", id)),
		cancel:                         cancel,
		pathSuffix:                     "/" + id,
	}

	if len(c.GroupsFilter) > 0 {
		srv, err := admin.NewService(ctx)
		if err != nil {
			cancel()
			return nil, fmt.Errorf("gcloud-iap: failed to create Admin Directory service (ensure workload identity or ADC is configured): %v", err)
		}
		conn.adminSrv = srv
	}

	return conn, nil
}

var _ connector.CallbackConnector = (*iapConnector)(nil)

type iapConnector struct {
	verifier                       *oidc.IDTokenVerifier
	adminSrv                       *admin.Service
	domain                         string
	customerID                     string
	groupsFilter                   []string
	fetchTransitiveGroupMembership bool
	logger                         *slog.Logger
	cancel                         context.CancelFunc
	pathSuffix                     string
}

func (c *iapConnector) Close() error {
	c.cancel()
	return nil
}

// LoginURL returns the URL to redirect the user to. Since IAP injects the JWT
// on every request, we redirect straight back to dex's own callback URL
// (the same pattern used by the authproxy connector).
func (c *iapConnector) LoginURL(s connector.Scopes, callbackURL, state string) (string, []byte, error) {
	u, err := url.Parse(callbackURL)
	if err != nil {
		return "", nil, fmt.Errorf("gcloud-iap: failed to parse callbackURL %q: %v", callbackURL, err)
	}
	u.Path += c.pathSuffix
	v := u.Query()
	v.Set("state", state)
	u.RawQuery = v.Encode()
	return u.String(), nil, nil
}

// HandleCallback validates the IAP JWT from the request header and returns the
// user's identity, optionally enriched with group membership.
func (c *iapConnector) HandleCallback(s connector.Scopes, _ []byte, r *http.Request) (connector.Identity, error) {
	rawJWT := r.Header.Get(iapJWTHeader)
	if rawJWT == "" {
		return connector.Identity{}, fmt.Errorf("gcloud-iap: missing required header %s", iapJWTHeader)
	}

	idToken, err := c.verifier.Verify(r.Context(), rawJWT)
	if err != nil {
		return connector.Identity{}, fmt.Errorf("gcloud-iap: failed to verify IAP JWT: %v", err)
	}

	var claims struct {
		Email string `json:"email"`
	}
	if err = idToken.Claims(&claims); err != nil {
		return connector.Identity{}, fmt.Errorf("gcloud-iap: failed to decode JWT claims: %v", err)
	}

	// Group lookup only happens when groupsFilter is configured. An empty
	// groupsFilter means the operator has not enabled group resolution at all —
	// no Admin Directory API call is made and the groups scope is ignored.
	var groups []string
	if c.adminSrv != nil {
		checkedGroups := make(map[string]struct{})
		groups, err = c.getGroups(claims.Email, c.fetchTransitiveGroupMembership, checkedGroups)
		if err != nil {
			return connector.Identity{}, fmt.Errorf("gcloud-iap: could not retrieve groups: %v", err)
		}

		groups = filterGroups(groups, c.groupsFilter)
		if len(groups) == 0 {
			return connector.Identity{}, fmt.Errorf("gcloud-iap: user %q does not belong to any group matching the configured groupsFilter", claims.Email)
		}
	}

	return connector.Identity{
		UserID:        idToken.Subject,
		Username:      claims.Email,
		Email:         claims.Email,
		EmailVerified: true, // a cryptographically verified IAP JWT guarantees the email is authenticated
		Groups:        groups,
	}, nil
}

// filterGroups returns the subset of groups whose email matches at least one of
// the provided glob patterns. Matching is case-insensitive; patterns follow
// path.Match syntax. A bare "*" short-circuits and returns all groups as-is.
func filterGroups(groups, patterns []string) []string {
	for _, p := range patterns {
		if p == "*" {
			return groups
		}
	}

	var matched []string
	for _, group := range groups {
		lower := strings.ToLower(group)
		for _, pattern := range patterns {
			// path.Match errors are impossible here: patterns were validated in Open.
			ok, _ := path.Match(strings.ToLower(pattern), lower)
			if ok {
				matched = append(matched, group)
				break
			}
		}
	}
	return matched
}

// getGroups lists all Google Workspace groups the given email is a member of,
// returning each group's email address. When fetchTransitive is true, group
// memberships are resolved recursively.
func (c *iapConnector) getGroups(email string, fetchTransitive bool, checkedGroups map[string]struct{}) ([]string, error) {
	var userGroups []string
	groupsList := &admin.Groups{}

	for {
		var err error
		req := c.adminSrv.Groups.List().
			UserKey(email).PageToken(groupsList.NextPageToken)
		if c.customerID != "" {
			req = req.Customer(c.customerID)
		} else {
			req = req.Domain(c.domain)
		}
		groupsList, err = req.Do()
		if err != nil {
			return nil, fmt.Errorf("could not list groups: %v", err)
		}

		for _, group := range groupsList.Groups {
			if _, exists := checkedGroups[group.Email]; exists {
				continue
			}

			checkedGroups[group.Email] = struct{}{}
			userGroups = append(userGroups, group.Email)

			if !fetchTransitive {
				continue
			}

			transitiveGroups, err := c.getGroups(group.Email, fetchTransitive, checkedGroups)
			if err != nil {
				return nil, fmt.Errorf("could not list transitive groups: %v", err)
			}

			userGroups = append(userGroups, transitiveGroups...)
		}

		if groupsList.NextPageToken == "" {
			break
		}
	}

	return userGroups, nil
}
