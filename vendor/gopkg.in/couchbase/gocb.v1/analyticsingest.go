package gocb

// UNCOMMITTED: This API may change.

import (
	"encoding/json"
	"errors"
	"time"

	"github.com/google/uuid"
)

type ingestMethod func(bucket *Bucket, key string, val interface{}) error

// IngestMethodInsert indicates that the Insert function should be used for kv ingest.
func IngestMethodInsert(bucket *Bucket, key string, val interface{}) error {
	_, err := bucket.Insert(key, val, 0)
	return err
}

// IngestMethodUpsert indicates that the Upsert function should be used for kv ingest.
func IngestMethodUpsert(bucket *Bucket, key string, val interface{}) error {
	_, err := bucket.Upsert(key, val, 0)
	return err
}

// IngestMethodReplace indicates that the Replace function should be used for kv ingest.
func IngestMethodReplace(bucket *Bucket, key string, val interface{}) error {
	_, err := bucket.Replace(key, val, 0, 0)
	return err
}

// IdGeneratorFunction is called to create an ID for a document.
type IdGeneratorFunction func(doc interface{}) (string, error)

// DataConverterFunction is called to convert from analytics document format
// to kv document
type DataConverterFunction func(docBytes []byte) (interface{}, error)

// UUIDIdGeneratorFunction is a IdGeneratorFunction that creates a UUID ID for each document.
func UUIDIdGeneratorFunction(doc interface{}) (string, error) {
	return uuid.New().String(), nil
}

// PassthroughDataConverterFunction is a DataConverterFunction that returns the data that is
// passed to it. The interface out of this represents a map[string]interface{}.
func PassthroughDataConverterFunction(docBytes []byte) (interface{}, error) {
	var doc interface{}
	err := json.Unmarshal(docBytes, &doc)
	if err != nil {
		return nil, err
	}
	return doc, nil
}

// AnalyticsIngestOptions contains the options for an Analytics query to KV ingest.
type AnalyticsIngestOptions struct {
	analyticsTimeout  time.Duration
	ingestMethod      ingestMethod
	ignoreIngestError bool
	idGenerator       IdGeneratorFunction
	dataConverter     DataConverterFunction
	kvRetryBehavior   QueryRetryBehavior
	retryOn           []error
}

// DefaultAnalyticsIngestOptions creates a new AnalyticsIngestOptions from a set of defaults.
//
// UNCOMMITTED: This API may change.
func DefaultAnalyticsIngestOptions() *AnalyticsIngestOptions {
	return &AnalyticsIngestOptions{
		ingestMethod:      IngestMethodUpsert,
		idGenerator:       UUIDIdGeneratorFunction,
		dataConverter:     PassthroughDataConverterFunction,
		ignoreIngestError: true,
		kvRetryBehavior:   NewQueryDelayRetryBehavior(10, 2, 500*time.Millisecond, QueryExponentialDelayFunction),
		retryOn:           []error{ErrTmpFail, ErrBusy},
	}
}

// AnalyticsTimeout sets the timeout value that will be used for execution of the AnalyticsQuery.
func (ai *AnalyticsIngestOptions) AnalyticsTimeout(timeout time.Duration) *AnalyticsIngestOptions {
	ai.analyticsTimeout = timeout
	return ai
}

// IngestMethod sets ingestMethod that will be used for KV operations
func (ai *AnalyticsIngestOptions) IngestMethod(method ingestMethod) *AnalyticsIngestOptions {
	ai.ingestMethod = method
	return ai
}

// IgnoreIngestError sets whether errors will be ignored when performing KV operations
func (ai *AnalyticsIngestOptions) IgnoreIngestError(ignore bool) *AnalyticsIngestOptions {
	ai.ignoreIngestError = ignore
	return ai
}

// IdGenerator sets the IdGeneratorFunction to use for generation of IDs
func (ai *AnalyticsIngestOptions) IdGenerator(fn IdGeneratorFunction) *AnalyticsIngestOptions {
	ai.idGenerator = fn
	return ai
}

