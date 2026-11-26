package grants

import (
	"context"
	"crypto/subtle"
	"log/slog"
	"net/http"
	"net/url"

	"github.com/dexidp/dex/server/oauth2"
	"github.com/dexidp/dex/storage"
)

// ClientCredentials returns the client credentials of a request: from the
// Authorization header when present, otherwise from form, the parsed form the
// endpoint reads its other parameters from.
func ClientCredentials(r *http.Request, form url.Values) (id, secret string, oerr *oauth2.Error) {
	basicID, basicSecret, ok := r.BasicAuth()
	if !ok {
		return form.Get("client_id"), form.Get("client_secret"), nil
	}

	var err error
	if id, err = url.QueryUnescape(basicID); err != nil {
		return "", "", &oauth2.Error{Type: oauth2.InvalidRequest, Description: "client_id improperly encoded", Status: http.StatusBadRequest}
	}
	if secret, err = url.QueryUnescape(basicSecret); err != nil {
		return "", "", &oauth2.Error{Type: oauth2.InvalidRequest, Description: "client_secret improperly encoded", Status: http.StatusBadRequest}
	}
	return id, secret, nil
}

// clientAuthMethod names the token endpoint authentication method a request
// used, in the terms of RFC 7591 token_endpoint_auth_method. It mirrors
// ClientCredentials: the Authorization header wins over the form. secret is the
// secret ClientCredentials resolved.
func clientAuthMethod(r *http.Request, secret string) string {
	// A request without a secret did not authenticate, however the client
	// ID was sent; public clients commonly send it in a Basic header.
	if secret == "" {
		return "none"
	}
	if _, _, ok := r.BasicAuth(); ok {
		return "client_secret_basic"
	}
	return "client_secret_post"
}

// AuthenticateClient looks the client up and checks its secret in constant time.
// A client with an empty secret authenticates without one.
func AuthenticateClient(ctx context.Context, s storage.Storage, logger *slog.Logger, id, secret string) (storage.Client, *oauth2.Error) {
	client, err := s.GetClient(ctx, id)
	if err != nil {
		if err != storage.ErrNotFound {
			logger.ErrorContext(ctx, "failed to get client", "err", err)
			return storage.Client{}, &oauth2.Error{Type: oauth2.ServerError, Status: http.StatusInternalServerError}
		}
		return storage.Client{}, &oauth2.Error{Type: oauth2.InvalidClient, Description: "Invalid client credentials.", Status: http.StatusUnauthorized}
	}

	if subtle.ConstantTimeCompare([]byte(client.Secret), []byte(secret)) != 1 {
		if secret == "" {
			logger.InfoContext(ctx, "missing client_secret", "client_id", client.ID)
		} else {
			logger.InfoContext(ctx, "invalid client_secret", "client_id", client.ID)
		}
		return storage.Client{}, &oauth2.Error{Type: oauth2.InvalidClient, Description: "Invalid client credentials.", Status: http.StatusUnauthorized}
	}

	return client, nil
}
