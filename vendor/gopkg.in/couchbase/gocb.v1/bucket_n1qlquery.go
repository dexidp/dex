package gocb

import (
	"github.com/opentracing/opentracing-go"
)

// ExecuteN1qlQuery performs a n1ql query and returns a list of rows or an error.
func (b *Bucket) ExecuteN1qlQuery(q *N1qlQuery, params interface{}) (QueryResults, error) {
	span := b.tracer.StartSpan("ExecuteSearchQuery",
		opentracing.Tag{Key: "couchbase.service", Value: "n1ql"})
	defer span.Finish()

	return b.cluster.doN1qlQuery(span.Context(), b, q, params)
}
