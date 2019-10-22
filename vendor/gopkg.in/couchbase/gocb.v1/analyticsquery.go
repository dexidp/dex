package gocb

import "time"

// AnalyticsQuery represents a pending Analytics query.
type AnalyticsQuery struct {
	options map[string]interface{}
}

// NewAnalyticsQuery creates a new AnalyticsQuery object from a query string.
func NewAnalyticsQuery(statement string) *AnalyticsQuery {
	nq := &AnalyticsQuery{
		options: make(map[string]interface{}),
	}
	nq.options["statement"] = statement
	return nq
}

// ServerSideTimeout indicates the maximum time to wait for this query to complete.
func (aq *AnalyticsQuery) ServerSideTimeout(timeout time.Duration) *AnalyticsQuery {
	aq.options["timeout"] = timeout.String()
	return aq
}

// Pretty indicates whether the response should be nicely formatted.
func (aq *AnalyticsQuery) Pretty(pretty bool) *AnalyticsQuery {
	aq.options["pretty"] = pretty
	return aq
}

// ContextId sets the client context id for the request, for use with tracing.
func (aq *AnalyticsQuery) ContextId(clientContextId string) *AnalyticsQuery {
	aq.options["client_context_id"] = clientContextId
	return aq
}

// RawParam allows specifying custom query options.
func (aq *AnalyticsQuery) RawParam(name string, value interface{}) *AnalyticsQuery {
	aq.options[name] = value
	return aq
}

// Priority sets whether or not the query should be run with priority status.
func (aq *AnalyticsQuery) Priority(priority bool) *AnalyticsQuery {
	if priority {
		aq.options["priority"] = -1
	} else {
		delete(aq.options, "priority")
	}
	return aq
}

// Deferred sets whether or not the query should be run as a deferred query.
//
// Experimental: This API is subject to change at any time.
func (aq *AnalyticsQuery) Deferred(deferred bool) *AnalyticsQuery {
	if deferred {
		aq.options["mode"] = "async"
	} else {
		delete(aq.options, "mode")
	}
	return aq
}
