package registration

import (
	"crypto/sha256"
	"crypto/subtle"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"mime"
	"net"
	"net/http"
	"net/url"
	"slices"
	"strings"
	"time"

	"github.com/dexidp/dex/server/oauth2"
	"github.com/dexidp/dex/server/router"
	"github.com/dexidp/dex/server/tokens"
	"github.com/dexidp/dex/storage"
)

const (
	maxRequestBodyBytes = 64 << 10
	maxRedirectURIs     = 20
	maxRedirectURIBytes = 2048
	maxClientNameBytes  = 256
	maxLogoURIBytes     = 2048
	maxMetadataValues   = 20
	maxScopeBytes       = 4096
)

const (
	errorInvalidRedirectURI    = "invalid_redirect_uri"
	errorInvalidClientMetadata = "invalid_client_metadata"
	errorInvalidToken          = "invalid_token"
)

// Handler serves the protected RFC 7591 client registration endpoint.
type Handler struct {
	Storage            storage.Storage
	Logger             *slog.Logger
	InitialAccessToken string
	GrantTypes         []string
	ResponseTypes      map[string]bool
	Now                func() time.Time
}

// Request is the client metadata Dex accepts during registration. Unknown
// metadata is ignored as required by RFC 7591 section 2.
type Request struct {
	RedirectURIs            []string `json:"redirect_uris"`
	ClientName              string   `json:"client_name,omitempty"`
	TokenEndpointAuthMethod string   `json:"token_endpoint_auth_method,omitempty"`
	GrantTypes              []string `json:"grant_types,omitempty"`
	ResponseTypes           []string `json:"response_types,omitempty"`
	Scope                   string   `json:"scope,omitempty"`
	LogoURI                 string   `json:"logo_uri,omitempty"`
}

// Response is an RFC 7591 client information response.
type Response struct {
	ClientID                string   `json:"client_id"`
	ClientSecret            string   `json:"client_secret,omitempty"`
	ClientIDIssuedAt        int64    `json:"client_id_issued_at,omitempty"`
	ClientSecretExpiresAt   *int64   `json:"client_secret_expires_at,omitempty"`
	ClientName              string   `json:"client_name,omitempty"`
	RedirectURIs            []string `json:"redirect_uris"`
	TokenEndpointAuthMethod string   `json:"token_endpoint_auth_method"`
	GrantTypes              []string `json:"grant_types"`
	ResponseTypes           []string `json:"response_types"`
	Scope                   string   `json:"scope,omitempty"`
	LogoURI                 string   `json:"logo_uri,omitempty"`
}

// Mount registers the dynamic client registration endpoint.
func (h *Handler) Mount(m router.Mux) {
	m.HandleFunc("/register", h.handle)
}

