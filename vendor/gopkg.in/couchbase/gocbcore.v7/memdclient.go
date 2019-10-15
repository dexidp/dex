package gocbcore

import (
	"encoding/binary"
	"sync"
	"sync/atomic"
	"time"
	"unsafe"

	"github.com/golang/snappy"
)

func isCompressibleOp(command commandCode) bool {
	switch command {
	case cmdSet:
		fallthrough
	case cmdAdd:
		fallthrough
	case cmdReplace:
		fallthrough
	case cmdAppend:
		fallthrough
	case cmdPrepend:
		return true
	}
	return false
}

type memdClient struct {
	lastActivity          int64
	dcpAckSize            int
	dcpFlowRecv           int
	closeNotify           chan bool
	connId                string
	closed                bool
	parent                *Agent
	conn                  memdConn
	opList                memdOpMap
	errorMap              *kvErrorMap
	features              []HelloFeature
	lock                  sync.Mutex
	streamEndNotSupported bool
}

func newMemdClient(parent *Agent, conn memdConn) *memdClient {
	client := memdClient{
		parent:      parent,
		conn:        conn,
		closeNotify: make(chan bool),
		connId:      parent.clientId + "/" + formatCbUid(randomCbUid()),
	}
	client.run()
	return &client
}

func (client *memdClient) SupportsFeature(feature HelloFeature) bool {
	return checkSupportsFeature(client.features, feature)
}

func (client *memdClient) EnableDcpBufferAck(bufferAckSize int) {
	client.dcpAckSize = bufferAckSize
}

func (client *memdClient) maybeSendDcpBufferAck(packet *memdPacket) {
	packetLen := 24 + len(packet.Extras) + len(packet.Key) + len(packet.Value)

	client.dcpFlowRecv += packetLen
	if client.dcpFlowRecv < client.dcpAckSize {
		return
	}

	ackAmt := client.dcpFlowRecv

	extrasBuf := make([]byte, 4)
	binary.BigEndian.PutUint32(extrasBuf, uint32(ackAmt))

	err := client.conn.WritePacket(&memdPacket{
		Magic:  reqMagic,
		Opcode: cmdDcpBufferAck,
		Extras: extrasBuf,
	})
	if err != nil {
		logWarnf("Failed to dispatch DCP buffer ack: %s", err)
	}

	client.dcpFlowRecv -= ackAmt
}

func (client *memdClient) SetErrorMap(errorMap *kvErrorMap) {
	client.errorMap = errorMap
}

func (client *memdClient) Address() string {
	return client.conn.RemoteAddr()
}

func (client *memdClient) CloseNotify() chan bool {
	return client.closeNotify
}

func (client *memdClient) takeRequestOwnership(req *memdQRequest) bool {
	client.lock.Lock()
	defer client.lock.Unlock()

	if client.closed {
		logDebugf("Attempted to put dispatched op in drained opmap")
		return false
	}

	if !atomic.CompareAndSwapPointer(&req.waitingIn, nil, unsafe.Pointer(client)) {
		logDebugf("Attempted to put dispatched op in new opmap")
		return false
	}

	if req.isCancelled() {
		atomic.CompareAndSwapPointer(&req.waitingIn, unsafe.Pointer(client), nil)
		return false
	}

	client.opList.Add(req)
	return true
}

func (client *memdClient) CancelRequest(req *memdQRequest) bool {
	client.lock.Lock()
	defer client.lock.Unlock()

	if client.closed {
		logDebugf("Attempted to remove op from drained opmap")
		return false
	}

	removed := client.opList.Remove(req)
	if removed {
		atomic.CompareAndSwapPointer(&req.waitingIn, unsafe.Pointer(client), nil)
	}

	return removed
}

func (client *memdClient) SendRequest(req *memdQRequest) error {
	addSuccess := client.takeRequestOwnership(req)
	if !addSuccess {
		return ErrCancelled
	}

	packet := &req.memdPacket
	if client.SupportsFeature(FeatureSnappy) {
		isCompressed := (packet.Datatype & uint8(DatatypeFlagCompressed)) != 0
		packetSize := len(packet.Value)
		if !isCompressed && packetSize > client.parent.compressionMinSize && isCompressibleOp(packet.Opcode) {
			compressedValue := snappy.Encode(nil, packet.Value)
			if float64(len(compressedValue))/float64(packetSize) <= client.parent.compressionMinRatio {
				newPacket := *packet
				newPacket.Value = compressedValue
				newPacket.Datatype = newPacket.Datatype | uint8(DatatypeFlagCompressed)
				packet = &newPacket
			}
		}
	}

	logSchedf("Writing request. %s to %s OP=0x%x. Opaque=%d", client.conn.LocalAddr(), client.Address(), req.Opcode, req.Opaque)

	client.parent.startNetTrace(req)

	err := client.conn.WritePacket(packet)
	if err != nil {
		logDebugf("memdClient write failure: %v", err)
		client.CancelRequest(req)
		return err
	}

	return nil
}

