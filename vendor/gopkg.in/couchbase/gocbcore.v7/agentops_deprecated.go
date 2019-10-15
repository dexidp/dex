package gocbcore

// GetCallback is invoked with the results of `Get` operations.
// DEPRECATED
type GetCallback func([]byte, uint32, Cas, error)

// Get retrieves a document.
// DEPRECATED: See GetEx
func (agent *Agent) Get(key []byte, cb GetCallback) (PendingOp, error) {
	return agent.GetEx(GetOptions{
		Key: key,
	}, func(resp *GetResult, err error) {
		if resp != nil {
			cb(resp.Value, resp.Flags, resp.Cas, err)
			return
		}

		cb(nil, 0, 0, err)
	})
}

// GetAndTouch retrieves a document and updates its expiry.
// DEPRECATED: See GetAndTouchEx
func (agent *Agent) GetAndTouch(key []byte, expiry uint32, cb GetCallback) (PendingOp, error) {
	return agent.GetAndTouchEx(GetAndTouchOptions{
		Key:    key,
		Expiry: expiry,
	}, func(resp *GetAndTouchResult, err error) {
		if resp != nil {
			cb(resp.Value, resp.Flags, resp.Cas, err)
			return
		}

		cb(nil, 0, 0, err)
	})
}

// GetAndLock retrieves a document and locks it.
// DEPRECATED: See GetAndLockEx
func (agent *Agent) GetAndLock(key []byte, lockTime uint32, cb GetCallback) (PendingOp, error) {
	return agent.GetAndLockEx(GetAndLockOptions{
		Key:      key,
		LockTime: lockTime,
	}, func(resp *GetAndLockResult, err error) {
		if resp != nil {
			cb(resp.Value, resp.Flags, resp.Cas, err)
			return
		}

		cb(nil, 0, 0, err)
	})
}

// GetReplica retrieves a document from a replica server.
// DEPRECATED: See GetReplicaEx
func (agent *Agent) GetReplica(key []byte, replicaIdx int, cb GetCallback) (PendingOp, error) {
	return agent.GetReplicaEx(GetReplicaOptions{
		Key:        key,
		ReplicaIdx: replicaIdx,
	}, func(resp *GetReplicaResult, err error) {
		if resp != nil {
			cb(resp.Value, resp.Flags, resp.Cas, err)
			return
		}

		cb(nil, 0, 0, err)
	})
}

// TouchCallback is invoked with the results of `Touch` operations.
// DEPRECATED
type TouchCallback func(Cas, MutationToken, error)

// Touch updates the expiry for a document.
// DEPRECATED: See TouchEx
func (agent *Agent) Touch(key []byte, cas Cas, expiry uint32, cb TouchCallback) (PendingOp, error) {
	return agent.TouchEx(TouchOptions{
		Key:    key,
		Cas:    cas,
		Expiry: expiry,
	}, func(resp *TouchResult, err error) {
		if resp != nil {
			cb(resp.Cas, resp.MutationToken, err)
			return
		}

		cb(0, MutationToken{}, err)
	})
}

// UnlockCallback is invoked with the results of `Unlock` operations.
// DEPRECATED
type UnlockCallback func(Cas, MutationToken, error)

// Unlock unlocks a locked document.
// DEPRECATED: See UnlockEx
func (agent *Agent) Unlock(key []byte, cas Cas, cb UnlockCallback) (PendingOp, error) {
	return agent.UnlockEx(UnlockOptions{
		Key: key,
		Cas: cas,
	}, func(resp *UnlockResult, err error) {
		if resp != nil {
			cb(resp.Cas, resp.MutationToken, err)
			return
		}

		cb(0, MutationToken{}, err)
	})
}

// RemoveCallback is invoked with the results of `Remove` operations.
// DEPRECATED
type RemoveCallback func(Cas, MutationToken, error)

// Remove removes a document.
// DEPRECATED: See DeleteEx
func (agent *Agent) Remove(key []byte, cas Cas, cb RemoveCallback) (PendingOp, error) {
	return agent.DeleteEx(DeleteOptions{
		Key: key,
		Cas: cas,
	}, func(resp *DeleteResult, err error) {
		if resp != nil {
			cb(resp.Cas, resp.MutationToken, err)
			return
		}

		cb(0, MutationToken{}, err)
	})
}

