package oauth2

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"
)

// Error is an OAuth2 error response a token grant can return instead of writing
// it: the token endpoint turns it into the JSON error body. It lets a grant
// handler report failures as values rather than writing to the ResponseWriter.
type Error struct {
	Type        string
	Description string
	Status      int
}

func (e *Error) Error() string {
	if e.Description == "" {
		return e.Type
	}
	return e.Type + ": " + e.Description
}

// Errorf builds an *Error with a formatted description.
func Errorf(typ string, status int, format string, args ...any) *Error {
	return &Error{Type: typ, Description: fmt.Sprintf(format, args...), Status: status}
}

// WriteError writes an OAuth2 error response: a JSON body with the error code
// and an optional description, at the given HTTP status.
func WriteError(w http.ResponseWriter, typ, description string, statusCode int) error {
	data := struct {
		Error       string `json:"error"`
		Description string `json:"error_description,omitempty"`
	}{typ, description}
	body, err := json.Marshal(data)
	if err != nil {
		return fmt.Errorf("failed to marshal token error response: %v", err)
	}
	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Content-Length", strconv.Itoa(len(body)))
	w.WriteHeader(statusCode)
	w.Write(body)
	return nil
}
