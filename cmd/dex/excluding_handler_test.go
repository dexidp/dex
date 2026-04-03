package main

import (
	"bytes"
	"context"
	"encoding/json"
	"log/slog"
	"testing"
)

func TestExcludingHandler(t *testing.T) {
	tests := []struct {
		name       string
		exclude    []string
		logAttrs   []slog.Attr
		wantKeys   []string
		absentKeys []string
	}{
		{
			name:    "no exclusions",
			exclude: nil,
			logAttrs: []slog.Attr{
				slog.String("email", "user@example.com"),
				slog.String("connector_id", "github"),
			},
			wantKeys: []string{"email", "connector_id"},
		},
		{
			name:    "exclude email",
			exclude: []string{"email"},
			logAttrs: []slog.Attr{
				slog.String("email", "user@example.com"),
				slog.String("connector_id", "github"),
			},
			wantKeys:   []string{"connector_id"},
			absentKeys: []string{"email"},
		},
		{
			name:    "exclude multiple fields",
			exclude: []string{"email", "username", "groups"},
			logAttrs: []slog.Attr{
				slog.String("email", "user@example.com"),
				slog.String("username", "johndoe"),
				slog.String("connector_id", "github"),
				slog.Any("groups", []string{"admin"}),
			},
			wantKeys:   []string{"connector_id"},
			absentKeys: []string{"email", "username", "groups"},
		},
		{
			name:    "exclude non-existent field is harmless",
			exclude: []string{"nonexistent"},
			logAttrs: []slog.Attr{
				slog.String("email", "user@example.com"),
			},
			wantKeys: []string{"email"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var buf bytes.Buffer
			inner := slog.NewJSONHandler(&buf, &slog.HandlerOptions{Level: slog.LevelInfo})
			handler := newExcludingHandler(inner, tt.exclude)
			logger := slog.New(handler)

			attrs := make([]any, 0, len(tt.logAttrs)*2)
			for _, a := range tt.logAttrs {
				attrs = append(attrs, a)
			}
			logger.Info("test message", attrs...)

			var result map[string]any
			if err := json.Unmarshal(buf.Bytes(), &result); err != nil {
				t.Fatalf("failed to parse log output: %v", err)
			}

			for _, key := range tt.wantKeys {
				if _, ok := result[key]; !ok {
					t.Errorf("expected key %q in log output", key)
				}
			}
			for _, key := range tt.absentKeys {
				if _, ok := result[key]; ok {
					t.Errorf("expected key %q to be absent from log output", key)
				}
			}
		})
	}
}

func TestExcludingHandlerWithAttrs(t *testing.T) {
	var buf bytes.Buffer
	inner := slog.NewJSONHandler(&buf, &slog.HandlerOptions{Level: slog.LevelInfo})
	handler := newExcludingHandler(inner, []string{"email"})
	logger := slog.New(handler)

	// Pre-bind an excluded attr via With
	child := logger.With("email", "user@example.com", "connector_id", "github")
	child.Info("login successful")

	var result map[string]any
	if err := json.Unmarshal(buf.Bytes(), &result); err != nil {
		t.Fatalf("failed to parse log output: %v", err)
	}

	if _, ok := result["email"]; ok {
		t.Error("expected email to be excluded from WithAttrs output")
	}
	if _, ok := result["connector_id"]; !ok {
		t.Error("expected connector_id to be present")
	}
}

func TestExcludingHandlerEnabled(t *testing.T) {
	inner := slog.NewJSONHandler(&bytes.Buffer{}, &slog.HandlerOptions{Level: slog.LevelWarn})
	handler := newExcludingHandler(inner, []string{"email"})

	if handler.Enabled(context.Background(), slog.LevelInfo) {
		t.Error("expected Info to be disabled when handler level is Warn")
	}
	if !handler.Enabled(context.Background(), slog.LevelWarn) {
		t.Error("expected Warn to be enabled")
	}
}

func TestExcludingHandlerNilFields(t *testing.T) {
	var buf bytes.Buffer
	inner := slog.NewJSONHandler(&buf, &slog.HandlerOptions{Level: slog.LevelInfo})

	// With nil/empty fields, should return the inner handler directly
	handler := newExcludingHandler(inner, nil)
	if _, ok := handler.(*excludingHandler); ok {
		t.Error("expected nil fields to return inner handler directly, not wrap it")
	}

	handler = newExcludingHandler(inner, []string{})
	if _, ok := handler.(*excludingHandler); ok {
		t.Error("expected empty fields to return inner handler directly, not wrap it")
	}
}