// StoreCallback is invoked with the results of any basic storage operations.
// DEPRECATED
type StoreCallback func(Cas, MutationToken, error)

// Add stores a document as long as it does not already exist.
// DEPRECATED: See AddEx
func (agent *Agent) Add(key, value []byte, flags uint32, expiry uint32, cb StoreCallback) (PendingOp, error) {
	return agent.AddEx(AddOptions{
		Key:    key,
		Value:  value,
		Flags:  flags,
		Expiry: expiry,
	}, func(resp *StoreResult, err error) {
		if resp != nil {
			cb(resp.Cas, resp.MutationToken, err)
			return
		}

		cb(0, MutationToken{}, err)
	})
}

// Set stores a document.
// DEPRECATED: See SetEx
func (agent *Agent) Set(key, value []byte, flags uint32, expiry uint32, cb StoreCallback) (PendingOp, error) {
	return agent.SetEx(SetOptions{
		Key:    key,
		Value:  value,
		Flags:  flags,
		Expiry: expiry,
	}, func(resp *StoreResult, err error) {
		if resp != nil {
			cb(resp.Cas, resp.MutationToken, err)
			return
		}

		cb(0, MutationToken{}, err)
	})
}

// Replace replaces the value of a Couchbase document with another value.
// DEPRECATED: See ReplaceEx
func (agent *Agent) Replace(key, value []byte, flags uint32, cas Cas, expiry uint32, cb StoreCallback) (PendingOp, error) {
	return agent.ReplaceEx(ReplaceOptions{
		Key:    key,
		Value:  value,
		Flags:  flags,
		Cas:    cas,
		Expiry: expiry,
	}, func(resp *StoreResult, err error) {
		if resp != nil {
			cb(resp.Cas, resp.MutationToken, err)
			return
		}

		cb(0, MutationToken{}, err)
	})
}

// Append appends some bytes to a document.
// DEPRECATED: See AppendEx
func (agent *Agent) Append(key, value []byte, cb StoreCallback) (PendingOp, error) {
	return agent.AppendEx(AdjoinOptions{
		Key:   key,
		Value: value,
	}, func(resp *AdjoinResult, err error) {
		if resp != nil {
			cb(resp.Cas, resp.MutationToken, err)
			return
		}

		cb(0, MutationToken{}, err)
	})
}

// Prepend prepends some bytes to a document.
// DEPRECATED: See PrependEx
func (agent *Agent) Prepend(key, value []byte, cb StoreCallback) (PendingOp, error) {
	return agent.PrependEx(AdjoinOptions{
		Key:   key,
		Value: value,
	}, func(resp *AdjoinResult, err error) {
		if resp != nil {
			cb(resp.Cas, resp.MutationToken, err)
			return
		}

		cb(0, MutationToken{}, err)
	})
}

// CounterCallback is invoked with the results of `Counter` operations.
// DEPRECATED
type CounterCallback func(uint64, Cas, MutationToken, error)

// Increment increments the unsigned integer value in a document.
// DEPRECATED: See IncrementEx
func (agent *Agent) Increment(key []byte, delta, initial uint64, expiry uint32, cb CounterCallback) (PendingOp, error) {
	return agent.IncrementEx(CounterOptions{
		Key:     key,
		Delta:   delta,
		Initial: initial,
		Expiry:  expiry,
	}, func(resp *CounterResult, err error) {
		if resp != nil {
			cb(resp.Value, resp.Cas, resp.MutationToken, err)
			return
		}

		cb(0, 0, MutationToken{}, err)
	})
}

// Decrement decrements the unsigned integer value in a document.
// DEPRECATED: See DecrementEx
func (agent *Agent) Decrement(key []byte, delta, initial uint64, expiry uint32, cb CounterCallback) (PendingOp, error) {
	return agent.DecrementEx(CounterOptions{
		Key:     key,
		Delta:   delta,
		Initial: initial,
		Expiry:  expiry,
	}, func(resp *CounterResult, err error) {
		if resp != nil {
			cb(resp.Value, resp.Cas, resp.MutationToken, err)
			return
		}

		cb(0, 0, MutationToken{}, err)
	})
}

// GetRandomCallback is invoked with the results of `GetRandom` operations.
// DEPRECATED
type GetRandomCallback func([]byte, []byte, uint32, Cas, error)

