package gocb

import (
	"strconv"
	"time"
)

// ConsistencyMode indicates the level of data consistency desired for a query.
type ConsistencyMode int

const (
	// NotBounded indicates no data consistency is required.
	NotBounded = ConsistencyMode(1)
	// RequestPlus indicates that request-level data consistency is required.
	RequestPlus = ConsistencyMode(2)
	// StatementPlus inidcates that statement-level data consistency is required.
	StatementPlus = ConsistencyMode(3)
)

// N1qlQuery represents a pending N1QL query.
type N1qlQuery struct {
	options map[string]interface{}
	adHoc   bool
}

// Consistency specifies the level of consistency required for this query.
func (nq *N1qlQuery) Consistency(stale ConsistencyMode) *N1qlQuery {
	if _, ok := nq.options["scan_vectors"]; ok {
		panic("Consistent and ConsistentWith must be used exclusively")
	}
	if stale == NotBounded {
		nq.options["scan_consistency"] = "not_bounded"
	} else if stale == RequestPlus {
		nq.options["scan_consistency"] = "request_plus"
	} else if stale == StatementPlus {
		nq.options["scan_consistency"] = "statement_plus"
	} else {
		panic("Unexpected consistency option")
	}
	return nq
}

// ConsistentWith specifies a mutation state to be consistent with for this query.
func (nq *N1qlQuery) ConsistentWith(state *MutationState) *N1qlQuery {
	if _, ok := nq.options["scan_consistency"]; ok {
		panic("Consistent and ConsistentWith must be used exclusively")
	}
	nq.options["scan_consistency"] = "at_plus"
	nq.options["scan_vectors"] = state
	return nq
}

// AdHoc specifies that this query is adhoc and should not be prepared.
func (nq *N1qlQuery) AdHoc(adhoc bool) *N1qlQuery {
	nq.adHoc = adhoc
	return nq
}

// Profile specifies a profiling mode to use for this N1QL query.
func (nq *N1qlQuery) Profile(profileMode QueryProfileType) *N1qlQuery {
	nq.options["profile"] = profileMode
	return nq
}

// ScanCap specifies the maximum buffered channel size between the indexer
// client and the query service for index scans. This parameter controls
// when to use scan backfill. Use 0 or a negative number to disable.
func (nq *N1qlQuery) ScanCap(scanCap int) *N1qlQuery {
	nq.options["scan_cap"] = strconv.Itoa(scanCap)
	return nq
}

// PipelineBatch controls the number of items execution operators can
// batch for fetch from the KV node.
func (nq *N1qlQuery) PipelineBatch(pipelineBatch int) *N1qlQuery {
	nq.options["pipeline_batch"] = strconv.Itoa(pipelineBatch)
	return nq
}

// PipelineCap controls the maximum number of items each execution operator
// can buffer between various operators.
func (nq *N1qlQuery) PipelineCap(pipelineCap int) *N1qlQuery {
	nq.options["pipeline_cap"] = strconv.Itoa(pipelineCap)
	return nq
}

// ReadOnly controls whether a query can change a resulting recordset.  If
// readonly is true, then only SELECT statements are permitted.
func (nq *N1qlQuery) ReadOnly(readOnly bool) *N1qlQuery {
	nq.options["readonly"] = readOnly
	return nq
}

// Custom allows specifying custom query options.
func (nq *N1qlQuery) Custom(name string, value interface{}) *N1qlQuery {
	nq.options[name] = value
	return nq
}

// Timeout indicates the maximum time to wait for this query to complete.
func (nq *N1qlQuery) Timeout(timeout time.Duration) *N1qlQuery {
	nq.options["timeout"] = timeout.String()
	return nq
}

// NewN1qlQuery creates a new N1qlQuery object from a query string.
func NewN1qlQuery(statement string) *N1qlQuery {
	nq := &N1qlQuery{
		options: make(map[string]interface{}),
		adHoc:   true,
	}
	nq.options["statement"] = statement
	return nq
}
