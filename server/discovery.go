package server

// discovery.go serves the OIDC discovery document
// (/.well-known/openid-configuration) and the JWKS endpoint (/keys).

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"sort"
	"strconv"
	"time"

	"github.com/go-jose/go-jose/v4"
)

func (s *Server) handlePublicKeys(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	// TODO(ericchiang): Cache this.
	keys, err := s.signer.ValidationKeys(ctx)
	if err != nil {
		s.logger.ErrorContext(r.Context(), "failed to get keys", "err", err)
		s.renderError(r, w, http.StatusInternalServerError, "Internal server error.")
		return
	}

	if len(keys) == 0 {
		s.logger.ErrorContext(r.Context(), "no public keys found.")
		s.renderError(r, w, http.StatusInternalServerError, "Internal server error.")
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
		s.logger.ErrorContext(r.Context(), "failed to marshal discovery data", "err", err)
		s.renderError(r, w, http.StatusInternalServerError, "Internal server error.")
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

type discovery struct {
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

func (s *Server) discoveryHandler(ctx context.Context) (http.HandlerFunc, error) {
	d := s.constructDiscovery(ctx)

	data, err := json.MarshalIndent(d, "", "  ")
	if err != nil {
		return nil, fmt.Errorf("failed to marshal discovery data: %v", err)
	}

	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.Header().Set("Content-Length", strconv.Itoa(len(data)))
		w.Write(data)
	}), nil
}

func (s *Server) constructDiscovery(ctx context.Context) discovery {
	d := discovery{
		Issuer:            s.issuerURL.String(),
		Auth:              s.absURL("/auth"),
		Token:             s.absURL("/token"),
		Keys:              s.absURL("/keys"),
		UserInfo:          s.absURL("/userinfo"),
		DeviceEndpoint:    s.absURL("/device/code"),
		Introspect:        s.absURL("/token/introspect"),
		Subjects:          []string{"public"},
		IDTokenAlgs:       []string{string(jose.RS256)},
		CodeChallengeAlgs: s.pkce.CodeChallengeMethodsSupported,
		Scopes:            []string{"openid", "email", "groups", "profile", "offline_access"},
		AuthMethods:       []string{"client_secret_basic", "client_secret_post"},
		Claims: []string{
			"iss", "sub", "aud", "iat", "exp", "email", "email_verified",
			"locale", "name", "preferred_username", "at_hash",
		},
	}

	// Determine signing algorithm from signer
	signingAlg, err := s.signer.Algorithm(ctx)
	if err != nil {
		s.logger.Error("failed to get signing algorithm", "err", err)
	} else {
		d.IDTokenAlgs = []string{string(signingAlg)}
	}

	for responseType := range s.supportedResponseTypes {
		d.ResponseTypes = append(d.ResponseTypes, responseType)
	}
	sort.Strings(d.ResponseTypes)

	d.GrantTypes = s.supportedGrantTypes

	if s.sessionConfig != nil {
		d.EndSession = s.absURL("/logout")
	}

	return d
}