// GetRandom retrieves the key and value of a random document stored within Couchbase Server
// DEPRECATED: See GetRandomEx.
func (agent *Agent) GetRandom(cb GetRandomCallback) (PendingOp, error) {
	return agent.GetRandomEx(GetRandomOptions{}, func(resp *GetRandomResult, err error) {
		if resp != nil {
			cb(resp.Key, resp.Value, resp.Flags, resp.Cas, err)
			return
		}

		cb(nil, nil, 0, 0, err)
	})
}

// ServerStatsCallback is invoked with the results of `Stats` operations.
// DEPRECATED
type ServerStatsCallback func(stats map[string]SingleServerStats)

// Stats retrieves statistics information from the server.  Note that as this
// function is an aggregator across numerous servers, there are no guarantees
// about the consistency of the results.  Occasionally, some nodes may not be
// represented in the results, or there may be conflicting information between
// multiple nodes (a vbucket active on two separate nodes at once).
// DEPRECATED: See StatsEx
func (agent *Agent) Stats(key string, cb ServerStatsCallback) (PendingOp, error) {
	return agent.StatsEx(StatsOptions{
		Key: key,
	}, func(resp *StatsResult, err error) {
		if resp != nil {
			cb(resp.Servers)
			return
		}
		cb(nil)
	})
}

// PingCallback is invoked with the results of a multi-node ping operation.
// DEPRECATED
type PingCallback func(services []PingResult)

// Ping pings all of the servers we are connected to and returns
// a report regarding the pings that were performed.
// DEPRECATED: See PingKvEx
func (agent *Agent) Ping(cb PingCallback) (PendingOp, error) {
	return agent.PingKvEx(PingKvOptions{}, func(resp *PingKvResult, err error) {
		if resp != nil {
			cb(resp.Services)
			return
		}

		cb(nil)
	})
}

// ObserveCallback is invoked with the results of `Observe` operations.
// DEPRECATED
type ObserveCallback func(KeyState, Cas, error)

// Observe retrieves the current CAS and persistence state for a document.
// DEPRECATED: See ObserveEx
func (agent *Agent) Observe(key []byte, replicaIdx int, cb ObserveCallback) (PendingOp, error) {
	return agent.ObserveEx(ObserveOptions{
		Key:        key,
		ReplicaIdx: replicaIdx,
	}, func(resp *ObserveResult, err error) {
		if resp != nil {
			cb(resp.KeyState, resp.Cas, err)
			return
		}
		cb(0, 0, err)
	})
}

// ObserveSeqNoStats represents the stats returned from an observe operation.
// DEPRECATED
type ObserveSeqNoStats struct {
	DidFailover  bool
	VbId         uint16
	VbUuid       VbUuid
	PersistSeqNo SeqNo
	CurrentSeqNo SeqNo
	OldVbUuid    VbUuid
	LastSeqNo    SeqNo
}

// ObserveSeqNoCallback is invoked with the results of `ObserveSeqNo` operations.
// DEPRECATED
type ObserveSeqNoCallback func(SeqNo, SeqNo, error)

// ObserveSeqNo retrieves the persistence state sequence numbers for a particular VBucket.
// DEPRECATED: See ObserveVbEx
func (agent *Agent) ObserveSeqNo(key []byte, vbUuid VbUuid, replicaIdx int, cb ObserveSeqNoCallback) (PendingOp, error) {
	vbId := agent.KeyToVbucket(key)
	return agent.ObserveSeqNoEx(vbId, vbUuid, replicaIdx, func(stats *ObserveSeqNoStats, err error) {
		if err != nil {
			cb(0, 0, err)
			return
		}

		if !stats.DidFailover {
			cb(stats.CurrentSeqNo, stats.PersistSeqNo, nil)
		} else {
			cb(stats.LastSeqNo, stats.LastSeqNo, nil)
		}
	})
}

// ObserveSeqNoExCallback is invoked with the results of `ObserveSeqNoEx` operations.
// DEPRECATED
type ObserveSeqNoExCallback func(*ObserveSeqNoStats, error)

