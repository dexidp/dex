package gocbcore

import (
	"encoding/binary"

	"github.com/opentracing/opentracing-go"
)

// SubDocResult encapsulates the results from a single sub-document operation.
type SubDocResult struct {
	Err   error
	Value []byte
}

// GetInOptions encapsulates the parameters for a GetInEx operation.
type GetInOptions struct {
	Key          []byte
	Path         string
	Flags        SubdocFlag
	TraceContext opentracing.SpanContext
}

// GetInResult encapsulates the result of a GetInEx operation.
type GetInResult struct {
	Value []byte
	Cas   Cas
}

// GetInExCallback is invoked upon completion of a GetInEx operation.
type GetInExCallback func(*GetInResult, error)

// GetInEx retrieves the value at a particular path within a JSON document.
func (agent *Agent) GetInEx(opts GetInOptions, cb GetInExCallback) (PendingOp, error) {
	tracer := agent.createOpTrace("GetInEx", nil)

	handler := func(resp *memdQResponse, _ *memdQRequest, err error) {
		if err != nil {
			tracer.Finish()
			cb(nil, err)
			return
		}

		tracer.Finish()
		cb(&GetInResult{
			Value: resp.Value,
			Cas:   Cas(resp.Cas),
		}, nil)
	}

	pathBytes := []byte(opts.Path)

	extraBuf := make([]byte, 3)
	binary.BigEndian.PutUint16(extraBuf[0:], uint16(len(pathBytes)))
	extraBuf[2] = uint8(opts.Flags)

	req := &memdQRequest{
		memdPacket: memdPacket{
			Magic:    reqMagic,
			Opcode:   cmdSubDocGet,
			Datatype: 0,
			Cas:      0,
			Extras:   extraBuf,
			Key:      opts.Key,
			Value:    pathBytes,
		},
		Callback:         handler,
		RootTraceContext: tracer.RootContext(),
	}
	return agent.dispatchOp(req)
}

// ExistsInOptions encapsulates the parameters for a ExistsInEx operation.
type ExistsInOptions struct {
	Key          []byte
	Path         string
	Flags        SubdocFlag
	TraceContext opentracing.SpanContext
}

// ExistsInResult encapsulates the result of a ExistsInEx operation.
type ExistsInResult struct {
	Cas Cas
}

// ExistsInExCallback is invoked upon completion of a ExistsInEx operation.
type ExistsInExCallback func(*ExistsInResult, error)

// ExistsInEx returns whether a particular path exists within a document.
func (agent *Agent) ExistsInEx(opts ExistsInOptions, cb ExistsInExCallback) (PendingOp, error) {
	tracer := agent.createOpTrace("ExistsInEx", nil)

	handler := func(resp *memdQResponse, _ *memdQRequest, err error) {
		if err != nil {
			tracer.Finish()
			cb(nil, err)
			return
		}

		tracer.Finish()
		cb(&ExistsInResult{
			Cas: Cas(resp.Cas),
		}, nil)
	}

	pathBytes := []byte(opts.Path)

	extraBuf := make([]byte, 3)
	binary.BigEndian.PutUint16(extraBuf[0:], uint16(len(pathBytes)))
	extraBuf[2] = uint8(opts.Flags)

	req := &memdQRequest{
		memdPacket: memdPacket{
			Magic:    reqMagic,
			Opcode:   cmdSubDocExists,
			Datatype: 0,
			Cas:      0,
			Extras:   extraBuf,
			Key:      opts.Key,
			Value:    pathBytes,
		},
		Callback:         handler,
		RootTraceContext: tracer.RootContext(),
	}
	return agent.dispatchOp(req)
}

// StoreInOptions encapsulates the parameters for a SetInEx, AddInEx, ReplaceInEx,
// PushFrontInEx, PushBackInEx, ArrayInsertInEx or AddUniqueInEx operation.
type StoreInOptions struct {
	Key          []byte
	Path         string
	Value        []byte
	Flags        SubdocFlag
	Cas          Cas
	Expiry       uint32
	TraceContext opentracing.SpanContext
}

// StoreInResult encapsulates the result of a SetInEx, AddInEx, ReplaceInEx,
// PushFrontInEx, PushBackInEx, ArrayInsertInEx or AddUniqueInEx operation.
type StoreInResult struct {
	Cas           Cas
	MutationToken MutationToken
}

