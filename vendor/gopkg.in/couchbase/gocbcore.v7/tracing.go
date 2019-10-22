package gocbcore

import (
	"encoding/json"
	"fmt"

	"sort"
	"time"

	"github.com/opentracing/opentracing-go"
)

type opTracer struct {
	parentContext opentracing.SpanContext
	opSpan        opentracing.Span
}

func (tracer *opTracer) Finish() {
	if tracer.opSpan != nil {
		tracer.opSpan.Finish()
	}
}

func (tracer *opTracer) RootContext() opentracing.SpanContext {
	if tracer.opSpan != nil {
		return tracer.opSpan.Context()
	}

	return tracer.parentContext
}

func (agent *Agent) createOpTrace(operationName string, parentContext opentracing.SpanContext) *opTracer {
	if agent.noRootTraceSpans {
		return &opTracer{
			parentContext: parentContext,
			opSpan:        nil,
		}
	}

	opSpan := agent.tracer.StartSpan(operationName,
		opentracing.ChildOf(parentContext),
		opentracing.Tag{Key: "component", Value: "couchbase-go-sdk"},
		opentracing.Tag{Key: "db.instance", Value: agent.bucket},
		opentracing.Tag{Key: "db.type", Value: "couchbase"},
		opentracing.Tag{Key: "span.kind", Value: "client"})

	return &opTracer{
		parentContext: parentContext,
		opSpan:        opSpan,
	}
}

func (agent *Agent) startCmdTrace(req *memdQRequest) {
	if req.cmdTraceSpan != nil {
		logWarnf("Attempted to start tracing on traced request")
		return
	}

	if req.RootTraceContext == nil {
		return
	}

	req.processingLock.Lock()
	req.cmdTraceSpan = agent.tracer.StartSpan(
		getCommandName(req.memdPacket.Opcode),
		opentracing.ChildOf(req.RootTraceContext),
		opentracing.Tag{Key: "retry", Value: req.retryCount})
	req.processingLock.Unlock()
}

func (agent *Agent) stopCmdTrace(req *memdQRequest) {
	if req.RootTraceContext == nil {
		return
	}

	if req.cmdTraceSpan == nil {
		logWarnf("Attempted to stop tracing on untraced request")
		return
	}

	req.cmdTraceSpan.Finish()
	req.cmdTraceSpan = nil
}

func (agent *Agent) startNetTrace(req *memdQRequest) {
	if req.cmdTraceSpan == nil {
		return
	}

	if req.netTraceSpan != nil {
		logWarnf("Attempted to start net tracing on traced request")
		return
	}

	req.processingLock.Lock()
	req.netTraceSpan = agent.tracer.StartSpan(
		"rpc",
		opentracing.ChildOf(req.cmdTraceSpan.Context()),
		opentracing.Tag{Key: "span.kind", Value: "client"})
	req.processingLock.Unlock()
}

func (agent *Agent) stopNetTrace(req *memdQRequest, resp *memdQResponse, client *memdClient) {
	if req.cmdTraceSpan == nil {
		return
	}

	if req.netTraceSpan == nil {
		logWarnf("Attempted to stop net tracing on an untraced request")
		return
	}

	req.netTraceSpan.SetTag("couchbase.operation_id", fmt.Sprintf("0x%x", resp.Opaque))
	req.netTraceSpan.SetTag("couchbase.local_id", resp.sourceConnId)
	if isLogRedactionLevelNone() {
		req.netTraceSpan.SetTag("couchbase.document_key", string(req.Key))
	}
	req.netTraceSpan.SetTag("local.address", client.conn.LocalAddr())
	req.netTraceSpan.SetTag("peer.address", client.conn.RemoteAddr())
	if resp.FrameExtras != nil && resp.FrameExtras.HasSrvDuration {
		req.netTraceSpan.SetTag("server_duration", resp.FrameExtras.SrvDuration)
	}

	req.netTraceSpan.Finish()
	req.netTraceSpan = nil
}

func (agent *Agent) cancelReqTrace(req *memdQRequest, err error) {
	if req.cmdTraceSpan != nil {
		if req.netTraceSpan != nil {
			req.netTraceSpan.Finish()
		}

		req.cmdTraceSpan.Finish()
	}
}

type zombieLogEntry struct {
	connectionID string
	operationID  string
	endpoint     string
	duration     time.Duration
	serviceType  string
}

