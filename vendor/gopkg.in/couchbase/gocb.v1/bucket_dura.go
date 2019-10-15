package gocb

import (
	"github.com/opentracing/opentracing-go"
	"gopkg.in/couchbase/gocbcore.v7"
)

func (b *Bucket) observeOnceCas(tracectx opentracing.SpanContext, key []byte, cas Cas, forDelete bool, replicaIdx int, commCh chan uint) (pendingOp, error) {
	return b.client.ObserveEx(gocbcore.ObserveOptions{
		Key:          key,
		ReplicaIdx:   replicaIdx,
		TraceContext: tracectx,
	}, func(res *gocbcore.ObserveResult, err error) {
		if err != nil || res == nil {
			commCh <- 0
			return
		}

		didReplicate := false
		didPersist := false

		if res.KeyState == gocbcore.KeyStatePersisted {
			if !forDelete {
				if Cas(res.Cas) == cas {
					if replicaIdx != 0 {
						didReplicate = true
					}
					didPersist = true
				}
			}
		} else if res.KeyState == gocbcore.KeyStateNotPersisted {
			if !forDelete {
				if Cas(res.Cas) == cas {
					if replicaIdx != 0 {
						didReplicate = true
					}
				}
			}
		} else if res.KeyState == gocbcore.KeyStateDeleted {
			if forDelete {
				didReplicate = true
			}
		} else {
			if forDelete {
				didReplicate = true
				didPersist = true
			}
		}

		var out uint
		if didReplicate {
			out |= 1
		}
		if didPersist {
			out |= 2
		}
		commCh <- out
	})
}

func (b *Bucket) observeOnceSeqNo(tracectx opentracing.SpanContext, mt MutationToken, replicaIdx int, commCh chan uint) (pendingOp, error) {
	return b.client.ObserveVbEx(gocbcore.ObserveVbOptions{
		VbId:         mt.token.VbId,
		VbUuid:       mt.token.VbUuid,
		ReplicaIdx:   replicaIdx,
		TraceContext: tracectx,
	}, func(res *gocbcore.ObserveVbResult, err error) {
		if err != nil || res == nil {
			commCh <- 0
			return
		}

		didReplicate := res.CurrentSeqNo >= mt.token.SeqNo
		didPersist := res.PersistSeqNo >= mt.token.SeqNo

		var out uint
		if didReplicate {
			out |= 1
		}
		if didPersist {
			out |= 2
		}
		commCh <- out
	})
}

func (b *Bucket) observeOne(tracectx opentracing.SpanContext, key []byte, mt MutationToken, cas Cas, forDelete bool, replicaIdx int, replicaCh, persistCh chan bool) {
	observeOnce := func(commCh chan uint) (pendingOp, error) {
		if mt.token.VbUuid != 0 && mt.token.SeqNo != 0 {
			return b.observeOnceSeqNo(tracectx, mt, replicaIdx, commCh)
		}
		return b.observeOnceCas(tracectx, key, cas, forDelete, replicaIdx, commCh)
	}

	sentReplicated := false
	sentPersisted := false

	failMe := func() {
		if !sentReplicated {
			replicaCh <- false
			sentReplicated = true
		}
		if !sentPersisted {
			persistCh <- false
			sentPersisted = true
		}
	}

	timeoutTmr := gocbcore.AcquireTimer(b.duraTimeout)

	commCh := make(chan uint)
	for {
		op, err := observeOnce(commCh)
		if err != nil {
			gocbcore.ReleaseTimer(timeoutTmr, false)
			failMe()
			return
		}

		select {
		case val := <-commCh:
			// Got Value
			if (val&1) != 0 && !sentReplicated {
				replicaCh <- true
				sentReplicated = true
			}
			if (val&2) != 0 && !sentPersisted {
				persistCh <- true
				sentPersisted = true
			}

			if sentReplicated && sentPersisted {
				return
			}

			waitTmr := gocbcore.AcquireTimer(b.duraPollTimeout)
			select {
			case <-waitTmr.C:
				gocbcore.ReleaseTimer(waitTmr, true)
				// Fall through to outside for loop
			case <-timeoutTmr.C:
				gocbcore.ReleaseTimer(waitTmr, false)
				gocbcore.ReleaseTimer(timeoutTmr, true)
				failMe()
				return
			}

		case <-timeoutTmr.C:
			// Timed out
			op.Cancel()
			gocbcore.ReleaseTimer(timeoutTmr, true)
			failMe()
			return
		}
	}
}