// StoreInExCallback is invoked upon completion of a SetInEx, AddInEx,
// ReplaceInEx, PushFrontInEx, PushBackInEx, ArrayInsertInEx or
// AddUniqueInEx operation.
type StoreInExCallback func(*StoreInResult, error)

func (agent *Agent) storeInEx(opName string, opcode commandCode, opts StoreInOptions, cb StoreInExCallback) (PendingOp, error) {
	tracer := agent.createOpTrace(opName, nil)

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
		cb(&StoreInResult{
			Cas:           Cas(resp.Cas),
			MutationToken: mutToken,
		}, nil)
	}

	pathBytes := []byte(opts.Path)

	valueBuf := make([]byte, len(pathBytes)+len(opts.Value))
	copy(valueBuf[0:], pathBytes)
	copy(valueBuf[len(pathBytes):], opts.Value)

	var extraBuf []byte
	if opts.Expiry != 0 {
		extraBuf = make([]byte, 7)
	} else {
		extraBuf = make([]byte, 3)
	}
	binary.BigEndian.PutUint16(extraBuf[0:], uint16(len(pathBytes)))
	extraBuf[2] = uint8(opts.Flags)
	if len(extraBuf) >= 7 {
		binary.BigEndian.PutUint32(extraBuf[3:], opts.Expiry)
	}

	req := &memdQRequest{
		memdPacket: memdPacket{
			Magic:    reqMagic,
			Opcode:   opcode,
			Datatype: 0,
			Cas:      uint64(opts.Cas),
			Extras:   extraBuf,
			Key:      opts.Key,
			Value:    valueBuf,
		},
		Callback:         handler,
		RootTraceContext: tracer.RootContext(),
	}
	return agent.dispatchOp(req)
}

// SetInEx sets the value at a path within a document.
func (agent *Agent) SetInEx(opts StoreInOptions, cb StoreInExCallback) (PendingOp, error) {
	return agent.storeInEx("SetInEx", cmdSubDocDictSet, opts, cb)
}

// AddInEx adds a value at the path within a document.  This method
// works like SetIn, but only only succeeds if the path does not
// currently exist.
func (agent *Agent) AddInEx(opts StoreInOptions, cb StoreInExCallback) (PendingOp, error) {
	return agent.storeInEx("AddInEx", cmdSubDocDictAdd, opts, cb)
}

// ReplaceInEx replaces the value at the path within a document.
// This method works like SetIn, but only only succeeds
// if the path currently exists.
func (agent *Agent) ReplaceInEx(opts StoreInOptions, cb StoreInExCallback) (PendingOp, error) {
	return agent.storeInEx("ReplaceInEx", cmdSubDocReplace, opts, cb)
}

// PushFrontInEx pushes an entry to the front of an array at a path within a document.
func (agent *Agent) PushFrontInEx(opts StoreInOptions, cb StoreInExCallback) (PendingOp, error) {
	return agent.storeInEx("PushFrontInEx", cmdSubDocArrayPushFirst, opts, cb)
}

// PushBackInEx pushes an entry to the back of an array at a path within a document.
func (agent *Agent) PushBackInEx(opts StoreInOptions, cb StoreInExCallback) (PendingOp, error) {
	return agent.storeInEx("PushBackInEx", cmdSubDocArrayPushLast, opts, cb)
}

// ArrayInsertInEx inserts an entry to an array at a path within the document.
func (agent *Agent) ArrayInsertInEx(opts StoreInOptions, cb StoreInExCallback) (PendingOp, error) {
	return agent.storeInEx("ArrayInsertInEx", cmdSubDocArrayInsert, opts, cb)
}

// AddUniqueInEx adds an entry to an array at a path but only if the value doesn't already exist in the array.
func (agent *Agent) AddUniqueInEx(opts StoreInOptions, cb StoreInExCallback) (PendingOp, error) {
	return agent.storeInEx("AddUniqueInEx", cmdSubDocArrayAddUnique, opts, cb)
}

// CounterInOptions encapsulates the parameters for a CounterInEx operation.
type CounterInOptions StoreInOptions

// CounterInResult encapsulates the result of a CounterInEx operation.
type CounterInResult struct {
	Value         []byte
	Cas           Cas
	MutationToken MutationToken
}

// CounterInExCallback is invoked upon completion of a CounterInEx operation.
type CounterInExCallback func(*CounterInResult, error)

