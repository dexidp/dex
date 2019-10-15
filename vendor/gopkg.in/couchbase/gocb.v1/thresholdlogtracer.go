package gocb

import (
	"encoding/json"
	"sort"
	"sync"
	"sync/atomic"
	"time"

	"github.com/opentracing/opentracing-go"
	otlog "github.com/opentracing/opentracing-go/log"
)

var defaultThresholdLogTracer ThresholdLoggingTracer

type thresholdLogGroup struct {
	name  string
	floor time.Duration
	ops   []*thresholdLogSpan
	lock  sync.RWMutex
}

func (g *thresholdLogGroup) init(name string, floor time.Duration, size uint32) {
	g.name = name
	g.floor = floor
	g.ops = make([]*thresholdLogSpan, 0, size)
}

func (g *thresholdLogGroup) recordOp(span *thresholdLogSpan) {
	if span.duration < g.floor {
		return
	}

	// Preemptively check that we actually need to be inserted using a read lock first
	// this is a performance improvement measure to avoid locking the mutex all the time.
	g.lock.RLock()
	if len(g.ops) == cap(g.ops) && span.duration < g.ops[0].duration {
		// we are at capacity and we are faster than the fastest slow op
		g.lock.RUnlock()
		return
	}
	g.lock.RUnlock()

	g.lock.Lock()
	if len(g.ops) == cap(g.ops) && span.duration < g.ops[0].duration {
		// we are at capacity and we are faster than the fastest slow op
		g.lock.Unlock()
		return
	}

	l := len(g.ops)
	i := sort.Search(l, func(i int) bool { return span.duration < g.ops[i].duration })

	// i represents the slot where it should be inserted

	if len(g.ops) < cap(g.ops) {
		if i == l {
			g.ops = append(g.ops, span)
		} else {
			g.ops = append(g.ops, nil)
			copy(g.ops[i+1:], g.ops[i:])
			g.ops[i] = span
		}
	} else {
		if i == 0 {
			g.ops[i] = span
		} else {
			copy(g.ops[0:i-1], g.ops[1:i])
			g.ops[i-1] = span
		}
	}

	g.lock.Unlock()
}

type thresholdLogItem struct {
	OperationName          string `json:"operation_name,omitempty"`
	TotalTimeUs            uint64 `json:"total_us,omitempty"`
	EncodeDurationUs       uint64 `json:"encode_us,omitempty"`
	DispatchDurationUs     uint64 `json:"dispatch_us,omitempty"`
	ServerDurationUs       uint64 `json:"server_us,omitempty"`
	DecodeDurationUs       uint64 `json:"decode_us,omitempty"`
	LastRemoteAddress      string `json:"last_remote_address,omitempty"`
	LastLocalAddress       string `json:"last_local_address,omitempty"`
	LastDispatchDurationUs uint64 `json:"last_dispatch_us,omitempty"`
	LastOperationID        string `json:"last_operation_id,omitempty"`
	LastLocalID            string `json:"last_local_id,omitempty"`
	DocumentKey            string `json:"document_key,omitempty"`
}

type thresholdLogService struct {
	Service string             `json:"service"`
	Count   int                `json:"count"`
	Top     []thresholdLogItem `json:"top"`
}

func (g *thresholdLogGroup) logRecordedRecords(sampleSize uint32) {
	// Preallocate space to copy the ops into...
	oldOps := make([]*thresholdLogSpan, sampleSize)

	g.lock.Lock()
	// Escape early if we have no ops to log...
	if len(g.ops) == 0 {
		g.lock.Unlock()
		return
	}

	// Copy out our ops so we can cheaply print them out without blocking
	// our ops from actually being recorded in other goroutines (which would
	// effectively slow down the op pipeline for logging).

	oldOps = oldOps[0:len(g.ops)]
	copy(oldOps, g.ops)
	g.ops = g.ops[:0]

	g.lock.Unlock()

	jsonData := thresholdLogService{
		Service: g.name,
	}

	for i := len(oldOps) - 1; i >= 0; i-- {
		op := oldOps[i]

		jsonData.Top = append(jsonData.Top, thresholdLogItem{
			OperationName:          op.opName,
			TotalTimeUs:            uint64(op.duration / time.Microsecond),
			DispatchDurationUs:     uint64(op.totalDispatchDuration / time.Microsecond),
			ServerDurationUs:       uint64(op.totalServerDuration / time.Microsecond),
			EncodeDurationUs:       uint64(op.totalEncodeDuration / time.Microsecond),
			DecodeDurationUs:       uint64(op.totalDecodeDuration / time.Microsecond),
			LastRemoteAddress:      op.lastDispatchPeer,
			LastDispatchDurationUs: uint64(op.lastDispatchDuration / time.Microsecond),
			LastOperationID:        op.lastOperationID,
			LastLocalID:            op.lastLocalID,
			DocumentKey:            op.documentKey,
		})
	}

	jsonData.Count = len(jsonData.Top)

	jsonBytes, err := json.Marshal(jsonData)
	if err != nil {
		logDebugf("Failed to generate threshold logging service JSON: %s", err)
	}

	logInfof("Threshold Log: %s", jsonBytes)
}

