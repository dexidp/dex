package gocb

import (
	"errors"
	"strings"

	"gopkg.in/couchbase/gocbcore.v7"
)

// MultiError encapsulates multiple errors that may be returned by one method.
type MultiError struct {
	Errors []error
}

func (e *MultiError) add(err error) {
	if multiErr, ok := err.(*MultiError); ok {
		e.Errors = append(e.Errors, multiErr.Errors...)
	} else {
		e.Errors = append(e.Errors, err)
	}
}

func (e *MultiError) get() error {
	if len(e.Errors) == 0 {
		return nil
	} else if len(e.Errors) == 1 {
		return e.Errors[0]
	} else {
		return e
	}
}

func (e *MultiError) Error() string {
	var errors []string
	for _, err := range e.Errors {
		errors = append(errors, err.Error())
	}
	return strings.Join(errors, ", ")
}

type clientError struct {
	message string
}

func (e clientError) Error() string {
	return e.message
}

var (
	// ErrNotEnoughReplicas occurs when not enough replicas exist to match the specified durability requirements.
	ErrNotEnoughReplicas = errors.New("Not enough replicas to match durability requirements.")
	// ErrDurabilityTimeout occurs when the server took too long to meet the specified durability requirements.
	ErrDurabilityTimeout = errors.New("Failed to meet durability requirements in time.")
	// ErrNoResults occurs when no results are available to a query.
	ErrNoResults = errors.New("No results returned.")
	// ErrNoOpenBuckets occurs when a cluster-level operation is performed before any buckets are opened.
	ErrNoOpenBuckets = errors.New("You must open a bucket before you can perform cluster level operations.")
	// ErrIndexInvalidName occurs when an invalid name was specified for an index.
	ErrIndexInvalidName = errors.New("An invalid index name was specified.")
	// ErrIndexNoFields occurs when an index with no fields is created.
	ErrIndexNoFields = errors.New("You must specify at least one field to index.")
	// ErrIndexNotFound occurs when an operation expects an index but it was not found.
	ErrIndexNotFound = errors.New("The index specified does not exist.")
	// ErrIndexAlreadyExists occurs when an operation expects an index not to exist, but it was found.
	ErrIndexAlreadyExists = errors.New("The index specified already exists.")
	// ErrFacetNoRanges occurs when a range-based facet is specified but no ranges were indicated.
	ErrFacetNoRanges = errors.New("At least one range must be specified on a facet.")

	// ErrSearchIndexInvalidName occurs when an invalid name was specified for a search index.
	ErrSearchIndexInvalidName = errors.New("An invalid search index name was specified.")
	// ErrSearchIndexMissingType occurs when no type was specified for a search index.
	ErrSearchIndexMissingType = errors.New("No search index type was specified.")
	// ErrSearchIndexInvalidSourceType occurs when an invalid source type was specific for a search index.
	ErrSearchIndexInvalidSourceType = errors.New("An invalid search index source type was specified.")
	// ErrSearchIndexInvalidSourceName occurs when an invalid source name was specific for a search index.
	ErrSearchIndexInvalidSourceName = errors.New("An invalid search index source name was specified.")
	// ErrSearchIndexAlreadyExists occurs when an invalid source name was specific for a search index.
	ErrSearchIndexAlreadyExists = errors.New("The search index specified already exists.")
	// ErrSearchIndexInvalidIngestControlOp occurs when an invalid ingest control op was specific for a search index.
	ErrSearchIndexInvalidIngestControlOp = errors.New("An invalid search index ingest control op was specified.")
	// ErrSearchIndexInvalidQueryControlOp occurs when an invalid query control op was specific for a search index.
	ErrSearchIndexInvalidQueryControlOp = errors.New("An invalid search index query control op was specified.")
	// ErrSearchIndexInvalidPlanFreezeControlOp occurs when an invalid plan freeze control op was specific for a search index.
	ErrSearchIndexInvalidPlanFreezeControlOp = errors.New("An invalid search index plan freeze control op was specified.")

	// ErrMixedAuthentication occurs when a combination of certification authentication and password authentication are used.
	ErrMixedAuthentication = errors.New("Invalid mixed authentication configuration, cannot use cluster level authentication with bucket password authentication.")
	// ErrMixedCertAuthentication occurs when client certificate authentication is setup but CertAuthenticator is not used or vise versa.
	ErrMixedCertAuthentication = errors.New("Invalid mixed authentication configuration, client certificate and CertAuthenticator must be used together.")
	// ErrDispatchFail occurs when we failed to execute an operation due to internal routing issues.
	ErrDispatchFail = gocbcore.ErrDispatchFail
	// ErrBadHosts occurs when an invalid list of hosts is specified for bootstrapping.
	ErrBadHosts = gocbcore.ErrBadHosts
	// ErrProtocol occurs when an invalid protocol is specified for bootstrapping.
	ErrProtocol = gocbcore.ErrProtocol
	// ErrNoReplicas occurs when an operation expecting replicas is performed, but no replicas are available.
	ErrNoReplicas = gocbcore.ErrNoReplicas
	// ErrInvalidServer occurs when a specified server index is invalid.
	ErrInvalidServer = gocbcore.ErrInvalidServer
	// ErrInvalidVBucket occurs when a specified vbucket index is invalid.
	ErrInvalidVBucket = gocbcore.ErrInvalidVBucket
	// ErrInvalidReplica occurs when a specified replica index is invalid.
	ErrInvalidReplica = gocbcore.ErrInvalidReplica
	// ErrInvalidCert occurs when the specified certificate is not valid.
	ErrInvalidCert = gocbcore.ErrInvalidCert
	// ErrInvalidCredentials is returned when an invalid set of credentials is provided for a service.
	ErrInvalidCredentials = gocbcore.ErrInvalidCredentials
	// ErrNonZeroCas occurs when an operation that require a CAS value of 0 is used with a non-zero value.
	ErrNonZeroCas = gocbcore.ErrNonZeroCas

	// ErrShutdown occurs when an operation is performed on a bucket that has been closed.
	ErrShutdown = gocbcore.ErrShutdown
	// ErrOverload occurs when more operations were dispatched than the client is capable of writing.
	ErrOverload = gocbcore.ErrOverload
	// ErrNetwork occurs when various generic network errors occur.
	ErrNetwork = gocbcore.ErrNetwork
	// ErrTimeout occurs when an operation times out.
	ErrTimeout = gocbcore.ErrTimeout
	// ErrCliInternalError indicates an internal error occurred within the client.
	ErrCliInternalError = gocbcore.ErrCliInternalError

	// ErrStreamClosed occurs when an error is related to a stream closing.
	ErrStreamClosed = gocbcore.ErrStreamClosed
	// ErrStreamStateChanged occurs when an error is related to a cluster rebalance.
	ErrStreamStateChanged = gocbcore.ErrStreamStateChanged
	// ErrStreamDisconnected occurs when a stream is closed due to a connection dropping.
	ErrStreamDisconnected = gocbcore.ErrStreamDisconnected
	// ErrStreamTooSlow occurs when a stream is closed due to being too slow at consuming data.
	ErrStreamTooSlow = gocbcore.ErrStreamTooSlow

	// ErrKeyNotFound occurs when the key is not found on the server.
	ErrKeyNotFound = gocbcore.ErrKeyNotFound
	// ErrKeyExists occurs when the key already exists on the server.
	ErrKeyExists = gocbcore.ErrKeyExists
	// ErrTooBig occurs when the document is too big to be stored.
	ErrTooBig = gocbcore.ErrTooBig
	// ErrNotStored occurs when an item fails to be stored.  Usually an append/prepend to missing key.
	ErrNotStored = gocbcore.ErrNotStored
	// ErrAuthError occurs when there is an issue with authentication (bad password?).
	ErrAuthError = gocbcore.ErrAuthError
	// ErrRangeError occurs when an invalid range is specified.
	ErrRangeError = gocbcore.ErrRangeError
	// ErrRollback occurs when a server rollback has occurred making the operation no longer valid.
	ErrRollback = gocbcore.ErrRollback
	// ErrAccessError occurs when you do not have access to the specified resource.
	ErrAccessError = gocbcore.ErrAccessError
	// ErrOutOfMemory occurs when the server has run out of memory to process requests.
	ErrOutOfMemory = gocbcore.ErrOutOfMemory
	// ErrNotSupported occurs when an operation is performed which is not supported.
	ErrNotSupported = gocbcore.ErrNotSupported
	// ErrInternalError occurs when an internal error has prevented an operation from succeeding.
	ErrInternalError = gocbcore.ErrInternalError
	// ErrBusy occurs when the server is too busy to handle your operation.
	ErrBusy = gocbcore.ErrBusy
	// ErrTmpFail occurs when the server is not immediately able to handle your request.
	ErrTmpFail = gocbcore.ErrTmpFail

	// ErrSubDocPathNotFound occurs when a sub-document operation targets a path
	// which does not exist in the specifie document.
	ErrSubDocPathNotFound = gocbcore.ErrSubDocPathNotFound
	// ErrSubDocPathMismatch occurs when a sub-document operation specifies a path
	// which does not match the document structure (field access on an array).
	ErrSubDocPathMismatch = gocbcore.ErrSubDocPathMismatch
	// ErrSubDocPathInvalid occurs when a sub-document path could not be parsed.
	ErrSubDocPathInvalid = gocbcore.ErrSubDocPathInvalid
	// ErrSubDocPathTooBig occurs when a sub-document path is too big.
	ErrSubDocPathTooBig = gocbcore.ErrSubDocPathTooBig
	// ErrSubDocDocTooDeep occurs when an operation would cause a document to be
	// nested beyond the depth limits allowed by the sub-document specification.
	ErrSubDocDocTooDeep = gocbcore.ErrSubDocDocTooDeep
	// ErrSubDocCantInsert occurs when a sub-document operation could not insert.
	ErrSubDocCantInsert = gocbcore.ErrSubDocCantInsert
	// ErrSubDocNotJson occurs when a sub-document operation is performed on a
	// document which is not JSON.
	ErrSubDocNotJson = gocbcore.ErrSubDocNotJson
	// ErrSubDocBadRange occurs when a sub-document operation is performed with
	// a bad range.
	ErrSubDocBadRange = gocbcore.ErrSubDocBadRange
	// ErrSubDocBadDelta occurs when a sub-document counter operation is performed
	// and the specified delta is not valid.
	ErrSubDocBadDelta = gocbcore.ErrSubDocBadDelta
	// ErrSubDocPathExists occurs when a sub-document operation expects a path not
	// to exists, but the path was found in the document.
	ErrSubDocPathExists = gocbcore.ErrSubDocPathExists
	// ErrSubDocValueTooDeep occurs when a sub-document operation specifies a value
	// which is deeper than the depth limits of the sub-document specification.
	ErrSubDocValueTooDeep = gocbcore.ErrSubDocValueTooDeep
	// ErrSubDocBadCombo occurs when a multi-operation sub-document operation is
	// performed and operations within the package of ops conflict with each other.
	ErrSubDocBadCombo = gocbcore.ErrSubDocBadCombo
	// ErrSubDocBadMulti occurs when a multi-operation sub-document operation is
	// performed and operations within the package of ops conflict with each other.
	ErrSubDocBadMulti = gocbcore.ErrSubDocBadMulti
	// ErrSubDocSuccessDeleted occurs when a multi-operation sub-document operation
	// is performed on a soft-deleted document.
	ErrSubDocSuccessDeleted = gocbcore.ErrSubDocSuccessDeleted

	// ErrSubDocXattrInvalidFlagCombo occurs when an invalid set of
	// extended-attribute flags is passed to a sub-document operation.
	ErrSubDocXattrInvalidFlagCombo = gocbcore.ErrSubDocXattrInvalidFlagCombo
	// ErrSubDocXattrInvalidKeyCombo occurs when an invalid set of key operations
	// are specified for a extended-attribute sub-document operation.
	ErrSubDocXattrInvalidKeyCombo = gocbcore.ErrSubDocXattrInvalidKeyCombo
	// ErrSubDocXattrUnknownMacro occurs when an invalid macro value is specified.
	ErrSubDocXattrUnknownMacro = gocbcore.ErrSubDocXattrUnknownMacro
	// ErrSubDocXattrUnknownVAttr occurs when an invalid virtual attribute is specified.
	ErrSubDocXattrUnknownVAttr = gocbcore.ErrSubDocXattrUnknownVAttr
	// ErrSubDocXattrCannotModifyVAttr occurs when a mutation is attempted upon
	// a virtual attribute (which are immutable by definition).
	ErrSubDocXattrCannotModifyVAttr = gocbcore.ErrSubDocXattrCannotModifyVAttr
	// ErrSubDocMultiPathFailureDeleted occurs when a Multi Path Failure occurs on
	// a soft-deleted document.
	ErrSubDocMultiPathFailureDeleted = gocbcore.ErrSubDocMultiPathFailureDeleted
)

