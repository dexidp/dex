package gocb

import (
	"github.com/opentracing/opentracing-go"
	"gopkg.in/couchbase/gocbcore.v7"
)

// BucketInternal holds various internally used bucket extension methods.
//
// Internal: This should never be used and is not supported.
type BucketInternal struct {
	b *Bucket
}

// GetRandom retrieves a document from the bucket
func (bi *BucketInternal) GetRandom(valuePtr interface{}) (string, Cas, error) {
	span := bi.b.startKvOpTrace("GetRandom")
	defer span.Finish()

	return bi.b.getRandom(span.Context(), valuePtr)
}

// UpsertMeta inserts or replaces (with metadata) a document in a bucket.
func (bi *BucketInternal) UpsertMeta(key string, value, extra []byte, datatype uint8,
	options, flags, expiry uint32, cas, revseqno uint64) (Cas, error) {
	span := bi.b.startKvOpTrace("UpsertMeta")
	defer span.Finish()

	outcas, _, err := bi.b.upsertMeta(span.Context(), key, value, extra, datatype, options,
		flags, expiry, cas, revseqno)
	return outcas, err
}

// RemoveMeta removes a document (with metadata) from the bucket.
func (bi *BucketInternal) RemoveMeta(key string, value, extra []byte, datatype uint8,
	options, flags, expiry uint32, cas, revseqno uint64) (Cas, error) {
	span := bi.b.startKvOpTrace("RemoveMeta")
	defer span.Finish()

	outcas, _, err := bi.b.removeMeta(span.Context(), key, value, extra, datatype, options,
		flags, expiry, cas, revseqno)
	return outcas, err
}

func (b *Bucket) getRandom(tracectx opentracing.SpanContext,
	valuePtr interface{}) (keyOut string, casOut Cas, errOut error) {
	ctrl := b.newOpManager(tracectx)
	err := ctrl.Wait(b.client.GetRandomEx(gocbcore.GetRandomOptions{
		TraceContext: tracectx,
	}, func(res *gocbcore.GetRandomResult, err error) {
		if res != nil {
			casOut = Cas(res.Cas)
			keyOut = string(res.Key)
			if err == nil {
				err = ctrl.Decode(res.Value, res.Flags, valuePtr)
			}
		}
		ctrl.Resolve(err)
	}))
	if err != nil {
		return "", 0, err
	}

	return
}

func (b *Bucket) upsertMeta(tracectx opentracing.SpanContext, key string, value, extra []byte, datatype uint8,
	options, flags uint32, expiry uint32, cas, revseqno uint64) (casOut Cas, mtOut MutationToken, errOut error) {
	ctrl := b.newOpManager(tracectx)
	err := ctrl.Wait(b.client.SetMetaEx(gocbcore.SetMetaOptions{
		Key:          []byte(key),
		Value:        value,
		Extra:        extra,
		Datatype:     datatype,
		Options:      options,
		Flags:        flags,
		Expiry:       expiry,
		Cas:          gocbcore.Cas(cas),
		RevNo:        revseqno,
		TraceContext: tracectx,
	}, func(res *gocbcore.SetMetaResult, err error) {
		if res != nil {
			casOut = Cas(res.Cas)
			mtOut = MutationToken{res.MutationToken, b}
		}
		ctrl.Resolve(err)
	}))
	if err != nil {
		return 0, MutationToken{}, err
	}
	return
}

func (b *Bucket) removeMeta(tracectx opentracing.SpanContext, key string, value, extra []byte, datatype uint8,
	options, flags uint32, expiry uint32, cas, revseqno uint64) (casOut Cas, mtOut MutationToken, errOut error) {
	ctrl := b.newOpManager(tracectx)
	err := ctrl.Wait(b.client.DeleteMetaEx(gocbcore.DeleteMetaOptions{
		Key:          []byte(key),
		Value:        value,
		Extra:        extra,
		Datatype:     datatype,
		Options:      options,
		Flags:        flags,
		Expiry:       expiry,
		Cas:          gocbcore.Cas(cas),
		RevNo:        revseqno,
		TraceContext: tracectx,
	}, func(res *gocbcore.DeleteMetaResult, err error) {
		if res != nil {
			casOut = Cas(res.Cas)
			mtOut = MutationToken{res.MutationToken, b}
		}
		ctrl.Resolve(err)
	}))
	if err != nil {
		return 0, MutationToken{}, err
	}
	return
}