// ObserveSeqNoEx retrieves the persistence state sequence numbers for a particular VBucket
// and includes additional details not included by the basic version.
// DEPRECATED: See ObserveVbEx
func (agent *Agent) ObserveSeqNoEx(vbId uint16, vbUuid VbUuid, replicaIdx int, cb ObserveSeqNoExCallback) (PendingOp, error) {
	return agent.ObserveVbEx(ObserveVbOptions{
		VbId:       vbId,
		VbUuid:     vbUuid,
		ReplicaIdx: replicaIdx,
	}, func(resp *ObserveVbResult, err error) {
		if resp != nil {
			cb(&ObserveSeqNoStats{
				DidFailover:  resp.DidFailover,
				VbId:         resp.VbId,
				VbUuid:       resp.VbUuid,
				PersistSeqNo: resp.PersistSeqNo,
				CurrentSeqNo: resp.CurrentSeqNo,
				OldVbUuid:    resp.OldVbUuid,
				LastSeqNo:    resp.LastSeqNo,
			}, err)
			return
		}
		cb(nil, err)
	})
}

// GetMetaCallback is invoked with the results of `GetMeta` operations.
// DEPRECATED
type GetMetaCallback func([]byte, uint32, Cas, uint32, SeqNo, uint8, uint32, error)

// GetMeta retrieves a document along with some internal Couchbase meta-data.
// DEPRECATED: See GetMetaEx
func (agent *Agent) GetMeta(key []byte, cb GetMetaCallback) (PendingOp, error) {
	return agent.GetMetaEx(GetMetaOptions{
		Key: key,
	}, func(resp *GetMetaResult, err error) {
		if resp != nil {
			cb(resp.Value, resp.Flags, resp.Cas, resp.Expiry, resp.SeqNo, resp.Datatype, resp.Deleted, err)
			return
		}

		cb(nil, 0, 0, 0, 0, 0, 0, err)
	})
}

// SetMeta stores a document along with setting some internal Couchbase meta-data.
// DEPRECATED: See SetMetaEx
func (agent *Agent) SetMeta(key, value, extra []byte, datatype uint8, options, flags, expiry uint32, cas, revNo uint64, cb StoreCallback) (PendingOp, error) {
	return agent.SetMetaEx(SetMetaOptions{
		Key:      key,
		Value:    value,
		Extra:    extra,
		Datatype: datatype,
		Options:  options,
		Flags:    flags,
		Expiry:   expiry,
		Cas:      Cas(cas),
		RevNo:    revNo,
	}, func(resp *SetMetaResult, err error) {
		if resp != nil {
			cb(resp.Cas, resp.MutationToken, err)
			return
		}

		cb(0, MutationToken{}, err)
	})
}

// DeleteMeta deletes a document along with setting some internal Couchbase meta-data.
// DEPRECATED: See DeleteMetaEx
func (agent *Agent) DeleteMeta(key, value, extra []byte, datatype uint8, options, flags, expiry uint32, cas, revNo uint64, cb RemoveCallback) (PendingOp, error) {
	return agent.DeleteMetaEx(DeleteMetaOptions{
		Key:      key,
		Value:    value,
		Extra:    extra,
		Datatype: datatype,
		Options:  options,
		Flags:    flags,
		Expiry:   expiry,
		Cas:      Cas(cas),
		RevNo:    revNo,
	}, func(resp *DeleteMetaResult, err error) {
		if resp != nil {
			cb(resp.Cas, resp.MutationToken, err)
			return
		}

		cb(0, MutationToken{}, err)
	})
}

// GetInCallback is invoked with the results of `GetIn` operations.
// DEPRECATED
type GetInCallback func([]byte, Cas, error)

// GetIn retrieves the value at a particular path within a JSON document.
// DEPRECATED: See GetInEx
func (agent *Agent) GetIn(key []byte, path string, flags SubdocFlag, cb GetInCallback) (PendingOp, error) {
	return agent.GetInEx(GetInOptions{
		Key:   key,
		Path:  path,
		Flags: flags,
	}, func(resp *GetInResult, err error) {
		if resp != nil {
			cb(resp.Value, resp.Cas, err)
			return
		}

		cb(nil, 0, err)
	})
}

// ExistsInCallback is invoked with the results of `ExistsIn` operations.
// DEPRECATED
type ExistsInCallback func(Cas, error)

// ExistsIn returns whether a particular path exists within a document.
// DEPRECATED: See ExistsInEx
func (agent *Agent) ExistsIn(key []byte, path string, flags SubdocFlag, cb ExistsInCallback) (PendingOp, error) {
	return agent.ExistsInEx(ExistsInOptions{
		Key:   key,
		Path:  path,
		Flags: flags,
	}, func(resp *ExistsInResult, err error) {
		if resp != nil {
			cb(resp.Cas, err)
			return
		}

		cb(0, err)
	})
}

