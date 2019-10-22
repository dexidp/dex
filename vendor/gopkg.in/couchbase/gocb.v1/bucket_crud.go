package gocb

import (
	"github.com/opentracing/opentracing-go"
	"gopkg.in/couchbase/gocbcore.v7"
)

// Get retrieves a document from the bucket
func (b *Bucket) Get(key string, valuePtr interface{}) (Cas, error) {
	span := b.startKvOpTrace("Get")
	defer span.Finish()

	return b.get(span.Context(), key, valuePtr)
}

// GetAndTouch retrieves a document and simultaneously updates its expiry time.
func (b *Bucket) GetAndTouch(key string, expiry uint32, valuePtr interface{}) (Cas, error) {
	span := b.startKvOpTrace("GetAndTouch")
	defer span.Finish()

	return b.getAndTouch(span.Context(), key, expiry, valuePtr)
}

// GetAndLock locks a document for a period of time, providing exclusive RW access to it.
func (b *Bucket) GetAndLock(key string, lockTime uint32, valuePtr interface{}) (Cas, error) {
	span := b.startKvOpTrace("GetAndLock")
	defer span.Finish()

	return b.getAndLock(span.Context(), key, lockTime, valuePtr)
}

// Unlock unlocks a document which was locked with GetAndLock.
func (b *Bucket) Unlock(key string, cas Cas) (Cas, error) {
	span := b.startKvOpTrace("Unlock")
	defer span.Finish()

	cas, _, err := b.unlock(span.Context(), key, cas)
	return cas, err
}

// GetReplica returns the value of a particular document from a replica server.
func (b *Bucket) GetReplica(key string, valuePtr interface{}, replicaIdx int) (Cas, error) {
	span := b.startKvOpTrace("GetReplica")
	defer span.Finish()

	cas, err := b.getReplica(span.Context(), key, valuePtr, replicaIdx)
	return cas, err
}

// Touch touches a document, specifying a new expiry time for it.
// The Cas value must be 0.
func (b *Bucket) Touch(key string, cas Cas, expiry uint32) (Cas, error) {
	span := b.startKvOpTrace("Touch")
	defer span.Finish()

	if cas != 0 {
		return 0, ErrNonZeroCas
	}

	cas, _, err := b.touch(span.Context(), key, expiry)
	return cas, err
}

// Remove removes a document from the bucket.
func (b *Bucket) Remove(key string, cas Cas) (Cas, error) {
	span := b.startKvOpTrace("Remove")
	defer span.Finish()

	cas, _, err := b.remove(span.Context(), key, cas)
	return cas, err
}

// Upsert inserts or replaces a document in the bucket.
func (b *Bucket) Upsert(key string, value interface{}, expiry uint32) (Cas, error) {
	span := b.startKvOpTrace("Upsert")
	defer span.Finish()

	cas, _, err := b.upsert(span.Context(), key, value, expiry)
	return cas, err
}

// Insert inserts a new document to the bucket.
func (b *Bucket) Insert(key string, value interface{}, expiry uint32) (Cas, error) {
	span := b.startKvOpTrace("Insert")
	defer span.Finish()

	cas, _, err := b.insert(span.Context(), key, value, expiry)
	return cas, err
}

// Replace replaces a document in the bucket.
func (b *Bucket) Replace(key string, value interface{}, cas Cas, expiry uint32) (Cas, error) {
	span := b.startKvOpTrace("Replace")
	defer span.Finish()

	cas, _, err := b.replace(span.Context(), key, value, cas, expiry)
	return cas, err
}

// Append appends a string value to a document.
func (b *Bucket) Append(key, value string) (Cas, error) {
	span := b.startKvOpTrace("Append")
	defer span.Finish()

	cas, _, err := b.append(span.Context(), key, value)
	return cas, err
}

// Prepend prepends a string value to a document.
func (b *Bucket) Prepend(key, value string) (Cas, error) {
	span := b.startKvOpTrace("Prepend")
	defer span.Finish()

	cas, _, err := b.prepend(span.Context(), key, value)
	return cas, err
}

