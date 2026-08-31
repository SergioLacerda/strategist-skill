package telemetry

import (
	"sort"
	"sync"
	"time"
)

// sequenceByRun holds the last Sequence number handed out per RunID.
// Package-level, in-memory, per-process state: sufficient for tracking one
// mission-runtime execution's own event stream (this envelope's scope) —
// not a durable or cross-process sequence source.
var (
	sequenceMu    sync.Mutex
	sequenceByRun = map[string]uint64{}
)

// NextSequence returns the next monotonically increasing Sequence number
// for runID, starting at 1. Safe for concurrent use from multiple
// goroutines emitting events for the same run.
func NextSequence(runID string) uint64 {
	sequenceMu.Lock()
	defer sequenceMu.Unlock()
	sequenceByRun[runID]++
	return sequenceByRun[runID]
}

// NewEvent builds an Event with Timestamp, RunID, Sequence, and
// SchemaVersion populated automatically — Sequence via
// NextSequence(runID), SchemaVersion via CurrentEventSchemaVersion. Body,
// Attributes, TraceID, and SpanID are left for the caller to set on the
// returned value (Event is a plain struct, so this is a normal field
// assignment, not a builder chain).
//
// NewEvent is additive sugar for new call sites; it does not replace bare
// Event{...} literals. Existing call sites (e.g.
// internal/plugins/conformance.PluginTelemetryEvent) keep compiling and
// behaving exactly as before — their events simply carry a zero Sequence
// and empty RunID/SchemaVersion, which ValidateSequence treats as
// "no ordering information to validate" rather than an error.
func NewEvent(name string, severity SeverityNumber, runID string, complete bool) Event {
	return Event{
		Name:           name,
		Timestamp:      time.Now().UTC(),
		SeverityNumber: severity,
		RunID:          runID,
		Sequence:       NextSequence(runID),
		SchemaVersion:  CurrentEventSchemaVersion,
		Complete:       complete,
	}
}

// Gap describes a break in an event stream's Sequence numbering for one
// RunID: evidence that at least one event was assigned a Sequence number
// (via NextSequence/NewEvent) but is absent from the slice under
// validation — e.g. dropped by a sink, or lost before it could be
// persisted. This is what lets a downstream reader distinguish "this
// phase's event was lost" (a Gap exists between two observed sequence
// numbers) from "this phase never ran" (no event, no sequence number ever
// assigned, therefore no gap).
type Gap struct {
	// RunID is the run in which the gap was found.
	RunID string
	// LastSeen is the last Sequence number observed before the gap.
	LastSeen uint64
	// NextSeen is the next Sequence number observed after the gap.
	NextSeen uint64
}

// Missing returns how many sequence numbers are unaccounted for between
// g.LastSeen and g.NextSeen.
func (g Gap) Missing() uint64 {
	if g.NextSeen <= g.LastSeen+1 {
		return 0
	}
	return g.NextSeen - g.LastSeen - 1
}

// ValidateSequence reports every gap in events' Sequence numbering, one Gap
// per break. events need not be pre-sorted or pre-filtered to a single
// RunID — ValidateSequence groups by RunID and sorts each group by Sequence
// before checking, so passing an unfiltered, out-of-order slice (e.g. as
// read back from a jsonl event log) degrades gracefully instead of
// producing spurious gaps at RunID boundaries. Events with Sequence == 0
// (never assigned — built via a bare Event{...} literal rather than
// NewEvent) are ignored: they carry no ordering information to validate.
// Duplicate sequence numbers within a run are tolerated and do not count as
// gaps. Returned gaps are ordered by RunID, then by position in the run.
func ValidateSequence(events []Event) []Gap {
	byRun := make(map[string][]Event)
	for _, e := range events {
		if e.Sequence == 0 {
			continue
		}
		byRun[e.RunID] = append(byRun[e.RunID], e)
	}

	runIDs := make([]string, 0, len(byRun))
	for id := range byRun {
		runIDs = append(runIDs, id)
	}
	sort.Strings(runIDs)

	var gaps []Gap
	for _, id := range runIDs {
		seq := byRun[id]
		sort.Slice(seq, func(i, j int) bool { return seq[i].Sequence < seq[j].Sequence })
		for i := 1; i < len(seq); i++ {
			prev, cur := seq[i-1].Sequence, seq[i].Sequence
			if cur > prev+1 {
				gaps = append(gaps, Gap{RunID: id, LastSeen: prev, NextSeen: cur})
			}
		}
	}
	return gaps
}