// IsKeyExistsError indicates whether the passed error is a
// key-value "Key Already Exists" error.
//
// Experimental: This API is subject to change at any time.
func IsKeyExistsError(err error) bool {
	return gocbcore.IsErrorStatus(err, gocbcore.StatusKeyExists)
}

// IsKeyNotFoundError indicates whether the passed error is a
// key-value "Key Not Found" error.
//
// Experimental: This API is subject to change at any time.
func IsKeyNotFoundError(err error) bool {
	return gocbcore.IsErrorStatus(err, gocbcore.StatusKeyNotFound)
}

// IsStatusBusyError indicates whether the passed error is a
// key-value "server is busy, try again later" error.
//
// Experimental: This API is subject to change at any time.
func IsStatusBusyError(err error) bool {
	return gocbcore.IsErrorStatus(err, gocbcore.StatusBusy)
}

// IsTmpFailError indicates whether the passed error is a
// key-value "temporary failure, try again later" error.
//
// Experimental: This API is subject to change at any time.
func IsTmpFailError(err error) bool {
	return gocbcore.IsErrorStatus(err, gocbcore.StatusTmpFail)
}

// ErrorCause returns the underlying error for an enhanced error.
func ErrorCause(err error) error {
	return gocbcore.ErrorCause(err)
}