// Counter performs an atomic addition or subtraction for an integer document.  Passing a
// non-negative `initial` value will cause the document to be created if it did  not
// already exist.
func (b *Bucket) Counter(key string, delta, initial int64, expiry uint32) (uint64, Cas, error) {
	span := b.startKvOpTrace("Counter")
	defer span.Finish()

	val, cas, _, err := b.counter(span.Context(), key, delta, initial, expiry)
	return val, cas, err
}

// ServerStats is a tree of statistics information returned from the server.
//   stats := cb.Stats(...)
//   for server := stats {
//       for statName, stat := server {
//       //...
//       }
//   }
type ServerStats map[string]map[string]string

// Stats returns various server statistics from the cluster.
func (b *Bucket) Stats(key string) (ServerStats, error) {
	span := b.startKvOpTrace("Stats")
	defer span.Finish()

	stats, err := b.stats(span.Context(), key)
	return stats, err
}

type opManager struct {
	b        *Bucket
	signal   chan error
	tracectx opentracing.SpanContext
}

func (ctrl *opManager) Resolve(err error) {
	ctrl.signal <- err
}

func (ctrl *opManager) Wait(op gocbcore.PendingOp, err error) error {
	if err != nil {
		return err
	}

	timeoutTmr := gocbcore.AcquireTimer(ctrl.b.opTimeout)
	select {
	case err = <-ctrl.signal:
		gocbcore.ReleaseTimer(timeoutTmr, false)
		return err
	case <-timeoutTmr.C:
		gocbcore.ReleaseTimer(timeoutTmr, true)
		if !op.Cancel() {
			err = <-ctrl.signal
			return err
		}

		return ErrTimeout
	}
}

func (ctrl *opManager) Decode(bytes []byte, flags uint32, valuePtr interface{}) error {
	dspan := ctrl.b.tracer.StartSpan("decode",
		opentracing.ChildOf(ctrl.tracectx))

	err := ctrl.b.transcoder.Decode(bytes, flags, valuePtr)

	dspan.Finish()

	return err
}

func (ctrl *opManager) Encode(value interface{}) ([]byte, uint32, error) {
	espan := ctrl.b.tracer.StartSpan("encode",
		opentracing.ChildOf(ctrl.tracectx))

	bytes, flags, err := ctrl.b.transcoder.Encode(value)

	espan.Finish()

	return bytes, flags, err
}

func (b *Bucket) newOpManager(tracectx opentracing.SpanContext) *opManager {
	return &opManager{
		b:        b,
		signal:   make(chan error, 1),
		tracectx: tracectx,
	}
}

func (b *Bucket) get(tracectx opentracing.SpanContext, key string, valuePtr interface{}) (casOut Cas, errOut error) {
	ctrl := b.newOpManager(tracectx)
	err := ctrl.Wait(b.client.GetEx(gocbcore.GetOptions{
		Key:          []byte(key),
		TraceContext: tracectx,
	}, func(res *gocbcore.GetResult, err error) {
		if res != nil {
			casOut = Cas(res.Cas)
			if err == nil {
				err = ctrl.Decode(res.Value, res.Flags, valuePtr)
			}
		}
		ctrl.Resolve(err)
	}))
	if err != nil {
		return 0, err
	}
	return
}

func (b *Bucket) getAndTouch(tracectx opentracing.SpanContext, key string, expiry uint32, valuePtr interface{}) (casOut Cas, errOut error) {
	ctrl := b.newOpManager(tracectx)
	err := ctrl.Wait(b.client.GetAndTouchEx(gocbcore.GetAndTouchOptions{
		Key:          []byte(key),
		Expiry:       expiry,
		TraceContext: tracectx,
	}, func(res *gocbcore.GetAndTouchResult, err error) {
		if res != nil {
			casOut = Cas(res.Cas)
			if err == nil {
				err = ctrl.Decode(res.Value, res.Flags, valuePtr)
			}
		}
		ctrl.Resolve(err)
	}))
	if err != nil {
		return 0, err
	}
	return
}

func (b *Bucket) getAndLock(tracectx opentracing.SpanContext, key string, lockTime uint32, valuePtr interface{}) (casOut Cas, errOut error) {
	ctrl := b.newOpManager(tracectx)
	err := ctrl.Wait(b.client.GetAndLockEx(gocbcore.GetAndLockOptions{
		Key:          []byte(key),
		LockTime:     lockTime,
		TraceContext: tracectx,
	}, func(res *gocbcore.GetAndLockResult, err error) {
		if res != nil {
			casOut = Cas(res.Cas)
			if err == nil {
				err = ctrl.Decode(res.Value, res.Flags, valuePtr)
			}
		}
		ctrl.Resolve(err)
	}))
	if err != nil {
		return 0, err
	}
	return
}

