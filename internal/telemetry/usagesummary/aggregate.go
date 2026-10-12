// Package usagesummary aggregates a mission's declared token observations
// without double counting. It is shared by the read model, the Approval Gate
// criteria and any other consumer, and it has no dependency on them: a replayed
// record id is dropped, incremental observations add, and cumulative snapshots
// contribute only the delta inside one session and counter.
package usagesummary

import (
	"sort"
	"time"

	"github.com/SergioLacerda/strategist-skill/internal/domain"
	"github.com/SergioLacerda/strategist-skill/internal/telemetry"
)

// Quality labels of a bucket that no single observation quality describes.
const (
	QualityMixed       = "mixed"
	QualityUnavailable = "unavailable"
)

// Tokens holds token counts. Every member is null until usage is observed: an
// unobserved value is never zero.
type Tokens struct {
	Input  *int64 `json:"input"`
	Output *int64 `json:"output"`
	Total  *int64 `json:"total"`
}

// Bucket is the token usage of one phase row, of the unattributed
// consumption or of the whole mission. A member is null when it was not
// observed; quality says how far the numbers may be trusted.
type Bucket struct {
	Tokens           Tokens         `json:"tokens"`
	Quality          string         `json:"quality"`
	QualityBreakdown map[string]int `json:"quality_breakdown,omitempty"`
	Observations     int            `json:"observations"`
}

// Coverage states how much of the mission the numbers cover. It never
// converts missing volume into a percentage: that volume is unknown.
type Coverage struct {
	InvocationsKnown    int `json:"invocations_known"`
	InvocationsObserved int `json:"invocations_observed"`
	InvocationsMissing  int `json:"invocations_missing"`
	UnkeyedObservations int `json:"unkeyed_observations"`
	BaselineSnapshots   int `json:"baseline_snapshots"`
	ResetIntervals      int `json:"reset_intervals"`
}

// Summary is the aggregation of a mission's usage ledger.
type Summary struct {
	ByPhase map[string]Bucket
	// ByRun retains usage only for observations with an explicit phase/run
	// correlation key. It deliberately does not infer a run for legacy data.
	ByRun        map[string]Bucket
	Unattributed Bucket
	Total        Bucket
	Coverage     Coverage
}

// Summarize aggregates declared observations without double counting: a
// repeated record id is dropped, incremental observations add, and cumulative
// snapshots contribute only the delta between two snapshots of the same
// session and counter (a reset or a new session yields no interval).
// knownInvocations feeds the coverage counters.
func Summarize(recs []telemetry.MissionUsageRecord, knownInvocations []string) Summary {
	unique := dedupeUsage(recs)
	contribs, cov := usageContributions(unique)
	cov = withInvocationCoverage(cov, unique, knownInvocations)

	byPhase, byRun, unattributed, total := aggregateContributions(contribs)
	summary := Summary{
		ByPhase: map[string]Bucket{}, ByRun: map[string]Bucket{}, Unattributed: unattributed.bucket(),
		Total: total.bucket(), Coverage: cov,
	}
	fillBuckets(summary.ByPhase, byPhase)
	fillBuckets(summary.ByRun, byRun)
	return summary
}

// usageRowKey maps a usage phase to its timeline row: bootstrap and intake
// share one Intake row, as they do in the lifecycle timeline.
func usageRowKey(phase string) string {
	if phase == string(domain.PhaseBootstrap) {
		return string(domain.PhaseIntake)
	}
	return phase
}

func dedupeUsage(recs []telemetry.MissionUsageRecord) []telemetry.MissionUsageRecord {
	seen := map[string]bool{}
	var out []telemetry.MissionUsageRecord
	for _, r := range recs {
		if seen[r.RecordID] {
			continue
		}
		seen[r.RecordID] = true
		out = append(out, r)
	}
	sort.SliceStable(out, func(i, j int) bool { return observedBefore(out[i].ObservedAt, out[j].ObservedAt) })
	return out
}

func observedBefore(a, b string) bool {
	ta, errA := time.Parse(time.RFC3339Nano, a)
	tb, errB := time.Parse(time.RFC3339Nano, b)
	return errA == nil && errB == nil && ta.Before(tb)
}

func usageContributions(recs []telemetry.MissionUsageRecord) ([]contribution, Coverage) {
	var contribs []contribution
	var cov Coverage
	series := map[string]snapshotPoint{}
	for _, r := range recs {
		if r.DedupUnavailable {
			cov.UnkeyedObservations++
		}
		c, ok := contributionOf(r, series, &cov)
		if ok {
			contribs = append(contribs, c)
		}
	}
	return contribs, cov
}

// contributionOf turns one record into a contribution. An incremental record
// contributes its counts; a snapshot contributes the delta from the previous
// snapshot of the same session and counter, or nothing when there is none or
// the counter went backwards.
func contributionOf(r telemetry.MissionUsageRecord, series map[string]snapshotPoint, cov *Coverage) (contribution, bool) {
	if r.RecordKind != string(domain.UsageKindCumulativeSnapshot) {
		return contribution{phase: r.Phase, phaseRunID: r.PhaseRunID, in: r.TokensInput, out: r.TokensOutput, quality: r.MeasurementQuality}, true
	}
	key := r.SessionID + "|" + r.UsageScope
	prev, hasPrev := series[key]
	series[key] = snapshotPoint{in: r.TokensInput, out: r.TokensOutput}
	if !hasPrev {
		cov.BaselineSnapshots++
		return contribution{}, false
	}
	in, out := delta(prev.in, r.TokensInput), delta(prev.out, r.TokensOutput)
	if in == nil && out == nil {
		cov.ResetIntervals++
		return contribution{}, false
	}
	return contribution{phase: r.Phase, phaseRunID: r.PhaseRunID, in: in, out: out, quality: r.MeasurementQuality}, true
}

// delta returns cur-prev, or nil when either side is unobserved or the
// counter went backwards (a reset): the interval is unavailable, not negative.
func delta(prev, cur *int64) *int64 {
	if prev == nil || cur == nil || *cur < *prev {
		return nil
	}
	d := *cur - *prev
	return &d
}

func withInvocationCoverage(cov Coverage, recs []telemetry.MissionUsageRecord, known []string) Coverage {
	observed := map[string]bool{}
	for _, r := range recs {
		observed[r.InvocationID] = true
	}
	seen := map[string]bool{}
	for _, id := range known {
		if id == "" || seen[id] {
			continue
		}
		seen[id] = true
		cov.InvocationsKnown++
		if observed[id] {
			cov.InvocationsObserved++
		}
	}
	cov.InvocationsMissing = cov.InvocationsKnown - cov.InvocationsObserved
	return cov
}
