package gocbcore

import (
	"encoding/binary"
	"regexp"
	"sync"

	"github.com/opentracing/opentracing-go"
)

var vbTakeoverRegex = regexp.MustCompile(`dcp-vbtakeover (\d+$|\d+ \S+)`)

// GetOptions encapsulates the parameters for a GetEx operation.
type GetOptions struct {
	Key          []byte
	TraceContext opentracing.SpanContext
}

// GetResult encapsulates the result of a GetEx operation.
type GetResult struct {
	Value    []byte
	Flags    uint32
	Datatype uint8
	Cas      Cas
}

// GetExCallback is invoked upon completion of a GetEx operation.
type GetExCallback func(*GetResult, error)

// GetEx retrieves a document.
func (agent *Agent) GetEx(opts GetOptions, cb GetExCallback) (PendingOp, error) {
	tracer := agent.createOpTrace("GetEx", opts.TraceContext)

	handler := func(resp *memdQResponse, req *memdQRequest, err error) {
		if err != nil {
			tracer.Finish()
			cb(nil, err)
			return
		}

		if len(resp.Extras) != 4 {
			tracer.Finish()
			cb(nil, ErrProtocol)
			return
		}

		res := GetResult{}
		res.Value = resp.Value
		res.Flags = binary.BigEndian.Uint32(resp.Extras[0:])
		res.Cas = Cas(resp.Cas)
		res.Datatype = resp.Datatype

		tracer.Finish()
		cb(&res, nil)
	}
	req := &memdQRequest{
		memdPacket: memdPacket{
			Magic:    reqMagic,
			Opcode:   cmdGet,
			Datatype: 0,
			Cas:      0,
			Extras:   nil,
			Key:      opts.Key,
			Value:    nil,
		},
		Callback:         handler,
		RootTraceContext: tracer.RootContext(),
	}
	return agent.dispatchOp(req)
}

// GetAndTouchOptions encapsulates the parameters for a GetAndTouchEx operation.
type GetAndTouchOptions struct {
	Key          []byte
	Expiry       uint32
	TraceContext opentracing.SpanContext
}

// GetAndTouchResult encapsulates the result of a GetAndTouchEx operation.
type GetAndTouchResult struct {
	Value    []byte
	Flags    uint32
	Datatype uint8
	Cas      Cas
}

// GetAndTouchExCallback is invoked upon completion of a GetAndTouchEx operation.
type GetAndTouchExCallback func(*GetAndTouchResult, error)

// GetAndTouchEx retrieves a document and updates its expiry.
func (agent *Agent) GetAndTouchEx(opts GetAndTouchOptions, cb GetAndTouchExCallback) (PendingOp, error) {
	tracer := agent.createOpTrace("GetAndTouchEx", opts.TraceContext)

	handler := func(resp *memdQResponse, _ *memdQRequest, err error) {
		if err != nil {
			tracer.Finish()
			cb(nil, err)
			return
		}

		if len(resp.Extras) != 4 {
			tracer.Finish()
			cb(nil, ErrProtocol)
			return
		}

		flags := binary.BigEndian.Uint32(resp.Extras[0:])

		tracer.Finish()
		cb(&GetAndTouchResult{
			Value:    resp.Value,
			Flags:    flags,
			Cas:      Cas(resp.Cas),
			Datatype: resp.Datatype,
		}, nil)
	}

	extraBuf := make([]byte, 4)
	binary.BigEndian.PutUint32(extraBuf[0:], opts.Expiry)

	req := &memdQRequest{
		memdPacket: memdPacket{
			Magic:    reqMagic,
			Opcode:   cmdGAT,
			Datatype: 0,
			Cas:      0,
			Extras:   extraBuf,
			Key:      opts.Key,
			Value:    nil,
		},
		Callback: handler,
	}
	return agent.dispatchOp(req)
}

// GetAndLockOptions encapsulates the parameters for a GetAndLockEx operation.
type GetAndLockOptions struct {
	Key          []byte
	LockTime     uint32
	TraceContext opentracing.SpanContext
}

