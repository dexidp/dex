package grants

import (
	"context"
	"log/slog"
	"net/http"
	"net/url"
	"slices"
	"strings"
	"time"

	"github.com/prometheus/client_golang/prometheus"

	"github.com/dexidp/dex/server/internal"
	"github.com/dexidp/dex/server/oauth2"
	"github.com/dexidp/dex/server/session"
	"github.com/dexidp/dex/server/tokens"
	"github.com/dexidp/dex/storage"
)

// IDJAGConfig enables issuing Identity Assertion JWT Authorization Grants
// through token exchange (draft-ietf-oauth-identity-assertion-authz-grant).
type IDJAGConfig struct {
	// ValidFor is the lifetime of an issued ID-JAG.
	ValidFor time.Duration
	// Policies maps a client ID to the audiences and scopes it may request. A
	// client without a policy cannot obtain an ID-JAG.
	Policies map[string]IDJAGPolicy
}

// IDJAGPolicy lists the Resource Authorization Servers (audiences) a client may
// request an ID-JAG for, and the scopes it may request at them.
type IDJAGPolicy struct {
	AllowedAudiences []string
	AllowedScopes    []string
}

// IDJAGMetrics counts ID-JAG requests. A nil *IDJAGMetrics records nothing.
type IDJAGMetrics struct {
	requests         *prometheus.CounterVec
	policyRejections *prometheus.CounterVec
	scopeReductions  prometheus.Counter
}

// NewIDJAGMetrics creates the ID-JAG counters and registers them with reg.
func NewIDJAGMetrics(reg prometheus.Registerer) *IDJAGMetrics {
	m := &IDJAGMetrics{
		requests: prometheus.NewCounterVec(prometheus.CounterOpts{
			Name: "dex_id_jag_requests_total",
			Help: "Total number of ID-JAG token exchange requests.",
		}, []string{"result"}),
		policyRejections: prometheus.NewCounterVec(prometheus.CounterOpts{
			Name: "dex_id_jag_policy_rejections_total",
			Help: "Total number of ID-JAG policy rejections by reason.",
		}, []string{"reason"}),
		scopeReductions: prometheus.NewCounter(prometheus.CounterOpts{
			Name: "dex_id_jag_scope_modifications_total",
			Help: "Total number of ID-JAG requests where policy reduced the requested scopes.",
		}),
	}
	reg.MustRegister(m.requests, m.policyRejections, m.scopeReductions)
	return m
}

// ID-JAG policy denial reasons, used in logs and the policy rejection metric.
const (
	idJAGUnknownAudience    = "unknown_audience"
	idJAGClientHasNoPolicy  = "client_has_no_policy"
	idJAGAudienceNotAllowed = "audience_not_allowed"
	idJAGNoScopeAllowed     = "no_scope_allowed"
)

// idJAG issues ID-JAGs for token-exchange requests with requested_token_type
// urn:ietf:params:oauth:token-type:id-jag.
type idJAG struct {
	config   IDJAGConfig
	issuer   *tokens.Issuer
	sessions *session.Manager
	metrics  *IDJAGMetrics
	logger   *slog.Logger
}

func newIDJAG(config *IDJAGConfig, issuer *tokens.Issuer, sessions *session.Manager, metrics *IDJAGMetrics, logger *slog.Logger) *idJAG {
	if config == nil {
		return nil
	}
	g := &idJAG{config: *config, issuer: issuer, sessions: sessions, metrics: metrics, logger: logger}
	if g.config.ValidFor == 0 {
		g.config.ValidFor = 5 * time.Minute
	}
	return g
}