// CounterInEx performs an arithmetic add or subtract on a value at a path in the document.
func (agent *Agent) CounterInEx(opts CounterInOptions, cb CounterInExCallback) (PendingOp, error) {
	tracer := agent.createOpTrace("CounterInEx", nil)

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
		cb(&CounterInResult{
			Value:         resp.Value,
			Cas:           Cas(resp.Cas),
			MutationToken: mutToken,
		}, nil)
	}

	pathBytes := []byte(opts.Path)

	valueBuf := make([]byte, len(pathBytes)+len(opts.Value))
	copy(valueBuf[0:], pathBytes)
	copy(valueBuf[len(pathBytes):], opts.Value)

	var extraBuf []byte
	if opts.Expiry != 0 {
		extraBuf = make([]byte, 7)
	} else {
		extraBuf = make([]byte, 3)
	}
	binary.BigEndian.PutUint16(extraBuf[0:], uint16(len(pathBytes)))
	extraBuf[2] = uint8(opts.Flags)
	if len(extraBuf) >= 7 {
		binary.BigEndian.PutUint32(extraBuf[3:], opts.Expiry)
	}

	req := &memdQRequest{
		memdPacket: memdPacket{
			Magic:    reqMagic,
			Opcode:   cmdSubDocCounter,
			Datatype: 0,
			Cas:      uint64(opts.Cas),
			Extras:   extraBuf,
			Key:      opts.Key,
			Value:    valueBuf,
		},
		Callback:         handler,
		RootTraceContext: tracer.RootContext(),
	}
	return agent.dispatchOp(req)
}

// DeleteInOptions encapsulates the parameters for a DeleteInEx operation.
type DeleteInOptions struct {
	Key          []byte
	Path         string
	Cas          Cas
	Expiry       uint32
	Flags        SubdocFlag
	TraceContext opentracing.SpanContext
}

// DeleteInResult encapsulates the result of a DeleteInEx operation.
type DeleteInResult struct {
	Cas           Cas
	MutationToken MutationToken
}

// DeleteInExCallback is invoked upon completion of a DeleteInEx operation.
type DeleteInExCallback func(*DeleteInResult, error)

// DeleteInEx removes the value at a path within the document.
func (agent *Agent) DeleteInEx(opts DeleteInOptions, cb DeleteInExCallback) (PendingOp, error) {
	tracer := agent.createOpTrace("DeleteInEx", nil)

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
		cb(&DeleteInResult{
			Cas:           Cas(resp.Cas),
			MutationToken: mutToken,
		}, nil)
	}

	pathBytes := []byte(opts.Path)

	var extraBuf []byte
	if opts.Expiry != 0 {
		extraBuf = make([]byte, 7)
	} else {
		extraBuf = make([]byte, 3)
	}
	binary.BigEndian.PutUint16(extraBuf[0:], uint16(len(pathBytes)))
	extraBuf[2] = uint8(opts.Flags)
	if len(extraBuf) >= 7 {
		binary.BigEndian.PutUint32(extraBuf[3:], opts.Expiry)
	}

	req := &memdQRequest{
		memdPacket: memdPacket{
			Magic:    reqMagic,
			Opcode:   cmdSubDocDelete,
			Datatype: 0,
			Cas:      uint64(opts.Cas),
			Extras:   extraBuf,
			Key:      opts.Key,
			Value:    pathBytes,
		},
		Callback:         handler,
		RootTraceContext: tracer.RootContext(),
	}
	return agent.dispatchOp(req)
}

// SubDocOp defines a per-operation structure to be passed to MutateIn
// or LookupIn for performing many sub-document operations.
type SubDocOp struct {
	Op    SubDocOpType
	Flags SubdocFlag
	Path  string
	Value []byte
}

// LookupInOptions encapsulates the parameters for a LookupInEx operation.
type LookupInOptions struct {
	Key          []byte
	Flags        SubdocDocFlag
	Ops          []SubDocOp
	TraceContext opentracing.SpanContext
}

// LookupInResult encapsulates the result of a LookupInEx operation.
type LookupInResult struct {
	Cas Cas
	Ops []SubDocResult
}

// LookupInExCallback is invoked upon completion of a LookupInEx operation.
type LookupInExCallback func(*LookupInResult, error)

