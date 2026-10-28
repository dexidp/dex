// Package device implements the browser-facing side of the OAuth2 device
// authorization grant (RFC 8628): the /device user-code entry page, the
// /device/code authorization request, user-code verification, and the callback
// that completes the flow. The device_code token grant that the device polls for
// lives with the token endpoint.
package device