// DataConverter sets the DataConverterFunction to use for conversion of Analytics documents to
// KV documents.
func (ai *AnalyticsIngestOptions) DataConverter(fn DataConverterFunction) *AnalyticsIngestOptions {
	ai.dataConverter = fn
	return ai
}

// KVRetryBehavior sets the QueryRetryBehavior to use for retrying KV operations when a temporary
// or overload error occurs.
func (ai *AnalyticsIngestOptions) KVRetryBehavior(behavior QueryRetryBehavior) *AnalyticsIngestOptions {
	ai.kvRetryBehavior = behavior
	return ai
}

// KVRetryOn sets the errors to perform retries on for kv operation errors.
func (ai *AnalyticsIngestOptions) KVRetryOn(errors []error) *AnalyticsIngestOptions {
	ai.retryOn = errors
	return ai
}

// AnalyticsIngest executes an Analytics query and inserts/updates/replaces the transformed results into a bucket.
//
// UNCOMMITTED: This API may change.
func (b *Bucket) AnalyticsIngest(analyticsQuery *AnalyticsQuery, analyticsParams []interface{}, opts *AnalyticsIngestOptions) error {
	return b.analyticsIngest(new(defaultIngestQueryRunner), analyticsQuery, analyticsParams, opts)
}

func (b *Bucket) analyticsIngest(queryRunner ingestQueryRunner, analyticsQuery *AnalyticsQuery, analyticsParams []interface{}, opts *AnalyticsIngestOptions) error {
	if analyticsQuery == nil {
		return errors.New("query cannot be nil")
	}
	if opts == nil {
		opts = DefaultAnalyticsIngestOptions()
	}
	if opts.idGenerator == nil {
		opts.idGenerator = UUIDIdGeneratorFunction
	}
	if opts.dataConverter == nil {
		opts.dataConverter = PassthroughDataConverterFunction
	}
	if opts.ingestMethod == nil {
		opts.ingestMethod = IngestMethodUpsert
	}

	analyticsTimeout := opts.analyticsTimeout
	if analyticsTimeout == 0 {
		analyticsTimeout = b.AnalyticsTimeout()
	}
	analyticsQuery.ServerSideTimeout(analyticsTimeout)

	qResults, err := queryRunner.ExecuteQuery(b, analyticsQuery, analyticsParams)
	if err != nil {
		return err
	}

	for {
		qBytes := qResults.NextBytes()
		if qBytes == nil {
			break
		}

		converted, err := opts.dataConverter(qBytes)
		if err != nil {
			if opts.ignoreIngestError {
				continue
			} else {
				return err
			}
		}
		id, err := opts.idGenerator(converted)
		if err != nil {
			if opts.ignoreIngestError {
				continue
			} else {
				return err
			}
		}

		var retries uint
		for {
			err = b.ingest(id, converted, opts.ingestMethod)
			if err == nil {
				break
			}

			if isRetryableError(err, opts.retryOn) {
				if opts.kvRetryBehavior == nil || !opts.kvRetryBehavior.CanRetry(retries) {
					break
				}
			} else {
				break
			}

			retries++
			time.Sleep(opts.kvRetryBehavior.NextInterval(retries))
		}
		if err != nil {
			if opts.ignoreIngestError {
				continue
			} else {
				return err
			}
		}
	}

	return nil
}

type ingestQueryRunner interface {
	ExecuteQuery(bucket *Bucket, query *AnalyticsQuery, params []interface{}) (AnalyticsResults, error)
}

type defaultIngestQueryRunner struct {
}

func (runner *defaultIngestQueryRunner) ExecuteQuery(bucket *Bucket, query *AnalyticsQuery, params []interface{}) (AnalyticsResults, error) {
	return bucket.ExecuteAnalyticsQuery(query, params)
}

func (b *Bucket) ingest(key string, converted interface{}, method ingestMethod) error {
	err := method(b, key, converted)
	if err != nil {
		return err
	}

	return nil
}

func isRetryableError(err error, errors []error) bool {
	for _, retryable := range errors {
		if err == retryable {
			return true
		}
	}
	return false
}
