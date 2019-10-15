package gocbcore

const (
	goCbCoreVersionStr = "v7.1.14"
)

type commandMagic uint8

const (
	reqMagic = commandMagic(0x80)
	resMagic = commandMagic(0x81)

	altResMagic = commandMagic(0x18)
)

type frameExtraType uint16

const (
	srvDurationFrameExtra = frameExtraType(0)
)

// commandCode for memcached packets.
type commandCode uint8

const (
	cmdGet                  = commandCode(0x00)
	cmdSet                  = commandCode(0x01)
	cmdAdd                  = commandCode(0x02)
	cmdReplace              = commandCode(0x03)
	cmdDelete               = commandCode(0x04)
	cmdIncrement            = commandCode(0x05)
	cmdDecrement            = commandCode(0x06)
	cmdNoop                 = commandCode(0x0a)
	cmdAppend               = commandCode(0x0e)
	cmdPrepend              = commandCode(0x0f)
	cmdStat                 = commandCode(0x10)
	cmdTouch                = commandCode(0x1c)
	cmdGAT                  = commandCode(0x1d)
	cmdHello                = commandCode(0x1f)
	cmdSASLListMechs        = commandCode(0x20)
	cmdSASLAuth             = commandCode(0x21)
	cmdSASLStep             = commandCode(0x22)
	cmdGetAllVBSeqnos       = commandCode(0x48)
	cmdDcpOpenConnection    = commandCode(0x50)
	cmdDcpAddStream         = commandCode(0x51)
	cmdDcpCloseStream       = commandCode(0x52)
	cmdDcpStreamReq         = commandCode(0x53)
	cmdDcpGetFailoverLog    = commandCode(0x54)
	cmdDcpStreamEnd         = commandCode(0x55)
	cmdDcpSnapshotMarker    = commandCode(0x56)
	cmdDcpMutation          = commandCode(0x57)
	cmdDcpDeletion          = commandCode(0x58)
	cmdDcpExpiration        = commandCode(0x59)
	cmdDcpFlush             = commandCode(0x5a)
	cmdDcpSetVbucketState   = commandCode(0x5b)
	cmdDcpNoop              = commandCode(0x5c)
	cmdDcpBufferAck         = commandCode(0x5d)
	cmdDcpControl           = commandCode(0x5e)
	cmdGetReplica           = commandCode(0x83)
	cmdSelectBucket         = commandCode(0x89)
	cmdObserveSeqNo         = commandCode(0x91)
	cmdObserve              = commandCode(0x92)
	cmdGetLocked            = commandCode(0x94)
	cmdUnlockKey            = commandCode(0x95)
	cmdGetMeta              = commandCode(0xa0)
	cmdSetMeta              = commandCode(0xa2)
	cmdDelMeta              = commandCode(0xa8)
	cmdGetClusterConfig     = commandCode(0xb5)
	cmdGetRandom            = commandCode(0xb6)
	cmdSubDocGet            = commandCode(0xc5)
	cmdSubDocExists         = commandCode(0xc6)
	cmdSubDocDictAdd        = commandCode(0xc7)
	cmdSubDocDictSet        = commandCode(0xc8)
	cmdSubDocDelete         = commandCode(0xc9)
	cmdSubDocReplace        = commandCode(0xca)
	cmdSubDocArrayPushLast  = commandCode(0xcb)
	cmdSubDocArrayPushFirst = commandCode(0xcc)
	cmdSubDocArrayInsert    = commandCode(0xcd)
	cmdSubDocArrayAddUnique = commandCode(0xce)
	cmdSubDocCounter        = commandCode(0xcf)
	cmdSubDocMultiLookup    = commandCode(0xd0)
	cmdSubDocMultiMutation  = commandCode(0xd1)
	cmdSubDocGetCount       = commandCode(0xd2)
	cmdGetErrorMap          = commandCode(0xfe)
)

