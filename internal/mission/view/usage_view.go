package view

import (
	"fmt"
	"io"
	"sort"
	"time"

	"github.com/SergioLacerda/strategist-skill/internal/domain"
	"github.com/SergioLacerda/strategist-skill/internal/telemetry"
)

// UsageSchemaVersion identifies the machine-readable `mission usage` output.
const UsageSchemaVersion = "strategist-mission-usage/v1"

// LegacyUsage is the provider-reported legacy ledger shown on its own row. It
// has unknown semantics and is never summed into a phase or the mission total.
type LegacyUsage struct {
	Availability string                              `json:"availability"`
	RecordCount  int                                 `json:"record_count"`
	Records      []telemetry.MissionTokenUsageRecord `json:"records,omitempty"`
}

// ChronologyEntry is one lifecycle marker or usage observation, in time order.
type ChronologyEntry struct {
	At     string `json:"at"`
	Kind   string `json:"kind"`
	Phase  string `json:"phase"`
	RunID  string `json:"run_id,omitempty"`
	Detail string `json:"detail,omitempty"`
}

// UsageView is the `mission usage` projection. Its phase timeline, including
// the usage totals, is the same value object embedded in the mission view.
type UsageView struct {
	Schema    string `json:"schema"`
	MissionID string `json:"mission_id"`
	// Source says where the lifecycle and usage records came from: the
	// ledgers or a replayed exported event stream.
	Source        string            `json:"source"`
	Lifecycle     Lifecycle         `json:"lifecycle"`
	PhaseTimeline PhaseTimeline     `json:"phase_timeline"`
	Legacy        LegacyUsage       `json:"legacy"`
	Chronology    []ChronologyEntry `json:"chronology,omitempty"`
	Diagnostics   []Diagnostic      `json:"diagnostics"`
}

// UsageInput contains values already loaded from their authorities.
type UsageInput struct {
	Status           domain.MissionEngineStatus
	Lifecycle        []telemetry.MissionLifecycleRecord
	LifecycleError   error
	Usage            []telemetry.MissionUsageRecord
	UsageError       error
	KnownInvocations []string
	TokenUsage       []telemetry.MissionTokenUsageRecord
	TokenUsageError  error
	// WithChronology adds the merged marker and observation timeline.
	WithChronology bool
	// FromEvents marks the records as rebuilt from an exported event stream.
	FromEvents bool
}

// BuildUsage composes the usage projection without persisting anything.
func BuildUsage(in UsageInput) UsageView {
	v := UsageView{
		Schema: UsageSchemaVersion, MissionID: in.Status.MissionID, Source: usageSourceLabel(in.FromEvents),
		Lifecycle: Lifecycle{Phase: string(in.Status.Phase), State: string(in.Status.State)},
		PhaseTimeline: BuildPhaseTimeline(in.Status, in.Lifecycle, in.LifecycleError).
			WithUsage(SummarizeUsage(in.Usage, in.KnownInvocations), in.UsageError),
		Legacy:      buildLegacyUsage(in.TokenUsage, in.TokenUsageError),
		Diagnostics: usageDiagnostics(in),
	}
	if in.WithChronology {
		v.Chronology = BuildChronology(in.Lifecycle, in.Usage)
	}
	return v
}

func usageSourceLabel(fromEvents bool) string {
	if fromEvents {
		return "events"
	}
	return "ledgers"
}

func usageDiagnostics(in UsageInput) []Diagnostic {
	out := []Diagnostic{}
	if in.LifecycleError != nil {
		out = append(out, Diagnostic{Reason: ReasonMissionViewLifecycleUnavailable, Action: "inspect mission-lifecycle ledger"})
	}
	if in.UsageError != nil {
		out = append(out, Diagnostic{Reason: ReasonMissionViewUsageUnavailable, Action: "inspect mission-usage ledger"})
	}
	if in.TokenUsageError != nil {
		out = append(out, Diagnostic{Reason: ReasonMissionViewTokenUsageUnavailable, Action: "inspect mission-token-usage ledger"})
	}
	return out
}

