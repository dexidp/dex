// Package log carries per-request logging attributes (request ID, client IP)
// through the context, from the server's middleware down into the handlers and
// the CLI logger. It is a neutral, low-level home so the server, the auth flow
// and its sub-packages, and the CLI can all reference the same keys.
package log