// ThresholdLoggingTracer is a specialized Tracer implementation which will automatically
// log operations which fall outside of a set of thresholds.  Note that this tracer is
// only safe for use within the Couchbase SDK, uses by external event sources are
// likely to fail.
// EXPERIMENTAL
type ThresholdLoggingTracer struct {
	Interval           time.Duration
	SampleSize         uint32
	KvThreshold        time.Duration
	ViewsThreshold     time.Duration
	N1qlThreshold      time.Duration
	SearchThreshold    time.Duration
	AnalyticsThreshold time.Duration

	killCh         chan struct{}
	refCount       int32
	nextTick       time.Time
	kvGroup        thresholdLogGroup
	viewsGroup     thresholdLogGroup
	queryGroup     thresholdLogGroup
	searchGroup    thresholdLogGroup
	analyticsGroup thresholdLogGroup
}

// AddRef is used internally to keep track of the number of Cluster instances referring to it.
// This is used to correctly shut down the aggregation routines once there are no longer any
// instances tracing to it.
func (t *ThresholdLoggingTracer) AddRef() int32 {
	newRefCount := atomic.AddInt32(&t.refCount, 1)
	if newRefCount == 1 {
		t.startLoggerRoutine()
	}
	return newRefCount
}

// DecRef is the counterpart to AddRef (see AddRef for more information).
func (t *ThresholdLoggingTracer) DecRef() int32 {
	newRefCount := atomic.AddInt32(&t.refCount, -1)
	if newRefCount == 0 {
		t.killCh <- struct{}{}
	}
	return newRefCount
}

func (t *ThresholdLoggingTracer) logRecordedRecords() {
	t.kvGroup.logRecordedRecords(t.SampleSize)
	t.viewsGroup.logRecordedRecords(t.SampleSize)
	t.queryGroup.logRecordedRecords(t.SampleSize)
	t.searchGroup.logRecordedRecords(t.SampleSize)
	t.analyticsGroup.logRecordedRecords(t.SampleSize)
}

func (t *ThresholdLoggingTracer) startLoggerRoutine() {
	if t.Interval == 0 {
		t.Interval = 10 * time.Second
	}
	if t.SampleSize == 0 {
		t.SampleSize = 10
	}
	if t.KvThreshold == 0 {
		t.KvThreshold = 500 * time.Millisecond
	}
	if t.ViewsThreshold == 0 {
		t.ViewsThreshold = 1 * time.Second
	}
	if t.N1qlThreshold == 0 {
		t.N1qlThreshold = 1 * time.Second
	}
	if t.SearchThreshold == 0 {
		t.SearchThreshold = 1 * time.Second
	}
	if t.AnalyticsThreshold == 0 {
		t.AnalyticsThreshold = 1 * time.Second
	}

	t.kvGroup.init("kv", t.KvThreshold, t.SampleSize)
	t.viewsGroup.init("views", t.ViewsThreshold, t.SampleSize)
	t.queryGroup.init("query", t.N1qlThreshold, t.SampleSize)
	t.searchGroup.init("search", t.SearchThreshold, t.SampleSize)
	t.analyticsGroup.init("analytics", t.AnalyticsThreshold, t.SampleSize)

	if t.killCh == nil {
		t.killCh = make(chan struct{})
	}

	if t.nextTick.IsZero() {
		t.nextTick = time.Now().Add(t.Interval)
	}

	go t.loggerRoutine()
}

func (t *ThresholdLoggingTracer) loggerRoutine() {
	for {
		select {
		case <-time.After(t.nextTick.Sub(time.Now())):
			t.nextTick = t.nextTick.Add(t.Interval)
			t.logRecordedRecords()
		case <-t.killCh:
			t.logRecordedRecords()
			return
		}
	}
}

func (t *ThresholdLoggingTracer) recordOp(span *thresholdLogSpan) {
	switch span.serviceName {
	case "kv":
		t.kvGroup.recordOp(span)
	case "views":
		t.viewsGroup.recordOp(span)
	case "n1ql":
		t.queryGroup.recordOp(span)
	case "fts":
		t.searchGroup.recordOp(span)
	case "cbas":
		t.analyticsGroup.recordOp(span)
	}
}

// StartSpan belongs to the Tracer interface.
func (t *ThresholdLoggingTracer) StartSpan(operationName string, opts ...opentracing.StartSpanOption) opentracing.Span {
	span := &thresholdLogSpan{
		tracer:    t,
		opName:    operationName,
		startTime: time.Now(),
	}

	for _, opt := range opts {
		switch opt := opt.(type) {
		case opentracing.SpanReference:
			if opt.Type == opentracing.ChildOfRef {
				if context, ok := opt.ReferencedContext.(*thresholdLogSpanContext); ok {
					span.parent = context.span
				}
			}
		case opentracing.Tag:
			span.SetTag(opt.Key, opt.Value)
		}
	}

	return span
}

