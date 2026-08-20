// Package otelsink provides a telemetry.EventSink that forwards each Event
// through the process's default log/slog logger. It does not open its own
// OTel exporter connection: telemetry.Init (internal/telemetry/setup.go)
// already bridges the default slog logger to the OTel log pipeline via
// otelslog when an endpoint is configured — this sink simply reuses that
// bridge, the same way the existing [Strategist] progress lines already do.
package otelsink

import (
	"context"
	"log/slog"

	"github.com/SergioLacerda/strategist-skill/internal/telemetry"
)

// Sink forwards each Event through the process's (OTel-bridged) default slog logger.
type Sink struct{}

// New returns a Sink.
func New() Sink { return Sink{} }

// Emit logs event through the default slog logger, carrying event.name and
// trace context as structured attributes so the OTel log pipeline receives
// them as LogRecord attributes.
func (Sink) Emit(ctx context.Context, event telemetry.Event) error {
	attrs := make([]any, 0, len(event.Attributes)*2+6)
	attrs = append(attrs, "event.name", event.Name, "severity_number", int(event.SeverityNumber))
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
