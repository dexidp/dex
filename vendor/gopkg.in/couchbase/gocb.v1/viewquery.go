package gocb

import (
	"bytes"
	"encoding/json"
	"net/url"
	"strconv"
	"strings"
)

// StaleMode specifies the consistency required for a view query.
type StaleMode int

const (
	// Before indicates to update the index before querying it.
	Before = StaleMode(1)
	// None indicates that no special behaviour should be used.
	None = StaleMode(2)
	// After indicates to update the index asynchronously after querying.
	After = StaleMode(3)
)

// SortOrder specifies the ordering for the view queries results.
type SortOrder int

const (
	// Ascending indicates the query results should be sorted from lowest to highest.
	Ascending = SortOrder(1)
	// Descending indicates the query results should be sorted from highest to lowest.
	Descending = SortOrder(2)
)

// ViewQuery represents a pending view query.
type ViewQuery struct {
	ddoc    string
	name    string
	options url.Values
	errs    MultiError
}

func (vq *ViewQuery) marshalJson(value interface{}) []byte {
	buf := new(bytes.Buffer)
	enc := json.NewEncoder(buf)
	enc.SetEscapeHTML(false)
	err := enc.Encode(value)
	if err != nil {
		vq.errs.add(err)
		return nil
	}
	return buf.Bytes()
}

// Stale specifies the level of consistency required for this query.
func (vq *ViewQuery) Stale(stale StaleMode) *ViewQuery {
	if stale == Before {
		vq.options.Set("stale", "false")
	} else if stale == None {
		vq.options.Set("stale", "ok")
	} else if stale == After {
		vq.options.Set("stale", "update_after")
	} else {
		panic("Unexpected stale option")
	}
	return vq
}

// Skip specifies how many results to skip at the beginning of the result list.
func (vq *ViewQuery) Skip(num uint) *ViewQuery {
	vq.options.Set("skip", strconv.FormatUint(uint64(num), 10))
	return vq
}

// Limit specifies a limit on the number of results to return.
func (vq *ViewQuery) Limit(num uint) *ViewQuery {
	vq.options.Set("limit", strconv.FormatUint(uint64(num), 10))
	return vq
}

// Order specifies the order to sort the view results in.
func (vq *ViewQuery) Order(order SortOrder) *ViewQuery {
	if order == Ascending {
		vq.options.Set("descending", "false")
	} else if order == Descending {
		vq.options.Set("descending", "true")
	} else {
		panic("Unexpected order option")
	}
	return vq
}

// Reduce specifies whether to run the reduce part of the map-reduce.
func (vq *ViewQuery) Reduce(reduce bool) *ViewQuery {
	if reduce == true {
		vq.options.Set("reduce", "true")
	} else {
		vq.options.Set("reduce", "false")
	}
	return vq
}

// Group specifies whether to group the map-reduce results.
func (vq *ViewQuery) Group(useGrouping bool) *ViewQuery {
	if useGrouping {
		vq.options.Set("group", "true")
	} else {
		vq.options.Set("group", "false")
	}
	return vq
}

// GroupLevel specifies at what level to group the map-reduce results.
func (vq *ViewQuery) GroupLevel(groupLevel uint) *ViewQuery {
	vq.options.Set("group_level", strconv.FormatUint(uint64(groupLevel), 10))
	return vq
}

// Key specifies a specific key to retrieve from the index.
func (vq *ViewQuery) Key(key interface{}) *ViewQuery {
	jsonKey := vq.marshalJson(key)
	vq.options.Set("key", string(jsonKey))
	return vq
}

// Keys specifies a list of specific keys to retrieve from the index.
func (vq *ViewQuery) Keys(keys []interface{}) *ViewQuery {
	jsonKeys := vq.marshalJson(keys)
	vq.options.Set("keys", string(jsonKeys))
	return vq
}

// Range specifies a value range to get results within.
func (vq *ViewQuery) Range(start, end interface{}, inclusiveEnd bool) *ViewQuery {
	// TODO(brett19): Not currently handling errors due to no way to return the error
	if start != nil {
		jsonStartKey := vq.marshalJson(start)
		vq.options.Set("startkey", string(jsonStartKey))
	} else {
		vq.options.Del("startkey")
	}
	if end != nil {
		jsonEndKey := vq.marshalJson(end)
		vq.options.Set("endkey", string(jsonEndKey))
	} else {
		vq.options.Del("endkey")
	}
	if start != nil || end != nil {
		if inclusiveEnd {
			vq.options.Set("inclusive_end", "true")
		} else {
			vq.options.Set("inclusive_end", "false")
		}
	} else {
		vq.options.Del("inclusive_end")
	}
	return vq
}

// IdRange specifies a range of document id's to get results within.
// Usually requires Range to be specified as well.
func (vq *ViewQuery) IdRange(start, end string) *ViewQuery {
	if start != "" {
		vq.options.Set("startkey_docid", start)
	} else {
		vq.options.Del("startkey_docid")
	}
	if end != "" {
		vq.options.Set("endkey_docid", end)
	} else {
		vq.options.Del("endkey_docid")
	}
	return vq
}

// Development specifies whether to query the production or development design document.
func (vq *ViewQuery) Development(val bool) *ViewQuery {
	if val {
		if !strings.HasPrefix(vq.ddoc, "dev_") {
			vq.ddoc = "dev_" + vq.ddoc
		}
	} else {
		vq.ddoc = strings.TrimPrefix(vq.ddoc, "dev_")
	}
	return vq
}

// Custom allows specifying custom query options.
func (vq *ViewQuery) Custom(name, value string) *ViewQuery {
	vq.options.Set(name, value)
	return vq
}

func (vq *ViewQuery) getInfo() (string, string, url.Values, error) {
	return vq.ddoc, vq.name, vq.options, vq.errs.get()
}

// NewViewQuery creates a new ViewQuery object from a design document and view name.
func NewViewQuery(ddoc, name string) *ViewQuery {
	return &ViewQuery{
		ddoc:    ddoc,
		name:    name,
		options: url.Values{},
	}
}
