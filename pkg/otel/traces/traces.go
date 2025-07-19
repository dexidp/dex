package traces

import (
	"context"
	"fmt"
	"net/http"
	"strconv"
	"strings"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracegrpc"
	"go.opentelemetry.io/otel/propagation"
	"go.opentelemetry.io/otel/sdk/resource"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	"go.opentelemetry.io/otel/trace"
	"google.golang.org/grpc"
)

func InitTracerProvider(ctx context.Context, res *resource.Resource, conn *grpc.ClientConn, samplerStr string) (func(context.Context) error, error) {
	// Set up a trace exporter
	traceExporter, err := otlptracegrpc.New(ctx, otlptracegrpc.WithGRPCConn(conn))
	if err != nil {
		return nil, fmt.Errorf("failed to create trace exporter: %w", err)
	}

	// Parse sampler
	var sampler sdktrace.Sampler
	switch samplerStr {
	case "always_on":
		sampler = sdktrace.AlwaysSample()
	case "always_off":
		sampler = sdktrace.NeverSample()
	default: // e.g., "traceidratio:0.5"
		if strings.HasPrefix(samplerStr, "traceidratio:") {
			ratioStr := strings.TrimPrefix(samplerStr, "traceidratio:")
			ratio, err := strconv.ParseFloat(ratioStr, 64)
			if err != nil {
				return nil, fmt.Errorf("invalid sampler ratio: %w", err)
			}
			sampler = sdktrace.ParentBased(sdktrace.TraceIDRatioBased(ratio))
		} else {
			sampler = sdktrace.AlwaysSample() // Fallback
		}
	}

	// Register the trace exporter with a TracerProvider, using a batch
	// span processor to aggregate spans before export.
	bsp := sdktrace.NewBatchSpanProcessor(traceExporter)
	tracerProvider := sdktrace.NewTracerProvider(
		sdktrace.WithSampler(sampler),
		sdktrace.WithResource(res),
		sdktrace.WithSpanProcessor(bsp),
	)
	otel.SetTracerProvider(tracerProvider)

	// Set global propagator to tracecontext (the default is no-op).
	otel.SetTextMapPropagator(propagation.TraceContext{})

	// Shutdown will flush any remaining spans and shut down the exporter.
	return tracerProvider.Shutdown, nil
}

const dexLibraryName = "github.com/dexidp/dex"

func InstrumentationTracer(ctx context.Context, spanName string) (context.Context, trace.Span) {
	return trace.SpanFromContext(ctx).TracerProvider().Tracer(dexLibraryName).Start(ctx, spanName)
}

func InstrumentHandler(r *http.Request) (context.Context, trace.Span) {
	ctx := r.Context()
	span := trace.SpanFromContext(ctx)
	span.SetName(fmt.Sprintf("%s %s", r.Method, r.URL.Path))
	return ctx, span
}
