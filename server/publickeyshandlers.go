package server

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"
	"time"

	"github.com/go-jose/go-jose/v4"

	"github.com/dexidp/dex/pkg/otel/traces"
)

func (s *Server) handlePublicKeys(w http.ResponseWriter, r *http.Request) {
	ctx, span := traces.InstrumentHandler(r)
	defer span.End()
	keys, err := s.storage.GetKeys(ctx)
	if err != nil {
		s.logger.ErrorContext(ctx, "failed to get keys", "err", err)
		s.renderError(r, w, http.StatusInternalServerError, "Internal server error.")
		return
	}
	if keys.SigningKeyPub == nil {
		s.logger.ErrorContext(ctx, "no public keys found.")
		s.renderError(r, w, http.StatusInternalServerError, "Internal server error.")
		return
	}

	jwks := jose.JSONWebKeySet{
		Keys: make([]jose.JSONWebKey, len(keys.VerificationKeys)+1),
	}
	jwks.Keys[0] = *keys.SigningKeyPub
	for i, verificationKey := range keys.VerificationKeys {
		jwks.Keys[i+1] = *verificationKey.PublicKey
	}
	data, err := json.MarshalIndent(jwks, "", "  ")
	if err != nil {
		s.logger.ErrorContext(ctx, "failed to marshal discovery data", "err", err)
		s.renderError(r, w, http.StatusInternalServerError, "Internal server error.")
		return
	}

	maxAge := keys.NextRotation.Sub(s.now())
	if maxAge < (time.Minute * 2) {
		maxAge = time.Minute * 2
	}

	w.Header().Set("Cache-Control", fmt.Sprintf("max-age=%d, must-revalidate", int(maxAge.Seconds())))
	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Content-Length", strconv.Itoa(len(data)))
	w.Write(data)
}