// HelloFeature represents a feature code included in a memcached
// HELLO operation.
type HelloFeature uint16

const (
	// FeatureDatatype indicates support for Datatype fields.
	FeatureDatatype = HelloFeature(0x01)

	// FeatureTls indicates support for TLS
	FeatureTls = HelloFeature(0x02)

	// FeatureTcpNoDelay indicates support for TCP no-delay.
	FeatureTcpNoDelay = HelloFeature(0x03)

	// FeatureSeqNo indicates support for mutation tokens.
	FeatureSeqNo = HelloFeature(0x04)

	// FeatureTcpDelay indicates support for TCP delay.
	FeatureTcpDelay = HelloFeature(0x05)

	// FeatureXattr indicates support for document xattrs.
	FeatureXattr = HelloFeature(0x06)

	// FeatureXerror indicates support for extended errors.
	FeatureXerror = HelloFeature(0x07)

	// FeatureSelectBucket indicates support for the SelectBucket operation.
	FeatureSelectBucket = HelloFeature(0x08)

	// FeatureCollections indicates support for collections.
	FeatureCollections = HelloFeature(0x09)

	// FeatureSnappy indicates support for snappy compressed documents.
	FeatureSnappy = HelloFeature(0x0a)

	// FeatureJson indicates support for JSON datatype data.
	FeatureJson = HelloFeature(0x0b)

	// FeatureDuplex indicates support for duplex communications.
	FeatureDuplex = HelloFeature(0x0c)

	// FeatureClusterMapNotif indicates support for cluster-map update notifications.
	FeatureClusterMapNotif = HelloFeature(0x0d)

	// FeatureUnorderedExec indicates support for unordered execution of operations.
	FeatureUnorderedExec = HelloFeature(0x0e)

	// FeatureDurations indicates support for server durations.
	FeatureDurations = HelloFeature(0xf)
)

// StatusCode represents a memcached response status.
type StatusCode uint16

