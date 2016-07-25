// Package internal contains support packages for the oidc package.
package internal

// ContextKey is just an empty struct. It exists so context keys can be an immutable
// public variable with a unique type. It's immutable because nobody else can create
// a ContextKey, being unexported.
type ContextKey struct{}
