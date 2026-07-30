package server

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"time"

	"github.com/coreos/go-oidc/v3/oidc"

	"github.com/dexidp/dex/examples/example-app/session"
)

// backchannelLogoutEvent is the member a logout token's "events" claim must
// carry, per OIDC Back-Channel Logout 1.0 §2.4. Its presence is what separates a
// logout token from an ID token that happens to be POSTed here.
const backchannelLogoutEvent = "http://schemas.openid.net/event/backchannel-logout"

// logoutTokenClaims are the parts of a logout token this app reads.
type logoutTokenClaims struct {
	Events    map[string]json.RawMessage `json:"events"`
	SessionID string                     `json:"sid"`
	Nonce     string                     `json:"nonce"`
}

// handleBackchannelLogout receives a logout token from the provider and ends the
// sessions it names.
//
// This is the half of single logout that does not involve the browser: dex POSTs
// here directly, so a user who signs out somewhere else stops being signed in
// here even if they never load a page. The app also runs a prompt=none check on
// page load, which catches the same thing eventually — the difference is that
// this arrives immediately, and works for a tab nobody is looking at.
//
// Validation follows Back-Channel Logout 1.0 §2.6. The signature, issuer,
// audience and expiry checks come from the same verifier the app uses for ID
// tokens; the rest is checked here.
func (s *Server) handleBackchannelLogout(w http.ResponseWriter, r *http.Request) {
	// §2.7: the response must not be cached, whatever it says.
	w.Header().Set("Cache-Control", "no-cache, no-store")

	if err := r.ParseForm(); err != nil {
		backchannelError(w, "invalid_request", "could not parse form")
		return
	}

	raw := r.PostFormValue("logout_token")
	if raw == "" {
		backchannelError(w, "invalid_request", "no logout_token in request")
		return
	}

	ctx := oidc.ClientContext(r.Context(), s.client)

	// Signature, iss, aud and exp. A logout token is signed and audienced exactly
	// like an ID token, so the same verifier applies — which is also why the
	// checks below matter: without them an ID token would pass for one.
	token, err := s.verifier.Verify(ctx, raw)
	if err != nil {
		backchannelError(w, "invalid_request", "logout token did not verify: "+err.Error())
		return
	}

	var claims logoutTokenClaims
	if err := token.Claims(&claims); err != nil {
		backchannelError(w, "invalid_request", "could not decode logout token claims")
		return
	}

	if _, ok := claims.Events[backchannelLogoutEvent]; !ok {
		backchannelError(w, "invalid_request", "logout token has no backchannel-logout event")
		return
	}

	// §2.4 forbids a nonce, precisely so that an ID token replayed to this
	// endpoint cannot be mistaken for a logout token.
	if claims.Nonce != "" {
		backchannelError(w, "invalid_request", "logout token must not contain a nonce")
		return
	}

	if token.Subject == "" && claims.SessionID == "" {
		backchannelError(w, "invalid_request", "logout token has neither sub nor sid")
		return
	}

	// Replay detection on jti is left out. It is optional in §2.6, and a repeated
	// logout token can only sign out a session that is already signed out. A real
	// relying party with side effects on logout should keep recently seen jti
	// values until their tokens expire.

	// The result is shown on the home page rather than logged: what makes this
	// worth seeing is when the token arrived relative to the next page load, and
	// that comparison only means anything on the page itself.
	s.sessions.SignOutByBackchannel(claims.SessionID, token.Subject)

	w.WriteHeader(http.StatusOK)
}

// backchannelError writes the JSON error body §2.7 asks for.
func backchannelError(w http.ResponseWriter, code, description string) {
	log.Printf("back-channel logout rejected: %s: %s", code, description)

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusBadRequest)
	_ = json.NewEncoder(w).Encode(map[string]string{
		"error":             code,
		"error_description": description,
	})
}

// handleEvents streams this browser's session events over SSE.
//
// The stream is what makes a back channel observable. Without it the earliest a
// page could learn its session had ended is the next request it makes — and a
// request is exactly when the app's own prompt=none check would have found out
// anyway, so a message delivered then demonstrates nothing.
func (s *Server) handleEvents(w http.ResponseWriter, r *http.Request) {
	flusher, ok := w.(http.Flusher)
	if !ok {
		http.Error(w, "streaming unsupported", http.StatusInternalServerError)
		return
	}

	// Watch an existing browser, never mint one. This endpoint is opened by a page
	// that already has a session; handing a cookie to anything that connects would
	// let a crawler fill the store with sessions nobody is behind.
	cookie, err := r.Cookie(session.CookieName)
	if err != nil || cookie.Value == "" {
		w.WriteHeader(http.StatusNoContent)
		return
	}

	notices, stop := s.sessions.Watch(cookie.Value)
	defer stop()

	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Connection", "keep-alive")
	w.WriteHeader(http.StatusOK)
	flusher.Flush()

	// Idle streams are kept alive with comments, which proxies and browsers both
	// leave alone.
	ping := time.NewTicker(25 * time.Second)
	defer ping.Stop()

	for {
		select {
		case <-r.Context().Done():
			return

		case <-ping.C:
			fmt.Fprint(w, ": ping\n\n")
			flusher.Flush()

		case notice := <-notices:
			payload, err := json.Marshal(notice)
			if err != nil {
				return
			}
			fmt.Fprintf(w, "event: backchannel-logout\ndata: %s\n\n", payload)
			flusher.Flush()
		}
	}
}