const (
	// StatusSuccess indicates the operation completed successfully.
	StatusSuccess = StatusCode(0x00)

	// StatusKeyNotFound occurs when an operation is performed on a key that does not exist.
	StatusKeyNotFound = StatusCode(0x01)

	// StatusKeyExists occurs when an operation is performed on a key that could not be found.
	StatusKeyExists = StatusCode(0x02)

	// StatusTooBig occurs when an operation attempts to store more data in a single document
	// than the server is capable of storing (by default, this is a 20MB limit).
	StatusTooBig = StatusCode(0x03)

	// StatusInvalidArgs occurs when the server receives invalid arguments for an operation.
	StatusInvalidArgs = StatusCode(0x04)

	// StatusNotStored occurs when the server fails to store a key.
	StatusNotStored = StatusCode(0x05)

	// StatusBadDelta occurs when an invalid delta value is specified to a counter operation.
	StatusBadDelta = StatusCode(0x06)

	// StatusNotMyVBucket occurs when an operation is dispatched to a server which is
	// non-authoritative for a specific vbucket.
	StatusNotMyVBucket = StatusCode(0x07)

	// StatusNoBucket occurs when no bucket was selected on a connection.
	StatusNoBucket = StatusCode(0x08)

	// StatusLocked occurs when an operation fails due to the document being locked.
	StatusLocked = StatusCode(0x09)

	// StatusAuthStale occurs when authentication credentials have become invalidated.
	StatusAuthStale = StatusCode(0x1f)

	// StatusAuthError occurs when the authentication information provided was not valid.
	StatusAuthError = StatusCode(0x20)

	// StatusAuthContinue occurs in multi-step authentication when more authentication
	// work needs to be performed in order to complete the authentication process.
	StatusAuthContinue = StatusCode(0x21)

	// StatusRangeError occurs when the range specified to the server is not valid.
	StatusRangeError = StatusCode(0x22)

	// StatusRollback occurs when a DCP stream fails to open due to a rollback having
	// previously occurred since the last time the stream was opened.
	StatusRollback = StatusCode(0x23)

	// StatusAccessError occurs when an access error occurs.
	StatusAccessError = StatusCode(0x24)

	// StatusNotInitialized is sent by servers which are still initializing, and are not
	// yet ready to accept operations on behalf of a particular bucket.
	StatusNotInitialized = StatusCode(0x25)

	// StatusUnknownCommand occurs when an unknown operation is sent to a server.
	StatusUnknownCommand = StatusCode(0x81)

	// StatusOutOfMemory occurs when the server cannot service a request due to memory
	// limitations.
	StatusOutOfMemory = StatusCode(0x82)

	// StatusNotSupported occurs when an operation is understood by the server, but that
	// operation is not supported on this server (occurs for a variety of reasons).
	StatusNotSupported = StatusCode(0x83)

	// StatusInternalError occurs when internal errors prevent the server from processing
	// your request.
	StatusInternalError = StatusCode(0x84)

	// StatusBusy occurs when the server is too busy to process your request right away.
	// Attempting the operation at a later time will likely succeed.
	StatusBusy = StatusCode(0x85)

	// StatusTmpFail occurs when a temporary failure is preventing the server from
	// processing your request.
	StatusTmpFail = StatusCode(0x86)

	// StatusSubDocPathNotFound occurs when a sub-document operation targets a path
	// which does not exist in the specifie document.
	StatusSubDocPathNotFound = StatusCode(0xc0)

	// StatusSubDocPathMismatch occurs when a sub-document operation specifies a path
	// which does not match the document structure (field access on an array).
	StatusSubDocPathMismatch = StatusCode(0xc1)

	// StatusSubDocPathInvalid occurs when a sub-document path could not be parsed.
	StatusSubDocPathInvalid = StatusCode(0xc2)

	// StatusSubDocPathTooBig occurs when a sub-document path is too big.
	StatusSubDocPathTooBig = StatusCode(0xc3)

	// StatusSubDocDocTooDeep occurs when an operation would cause a document to be
	// nested beyond the depth limits allowed by the sub-document specification.
	StatusSubDocDocTooDeep = StatusCode(0xc4)

	// StatusSubDocCantInsert occurs when a sub-document operation could not insert.
	StatusSubDocCantInsert = StatusCode(0xc5)

	// StatusSubDocNotJson occurs when a sub-document operation is performed on a
	// document which is not JSON.
	StatusSubDocNotJson = StatusCode(0xc6)

	// StatusSubDocBadRange occurs when a sub-document operation is performed with
	// a bad range.
	StatusSubDocBadRange = StatusCode(0xc7)

	// StatusSubDocBadDelta occurs when a sub-document counter operation is performed
	// and the specified delta is not valid.
	StatusSubDocBadDelta = StatusCode(0xc8)

	// StatusSubDocPathExists occurs when a sub-document operation expects a path not
	// to exists, but the path was found in the document.
	StatusSubDocPathExists = StatusCode(0xc9)

	// StatusSubDocValueTooDeep occurs when a sub-document operation specifies a value
	// which is deeper than the depth limits of the sub-document specification.
	StatusSubDocValueTooDeep = StatusCode(0xca)

	// StatusSubDocBadCombo occurs when a multi-operation sub-document operation is
	// performed and operations within the package of ops conflict with each other.
	StatusSubDocBadCombo = StatusCode(0xcb)

	// StatusSubDocBadMulti occurs when a multi-operation sub-document operation is
	// performed and operations within the package of ops conflict with each other.
	StatusSubDocBadMulti = StatusCode(0xcc)

	// StatusSubDocSuccessDeleted occurs when a multi-operation sub-document operation
	// is performed on a soft-deleted document.
	StatusSubDocSuccessDeleted = StatusCode(0xcd)

	// StatusSubDocXattrInvalidFlagCombo occurs when an invalid set of
	// extended-attribute flags is passed to a sub-document operation.
	StatusSubDocXattrInvalidFlagCombo = StatusCode(0xce)

	// StatusSubDocXattrInvalidKeyCombo occurs when an invalid set of key operations
	// are specified for a extended-attribute sub-document operation.
	StatusSubDocXattrInvalidKeyCombo = StatusCode(0xcf)

	// StatusSubDocXattrUnknownMacro occurs when an invalid macro value is specified.
	StatusSubDocXattrUnknownMacro = StatusCode(0xd0)

	// StatusSubDocXattrUnknownVAttr occurs when an invalid virtual attribute is specified.
	StatusSubDocXattrUnknownVAttr = StatusCode(0xd1)

	// StatusSubDocXattrCannotModifyVAttr occurs when a mutation is attempted upon
	// a virtual attribute (which are immutable by definition).
	StatusSubDocXattrCannotModifyVAttr = StatusCode(0xd2)

	// StatusSubDocMultiPathFailureDeleted occurs when a Multi Path Failure occurs on
	// a soft-deleted document.
	StatusSubDocMultiPathFailureDeleted = StatusCode(0xd3)
)

