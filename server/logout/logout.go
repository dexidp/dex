package logout

import (
	"context"
	"crypto/subtle"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"net/url"
	"slices"
	"time"

	"github.com/coreos/go-oidc/v3/oidc"

	"github.com/dexidp/dex/connector"
	"github.com/dexidp/dex/server/backchannel"
	"github.com/dexidp/dex/server/connectors"
	"github.com/dexidp/dex/server/internal"
	"github.com/dexidp/dex/server/oauth2"
	"github.com/dexidp/dex/server/router"
	"github.com/dexidp/dex/server/session"
	"github.com/dexidp/dex/server/signer"
	"github.com/dexidp/dex/server/templates"
	"github.com/dexidp/dex/server/tokens"
	"github.com/dexidp/dex/storage"
)

// Handler implements RP-Initiated Logout. It depends only on lower components
// (sessions, storage, connectors, the token issuer), so it carries no login-flow
// code.
type Handler struct {
	Storage    storage.Storage
	Templates  *templates.Templates
	Logger     *slog.Logger
	Sessions   *session.Manager
	Connectors *connectors.Cache
	Issuer     *tokens.Issuer
	Signer     signer.Signer
	IssuerURL  oauth2.IssuerURL
	Now        func() time.Time

	// Backchannel tells the session's relying parties that it ended. The same
	// notifier serves the gRPC API, which ends sessions too.
	Backchannel *backchannel.Notifier
}

// renderError renders a user-facing HTML error page.
func (h *Handler) renderError(r *http.Request, w http.ResponseWriter, status int, description string) {
	templates.RenderError(h.Templates, h.Logger, r, w, status, description)
}

// Mount registers the logout endpoints. Logout requires an active session, so it
// mounts only when sessions are enabled.
func (h *Handler) Mount(mux router.Mux) {
	if !h.Sessions.Enabled() {
		return
	}
	mux.HandleFunc("/logout", h.handleLogout)
	mux.HandleFunc("/logout/callback", h.handleLogoutCallback)
}

