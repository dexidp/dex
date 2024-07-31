package main

import (
	"context"
	"fmt"
	"log/slog"
	"os"
	"strings"

	"github.com/dexidp/dex/server"
)

var logFormats = []string{"json", "text"}

func newLogger(level slog.Level, format string) (*slog.Logger, error) {
	var handler slog.Handler
	switch strings.ToLower(format) {
	case "", "text":
		handler = slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{
			Level: level,
		})
	case "json":
		handler = slog.NewJSONHandler(os.Stderr, &slog.HandlerOptions{
			Level: level,
		})
	default:
		return nil, fmt.Errorf("log format is not one of the supported values (%s): %s", strings.Join(logFormats, ", "), format)
	}

	return slog.New(newRequestContextHandler(handler)), nil
}

var _ slog.Handler = requestContextHandler{}

type requestContextHandler struct {
	handler slog.Handler
}

func newRequestContextHandler(handler slog.Handler) slog.Handler {
	return requestContextHandler{
		handler: handler,
	}
}

func (h requestContextHandler) Enabled(ctx context.Context, level slog.Level) bool {
	return h.handler.Enabled(ctx, level)
}

func (h requestContextHandler) Handle(ctx context.Context, record slog.Record) error {
	if v, ok := ctx.Value(server.RequestKeyRemoteIP).(string); ok {
		record.AddAttrs(slog.String(string(server.RequestKeyRemoteIP), v))
	}

	if v, ok := ctx.Value(server.RequestKeyRequestID).(string); ok {
		record.AddAttrs(slog.String(string(server.RequestKeyRequestID), v))
	}

	return h.handler.Handle(ctx, record)
}

func (h requestContextHandler) WithAttrs(attrs []slog.Attr) slog.Handler {
	return requestContextHandler{h.handler.WithAttrs(attrs)}
}

func (h requestContextHandler) WithGroup(name string) slog.Handler {
	return h.handler.WithGroup(name)
}