// StoreInCallback is invoked with the results of any sub-document storage operations.
// DEPRECATED
type StoreInCallback func(Cas, MutationToken, error)

// SetIn sets the value at a path within a document.
// DEPRECATED: See SetInEx
func (agent *Agent) SetIn(key []byte, path string, value []byte, flags SubdocFlag, cas Cas, expiry uint32, cb StoreInCallback) (PendingOp, error) {
	return agent.SetInEx(StoreInOptions{
		Key:    key,
		Path:   path,
		Value:  value,
		Flags:  flags,
		Cas:    cas,
		Expiry: expiry,
	}, func(resp *StoreInResult, err error) {
		if resp != nil {
			cb(resp.Cas, resp.MutationToken, err)
			return
		}

		cb(0, MutationToken{}, err)
	})
}

// AddIn adds a value at the path within a document.  This method
// works like SetIn, but only only succeeds if the path does not
// currently exist.
// DEPRECATED: See AddInEx
func (agent *Agent) AddIn(key []byte, path string, value []byte, flags SubdocFlag, cas Cas, expiry uint32, cb StoreInCallback) (PendingOp, error) {
	return agent.AddInEx(StoreInOptions{
		Key:    key,
		Path:   path,
		Value:  value,
		Flags:  flags,
		Cas:    cas,
		Expiry: expiry,
	}, func(resp *StoreInResult, err error) {
		if resp != nil {
			cb(resp.Cas, resp.MutationToken, err)
			return
		}

		cb(0, MutationToken{}, err)
	})
}

// ReplaceIn replaces the value at the path within a document.
// This method works like SetIn, but only only succeeds
// if the path currently exists.
// DEPRECATED: See ReplaceInEx
func (agent *Agent) ReplaceIn(key []byte, path string, value []byte, cas Cas, expiry uint32, flags SubdocFlag, cb StoreInCallback) (PendingOp, error) {
	return agent.ReplaceInEx(StoreInOptions{
		Key:    key,
		Path:   path,
		Value:  value,
		Flags:  flags,
		Cas:    cas,
		Expiry: expiry,
	}, func(resp *StoreInResult, err error) {
		if resp != nil {
			cb(resp.Cas, resp.MutationToken, err)
			return
		}

		cb(0, MutationToken{}, err)
	})
}

// PushFrontIn pushes an entry to the front of an array at a path within a document.
// DEPRECATED: See PushFrontInEx
func (agent *Agent) PushFrontIn(key []byte, path string, value []byte, flags SubdocFlag, cas Cas, expiry uint32, cb StoreInCallback) (PendingOp, error) {
	return agent.PushFrontInEx(StoreInOptions{
		Key:    key,
		Path:   path,
		Value:  value,
		Flags:  flags,
		Cas:    cas,
		Expiry: expiry,
	}, func(resp *StoreInResult, err error) {
		if resp != nil {
			cb(resp.Cas, resp.MutationToken, err)
			return
		}

		cb(0, MutationToken{}, err)
	})
}

// PushBackIn pushes an entry to the back of an array at a path within a document.
// DEPRECATED: See PushBackInEx
func (agent *Agent) PushBackIn(key []byte, path string, value []byte, flags SubdocFlag, cas Cas, expiry uint32, cb StoreInCallback) (PendingOp, error) {
	return agent.PushBackInEx(StoreInOptions{
		Key:    key,
		Path:   path,
		Value:  value,
		Flags:  flags,
		Cas:    cas,
		Expiry: expiry,
	}, func(resp *StoreInResult, err error) {
		if resp != nil {
			cb(resp.Cas, resp.MutationToken, err)
			return
		}

		cb(0, MutationToken{}, err)
	})
}

// ArrayInsertIn inserts an entry to an array at a path within the document.
// DEPRECATED: See ArrayInsertInEx
func (agent *Agent) ArrayInsertIn(key []byte, path string, value []byte, cas Cas, expiry uint32, flags SubdocFlag, cb StoreInCallback) (PendingOp, error) {
	return agent.ArrayInsertInEx(StoreInOptions{
		Key:    key,
		Path:   path,
		Value:  value,
		Flags:  flags,
		Cas:    cas,
		Expiry: expiry,
	}, func(resp *StoreInResult, err error) {
		if resp != nil {
			cb(resp.Cas, resp.MutationToken, err)
			return
		}

		cb(0, MutationToken{}, err)
	})
}