// LookupInEx performs a multiple-lookup sub-document operation on a document.
func (agent *Agent) LookupInEx(opts LookupInOptions, cb LookupInExCallback) (PendingOp, error) {
	tracer := agent.createOpTrace("LookupInEx", opts.TraceContext)

	results := make([]SubDocResult, len(opts.Ops))

	handler := func(resp *memdQResponse, _ *memdQRequest, err error) {
		if err != nil &&
			!IsErrorStatus(err, StatusSubDocMultiPathFailureDeleted) &&
			!IsErrorStatus(err, StatusSubDocSuccessDeleted) &&
			!IsErrorStatus(err, StatusSubDocBadMulti) {
			tracer.Finish()
			cb(nil, err)
			return
		}

		respIter := 0
		for i := range results {
			if respIter+6 > len(resp.Value) {
				tracer.Finish()
				cb(nil, ErrProtocol)
				return
			}

			resError := StatusCode(binary.BigEndian.Uint16(resp.Value[respIter+0:]))
			resValueLen := int(binary.BigEndian.Uint32(resp.Value[respIter+2:]))

			if respIter+6+resValueLen > len(resp.Value) {
				tracer.Finish()
				cb(nil, ErrProtocol)
				return
			}

			results[i].Err = agent.makeBasicMemdError(resError)
			results[i].Value = resp.Value[respIter+6 : respIter+6+resValueLen]
			respIter += 6 + resValueLen
		}

		tracer.Finish()
		cb(&LookupInResult{
			Cas: Cas(resp.Cas),
			Ops: results,
		}, err)
	}

	pathBytesList := make([][]byte, len(opts.Ops))
	pathBytesTotal := 0
	for i, op := range opts.Ops {
		pathBytes := []byte(op.Path)
		pathBytesList[i] = pathBytes
		pathBytesTotal += len(pathBytes)
	}

	valueBuf := make([]byte, len(opts.Ops)*4+pathBytesTotal)

	valueIter := 0
	for i, op := range opts.Ops {
		if op.Op != SubDocOpGet && op.Op != SubDocOpExists &&
			op.Op != SubDocOpGetDoc && op.Op != SubDocOpGetCount {
			return nil, ErrInvalidArgs
		}
		if op.Value != nil {
			return nil, ErrInvalidArgs
		}

		pathBytes := pathBytesList[i]
		pathBytesLen := len(pathBytes)

		valueBuf[valueIter+0] = uint8(op.Op)
		valueBuf[valueIter+1] = uint8(op.Flags)
		binary.BigEndian.PutUint16(valueBuf[valueIter+2:], uint16(pathBytesLen))
		copy(valueBuf[valueIter+4:], pathBytes)
		valueIter += 4 + pathBytesLen
	}

	var extraBuf []byte
	if opts.Flags != 0 {
		extraBuf = append(extraBuf, uint8(opts.Flags))
	}

	req := &memdQRequest{
		memdPacket: memdPacket{
			Magic:    reqMagic,
			Opcode:   cmdSubDocMultiLookup,
			Datatype: 0,
			Cas:      0,
			Extras:   extraBuf,
			Key:      opts.Key,
			Value:    valueBuf,
		},
		Callback:         handler,
		RootTraceContext: tracer.RootContext(),
	}
	return agent.dispatchOp(req)
}

// MutateInOptions encapsulates the parameters for a MutateInEx operation.
type MutateInOptions struct {
	Key          []byte
	Flags        SubdocDocFlag
	Cas          Cas
	Expiry       uint32
	Ops          []SubDocOp
	TraceContext opentracing.SpanContext
}

// MutateInResult encapsulates the result of a MutateInEx operation.
type MutateInResult struct {
	Cas           Cas
	MutationToken MutationToken
	Ops           []SubDocResult
}

// MutateInExCallback is invoked upon completion of a MutateInEx operation.
type MutateInExCallback func(*MutateInResult, error)

