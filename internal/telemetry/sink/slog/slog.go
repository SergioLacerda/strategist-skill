// Package slogsink provides a telemetry.EventSink that logs each Event
// through the process's default log/slog logger — the local-only fallback
// selected when no OTel endpoint is configured (internal/telemetry/sink.Select).
package slogsink

import (
	"context"
	"log/slog"

	"github.com/SergioLacerda/strategist-skill/internal/telemetry"
)

// Sink logs each Event via log/slog at a level derived from event.SeverityNumber.
type Sink struct{}

// New returns a Sink.
func New() Sink { return Sink{} }

// Emit logs event through slog. It never returns an error — slog itself has
// no failure mode Strategist can observe.
func (Sink) Emit(ctx context.Context, event telemetry.Event) error {
	attrs := make([]any, 0, len(event.Attributes)*2+4)
	attrs = append(attrs, "event.name", event.Name)
	if event.TraceID != "" {
		attrs = append(attrs, "trace_id", event.TraceID)
	}
	if event.SpanID != "" {
		attrs = append(attrs, "span_id", event.SpanID)
	}
	for k, v := range event.Attributes {
		attrs = append(attrs, k, v)
	}
	slog.Log(ctx, severityToSlogLevel(event.SeverityNumber), event.Body, attrs...)
	return nil
}

// severityToSlogLevel maps the OTel SeverityNumber scale onto slog's four
// levels — the closest fit without inventing a parallel level taxonomy.
func severityToSlogLevel(s telemetry.SeverityNumber) slog.Level {
	switch {
	case s >= telemetry.SeverityError:
		return slog.LevelError
	case s >= telemetry.SeverityWarn:
		return slog.LevelWarn
	case s >= telemetry.SeverityInfo:
		return slog.LevelInfo
	default:
		return slog.LevelDebug
	}
}

var _ telemetry.EventSink = Sink{}