// GetAndLockResult encapsulates the result of a GetAndLockEx operation.
type GetAndLockResult struct {
	Value    []byte
	Flags    uint32
	Datatype uint8
	Cas      Cas
}

// GetAndLockExCallback is invoked upon completion of a GetAndLockEx operation.
type GetAndLockExCallback func(*GetAndLockResult, error)

// GetAndLockEx retrieves a document and locks it.
func (agent *Agent) GetAndLockEx(opts GetAndLockOptions, cb GetAndLockExCallback) (PendingOp, error) {
	tracer := agent.createOpTrace("GetAndLockEx", opts.TraceContext)

	handler := func(resp *memdQResponse, _ *memdQRequest, err error) {
		if err != nil {
			tracer.Finish()
			cb(nil, err)
			return
		}

		if len(resp.Extras) != 4 {
			tracer.Finish()
			cb(nil, ErrProtocol)
			return
		}

		flags := binary.BigEndian.Uint32(resp.Extras[0:])

		tracer.Finish()
		cb(&GetAndLockResult{
			Value:    resp.Value,
			Flags:    flags,
			Cas:      Cas(resp.Cas),
			Datatype: resp.Datatype,
		}, nil)
	}

	extraBuf := make([]byte, 4)
	binary.BigEndian.PutUint32(extraBuf[0:], opts.LockTime)

	req := &memdQRequest{
		memdPacket: memdPacket{
			Magic:    reqMagic,
			Opcode:   cmdGetLocked,
			Datatype: 0,
			Cas:      0,
			Extras:   extraBuf,
			Key:      opts.Key,
			Value:    nil,
		},
		Callback:         handler,
		RootTraceContext: tracer.RootContext(),
	}
	return agent.dispatchOp(req)
}

// GetReplicaOptions encapsulates the parameters for a GetReplicaEx operation.
type GetReplicaOptions struct {
	Key          []byte
	ReplicaIdx   int
	TraceContext opentracing.SpanContext
}

// GetReplicaResult encapsulates the result of a GetReplicaEx operation.
type GetReplicaResult struct {
	Value    []byte
	Flags    uint32
	Datatype uint8
	Cas      Cas
}

// GetReplicaExCallback is invoked upon completion of a GetReplicaEx operation.
type GetReplicaExCallback func(*GetReplicaResult, error)

func (agent *Agent) getOneReplica(tracer *opTracer, opts GetReplicaOptions, cb GetReplicaExCallback) (PendingOp, error) {
	if opts.ReplicaIdx <= 0 {
		return nil, ErrInvalidReplica
	}

	handler := func(resp *memdQResponse, _ *memdQRequest, err error) {
		if err != nil {
			cb(nil, err)
			return
		}

		if len(resp.Extras) != 4 {
			cb(nil, ErrProtocol)
			return
		}

		flags := binary.BigEndian.Uint32(resp.Extras[0:])

		cb(&GetReplicaResult{
			Value:    resp.Value,
			Flags:    flags,
			Cas:      Cas(resp.Cas),
			Datatype: resp.Datatype,
		}, nil)
	}

	req := &memdQRequest{
		memdPacket: memdPacket{
			Magic:    reqMagic,
			Opcode:   cmdGetReplica,
			Datatype: 0,
			Cas:      0,
			Extras:   nil,
			Key:      opts.Key,
			Value:    nil,
		},
		Callback:         handler,
		ReplicaIdx:       opts.ReplicaIdx,
		RootTraceContext: tracer.RootContext(),
	}
	return agent.dispatchOp(req)
}

