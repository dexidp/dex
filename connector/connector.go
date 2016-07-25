// Package connector defines interfaces for federated identity strategies.
package connector

import (
	"net/http"

	"github.com/coreos/poke/storage"
)

// Connector is a mechanism for federating login to a remote identity service.
//
// Implementations are expected to implement either the PasswordConnector or
// CallbackConnector interface.
type Connector interface {
	Close() error
}

// PasswordConnector is an optional interface for password based connectors.
type PasswordConnector interface {
	Login(username, password string) (identity storage.Identity, validPassword bool, err error)
}

// CallbackConnector is an optional interface for callback based connectors.
type CallbackConnector interface {
	LoginURL(callbackURL, state string) (string, error)
	HandleCallback(r *http.Request) (identity storage.Identity, state string, err error)
}

// GroupsConnector is an optional interface for connectors which can map a user to groups.
type GroupsConnector interface {
	Groups(identity storage.Identity) ([]string, error)
}
