package dojo

import (
	"fmt"

	"github.com/SergioLacerda/strategist-skill/internal/calibration"
	"github.com/SergioLacerda/strategist-skill/internal/domain"
)

// CheckInput is what a calibration case check needs besides its observations.
type CheckInput struct {
	Criteria domain.DojoCriteria
	Paths    CasePaths
	// Current is the configured approval threshold the handoff uses today.
	Current float64
	// ProfileVersion is the question profile version of the boundary now.
	ProfileVersion string
	// ExcludeOperators leaves operators out for an exploratory what-if; the result is labelled and no calibration follows from it.
	ExcludeOperators []string
	// LocalFlags backfills the local check flags of observations made before the
	// local checks were recorded, by case id. The checks are deterministic over the
	// case's fields, so the caller regenerates the cases (same corpus, same seed) and
	// computes them; the stored ledger is never rewritten.
	LocalFlags map[string][]string
	// WhatIfLocalScope delegates operators to the local checks for this evaluation only; the result is labelled exploratory.
	WhatIfLocalScope []string
	// JudgeOnly ignores the case's delegation to the local checks for this evaluation: the judge alone answers for every operator. Exploratory.
	JudgeOnly bool
	// ExcludeCriteria re-reads an atomic profile's stored answers without these criteria, a what-if labelled exploratory.
	ExcludeCriteria []string
	// GateRule overrides the case's gate rule for a recompute (gates-1 reproduces the first pilots' reading).
	GateRule string
	// Session selects the observations: "" for all, "latest" for the last session, or an id.
	Session string
}

// CheckOutcome is the evaluation of a case from its observations alone.
type CheckOutcome struct {
	// Observations are the ones the report was computed from.
	Observations []calibration.Observation
	// Backfilled counts observations whose local flags were derived now, not recorded at run time.
	Backfilled int
	Result     domain.DojoCheckResult
	Report     calibration.Report
	Audits     []AuditEntry
	Model      string
	Profile    string
	Version    string
}

// CheckCalibration evaluates a case from its observation ledger with no call: the
// same pass or fail a run ends with, so a new risk policy needs no new calls.
func CheckCalibration(in CheckInput) (CheckOutcome, error) {
	cal := *in.Criteria.Calibration
	observations, malformed, err := ReadObservations(in.Paths.ObservationsPath)
	if err != nil {
		return CheckOutcome{}, err
	}
	session := in.Session
	if session == "latest" {
		session = calibration.LatestSession(observations)
	}
	observations = calibration.SessionOf(observations, session)
	if observations, err = calibration.WithoutCriteria(observations, in.ExcludeCriteria); err != nil {
		return CheckOutcome{}, fmt.Errorf("exclude criterion: %w", err)
	}
	backfilled := backfill(observations, in.LocalFlags)
	filter, audits, err := LoadAudit(in.Paths.AuditPath)
	if err != nil {
		return CheckOutcome{}, err
	}
	cfg := evaluationConfig(in, cal)
	filter.ExcludedOperators = toSet(in.ExcludeOperators)
	cfg.Filter, cfg.Current = filter, in.Current
	report := calibration.Evaluate(observations, cfg)
	report.Exploratory = append(report.Exploratory, whatIfLabels(in)...)
	out := CheckOutcome{Report: report, Audits: audits, Observations: observations, Backfilled: backfilled}
	out.Model, out.Profile, out.Version = lastBinding(observations)
	out.Result = domain.DojoCheckResult{Scenario: in.Criteria.Scenario, Items: calibrationItems(in, out, len(observations), malformed)}
	out.Result.Calibration = summarize(out, cfg.Current)
	return out, nil
}

// evaluationConfig is the gate rule, risk policy and local scope a case is read under, with the what-if overrides.
func evaluationConfig(in CheckInput, cal domain.DojoCalibration) calibration.Config {
	rule := in.GateRule
	if rule == "" {
		rule = cal.GateRule
	}
	cfg := calibration.ConfigForRule(rule)
	cfg.Policy.MaxFalseApproval, cfg.Policy.MaxFalseRejection = cal.RiskPolicy.MaxFalseApproval, cal.RiskPolicy.MaxFalseRejection
	cfg.LocalScope = toSet(append(append([]string(nil), cal.LocalScope...), in.WhatIfLocalScope...))
	if in.JudgeOnly {
		cfg.LocalScope = nil
	}
	return cfg
}

// whatIfLabels name every override of an evaluation, so a report that was not read as recorded says so.
func whatIfLabels(in CheckInput) []string {
	var labels []string
	for _, name := range in.WhatIfLocalScope {
		labels = append(labels, "local_scope:"+name)
	}
	if in.JudgeOnly {
		labels = append(labels, "local_scope:none")
	}
	for _, name := range in.ExcludeCriteria {
		labels = append(labels, "criterion:"+name)
	}
	return labels
}

func lastBinding(observations []calibration.Observation) (model, profile, version string) {
	for i := len(observations) - 1; i >= 0; i-- {
		if observations[i].Model != "" {
			return observations[i].Model, observations[i].Profile, observations[i].ProfileVersion
		}
	}
	return "", "", ""
}

func summarize(out CheckOutcome, current float64) *domain.DojoCalibrationResult {
	r := out.Report
	return &domain.DojoCalibrationResult{
		Status: string(r.Status), FailingGate: r.FailingGate, Threshold: r.Threshold, Golden: r.Counts.Golden, Defects: r.Counts.Defects,
		CurrentFalseApproval: r.AtCurrent.FalseApproval.Value, CurrentFalseRejection: r.AtCurrent.FalseRejection.Value,
		CurrentThreshold: current, Model: out.Model, Profile: out.Profile,
	}
}

func item(label string, passed bool, format string, args ...any) domain.DojoCheckItem {
	return domain.DojoCheckItem{Label: label, Passed: passed, Detail: fmt.Sprintf(format, args...)}
}
