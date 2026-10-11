package gatecriteria

import (
	"github.com/SergioLacerda/strategist-skill/internal/domain"
	"github.com/SergioLacerda/strategist-skill/internal/telemetry"
	"github.com/SergioLacerda/strategist-skill/internal/usagesummary"
)

// Availability of the stage token volumes.
const (
	AvailabilityAvailable     = "available"
	AvailabilityNotApplicable = "not_applicable"
	AvailabilityUnavailable   = "unavailable"
)

const usageSource = telemetry.MissionUsageRelPath

// StageUsage is the token volume of one display stage. Members are null until
// observed: an unobserved volume is never zero. LowerBound is the sum of the
// members that were observed when the total is not computable.
type StageUsage struct {
	Stage string `json:"stage"`
	usagesummary.Tokens
	LowerBound   *int64 `json:"lower_bound"`
	Quality      string `json:"quality"`
	Observations int    `json:"observations"`
	// Pending marks a stage that has not run yet (delivery at the gate).
	Pending bool `json:"pending"`
}

// StageTokens is the per-stage volume report of the Approval Gate. It carries
// volumes and their quality only; there is no cost, price or threshold.
type StageTokens struct {
	Availability string              `json:"availability"`
	Stages       []StageUsage        `json:"stages"`
	Unattributed usagesummary.Bucket `json:"unattributed"`
	Total        usagesummary.Bucket `json:"total"`
}

// stageLabels maps the canonical usage rows to the stage names shown to the
// operator, in pipeline order. Intake is shown only when it was observed.
var stageLabels = []struct {
	row, label string
	optional   bool
}{
	{string(domain.PhaseIntake), "intake", true},
	{string(domain.PhaseDiscovery), "discovery", false},
	{string(domain.PhaseRefinement), "refining", false},
	{string(domain.PhaseExecution), "delivering", false},
}

// TokenStages reads the mission usage ledger and reports the volume per stage.
// It never recomputes semantics: aggregation is the shared usagesummary.
func TokenStages(in Input) StageTokens {
	recs, err := telemetry.ReadMissionUsageFor(telemetry.MissionUsagePath(in.Root), in.MissionID)
	sum := usagesummary.Summarize(recs, nil)
	st := StageTokens{
		Availability: availabilityOf(len(recs), sum, err),
		Stages:       stagesOf(sum),
		Unattributed: sum.Unattributed,
		Total:        sum.Total,
	}
	if err != nil {
		st.Total = usagesummary.Bucket{}
	}
	return st
}

func availabilityOf(records int, sum usagesummary.Summary, err error) string {
	switch {
	case err != nil:
		return AvailabilityUnavailable
	case records == 0:
		return AvailabilityNotApplicable
	case sum.Total.Observations == 0:
		return AvailabilityUnavailable
	}
	return AvailabilityAvailable
}

func stagesOf(sum usagesummary.Summary) []StageUsage {
	stages := make([]StageUsage, 0, len(stageLabels))
	for _, s := range stageLabels {
		bucket, observed := sum.ByPhase[s.row]
		if s.optional && !observed {
			continue
		}
		stages = append(stages, stageUsageOf(s.label, bucket, observed))
	}
	return stages
}

func stageUsageOf(label string, b usagesummary.Bucket, observed bool) StageUsage {
	u := StageUsage{Stage: label, Tokens: b.Tokens, Quality: b.Quality, Observations: b.Observations}
	u.LowerBound = lowerBound(b.Tokens)
	u.Pending = label == "delivering" && !observed
	return u
}

// lowerBound sums the observed members when the total is not computable.
func lowerBound(t usagesummary.Tokens) *int64 {
	if t.Total != nil || (t.Input == nil && t.Output == nil) {
		return nil
	}
	var sum int64
	for _, p := range []*int64{t.Input, t.Output} {
		if p != nil {
			sum += *p
		}
	}
	return &sum
}
