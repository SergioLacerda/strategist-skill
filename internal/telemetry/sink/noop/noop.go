// Package noop provides a zero-overhead telemetry.EventSink that discards
// every event — the sink selected when no telemetry destination is
// configured at all (internal/telemetry/sink.Select's first branch).
package noop

import (
	"context"

	"github.com/SergioLacerda/strategist-skill/internal/telemetry"
)

// Sink discards every Event.
type Sink struct{}

// New returns a Sink.
func New() Sink { return Sink{} }

// Emit discards event and always returns nil.
func (Sink) Emit(context.Context, telemetry.Event) error { return nil }

var _ telemetry.EventSink = Sink{}