// GetReplicaEx retrieves a document from a replica server.
func (agent *Agent) GetReplicaEx(opts GetReplicaOptions, cb GetReplicaExCallback) (PendingOp, error) {
	tracer := agent.createOpTrace("GetReplicaEx", opts.TraceContext)

	if opts.ReplicaIdx > 0 {
		return agent.getOneReplica(tracer, opts, func(resp *GetReplicaResult, err error) {
			tracer.Finish()
			cb(resp, err)
		})
	}

	if opts.ReplicaIdx < 0 {
		tracer.Finish()
		return nil, ErrInvalidReplica
	}

	numReplicas := agent.NumReplicas()

	if numReplicas == 0 {
		tracer.Finish()
		return nil, ErrInvalidReplica
	}

	var resultLock sync.Mutex
	var firstResult *GetReplicaResult

	op := new(multiPendingOp)
	expected := uint32(numReplicas)

	opHandledLocked := func() {
		completed := op.IncrementCompletedOps()
		if expected-completed == 0 {
			if firstResult == nil {
				tracer.Finish()
				cb(nil, ErrNoReplicas)
				return
			}

			tracer.Finish()
			cb(firstResult, nil)
		}
	}

	handler := func(resp *GetReplicaResult, err error) {
		resultLock.Lock()

		if err != nil {
			opHandledLocked()
			return
		}

		if firstResult == nil {
			newReplica := *resp
			firstResult = &newReplica
		}

		// Mark this op as completed
		opHandledLocked()

		// Try to cancel every other operation so we can
		// return as soon as possible to the user (and close
		// any open tracing spans)
		for _, op := range op.ops {
			if op.Cancel() {
				opHandledLocked()
			}
		}

		resultLock.Unlock()
	}

	// Dispatch a getReplica for each replica server
	for repIdx := 1; repIdx <= numReplicas; repIdx++ {
		subOp, err := agent.getOneReplica(tracer, GetReplicaOptions{
			Key:        opts.Key,
			ReplicaIdx: repIdx,
		}, handler)

		resultLock.Lock()

		if err != nil {
			opHandledLocked()
			resultLock.Unlock()
			continue
		}

		op.ops = append(op.ops, subOp)
		resultLock.Unlock()
	}

	return op, nil
}

// TouchOptions encapsulates the parameters for a TouchEx operation.
type TouchOptions struct {
	Key          []byte
	Cas          Cas
	Expiry       uint32
	TraceContext opentracing.SpanContext
}

// TouchResult encapsulates the result of a TouchEx operation.
type TouchResult struct {
	Cas           Cas
	MutationToken MutationToken
}

// TouchExCallback is invoked upon completion of a TouchEx operation.
type TouchExCallback func(*TouchResult, error)

// TouchEx updates the expiry for a document.
func (agent *Agent) TouchEx(opts TouchOptions, cb TouchExCallback) (PendingOp, error) {
	tracer := agent.createOpTrace("TouchEx", opts.TraceContext)
	if opts.Cas != 0 {
		tracer.Finish()
		return nil, ErrNonZeroCas
	}

	handler := func(resp *memdQResponse, req *memdQRequest, err error) {
		if err != nil {
			tracer.Finish()
			cb(nil, err)
			return
		}

		mutToken := MutationToken{}
		if len(resp.Extras) >= 16 {
			mutToken.VbId = req.Vbucket
			mutToken.VbUuid = VbUuid(binary.BigEndian.Uint64(resp.Extras[0:]))
			mutToken.SeqNo = SeqNo(binary.BigEndian.Uint64(resp.Extras[8:]))
		}

		tracer.Finish()
		cb(&TouchResult{
			Cas:           Cas(resp.Cas),
			MutationToken: mutToken,
		}, nil)
	}

	extraBuf := make([]byte, 4)
	binary.BigEndian.PutUint32(extraBuf[0:], opts.Expiry)

	req := &memdQRequest{
		memdPacket: memdPacket{
			Magic:    reqMagic,
			Opcode:   cmdTouch,
			Datatype: 0,
			Cas:      0,
			Extras:   extraBuf,
			Key:      opts.Key,
			Value:    nil,
		},
		Callback:         handler,
		RootTraceContext: tracer.RootContext(),
	}
	return agent.dispatchOp(req)
}

// UnlockOptions encapsulates the parameters for a UnlockEx operation.
type UnlockOptions struct {
	Key          []byte
	Cas          Cas
	TraceContext opentracing.SpanContext
}

// UnlockResult encapsulates the result of a UnlockEx operation.
type UnlockResult struct {
	Cas           Cas
	MutationToken MutationToken
}

