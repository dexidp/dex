// Package httpclient provides a configurable HTTP client constructor with
// support for custom CA certificates, root CAs, and TLS settings.
//
// Root CA entries read as files at construction are re-read before each request.
// Changed bundles replace the transport and close its idle connections. Unreadable
// files or files without usable PEM certificates fail the request until repaired,
// including when certificate verification is disabled. PEM parsing retains
// x509.CertPool.AppendCertsFromPEM semantics. Inline PEM and base64 entries remain
// fixed. File-backed clients use a wrapping RoundTripper rather than *http.Transport.
package httpclient
