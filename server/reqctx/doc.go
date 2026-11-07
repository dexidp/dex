// Package reqctx carries per-request values (request ID, client IP) through the
// context. The server's middleware sets them; the CLI logger reads them as log
// attributes and the auth flow reads the client IP for the session audit record.
// It is a neutral, low-level home so the server, the auth flow and its
// sub-packages, and the CLI can all reference the same keys without an import
// cycle.
package reqctx