// UnlockExCallback is invoked upon completion of a UnlockEx operation.
type UnlockExCallback func(*UnlockResult, error)

// UnlockEx unlocks a locked document.
func (agent *Agent) UnlockEx(opts UnlockOptions, cb UnlockExCallback) (PendingOp, error) {
	tracer := agent.createOpTrace("UnlockEx", opts.TraceContext)

	handler := func(resp *memdQResponse, req *memdQRequest, err error) {
		if err != nil {
			tracer.Finish()
			cb(nil, err)
			return
		}

		mutToken := MutationToken{}
		if len(resp.Extras) >= 16 {
			mutToken.VbId = req.Vbucket
			mutToken.VbUuid = VbUuid(binary.BigEndian.Uint64(resp.Extras[0:]))
			mutToken.SeqNo = SeqNo(binary.BigEndian.Uint64(resp.Extras[8:]))
		}

		tracer.Finish()
		cb(&UnlockResult{
			Cas:           Cas(resp.Cas),
			MutationToken: mutToken,
		}, nil)
	}

	req := &memdQRequest{
		memdPacket: memdPacket{
			Magic:    reqMagic,
			Opcode:   cmdUnlockKey,
			Datatype: 0,
			Cas:      uint64(opts.Cas),
			Extras:   nil,
			Key:      opts.Key,
			Value:    nil,
		},
		Callback:         handler,
		RootTraceContext: tracer.RootContext(),
	}
	return agent.dispatchOp(req)
}

// DeleteOptions encapsulates the parameters for a DeleteEx operation.
type DeleteOptions struct {
	Key          []byte
	Cas          Cas
	TraceContext opentracing.SpanContext
}

// DeleteResult encapsulates the result of a DeleteEx operation.
type DeleteResult struct {
	Cas           Cas
	MutationToken MutationToken
}

// DeleteExCallback is invoked upon completion of a DeleteEx operation.
type DeleteExCallback func(*DeleteResult, error)

// DeleteEx removes a document.
func (agent *Agent) DeleteEx(opts DeleteOptions, cb DeleteExCallback) (PendingOp, error) {
	tracer := agent.createOpTrace("DeleteEx", opts.TraceContext)

	handler := func(resp *memdQResponse, req *memdQRequest, err error) {
		if err != nil {
			tracer.Finish()
			cb(nil, err)
			return
		}

		mutToken := MutationToken{}
		if len(resp.Extras) >= 16 {
			mutToken.VbId = req.Vbucket
			mutToken.VbUuid = VbUuid(binary.BigEndian.Uint64(resp.Extras[0:]))
			mutToken.SeqNo = SeqNo(binary.BigEndian.Uint64(resp.Extras[8:]))
		}

		tracer.Finish()
		cb(&DeleteResult{
			Cas:           Cas(resp.Cas),
			MutationToken: mutToken,
		}, nil)
	}

	req := &memdQRequest{
		memdPacket: memdPacket{
			Magic:    reqMagic,
			Opcode:   cmdDelete,
			Datatype: 0,
			Cas:      uint64(opts.Cas),
			Extras:   nil,
			Key:      opts.Key,
			Value:    nil,
		},
		Callback:         handler,
		RootTraceContext: tracer.RootContext(),
	}
	return agent.dispatchOp(req)
}

type storeOptions struct {
	Key          []byte
	Value        []byte
	Flags        uint32
	Datatype     uint8
	Cas          Cas
	Expiry       uint32
	TraceContext opentracing.SpanContext
}

// StoreResult encapsulates the result of a AddEx, SetEx or ReplaceEx operation.
type StoreResult struct {
	Cas           Cas
	MutationToken MutationToken
}

// StoreExCallback is invoked upon completion of a AddEx, SetEx or ReplaceEx operation.
type StoreExCallback func(*StoreResult, error)