func buildLegacyUsage(records []telemetry.MissionTokenUsageRecord, err error) LegacyUsage {
	switch {
	case err != nil:
		return LegacyUsage{Availability: Unavailable}
	case len(records) == 0:
		return LegacyUsage{Availability: NotApplicable}
	}
	return LegacyUsage{Availability: Available, RecordCount: len(records), Records: records}
}

// BuildChronology merges lifecycle markers and usage observations in time
// order, so a cumulative counter can be read next to the phase boundaries.
func BuildChronology(markers []telemetry.MissionLifecycleRecord, usage []telemetry.MissionUsageRecord) []ChronologyEntry {
	var out []ChronologyEntry
	for _, m := range markers {
		detail := m.EndReason
		out = append(out, ChronologyEntry{At: m.ObservedAt, Kind: m.MarkerKind, Phase: m.Phase, RunID: m.PhaseRunID, Detail: detail})
	}
	for _, u := range usage {
		out = append(out, ChronologyEntry{
			At: u.ObservedAt, Kind: "usage_observation", Phase: u.Phase, RunID: u.PhaseRunID,
			Detail: fmt.Sprintf("%s in=%s out=%s quality=%s invocation=%s calls=%d", u.RecordKind, count(u.TokensInput), count(u.TokensOutput), u.MeasurementQuality, orCell(u.InvocationID), u.Calls),
		})
	}
	sort.SliceStable(out, func(i, j int) bool { return chronologyBefore(out[i].At, out[j].At) })
	return out
}

func chronologyBefore(a, b string) bool {
	ta, errA := time.Parse(time.RFC3339Nano, a)
	tb, errB := time.Parse(time.RFC3339Nano, b)
	return errA == nil && errB == nil && ta.Before(tb)
}

// RenderUsageHuman writes the `mission usage` output. The legacy ledger is a
// separate row outside every total.
func RenderUsageHuman(w io.Writer, v UsageView, details bool) error {
	if err := writef(w, "MISSION %s   phase: %s   state: %s   source: %s\n", v.MissionID, v.Lifecycle.Phase, v.Lifecycle.State, v.Source); err != nil {
		return err
	}
	if err := RenderPhaseTimeline(w, v.PhaseTimeline, details); err != nil {
		return err
	}
	if err := renderTotalWhenTimelineIsEmpty(w, v.PhaseTimeline); err != nil {
		return err
	}
	if err := renderLegacyUsage(w, v.Legacy, details); err != nil {
		return err
	}
	if err := renderChronology(w, v.Chronology); err != nil {
		return err
	}
	return renderDiagnostics(w, v.Diagnostics)
}

// renderTotalWhenTimelineIsEmpty keeps the mission total visible when the
// timeline stopped at its "no lifecycle recorded" note.
func renderTotalWhenTimelineIsEmpty(w io.Writer, tl PhaseTimeline) error {
	if tl.Note == "" || tl.UsageAvailability == Available {
		return nil
	}
	return renderUsageTotals(w, tl)
}

func renderChronology(w io.Writer, entries []ChronologyEntry) error {
	if len(entries) == 0 {
		return nil
	}
	if err := writef(w, "Timeline\n"); err != nil {
		return err
	}
	for _, e := range entries {
		if err := writef(w, "  %s  %-18s %-13s %-3s %s\n", e.At, e.Kind, e.Phase, e.RunID, e.Detail); err != nil {
			return err
		}
	}
	return nil
}

func renderLegacyUsage(w io.Writer, l LegacyUsage, details bool) error {
	if err := writef(w, "Legacy ledger (own row, outside the total, semantics unknown): %s, %d record(s)\n", l.Availability, l.RecordCount); err != nil {
		return err
	}
	if !details {
		return nil
	}
	for _, rec := range l.Records {
		if err := writef(w, "  legacy %s tokens_in=%d tokens_out=%d source=%s\n", rec.ReportedAt, rec.TokensIn, rec.TokensOut, rec.Source); err != nil {
			return err
		}
	}
	return nil
}