// handleLogout implements OIDC RP-Initiated Logout (https://openid.net/specs/openid-connect-rpinitiated-1_0.html).
//
// GET/POST /logout?id_token_hint=...&post_logout_redirect_uri=...&state=...
//
// The session cookie — not id_token_hint — decides whose session ends. The hint is
// merely a claim about which session the RP *believes* is current; treating it as
// authoritative would let anyone holding a leaked ID token log that user out, and
// the hint is deliberately accepted past its expiry, so leaked means leaked forever.
// Requiring the cookie means an attacker needs the victim's browser, at which point
// they have already won. The hint's remaining jobs are to name the requesting client
// (for post_logout_redirect_uri validation and refresh-token revocation) and, when it
// matches the current session, to let dex skip the confirmation prompt.
//
// Flow:
//  1. Resolve the session from the cookie, verifying its nonce against storage
//  2. Validate id_token_hint (signature + issuer; expiry and audience skipped per spec)
//     and cross-check its subject and sid against that session
//  3. Ask the user to confirm unless the hint vouches for the current session
//  4. Validate post_logout_redirect_uri against the client's registered URIs
//  5. Revoke the refresh tokens of the clients bound to this session
//  6. Notify every client in the session over the back channel
//  7. If the upstream connector implements LogoutCallbackConnector, redirect to it and
//     finish in handleLogoutCallback; otherwise delete the session, clear the cookie,
//     and redirect to post_logout_redirect_uri (or render the logout page)
func (h *Handler) handleLogout(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	if r.Method != http.MethodGet && r.Method != http.MethodPost {
		h.renderError(r, w, http.StatusMethodNotAllowed, "Method not allowed.")
		return
	}

	idTokenHint := r.FormValue("id_token_hint")
	postLogoutRedirectURI := r.FormValue("post_logout_redirect_uri")
	state := r.FormValue("state")

	// The session named by the cookie is the one that will end, if any. ValidSession
	// is the same check every other session-trusting path makes: nonce verified,
	// expiry enforced, and a cookie that survived its session cleared on the way out.
	authSession := h.Sessions.ValidSession(ctx, w, r)

	var clientID string
	hintMatchesSession := false

	if idTokenHint != "" {
		hint, err := h.parseIDTokenHint(ctx, idTokenHint)
		if err != nil {
			h.Logger.ErrorContext(ctx, "logout: invalid id_token_hint", "err", err)
			h.renderError(r, w, http.StatusBadRequest, "Invalid id_token_hint.")
			return
		}

		clientID = hint.clientID
		hintMatchesSession = hint.matches(authSession)

		h.Logger.DebugContext(ctx, "logout: parsed id_token_hint",
			"user_id", hint.userID, "connector_id", hint.connectorID,
			"client_id", clientID, "matches_session", hintMatchesSession)
	}

	// Ask the user to confirm in two cases, both from RP-Initiated Logout §2: "the OP
	// MUST ask the End-User this question if an id_token_hint was not provided or if
	// the supplied ID Token does not belong to the current OP session."
	//
	//   - A hint that names some other session is suspect whatever the method, so a
	//     cross-site POST cannot use a leaked token to end the current session either.
	//   - No hint at all needs confirming only on GET, which is what stops
	//     <img src="/logout"> from working as CSRF. The confirmation form itself POSTs
	//     back without a hint, and that submission is the one that must go through —
	//     which is why this case is method-sensitive and the one above is not.
	confirm := (idTokenHint != "" && !hintMatchesSession) ||
		(idTokenHint == "" && r.Method == http.MethodGet)
	if authSession != nil && confirm {
		if err := h.Templates.Logout(r, w, "", false, true); err != nil {
			h.Logger.ErrorContext(ctx, "server template error", "err", err)
		}
		return
	}

	// Validate post_logout_redirect_uri against registered client URIs.
	if postLogoutRedirectURI != "" {
		if clientID == "" {
			h.renderError(r, w, http.StatusBadRequest, "post_logout_redirect_uri requires id_token_hint.")
			return
		}

		client, err := h.Storage.GetClient(ctx, clientID)
		if err != nil {
			h.Logger.ErrorContext(ctx, "logout: failed to get client", "client_id", clientID, "err", err)
			h.renderError(r, w, http.StatusBadRequest, "Invalid client.")
			return
		}

		if !slices.Contains(client.PostLogoutRedirectURIs, postLogoutRedirectURI) {
			h.Logger.WarnContext(ctx, "logout: unregistered post_logout_redirect_uri",
				"uri", postLogoutRedirectURI, "client_id", clientID)
			h.renderError(r, w, http.StatusBadRequest, "Unregistered post_logout_redirect_uri.")
			return
		}
	}

	// Nothing to end: no cookie, or a cookie for a session that is gone or expired.
	// ValidSession has already cleared the cookie if there was one worth clearing.
	// Still honor the redirect, so a repeated logout lands the user where they asked.
	if authSession == nil {
		h.finishLogout(w, r, postLogoutRedirectURI, state, false)
		return
	}

	// Try upstream logout first. It needs a live auth session to park the logout
	// parameters in, so it must run before the session is deleted, and it finishes
	// the flow in handleLogoutCallback.
	//
	// Nothing is announced or revoked on this branch. The session does not end here —
	// it ends in the callback — and a user who abandons the upstream provider's logout
	// page never comes back to it. Acting now would leave the relying parties signed
	// out, and their credentials deleted, while dex's session and cookie live on, so
	// the next visit to any of them would silently sign the user back in through SSO
	// and the logout would appear to undo itself. Waiting until the session is really
	// gone costs a best-effort notification in the abandoned case and buys consistency.
	if redirectURL, ok := h.tryUpstreamLogout(ctx, authSession, postLogoutRedirectURI, state, clientID); ok {
		http.Redirect(w, r, redirectURL, http.StatusSeeOther)
		return
	}

	h.Logger.DebugContext(ctx, "logout: completing",
		"user_id", authSession.UserID, "connector_id", authSession.ConnectorID, "client_id", clientID)

	// Both read ClientStates, which dies with the session, so both run before it does.
	h.revokeSessionBoundTokens(ctx, authSession)
	h.Backchannel.Notify(ctx, authSession)

	loggedOut := h.deleteAuthSession(ctx, authSession.UserID, authSession.ConnectorID)
	h.Sessions.ClearCookie(w)
	h.finishLogout(w, r, postLogoutRedirectURI, state, loggedOut)
}

