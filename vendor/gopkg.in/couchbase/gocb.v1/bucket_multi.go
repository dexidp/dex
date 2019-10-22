package gocb

import (
	"gopkg.in/couchbase/gocbcore.v7"
)

type bulkOp struct {
	pendop gocbcore.PendingOp
}

func (op *bulkOp) cancel() bool {
	if op.pendop == nil {
		return true
	}
	res := op.pendop.Cancel()
	op.pendop = nil
	return res
}

// BulkOp represents a single operation that can be submitted (within a list of more operations) to .Do()
// You can create a bulk operation by instantiating one of the implementations of BulkOp,
// such as GetOp, UpsertOp, ReplaceOp, and more.
type BulkOp interface {
	execute(*Bucket, chan BulkOp)
	markError(err error)
	cancel() bool
}

// Do execute one or more `BulkOp` items in parallel.
func (b *Bucket) Do(ops []BulkOp) error {
	timeoutTmr := gocbcore.AcquireTimer(b.bulkOpTimeout)

	// Make the channel big enough to hold all our ops in case
	//   we get delayed inside execute (don't want to block the
	//   individual op handlers when they dispatch their signal).
	signal := make(chan BulkOp, len(ops))
	for _, item := range ops {
		item.execute(b, signal)
	}
	for range ops {
		select {
		case item := <-signal:
			// We're really just clearing the pendop from this thread,
			//   since it already completed, no cancel actually occurs
			item.cancel()
		case <-timeoutTmr.C:
			gocbcore.ReleaseTimer(timeoutTmr, true)
			for _, item := range ops {
				if !item.cancel() {
					<-signal
					continue
				}

				// We use this method to mark the individual items as
				// having timed out so we don't move `Err` in bulkOp
				// and break backwards compatibility.
				item.markError(ErrTimeout)
			}
			return ErrTimeout
		}
	}
	gocbcore.ReleaseTimer(timeoutTmr, false)
	return nil
}

// GetOp represents a type of `BulkOp` used for Get operations. See BulkOp.
type GetOp struct {
	bulkOp

	Key   string
	Value interface{}
	Cas   Cas
	Err   error
}

func (item *GetOp) markError(err error) {
	item.Err = err
}

func (item *GetOp) execute(b *Bucket, signal chan BulkOp) {
	op, err := b.client.Get([]byte(item.Key), func(bytes []byte, flags uint32, cas gocbcore.Cas, err error) {
		item.Err = err
		if item.Err == nil {
			item.Err = b.transcoder.Decode(bytes, flags, item.Value)
			if item.Err == nil {
				item.Cas = Cas(cas)
			}
		}
		signal <- item
	})
	if err != nil {
		item.Err = err
		signal <- item
	} else {
		item.bulkOp.pendop = op
	}
}

// GetAndTouchOp represents a type of `BulkOp` used for GetAndTouch operations. See BulkOp.
type GetAndTouchOp struct {
	bulkOp

	Key    string
	Value  interface{}
	Expiry uint32
	Cas    Cas
	Err    error
}

func (item *GetAndTouchOp) markError(err error) {
	item.Err = err
}

func (item *GetAndTouchOp) execute(b *Bucket, signal chan BulkOp) {
	op, err := b.client.GetAndTouch([]byte(item.Key), item.Expiry,
		func(bytes []byte, flags uint32, cas gocbcore.Cas, err error) {
			item.Err = err
			if item.Err == nil {
				item.Err = b.transcoder.Decode(bytes, flags, item.Value)
				if item.Err == nil {
					item.Cas = Cas(cas)
				}
			}
			signal <- item
		})
	if err != nil {
		item.Err = err
		signal <- item
	} else {
		item.bulkOp.pendop = op
	}
}

// TouchOp represents a type of `BulkOp` used for Touch operations. See BulkOp.
type TouchOp struct {
	bulkOp

	Key    string
	Expiry uint32
	Cas    Cas
	Err    error
}

func (item *TouchOp) markError(err error) {
	item.Err = err
}

func (item *TouchOp) execute(b *Bucket, signal chan BulkOp) {
	op, err := b.client.Touch([]byte(item.Key), gocbcore.Cas(item.Cas), item.Expiry,
		func(cas gocbcore.Cas, mutToken gocbcore.MutationToken, err error) {
			item.Err = err
			if item.Err == nil {
				item.Cas = Cas(cas)
			}
			signal <- item
		})
	if err != nil {
		item.Err = err
		signal <- item
	} else {
		item.bulkOp.pendop = op
	}
}

// RemoveOp represents a type of `BulkOp` used for Remove operations. See BulkOp.
type RemoveOp struct {
	bulkOp

	Key string
	Cas Cas
	Err error
}

func (item *RemoveOp) markError(err error) {
	item.Err = err
}

func (item *RemoveOp) execute(b *Bucket, signal chan BulkOp) {
	op, err := b.client.Remove([]byte(item.Key), gocbcore.Cas(item.Cas),
		func(cas gocbcore.Cas, mutToken gocbcore.MutationToken, err error) {
			item.Err = err
			if item.Err == nil {
				item.Cas = Cas(cas)
			}
			signal <- item
		})
	if err != nil {
		item.Err = err
		signal <- item
	} else {
		item.bulkOp.pendop = op
	}
}

// UpsertOp represents a type of `BulkOp` used for Upsert operations. See BulkOp.
type UpsertOp struct {
	bulkOp

	Key    string
	Value  interface{}
	Expiry uint32
	Cas    Cas
	Err    error
}

func (item *UpsertOp) markError(err error) {
	item.Err = err
}

