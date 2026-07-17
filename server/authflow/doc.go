// Package authflow implements dex's interactive browser-facing authorization
// flow: the /auth authorization endpoint, connector and password login, the
// session (SSO) shortcut, MFA (TOTP and WebAuthn), the consent/approval screen,
// and RP-initiated logout.
//
// The flow is a state machine over a storage.AuthRequest. Every endpoint does
// its step and then hands control to advance (nextstep.go) — the single place
// that reads nextAuthStep and dispatches: redirect to the next MFA factor or the
// consent screen, or issue the code. Handlers never pick the next hop or build
// its URL themselves.
//
//	/auth               parse the request (request.go); pick a connector or reuse a session
//	/auth/{c}, .../login connector or password login -> finalizeLogin -> advance
//	/callback           connector callback -> finalizeLogin -> advance
//	/mfa/*              verify a factor, then back to /approval
//	/approval           consent, then advance -> issue the response (response.go)
//
// The leaf domains live in sub-packages: session (cookie, SSO, auth-session
// CRUD), mfa (authenticator chain, TOTP/WebAuthn), and web (error rendering and
// issuer URLs).
package authflow