type streamEndStatus uint32

const (
	streamEndOK           = streamEndStatus(0x00)
	streamEndClosed       = streamEndStatus(0x01)
	streamEndStateChanged = streamEndStatus(0x02)
	streamEndDisconnected = streamEndStatus(0x03)
	streamEndTooSlow      = streamEndStatus(0x04)
)

type bucketType int

const (
	bktTypeInvalid   bucketType = 0
	bktTypeCouchbase            = iota
	bktTypeMemcached            = iota
)

// VbucketState represents the state of a particular vbucket on a particular server.
type VbucketState uint32

const (
	// VbucketStateActive indicates the vbucket is active on this server
	VbucketStateActive = VbucketState(0x01)

	// VbucketStateReplica indicates the vbucket is a replica on this server
	VbucketStateReplica = VbucketState(0x02)

	// VbucketStatePending indicates the vbucket is preparing to become active on this server.
	VbucketStatePending = VbucketState(0x03)

	// VbucketStateDead indicates the vbucket is no longer valid on this server.
	VbucketStateDead = VbucketState(0x04)
)

// SetMetaOption represents possible option values for a SetMeta operation.
type SetMetaOption uint32

const (
	// ForceMetaOp disables conflict resolution for the document and allows the
	// operation to be applied to an active, pending, or replica vbucket.
	ForceMetaOp = SetMetaOption(0x01)

	// UseLwwConflictResolution switches to Last-Write-Wins conflict resolution
	// for the document.
	UseLwwConflictResolution = SetMetaOption(0x02)

	// RegenerateCas causes the server to invalidate the current CAS value for
	// a document, and to generate a new one.
	RegenerateCas = SetMetaOption(0x04)

	// SkipConflictResolution disables conflict resolution for the document.
	SkipConflictResolution = SetMetaOption(0x08)

	// IsExpiration indicates that the message is for an expired document.
	IsExpiration = SetMetaOption(0x10)
)

// KeyState represents the various storage states of a key on the server.
type KeyState uint8

const (
	// KeyStateNotPersisted indicates the key is in memory, but not yet written to disk.
	KeyStateNotPersisted = KeyState(0x00)

	// KeyStatePersisted indicates that the key has been written to disk.
	KeyStatePersisted = KeyState(0x01)

	// KeyStateNotFound indicates that the key is not found in memory or on disk.
	KeyStateNotFound = KeyState(0x80)

	// KeyStateDeleted indicates that the key has been written to disk as deleted.
	KeyStateDeleted = KeyState(0x81)
)

// SubDocOpType specifies the type of a sub-document operation.
type SubDocOpType uint8

