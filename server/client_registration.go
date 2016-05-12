package server

import (
	"encoding/json"
	"net/http"

	"github.com/coreos/dex/pkg/log"

	"github.com/coreos/go-oidc/oauth2"
	"github.com/coreos/go-oidc/oidc"
)

const (
	invalidRedirectURI    = "invalid_redirect_uri"
	invalidClientMetadata = "invalid_client_metadata"
)

func (s *Server) handleClientRegistration(w http.ResponseWriter, r *http.Request) {
	resp, err := s.handleClientRegistrationRequest(r)
	if err != nil {
		code := http.StatusBadRequest
		if err.Type == oauth2.ErrorServerError {
			code = http.StatusInternalServerError
		}
		writeResponseWithBody(w, code, err)
	} else {
		writeResponseWithBody(w, http.StatusCreated, resp)
	}
}

func (s *Server) handleClientRegistrationRequest(r *http.Request) (*oidc.ClientRegistrationResponse, *apiError) {
	var clientMetadata oidc.ClientMetadata
	if err := json.NewDecoder(r.Body).Decode(&clientMetadata); err != nil {
		return nil, newAPIError(oauth2.ErrorInvalidRequest, err.Error())
	}
	if err := s.ProviderConfig().Supports(clientMetadata); err != nil {
		return nil, newAPIError(invalidClientMetadata, err.Error())
	}

	// metadata is guarenteed to have at least one redirect_uri by earlier validation.
	creds, err := s.ClientManager.New(clientMetadata)
	if err != nil {
		log.Errorf("Failed to create new client identity: %v", err)
		return nil, newAPIError(oauth2.ErrorServerError, "unable to save client metadata")
	}

	return &oidc.ClientRegistrationResponse{
		ClientID:       creds.ID,
		ClientSecret:   creds.Secret,
		ClientMetadata: clientMetadata,
	}, nil
}
