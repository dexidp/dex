// Package connector defines interfaces for federated identity strategies.
package connector

import "net/http"

// Connector is a mechanism for federating login to a remote identity service.
//
// Implementations are expected to implement either the PasswordConnector or
// CallbackConnector interface.
type Connector interface {
	Close() error
}

// Identity represents the ID Token claims supported by the server.
type Identity struct {
	UserID        string
	Username      string
	Email         string
	EmailVerified bool

	// ConnectorData holds data used by the connector for subsequent requests after initial
	// authentication, such as access tokens for upstream provides.
	//
	// This data is never shared with end users, OAuth clients, or through the API.
	ConnectorData []byte
}

// PasswordConnector is an optional interface for password based connectors.
type PasswordConnector interface {
	Login(username, password string) (identity Identity, validPassword bool, err error)
}

// CallbackConnector is an optional interface for callback based connectors.
type CallbackConnector interface {
	LoginURL(callbackURL, state string) (string, error)
	HandleCallback(r *http.Request) (identity Identity, err error)
}

// GroupsConnector is an optional interface for connectors which can map a user to groups.
type GroupsConnector interface {
	Groups(identity Identity) ([]string, error)
}
