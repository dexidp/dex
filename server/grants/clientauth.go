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

// ClientCredentials returns the client credentials of a request to the token or
// device authorization endpoint: from the Authorization header when present,
// otherwise from the form body.
func ClientCredentials(r *http.Request) (id, secret string, oerr *oauth2.Error) {
	basicID, basicSecret, ok := r.BasicAuth()
	if !ok {
		return r.PostFormValue("client_id"), r.PostFormValue("client_secret"), nil
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

// AuthenticateClient looks the client up and checks its secret in constant time.
// A public client has an empty secret, so it authenticates without one.
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