func (h *Handler) handle(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Cache-Control", "no-store")
	w.Header().Set("Pragma", "no-cache")
	if r.Method != http.MethodPost {
		w.Header().Set("Allow", http.MethodPost)
		h.writeError(w, oauth2.InvalidRequest, "method must be POST", http.StatusMethodNotAllowed)
		return
	}
	if !h.authenticated(r.Header.Get("Authorization")) {
		w.Header().Set("WWW-Authenticate", `Bearer error="invalid_token"`)
		h.writeError(w, errorInvalidToken, "invalid initial access token", http.StatusUnauthorized)
		return
	}

	mediaType, _, err := mime.ParseMediaType(r.Header.Get("Content-Type"))
	if err != nil || mediaType != "application/json" {
		h.writeError(w, oauth2.InvalidRequest, "content type must be application/json", http.StatusUnsupportedMediaType)
		return
	}

	r.Body = http.MaxBytesReader(w, r.Body, maxRequestBodyBytes)
	var req Request
	decoder := json.NewDecoder(r.Body)
	if err := decoder.Decode(&req); err != nil {
		var tooLarge *http.MaxBytesError
		if errors.As(err, &tooLarge) {
			h.writeError(w, oauth2.InvalidRequest, "request body is too large", http.StatusRequestEntityTooLarge)
			return
		}
		h.writeError(w, oauth2.InvalidRequest, "invalid JSON request body", http.StatusBadRequest)
		return
	}
	if err := rejectTrailingJSON(decoder); err != nil {
		h.writeError(w, oauth2.InvalidRequest, "request body must contain one JSON object", http.StatusBadRequest)
		return
	}

	if typ, description := h.normalizeAndValidate(&req); typ != "" {
		h.writeError(w, typ, description, http.StatusBadRequest)
		return
	}

	clientID := storage.NewID()
	isPublic := req.TokenEndpointAuthMethod == "none"
	clientSecret := ""
	if !isPublic {
		clientSecret = storage.NewID() + storage.NewID()
	}
	now := time.Now
	if h.Now != nil {
		now = h.Now
	}
	issuedAt := now().Unix()
	client := storage.Client{
		ID:                      clientID,
		Secret:                  clientSecret,
		RedirectURIs:            req.RedirectURIs,
		Name:                    req.ClientName,
		LogoURL:                 req.LogoURI,
		Public:                  isPublic,
		DynamicallyRegistered:   true,
		GrantTypes:              req.GrantTypes,
		ResponseTypes:           req.ResponseTypes,
		AllowedScopes:           strings.Fields(req.Scope),
		TokenEndpointAuthMethod: req.TokenEndpointAuthMethod,
		RegistrationTime:        issuedAt,
		RegistrationTokenID:     h.initialAccessTokenID(),
	}
	if err := h.Storage.CreateClient(r.Context(), client); err != nil {
		h.Logger.ErrorContext(r.Context(), "failed to dynamically register client", "err", err)
		h.writeError(w, oauth2.ServerError, "failed to register client", http.StatusInternalServerError)
		return
	}

	resp := Response{
		ClientID:                clientID,
		ClientSecret:            clientSecret,
		ClientIDIssuedAt:        issuedAt,
		ClientName:              req.ClientName,
		RedirectURIs:            req.RedirectURIs,
		TokenEndpointAuthMethod: req.TokenEndpointAuthMethod,
		GrantTypes:              req.GrantTypes,
		ResponseTypes:           req.ResponseTypes,
		Scope:                   req.Scope,
		LogoURI:                 req.LogoURI,
	}
	if !isPublic {
		neverExpires := int64(0)
		resp.ClientSecretExpiresAt = &neverExpires
	}
	h.Logger.InfoContext(r.Context(), "dynamically registered client",
		"client_id", clientID,
		"client_name", req.ClientName,
		"redirect_uri_count", len(req.RedirectURIs),
		"token_endpoint_auth_method", req.TokenEndpointAuthMethod,
		"grant_types", req.GrantTypes,
		"response_types", req.ResponseTypes,
		"scopes", strings.Fields(req.Scope),
		"registration_token_id", client.RegistrationTokenID,
	)

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	if err := json.NewEncoder(w).Encode(resp); err != nil {
		h.Logger.ErrorContext(r.Context(), "failed to encode client registration response", "err", err)
	}
}

func (h *Handler) initialAccessTokenID() string {
	digest := sha256.Sum256([]byte(h.InitialAccessToken))
	return fmt.Sprintf("sha256:%x", digest[:8])
}

func (h *Handler) authenticated(header string) bool {
	scheme, token, ok := strings.Cut(strings.TrimSpace(header), " ")
	if !ok || !strings.EqualFold(scheme, "Bearer") {
		return false
	}
	want := sha256.Sum256([]byte(h.InitialAccessToken))
	got := sha256.Sum256([]byte(strings.TrimSpace(token)))
	return subtle.ConstantTimeCompare(got[:], want[:]) == 1
}

func rejectTrailingJSON(decoder *json.Decoder) error {
	var extra any
	err := decoder.Decode(&extra)
	if errors.Is(err, io.EOF) {
		return nil
	}
	if err == nil {
		return errors.New("extra JSON value")
	}
	return err
}

