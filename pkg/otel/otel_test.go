package otel

import (
	"context"
	"testing"

	"go.opentelemetry.io/otel/sdk/resource"
	"google.golang.org/grpc"
)

func Test_initConn(t *testing.T) {
	type args struct {
		endpoint string
	}
	tests := []struct {
		name    string
		args    args
		want    *grpc.ClientConn
		wantErr bool
	}{
		{
			name:    "valid endpoint",
			args:    args{endpoint: "localhost:4317"},
			wantErr: false,
		},
		{
			name:    "invalid endpoint format",
			args:    args{endpoint: "invalid_endpoint"},
			wantErr: false, // gRPC.NewClient may not error immediately
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := InitConn(tt.args.endpoint)
			if (err != nil) != tt.wantErr {
				t.Errorf("initConn() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if tt.wantErr {
				if got != nil {
					t.Errorf("initConn() got = %v, want nil", got)
				}
			} else if got == nil {
				t.Errorf("initConn() got = nil, want non-nil")
			}
		})
	}
}

func Test_initLogProvider(t *testing.T) {
	ctx := context.Background()
	res := resource.Default()
	conn, _ := InitConn("localhost:4317") // Assume success for test

	type args struct {
		ctx  context.Context
		res  *resource.Resource
		conn *grpc.ClientConn
	}
	tests := []struct {
		name    string
		args    args
		wantErr bool
	}{
		{
			name:    "valid setup",
			args:    args{ctx: ctx, res: res, conn: conn},
			wantErr: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := InitLogProvider(tt.args.ctx, tt.args.res, tt.args.conn)
			if (err != nil) != tt.wantErr {
				t.Errorf("initLogProvider() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !tt.wantErr && got == nil {
				t.Errorf("initLogProvider() got = nil, want non-nil shutdown func")
			}
		})
	}
}

func Test_initMeterProvider(t *testing.T) {
	ctx := context.Background()
	res := resource.Default()
	conn, _ := InitConn("localhost:4317") // Assume success for test

	type args struct {
		ctx  context.Context
		res  *resource.Resource
		conn *grpc.ClientConn
	}
	tests := []struct {
		name    string
		args    args
		wantErr bool
	}{
		{
			name:    "valid setup",
			args:    args{ctx: ctx, res: res, conn: conn},
			wantErr: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := InitMeterProvider(tt.args.ctx, tt.args.res, tt.args.conn)
			if (err != nil) != tt.wantErr {
				t.Errorf("initMeterProvider() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !tt.wantErr && got == nil {
				t.Errorf("initMeterProvider() got = nil, want non-nil shutdown func")
			}
		})
	}
}