const (
	// SubDocOpGet indicates the operation is a sub-document `Get` operation.
	SubDocOpGet = SubDocOpType(cmdSubDocGet)

	// SubDocOpExists indicates the operation is a sub-document `Exists` operation.
	SubDocOpExists = SubDocOpType(cmdSubDocExists)

	// SubDocOpGetCount indicates the operation is a sub-document `GetCount` operation.
	SubDocOpGetCount = SubDocOpType(cmdSubDocGetCount)

	// SubDocOpDictAdd indicates the operation is a sub-document `Add` operation.
	SubDocOpDictAdd = SubDocOpType(cmdSubDocDictAdd)

	// SubDocOpDictSet indicates the operation is a sub-document `Set` operation.
	SubDocOpDictSet = SubDocOpType(cmdSubDocDictSet)

	// SubDocOpDelete indicates the operation is a sub-document `Remove` operation.
	SubDocOpDelete = SubDocOpType(cmdSubDocDelete)

	// SubDocOpReplace indicates the operation is a sub-document `Replace` operation.
	SubDocOpReplace = SubDocOpType(cmdSubDocReplace)

	// SubDocOpArrayPushLast indicates the operation is a sub-document `ArrayPushLast` operation.
	SubDocOpArrayPushLast = SubDocOpType(cmdSubDocArrayPushLast)

	// SubDocOpArrayPushFirst indicates the operation is a sub-document `ArrayPushFirst` operation.
	SubDocOpArrayPushFirst = SubDocOpType(cmdSubDocArrayPushFirst)

	// SubDocOpArrayInsert indicates the operation is a sub-document `ArrayInsert` operation.
	SubDocOpArrayInsert = SubDocOpType(cmdSubDocArrayInsert)

	// SubDocOpArrayAddUnique indicates the operation is a sub-document `ArrayAddUnique` operation.
	SubDocOpArrayAddUnique = SubDocOpType(cmdSubDocArrayAddUnique)

	// SubDocOpCounter indicates the operation is a sub-document `Counter` operation.
	SubDocOpCounter = SubDocOpType(cmdSubDocCounter)

	// SubDocOpGetDoc represents a full document retrieval, for use with extended attribute ops.
	SubDocOpGetDoc = SubDocOpType(cmdGet)

	// SubDocOpSetDoc represents a full document set, for use with extended attribute ops.
	SubDocOpSetDoc = SubDocOpType(cmdSet)

	// SubDocOpAddDoc represents a full document add, for use with extended attribute ops.
	SubDocOpAddDoc = SubDocOpType(cmdAdd)

	// SubDocOpDeleteDoc represents a full document delete, for use with extended attribute ops.
	SubDocOpDeleteDoc = SubDocOpType(cmdDelete)
)

// DcpOpenFlag specifies flags for DCP connections configured when the stream is opened.
type DcpOpenFlag uint32

const (
	// DcpOpenFlagProducer indicates this connection wants the other end to be a producer.
	DcpOpenFlagProducer = DcpOpenFlag(0x01)

	// DcpOpenFlagNotifier indicates this connection wants the other end to be a notifier.
	DcpOpenFlagNotifier = DcpOpenFlag(0x02)

	// DcpOpenFlagIncludeXattrs indicates the client wishes to receive extended attributes.
	DcpOpenFlagIncludeXattrs = DcpOpenFlag(0x04)

	// DcpOpenFlagNoValue indicates the client does not wish to receive mutation values.
	DcpOpenFlagNoValue = DcpOpenFlag(0x08)
)

// DcpStreamAddFlag specifies flags for DCP streams configured when the stream is opened.
type DcpStreamAddFlag uint32