// Inject belongs to the Tracer interface.
func (t *ThresholdLoggingTracer) Inject(sp opentracing.SpanContext, format interface{}, carrier interface{}) error {
	return nil
}

// Extract belongs to the Tracer interface.
func (t *ThresholdLoggingTracer) Extract(format interface{}, carrier interface{}) (opentracing.SpanContext, error) {
	return nil, opentracing.ErrSpanContextNotFound
}

type thresholdLogSpan struct {
	tracer                *ThresholdLoggingTracer
	parent                *thresholdLogSpan
	opName                string
	startTime             time.Time
	serviceName           string
	peerAddress           string
	serverDuration        time.Duration
	duration              time.Duration
	totalServerDuration   time.Duration
	totalDispatchDuration time.Duration
	totalEncodeDuration   time.Duration
	totalDecodeDuration   time.Duration
	lastDispatchPeer      string
	lastDispatchDuration  time.Duration
	lastOperationID       string
	lastLocalID           string
	documentKey           string
}

func (n *thresholdLogSpan) Context() opentracing.SpanContext {
	return &thresholdLogSpanContext{n}
}

func (n *thresholdLogSpan) SetBaggageItem(key, val string) opentracing.Span {
	return n
}

func (n *thresholdLogSpan) BaggageItem(key string) string {
	return ""
}

func (n *thresholdLogSpan) SetTag(key string, value interface{}) opentracing.Span {
	var ok bool

	switch key {
	case "server_duration":
		if n.serverDuration, ok = value.(time.Duration); !ok {
			logDebugf("Failed to cast span server_duration tag")
		}
	case "couchbase.service":
		if n.serviceName, ok = value.(string); !ok {
			logDebugf("Failed to cast span couchbase.service tag")
		}
	case "peer.address":
		if n.peerAddress, ok = value.(string); !ok {
			logDebugf("Failed to cast span peer.address tag")
		}
	case "couchbase.operation_id":
		if n.lastOperationID, ok = value.(string); !ok {
			logDebugf("Failed to cast span couchbase.operation_id tag")
		}
	case "couchbase.document_key":
		if n.documentKey, ok = value.(string); !ok {
			logDebugf("Failed to cast span couchbase.document_key tag")
		}
	case "couchbase.local_id":
		if n.lastLocalID, ok = value.(string); !ok {
			logDebugf("Failed to cast span couchbase.local_id tag")
		}
	}
	return n
}

func (n *thresholdLogSpan) LogFields(fields ...otlog.Field) {

}

func (n *thresholdLogSpan) LogKV(keyVals ...interface{}) {

}

func (n *thresholdLogSpan) Finish() {
	n.duration = time.Now().Sub(n.startTime)

	n.totalServerDuration += n.serverDuration
	if n.opName == "dispatch" {
		n.totalDispatchDuration += n.duration
		n.lastDispatchPeer = n.peerAddress
		n.lastDispatchDuration = n.duration
	}
	if n.opName == "encode" {
		n.totalEncodeDuration += n.duration
	}
	if n.opName == "decode" {
		n.totalDecodeDuration += n.duration
	}

	if n.parent != nil {
		n.parent.totalServerDuration += n.totalServerDuration
		n.parent.totalDispatchDuration += n.totalDispatchDuration
		n.parent.totalEncodeDuration += n.totalEncodeDuration
		n.parent.totalDecodeDuration += n.totalDecodeDuration
		if n.lastDispatchPeer != "" || n.lastDispatchDuration > 0 {
			n.parent.lastDispatchPeer = n.lastDispatchPeer
			n.parent.lastDispatchDuration = n.lastDispatchDuration
		}
		if n.lastOperationID != "" {
			n.parent.lastOperationID = n.lastOperationID
		}
		if n.lastLocalID != "" {
			n.parent.lastLocalID = n.lastLocalID
		}
		if n.documentKey != "" {
			n.parent.documentKey = n.documentKey
		}
	}

	if n.serviceName != "" {
		n.tracer.recordOp(n)
	}
}

func (n *thresholdLogSpan) FinishWithOptions(opts opentracing.FinishOptions) {
	n.Finish()
}

func (n *thresholdLogSpan) SetOperationName(operationName string) opentracing.Span {
	n.opName = operationName
	return n
}

func (n *thresholdLogSpan) Tracer() opentracing.Tracer {
	return n.tracer
}

func (n *thresholdLogSpan) LogEvent(event string) {

}

func (n *thresholdLogSpan) LogEventWithPayload(event string, payload interface{}) {

}

func (n *thresholdLogSpan) Log(data opentracing.LogData) {

}

type thresholdLogSpanContext struct {
	span *thresholdLogSpan
}

func (n *thresholdLogSpanContext) ForeachBaggageItem(handler func(k, v string) bool) {
}