func (agent *Agent) storeEx(opName string, opcode commandCode, opts storeOptions, cb StoreExCallback) (PendingOp, error) {
	tracer := agent.createOpTrace(opName, opts.TraceContext)

	handler := func(resp *memdQResponse, req *memdQRequest, err error) {
		if err != nil {
			tracer.Finish()
			cb(nil, err)
			return
		}

		mutToken := MutationToken{}
		if len(resp.Extras) >= 16 {
			mutToken.VbId = req.Vbucket
			mutToken.VbUuid = VbUuid(binary.BigEndian.Uint64(resp.Extras[0:]))
			mutToken.SeqNo = SeqNo(binary.BigEndian.Uint64(resp.Extras[8:]))
		}

		tracer.Finish()
		cb(&StoreResult{
			Cas:           Cas(resp.Cas),
			MutationToken: mutToken,
		}, nil)
	}

	extraBuf := make([]byte, 8)
	binary.BigEndian.PutUint32(extraBuf[0:], opts.Flags)
	binary.BigEndian.PutUint32(extraBuf[4:], opts.Expiry)
	req := &memdQRequest{
		memdPacket: memdPacket{
			Magic:    reqMagic,
			Opcode:   opcode,
			Datatype: opts.Datatype,
			Cas:      uint64(opts.Cas),
			Extras:   extraBuf,
			Key:      opts.Key,
			Value:    opts.Value,
		},
		Callback:         handler,
		RootTraceContext: tracer.RootContext(),
	}
	return agent.dispatchOp(req)
}

// AddOptions encapsulates the parameters for a AddEx operation.
type AddOptions struct {
	Key          []byte
	Value        []byte
	Flags        uint32
	Datatype     uint8
	Expiry       uint32
	TraceContext opentracing.SpanContext
}

// AddEx stores a document as long as it does not already exist.
func (agent *Agent) AddEx(opts AddOptions, cb StoreExCallback) (PendingOp, error) {
	return agent.storeEx("AddEx", cmdAdd, storeOptions{
		Key:          opts.Key,
		Value:        opts.Value,
		Flags:        opts.Flags,
		Datatype:     opts.Datatype,
		Cas:          0,
		Expiry:       opts.Expiry,
		TraceContext: opts.TraceContext,
	}, cb)
}

// SetOptions encapsulates the parameters for a SetEx operation.
type SetOptions struct {
	Key          []byte
	Value        []byte
	Flags        uint32
	Datatype     uint8
	Expiry       uint32
	TraceContext opentracing.SpanContext
}

// SetEx stores a document.
func (agent *Agent) SetEx(opts SetOptions, cb StoreExCallback) (PendingOp, error) {
	return agent.storeEx("SetEx", cmdSet, storeOptions{
		Key:          opts.Key,
		Value:        opts.Value,
		Flags:        opts.Flags,
		Datatype:     opts.Datatype,
		Cas:          0,
		Expiry:       opts.Expiry,
		TraceContext: opts.TraceContext,
	}, cb)
}

// ReplaceOptions encapsulates the parameters for a ReplaceEx operation.
type ReplaceOptions struct {
	Key          []byte
	Value        []byte
	Flags        uint32
	Datatype     uint8
	Cas          Cas
	Expiry       uint32
	TraceContext opentracing.SpanContext
}

// ReplaceEx replaces the value of a Couchbase document with another value.
func (agent *Agent) ReplaceEx(opts ReplaceOptions, cb StoreExCallback) (PendingOp, error) {
	return agent.storeEx("ReplaceEx", cmdSet, storeOptions{
		Key:          opts.Key,
		Value:        opts.Value,
		Flags:        opts.Flags,
		Datatype:     opts.Datatype,
		Cas:          opts.Cas,
		Expiry:       opts.Expiry,
		TraceContext: opts.TraceContext,
	}, cb)
}

// AdjoinOptions encapsulates the parameters for a AppendEx or PrependEx operation.
type AdjoinOptions struct {
	Key          []byte
	Value        []byte
	TraceContext opentracing.SpanContext
}

// AdjoinResult encapsulates the result of a AppendEx or PrependEx operation.
type AdjoinResult struct {
	Cas           Cas
	MutationToken MutationToken
}

