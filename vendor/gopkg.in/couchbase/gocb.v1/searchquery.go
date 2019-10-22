package gocb

import (
	"time"
)

// SearchHighlightStyle indicates the type of highlighting to use for a search query.
type SearchHighlightStyle string

const (
	// DefaultHighlightStyle specifies to use the default to highlight search result hits.
	DefaultHighlightStyle = SearchHighlightStyle("")

	// HtmlHighlightStyle specifies to use HTML tags to highlight search result hits.
	HtmlHighlightStyle = SearchHighlightStyle("html")

	// AnsiHightlightStyle specifies to use ANSI tags to highlight search result hits.
	AnsiHightlightStyle = SearchHighlightStyle("ansi")
)

type searchQueryHighlightData struct {
	Style  string   `json:"style,omitempty"`
	Fields []string `json:"fields,omitempty"`
}
type searchQueryConsistencyData struct {
	Level   string         `json:"level,omitempty"`
	Vectors *MutationState `json:"vectors,omitempty"`
}
type searchQueryCtlData struct {
	Timeout     uint                        `json:"timeout,omitempty"`
	Consistency *searchQueryConsistencyData `json:"consistency,omitempty"`
}
type searchQueryData struct {
	Query     interface{}               `json:"query,omitempty"`
	Size      int                       `json:"size,omitempty"`
	From      int                       `json:"from,omitempty"`
	Explain   bool                      `json:"explain,omitempty"`
	Highlight *searchQueryHighlightData `json:"highlight,omitempty"`
	Fields    []string                  `json:"fields,omitempty"`
	Sort      []interface{}             `json:"sort,omitempty"`
	Facets    map[string]interface{}    `json:"facets,omitempty"`
	Ctl       *searchQueryCtlData       `json:"ctl,omitempty"`
}

// SearchQuery represents a pending search query.
type SearchQuery struct {
	name string
	data searchQueryData
}

// Limit specifies a limit on the number of results to return.
func (sq *SearchQuery) Limit(value int) *SearchQuery {
	sq.data.Size = value
	return sq
}

// Skip specifies how many results to skip at the beginning of the result list.
func (sq *SearchQuery) Skip(value int) *SearchQuery {
	sq.data.From = value
	return sq
}

// Explain enables search query explanation which provides details on how a query is executed.
func (sq *SearchQuery) Explain(value bool) *SearchQuery {
	sq.data.Explain = value
	return sq
}

// Highlight specifies how to highlight the hits in the search result.
func (sq *SearchQuery) Highlight(style SearchHighlightStyle, fields ...string) *SearchQuery {
	if sq.data.Highlight == nil {
		sq.data.Highlight = &searchQueryHighlightData{}
	}
	sq.data.Highlight.Style = string(style)
	sq.data.Highlight.Fields = fields
	return sq
}

// Fields specifies which fields you wish to return in the results.
func (sq *SearchQuery) Fields(fields ...string) *SearchQuery {
	sq.data.Fields = fields
	return sq
}

// Sort specifies a sorting order for the results.  Only available in Couchbase Server 4.6+.
func (sq *SearchQuery) Sort(fields ...interface{}) *SearchQuery {
	sq.data.Sort = fields
	return sq
}

// AddFacet adds a new search facet to include in the results.
func (sq *SearchQuery) AddFacet(name string, facet interface{}) *SearchQuery {
	if sq.data.Facets == nil {
		sq.data.Facets = make(map[string]interface{})
	}
	sq.data.Facets[name] = facet
	return sq
}

// Timeout indicates the maximum time to wait for this query to complete.
func (sq *SearchQuery) Timeout(value time.Duration) *SearchQuery {
	if sq.data.Ctl == nil {
		sq.data.Ctl = &searchQueryCtlData{}
	}
	sq.data.Ctl.Timeout = uint(value / time.Millisecond)
	return sq
}

// Consistency specifies the level of consistency required for this query.
func (sq *SearchQuery) Consistency(stale ConsistencyMode) *SearchQuery {
	if sq.data.Ctl == nil {
		sq.data.Ctl = &searchQueryCtlData{}
	}
	if sq.data.Ctl.Consistency == nil {
		sq.data.Ctl.Consistency = &searchQueryConsistencyData{}
	}

	if sq.data.Ctl.Consistency.Vectors != nil {
		panic("Consistent and ConsistentWith must be used exclusively")
	}
	if stale == NotBounded {
		sq.data.Ctl.Consistency.Level = "not_bounded"
	} else {
		panic("Unexpected consistency option")
	}
	return sq
}

// ConsistentWith specifies a mutation state to be consistent with for this query.
func (sq *SearchQuery) ConsistentWith(state *MutationState) *SearchQuery {
	if sq.data.Ctl == nil {
		sq.data.Ctl = &searchQueryCtlData{}
	}
	if sq.data.Ctl.Consistency == nil {
		sq.data.Ctl.Consistency = &searchQueryConsistencyData{}
	}

	if sq.data.Ctl.Consistency.Level != "" {
		panic("Consistent and ConsistentWith must be used exclusively")
	}
	sq.data.Ctl.Consistency.Level = "at_plus"
	sq.data.Ctl.Consistency.Vectors = state
	return sq
}

func (sq *SearchQuery) indexName() string {
	return sq.name
}

func (sq *SearchQuery) queryData() interface{} {
	return sq.data
}

// NewSearchQuery creates a new SearchQuery object from an index name and query.
func NewSearchQuery(indexName string, query interface{}) *SearchQuery {
	q := &SearchQuery{
		name: indexName,
	}
	q.data.Query = query
	return q
}