// revokeSessionBoundTokens deletes the refresh tokens of the clients in this session
// that declared themselves bound to it, and deliberately leaves every other client's
// token alone.
//
// The narrow scope is the whole point, so please read this before widening it back.
//
// No standard settles the question. RP-Initiated Logout says nothing at all about
// refresh tokens — the OP's duties there are to end its own session and notify the
// relying parties. OIDC Core §11 notes that "the use of Refresh Tokens is not
// exclusive to the offline_access use case", and RFC 7009 puts revocation in the
// client's hands, not the OP's. Implementations diverge accordingly: Ory Hydra and
// Zitadel revoke nothing on logout and expose separate administrative APIs; Duende
// IdentityServer hides it behind a per-client CoordinateLifetimeWithUserSession flag;
// Keycloak tears down the SSO session but spares offline_access tokens; Auth0 revokes
// and openly describes doing so as going beyond the OIDC logout standards.
//
// Dex cannot borrow Keycloak's rule, because dex requires offline_access for *every*
// refresh token (see shouldIssueRefreshToken in server/grants). A browser RP and a
// kubectl credential arrive at the token endpoint looking identical, so there is no
// request-level signal to separate them — which is exactly how revoking on behalf of
// the whole user used to destroy a user's kubectl login every time they signed out of
// a web app. Dex's answer is the client's own RefreshTokenLifetime, which is the
// only place that distinction exists, and this walks the session by it.
//
// Redundant on its own: a session-bound token is refused and deleted the next time
// it is redeemed anyway (see sessionID in server/grants). Doing it here means the
// credential is gone at logout rather than at the attacker's convenience.
//
// Standalone tokens survive, including the requesting client's. An earlier version
// revoked that one on the grounds that the RP had already discarded it before
// redirecting here, which is true but is now the client's call to make: a client that
// wants its tokens to end with the session says so, and one that does not has said
// the opposite. Logging the other RPs out remains back-channel logout's job.
func (h *Handler) revokeSessionBoundTokens(ctx context.Context, authSession *storage.AuthSession) {
	var bound []string
	for clientID := range authSession.ClientStates {
		client, err := h.Storage.GetClient(ctx, clientID)
		if err != nil {
			h.Logger.DebugContext(ctx, "logout: revocation skipped, client not found",
				"client_id", clientID, "err", err)
			continue
		}
		if client.RefreshBoundToSession() {
			bound = append(bound, clientID)
		}
	}

	h.Issuer.Refresh.RevokeClients(ctx, authSession.UserID, authSession.ConnectorID, bound)
}

// idTokenHint holds the parts of a verified id_token_hint that logout cares about.
type idTokenHint struct {
	userID      string
	connectorID string
	clientID    string
	sessionID   string // "sid", empty for tokens minted before sid existed
}

// matches reports whether the hint describes the given session. A hint carrying a sid
// must match it exactly; one without a sid falls back to the subject alone, which is
// all a token issued by an older dex can offer.
func (h idTokenHint) matches(authSession *storage.AuthSession) bool {
	if authSession == nil {
		return false
	}
	if h.userID != authSession.UserID || h.connectorID != authSession.ConnectorID {
		return false
	}
	if h.sessionID == "" {
		return true
	}
	return h.sessionID == session.SessionID(authSession.Nonce)
}

