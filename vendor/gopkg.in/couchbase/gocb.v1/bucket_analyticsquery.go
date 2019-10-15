package gocb

import (
	"github.com/opentracing/opentracing-go"
)

// ExecuteAnalyticsQuery performs an analytics query and returns a list of rows or an error.
func (b *Bucket) ExecuteAnalyticsQuery(q *AnalyticsQuery, params interface{}) (AnalyticsResults, error) {
	span := b.tracer.StartSpan("ExecuteAnalyticsQuery",
		opentracing.Tag{Key: "couchbase.service", Value: "fts"})
	span.SetTag("bucket_name", b.name)
	defer span.Finish()

	return b.cluster.doAnalyticsQuery(span.Context(), b, q, params)
}
