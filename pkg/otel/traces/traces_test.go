package traces

import (
	"context"
	"net/http"
	"net/url"
	"testing"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/sdk/resource"
	tracesdk "go.opentelemetry.io/otel/sdk/trace"
	"go.opentelemetry.io/otel/sdk/trace/tracetest"
	"go.opentelemetry.io/otel/trace"
	nooptrace "go.opentelemetry.io/otel/trace/noop"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

func Test_initTracerProvider(t *testing.T) {
	ctx := context.Background()
	res := resource.Default()
	conn, _ := grpc.NewClient("localhost:4317",
		// Note the use of insecure transport here. TLS is recommended in production.
		grpc.WithTransportCredentials(insecure.NewCredentials()),
	) // Assume success for test

	type args struct {
		ctx        context.Context
		res        *resource.Resource
		conn       *grpc.ClientConn
		samplerStr string
	}
	tests := []struct {
		name    string
		args    args
		wantErr bool
	}{
		{
			name:    "valid always_on",
			args:    args{ctx: ctx, res: res, conn: conn, samplerStr: "always_on"},
			wantErr: false,
		},
		{
			name:    "valid always_off",
			args:    args{ctx: ctx, res: res, conn: conn, samplerStr: "always_off"},
			wantErr: false,
		},
		{
			name:    "valid traceidratio",
			args:    args{ctx: ctx, res: res, conn: conn, samplerStr: "traceidratio:0.5"},
			wantErr: false,
		},
		{
			name:    "invalid ratio",
			args:    args{ctx: ctx, res: res, conn: conn, samplerStr: "traceidratio:invalid"},
			wantErr: true,
		},
		{
			name:    "fallback sampler",
			args:    args{ctx: ctx, res: res, conn: conn, samplerStr: "unknown"},
			wantErr: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := InitTracerProvider(tt.args.ctx, tt.args.res, tt.args.conn, tt.args.samplerStr)
			if (err != nil) != tt.wantErr {
				t.Errorf("initTracerProvider() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !tt.wantErr && got == nil {
				t.Errorf("initTracerProvider() got = nil, want non-nil shutdown func")
			}
		})
	}
}

func TestInstrumentationTracer(t *testing.T) {
	type args struct {
		ctx      context.Context
		spanName string
	}
	tests := []struct {
		name          string
		args          args
		wantScopeName string
		wantSpanName  string
	}{
		{
			name: "basic tracer",
			args: args{
				ctx:      context.Background(),
				spanName: "test-span",
			},
			wantScopeName: dexLibraryName,
			wantSpanName:  "test-span",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			recorder := tracetest.NewSpanRecorder()
			provider := tracesdk.NewTracerProvider(tracesdk.WithSpanProcessor(recorder))
			otel.SetTracerProvider(provider)
			defer otel.SetTracerProvider(nooptrace.NewTracerProvider()) // Cleanup
			ctx, _ := provider.Tracer("test").Start(tt.args.ctx, "initial")

			gotCtx, span := InstrumentationTracer(ctx, tt.args.spanName)
			span.End()

			spans := recorder.Ended()
			if len(spans) != 1 {
				t.Errorf("expected 1 span, got %d", len(spans))
			}
			gotSpan := spans[0]

			if gotSpan.InstrumentationScope().Name != tt.wantScopeName {
				t.Errorf("InstrumentationTracer() scope name = %v, want %v", gotSpan.InstrumentationScope().Name, tt.wantScopeName)
			}
			if gotSpan.Name() != tt.wantSpanName {
				t.Errorf("InstrumentationTracer() span name = %v, want %v", gotSpan.Name(), tt.wantSpanName)
			}

			// Check that the returned context contains the new span
			returnedSpan := trace.SpanFromContext(gotCtx)
			if returnedSpan.SpanContext().TraceID() != gotSpan.SpanContext().TraceID() {
				t.Errorf("InstrumentationTracer() returned context does not contain the created span")
			}
		})
	}
}

func TestInstrumentHandler(t *testing.T) {
	type args struct {
		r *http.Request
	}
	tests := []struct {
		name         string
		args         args
		wantSpanName string
	}{
		{
			name: "basic handler",
			args: args{
				r: &http.Request{Method: "GET", URL: &url.URL{Path: "/test/path"}},
			},
			wantSpanName: "GET /test/path",
		},
		{
			name: "post method",
			args: args{
				r: &http.Request{Method: "POST", URL: &url.URL{Path: "/api/create"}},
			},
			wantSpanName: "POST /api/create",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			recorder := tracetest.NewSpanRecorder()
			provider := tracesdk.NewTracerProvider(tracesdk.WithSpanProcessor(recorder))
			otel.SetTracerProvider(provider)
			defer otel.SetTracerProvider(nooptrace.NewTracerProvider()) // Cleanup

			// Start an initial span and attach to request context
			ctx, initialSpan := provider.Tracer("test").Start(context.Background(), "initial")
			tt.args.r = tt.args.r.WithContext(ctx)

			// Call the function - it updates the existing span's name
			gotCtx, returnedSpan := InstrumentHandler(tt.args.r)

			// End the span
			initialSpan.End()

			spans := recorder.Ended()
			if len(spans) != 1 {
				t.Errorf("expected 1 span, got %d", len(spans))
			}
			gotSpan := spans[0]

			if gotSpan.Name() != tt.wantSpanName {
				t.Errorf("InstrumentHandler() updated span name = %v, want %v", gotSpan.Name(), tt.wantSpanName)
			}

			// Check returned context is the same as request's (since it doesn't create new ctx)
			if gotCtx != tt.args.r.Context() {
				t.Errorf("InstrumentHandler() returned different context")
			}

			// Check returned span is the same as the updated one
			if returnedSpan.SpanContext().SpanID() != gotSpan.SpanContext().SpanID() {
				t.Errorf("InstrumentHandler() returned wrong span")
			}
		})
	}
}
