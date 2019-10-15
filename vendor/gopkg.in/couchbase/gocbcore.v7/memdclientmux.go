package gocbcore

import (
	"container/list"
	"fmt"
)

type memdGetClientFunc func(hostPort string) (*memdClient, error)

type memdClientMux struct {
	pipelines []*memdPipeline
	deadPipe  *memdPipeline
}

func newMemdClientMux(hostPorts []string, poolSize int, queueSize int, getClientFn memdGetClientFunc) *memdClientMux {
	mux := &memdClientMux{}

	for _, hostPort := range hostPorts {
		hostPort := hostPort

		getCurClientFn := func() (*memdClient, error) {
			return getClientFn(hostPort)
		}
		pipeline := newPipeline(hostPort, poolSize, queueSize, getCurClientFn)

		mux.pipelines = append(mux.pipelines, pipeline)
	}

	mux.deadPipe = newDeadPipeline(queueSize)

	return mux
}

func (mux *memdClientMux) NumPipelines() int {
	return len(mux.pipelines)
}

func (mux *memdClientMux) GetPipeline(index int) *memdPipeline {
	if index < 0 || index >= len(mux.pipelines) {
		return mux.deadPipe
	}
	return mux.pipelines[index]
}

func (mux *memdClientMux) Start() {
	// Initialize new pipelines
	for _, pipeline := range mux.pipelines {
		pipeline.StartClients()
	}
}

func (mux *memdClientMux) Takeover(oldMux *memdClientMux) {
	oldPipelines := list.New()

	// Gather all our old pipelines up for takeover and what not
	if oldMux != nil {
		for _, pipeline := range oldMux.pipelines {
			oldPipelines.PushBack(pipeline)
		}
	}

	// Build a function to find an existing pipeline
	stealPipeline := func(address string) *memdPipeline {
		for e := oldPipelines.Front(); e != nil; e = e.Next() {
			pipeline, ok := e.Value.(*memdPipeline)
			if !ok {
				logErrorf("Failed to cast old pipeline")
				continue
			}

			if pipeline.Address() == address {
				oldPipelines.Remove(e)
				return pipeline
			}
		}

		return nil
	}

	// Initialize new pipelines (possibly with a takeover)
	for _, pipeline := range mux.pipelines {
		oldPipeline := stealPipeline(pipeline.Address())
		if oldPipeline != nil {
			pipeline.Takeover(oldPipeline)
		}

		pipeline.StartClients()
	}

	// Shut down any pipelines that were not taken over
	for e := oldPipelines.Front(); e != nil; e = e.Next() {
		pipeline, ok := e.Value.(*memdPipeline)
		if !ok {
			logErrorf("Failed to cast old pipeline")
			continue
		}

		err := pipeline.Close()
		if err != nil {
			logErrorf("Failed to properly close abandoned pipeline (%s)", err)
		}
	}

	if oldMux.deadPipe != nil {
		err := oldMux.deadPipe.Close()
		if err != nil {
			logErrorf("Failed to properly close abandoned dead pipe (%s)", err)
		}
	}
}

func (mux *memdClientMux) Close() error {
	var errs MultiError

	for _, pipeline := range mux.pipelines {
		err := pipeline.Close()
		if err != nil {
			errs.add(err)
		}
	}

	if mux.deadPipe != nil {
		err := mux.deadPipe.Close()
		if err != nil {
			errs.add(err)
		}
	}

	return errs.get()
}

// Drain will drain all requests from this muxers pipelines.  You must have
// called Takeover against this or Close on this muxer before invoking this...
func (mux *memdClientMux) Drain(cb func(*memdQRequest)) {
	for _, pipeline := range mux.pipelines {
		logDebugf("Draining queue %+v", pipeline)
		pipeline.Drain(cb)
	}
	if mux.deadPipe != nil {
		mux.deadPipe.Drain(cb)
	}
}

func (mux *memdClientMux) debugString() string {
	var outStr string

	for i, n := range mux.pipelines {
		outStr += fmt.Sprintf("Pipeline %d:\n", i)
		outStr += reindentLog("  ", n.debugString()) + "\n"
	}

	outStr += "Dead Pipeline:\n"
	if mux.deadPipe != nil {
		outStr += reindentLog("  ", mux.deadPipe.debugString()) + "\n"
	} else {
		outStr += "  Disabled\n"
	}

	return outStr
}