// AdjoinExCallback is invoked upon completion of a AppendEx or PrependEx operation.
type AdjoinExCallback func(*AdjoinResult, error)

func (agent *Agent) adjoinEx(opName string, opcode commandCode, opts AdjoinOptions, cb AdjoinExCallback) (PendingOp, error) {
	tracer := agent.createOpTrace(opName, opts.TraceContext)

	handler := func(resp *memdQResponse, req *memdQRequest, err error) {
		if err != nil {
			tracer.Finish()
			cb(nil, err)
			return
		}

		mutToken := MutationToken{}
		if len(resp.Extras) >= 16 {
			mutToken.VbId = req.Vbucket
			mutToken.VbUuid = VbUuid(binary.BigEndian.Uint64(resp.Extras[0:]))
			mutToken.SeqNo = SeqNo(binary.BigEndian.Uint64(resp.Extras[8:]))
		}

		tracer.Finish()
		cb(&AdjoinResult{
			Cas:           Cas(resp.Cas),
			MutationToken: mutToken,
		}, nil)
	}

	req := &memdQRequest{
		memdPacket: memdPacket{
			Magic:    reqMagic,
			Opcode:   opcode,
			Datatype: 0,
			Cas:      0,
			Extras:   nil,
			Key:      opts.Key,
			Value:    opts.Value,
		},
		Callback:         handler,
		RootTraceContext: tracer.RootContext(),
	}
	return agent.dispatchOp(req)
}

// AppendEx appends some bytes to a document.
func (agent *Agent) AppendEx(opts AdjoinOptions, cb AdjoinExCallback) (PendingOp, error) {
	return agent.adjoinEx("AppendEx", cmdAppend, opts, cb)
}

// PrependOptions encapsulates the parameters for a ReplaceEx operation.
type PrependOptions struct {
	Key          []byte
	Value        []byte
	TraceContext opentracing.SpanContext
}

// PrependEx prepends some bytes to a document.
func (agent *Agent) PrependEx(opts AdjoinOptions, cb AdjoinExCallback) (PendingOp, error) {
	return agent.adjoinEx("PrependEx", cmdPrepend, opts, cb)
}

// CounterOptions encapsulates the parameters for a IncrementEx or DecrementEx operation.
type CounterOptions struct {
	Key          []byte
	Delta        uint64
	Initial      uint64
	Expiry       uint32
	TraceContext opentracing.SpanContext
}

// CounterResult encapsulates the result of a IncrementEx or DecrementEx operation.
type CounterResult struct {
	Value         uint64
	Cas           Cas
	MutationToken MutationToken
}

// CounterExCallback is invoked upon completion of a IncrementEx or DecrementEx operation.
type CounterExCallback func(*CounterResult, error)

func (agent *Agent) counterEx(opName string, opcode commandCode, opts CounterOptions, cb CounterExCallback) (PendingOp, error) {
	tracer := agent.createOpTrace(opName, opts.TraceContext)

	handler := func(resp *memdQResponse, req *memdQRequest, err error) {
		if err != nil {
			tracer.Finish()
			cb(nil, err)
			return
		}

		if len(resp.Value) != 8 {
			tracer.Finish()
			cb(nil, ErrProtocol)
			return
		}
		intVal := binary.BigEndian.Uint64(resp.Value)

		mutToken := MutationToken{}
		if len(resp.Extras) >= 16 {
			mutToken.VbId = req.Vbucket
			mutToken.VbUuid = VbUuid(binary.BigEndian.Uint64(resp.Extras[0:]))
			mutToken.SeqNo = SeqNo(binary.BigEndian.Uint64(resp.Extras[8:]))
		}

		tracer.Finish()
		cb(&CounterResult{
			Value:         intVal,
			Cas:           Cas(resp.Cas),
			MutationToken: mutToken,
		}, nil)
	}

	// You cannot have an expiry when you do not want to create the document.
	if opts.Initial == uint64(0xFFFFFFFFFFFFFFFF) && opts.Expiry != 0 {
		return nil, ErrInvalidArgs
	}

	extraBuf := make([]byte, 20)
	binary.BigEndian.PutUint64(extraBuf[0:], opts.Delta)
	if opts.Initial != uint64(0xFFFFFFFFFFFFFFFF) {
		binary.BigEndian.PutUint64(extraBuf[8:], opts.Initial)
		binary.BigEndian.PutUint32(extraBuf[16:], opts.Expiry)
	} else {
		binary.BigEndian.PutUint64(extraBuf[8:], 0x0000000000000000)
		binary.BigEndian.PutUint32(extraBuf[16:], 0xFFFFFFFF)
	}

	req := &memdQRequest{
		memdPacket: memdPacket{
			Magic:    reqMagic,
			Opcode:   opcode,
			Datatype: 0,
			Cas:      0,
			Extras:   extraBuf,
			Key:      opts.Key,
			Value:    nil,
		},
		Callback:         handler,
		RootTraceContext: tracer.RootContext(),
	}
	return agent.dispatchOp(req)
}