// AddUniqueIn adds an entry to an array at a path but only if the value doesn't already exist in the array.
// DEPRECATED: See AddUniqueInEx
func (agent *Agent) AddUniqueIn(key []byte, path string, value []byte, flags SubdocFlag, cas Cas, expiry uint32, cb StoreInCallback) (PendingOp, error) {
	return agent.AddUniqueInEx(StoreInOptions{
		Key:    key,
		Path:   path,
		Value:  value,
		Flags:  flags,
		Cas:    cas,
		Expiry: expiry,
	}, func(resp *StoreInResult, err error) {
		if resp != nil {
			cb(resp.Cas, resp.MutationToken, err)
			return
		}

		cb(0, MutationToken{}, err)
	})
}

// CounterInCallback is invoked with the results of `CounterIn` operations.
// DEPRECATED
type CounterInCallback func([]byte, Cas, MutationToken, error)

// CounterIn performs an arithmetic add or subtract on a value at a path in the document.
// DEPRECATED: See CounterInEx
func (agent *Agent) CounterIn(key []byte, path string, value []byte, cas Cas, expiry uint32, flags SubdocFlag, cb CounterInCallback) (PendingOp, error) {
	return agent.CounterInEx(CounterInOptions{
		Key:    key,
		Path:   path,
		Value:  value,
		Flags:  flags,
		Cas:    cas,
		Expiry: expiry,
	}, func(resp *CounterInResult, err error) {
		if resp != nil {
			cb(resp.Value, resp.Cas, resp.MutationToken, err)
			return
		}

		cb(nil, 0, MutationToken{}, err)
	})
}

// RemoveInCallback is invoked with the results of `RemoveIn` operations.
// DEPRECATED
type RemoveInCallback func(Cas, MutationToken, error)

// RemoveIn removes the value at a path within the document.
// DEPRECATED: See DeleteInEx
func (agent *Agent) RemoveIn(key []byte, path string, cas Cas, expiry uint32, flags SubdocFlag, cb RemoveInCallback) (PendingOp, error) {
	return agent.DeleteInEx(DeleteInOptions{
		Key:    key,
		Path:   path,
		Flags:  flags,
		Cas:    cas,
		Expiry: expiry,
	}, func(resp *DeleteInResult, err error) {
		if resp != nil {
			cb(resp.Cas, resp.MutationToken, err)
			return
		}

		cb(0, MutationToken{}, err)
	})
}

// LookupInCallback is invoked with the results of `LookupIn` operations.
// DEPRECATED
type LookupInCallback func([]SubDocResult, Cas, error)

// SubDocLookup performs a multiple-lookup sub-document operation on a document.
// DEPRECATED: See LookupInEx
func (agent *Agent) SubDocLookup(key []byte, ops []SubDocOp, flags SubdocDocFlag, cb LookupInCallback) (PendingOp, error) {
	return agent.LookupInEx(LookupInOptions{
		Key:   key,
		Flags: flags,
		Ops:   ops,
	}, func(resp *LookupInResult, err error) {
		if resp != nil {
			cb(resp.Ops, resp.Cas, err)
			return
		}

		cb(nil, 0, err)
	})
}

// MutateInCallback is invoked with the results of `MutateIn` operations.
// DEPRECATED
type MutateInCallback func([]SubDocResult, Cas, MutationToken, error)

// SubDocMutate performs a multiple-mutation sub-document operation on a document.
// DEPRECATED: See MutateInEx
func (agent *Agent) SubDocMutate(key []byte, ops []SubDocOp, flags SubdocDocFlag, cas Cas, expiry uint32, cb MutateInCallback) (PendingOp, error) {
	return agent.MutateInEx(MutateInOptions{
		Key:    key,
		Flags:  flags,
		Ops:    ops,
		Cas:    cas,
		Expiry: expiry,
	}, func(resp *MutateInResult, err error) {
		if resp != nil {
			cb(resp.Ops, resp.Cas, resp.MutationToken, err)
			return
		}

		cb(nil, 0, MutationToken{}, err)
	})
}
