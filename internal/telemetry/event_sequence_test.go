package telemetry_test

import (
	"sync"
	"testing"

	"github.com/SergioLacerda/strategist-skill/internal/telemetry"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewEvent_PopulatesEnvelopeFields(t *testing.T) {
	t.Parallel()
	runID := "run-" + t.Name()
	ev := telemetry.NewEvent("strategist.discovery.done", telemetry.SeverityInfo, runID, true)

	assert.Equal(t, "strategist.discovery.done", ev.Name)
	assert.Equal(t, telemetry.SeverityInfo, ev.SeverityNumber)
	assert.Equal(t, runID, ev.RunID)
	assert.Equal(t, telemetry.CurrentEventSchemaVersion, ev.SchemaVersion)
	assert.True(t, ev.Complete)
	assert.False(t, ev.Timestamp.IsZero())
	assert.Equal(t, uint64(1), ev.Sequence, "first event for a fresh run_id must start at sequence 1")
}

func TestNewEvent_SequenceIncrementsPerRunID(t *testing.T) {
	t.Parallel()
	runID := "run-" + t.Name()

	first := telemetry.NewEvent("strategist.discovery.started", telemetry.SeverityInfo, runID, false)
	second := telemetry.NewEvent("strategist.discovery.done", telemetry.SeverityInfo, runID, true)
	third := telemetry.NewEvent("strategist.refinement.started", telemetry.SeverityInfo, runID, false)

	assert.Equal(t, uint64(1), first.Sequence)
	assert.Equal(t, uint64(2), second.Sequence)
	assert.Equal(t, uint64(3), third.Sequence)

	// A different run_id gets its own independent counter starting at 1.
	otherRun := telemetry.NewEvent("strategist.discovery.started", telemetry.SeverityInfo, "other-"+runID, false)
	assert.Equal(t, uint64(1), otherRun.Sequence)
}

func TestNextSequence_ConcurrentCallsProduceUniqueSequentialNumbers(t *testing.T) {
	t.Parallel()
	runID := "run-" + t.Name()
	const n = 100

	results := make([]uint64, n)
	var wg sync.WaitGroup
	wg.Add(n)
	for i := 0; i < n; i++ {
		go func(i int) {
			defer wg.Done()
			results[i] = telemetry.NextSequence(runID)
		}(i)
	}
	wg.Wait()

	seen := make(map[uint64]bool, n)
	for _, seq := range results {
		require.False(t, seen[seq], "sequence %d assigned more than once", seq)
		seen[seq] = true
		require.GreaterOrEqual(t, seq, uint64(1))
		require.LessOrEqual(t, seq, uint64(n))
	}
	assert.Len(t, seen, n)
}

func TestValidateSequence_NoGapsWhenComplete(t *testing.T) {
	t.Parallel()
	runID := "run-" + t.Name()
	events := []telemetry.Event{
		telemetry.NewEvent("strategist.discovery.started", telemetry.SeverityInfo, runID, false),
		telemetry.NewEvent("strategist.discovery.done", telemetry.SeverityInfo, runID, true),
		telemetry.NewEvent("strategist.refinement.started", telemetry.SeverityInfo, runID, false),
		telemetry.NewEvent("strategist.refinement.done", telemetry.SeverityInfo, runID, true),
	}

	gaps := telemetry.ValidateSequence(events)
	assert.Empty(t, gaps)
}

func TestValidateSequence_DetectsDeliberatelyDroppedEvent(t *testing.T) {
	t.Parallel()
	runID := "run-" + t.Name()
	all := []telemetry.Event{
		telemetry.NewEvent("strategist.discovery.started", telemetry.SeverityInfo, runID, false),
		telemetry.NewEvent("strategist.discovery.done", telemetry.SeverityInfo, runID, true),
		telemetry.NewEvent("strategist.refinement.started", telemetry.SeverityInfo, runID, false),
		telemetry.NewEvent("strategist.refinement.done", telemetry.SeverityInfo, runID, true),
	}
	// Simulate a lost event: refinement.started (sequence 3) never reached
	// the sink, so the validated slice jumps from 2 straight to 4.
	dropped := []telemetry.Event{all[0], all[1], all[3]}

	gaps := telemetry.ValidateSequence(dropped)
	require.Len(t, gaps, 1)
	assert.Equal(t, runID, gaps[0].RunID)
	assert.Equal(t, uint64(2), gaps[0].LastSeen)
	assert.Equal(t, uint64(4), gaps[0].NextSeen)
	assert.Equal(t, uint64(1), gaps[0].Missing())
}

func TestValidateSequence_IgnoresEventsWithNoSequenceAssigned(t *testing.T) {
	t.Parallel()
	// Bare Event{} literals (pre-existing call sites) carry Sequence == 0
	// and must not be reported as gaps — they carry no ordering
	// information, which is the "phase never ran"/legacy-event case, not
	// the "event was lost" case.
	events := []telemetry.Event{
		{Name: "strategist.legacy.event"},
		{Name: "strategist.legacy.event.2"},
	}
	assert.Empty(t, telemetry.ValidateSequence(events))
}

func TestValidateSequence_HandlesMultipleRunsIndependently(t *testing.T) {
	t.Parallel()
	runA := "run-a-" + t.Name()
	runB := "run-b-" + t.Name()

	eventsA := []telemetry.Event{
		telemetry.NewEvent("a.started", telemetry.SeverityInfo, runA, false),
		telemetry.NewEvent("a.done", telemetry.SeverityInfo, runA, true),
	}
	// runB has a real gap: its middle event (sequence 2) is dropped, so the
	// validated slice for runB jumps from sequence 1 straight to 3.
	bFirst := telemetry.NewEvent("b.started", telemetry.SeverityInfo, runB, false)
	_ = telemetry.NewEvent("b.running", telemetry.SeverityInfo, runB, false)
	bThird := telemetry.NewEvent("b.done", telemetry.SeverityInfo, runB, true)

	gaps := telemetry.ValidateSequence(append(eventsA, bFirst, bThird))
	require.Len(t, gaps, 1)
	assert.Equal(t, runB, gaps[0].RunID)
}

func TestGap_MissingCountsUnaccountedSequenceNumbers(t *testing.T) {
	t.Parallel()
	g := telemetry.Gap{RunID: "r", LastSeen: 5, NextSeen: 9}
	assert.Equal(t, uint64(3), g.Missing())

	adjacent := telemetry.Gap{RunID: "r", LastSeen: 5, NextSeen: 6}
	assert.Equal(t, uint64(0), adjacent.Missing())
}