func (client *memdClient) resolveRequest(resp *memdQResponse) {
	opIndex := resp.Opaque

	client.lock.Lock()
	// Find the request that goes with this response, don't check if the client is
	// closed so that we can handle orphaned responses.
	req := client.opList.FindAndMaybeRemove(opIndex, resp.Status != StatusSuccess)
	client.lock.Unlock()

	if req == nil {
		// There is no known request that goes with this response.  Ignore it.
		logDebugf("Received response with no corresponding request.")
		if client.parent.useZombieLogger {
			client.parent.recordZombieResponse(resp, client)
		}
		return
	}
	if !req.Persistent {
		atomic.CompareAndSwapPointer(&req.waitingIn, unsafe.Pointer(client), nil)
	}

	req.processingLock.Lock()

	if !req.Persistent {
		client.parent.stopNetTrace(req, resp, client)
	}

	isCompressed := (resp.Datatype & uint8(DatatypeFlagCompressed)) != 0
	if isCompressed && !client.parent.disableDecompression {
		newValue, err := snappy.Decode(nil, resp.Value)
		if err != nil {
			req.processingLock.Unlock()
			logDebugf("Failed to decompress value from the server for key `%s`.", req.Key)
			return
		}

		resp.Value = newValue
		resp.Datatype = resp.Datatype & ^uint8(DatatypeFlagCompressed)
	}

	// Give the agent an opportunity to intercept the response first
	var err error
	if resp.Magic == resMagic {
		if resp.Status != StatusSuccess {
			if ok, foundErr := findMemdError(resp.Status); ok {
				err = foundErr
			} else {
				err = newSimpleError(resp.Status)
			}
		}
	}

	if client.parent == nil {
		req.processingLock.Unlock()
	} else {
		if !req.Persistent {
			client.parent.stopCmdTrace(req)
		}

		req.processingLock.Unlock()
		if err != ErrCancelled {
			shortCircuited, routeErr := client.parent.handleOpRoutingResp(resp, req, err)
			if shortCircuited {
				logSchedf("Routing callback intercepted response")
				return
			}
			err = routeErr
		}
	}

	// Call the requests callback handler...
	logSchedf("Dispatching response callback. OP=0x%x. Opaque=%d", resp.Opcode, resp.Opaque)
	req.tryCallback(resp, err)
}

func (client *memdClient) run() {
	dcpBufferQ := make(chan *memdQResponse)
	dcpKillSwitch := make(chan bool)
	dcpKillNotify := make(chan bool)
	go func() {
		for {
			select {
			case resp, more := <-dcpBufferQ:
				if !more {
					dcpKillNotify <- true
					return
				}

				logSchedf("Resolving response OP=0x%x. Opaque=%d", resp.Opcode, resp.Opaque)
				client.resolveRequest(resp)

				// See below for information on why this is here.
				if !resp.isInternal {
					if client.dcpAckSize > 0 {
						client.maybeSendDcpBufferAck(&resp.memdPacket)
					}
				}
			case <-dcpKillSwitch:
				close(dcpBufferQ)
			}
		}
	}()

	go func() {
		for {
			resp := &memdQResponse{
				sourceAddr:   client.conn.RemoteAddr(),
				sourceConnId: client.connId,
			}

			err := client.conn.ReadPacket(&resp.memdPacket)
			if err != nil {
				if !client.closed {
					logErrorf("memdClient read failure: %v", err)
				}
				break
			}

			atomic.StoreInt64(&client.lastActivity, time.Now().UnixNano())

			// We handle DCP no-op's directly here so we can reply immediately.
			if resp.memdPacket.Opcode == cmdDcpNoop {
				err := client.conn.WritePacket(&memdPacket{
					Magic:  resMagic,
					Opcode: cmdDcpNoop,
					Opaque: resp.Opaque,
				})
				if err != nil {
					logWarnf("Failed to dispatch DCP noop reply: %s", err)
				}
				continue
			}

			// This is a fix for a bug in the server DCP implementation (MB-26363).  This
			// bug causes the server to fail to send a stream-end notification.  The server
			// does however synchronously stop the stream, and thus we can assume no more
			// packets will be received following the close response.
			if resp.Magic == resMagic && resp.Opcode == cmdDcpCloseStream && client.streamEndNotSupported {
				closeReq := client.opList.Find(resp.Opaque)
				if closeReq != nil {
					vbId := closeReq.Vbucket
					streamReq := client.opList.FindOpenStream(vbId)
					if streamReq != nil {
						endExtras := make([]byte, 4)
						binary.BigEndian.PutUint32(endExtras, uint32(streamEndClosed))
						endResp := &memdQResponse{
							memdPacket: memdPacket{
								Magic:   reqMagic,
								Opcode:  cmdDcpStreamEnd,
								Vbucket: vbId,
								Opaque:  streamReq.Opaque,
								Extras:  endExtras,
							},
							isInternal: true,
						}
						dcpBufferQ <- endResp
					}
				}
			}

			switch resp.memdPacket.Opcode {
			case cmdDcpDeletion:
				fallthrough
			case cmdDcpExpiration:
				fallthrough
			case cmdDcpMutation:
				fallthrough
			case cmdDcpSnapshotMarker:
				fallthrough
			case cmdDcpStreamEnd:
				dcpBufferQ <- resp
				continue
			default:
				logSchedf("Resolving response OP=0x%x. Opaque=%d", resp.Opcode, resp.Opaque)
				client.resolveRequest(resp)
			}
		}

		client.lock.Lock()
		if client.closed {
			client.lock.Unlock()
		} else {
			client.closed = true
			client.lock.Unlock()

			err := client.conn.Close()
			if err != nil {
				// Lets log an error, as this is non-fatal
				logErrorf("Failed to shut down client connection (%s)", err)
			}
		}

		dcpKillSwitch <- true
		<-dcpKillNotify

		client.opList.Drain(func(req *memdQRequest) {
			if !atomic.CompareAndSwapPointer(&req.waitingIn, unsafe.Pointer(client), nil) {
				logWarnf("Encountered an unowned request in a client opMap")
			}

			req.tryCallback(nil, ErrNetwork)
		})

		close(client.closeNotify)
	}()
}

func (client *memdClient) Close() error {
	client.lock.Lock()
	client.closed = true
	client.lock.Unlock()

	return client.conn.Close()
}