func (b *Bucket) unlock(tracectx opentracing.SpanContext, key string, cas Cas) (casOut Cas, mtOut MutationToken, errOut error) {
	ctrl := b.newOpManager(tracectx)
	err := ctrl.Wait(b.client.UnlockEx(gocbcore.UnlockOptions{
		Key:          []byte(key),
		Cas:          gocbcore.Cas(cas),
		TraceContext: tracectx,
	}, func(res *gocbcore.UnlockResult, err error) {
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

func (b *Bucket) getReplica(tracectx opentracing.SpanContext, key string, valuePtr interface{}, replicaIdx int) (casOut Cas, errOut error) {
	ctrl := b.newOpManager(tracectx)
	err := ctrl.Wait(b.client.GetReplicaEx(gocbcore.GetReplicaOptions{
		Key:          []byte(key),
		ReplicaIdx:   replicaIdx,
		TraceContext: tracectx,
	}, func(res *gocbcore.GetReplicaResult, err error) {
		if res != nil {
			casOut = Cas(res.Cas)
			if err == nil {
				err = ctrl.Decode(res.Value, res.Flags, valuePtr)
			}
		}
		ctrl.Resolve(err)
	}))
	if err != nil {
		return 0, err
	}
	return
}

func (b *Bucket) touch(tracectx opentracing.SpanContext, key string, expiry uint32) (casOut Cas, mtOut MutationToken, errOut error) {
	ctrl := b.newOpManager(tracectx)
	err := ctrl.Wait(b.client.TouchEx(gocbcore.TouchOptions{
		Key:          []byte(key),
		Expiry:       expiry,
		TraceContext: tracectx,
	}, func(res *gocbcore.TouchResult, err error) {
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

func (b *Bucket) remove(tracectx opentracing.SpanContext, key string, cas Cas) (casOut Cas, mtOut MutationToken, errOut error) {
	ctrl := b.newOpManager(tracectx)
	err := ctrl.Wait(b.client.DeleteEx(gocbcore.DeleteOptions{
		Key:          []byte(key),
		Cas:          gocbcore.Cas(cas),
		TraceContext: tracectx,
	}, func(res *gocbcore.DeleteResult, err error) {
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

func (b *Bucket) upsert(tracectx opentracing.SpanContext, key string, value interface{}, expiry uint32) (casOut Cas, mtOut MutationToken, errOut error) {
	ctrl := b.newOpManager(tracectx)

	bytes, flags, err := ctrl.Encode(value)
	if err != nil {
		return 0, MutationToken{}, err
	}

	err = ctrl.Wait(b.client.SetEx(gocbcore.SetOptions{
		Key:          []byte(key),
		Value:        bytes,
		Flags:        flags,
		Expiry:       expiry,
		TraceContext: tracectx,
	}, func(res *gocbcore.StoreResult, err error) {
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

func (b *Bucket) insert(tracectx opentracing.SpanContext, key string, value interface{}, expiry uint32) (casOut Cas, mtOut MutationToken, errOut error) {
	ctrl := b.newOpManager(tracectx)

	bytes, flags, err := ctrl.Encode(value)
	if err != nil {
		return 0, MutationToken{}, err
	}

	err = ctrl.Wait(b.client.AddEx(gocbcore.AddOptions{
		Key:          []byte(key),
		Value:        bytes,
		Flags:        flags,
		Expiry:       expiry,
		TraceContext: tracectx,
	}, func(res *gocbcore.StoreResult, err error) {
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

func (b *Bucket) replace(tracectx opentracing.SpanContext, key string, value interface{}, cas Cas, expiry uint32) (casOut Cas, mtOut MutationToken, errOut error) {
	ctrl := b.newOpManager(tracectx)

	bytes, flags, err := ctrl.Encode(value)
	if err != nil {
		return 0, MutationToken{}, err
	}

	err = ctrl.Wait(b.client.ReplaceEx(gocbcore.ReplaceOptions{
		Key:          []byte(key),
		Cas:          gocbcore.Cas(cas),
		Value:        bytes,
		Flags:        flags,
		Expiry:       expiry,
		TraceContext: tracectx,
	}, func(res *gocbcore.StoreResult, err error) {
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

func (b *Bucket) append(tracectx opentracing.SpanContext, key, value string) (casOut Cas, mtOut MutationToken, errOut error) {
	ctrl := b.newOpManager(tracectx)
	err := ctrl.Wait(b.client.AppendEx(gocbcore.AdjoinOptions{
		Key:          []byte(key),
		Value:        []byte(value),
		TraceContext: tracectx,
	}, func(res *gocbcore.AdjoinResult, err error) {
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

func (b *Bucket) prepend(tracectx opentracing.SpanContext, key, value string) (casOut Cas, mtOut MutationToken, errOut error) {
	ctrl := b.newOpManager(tracectx)
	err := ctrl.Wait(b.client.PrependEx(gocbcore.AdjoinOptions{
		Key:          []byte(key),
		Value:        []byte(value),
		TraceContext: tracectx,
	}, func(res *gocbcore.AdjoinResult, err error) {
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

func (b *Bucket) counterInc(tracectx opentracing.SpanContext, key string, delta, initial uint64, expiry uint32) (valueOut uint64, casOut Cas, mtOut MutationToken, errOut error) {
	ctrl := b.newOpManager(tracectx)
	err := ctrl.Wait(b.client.IncrementEx(gocbcore.CounterOptions{
		Key:          []byte(key),
		Delta:        delta,
		Initial:      initial,
		Expiry:       expiry,
		TraceContext: tracectx,
	}, func(res *gocbcore.CounterResult, err error) {
		if res != nil {
			valueOut = res.Value
			casOut = Cas(res.Cas)
			mtOut = MutationToken{res.MutationToken, b}
		}
		ctrl.Resolve(err)
	}))
	if err != nil {
		return 0, 0, MutationToken{}, err
	}

	return
}

func (b *Bucket) counterDec(tracectx opentracing.SpanContext, key string, delta, initial uint64, expiry uint32) (valueOut uint64, casOut Cas, mtOut MutationToken, errOut error) {
	ctrl := b.newOpManager(tracectx)
	err := ctrl.Wait(b.client.DecrementEx(gocbcore.CounterOptions{
		Key:          []byte(key),
		Delta:        delta,
		Initial:      initial,
		Expiry:       expiry,
		TraceContext: tracectx,
	}, func(res *gocbcore.CounterResult, err error) {
		if res != nil {
			valueOut = res.Value
			casOut = Cas(res.Cas)
			mtOut = MutationToken{res.MutationToken, b}
		}
		ctrl.Resolve(err)
	}))
	if err != nil {
		return 0, 0, MutationToken{}, err
	}

	return
}

func (b *Bucket) counter(tracectx opentracing.SpanContext, key string, delta, initial int64, expiry uint32) (uint64, Cas, MutationToken, error) {
	realInitial := uint64(0xFFFFFFFFFFFFFFFF)
	if initial >= 0 {
		realInitial = uint64(initial)
	}

	if delta > 0 {
		return b.counterInc(tracectx, key, uint64(delta), realInitial, expiry)
	} else if delta < 0 {
		return b.counterDec(tracectx, key, uint64(-delta), realInitial, expiry)
	} else {
		return 0, 0, MutationToken{}, clientError{"Delta must be a non-zero value."}
	}
}

func (b *Bucket) stats(tracectx opentracing.SpanContext, key string) (statsOut ServerStats, errOut error) {
	ctrl := b.newOpManager(tracectx)
	statsOut = make(ServerStats)

	err := ctrl.Wait(b.client.StatsEx(gocbcore.StatsOptions{
		Key:          key,
		TraceContext: tracectx,
	}, func(res *gocbcore.StatsResult, err error) {
		if res != nil {
			for curServer, curStats := range res.Servers {
				if curStats.Error != nil && err == nil {
					err = curStats.Error
				}
				statsOut[curServer] = curStats.Stats
			}
		}
		ctrl.Resolve(err)
	}))
	if err != nil {
		return nil, err
	}

	return
}
