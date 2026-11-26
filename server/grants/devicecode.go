package grants

import (
	"context"
	"encoding/json"
	"log/slog"
	"net/http"
	"time"

	"github.com/dexidp/dex/server/connectors"
	"github.com/dexidp/dex/server/oauth2"
	"github.com/dexidp/dex/server/tokens"
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
			return nil, &oauth2.Error{Type: oauth2.ServerError, Status: http.StatusInternalServerError}
		}
		// Either the code never existed or it was already redeemed and deleted.
		// A used-up device code is an invalid grant under RFC 6749 section 5.2,
		// the same ruling the authorization_code grant makes on a missing code.
		return nil, &oauth2.Error{Type: oauth2.InvalidGrant, Description: "Invalid or expired device code.", Status: http.StatusBadRequest}
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

		// Spend the code before the response goes out. Deleting it is the atomic
		// single-use gate, the same one ExchangeAuthCode uses for the authorization
		// code the browser callback consumed to mint this response: a single DELETE
		// serializes concurrent redemptions on every backend, so the winner removes
		// the row and any loser finds it already gone. The stored response is the
		// access and refresh tokens themselves, so it does not outlive its delivery.
		// Consuming before decoding means a later failure leaves the code spent,
		// which is the trade ExchangeAuthCode already makes: replay safety over a
		// retry.
		if err := g.storage.DeleteDeviceToken(ctx, req.DeviceCode); err != nil {
			if err == storage.ErrNotFound {
				return nil, &oauth2.Error{Type: oauth2.InvalidGrant, Description: "Invalid or expired device code.", Status: http.StatusBadRequest}
			}
			g.logger.ErrorContext(ctx, "failed to spend device token", "err", err)
			return nil, &oauth2.Error{Type: oauth2.ServerError, Status: http.StatusInternalServerError}
		}

		// The token was minted by the browser callback and stored as a serialized
		// tokens.Response. Decode it so it is written like every other grant's response.
		var resp tokens.Response
		if err := json.Unmarshal([]byte(deviceToken.Token), &resp); err != nil {
			g.logger.ErrorContext(ctx, "failed to decode stored device token response", "err", err)
			return nil, &oauth2.Error{Type: oauth2.ServerError, Status: http.StatusInternalServerError}
		}
		return resp, nil

	default:
		return nil, &oauth2.Error{Type: oauth2.InvalidGrant, Description: "Invalid or expired device code.", Status: http.StatusBadRequest}
	}
}
