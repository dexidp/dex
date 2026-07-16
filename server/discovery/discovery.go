package discovery

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"
	"sort"
	"strconv"
	"sync"
	"time"

	jose "github.com/go-jose/go-jose/v4"

	"github.com/dexidp/dex/server/router"
	"github.com/dexidp/dex/server/signer"
)

// Handler serves the discovery document and the JWKS. AbsURL builds an absolute
// URL under the issuer; RenderError renders an HTML error page. Both are
// supplied by the server so the handler does not depend on the whole Server.
type Handler struct {
	Issuer          string
	AbsURL          func(...string) string
	RenderError     func(*http.Request, http.ResponseWriter, int, string)
	Signer          signer.Signer
	Logger          *slog.Logger
	ResponseTypes   map[string]bool
	GrantTypes      []string
	PKCEMethods     []string
	SessionsEnabled bool

	docOnce sync.Once
	docData []byte
	docErr  error
}

// Mount registers the discovery routes.
func (h *Handler) Mount(m router.Mux) {
	m.HandleCORS("/.well-known/openid-configuration", h.serveDocument)
	m.HandleCORS("/keys", h.Keys)
}

// Document is the OIDC discovery document.
type Document struct {
	Issuer            string   `json:"issuer"`
	Auth              string   `json:"authorization_endpoint"`
	Token             string   `json:"token_endpoint"`
	Keys              string   `json:"jwks_uri"`
	UserInfo          string   `json:"userinfo_endpoint"`
	DeviceEndpoint    string   `json:"device_authorization_endpoint"`
	Introspect        string   `json:"introspection_endpoint"`
	EndSession        string   `json:"end_session_endpoint,omitempty"`
	GrantTypes        []string `json:"grant_types_supported"`
	ResponseTypes     []string `json:"response_types_supported"`
	Subjects          []string `json:"subject_types_supported"`
	IDTokenAlgs       []string `json:"id_token_signing_alg_values_supported"`
	CodeChallengeAlgs []string `json:"code_challenge_methods_supported"`
	Scopes            []string `json:"scopes_supported"`
	AuthMethods       []string `json:"token_endpoint_auth_methods_supported"`
	Claims            []string `json:"claims_supported"`
}

// Keys serves the JSON Web Key Set.
func (h *Handler) Keys(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	// TODO(ericchiang): Cache this.
	keys, err := h.Signer.ValidationKeys(ctx)
	if err != nil {
		h.Logger.ErrorContext(ctx, "failed to get keys", "err", err)
		h.RenderError(r, w, http.StatusInternalServerError, "Internal server error.")
		return
	}

	if len(keys) == 0 {
		h.Logger.ErrorContext(ctx, "no public keys found.")
		h.RenderError(r, w, http.StatusInternalServerError, "Internal server error.")
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
		h.RenderError(r, w, http.StatusInternalServerError, "Internal server error.")
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

// serveDocument serves the discovery document, marshaling it once on first use.
func (h *Handler) serveDocument(w http.ResponseWriter, r *http.Request) {
	h.docOnce.Do(func() {
		h.docData, h.docErr = json.MarshalIndent(h.Construct(r.Context()), "", "  ")
	})
	if h.docErr != nil {
		h.Logger.ErrorContext(r.Context(), "failed to marshal discovery data", "err", h.docErr)
		h.RenderError(r, w, http.StatusInternalServerError, "Internal server error.")
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Content-Length", strconv.Itoa(len(h.docData)))
	w.Write(h.docData)
}

// Construct builds the discovery document from the current configuration.
func (h *Handler) Construct(ctx context.Context) Document {
	d := Document{
		Issuer:            h.Issuer,
		Auth:              h.AbsURL("/auth"),
		Token:             h.AbsURL("/token"),
		Keys:              h.AbsURL("/keys"),
		UserInfo:          h.AbsURL("/userinfo"),
		DeviceEndpoint:    h.AbsURL("/device/code"),
		Introspect:        h.AbsURL("/token/introspect"),
		Subjects:          []string{"public"},
		IDTokenAlgs:       []string{string(jose.RS256)},
		CodeChallengeAlgs: h.PKCEMethods,
		Scopes:            []string{"openid", "email", "groups", "profile", "offline_access"},
		AuthMethods:       []string{"client_secret_basic", "client_secret_post"},
		Claims: []string{
			"iss", "sub", "aud", "iat", "exp", "email", "email_verified",
			"locale", "name", "preferred_username", "at_hash",
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
		d.EndSession = h.AbsURL("/logout")
	}

	return d
}
