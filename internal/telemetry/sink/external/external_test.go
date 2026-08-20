package externalsink_test

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/SergioLacerda/strategist-skill/internal/telemetry"
	externalsink "github.com/SergioLacerda/strategist-skill/internal/telemetry/sink/external"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type recordingSink struct {
	events []telemetry.Event
	err    error
}

func (r *recordingSink) Emit(_ context.Context, event telemetry.Event) error {
	r.events = append(r.events, event)
	return r.err
}

func TestSink_Emit_ForwardsToInner(t *testing.T) {
	t.Parallel()
	rec := &recordingSink{}
	s := externalsink.New(rec)

	event := telemetry.Event{Name: "strategist.governance.decision", Timestamp: time.Now()}
	require.NoError(t, s.Emit(context.Background(), event))

	require.Len(t, rec.events, 1)
	assert.Equal(t, event.Name, rec.events[0].Name)
}

func TestSink_Emit_NilForwardIsNoop(t *testing.T) {
	t.Parallel()
	s := externalsink.New(nil)
	err := s.Emit(context.Background(), telemetry.Event{Name: "x", Timestamp: time.Now()})
	require.NoError(t, err)
}

func TestSink_Emit_PropagatesInnerError(t *testing.T) {
	t.Parallel()
	rec := &recordingSink{err: errors.New("boom")}
	s := externalsink.New(rec)
	err := s.Emit(context.Background(), telemetry.Event{Name: "x", Timestamp: time.Now()})
	require.ErrorContains(t, err, "boom")
}