func (item *UpsertOp) execute(b *Bucket, signal chan BulkOp) {
	bytes, flags, err := b.transcoder.Encode(item.Value)
	if err != nil {
		item.Err = err
		signal <- item
	} else {
		op, err := b.client.Set([]byte(item.Key), bytes, flags, item.Expiry,
			func(cas gocbcore.Cas, mutToken gocbcore.MutationToken, err error) {
				item.Err = err
				if item.Err == nil {
					item.Cas = Cas(cas)
				}
				signal <- item
			})
		if err != nil {
			item.Err = err
			signal <- item
		} else {
			item.bulkOp.pendop = op
		}
	}
}

// InsertOp represents a type of `BulkOp` used for Insert operations. See BulkOp.
type InsertOp struct {
	bulkOp

	Key    string
	Value  interface{}
	Expiry uint32
	Cas    Cas
	Err    error
}

func (item *InsertOp) markError(err error) {
	item.Err = err
}

func (item *InsertOp) execute(b *Bucket, signal chan BulkOp) {
	bytes, flags, err := b.transcoder.Encode(item.Value)
	if err != nil {
		item.Err = err
		signal <- item
	} else {
		op, err := b.client.Add([]byte(item.Key), bytes, flags, item.Expiry,
			func(cas gocbcore.Cas, mutToken gocbcore.MutationToken, err error) {
				item.Err = err
				if item.Err == nil {
					item.Cas = Cas(cas)
				}
				signal <- item
			})
		if err != nil {
			item.Err = err
			signal <- item
		} else {
			item.bulkOp.pendop = op
		}
	}
}

// ReplaceOp represents a type of `BulkOp` used for Replace operations. See BulkOp.
type ReplaceOp struct {
	bulkOp

	Key    string
	Value  interface{}
	Expiry uint32
	Cas    Cas
	Err    error
}

func (item *ReplaceOp) markError(err error) {
	item.Err = err
}

func (item *ReplaceOp) execute(b *Bucket, signal chan BulkOp) {
	bytes, flags, err := b.transcoder.Encode(item.Value)
	if err != nil {
		item.Err = err
		signal <- item
	} else {
		op, err := b.client.Replace([]byte(item.Key), bytes, flags, gocbcore.Cas(item.Cas), item.Expiry,
			func(cas gocbcore.Cas, mutToken gocbcore.MutationToken, err error) {
				item.Err = err
				if item.Err == nil {
					item.Cas = Cas(cas)
				}
				signal <- item
			})
		if err != nil {
			item.Err = err
			signal <- item
		} else {
			item.bulkOp.pendop = op
		}
	}
}

// AppendOp represents a type of `BulkOp` used for Append operations. See BulkOp.
type AppendOp struct {
	bulkOp

	Key   string
	Value string
	Cas   Cas
	Err   error
}

func (item *AppendOp) markError(err error) {
	item.Err = err
}

func (item *AppendOp) execute(b *Bucket, signal chan BulkOp) {
	op, err := b.client.Append([]byte(item.Key), []byte(item.Value),
		func(cas gocbcore.Cas, mutToken gocbcore.MutationToken, err error) {
			item.Err = err
			if item.Err == nil {
				item.Cas = Cas(cas)
			}
			signal <- item
		})
	if err != nil {
		item.Err = err
		signal <- item
	} else {
		item.bulkOp.pendop = op
	}
}

// PrependOp represents a type of `BulkOp` used for Prepend operations. See BulkOp.
type PrependOp struct {
	bulkOp

	Key   string
	Value string
	Cas   Cas
	Err   error
}

func (item *PrependOp) markError(err error) {
	item.Err = err
}

func (item *PrependOp) execute(b *Bucket, signal chan BulkOp) {
	op, err := b.client.Prepend([]byte(item.Key), []byte(item.Value),
		func(cas gocbcore.Cas, mutToken gocbcore.MutationToken, err error) {
			item.Err = err
			if item.Err == nil {
				item.Cas = Cas(cas)
			}
			signal <- item
		})
	if err != nil {
		item.Err = err
		signal <- item
	} else {
		item.bulkOp.pendop = op
	}
}

// CounterOp represents a type of `BulkOp` used for Counter operations. See BulkOp.
type CounterOp struct {
	bulkOp

	Key     string
	Delta   int64
	Initial int64
	Expiry  uint32
	Cas     Cas
	Value   uint64
	Err     error
}

func (item *CounterOp) markError(err error) {
	item.Err = err
}

func (item *CounterOp) execute(b *Bucket, signal chan BulkOp) {
	realInitial := uint64(0xFFFFFFFFFFFFFFFF)
	if item.Initial > 0 {
		realInitial = uint64(item.Initial)
	}

	if item.Delta > 0 {
		op, err := b.client.Increment([]byte(item.Key), uint64(item.Delta), realInitial, item.Expiry,
			func(value uint64, cas gocbcore.Cas, mutToken gocbcore.MutationToken, err error) {
				item.Err = err
				if item.Err == nil {
					item.Value = value
					item.Cas = Cas(cas)
				}
				signal <- item
			})
		if err != nil {
			item.Err = err
			signal <- item
		} else {
			item.bulkOp.pendop = op
		}
	} else if item.Delta < 0 {
		op, err := b.client.Decrement([]byte(item.Key), uint64(-item.Delta), realInitial, item.Expiry,
			func(value uint64, cas gocbcore.Cas, mutToken gocbcore.MutationToken, err error) {
				item.Err = err
				if item.Err == nil {
					item.Value = value
					item.Cas = Cas(cas)
				}
				signal <- item
			})
		if err != nil {
			item.Err = err
			signal <- item
		} else {
			item.bulkOp.pendop = op
		}
	} else {
		item.Err = clientError{"Delta must be a non-zero value."}
		signal <- item
	}
}
