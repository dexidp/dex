package gocbcore

import (
	"sync"
	"sync/atomic"
	"time"

	"github.com/opentracing/opentracing-go"
)

// PingResult contains the results of a ping to a single server.
type PingResult struct {
	Endpoint string
	Error    error
	Latency  time.Duration
}

type pingSubOp struct {
	op       PendingOp
	endpoint string
}

type pingOp struct {
	lock      sync.Mutex
	subops    []pingSubOp
	remaining int32
	results   []PingResult
	callback  PingKvExCallback
}

func (pop *pingOp) Cancel() bool {
	for _, subop := range pop.subops {
		if subop.op.Cancel() {
			pop.lock.Lock()
			pop.results = append(pop.results, PingResult{
				Endpoint: subop.endpoint,
				Error:    ErrCancelled,
				Latency:  0,
			})
			pop.handledOneLocked()
			pop.lock.Unlock()
		}
	}
	return false
}

func (pop *pingOp) handledOneLocked() {
	remaining := atomic.AddInt32(&pop.remaining, -1)
	if remaining == 0 {
		pop.callback(&PingKvResult{
			Services: pop.results,
		}, nil)
	}
}

// PingKvOptions encapsulates the parameters for a PingKvEx operation.
type PingKvOptions struct {
	TraceContext opentracing.SpanContext
}

// PingKvResult encapsulates the result of a PingKvEx operation.
type PingKvResult struct {
	Services []PingResult
}

// PingKvExCallback is invoked upon completion of a PingKvEx operation.
type PingKvExCallback func(*PingKvResult, error)

// PingKvEx pings all of the servers we are connected to and returns
// a report regarding the pings that were performed.
func (agent *Agent) PingKvEx(opts PingKvOptions, cb PingKvExCallback) (PendingOp, error) {
	config := agent.routingInfo.Get()
	if config == nil {
		return nil, ErrShutdown
	}

	op := &pingOp{
		callback:  cb,
		remaining: 1,
	}

	pingStartTime := time.Now()

	kvHandler := func(resp *memdQResponse, req *memdQRequest, err error) {
		serverAddress := resp.sourceAddr

		pingLatency := time.Now().Sub(pingStartTime)

		op.lock.Lock()
		op.results = append(op.results, PingResult{
			Endpoint: serverAddress,
			Error:    err,
			Latency:  pingLatency,
		})
		op.handledOneLocked()
		op.lock.Unlock()
	}

	for serverIdx := 0; serverIdx < config.clientMux.NumPipelines(); serverIdx++ {
		pipeline := config.clientMux.GetPipeline(serverIdx)
		serverAddress := pipeline.Address()

		req := &memdQRequest{
			memdPacket: memdPacket{
				Magic:    reqMagic,
				Opcode:   cmdNoop,
				Datatype: 0,
				Cas:      0,
				Key:      nil,
				Value:    nil,
			},
			Callback: kvHandler,
		}

		curOp, err := agent.dispatchOpToAddress(req, serverAddress)
		if err != nil {
			op.lock.Lock()
			op.results = append(op.results, PingResult{
				Endpoint: serverAddress,
				Error:    err,
				Latency:  0,
			})
			op.lock.Unlock()
			continue
		}

		op.lock.Lock()
		op.subops = append(op.subops, pingSubOp{
			endpoint: serverAddress,
			op:       curOp,
		})
		atomic.AddInt32(&op.remaining, 1)
		op.lock.Unlock()
	}

	// We initialized remaining to one to ensure that the callback is not
	// invoked until all of the operations have been dispatched first.  This
	// final handling is to indicate that all operations were dispatched.
	op.lock.Lock()
	op.handledOneLocked()
	op.lock.Unlock()

	return op, nil
}

// MemdConnInfo represents information we know about a particular
// memcached connection reported in a diagnostics report.
type MemdConnInfo struct {
	LocalAddr    string
	RemoteAddr   string
	LastActivity time.Time
}

// DiagnosticInfo is returned by the Diagnostics method and includes
// information about the overall health of the clients connections.
type DiagnosticInfo struct {
	ConfigRev int64
	MemdConns []MemdConnInfo
}

// Diagnostics returns diagnostics information about the client.
// Mainly containing a list of open connections and their current
// states.
func (agent *Agent) Diagnostics() (*DiagnosticInfo, error) {
	for {
		config := agent.routingInfo.Get()
		if config == nil {
			return nil, ErrShutdown
		}

		var conns []MemdConnInfo

		for _, pipeline := range config.clientMux.pipelines {
			remoteAddr := pipeline.address

			pipeline.clientsLock.Lock()
			for _, pipecli := range pipeline.clients {
				localAddr := ""
				var lastActivity time.Time

				pipecli.lock.Lock()
				if pipecli.client != nil {
					localAddr = pipecli.client.Address()
					lastActivityUs := atomic.LoadInt64(&pipecli.client.lastActivity)
					if lastActivityUs != 0 {
						lastActivity = time.Unix(0, lastActivityUs)
					}
				}
				pipecli.lock.Unlock()

				conns = append(conns, MemdConnInfo{
					LocalAddr:    localAddr,
					RemoteAddr:   remoteAddr,
					LastActivity: lastActivity,
				})
			}
			pipeline.clientsLock.Unlock()
		}

		endConfig := agent.routingInfo.Get()
		if endConfig == config {
			return &DiagnosticInfo{
				ConfigRev: config.revId,
				MemdConns: conns,
			}, nil
		}
	}
}