type zombieLogItem struct {
	ConnectionID     string `json:"c"`
	OperationID      string `json:"i"`
	Endpoint         string `json:"r"`
	ServerDurationUs uint64 `json:"d"`
	ServiceType      string `json:"s"`
}

type zombieLogService struct {
	Service string          `json:"service"`
	Count   int             `json:"count"`
	Top     []zombieLogItem `json:"top"`
}

func (agent *Agent) zombieLogger(interval time.Duration, sampleSize int) {
	lastTick := time.Now()

	for {
		// We tick every 1 second to make sure that the goroutines
		// are cleaned up promptly after agent shutdown.
		<-time.After(1 * time.Second)

		routingInfo := agent.routingInfo.Get()
		if routingInfo == nil {
			// If the routing info is gone, the agent shut down and we should close
			return
		}

		if time.Now().Sub(lastTick) < interval {
			continue
		}

		lastTick = lastTick.Add(interval)

		// Preallocate space to copy the ops into...
		oldOps := make([]*zombieLogEntry, sampleSize)

		agent.zombieLock.Lock()
		// Escape early if we have no ops to log...
		if len(agent.zombieOps) == 0 {
			agent.zombieLock.Unlock()
			return
		}

		// Copy out our ops so we can cheaply print them out without blocking
		// our ops from actually being recorded in other goroutines (which would
		// effectively slow down the op pipeline for logging).

		oldOps = oldOps[0:len(agent.zombieOps)]
		copy(oldOps, agent.zombieOps)
		agent.zombieOps = agent.zombieOps[:0]

		agent.zombieLock.Unlock()

		jsonData := zombieLogService{
			Service: "kv",
		}

		for i := len(oldOps) - 1; i >= 0; i-- {
			op := oldOps[i]

			jsonData.Top = append(jsonData.Top, zombieLogItem{
				OperationID:      op.operationID,
				ConnectionID:     op.connectionID,
				Endpoint:         op.endpoint,
				ServerDurationUs: uint64(op.duration / time.Microsecond),
				ServiceType:      op.serviceType,
			})
		}

		jsonData.Count = len(jsonData.Top)

		jsonBytes, err := json.Marshal(jsonData)
		if err != nil {
			logDebugf("Failed to generate zombie logging JSON: %s", err)
		}

		logWarnf("Orphaned responses observed:\n %s", jsonBytes)
	}
}

func (agent *Agent) recordZombieResponse(resp *memdQResponse, client *memdClient) {
	entry := &zombieLogEntry{
		connectionID: client.connId,
		operationID:  fmt.Sprintf("0x%x", resp.Opaque),
		endpoint:     client.Address(),
		duration:     0,
		serviceType:  fmt.Sprintf("kv:%s", getCommandName(resp.Opcode)),
	}

	if resp.FrameExtras != nil && resp.FrameExtras.HasSrvDuration {
		entry.duration = resp.FrameExtras.SrvDuration
	}

	agent.zombieLock.RLock()
	if len(agent.zombieOps) == cap(agent.zombieOps) && entry.duration < agent.zombieOps[0].duration {
		// we are at capacity and we are faster than the fastest slow op
		agent.zombieLock.RUnlock()
		return
	}
	agent.zombieLock.RUnlock()

	agent.zombieLock.Lock()
	if len(agent.zombieOps) == cap(agent.zombieOps) && entry.duration < agent.zombieOps[0].duration {
		// we are at capacity and we are faster than the fastest slow op
		agent.zombieLock.Unlock()
		return
	}

	l := len(agent.zombieOps)
	i := sort.Search(l, func(i int) bool { return entry.duration < agent.zombieOps[i].duration })

	// i represents the slot where it should be inserted

	if len(agent.zombieOps) < cap(agent.zombieOps) {
		if i == l {
			agent.zombieOps = append(agent.zombieOps, entry)
		} else {
			agent.zombieOps = append(agent.zombieOps, nil)
			copy(agent.zombieOps[i+1:], agent.zombieOps[i:])
			agent.zombieOps[i] = entry
		}
	} else {
		if i == 0 {
			agent.zombieOps[i] = entry
		} else {
			copy(agent.zombieOps[0:i-1], agent.zombieOps[1:i])
			agent.zombieOps[i-1] = entry
		}
	}

	agent.zombieLock.Unlock()
}
