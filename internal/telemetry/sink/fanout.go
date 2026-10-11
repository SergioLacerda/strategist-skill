package sink

import (
	"context"
	"errors"

	"github.com/SergioLacerda/strategist-skill/internal/telemetry"
)

// Fanout emits each event to every sink. One sink failing never stops the
// others from receiving the event; the joined errors are returned so the
// resilience policy above decides whether the failure is swallowed or strict.
func Fanout(sinks ...telemetry.EventSink) telemetry.EventSink {
	return fanout(sinks)
}

type fanout []telemetry.EventSink

func (f fanout) Emit(ctx context.Context, event telemetry.Event) error {
	var errs []error
	for _, s := range f {
		if err := s.Emit(ctx, event); err != nil {
			errs = append(errs, err)
		}
	}
	return errors.Join(errs...)
}

var _ telemetry.EventSink = fanout(nil)
