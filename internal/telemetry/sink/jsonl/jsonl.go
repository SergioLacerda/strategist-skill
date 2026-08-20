// Package jsonlsink provides a telemetry.EventSink that appends each Event
// as one JSON line to a file, reusing internal/telemetry's existing
// atomic-append-with-flock discipline (the same pattern already proven by
// outcomes.jsonl, ADR-0004) rather than a separate, unproven implementation.
package jsonlsink

import (
	"context"

	"github.com/SergioLacerda/strategist-skill/internal/telemetry"
)

// Sink appends each Event as a JSON line to Path.
type Sink struct {
	Path string
}

// New returns a Sink writing to path.
func New(path string) Sink { return Sink{Path: path} }

// Emit appends event to s.Path.
func (s Sink) Emit(_ context.Context, event telemetry.Event) error {
	return telemetry.AppendEventLine(s.Path, event)
}

var _ telemetry.EventSink = Sink{}
