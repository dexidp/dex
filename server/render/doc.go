// Package render is the shared browser-facing infrastructure for the interactive
// auth flow: HTML error rendering and issuer-relative URL building. The flow's
// Handler and its MFA component embed a *UI so they render errors and build
// URLs the same way, without duplicating the logic.
package render
