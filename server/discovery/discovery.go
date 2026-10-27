package discovery

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"
	"sort"
	"strconv"
	"sync/atomic"
	"time"

	jose "github.com/go-jose/go-jose/v4"

	"github.com/dexidp/dex/server/oauth2"
	"github.com/dexidp/dex/server/router"
	"github.com/dexidp/dex/server/signer"
	"github.com/dexidp/dex/server/templates"
)

// Handler serves the discovery document and the JWKS. Like every other domain
// handler it takes the issuer URL and the templates directly, so it can be
// built without a Server.
type Handler struct {
	IssuerURL       oauth2.IssuerURL
	Templates       *templates.Templates
	Signer          signer.Signer
	Logger          *slog.Logger
	ResponseTypes   map[string]bool
	GrantTypes      []string
	PKCEMethods     []string
	SessionsEnabled bool

	// doc caches the marshaled discovery document. It is only filled once the
	// signing algorithm has been resolved, so a failed lookup is retried by the
	// next request instead of being served for the life of the process.
	doc atomic.Pointer[[]byte]
}

// renderError renders a user-facing HTML error page.
func (h *Handler) renderError(r *http.Request, w http.ResponseWriter, status int, description string) {
	templates.RenderError(h.Templates, h.Logger, r, w, status, description)
}

// Mount registers the discovery routes.
func (h *Handler) Mount(m router.Mux) {
	m.HandleCORS("/.well-known/openid-configuration", h.serveDocument)
	m.HandleCORS("/keys", h.Keys)
}

// Document is the OIDC discovery document.
type Document struct {
	Issuer         string `json:"issuer"`
	Auth           string `json:"authorization_endpoint"`
	Token          string `json:"token_endpoint"`
	Keys           string `json:"jwks_uri"`
	UserInfo       string `json:"userinfo_endpoint"`
	DeviceEndpoint string `json:"device_authorization_endpoint"`
	Introspect     string `json:"introspection_endpoint"`
	EndSession     string `json:"end_session_endpoint,omitempty"`
	// BackchannelLogout and BackchannelLogoutSession advertise OIDC Back-Channel
	// Logout 1.0. Both are omitted rather than sent as false when sessions are off,
	// matching how end_session_endpoint disappears with them.
	BackchannelLogout        bool     `json:"backchannel_logout_supported,omitempty"`
	BackchannelLogoutSession bool     `json:"backchannel_logout_session_supported,omitempty"`
	GrantTypes               []string `json:"grant_types_supported"`
	ResponseTypes            []string `json:"response_types_supported"`
	Subjects                 []string `json:"subject_types_supported"`
	IDTokenAlgs              []string `json:"id_token_signing_alg_values_supported"`
	CodeChallengeAlgs        []string `json:"code_challenge_methods_supported"`
	Scopes                   []string `json:"scopes_supported"`
	AuthMethods              []string `json:"token_endpoint_auth_methods_supported"`
	Claims                   []string `json:"claims_supported"`
}

// Keys serves the JSON Web Key Set.
func (h *Handler) Keys(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	// TODO(ericchiang): Cache this.
	keys, err := h.Signer.ValidationKeys(ctx)
	if err != nil {
		h.Logger.ErrorContext(ctx, "failed to get keys", "err", err)
		h.renderError(r, w, http.StatusInternalServerError, "Internal server error.")
		return
	}

	if len(keys) == 0 {
		h.Logger.ErrorContext(ctx, "no public keys found.")
		h.renderError(r, w, http.StatusInternalServerError, "Internal server error.")
		return
	}

	jwks := jose.JSONWebKeySet{
		Keys: make([]jose.JSONWebKey, len(keys)),
	}
	for i, key := range keys {
		jwks.Keys[i] = *key
	}

	data, err := json.MarshalIndent(jwks, "", "  ")
	if err != nil {
		h.Logger.ErrorContext(ctx, "failed to marshal discovery data", "err", err)
		h.renderError(r, w, http.StatusInternalServerError, "Internal server error.")
		return
	}

	// We don't have NextRotation info from Signer interface easily,
	// so we'll just set a reasonable default cache time.
	maxAge := time.Minute * 10

	w.Header().Set("Cache-Control", fmt.Sprintf("max-age=%d, must-revalidate", int(maxAge.Seconds())))
	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Content-Length", strconv.Itoa(len(data)))
	w.Write(data)
}

// serveDocument serves the discovery document.
func (h *Handler) serveDocument(w http.ResponseWriter, r *http.Request) {
	data, err := h.document(r.Context())
	if err != nil {
		h.Logger.ErrorContext(r.Context(), "failed to marshal discovery data", "err", err)
		h.renderError(r, w, http.StatusInternalServerError, "Internal server error.")
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Content-Length", strconv.Itoa(len(data)))
	w.Write(data)
}

// document returns the marshaled discovery document, from the cache when it
// has been built successfully before.
func (h *Handler) document(ctx context.Context) ([]byte, error) {
	if data := h.doc.Load(); data != nil {
		return *data, nil
	}

	doc, algErr := h.construct(ctx)
	data, err := json.MarshalIndent(doc, "", "  ")
	if err != nil {
		return nil, err
	}
	// A document built after a failed signing algorithm lookup carries the RS256
	// fallback, which is wrong for an ES256 signer. Serve it for this request but
	// do not cache it.
	if algErr == nil {
		h.doc.Store(&data)
	}
	return data, nil
}

// Construct builds the discovery document from the current configuration.
func (h *Handler) Construct(ctx context.Context) Document {
	d, _ := h.construct(ctx)
	return d
}

// construct builds the discovery document. The returned error reports a failed
// signing algorithm lookup, in which case the document advertises RS256.
func (h *Handler) construct(ctx context.Context) (Document, error) {
	d := Document{
		Issuer:            h.IssuerURL.String(),
		Auth:              h.IssuerURL.AbsURL("/auth"),
		Token:             h.IssuerURL.AbsURL("/token"),
		Keys:              h.IssuerURL.AbsURL("/keys"),
		UserInfo:          h.IssuerURL.AbsURL("/userinfo"),
		DeviceEndpoint:    h.IssuerURL.AbsURL("/device/code"),
		Introspect:        h.IssuerURL.AbsURL("/token/introspect"),
		Subjects:          []string{"public"},
		IDTokenAlgs:       []string{string(jose.RS256)},
		CodeChallengeAlgs: h.PKCEMethods,
		Scopes:            []string{"openid", "email", "groups", "profile", "offline_access"},
		AuthMethods:       []string{"client_secret_basic", "client_secret_post"},
		Claims: []string{
			"iss", "sub", "aud", "iat", "exp", "email", "email_verified",
			"locale", "name", "preferred_username", "at_hash", "groups",
			"federated_claims",
		},
	}

	// Determine signing algorithm from signer.
	signingAlg, err := h.Signer.Algorithm(ctx)
	if err != nil {
		h.Logger.Error("failed to get signing algorithm", "err", err)
	} else {
		d.IDTokenAlgs = []string{string(signingAlg)}
	}

	for responseType := range h.ResponseTypes {
		d.ResponseTypes = append(d.ResponseTypes, responseType)
	}
	sort.Strings(d.ResponseTypes)

	d.GrantTypes = h.GrantTypes

	if h.SessionsEnabled {
		d.EndSession = h.IssuerURL.AbsURL("/logout")
		d.BackchannelLogout = true
		// Dex always puts a sid in its logout tokens, so clients never need to set
		// backchannel_logout_session_required to get one.
		d.BackchannelLogoutSession = true
		d.Claims = append(d.Claims, "sid")
	}

	return d, err
}
