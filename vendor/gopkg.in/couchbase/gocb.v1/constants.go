package gocb

import (
	"gopkg.in/couchbase/gocbcore.v7"
)

const (
	goCbVersionStr = "v1.6.3"
)

const (
	// Legacy flag format for JSON data.
	lfJson = 0

	// Common flags mask
	cfMask = 0xFF000000
	// Common flags mask for data format
	cfFmtMask = 0x0F000000
	// Common flags mask for compression mode.
	cfCmprMask = 0xE0000000

	// Common flag format for sdk-private data.
	cfFmtPrivate = 1 << 24
	// Common flag format for JSON data.
	cfFmtJson = 2 << 24
	// Common flag format for binary data.
	cfFmtBinary = 3 << 24
	// Common flag format for string data.
	cfFmtString = 4 << 24

	// Common flags compression for disabled compression.
	cfCmprNone = 0 << 29
)

// IndexType provides information on the type of indexer used for an index.
type IndexType string

const (
	// IndexTypeN1ql indicates that GSI was used to build the index.
	IndexTypeN1ql = IndexType("gsi")

	// IndexTypeView indicates that views were used to build the index.
	IndexTypeView = IndexType("views")
)

// QueryProfileType specifies the profiling mode to use during a query.
type QueryProfileType string

const (
	// QueryProfileNone disables query profiling
	QueryProfileNone = QueryProfileType("off")

	// QueryProfilePhases includes phase profiling information in the query response
	QueryProfilePhases = QueryProfileType("phases")

	// QueryProfileTimings includes timing profiling information in the query response
	QueryProfileTimings = QueryProfileType("timings")
)

// SubdocFlag provides special handling flags for sub-document operations
type SubdocFlag gocbcore.SubdocFlag

const (
	// SubdocFlagNone indicates no special behaviours
	SubdocFlagNone = SubdocFlag(gocbcore.SubdocFlagNone)

	// SubdocFlagCreatePath indicates you wish to recursively create the tree of paths
	// if it does not already exist within the document.
	SubdocFlagCreatePath = SubdocFlag(gocbcore.SubdocFlagMkDirP)

	// SubdocFlagXattr indicates your path refers to an extended attribute rather than the document.
	SubdocFlagXattr = SubdocFlag(gocbcore.SubdocFlagXattrPath)

	// SubdocFlagUseMacros indicates that you wish macro substitution to occur on the value
	SubdocFlagUseMacros = SubdocFlag(gocbcore.SubdocFlagExpandMacros)
)

// SubdocDocFlag specifies document-level flags for a sub-document operation.
type SubdocDocFlag gocbcore.SubdocDocFlag

const (
	// SubdocDocFlagNone indicates no special behaviours
	SubdocDocFlagNone = SubdocDocFlag(gocbcore.SubdocDocFlagNone)

	// SubdocDocFlagMkDoc indicates that the document should be created if it does not already exist.
	SubdocDocFlagMkDoc = SubdocDocFlag(gocbcore.SubdocDocFlagMkDoc)

	// SubdocDocFlagReplaceDoc indices that this operation should be a replace rather than upsert.
	SubdocDocFlagReplaceDoc = SubdocDocFlag(gocbcore.SubdocDocFlagReplaceDoc)

	// SubdocDocFlagAccessDeleted indicates that you wish to receive soft-deleted documents.
	SubdocDocFlagAccessDeleted = SubdocDocFlag(gocbcore.SubdocDocFlagAccessDeleted)
)

// ServiceType specifies a particular Couchbase service type.
type ServiceType gocbcore.ServiceType

const (
	// MemdService represents a memcached service.
	MemdService = ServiceType(gocbcore.MemdService)

	// MgmtService represents a management service (typically ns_server).
	MgmtService = ServiceType(gocbcore.MgmtService)

	// CapiService represents a CouchAPI service (typically for views).
	CapiService = ServiceType(gocbcore.CapiService)

	// N1qlService represents a N1QL service (typically for query).
	N1qlService = ServiceType(gocbcore.N1qlService)

	// FtsService represents a full-text-search service.
	FtsService = ServiceType(gocbcore.FtsService)

	// CbasService represents an analytics service.
	CbasService = ServiceType(gocbcore.CbasService)
)
