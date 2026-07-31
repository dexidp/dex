package backchannel

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"maps"
	"net/http"
	"net/url"
	"slices"
	"strings"
	"sync"
	"time"

	"github.com/google/uuid"

	"github.com/dexidp/dex/server/internal"
	"github.com/dexidp/dex/server/oauth2"
	"github.com/dexidp/dex/server/signer"
	"github.com/dexidp/dex/storage"
)

const (
	// backchannelLogoutEvent is the event identifier a logout token must carry, per
	// OIDC Back-Channel Logout 1.0 §2.4.
	backchannelLogoutEvent = "http://schemas.openid.net/event/backchannel-logout"

	// backchannelTokenLifetime bounds the replay window for a logout token. The spec
	// recommends no more than two minutes.
	backchannelTokenLifetime = 2 * time.Minute

	// backchannelTimeout caps how long dex waits on one RP. Logout must not hang on a
	// wedged relying party.
	backchannelTimeout = 5 * time.Second
)

// Notifier posts logout tokens to the relying parties of a session that has ended.
//
// It lives outside server/logout because a session also ends by an operator's hand
// over the gRPC API, and every path that ends one goes through here.
type Notifier struct {
	Storage   storage.Storage
	Signer    signer.Signer
	IssuerURL oauth2.IssuerURL
	Logger    *slog.Logger

	// Now is the clock, for tests. Defaults to time.Now.
	Now func() time.Time

	// HTTPClient delivers the logout tokens. Defaults to one that refuses redirects.
	HTTPClient *http.Client
}

// logoutTokenClaims is the JWT dex POSTs to a relying party's backchannel_logout_uri.
//
// Note the absences: there is no "nonce" (the spec forbids it, to keep a logout token
// from being mistaken for an ID token) and no "events" payload beyond an empty object.
type logoutTokenClaims struct {
	Issuer    string                     `json:"iss"`
	Subject   string                     `json:"sub"`
	Audience  string                     `json:"aud"`
	IssuedAt  int64                      `json:"iat"`
	Expiry    int64                      `json:"exp"`
	JWTID     string                     `json:"jti"`
	SessionID string                     `json:"sid"`
	Events    map[string]json.RawMessage `json:"events"`
}

// Notify tells every relying party in the session that it is over.
//
// Delivery is best-effort and fire-and-forget: RP-Initiated Logout treats notifying
// other RPs as a courtesy, and a relying party that is down must not be able to block
// or fail the user's logout. Failures are logged and dropped.
//
// ponytail: no retries and no durable queue. An RP that is unreachable for these few
// seconds keeps its session until it expires on its own. If that becomes a real
// problem, the upgrade path is to persist pending notifications and drain them from
// the garbage collector, not to make the user wait here.
func (n *Notifier) Notify(ctx context.Context, authSession *storage.AuthSession) {
	if len(authSession.ClientStates) == 0 {
		return
	}

	subject, err := internal.Marshal(&internal.IDTokenSubject{
		UserId: authSession.UserID,
		ConnId: authSession.ConnectorID,
	})
	if err != nil {
		n.Logger.ErrorContext(ctx, "logout: failed to marshal backchannel subject", "err", err)
		return
	}

	sid := authSession.ID
	clientIDs := slices.Sorted(maps.Keys(authSession.ClientStates))

	// Read above while the session is still in hand: the caller deletes it the moment
	// this returns. Delivery runs off the request — one wedged relying party would
	// otherwise hold the user's redirect for the whole timeout — so it gets a context
	// that outlives the one dying at that redirect.
	ctx = context.WithoutCancel(ctx)
	go func() {
		ctx, cancel := context.WithTimeout(ctx, backchannelTimeout)
		defer cancel()

		var wg sync.WaitGroup
		for _, clientID := range clientIDs {
			wg.Go(func() {
				client, err := n.Storage.GetClient(ctx, clientID)
				if err != nil {
					n.Logger.DebugContext(ctx, "logout: backchannel skipped, client not found",
						"client_id", clientID, "err", err)
					return
				}
				if client.BackchannelLogoutURI == "" {
					return
				}
				n.deliverLogoutToken(ctx, client, subject, sid)
			})
		}
		wg.Wait()
	}()
}

// deliverLogoutToken mints a logout token for one client and POSTs it.
func (n *Notifier) deliverLogoutToken(ctx context.Context, client storage.Client, subject, sid string) {
	token, err := n.signLogoutToken(ctx, client.ID, subject, sid)
	if err != nil {
		n.Logger.ErrorContext(ctx, "logout: failed to sign logout token",
			"client_id", client.ID, "err", err)
		return
	}

	body := url.Values{"logout_token": {token}}.Encode()
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, client.BackchannelLogoutURI, strings.NewReader(body))
	if err != nil {
		n.Logger.ErrorContext(ctx, "logout: failed to build backchannel request",
			"client_id", client.ID, "err", err)
		return
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.Header.Set("Cache-Control", "no-cache, no-store")

	resp, err := n.client().Do(req)
	if err != nil {
		n.Logger.WarnContext(ctx, "logout: backchannel delivery failed",
			"client_id", client.ID, "uri", client.BackchannelLogoutURI, "err", err)
		return
	}
	defer resp.Body.Close()

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		n.Logger.WarnContext(ctx, "logout: backchannel delivery rejected",
			"client_id", client.ID, "uri", client.BackchannelLogoutURI, "status", resp.StatusCode)
		return
	}

	n.Logger.DebugContext(ctx, "logout: backchannel delivered", "client_id", client.ID)
}

// signLogoutToken builds and signs the logout token for one audience.
func (n *Notifier) signLogoutToken(ctx context.Context, clientID, subject, sid string) (string, error) {
	now := time.Now()
	if n.Now != nil {
		now = n.Now()
	}

	claims := logoutTokenClaims{
		Issuer:    n.IssuerURL.String(),
		Subject:   subject,
		Audience:  clientID,
		IssuedAt:  now.Unix(),
		Expiry:    now.Add(backchannelTokenLifetime).Unix(),
		JWTID:     uuid.New().String(),
		SessionID: sid,
		Events:    map[string]json.RawMessage{backchannelLogoutEvent: json.RawMessage(`{}`)},
	}

	payload, err := json.Marshal(claims)
	if err != nil {
		return "", fmt.Errorf("marshal logout token: %w", err)
	}

	token, err := n.Signer.Sign(ctx, payload)
	if err != nil {
		return "", fmt.Errorf("sign logout token: %w", err)
	}
	return token, nil
}

// client returns the HTTP client used for delivery, defaulting to one with
// no redirect following: a logout token must reach the URI the client registered, not
// wherever that URI happens to point today.
func (n *Notifier) client() *http.Client {
	if n.HTTPClient != nil {
		return n.HTTPClient
	}
	return &http.Client{
		CheckRedirect: func(*http.Request, []*http.Request) error {
			return http.ErrUseLastResponse
		},
	}
}
