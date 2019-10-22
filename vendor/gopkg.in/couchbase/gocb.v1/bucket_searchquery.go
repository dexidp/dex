package gocb

import (
	"github.com/opentracing/opentracing-go"
)

// ExecuteSearchQuery performs a view query and returns a list of rows or an error.
func (b *Bucket) ExecuteSearchQuery(q *SearchQuery) (SearchResults, error) {
	span := b.tracer.StartSpan("ExecuteSearchQuery",
		opentracing.Tag{Key: "couchbase.service", Value: "fts"})
	span.SetTag("bucket_name", b.name)
	defer span.Finish()

	return b.cluster.doSearchQuery(span.Context(), b, q)
}
