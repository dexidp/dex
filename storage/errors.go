package storage

import "fmt"

const (
	// ErrNotFound is the error returned by a storage implementation if a resource cannot be found.
	ErrNotFound = "not found"

	// ErrAlreadyExists is the error returned by a storage implementation if a resource ID is taken during a create.
	ErrAlreadyExists = "already exists"

	// ErrNotImplemented is the error returned by a storage implementation if a call is not implemented quite yet.
	ErrNotImplemented = "not implemented"

	// ErrStorageMisconfigured is an error if the storage provider is misconfigured
	// This error would be considered permanent until the configuration for server has been updated and should
	// not be retried right away
	// Examples may include invalid database credentials
	ErrStorageMisconfigured = "storage provider misconfigured"

	// ErrStorageProviderOffline is an error if the backing persistent storage is currently down, but not necessarily
	// misconfigured.  An example of this is that a database server cannot be reached right now.
	// This error should be provided if we think retrying the operation later might result in a better result
	ErrStorageProviderOffline = "storage provider currently offline"

	// ErrStorageProviderInternalError is an error if the backing storage provider has an error that is generic but
	// specific to the storage provider.  It is sort of a catch all of "not dex's code, but rather the storage provider's
	ErrStorageProviderInternalError = "storage provider internal error"
)

// ErrorCode is a list of all errors for the storage package
type ErrorCode string

// Error is a storage specific error type
// While a provider can choose to provide any error that complies with the `error` interface, providing an error
// of type `Error` will allow customers of the storage package to be able to make more informed decisions about how to
// handle the error
type Error struct {
	Code    ErrorCode
	Details string
}

// Error satisfies the error interface
func (c Error) Error() string {
	if c.Details != "" {
		return fmt.Sprintf("%s - %s", string(c.Code), c.Details)
	}
	return string(c.Code)
}

// IsErrorCode is a helper function to make it simple to check if an error belongs to this package and is a specific error code
func IsErrorCode(err error, code ErrorCode) bool {
	// see if type assertion works
	e, ok := err.(Error)
	if ok {
		return e.Code == code
	}

	// in case we are given a storage.Error reference
	eRef, ok := err.(*Error)
	if ok {
		return eRef.Code == code
	}
	return false
}