// parseIDTokenHint verifies the hint and extracts the subject, requesting client and
// session id from it.
func (h *Handler) parseIDTokenHint(ctx context.Context, raw string) (idTokenHint, error) {
	idToken, err := h.validateIDTokenHint(ctx, raw)
	if err != nil {
		return idTokenHint{}, err
	}

	sub := new(internal.IDTokenSubject)
	if err := internal.Unmarshal(idToken.Subject, sub); err != nil {
		return idTokenHint{}, fmt.Errorf("unmarshal subject: %w", err)
	}

	// With cross-client (trusted peers) scopes a token can carry several audiences; the
	// requesting client is then in "azp". Same resolution as token introspection uses.
	var claims struct {
		AuthorizingParty string `json:"azp"`
		SessionID        string `json:"sid"`
	}
	if err := idToken.Claims(&claims); err != nil {
		return idTokenHint{}, fmt.Errorf("decode claims: %w", err)
	}

	hint := idTokenHint{
		userID:      sub.UserId,
		connectorID: sub.ConnId,
		sessionID:   claims.SessionID,
	}

	switch len(idToken.Audience) {
	case 0:
		// No audience — cannot determine the client.
	case 1:
		hint.clientID = idToken.Audience[0]
	default:
		hint.clientID = claims.AuthorizingParty
	}

	return hint, nil
}

// handleLogoutCallback receives the redirect back from the upstream provider
// after it has completed its logout.
//
// Identity is resolved from the session cookie (HttpOnly, Secure, SameSite=Lax).
// The session must still exist in storage with a non-nil LogoutState (set before
// the upstream redirect). After validation, the session is deleted.
func (h *Handler) handleLogoutCallback(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	// Resolve identity from the session cookie.
	userID, connectorID, nonce, ok := h.Sessions.ParseCookie(r)
	if !ok {
		h.renderError(r, w, http.StatusBadRequest, "Invalid session cookie.")
		return
	}

	// Load the session and verify nonce.
	session, err := h.Storage.GetAuthSession(ctx, userID, connectorID)
	if err != nil {
		h.Logger.ErrorContext(ctx, "logout callback: session not found", "err", err)
		h.renderError(r, w, http.StatusBadRequest, "Session not found.")
		return
	}

	if subtle.ConstantTimeCompare([]byte(session.Nonce), []byte(nonce)) != 1 {
		h.renderError(r, w, http.StatusBadRequest, "Invalid session.")
		return
	}

	if session.LogoutState == nil {
		h.renderError(r, w, http.StatusBadRequest, "No logout in progress.")
		return
	}

	ls := session.LogoutState

	// Let the connector validate the upstream logout response if it supports it.
	if ls.ConnectorID != "" {
		conn, err := h.Connectors.Get(ctx, ls.ConnectorID)
		if err == nil {
			if logoutConn, ok := conn.Connector.(connector.LogoutCallbackConnector); ok {
				if err := logoutConn.HandleLogoutCallback(ctx, r); err != nil {
					h.Logger.ErrorContext(ctx, "logout: upstream logout response validation failed",
						"connector_id", ls.ConnectorID, "err", err)
				}
			}
		}
	}

	// This is where the session actually ends on the upstream path, so this is where
	// the tokens bound to it go and the other relying parties are told. Doing either
	// back in handleLogout would have signed them out while dex's session was still
	// alive, waiting for a callback that a user who closed the tab never sends.
	h.revokeSessionBoundTokens(ctx, &session)
	h.Backchannel.Notify(ctx, &session)

	// Session kept alive until now — delete it and clear the cookie.
	h.deleteAuthSession(ctx, userID, connectorID)
	h.Sessions.ClearCookie(w)
	h.finishLogout(w, r, ls.PostLogoutRedirectURI, ls.State, true)
}

