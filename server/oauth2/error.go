package oauth2

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"
)

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
