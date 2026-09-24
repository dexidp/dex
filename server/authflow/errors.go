package authflow

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

	// ErrMsgRequestAlreadyCompleted is shown when a login request is resubmitted
	// after its AuthRequest is no longer in storage: most commonly because it was
	// already finalized by an earlier, still-in-flight submission (e.g. a
	// double-submitted login form, or a stale page resubmitted via the browser
	// back button), but also possible if the request expired in the meantime.
	ErrMsgRequestAlreadyCompleted = "This login request is no longer valid. It may have already been completed, or it may have expired. Please close this tab, or go back and start over if you need to sign in again."
)