// MutateInEx performs a multiple-mutation sub-document operation on a document.
func (agent *Agent) MutateInEx(opts MutateInOptions, cb MutateInExCallback) (PendingOp, error) {
	tracer := agent.createOpTrace("MutateInEx", opts.TraceContext)

	results := make([]SubDocResult, len(opts.Ops))

	handler := func(resp *memdQResponse, req *memdQRequest, err error) {
		if err != nil &&
			!IsErrorStatus(err, StatusSubDocSuccessDeleted) &&
			!IsErrorStatus(err, StatusSubDocBadMulti) {
			tracer.Finish()
			cb(nil, err)
			return
		}

		if IsErrorStatus(err, StatusSubDocBadMulti) {
			if len(resp.Value) != 3 {
				tracer.Finish()
				cb(nil, ErrProtocol)
				return
			}

			opIndex := int(resp.Value[0])
			resError := StatusCode(binary.BigEndian.Uint16(resp.Value[1:]))

			err := SubDocMutateError{
				Err:     agent.makeBasicMemdError(resError),
				OpIndex: opIndex,
			}
			tracer.Finish()
			cb(nil, err)
			return
		}

		for readPos := uint32(0); readPos < uint32(len(resp.Value)); {
			opIndex := int(resp.Value[readPos+0])
			opStatus := StatusCode(binary.BigEndian.Uint16(resp.Value[readPos+1:]))
			results[opIndex].Err = agent.makeBasicMemdError(opStatus)
			readPos += 3

			if opStatus == StatusSuccess {
				valLength := binary.BigEndian.Uint32(resp.Value[readPos:])
				results[opIndex].Value = resp.Value[readPos+4 : readPos+4+valLength]
				readPos += 4 + valLength
			}
		}

		mutToken := MutationToken{}
		if len(resp.Extras) >= 16 {
			mutToken.VbId = req.Vbucket
			mutToken.VbUuid = VbUuid(binary.BigEndian.Uint64(resp.Extras[0:]))
			mutToken.SeqNo = SeqNo(binary.BigEndian.Uint64(resp.Extras[8:]))
		}

		tracer.Finish()
		cb(&MutateInResult{
			Cas:           Cas(resp.Cas),
			MutationToken: mutToken,
			Ops:           results,
		}, nil)
	}

	pathBytesList := make([][]byte, len(opts.Ops))
	pathBytesTotal := 0
	valueBytesTotal := 0
	for i, op := range opts.Ops {
		pathBytes := []byte(op.Path)
		pathBytesList[i] = pathBytes
		pathBytesTotal += len(pathBytes)
		valueBytesTotal += len(op.Value)
	}

	valueBuf := make([]byte, len(opts.Ops)*8+pathBytesTotal+valueBytesTotal)

	valueIter := 0
	for i, op := range opts.Ops {
		if op.Op != SubDocOpDictAdd && op.Op != SubDocOpDictSet &&
			op.Op != SubDocOpDelete && op.Op != SubDocOpReplace &&
			op.Op != SubDocOpArrayPushLast && op.Op != SubDocOpArrayPushFirst &&
			op.Op != SubDocOpArrayInsert && op.Op != SubDocOpArrayAddUnique &&
			op.Op != SubDocOpCounter && op.Op != SubDocOpSetDoc &&
			op.Op != SubDocOpAddDoc && op.Op != SubDocOpDeleteDoc {
			return nil, ErrInvalidArgs
		}

		pathBytes := pathBytesList[i]
		pathBytesLen := len(pathBytes)
		valueBytesLen := len(op.Value)

		valueBuf[valueIter+0] = uint8(op.Op)
		valueBuf[valueIter+1] = uint8(op.Flags)
		binary.BigEndian.PutUint16(valueBuf[valueIter+2:], uint16(pathBytesLen))
		binary.BigEndian.PutUint32(valueBuf[valueIter+4:], uint32(valueBytesLen))
		copy(valueBuf[valueIter+8:], pathBytes)
		copy(valueBuf[valueIter+8+pathBytesLen:], op.Value)
		valueIter += 8 + pathBytesLen + valueBytesLen
	}

	var extraBuf []byte
	if opts.Expiry != 0 {
		tmpBuf := make([]byte, 4)
		binary.BigEndian.PutUint32(tmpBuf[0:], opts.Expiry)
		extraBuf = append(extraBuf, tmpBuf...)
	}
	if opts.Flags != 0 {
		extraBuf = append(extraBuf, uint8(opts.Flags))
	}

	req := &memdQRequest{
		memdPacket: memdPacket{
			Magic:    reqMagic,
			Opcode:   cmdSubDocMultiMutation,
			Datatype: 0,
			Cas:      uint64(opts.Cas),
			Extras:   extraBuf,
			Key:      opts.Key,
			Value:    valueBuf,
		},
		Callback:         handler,
		RootTraceContext: tracer.RootContext(),
	}
	return agent.dispatchOp(req)
}
