package gocbcore

import (
	"encoding/binary"

	"github.com/opentracing/opentracing-go"
)

// ObserveOptions encapsulates the parameters for a ObserveEx operation.
type ObserveOptions struct {
	Key          []byte
	ReplicaIdx   int
	TraceContext opentracing.SpanContext
}

// ObserveResult encapsulates the result of a ObserveEx operation.
type ObserveResult struct {
	KeyState KeyState
	Cas      Cas
}

// ObserveExCallback is invoked upon completion of a ObserveEx operation.
type ObserveExCallback func(*ObserveResult, error)

// ObserveEx retrieves the current CAS and persistence state for a document.
func (agent *Agent) ObserveEx(opts ObserveOptions, cb ObserveExCallback) (PendingOp, error) {
	tracer := agent.createOpTrace("ObserveEx", opts.TraceContext)

	if agent.bucketType() != bktTypeCouchbase {
		tracer.Finish()
		return nil, ErrNotSupported
	}

	handler := func(resp *memdQResponse, _ *memdQRequest, err error) {
		if err != nil {
			tracer.Finish()
			cb(nil, err)
			return
		}

		if len(resp.Value) < 4 {
			tracer.Finish()
			cb(nil, ErrProtocol)
			return
		}
		keyLen := int(binary.BigEndian.Uint16(resp.Value[2:]))

		if len(resp.Value) != 2+2+keyLen+1+8 {
			tracer.Finish()
			cb(nil, ErrProtocol)
			return
		}
		keyState := KeyState(resp.Value[2+2+keyLen])
		cas := binary.BigEndian.Uint64(resp.Value[2+2+keyLen+1:])

		tracer.Finish()
		cb(&ObserveResult{
			KeyState: keyState,
			Cas:      Cas(cas),
		}, nil)
	}

	vbId := agent.KeyToVbucket(opts.Key)
	keyLen := len(opts.Key)

	valueBuf := make([]byte, 2+2+keyLen)
	binary.BigEndian.PutUint16(valueBuf[0:], vbId)
	binary.BigEndian.PutUint16(valueBuf[2:], uint16(keyLen))
	copy(valueBuf[4:], opts.Key)

	req := &memdQRequest{
		memdPacket: memdPacket{
			Magic:    reqMagic,
			Opcode:   cmdObserve,
			Datatype: 0,
			Cas:      0,
			Extras:   nil,
			Key:      nil,
			Value:    valueBuf,
			Vbucket:  vbId,
		},
		ReplicaIdx:       opts.ReplicaIdx,
		Callback:         handler,
		RootTraceContext: tracer.RootContext(),
	}
	return agent.dispatchOp(req)
}

// ObserveVbOptions encapsulates the parameters for a ObserveVbEx operation.
type ObserveVbOptions struct {
	VbId         uint16
	VbUuid       VbUuid
	ReplicaIdx   int
	TraceContext opentracing.SpanContext
}

// ObserveVbResult encapsulates the result of a ObserveVbEx operation.
type ObserveVbResult struct {
	DidFailover  bool
	VbId         uint16
	VbUuid       VbUuid
	PersistSeqNo SeqNo
	CurrentSeqNo SeqNo
	OldVbUuid    VbUuid
	LastSeqNo    SeqNo
}

// ObserveVbExCallback is invoked upon completion of a ObserveVbEx operation.
type ObserveVbExCallback func(*ObserveVbResult, error)

// ObserveVbEx retrieves the persistence state sequence numbers for a particular VBucket
// and includes additional details not included by the basic version.
func (agent *Agent) ObserveVbEx(opts ObserveVbOptions, cb ObserveVbExCallback) (PendingOp, error) {
	tracer := agent.createOpTrace("ObserveVbEx", nil)

	if agent.bucketType() != bktTypeCouchbase {
		tracer.Finish()
		return nil, ErrNotSupported
	}

	handler := func(resp *memdQResponse, _ *memdQRequest, err error) {
		if err != nil {
			tracer.Finish()
			cb(nil, err)
			return
		}

		if len(resp.Value) < 1 {
			tracer.Finish()
			cb(nil, ErrProtocol)
			return
		}

		formatType := resp.Value[0]
		if formatType == 0 {
			// Normal
			if len(resp.Value) < 27 {
				tracer.Finish()
				cb(nil, ErrProtocol)
				return
			}

			vbId := binary.BigEndian.Uint16(resp.Value[1:])
			vbUuid := binary.BigEndian.Uint64(resp.Value[3:])
			persistSeqNo := binary.BigEndian.Uint64(resp.Value[11:])
			currentSeqNo := binary.BigEndian.Uint64(resp.Value[19:])

			tracer.Finish()
			cb(&ObserveVbResult{
				DidFailover:  false,
				VbId:         vbId,
				VbUuid:       VbUuid(vbUuid),
				PersistSeqNo: SeqNo(persistSeqNo),
				CurrentSeqNo: SeqNo(currentSeqNo),
			}, nil)
			return
		} else if formatType == 1 {
			// Hard Failover
			if len(resp.Value) < 43 {
				cb(nil, ErrProtocol)
				return
			}

			vbId := binary.BigEndian.Uint16(resp.Value[1:])
			vbUuid := binary.BigEndian.Uint64(resp.Value[3:])
			persistSeqNo := binary.BigEndian.Uint64(resp.Value[11:])
			currentSeqNo := binary.BigEndian.Uint64(resp.Value[19:])
			oldVbUuid := binary.BigEndian.Uint64(resp.Value[27:])
			lastSeqNo := binary.BigEndian.Uint64(resp.Value[35:])

			tracer.Finish()
			cb(&ObserveVbResult{
				DidFailover:  true,
				VbId:         vbId,
				VbUuid:       VbUuid(vbUuid),
				PersistSeqNo: SeqNo(persistSeqNo),
				CurrentSeqNo: SeqNo(currentSeqNo),
				OldVbUuid:    VbUuid(oldVbUuid),
				LastSeqNo:    SeqNo(lastSeqNo),
			}, nil)
			return
		} else {
			tracer.Finish()
			cb(nil, ErrProtocol)
			return
		}
	}

	valueBuf := make([]byte, 8)
	binary.BigEndian.PutUint64(valueBuf[0:], uint64(opts.VbUuid))

	req := &memdQRequest{
		memdPacket: memdPacket{
			Magic:    reqMagic,
			Opcode:   cmdObserveSeqNo,
			Datatype: 0,
			Cas:      0,
			Extras:   nil,
			Key:      nil,
			Value:    valueBuf,
			Vbucket:  opts.VbId,
		},
		ReplicaIdx:       opts.ReplicaIdx,
		Callback:         handler,
		RootTraceContext: tracer.RootContext(),
	}
	return agent.dispatchOp(req)
}
