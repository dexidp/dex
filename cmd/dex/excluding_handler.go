package main

import (
	"context"
	"log/slog"
)

// excludingHandler is an slog.Handler wrapper that drops log attributes
// whose keys match a configured set. This allows PII fields like email,
// username, or groups to be redacted at the logger level rather than
// requiring per-callsite suppression logic.
type excludingHandler struct {
	inner   slog.Handler
	exclude map[string]bool
}

func newExcludingHandler(inner slog.Handler, fields []string) slog.Handler {
	if len(fields) == 0 {
		return inner
	}
	m := make(map[string]bool, len(fields))
	for _, f := range fields {
		m[f] = true
	}
	return &excludingHandler{inner: inner, exclude: m}
}

func (h *excludingHandler) Enabled(ctx context.Context, level slog.Level) bool {
	return h.inner.Enabled(ctx, level)
}

func (h *excludingHandler) Handle(ctx context.Context, record slog.Record) error {
	// Rebuild the record without excluded attributes.
	filtered := slog.NewRecord(record.Time, record.Level, record.Message, record.PC)
	record.Attrs(func(a slog.Attr) bool {
		if !h.exclude[a.Key] {
			filtered.AddAttrs(a)
		}
		return true
	})
	return h.inner.Handle(ctx, filtered)
}

func (h *excludingHandler) WithAttrs(attrs []slog.Attr) slog.Handler {
	var kept []slog.Attr
	for _, a := range attrs {
		if !h.exclude[a.Key] {
			kept = append(kept, a)
		}
	}
	return &excludingHandler{inner: h.inner.WithAttrs(kept), exclude: h.exclude}
}

func (h *excludingHandler) WithGroup(name string) slog.Handler {
	return &excludingHandler{inner: h.inner.WithGroup(name), exclude: h.exclude}
}
