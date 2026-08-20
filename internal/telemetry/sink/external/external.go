// Package externalsink provides a telemetry.EventSink that forwards each
// Event to an injected downstream sink — the pluggability point for routing
// events to an external observability/governance system, selected by
// internal/telemetry/sink.Select when a governancebridge.GovernanceBridge is
// configured. Strategist ships no concrete external transport itself;
// callers inject one (e.g. an OTLP exporter pointed at the external
// system, or an HTTP client sink they own).
package externalsink

import (
	"context"

	"github.com/SergioLacerda/strategist-skill/internal/telemetry"
)

// Sink forwards each Event to Forward. A nil Forward makes Emit a no-op —
// callers that only need the "external governance is present" branch of the
// selection policy without a concrete transport yet can use the zero value.
type Sink struct {
	Forward telemetry.EventSink
}

// New returns a Sink forwarding to forward.
func New(forward telemetry.EventSink) Sink { return Sink{Forward: forward} }

// Emit forwards event to s.Forward, or does nothing if s.Forward is nil.
func (s Sink) Emit(ctx context.Context, event telemetry.Event) error {
	if s.Forward == nil {
		return nil
	}
	return s.Forward.Emit(ctx, event)
}

var _ telemetry.EventSink = Sink{}
