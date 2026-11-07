// Package session owns dex's browser session: the session cookie, SSO session
// lookup, and auth-session CRUD. The interactive auth flow delegates to a
// Manager; it never touches these internals directly.
package session
