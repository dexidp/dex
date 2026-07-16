package grants

import (
	"context"
	"log/slog"
	"net/http"
	"time"

	"github.com/dexidp/dex/server/connectors"
	"github.com/dexidp/dex/server/oauth2"
	"github.com/dexidp/dex/storage"
)

// deviceCode serves the RFC 8628 device_code grant: the device polls for the
// token minted and stored by the browser callback once the user authorizes it.
// It issues nothing itself — a Minter returning the stored token — and drives the
// authorization_pending / slow_down polling protocol.
type deviceCode struct {
	storage storage.Storage
	now     func() time.Time
	logger  *slog.Logger
}

func (g *deviceCode) GrantType() string {
	return oauth2.GrantTypeDeviceCode
}

// RequiresClientAuth is false: the device is identified by the device code, not
// client credentials.
func (g *deviceCode) RequiresClientAuth() bool {
	return false
}

func (g *deviceCode) ScopePolicy() ScopePolicy {
	return ScopePolicy{}
}

func (g *deviceCode) ConnectorID(ctx context.Context, req *Request, client storage.Client) (string, *oauth2.Error) {
	return "", nil
}

func (g *deviceCode) Authorize(ctx context.Context, req *Request, client storage.Client, conn connectors.Connector) (Responder, error) {
	if req.DeviceCode == "" {
		return nil, &oauth2.Error{Type: oauth2.InvalidRequest, Description: "No device code received", Status: http.StatusBadRequest}
	}

	now := g.now()
	deviceToken, err := g.storage.GetDeviceToken(ctx, req.DeviceCode)
	if err != nil {
		if err != storage.ErrNotFound {
			g.logger.ErrorContext(ctx, "failed to get device code", "err", err)
		}
		return nil, &oauth2.Error{Type: oauth2.InvalidRequest, Description: "Invalid Device code.", Status: http.StatusBadRequest}
	}
	if now.After(deviceToken.Expiry) {
		return nil, &oauth2.Error{Type: oauth2.DeviceTokenExpired, Status: http.StatusBadRequest}
	}

	// Rate limiting: increase the poll interval until the device waits long enough.
	slowDown := false
	pollInterval := deviceToken.PollIntervalSeconds
	if now.Before(deviceToken.LastRequestTime.Add(time.Second * time.Duration(pollInterval))) {
		slowDown = true
		pollInterval += 5
	} else {
		pollInterval = 5
	}

	switch deviceToken.Status {
	case oauth2.DeviceTokenPending:
		updater := func(old storage.DeviceToken) (storage.DeviceToken, error) {
			old.PollIntervalSeconds = pollInterval
			old.LastRequestTime = now
			return old, nil
		}
		if err := g.storage.UpdateDeviceToken(ctx, req.DeviceCode, updater); err != nil {
			g.logger.ErrorContext(ctx, "failed to update device token", "err", err)
			return nil, &oauth2.Error{Type: oauth2.ServerError, Status: http.StatusInternalServerError}
		}
		if slowDown {
			return nil, &oauth2.Error{Type: oauth2.DeviceTokenSlowDown, Status: http.StatusBadRequest}
		}
		return nil, &oauth2.Error{Type: oauth2.DeviceTokenPending, Status: http.StatusBadRequest}

	case oauth2.DeviceTokenComplete:
		if oerr := verifyPKCE(req.CodeVerifier, deviceToken.PKCE); oerr != nil {
			return nil, oerr
		}
		// The token was minted and stored by the browser callback; relay it verbatim.
		return storedResponse(deviceToken.Token), nil

	default:
		return nil, &oauth2.Error{Type: oauth2.InvalidRequest, Description: "Invalid Device code.", Status: http.StatusBadRequest}
	}
}

// storedResponse writes an already-serialized token response verbatim.
type storedResponse string

func (s storedResponse) Write(w http.ResponseWriter) error {
	w.Header().Set("Content-Type", "application/json")
	_, err := w.Write([]byte(s))
	return err
}

// verifyPKCE checks a code_verifier against a stored PKCE challenge (RFC 7636),
// shared by the authorization_code and device_code grants.
func verifyPKCE(codeVerifier string, pkce storage.PKCE) *oauth2.Error {
	switch {
	case codeVerifier != "" && pkce.CodeChallenge != "":
		calculated, err := oauth2.CalculateCodeChallenge(codeVerifier, pkce.CodeChallengeMethod)
		if err != nil {
			return &oauth2.Error{Type: oauth2.ServerError, Status: http.StatusInternalServerError}
		}
		if pkce.CodeChallenge != calculated {
			return &oauth2.Error{Type: oauth2.InvalidGrant, Description: "Invalid code_verifier.", Status: http.StatusBadRequest}
		}
	case codeVerifier != "":
		// No code_challenge on /auth, but a code_verifier on /token.
		return &oauth2.Error{Type: oauth2.InvalidRequest, Description: "No PKCE flow started. Cannot check code_verifier.", Status: http.StatusBadRequest}
	case pkce.CodeChallenge != "":
		// PKCE started on /auth, but no code_verifier on /token.
		return &oauth2.Error{Type: oauth2.InvalidGrant, Description: "Expecting parameter code_verifier in PKCE flow.", Status: http.StatusBadRequest}
	}
	return nil
}