// Authorize verifies the subject token, an ID token this server issued to the
// client, applies the client's policy and issues the ID-JAG.
func (g *idJAG) Authorize(ctx context.Context, req *Request, client storage.Client) (Responder, error) {
	logAttrs := []any{
		"client_id", client.ID, "connector_id", req.ConnectorID,
		"audience", req.Audience, "resource", req.Resource, "requested_scope", req.Scopes,
	}
	reject := func(reason string, oerr *oauth2.Error, attrs ...any) error {
		g.logger.InfoContext(ctx, "ID-JAG token exchange rejected", append(append(logAttrs, "reason", reason), attrs...)...)
		g.metrics.reject()
		return oerr
	}
	deny := func(reason string, oerr *oauth2.Error) error {
		g.logger.InfoContext(ctx, "ID-JAG token exchange denied by policy", append(logAttrs, "reason", reason)...)
		g.metrics.deny(reason)
		return oerr
	}

	// A public client, or one without a secret, has not authenticated, which
	// the draft requires.
	if client.Public || client.Secret == "" {
		return nil, reject("public_client", &oauth2.Error{Type: oauth2.InvalidClient, Description: "Client authentication is required.", Status: http.StatusUnauthorized})
	}
	if req.ConnectorID == "" {
		return nil, reject("missing_connector_id", &oauth2.Error{Type: oauth2.InvalidRequest, Description: "Missing connector_id.", Status: http.StatusBadRequest})
	}
	if len(req.Audience) != 1 || req.Audience[0] == "" {
		return nil, reject("invalid_audience", &oauth2.Error{Type: oauth2.InvalidRequest, Description: "Exactly one audience is required.", Status: http.StatusBadRequest})
	}
	if len(req.Resource) > 1 {
		return nil, reject("invalid_resource", &oauth2.Error{Type: oauth2.InvalidRequest, Description: "At most one resource is supported.", Status: http.StatusBadRequest})
	}
	if len(req.Resource) == 1 && !validResource(req.Resource[0]) {
		return nil, reject("invalid_resource", &oauth2.Error{Type: oauth2.InvalidTarget, Description: "resource must be an absolute URI without a fragment.", Status: http.StatusBadRequest})
	}
	if req.SubjectTokenType != oauth2.TokenTypeID {
		return nil, reject("invalid_subject_token_type", &oauth2.Error{Type: oauth2.RequestNotSupported, Description: "Invalid subject_token_type.", Status: http.StatusBadRequest})
	}

	idToken, err := g.issuer.VerifyIDToken(ctx, req.SubjectToken, client.ID)
	if err != nil {
		return nil, reject("invalid_subject_token", &oauth2.Error{Type: oauth2.InvalidRequest, Description: "Invalid subject_token.", Status: http.StatusBadRequest}, "err", err)
	}
	var subject internal.IDTokenSubject
	if err := internal.Unmarshal(idToken.Subject, &subject); err != nil || subject.ConnId != req.ConnectorID {
		return nil, reject("connector_mismatch", &oauth2.Error{Type: oauth2.InvalidRequest, Description: "subject_token was not issued through the requested connector.", Status: http.StatusBadRequest})
	}
	logAttrs = append(logAttrs, "sub", idToken.Subject)

	var subjectClaims struct {
		Email         string `json:"email"`
		EmailVerified bool   `json:"email_verified"`
		SessionID     string `json:"sid"`
	}
	if err := idToken.Claims(&subjectClaims); err != nil {
		g.logger.ErrorContext(ctx, "failed to decode subject token claims", "err", err)
		return nil, &oauth2.Error{Type: oauth2.ServerError, Status: http.StatusInternalServerError}
	}
	// Same rule as introspection: ending the session revokes the tokens of a
	// client bound to it.
	if subjectClaims.SessionID != "" && client.RefreshBoundToSession() && !g.sessions.Alive(ctx, subjectClaims.SessionID) {
		return nil, reject("session_ended", &oauth2.Error{Type: oauth2.InvalidRequest, Description: "Invalid subject_token.", Status: http.StatusBadRequest})
	}

	audience := req.Audience[0]
	scopes, reason := g.config.evaluate(client.ID, audience, req.Scopes)
	switch reason {
	case "":
	case idJAGUnknownAudience:
		return nil, deny(reason, &oauth2.Error{Type: oauth2.InvalidTarget, Status: http.StatusBadRequest})
	case idJAGNoScopeAllowed:
		return nil, deny(reason, &oauth2.Error{Type: oauth2.InvalidScope, Status: http.StatusBadRequest})
	default:
		return nil, deny(reason, &oauth2.Error{Type: oauth2.AccessDenied, Status: http.StatusBadRequest})
	}

	claims := tokens.IDJAGClaims{
		Subject:  idToken.Subject,
		Audience: audience,
		ClientID: client.ID,
		Scope:    strings.Join(scopes, " "),
	}
	if len(req.Resource) == 1 {
		claims.Resource = req.Resource[0]
	}
	// The Resource AS may use email to resolve or provision the user, so only a
	// verified one is passed on.
	if subjectClaims.EmailVerified {
		claims.Email = subjectClaims.Email
	}

	token, claims, err := g.issuer.SignIDJAG(ctx, claims, g.config.ValidFor)
	if err != nil {
		g.logger.ErrorContext(ctx, "failed to sign ID-JAG", "err", err)
		return nil, &oauth2.Error{Type: oauth2.ServerError, Status: http.StatusInternalServerError}
	}

	scopesReduced := claims.Scope != strings.Join(req.Scopes, " ")
	g.logger.InfoContext(ctx, "ID-JAG issued", append(logAttrs, "granted_scope", scopes, "jti", claims.JWTID)...)
	g.metrics.issue(scopesReduced)

	resp := tokens.Response{
		AccessToken:     token,
		IssuedTokenType: oauth2.TokenTypeIDJAG,
		TokenType:       "N_A",
		ExpiresIn:       int(g.config.ValidFor.Seconds()),
	}
	if scopesReduced {
		resp.Scope = claims.Scope
	}
	return resp, nil
}

