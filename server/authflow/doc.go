// Package authflow implements dex's interactive, browser-facing authorization
// flow: the /auth authorization endpoint, connector and password login, the
// session (SSO) shortcut, and the connector callback.
//
// The flow is a state machine over a storage.AuthRequest. /auth is the
// dispatcher (dispatch.go): it parses the request, starts login, and on each
// return decides the next step from persisted state — hand off to MFA, to the
// consent screen, or issue the response (response.go). Steps never route to one
// another; each returns to /auth carrying an HMAC verifier that proves the
// transition ("continue" after login or a factor, "approved" after consent).
//
//	/auth                parse the request; pick a connector, reuse a session, or dispatch the next step
//	/auth/{c}, .../login connector or password login -> finalizeLogin -> /auth
//	/callback            connector callback -> finalizeLogin -> /auth
//
// The MFA, consent and logout steps live in sibling packages (server/mfa,
// server/consent, server/logout); they mount their own routes (/mfa, /approval,
// /logout), and the dispatcher sends users there and back. Shared session state
// lives in server/session (cookie, SSO, auth-session CRUD).
package authflow
