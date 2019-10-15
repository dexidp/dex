package gocb

import (
	"github.com/opentracing/opentracing-go"
)

func tracerAddRef(tracer opentracing.Tracer) {
	if tracer == nil {
		return
	}
	if refTracer, ok := tracer.(interface {
		AddRef() int32
	}); ok {
		refTracer.AddRef()
	}
}

func tracerDecRef(tracer opentracing.Tracer) {
	if tracer == nil {
		return
	}
	if refTracer, ok := tracer.(interface {
		DecRef() int32
	}); ok {
		refTracer.DecRef()
	}
}