// evaluate returns the scopes the client may obtain for audience, or the reason
// the request is denied. An audience that no policy lists is unknown to dex,
// which the draft reports differently from a known one the client may not use.
func (c IDJAGConfig) evaluate(clientID, audience string, scopes []string) ([]string, string) {
	known := false
	for _, p := range c.Policies {
		if slices.Contains(p.AllowedAudiences, audience) {
			known = true
			break
		}
	}
	if !known {
		return nil, idJAGUnknownAudience
	}

	policy, ok := c.Policies[clientID]
	if !ok {
		return nil, idJAGClientHasNoPolicy
	}
	if !slices.Contains(policy.AllowedAudiences, audience) {
		return nil, idJAGAudienceNotAllowed
	}

	var granted []string
	for _, scope := range scopes {
		if slices.Contains(policy.AllowedScopes, scope) && !slices.Contains(granted, scope) {
			granted = append(granted, scope)
		}
	}
	if len(scopes) > 0 && len(granted) == 0 {
		return nil, idJAGNoScopeAllowed
	}
	return granted, ""
}

// validResource reports whether resource is an absolute URI without a fragment
// (RFC 8707 §2).
func validResource(resource string) bool {
	u, err := url.Parse(resource)
	return err == nil && u.IsAbs() && u.Fragment == "" && !strings.Contains(resource, "#")
}

func (m *IDJAGMetrics) reject() {
	if m == nil {
		return
	}
	m.requests.WithLabelValues("rejected").Inc()
}

func (m *IDJAGMetrics) deny(reason string) {
	if m == nil {
		return
	}
	m.requests.WithLabelValues("denied").Inc()
	m.policyRejections.WithLabelValues(reason).Inc()
}

func (m *IDJAGMetrics) issue(scopesReduced bool) {
	if m == nil {
		return
	}
	m.requests.WithLabelValues("issued").Inc()
	if scopesReduced {
		m.scopeReductions.Inc()
	}
}
