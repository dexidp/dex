package gocbcore

import (
	"sync"
	"sync/atomic"
	"time"
	"unsafe"

	"github.com/opentracing/opentracing-go"
)

// The data for a response from a server.  This includes the
// packets data along with some useful meta-data related to
// the response.
type memdQResponse struct {
	memdPacket

	sourceAddr   string
	sourceConnId string
	isInternal   bool
}

type callback func(*memdQResponse, *memdQRequest, error)

// The data for a request that can be queued with a memdqueueconn,
// and can potentially be rerouted to multiple servers due to
// configuration changes.
type memdQRequest struct {
	memdPacket

	// Static routing properties
	ReplicaIdx int
	Callback   callback
	Persistent bool

	// Owner represents the agent which created and currently owns
	// this request.  This is used for specialized routing such as
	// not-my-vbucket and enhanced errors.
	owner *Agent

	// This tracks when the request was dispatched so that we can
	//  properly prioritize older requests to try and meet timeout
	//  requirements.
	dispatchTime time.Time

	// This stores a pointer to the server that currently own
	//   this request.  This allows us to remove it from that list
	//   whenever the request is cancelled.
	queuedWith unsafe.Pointer

	// This stores a pointer to the opList that currently is holding
	//  this request.  This allows us to remove it form that list
	//  whenever the request is cancelled
	waitingIn unsafe.Pointer

	// This keeps track of whether the request has been 'completed'
	//  which is synonymous with the callback having been invoked.
	//  This is an integer to allow us to atomically control it.
	isCompleted uint32

	// This is used to lock access to the request when processing
	// a timeout, a response or spans
	processingLock sync.Mutex

	// This stores the number of times that the item has been
	// retried, and is used for various non-linear retry
	// algorithms.
	retryCount uint32

	RootTraceContext opentracing.SpanContext
	cmdTraceSpan     opentracing.Span
	netTraceSpan     opentracing.Span
}

func (req *memdQRequest) tryCallback(resp *memdQResponse, err error) bool {
	if req.Persistent {
		if atomic.LoadUint32(&req.isCompleted) == 0 {
			req.Callback(resp, req, err)
			return true
		}
	} else {
		if atomic.SwapUint32(&req.isCompleted, 1) == 0 {
			req.Callback(resp, req, err)
			return true
		}
	}

	return false
}

func (req *memdQRequest) isCancelled() bool {
	return atomic.LoadUint32(&req.isCompleted) != 0
}

func (req *memdQRequest) Cancel() bool {
	req.processingLock.Lock()

	if atomic.SwapUint32(&req.isCompleted, 1) != 0 {
		// Someone already completed this request
		req.processingLock.Unlock()
		return false
	}

	queuedWith := (*memdOpQueue)(atomic.LoadPointer(&req.queuedWith))
	if queuedWith != nil {
		queuedWith.Remove(req)
	}

	waitingIn := (*memdClient)(atomic.LoadPointer(&req.waitingIn))
	if waitingIn != nil {
		waitingIn.CancelRequest(req)
	}

	req.owner.cancelReqTrace(req, ErrCancelled)
	req.processingLock.Unlock()
	return true
}