func (h *Handler) normalizeAndValidate(req *Request) (string, string) {
	if len(req.ClientName) > maxClientNameBytes {
		return errorInvalidClientMetadata, "client_name is too long"
	}
	if len(req.Scope) > maxScopeBytes {
		return errorInvalidClientMetadata, "scope is too long"
	}
	if req.Scope == "" {
		req.Scope = tokens.ScopeOpenID
	}
	if len(req.GrantTypes) > maxMetadataValues || len(req.ResponseTypes) > maxMetadataValues {
		return errorInvalidClientMetadata, "too many grant_types or response_types"
	}
	if len(req.RedirectURIs) > maxRedirectURIs {
		return errorInvalidRedirectURI, "too many redirect_uris"
	}

	if req.TokenEndpointAuthMethod == "" {
		req.TokenEndpointAuthMethod = "client_secret_basic"
	}
	switch req.TokenEndpointAuthMethod {
	case "client_secret_basic", "client_secret_post", "none":
	default:
		return errorInvalidClientMetadata, fmt.Sprintf("unsupported token_endpoint_auth_method %q", req.TokenEndpointAuthMethod)
	}

	if len(req.RedirectURIs) == 0 {
		return errorInvalidRedirectURI, "redirect_uris is required"
	}
	seen := make(map[string]struct{}, len(req.RedirectURIs))
	for _, raw := range req.RedirectURIs {
		if len(raw) > maxRedirectURIBytes || !validRedirectURI(raw, req.TokenEndpointAuthMethod == "none") {
			return errorInvalidRedirectURI, fmt.Sprintf("invalid redirect URI %q", raw)
		}
		if _, duplicate := seen[raw]; duplicate {
			return errorInvalidRedirectURI, fmt.Sprintf("duplicate redirect URI %q", raw)
		}
		seen[raw] = struct{}{}
	}

	if len(req.GrantTypes) == 0 {
		req.GrantTypes = []string{oauth2.GrantTypeAuthorizationCode}
	}
	seenGrants := make(map[string]struct{}, len(req.GrantTypes))
	for _, grantType := range req.GrantTypes {
		if !slices.Contains(h.GrantTypes, grantType) {
			return errorInvalidClientMetadata, fmt.Sprintf("unsupported grant type %q", grantType)
		}
		switch grantType {
		case oauth2.GrantTypeAuthorizationCode, oauth2.GrantTypeRefreshToken, oauth2.GrantTypeImplicit:
		default:
			return errorInvalidClientMetadata, fmt.Sprintf("grant type %q is not supported for dynamic registration", grantType)
		}
		if _, duplicate := seenGrants[grantType]; duplicate {
			return errorInvalidClientMetadata, fmt.Sprintf("duplicate grant type %q", grantType)
		}
		seenGrants[grantType] = struct{}{}
	}
	if !slices.Contains(req.GrantTypes, oauth2.GrantTypeAuthorizationCode) && !slices.Contains(req.GrantTypes, oauth2.GrantTypeImplicit) {
		return errorInvalidClientMetadata, "dynamic registration requires an authorization_code or implicit grant"
	}
	// Defaults to "code" per RFC 7591 section 2.
	if len(req.ResponseTypes) == 0 {
		req.ResponseTypes = []string{oauth2.ResponseTypeCode}
	}
	seenResponses := make(map[string]struct{}, len(req.ResponseTypes))
	for i, responseType := range req.ResponseTypes {
		parts := strings.Fields(responseType)
		if len(parts) == 0 {
			return errorInvalidClientMetadata, "response types must not be empty"
		}
		slices.Sort(parts)
		canonical := strings.Join(parts, " ")
		if !supportsResponseType(h.ResponseTypes, canonical) {
			return errorInvalidClientMetadata, fmt.Sprintf("unsupported response type %q", responseType)
		}
		if _, duplicate := seenResponses[canonical]; duplicate {
			return errorInvalidClientMetadata, fmt.Sprintf("duplicate response type %q", responseType)
		}
		seenResponses[canonical] = struct{}{}
		req.ResponseTypes[i] = canonical
	}

	hasCodeResponse := responseTypesContain(req.ResponseTypes, oauth2.ResponseTypeCode)
	hasCodeGrant := slices.Contains(req.GrantTypes, oauth2.GrantTypeAuthorizationCode)
	if hasCodeResponse != hasCodeGrant {
		return errorInvalidClientMetadata, "authorization_code grant and code response type must be requested together"
	}
	hasImplicitResponse := responseTypesContain(req.ResponseTypes, oauth2.ResponseTypeToken) || responseTypesContain(req.ResponseTypes, oauth2.ResponseTypeIDToken)
	hasImplicitGrant := slices.Contains(req.GrantTypes, oauth2.GrantTypeImplicit)
	if hasImplicitResponse != hasImplicitGrant {
		return errorInvalidClientMetadata, "implicit grant and token or id_token response type must be requested together"
	}
	if len(req.LogoURI) > maxLogoURIBytes {
		return errorInvalidClientMetadata, "logo_uri is too long"
	}
	if req.LogoURI != "" {
		u, err := url.Parse(req.LogoURI)
		if err != nil || !u.IsAbs() || u.Fragment != "" || u.User != nil || !strings.EqualFold(u.Scheme, "https") || u.Host == "" || !sameOriginAsRedirect(u, req.RedirectURIs) {
			return errorInvalidClientMetadata, "logo_uri must be HTTPS and have the same origin as a redirect_uri"
		}
	}
	seenScopes := make(map[string]struct{})
	for _, scope := range strings.Fields(req.Scope) {
		if !slices.Contains([]string{tokens.ScopeOpenID, tokens.ScopeOfflineAccess, tokens.ScopeEmail, tokens.ScopeProfile, tokens.ScopeGroups, tokens.ScopeFederatedID}, scope) {
			return errorInvalidClientMetadata, fmt.Sprintf("unsupported scope %q", scope)
		}
		if _, duplicate := seenScopes[scope]; duplicate {
			return errorInvalidClientMetadata, fmt.Sprintf("duplicate scope %q", scope)
		}
		seenScopes[scope] = struct{}{}
	}
	if _, ok := seenScopes[tokens.ScopeOpenID]; !ok {
		return errorInvalidClientMetadata, "openid scope is required"
	}
	if slices.Contains(strings.Fields(req.Scope), tokens.ScopeOfflineAccess) && !slices.Contains(req.GrantTypes, oauth2.GrantTypeRefreshToken) {
		return errorInvalidClientMetadata, "offline_access scope requires the refresh_token grant type"
	}
	return "", ""
}