const (
	//DcpStreamAddFlagDiskOnly indicates that stream should only send items if they are on disk
	DcpStreamAddFlagDiskOnly = DcpStreamAddFlag(0x02)

	// DcpStreamAddFlagLatest indicates this stream wants to get data up to the latest seqno.
	DcpStreamAddFlagLatest = DcpStreamAddFlag(0x04)

	// DcpStreamAddFlagActiveOnly indicates this stream should only connect to an active vbucket.
	DcpStreamAddFlagActiveOnly = DcpStreamAddFlag(0x10)

	// DcpStreamAddFlagStrictVBUUID indicates the vbuuid must match unless the start seqno
	// is 0 and the vbuuid is also 0.
	DcpStreamAddFlagStrictVBUUID = DcpStreamAddFlag(0x20)
)

// DatatypeFlag specifies data flags for the value of a document.
type DatatypeFlag uint8

const (
	// DatatypeFlagJson indicates the server believes the value payload to be JSON.
	DatatypeFlagJson = DatatypeFlag(0x01)

	// DatatypeFlagCompressed indicates the value payload is compressed.
	DatatypeFlagCompressed = DatatypeFlag(0x02)

	// DatatypeFlagXattrs indicates the inclusion of xattr data in the value payload.
	DatatypeFlagXattrs = DatatypeFlag(0x04)
)

// SubdocFlag specifies flags for a sub-document operation.
type SubdocFlag uint8

const (
	// SubdocFlagNone indicates no special treatment for this operation.
	SubdocFlagNone = SubdocFlag(0x00)

	// SubdocFlagMkDirP indicates that the path should be created if it does not already exist.
	SubdocFlagMkDirP = SubdocFlag(0x01)

	// 0x02 is unused, formally SubdocFlagMkDoc

	// SubdocFlagXattrPath indicates that the path refers to an Xattr rather than the document body.
	SubdocFlagXattrPath = SubdocFlag(0x04)

	// 0x08 is unused, formally SubdocFlagAccessDeleted

	// SubdocFlagExpandMacros indicates that the value portion of any sub-document mutations
	// should be expanded if they contain macros such as ${Mutation.CAS}.
	SubdocFlagExpandMacros = SubdocFlag(0x10)
)

// SubdocDocFlag specifies document-level flags for a sub-document operation.
type SubdocDocFlag uint8

const (
	// SubdocDocFlagNone indicates no special treatment for this operation.
	SubdocDocFlagNone = SubdocDocFlag(0x00)

	// SubdocDocFlagMkDoc indicates that the document should be created if it does not already exist.
	SubdocDocFlagMkDoc = SubdocDocFlag(0x01)

	// SubdocDocFlagReplaceDoc indices that this operation should be a replace rather than upsert.
	SubdocDocFlagReplaceDoc = SubdocDocFlag(0x02)

	// SubdocDocFlagAccessDeleted indicates that you wish to receive soft-deleted documents.
	// Internal: This should never be used and is not supported.
	SubdocDocFlagAccessDeleted = SubdocDocFlag(0x04)
)

// ServiceType specifies a particular Couchbase service type.
type ServiceType int

const (
	// MemdService represents a memcached service.
	MemdService = ServiceType(1)

	// MgmtService represents a management service (typically ns_server).
	MgmtService = ServiceType(2)

	// CapiService represents a CouchAPI service (typically for views).
	CapiService = ServiceType(3)

	// N1qlService represents a N1QL service (typically for query).
	N1qlService = ServiceType(4)

	// FtsService represents a full-text-search service.
	FtsService = ServiceType(5)

	// CbasService represents an analytics service.
	CbasService = ServiceType(6)
)

// DcpAgentPriority specifies the priority level for a dcp stream
type DcpAgentPriority uint8

const (
	// DcpAgentPriorityLow sets the priority for the dcp stream to low
	DcpAgentPriorityLow = DcpAgentPriority(0)

	// DcpAgentPriorityMed sets the priority for the dcp stream to medium
	DcpAgentPriorityMed = DcpAgentPriority(1)

	// DcpAgentPriorityHigh sets the priority for the dcp stream to high
	DcpAgentPriorityHigh = DcpAgentPriority(2)
)
