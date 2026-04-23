package server

// Safe error messages for user-facing responses.
// These messages are intentionally generic to avoid leaking internal details.
// All actual error details should be logged server-side.

const (
	// ErrMsgLoginError is a generic login error message shown to users.
	// Used when authentication fails due to internal server errors.
	ErrMsgLoginError = "Login error. Please contact your administrator or try again later."

	// ErrMsgAuthenticationFailed is shown when callback/SAML authentication fails.
	ErrMsgAuthenticationFailed = "Authentication failed. Please contact your administrator or try again later."

	// ErrMsgInternalServerError is a generic internal server error message.
	ErrMsgInternalServerError = "Internal server error. Please contact your administrator or try again later."

	// ErrMsgDatabaseError is shown when database operations fail.
	ErrMsgDatabaseError = "A database error occurred. Please try again later."

	// ErrMsgInvalidRequest is shown when request parsing fails.
	ErrMsgInvalidRequest = "Invalid request. Please try again."

	// ErrMsgMethodNotAllowed is shown when an unsupported HTTP method is used.
	ErrMsgMethodNotAllowed = "Method not allowed."

	// ErrMsgNotInRequiredGroups is shown when a user authenticates successfully
	// but is not a member of any of the groups required by the connector.
	ErrMsgNotInRequiredGroups = "You are not a member of any of the required groups to authenticate."
)