// IncrementEx increments the unsigned integer value in a document.
func (agent *Agent) IncrementEx(opts CounterOptions, cb CounterExCallback) (PendingOp, error) {
	return agent.counterEx("IncrementEx", cmdIncrement, opts, cb)
}

// DecrementEx decrements the unsigned integer value in a document.
func (agent *Agent) DecrementEx(opts CounterOptions, cb CounterExCallback) (PendingOp, error) {
	return agent.counterEx("DecrementEx", cmdDecrement, opts, cb)
}

// GetRandomOptions encapsulates the parameters for a GetRandomEx operation.
type GetRandomOptions struct {
	TraceContext opentracing.SpanContext
}

// GetRandomResult encapsulates the result of a GetRandomEx operation.
type GetRandomResult struct {
	Key      []byte
	Value    []byte
	Flags    uint32
	Datatype uint8
	Cas      Cas
}

// GetRandomExCallback is invoked upon completion of a GetRandomEx operation.
type GetRandomExCallback func(*GetRandomResult, error)

// GetRandomEx retrieves the key and value of a random document stored within Couchbase Server.
func (agent *Agent) GetRandomEx(opts GetRandomOptions, cb GetRandomExCallback) (PendingOp, error) {
	tracer := agent.createOpTrace("GetRandomEx", opts.TraceContext)

	handler := func(resp *memdQResponse, _ *memdQRequest, err error) {
		if err != nil {
			tracer.Finish()
			cb(nil, err)
			return
		}

		if len(resp.Extras) != 4 {
			tracer.Finish()
			cb(nil, ErrProtocol)
			return
		}

		flags := binary.BigEndian.Uint32(resp.Extras[0:])

		tracer.Finish()
		cb(&GetRandomResult{
			Key:      resp.Key,
			Value:    resp.Value,
			Flags:    flags,
			Cas:      Cas(resp.Cas),
			Datatype: resp.Datatype,
		}, nil)
	}
	req := &memdQRequest{
		memdPacket: memdPacket{
			Magic:    reqMagic,
			Opcode:   cmdGetRandom,
			Datatype: 0,
			Cas:      0,
			Extras:   nil,
			Key:      nil,
			Value:    nil,
		},
		Callback:         handler,
		RootTraceContext: tracer.RootContext(),
	}
	return agent.dispatchOp(req)
}

// SingleServerStats represents the stats returned from a single server.
type SingleServerStats struct {
	Stats map[string]string
	Error error
}

// StatsTarget is used for providing a specific target to the StatsEx operation.
type StatsTarget interface {
}

// VBucketIdStatsTarget indicates that a specific vbucket should be targeted by the StatsEx operation.
type VBucketIdStatsTarget struct {
	Vbid uint16
}

// StatsOptions encapsulates the parameters for a StatsEx operation.
type StatsOptions struct {
	Key string
	// Target indicates that something specific should be targeted by the operation. If left nil
	// then the stats command will be sent to all servers.
	Target       StatsTarget
	TraceContext opentracing.SpanContext
}

// StatsResult encapsulates the result of a StatsEx operation.
type StatsResult struct {
	Servers map[string]SingleServerStats
}