// finishLogout redirects to post_logout_redirect_uri when the RP supplied one that
// passed validation, as RP-Initiated Logout §4 requires, and otherwise renders the
// logout page. loggedOut indicates whether an active session was actually terminated.
//
// Callers must have validated postLogoutRedirectURI against the client's registered
// URIs before reaching here; this function redirects to whatever it is handed.
func (h *Handler) finishLogout(w http.ResponseWriter, r *http.Request, postLogoutRedirectURI, state string, loggedOut bool) {
	if postLogoutRedirectURI != "" {
		if u, err := url.Parse(postLogoutRedirectURI); err == nil {
			if state != "" {
				q := u.Query()
				q.Set("state", state)
				u.RawQuery = q.Encode()
			}
			http.Redirect(w, r, u.String(), http.StatusFound)
			return
		}
		h.Logger.ErrorContext(r.Context(), "logout: failed to parse post_logout_redirect_uri",
			"uri", postLogoutRedirectURI)
	}

	if err := h.Templates.Logout(r, w, "", loggedOut, false); err != nil {
		h.Logger.ErrorContext(r.Context(), "server template error", "err", err)
	}
}

// tryUpstreamLogout attempts to redirect to the upstream provider's logout endpoint.
// It stores LogoutState in the auth session before redirecting so the callback can
// read it back. Returns the redirect URL and true on success, or ("", false) if
// upstream logout is not possible (connector doesn't support it, etc.).
func (h *Handler) tryUpstreamLogout(ctx context.Context, authSession *storage.AuthSession, postLogoutRedirectURI, state, clientID string) (string, bool) {
	userID, connectorID := authSession.UserID, authSession.ConnectorID
	if connectorID == "" {
		return "", false
	}

	conn, err := h.Connectors.Get(ctx, connectorID)
	if err != nil {
		return "", false
	}

	logoutConn, ok := conn.Connector.(connector.LogoutCallbackConnector)
	if !ok {
		return "", false
	}

	// Store logout parameters in the session.
	if err := h.Storage.UpdateAuthSession(ctx, userID, connectorID, func(old storage.AuthSession) (storage.AuthSession, error) {
		old.LogoutState = &storage.LogoutState{
			PostLogoutRedirectURI: postLogoutRedirectURI,
			State:                 state,
			ClientID:              clientID,
			ConnectorID:           connectorID,
		}
		return old, nil
	}); err != nil {
		h.Logger.ErrorContext(ctx, "logout: failed to save logout state", "err", err)
		return "", false
	}

	callbackURI := h.IssuerURL.AbsURL("/logout/callback")
	upstreamURL, err := logoutConn.LogoutURL(ctx, callbackURI)
	if err != nil {
		h.Logger.ErrorContext(ctx, "logout: upstream connector error", "err", err)
		return "", false
	}
	if upstreamURL == "" {
		return "", false
	}

	u, err := url.Parse(upstreamURL)
	if err != nil {
		h.Logger.ErrorContext(ctx, "logout: failed to parse upstream URL", "err", err)
		return "", false
	}

	return u.String(), true
}

// deleteAuthSession deletes the session and returns true if it existed.
func (h *Handler) deleteAuthSession(ctx context.Context, userID, connectorID string) bool {
	if userID == "" || connectorID == "" {
		return false
	}
	if err := h.Storage.DeleteAuthSession(ctx, userID, connectorID); err != nil {
		if !errors.Is(err, storage.ErrNotFound) {
			h.Logger.ErrorContext(ctx, "logout: failed to delete auth session", "err", err)
		}
		return false
	}
	h.Logger.InfoContext(ctx, "logout successful", "user_id", userID, "connector_id", connectorID)
	return true
}

// validateIDTokenHint verifies an id_token_hint was issued by this server. The
// signature check (via signer.KeySet) is what establishes trust; expiry and
// audience are intentionally skipped per RP-Initiated Logout.
func (h *Handler) validateIDTokenHint(ctx context.Context, hint string) (*oidc.IDToken, error) {
	verifier := oidc.NewVerifier(h.IssuerURL.String(), &signer.KeySet{Signer: h.Signer}, &oidc.Config{
		SkipExpiryCheck:   true,
		SkipClientIDCheck: true,
	})
	return verifier.Verify(ctx, hint)
}