func (b *Bucket) durability(tracectx opentracing.SpanContext, key string, cas Cas, mt MutationToken, replicaTo, persistTo uint, forDelete bool) error {
	numServers := b.client.NumReplicas() + 1

	if replicaTo > uint(numServers-1) || persistTo > uint(numServers) {
		return ErrNotEnoughReplicas
	}

	keyBytes := []byte(key)

	replicaCh := make(chan bool, numServers)
	persistCh := make(chan bool, numServers)

	for replicaIdx := 0; replicaIdx < numServers; replicaIdx++ {
		go b.observeOne(tracectx, keyBytes, mt, cas, forDelete, replicaIdx, replicaCh, persistCh)
	}

	results := int(0)
	replicas := uint(0)
	persists := uint(0)

	for {
		select {
		case rV := <-replicaCh:
			if rV {
				replicas++
			}
			results++
		case pV := <-persistCh:
			if pV {
				persists++
			}
			results++
		}

		if replicas >= replicaTo && persists >= persistTo {
			return nil
		} else if results == (numServers * 2) {
			return ErrDurabilityTimeout
		}
	}
}

// TouchDura touches a document, specifying a new expiry time for it.  Additionally checks document durability.
// The Cas value must be 0.
func (b *Bucket) TouchDura(key string, cas Cas, expiry uint32, replicateTo, persistTo uint) (Cas, error) {
	span := b.startKvOpTrace("TouchDura")
	defer span.Finish()

	if cas != 0 {
		return 0, ErrNonZeroCas
	}

	cas, mt, err := b.touch(span.Context(), key, expiry)
	if err != nil {
		return cas, err
	}
	return cas, b.durability(span.Context(), key, cas, mt, replicateTo, persistTo, false)
}

// RemoveDura removes a document from the bucket.  Additionally checks document durability.
func (b *Bucket) RemoveDura(key string, cas Cas, replicateTo, persistTo uint) (Cas, error) {
	span := b.startKvOpTrace("RemoveDura")
	defer span.Finish()

	cas, mt, err := b.remove(span.Context(), key, cas)
	if err != nil {
		return cas, err
	}
	return cas, b.durability(span.Context(), key, cas, mt, replicateTo, persistTo, true)
}

// UpsertDura inserts or replaces a document in the bucket.  Additionally checks document durability.
func (b *Bucket) UpsertDura(key string, value interface{}, expiry uint32, replicateTo, persistTo uint) (Cas, error) {
	span := b.startKvOpTrace("UpsertDura")
	defer span.Finish()

	cas, mt, err := b.upsert(span.Context(), key, value, expiry)
	if err != nil {
		return cas, err
	}
	return cas, b.durability(span.Context(), key, cas, mt, replicateTo, persistTo, false)
}

// InsertDura inserts a new document to the bucket.  Additionally checks document durability.
func (b *Bucket) InsertDura(key string, value interface{}, expiry uint32, replicateTo, persistTo uint) (Cas, error) {
	span := b.startKvOpTrace("InsertDura")
	defer span.Finish()

	cas, mt, err := b.insert(span.Context(), key, value, expiry)
	if err != nil {
		return cas, err
	}
	return cas, b.durability(span.Context(), key, cas, mt, replicateTo, persistTo, false)
}

// ReplaceDura replaces a document in the bucket.  Additionally checks document durability.
func (b *Bucket) ReplaceDura(key string, value interface{}, cas Cas, expiry uint32, replicateTo, persistTo uint) (Cas, error) {
	span := b.startKvOpTrace("ReplaceDura")
	defer span.Finish()

	cas, mt, err := b.replace(span.Context(), key, value, cas, expiry)
	if err != nil {
		return cas, err
	}
	return cas, b.durability(span.Context(), key, cas, mt, replicateTo, persistTo, false)
}

// AppendDura appends a string value to a document.  Additionally checks document durability.
func (b *Bucket) AppendDura(key, value string, replicateTo, persistTo uint) (Cas, error) {
	span := b.startKvOpTrace("AppendDura")
	defer span.Finish()

	cas, mt, err := b.append(span.Context(), key, value)
	if err != nil {
		return cas, err
	}
	return cas, b.durability(span.Context(), key, cas, mt, replicateTo, persistTo, false)
}

// PrependDura prepends a string value to a document.  Additionally checks document durability.
func (b *Bucket) PrependDura(key, value string, replicateTo, persistTo uint) (Cas, error) {
	span := b.startKvOpTrace("PrependDura")
	defer span.Finish()

	cas, mt, err := b.prepend(span.Context(), key, value)
	if err != nil {
		return cas, err
	}
	return cas, b.durability(span.Context(), key, cas, mt, replicateTo, persistTo, false)
}

// CounterDura performs an atomic addition or subtraction for an integer document.  Additionally checks document durability.
func (b *Bucket) CounterDura(key string, delta, initial int64, expiry uint32, replicateTo, persistTo uint) (uint64, Cas, error) {
	span := b.startKvOpTrace("CounterDura")
	defer span.Finish()

	val, cas, mt, err := b.counter(span.Context(), key, delta, initial, expiry)
	if err != nil {
		return val, cas, err
	}
	return val, cas, b.durability(span.Context(), key, cas, mt, replicateTo, persistTo, false)
}