func validRedirectURI(raw string, public bool) bool {
	u, err := url.Parse(raw)
	if err != nil || !u.IsAbs() || u.Fragment != "" || u.User != nil || u.Opaque != "" {
		return false
	}
	switch strings.ToLower(u.Scheme) {
	case "https":
		return u.Host != "" && u.Hostname() != ""
	case "http":
		if !public || u.Host == "" {
			return false
		}
		host := u.Hostname()
		return strings.EqualFold(host, "localhost") || (net.ParseIP(host) != nil && net.ParseIP(host).IsLoopback())
	case "javascript", "data", "file", "ftp", "mailto", "ws", "wss":
		return false
	default:
		return public && strings.HasPrefix(u.Path, "/")
	}
}

func sameOriginAsRedirect(logo *url.URL, redirects []string) bool {
	for _, raw := range redirects {
		u, err := url.Parse(raw)
		if err == nil && strings.EqualFold(u.Scheme, logo.Scheme) && strings.EqualFold(u.Host, logo.Host) {
			return true
		}
	}
	return false
}

func supportsResponseType(supported map[string]bool, requested string) bool {
	for responseType, enabled := range supported {
		if !enabled {
			continue
		}
		parts := strings.Fields(responseType)
		slices.Sort(parts)
		if strings.Join(parts, " ") == requested {
			return true
		}
	}
	return false
}

func responseTypesContain(responseTypes []string, target string) bool {
	for _, responseType := range responseTypes {
		if slices.Contains(strings.Fields(responseType), target) {
			return true
		}
	}
	return false
}

func (h *Handler) writeError(w http.ResponseWriter, typ, description string, status int) {
	oauth2.WriteErrorResponse(h.Logger, w, typ, description, status)
}
