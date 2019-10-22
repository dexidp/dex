package gocbcore

import (
	"errors"
	"fmt"
	"strings"
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

// SubDocMutateError encapsulates errors that occur during sub-document mutations.
type SubDocMutateError struct {
	Err     error
	OpIndex int
}

func (e SubDocMutateError) Error() string {
	return fmt.Sprintf("subdocument mutation %d failed (%s)", e.OpIndex, e.Err.Error())
}

type timeoutError struct {
}

func (e timeoutError) Error() string {
	return "operation has timed out"
}
func (e timeoutError) Timeout() bool {
	return true
}

type networkError struct {
}

func (e networkError) Error() string {
	return "network error"
}

// Included for legacy support.
func (e networkError) NetworkError() bool {
	return true
}

type overloadError struct {
}

func (e overloadError) Error() string {
	return "queue overflowed"
}
func (e overloadError) Overload() bool {
	return true
}

type shutdownError struct {
}

func (e shutdownError) Error() string {
	return "connection shut down"
}

// Legacy
func (e shutdownError) ShutdownError() bool {
	return true
}

// KvError wraps key-value errors that occur within the SDK.
type KvError struct {
	Code        StatusCode
	Name        string
	Description string
	Context     string
	Ref         string
}

func getMemdErrorDesc(code StatusCode) string {
	switch code {
	case StatusSuccess:
		return "success"
	case StatusKeyNotFound:
		return "key not found"
	case StatusKeyExists:
		return "key already exists, if a cas was provided the key exists with a different cas"
	case StatusTooBig:
		return "document value was too large"
	case StatusInvalidArgs:
		return "invalid arguments"
	case StatusNotStored:
		return "document could not be stored"
	case StatusBadDelta:
		return "invalid delta was passed"
	case StatusNotMyVBucket:
		return "operation sent to incorrect server"
	case StatusNoBucket:
		return "not connected to a bucket"
	case StatusAuthStale:
		return "authentication context is stale, try re-authenticating"
	case StatusAuthError:
		return "authentication error"
	case StatusAuthContinue:
		return "more authentication steps needed"
	case StatusRangeError:
		return "requested value is outside range"
	case StatusAccessError:
		return "no access"
	case StatusNotInitialized:
		return "cluster is being initialized, requests are blocked"
	case StatusRollback:
		return "rollback is required"
	case StatusUnknownCommand:
		return "unknown command was received"
	case StatusOutOfMemory:
		return "server is out of memory"
	case StatusNotSupported:
		return "server does not support this command"
	case StatusInternalError:
		return "internal server error"
	case StatusBusy:
		return "server is busy, try again later"
	case StatusTmpFail:
		return "temporary failure occurred, try again later"
	case StatusSubDocPathNotFound:
		return "sub-document path does not exist"
	case StatusSubDocPathMismatch:
		return "type of element in sub-document path conflicts with type in document"
	case StatusSubDocPathInvalid:
		return "malformed sub-document path"
	case StatusSubDocPathTooBig:
		return "sub-document contains too many components"
	case StatusSubDocDocTooDeep:
		return "existing document contains too many levels of nesting"
	case StatusSubDocCantInsert:
		return "subdocument operation would invalidate the JSON"
	case StatusSubDocNotJson:
		return "existing document is not valid JSON"
	case StatusSubDocBadRange:
		return "existing numeric value is too large"
	case StatusSubDocBadDelta:
		return "numeric operation would yield a number that is too large, or " +
			"a zero delta was specified"
	case StatusSubDocPathExists:
		return "given path already exists in the document"
	case StatusSubDocValueTooDeep:
		return "value is too deep to insert"
	case StatusSubDocBadCombo:
		return "incorrectly matched subdocument operation types"
	case StatusSubDocBadMulti:
		return "could not execute one or more multi lookups or mutations"
	case StatusSubDocSuccessDeleted:
		return "document is soft-deleted"
	case StatusSubDocXattrInvalidFlagCombo:
		return "invalid xattr flag combination"
	case StatusSubDocXattrInvalidKeyCombo:
		return "invalid xattr key combination"
	case StatusSubDocXattrUnknownMacro:
		return "unknown xattr macro"
	case StatusSubDocXattrUnknownVAttr:
		return "unknown xattr virtual attribute"
	case StatusSubDocXattrCannotModifyVAttr:
		return "cannot modify virtual attributes"
	case StatusSubDocMultiPathFailureDeleted:
		return "sub-document multi-path error"
	}

	return ""
}

// Error returns the string representation of a kv error.
func (e KvError) Error() string {
	if e.Context != "" && e.Ref != "" {
		return fmt.Sprintf("%s (%s, context: %s, ref: %s)", e.Description, e.Name, e.Context, e.Ref)
	} else if e.Context != "" {
		return fmt.Sprintf("%s (%s, context: %s)", e.Description, e.Name, e.Context)
	} else if e.Ref != "" {
		return fmt.Sprintf("%s (%s, ref: %s)", e.Description, e.Name, e.Ref)
	} else if e.Name != "" && e.Description != "" {
		return fmt.Sprintf("%s (%s)", e.Description, e.Name)
	} else if e.Description != "" {
		return e.Description
	}

	return fmt.Sprintf("an unknown error occurred (%d)", e.Code)
}

// Temporary indicates whether this error is known to be temporary, and that
// attempting the operation again after a short delay should succeed.
func (e KvError) Temporary() bool {
	return e.Code == StatusOutOfMemory || e.Code == StatusTmpFail || e.Code == StatusBusy
}

// Success is a method to check if the error represents a successful operation.
//
// Deprecated:  This API should no longer be relied on.
func (e KvError) Success() bool {
	return e.Code == StatusSuccess
}

// KeyNotFound checks for the StatusKeyNotFound status code.
//
// Deprecated:  This API should no longer be relied on.
func (e KvError) KeyNotFound() bool {
	return e.Code == StatusKeyNotFound
}

// KeyExists checks for the StatusKeyExists status code.
//
// Deprecated:  This API should no longer be relied on.
func (e KvError) KeyExists() bool {
	return e.Code == StatusKeyExists
}

// AuthStale checks for the StatusAuthStale status code.
//
// Deprecated:  This API should no longer be relied on.
func (e KvError) AuthStale() bool {
	return e.Code == StatusAuthStale
}

// AuthError checks for the StatusAuthError status code.
//
// Deprecated:  This API should no longer be relied on.
func (e KvError) AuthError() bool {
	return e.Code == StatusAuthError
}

// AuthContinue checks for the StatusAuthContinue status code.
//
// Deprecated:  This API should no longer be relied on.
func (e KvError) AuthContinue() bool {
	return e.Code == StatusAuthContinue
}

// ValueTooBig checks for the StatusTooBig status code.
//
// Deprecated:  This API should no longer be relied on.
func (e KvError) ValueTooBig() bool {
	return e.Code == StatusTooBig
}

// NotStored checks for the StatusNotStored status code.
//
// Deprecated:  This API should no longer be relied on.
func (e KvError) NotStored() bool {
	return e.Code == StatusNotStored
}

// BadDelta checks for the StatusBadDelta status code.
//
// Deprecated:  This API should no longer be relied on.
func (e KvError) BadDelta() bool {
	return e.Code == StatusBadDelta
}

// NotMyVBucket checks for the StatusNotMyVBucket status code.
//
// Deprecated:  This API should no longer be relied on.
func (e KvError) NotMyVBucket() bool {
	return e.Code == StatusNotMyVBucket
}

// NoBucket checks for the StatusNoBucket status code.
//
// Deprecated:  This API should no longer be relied on.
func (e KvError) NoBucket() bool {
	return e.Code == StatusNoBucket
}

// RangeError checks for the StatusRangeError status code.
//
// Deprecated:  This API should no longer be relied on.
func (e KvError) RangeError() bool {
	return e.Code == StatusRangeError
}

// AccessError checks for the StatusAccessError status code.
//
// Deprecated:  This API should no longer be relied on.
func (e KvError) AccessError() bool {
	return e.Code == StatusAccessError
}

// NotIntializedError checks for the StatusNotInitialized status code.
//
// Deprecated:  This API should no longer be relied on.
func (e KvError) NotIntializedError() bool {
	return e.Code == StatusNotInitialized
}

// Rollback checks for the StatusRollback status code.
//
// Deprecated:  This API should no longer be relied on.
func (e KvError) Rollback() bool {
	return e.Code == StatusRollback
}

// UnknownCommandError checks for the StatusUnknownCommand status code.
//
// Deprecated:  This API should no longer be relied on.
func (e KvError) UnknownCommandError() bool {
	return e.Code == StatusUnknownCommand
}

// NotSupportedError checks for the StatusNotSupported status code.
//
// Deprecated:  This API should no longer be relied on.
func (e KvError) NotSupportedError() bool {
	return e.Code == StatusNotSupported
}

// InternalError checks for the StatusInternalError status code.
//
// Deprecated:  This API should no longer be relied on.
func (e KvError) InternalError() bool {
	return e.Code == StatusInternalError
}

// BusyError checks for the StatusBusy status code.
//
// Deprecated:  This API should no longer be relied on.
func (e KvError) BusyError() bool {
	return e.Code == StatusBusy
}

func newSimpleError(code StatusCode) *KvError {
	return &KvError{
		Code:        code,
		Description: getMemdErrorDesc(code),
	}
}

type streamEndError struct {
	code streamEndStatus
}

func (e streamEndError) Error() string {
	switch e.code {
	case streamEndOK:
		return "success"
	case streamEndClosed:
		return "stream closed"
	case streamEndStateChanged:
		return "state changed"
	case streamEndDisconnected:
		return "disconnected"
	case streamEndTooSlow:
		return "too slow"
	default:
		return fmt.Sprintf("stream closed for unknown reason (%d)", e.code)
	}
}

func (e streamEndError) Success() bool {
	return e.code == streamEndOK
}
func (e streamEndError) Closed() bool {
	return e.code == streamEndClosed
}
func (e streamEndError) StateChanged() bool {
	return e.code == streamEndStateChanged
}
func (e streamEndError) Disconnected() bool {
	return e.code == streamEndDisconnected
}
func (e streamEndError) TooSlow() bool {
	return e.code == streamEndTooSlow
}

var (
	// ErrNoAuthMethod occurs when the server does not support any of the
	// authentication methods that the client finds suitable.
	ErrNoAuthMethod = errors.New("no supported auth")

	// ErrDispatchFail occurs when the request router fails to dispatch an operation
	ErrDispatchFail = errors.New("failed to dispatch operation")

	// ErrBadHosts occurs when the list of hosts specified cannot be contacted.
	ErrBadHosts = errors.New("failed to connect to any of the specified hosts")

	// ErrProtocol occurs when the server responds with unexpected or unparseable data.
	ErrProtocol = errors.New("failed to parse server response")

	// ErrNoReplicas occurs when no replicas respond in time
	ErrNoReplicas = errors.New("no replicas responded in time")

	// ErrNoServer occurs when no server is available to service a keys vbucket.
	ErrNoServer = errors.New("no server available for this vbucket")

	// ErrInvalidServer occurs when an explicit, but invalid server index is specified.
	ErrInvalidServer = errors.New("specific server index is invalid")

	// ErrInvalidVBucket occurs when an explicit, but invalid vbucket index is specified.
	ErrInvalidVBucket = errors.New("specific vbucket index is invalid")

	// ErrInvalidReplica occurs when an explicit, but invalid replica index is specified.
	ErrInvalidReplica = errors.New("specific server index is invalid")

	// ErrInvalidCert occurs when a certificate that is not useable is passed to an Agent.
	ErrInvalidCert = errors.New("certificate is invalid")

	// ErrCliInternalError indicates an internal error occurred within the client.
	ErrCliInternalError = errors.New("client internal error")

	// ErrCancelled occurs when an operation has been cancelled by the user.
	ErrCancelled = errors.New("Operation was cancelled by the user.")

	// ErrInvalidCredentials is returned when an invalid set of credentials is provided for a service.
	ErrInvalidCredentials = errors.New("An invalid set of credentials was provided.")

	// ErrInvalidService occurs when an invalid service is specified for an operation.
	ErrInvalidService = errors.New("Invalid service specified")

	// ErrNoMgmtService occurs when no mgmt services are available for a request.
	ErrNoMgmtService = errors.New("No available management nodes.")

	// ErrNoCapiService occurs when no capi services are available for a request.
	ErrNoCapiService = errors.New("No available capi nodes.")

	// ErrNoN1qlService occurs when no N1QL services are available for a request.
	ErrNoN1qlService = errors.New("No available n1ql nodes.")

	// ErrNoFtsService occurs when no FTS services are available for a request.
	ErrNoFtsService = errors.New("No available fts nodes.")

	// ErrNoCbasService occurs when no CBAS services are available for a request.
	ErrNoCbasService = errors.New("No available cbas nodes.")

	// ErrNonZeroCas occurs when an operation that require a CAS value of 0 is used with a non-zero value.
	ErrNonZeroCas = errors.New("Cas value must be 0.")

	// ErrUnsupportedStatsTarget occurs when a stats operation is performed with an unsupported Target.
	ErrUnsupportedStatsTarget = errors.New("Must specify a supported StatsTarget.")

	// ErrShutdown occurs when operations are performed on a previously closed Agent.
	ErrShutdown = &shutdownError{}

	// ErrOverload occurs when too many operations are dispatched and all queues are full.
	ErrOverload = &overloadError{}

	// ErrNetwork occurs when network failures prevent an operation from succeeding.
	ErrNetwork = &networkError{}

	// ErrTimeout occurs when an operation does not receive a response in a timely manner.
	ErrTimeout = &timeoutError{}

	// ErrStreamClosed occurs when a DCP stream is closed gracefully.
	ErrStreamClosed = &streamEndError{streamEndClosed}

	// ErrStreamStateChanged occurs when a DCP stream is interrupted by failover.
	ErrStreamStateChanged = &streamEndError{streamEndStateChanged}

	// ErrStreamDisconnected occurs when a DCP stream is disconnected.
	ErrStreamDisconnected = &streamEndError{streamEndDisconnected}

	// ErrStreamTooSlow occurs when a DCP stream is cancelled due to the application
	// not keeping up with the rate of flow of DCP events sent by the server.
	ErrStreamTooSlow = &streamEndError{streamEndTooSlow}

	// ErrKeyNotFound occurs when an operation is performed on a key that does not exist.
	ErrKeyNotFound = newSimpleError(StatusKeyNotFound)

	// ErrKeyExists occurs when an operation is performed on a key that could not be found.
	ErrKeyExists = newSimpleError(StatusKeyExists)

	// ErrTooBig occurs when an operation attempts to store more data in a single document
	// than the server is capable of storing (by default, this is a 20MB limit).
	ErrTooBig = newSimpleError(StatusTooBig)

	// ErrInvalidArgs occurs when the server receives invalid arguments for an operation.
	ErrInvalidArgs = newSimpleError(StatusInvalidArgs)

	// ErrNotStored occurs when the server fails to store a key.
	ErrNotStored = newSimpleError(StatusNotStored)

	// ErrBadDelta occurs when an invalid delta value is specified to a counter operation.
	ErrBadDelta = newSimpleError(StatusBadDelta)

	// ErrNotMyVBucket occurs when an operation is dispatched to a server which is
	// non-authoritative for a specific vbucket.
	ErrNotMyVBucket = newSimpleError(StatusNotMyVBucket)

	// ErrNoBucket occurs when no bucket was selected on a connection.
	ErrNoBucket = newSimpleError(StatusNoBucket)

	// ErrAuthStale occurs when authentication credentials have become invalidated.
	ErrAuthStale = newSimpleError(StatusAuthStale)

	// ErrAuthError occurs when the authentication information provided was not valid.
	ErrAuthError = newSimpleError(StatusAuthError)

	// ErrAuthContinue occurs in multi-step authentication when more authentication
	// work needs to be performed in order to complete the authentication process.
	ErrAuthContinue = newSimpleError(StatusAuthContinue)

	// ErrRangeError occurs when the range specified to the server is not valid.
	ErrRangeError = newSimpleError(StatusRangeError)

	// ErrRollback occurs when a DCP stream fails to open due to a rollback having
	// previously occurred since the last time the stream was opened.
	ErrRollback = newSimpleError(StatusRollback)

	// ErrAccessError occurs when an access error occurs.
	ErrAccessError = newSimpleError(StatusAccessError)

	// ErrNotInitialized is sent by servers which are still initializing, and are not
	// yet ready to accept operations on behalf of a particular bucket.
	ErrNotInitialized = newSimpleError(StatusNotInitialized)

	// ErrUnknownCommand occurs when an unknown operation is sent to a server.
	ErrUnknownCommand = newSimpleError(StatusUnknownCommand)

	// ErrOutOfMemory occurs when the server cannot service a request due to memory
	// limitations.
	ErrOutOfMemory = newSimpleError(StatusOutOfMemory)

	// ErrNotSupported occurs when an operation is understood by the server, but that
	// operation is not supported on this server (occurs for a variety of reasons).
	ErrNotSupported = newSimpleError(StatusNotSupported)

	// ErrInternalError occurs when internal errors prevent the server from processing
	// your request.
	ErrInternalError = newSimpleError(StatusInternalError)

	// ErrBusy occurs when the server is too busy to process your request right away.
	// Attempting the operation at a later time will likely succeed.
	ErrBusy = newSimpleError(StatusBusy)

	// ErrTmpFail occurs when a temporary failure is preventing the server from
	// processing your request.
	ErrTmpFail = newSimpleError(StatusTmpFail)

	// ErrSubDocPathNotFound occurs when a sub-document operation targets a path
	// which does not exist in the specifie document.
	ErrSubDocPathNotFound = newSimpleError(StatusSubDocPathNotFound)

	// ErrSubDocPathMismatch occurs when a sub-document operation specifies a path
	// which does not match the document structure (field access on an array).
	ErrSubDocPathMismatch = newSimpleError(StatusSubDocPathMismatch)

	// ErrSubDocPathInvalid occurs when a sub-document path could not be parsed.
	ErrSubDocPathInvalid = newSimpleError(StatusSubDocPathInvalid)

	// ErrSubDocPathTooBig occurs when a sub-document path is too big.
	ErrSubDocPathTooBig = newSimpleError(StatusSubDocPathTooBig)

	// ErrSubDocDocTooDeep occurs when an operation would cause a document to be
	// nested beyond the depth limits allowed by the sub-document specification.
	ErrSubDocDocTooDeep = newSimpleError(StatusSubDocDocTooDeep)

	// ErrSubDocCantInsert occurs when a sub-document operation could not insert.
	ErrSubDocCantInsert = newSimpleError(StatusSubDocCantInsert)

	// ErrSubDocNotJson occurs when a sub-document operation is performed on a
	// document which is not JSON.
	ErrSubDocNotJson = newSimpleError(StatusSubDocNotJson)

	// ErrSubDocBadRange occurs when a sub-document operation is performed with
	// a bad range.
	ErrSubDocBadRange = newSimpleError(StatusSubDocBadRange)

	// ErrSubDocBadDelta occurs when a sub-document counter operation is performed
	// and the specified delta is not valid.
	ErrSubDocBadDelta = newSimpleError(StatusSubDocBadDelta)

	// ErrSubDocPathExists occurs when a sub-document operation expects a path not
	// to exists, but the path was found in the document.
	ErrSubDocPathExists = newSimpleError(StatusSubDocPathExists)

	// ErrSubDocValueTooDeep occurs when a sub-document operation specifies a value
	// which is deeper than the depth limits of the sub-document specification.
	ErrSubDocValueTooDeep = newSimpleError(StatusSubDocValueTooDeep)

	// ErrSubDocBadCombo occurs when a multi-operation sub-document operation is
	// performed and operations within the package of ops conflict with each other.
	ErrSubDocBadCombo = newSimpleError(StatusSubDocBadCombo)

	// ErrSubDocBadMulti occurs when a multi-operation sub-document operation is
	// performed and operations within the package of ops conflict with each other.
	ErrSubDocBadMulti = newSimpleError(StatusSubDocBadMulti)

	// ErrSubDocSuccessDeleted occurs when a multi-operation sub-document operation
	// is performed on a soft-deleted document.
	ErrSubDocSuccessDeleted = newSimpleError(StatusSubDocSuccessDeleted)

	// ErrSubDocXattrInvalidFlagCombo occurs when an invalid set of
	// extended-attribute flags is passed to a sub-document operation.
	ErrSubDocXattrInvalidFlagCombo = newSimpleError(StatusSubDocXattrInvalidFlagCombo)

	// ErrSubDocXattrInvalidKeyCombo occurs when an invalid set of key operations
	// are specified for a extended-attribute sub-document operation.
	ErrSubDocXattrInvalidKeyCombo = newSimpleError(StatusSubDocXattrInvalidKeyCombo)

	// ErrSubDocXattrUnknownMacro occurs when an invalid macro value is specified.
	ErrSubDocXattrUnknownMacro = newSimpleError(StatusSubDocXattrUnknownMacro)

	// ErrSubDocXattrUnknownVAttr occurs when an invalid virtual attribute is specified.
	ErrSubDocXattrUnknownVAttr = newSimpleError(StatusSubDocXattrUnknownVAttr)

	// ErrSubDocXattrCannotModifyVAttr occurs when a mutation is attempted upon
	// a virtual attribute (which are immutable by definition).
	ErrSubDocXattrCannotModifyVAttr = newSimpleError(StatusSubDocXattrCannotModifyVAttr)

	// ErrSubDocMultiPathFailureDeleted occurs when a Multi Path Failure occurs on
	// a soft-deleted document.
	ErrSubDocMultiPathFailureDeleted = newSimpleError(StatusSubDocMultiPathFailureDeleted)
)

func getStreamEndError(code streamEndStatus) error {
	switch code {
	case streamEndOK:
		return nil
	case streamEndClosed:
		return ErrStreamClosed
	case streamEndStateChanged:
		return ErrStreamStateChanged
	case streamEndDisconnected:
		return ErrStreamDisconnected
	case streamEndTooSlow:
		return ErrStreamTooSlow
	default:
		return &streamEndError{code}
	}
}

func findMemdError(code StatusCode) (bool, error) {
	switch code {
	case StatusSuccess:
		return true, nil
	case StatusKeyNotFound:
		return true, ErrKeyNotFound
	case StatusKeyExists:
		return true, ErrKeyExists
	case StatusTooBig:
		return true, ErrTooBig
	case StatusInvalidArgs:
		return true, ErrInvalidArgs
	case StatusNotStored:
		return true, ErrNotStored
	case StatusBadDelta:
		return true, ErrBadDelta
	case StatusNotMyVBucket:
		return true, ErrNotMyVBucket
	case StatusNoBucket:
		return true, ErrNoBucket
	case StatusAuthStale:
		return true, ErrAuthStale
	case StatusAuthError:
		return true, ErrAuthError
	case StatusAuthContinue:
		return true, ErrAuthContinue
	case StatusRangeError:
		return true, ErrRangeError
	case StatusAccessError:
		return true, ErrAccessError
	case StatusNotInitialized:
		return true, ErrNotInitialized
	case StatusRollback:
		return true, ErrRollback
	case StatusUnknownCommand:
		return true, ErrUnknownCommand
	case StatusOutOfMemory:
		return true, ErrOutOfMemory
	case StatusNotSupported:
		return true, ErrNotSupported
	case StatusInternalError:
		return true, ErrInternalError
	case StatusBusy:
		return true, ErrBusy
	case StatusTmpFail:
		return true, ErrTmpFail
	case StatusSubDocPathNotFound:
		return true, ErrSubDocPathNotFound
	case StatusSubDocPathMismatch:
		return true, ErrSubDocPathMismatch
	case StatusSubDocPathInvalid:
		return true, ErrSubDocPathInvalid
	case StatusSubDocPathTooBig:
		return true, ErrSubDocPathTooBig
	case StatusSubDocDocTooDeep:
		return true, ErrSubDocDocTooDeep
	case StatusSubDocCantInsert:
		return true, ErrSubDocCantInsert
	case StatusSubDocNotJson:
		return true, ErrSubDocNotJson
	case StatusSubDocBadRange:
		return true, ErrSubDocBadRange
	case StatusSubDocBadDelta:
		return true, ErrSubDocBadDelta
	case StatusSubDocPathExists:
		return true, ErrSubDocPathExists
	case StatusSubDocValueTooDeep:
		return true, ErrSubDocValueTooDeep
	case StatusSubDocBadCombo:
		return true, ErrSubDocBadCombo
	case StatusSubDocBadMulti:
		return true, ErrSubDocBadMulti
	case StatusSubDocSuccessDeleted:
		return true, ErrSubDocSuccessDeleted
	case StatusSubDocXattrInvalidFlagCombo:
		return true, ErrSubDocXattrInvalidFlagCombo
	case StatusSubDocXattrInvalidKeyCombo:
		return true, ErrSubDocXattrInvalidKeyCombo
	case StatusSubDocXattrUnknownMacro:
		return true, ErrSubDocXattrUnknownMacro
	case StatusSubDocXattrUnknownVAttr:
		return true, ErrSubDocXattrUnknownVAttr
	case StatusSubDocXattrCannotModifyVAttr:
		return true, ErrSubDocXattrCannotModifyVAttr
	case StatusSubDocMultiPathFailureDeleted:
		return true, ErrSubDocMultiPathFailureDeleted
	}
	return false, nil
}

// IsErrorStatus is a helper function which allows you to quickly check
// if a particular error object corresponds with a specific memcached
// status code in a single operation.
func IsErrorStatus(err error, code StatusCode) bool {
	if memdErr, ok := err.(*KvError); ok {
		return memdErr.Code == code
	}
	return false
}

// ErrorCause returns an error object representing the underlying cause
// for an error (without detailed information).
func ErrorCause(err error) error {
	if typedErr, ok := err.(*KvError); ok {
		if ok, err := findMemdError(typedErr.Code); ok {
			return err
		}
		return newSimpleError(typedErr.Code)
	}
	return err
}
