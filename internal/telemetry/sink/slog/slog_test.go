package slogsink_test

import (
	"bytes"
	"context"
	"log/slog"
	"testing"
	"time"

	"github.com/SergioLacerda/strategist-skill/internal/telemetry"
	slogsink "github.com/SergioLacerda/strategist-skill/internal/telemetry/sink/slog"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestSink_Emit_LogsThroughDefaultSlog(t *testing.T) {
	var buf bytes.Buffer
	prev := slog.Default()
	slog.SetDefault(slog.New(slog.NewTextHandler(&buf, nil)))
	t.Cleanup(func() { slog.SetDefault(prev) })

	s := slogsink.New()
	err := s.Emit(context.Background(), telemetry.Event{
		Name:           "strategist.governance.decision",
		Timestamp:      time.Now(),
		SeverityNumber: telemetry.SeverityWarn,
		Body:           "mandate missing",
		Attributes:     map[string]any{telemetry.AttrEventContractID: "compile-domain"},
	})
	require.NoError(t, err)

	out := buf.String()
	assert.Contains(t, out, "mandate missing")
	assert.Contains(t, out, "event.name=strategist.governance.decision")
	assert.Contains(t, out, "level=WARN")
}

func TestSink_Emit_SeverityLevelMapping(t *testing.T) {
	tests := []struct {
		name     string
		severity telemetry.SeverityNumber
		wantText string
	}{
		{"error maps to ERROR", telemetry.SeverityError, "level=ERROR"},
		{"warn maps to WARN", telemetry.SeverityWarn, "level=WARN"},
		{"info maps to INFO", telemetry.SeverityInfo, "level=INFO"},
		{"debug maps to DEBUG", telemetry.SeverityDebug, "level=DEBUG"},
		{"below debug still maps to DEBUG (default branch)", telemetry.SeverityNumber(0), "level=DEBUG"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var buf bytes.Buffer
			prev := slog.Default()
			slog.SetDefault(slog.New(slog.NewTextHandler(&buf, &slog.HandlerOptions{Level: slog.LevelDebug})))
			t.Cleanup(func() { slog.SetDefault(prev) })

			s := slogsink.New()
			err := s.Emit(context.Background(), telemetry.Event{
				Name:           "strategist.test.event",
				SeverityNumber: tt.severity,
				Body:           "body",
			})
			require.NoError(t, err)
			assert.Contains(t, buf.String(), tt.wantText)
		})
	}
}

func TestSink_Emit_IncludesSpanIDAndAttributes(t *testing.T) {
	var buf bytes.Buffer
	prev := slog.Default()
	slog.SetDefault(slog.New(slog.NewTextHandler(&buf, nil)))
	t.Cleanup(func() { slog.SetDefault(prev) })

	s := slogsink.New()
	err := s.Emit(context.Background(), telemetry.Event{
		Name:           "strategist.test.event",
		SeverityNumber: telemetry.SeverityInfo,
		Body:           "body",
		TraceID:        "trace-1",
		SpanID:         "span-1",
		Attributes:     map[string]any{"key": "value"},
	})
	require.NoError(t, err)

	out := buf.String()
	assert.Contains(t, out, "trace_id=trace-1")
	assert.Contains(t, out, "span_id=span-1")
	assert.Contains(t, out, "key=value")
}