// StatsExCallback is invoked upon completion of a StatsEx operation.
type StatsExCallback func(*StatsResult, error)

// StatsEx retrieves statistics information from the server.  Note that as this
// function is an aggregator across numerous servers, there are no guarantees
// about the consistency of the results.  Occasionally, some nodes may not be
// represented in the results, or there may be conflicting information between
// multiple nodes (a vbucket active on two separate nodes at once).
func (agent *Agent) StatsEx(opts StatsOptions, cb StatsExCallback) (PendingOp, error) {
	tracer := agent.createOpTrace("StatsEx", opts.TraceContext)

	config := agent.routingInfo.Get()
	if config == nil {
		tracer.Finish()
		return nil, ErrShutdown
	}

	stats := make(map[string]SingleServerStats)
	var statsLock sync.Mutex

	op := new(multiPendingOp)
	var expected uint32

	pipelines := make([]*memdPipeline, 0)

	switch target := opts.Target.(type) {
	case nil:
		expected = uint32(config.clientMux.NumPipelines())

		for i := 0; i < config.clientMux.NumPipelines(); i++ {
			pipelines = append(pipelines, config.clientMux.GetPipeline(i))
		}
	case VBucketIdStatsTarget:
		expected = 1

		srvIdx, err := config.vbMap.NodeByVbucket(target.Vbid, 0)
		if err != nil {
			return nil, err
		}

		pipelines = append(pipelines, config.clientMux.GetPipeline(srvIdx))
	default:
		return nil, ErrUnsupportedStatsTarget
	}

	opHandledLocked := func() {
		completed := op.IncrementCompletedOps()
		if expected-completed == 0 {
			tracer.Finish()
			cb(&StatsResult{
				Servers: stats,
			}, nil)
		}
	}

	handler := func(resp *memdQResponse, req *memdQRequest, err error) {
		serverAddress := resp.sourceAddr

		statsLock.Lock()
		defer statsLock.Unlock()

		// Fetch the specific stats key for this server.  Creating a new entry
		// for the server if we did not previously have one.
		curStats, ok := stats[serverAddress]
		if !ok {
			stats[serverAddress] = SingleServerStats{
				Stats: make(map[string]string),
			}
			curStats = stats[serverAddress]
		}

		if err != nil {
			// Store the first (and hopefully only) error into the Error field of this
			// server's stats entry.
			if curStats.Error == nil {
				curStats.Error = err
			} else {
				logDebugf("Got additional error for stats: %s: %v", serverAddress, err)
			}

			// When an error occurs, we need to cancel our persistent op.  However, because
			// a previous error may already have cancelled this and then raced, we should
			// ensure only a single completion is counted.
			if req.Cancel() {
				opHandledLocked()
			}

			return
		}

		// Check if the key length is zero.  This indicates that we have reached
		// the ending of the stats listing by this server.
		if len(resp.Key) == 0 {
			// As this is a persistent request, we must manually cancel it to remove
			// it from the pending ops list.  To ensure we do not race multiple cancels,
			// we only handle it as completed the one time cancellation succeeds.
			if req.Cancel() {
				opHandledLocked()
			}

			return
		}

		// Add the stat for this server to the list of stats.
		curStats.Stats[string(resp.Key)] = string(resp.Value)
	}

	for _, pipeline := range pipelines {
		serverAddress := pipeline.Address()
		req := &memdQRequest{
			memdPacket: memdPacket{
				Magic:    reqMagic,
				Opcode:   cmdStat,
				Datatype: 0,
				Cas:      0,
				Key:      []byte(opts.Key),
				Value:    nil,
			},
			Persistent:       true,
			Callback:         handler,
			RootTraceContext: tracer.RootContext(),
		}

		curOp, err := agent.dispatchOpToAddress(req, serverAddress)
		if err != nil {
			statsLock.Lock()
			stats[serverAddress] = SingleServerStats{
				Error: err,
			}
			opHandledLocked()
			statsLock.Unlock()

			continue
		}

		op.ops = append(op.ops, curOp)
	}

	return op, nil
}
