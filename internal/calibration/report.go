package calibration

import "sort"

// Status is the outcome of a calibration.
type Status string

// Calibration statuses.
const (
	StatusRecommended         Status = "recommended"
	StatusNotFit              Status = "not_fit"
	StatusInsufficientSample  Status = "insufficient_sample"
	StatusNoFeasibleThreshold Status = "no_feasible_threshold"
)

// Config is everything Evaluate needs besides the observations.
type Config struct {
	Policy Policy
	Limits Limits
	Filter Filter
	// Current is the configured threshold, evaluated for reference (0 means the default 0.80).
	Current float64
	// LocalScope lists the operators whose protection is delegated to the local
	// checks: the judge gates and rates do not read them, the local coverage gate does.
	LocalScope map[string]bool
}

// SetResult is the result of one case set at a threshold.
type SetResult struct {
	Threshold      float64 `json:"threshold"`
	Positives      int     `json:"positives"`
	Defects        int     `json:"defects"`
	FalseApproval  Rate    `json:"false_approval"`
	FalseRejection Rate    `json:"false_rejection"`
}

// Counts summarize the data a report stands on.
type Counts struct {
	Observations int `json:"observations"`
	Failed       int `json:"failed_calls"`
	Golden       int `json:"golden_positives"`
	Defects      int `json:"defect_negatives"`
	Neutral      int `json:"neutral"`
	Pairs        int `json:"pairs"`
	HeldOutPos   int `json:"held_out_positives"`
	HeldOutNeg   int `json:"held_out_defects"`
}

// Report is the full, recomputable evaluation of a set of observations.
type Report struct {
	Status      Status     `json:"status"`
	FailingGate string     `json:"failing_gate,omitempty"`
	Threshold   *float64   `json:"threshold,omitempty"`
	Notes       []string   `json:"notes,omitempty"`
	Policy      Policy     `json:"risk_policy"`
	Limits      Limits     `json:"limits"`
	Counts      Counts     `json:"counts"`
	Gates       []Gate     `json:"gates"`
	AUC         float64    `json:"auc"`
	ECE         float64    `json:"ece"`
	Brier       float64    `json:"brier"`
	Sweep       []SweepRow `json:"sweep,omitempty"`
	Selection   *SetResult `json:"selection,omitempty"`
	HeldOut     *SetResult `json:"held_out,omitempty"`
	// Criteria is the per-criterion answer of an atomic profile, by group of cases.
	Criteria *CriteriaMatrix `json:"criteria,omitempty"`
	// CrossValidation is the out-of-fold reading that confirms a threshold under gates-3.
	CrossValidation *CrossValidation `json:"cross_validation,omitempty"`
	AtCurrent       SetResult        `json:"at_current"`
	Operators       []OperatorResult `json:"operators,omitempty"`
	// Exploratory lists the operators left out of this evaluation; a non-empty list
	// means the report is a what-if and no calibration may be published from it.
	Exploratory []string `json:"exploratory_excluded_operators,omitempty"`
	// LocalScope lists the operators delegated to the local checks, kept in the report and the record.
	LocalScope []string `json:"local_scope,omitempty"`
	// Policy is the measure of the whole decision, judge and local checks, at the
	// recommended threshold or, without one, at the reference threshold.
	PolicyLevel *PolicyLevel `json:"policy_level,omitempty"`
	// Refusals explains why the golden parents were not approved at the reference threshold.
	Refusals Refusals `json:"refusals"`
}

// DefaultConfig uses the default policy, limits and reference threshold.
func DefaultConfig() Config {
	return Config{Policy: DefaultPolicy(), Limits: DefaultLimits(), Current: 0.80}
}

func setResult(stats []CaseStat, threshold, rho float64) SetResult {
	pos, neg := positives(stats), defects(stats)
	return SetResult{Threshold: threshold, Positives: len(pos), Defects: len(neg), FalseApproval: falseApproval(neg, threshold, rho), FalseRejection: falseRejection(pos, threshold)}
}

// Evaluate computes the report from observations alone. The threshold is chosen
// on the selection (non held-out) cases only, after the fitness gates and the
// sample-size rule, and then confirmed on the held-out cases.
func Evaluate(observations []Observation, cfg Config) Report {
	all := Stats(observations, cfg.Filter)
	stats := withoutOperators(all, cfg.LocalScope)
	dev, held := split(stats, false), split(stats, true)
	report := Report{Exploratory: sortedOperators(cfg.Filter.ExcludedOperators), LocalScope: sortedOperators(cfg.LocalScope), Policy: cfg.Policy, Limits: cfg.Limits,
		Counts: counts(observations, stats, held), Operators: PairedResults(stats)}
	report.AtCurrent = setResult(stats, cfg.Current, cfg.Policy.ClusterRho)
	report.Criteria = BuildCriteriaMatrix(all)
	pos, neg := positives(stats), defects(stats)
	report.AUC = AUC(scoresOf(pos), scoresOf(neg))
	report.ECE, report.Brier = ECE(scored(pos, neg)), Brier(scored(pos, neg))
	sel := selectionSet(dev, stats, cfg)
	report.Gates = FitnessGates(sel, cfg.Limits)
	if reference := policyLevel(all, cfg.Current, cfg); reference != nil {
		if gate, ok := LocalCoverageGate(reference.Coverage, cfg.LocalScope, cfg.Limits.MinLocalCoverage); ok {
			report.Gates = append(report.Gates, gate)
		}
	}
	report.Sweep = Sweep(sel, cfg.Policy)
	decideStatus(&report, sel, held, cfg)
	report.PolicyLevel = policyLevel(all, policyThreshold(report, cfg), cfg)
	report.Refusals = refusalsOf(stats, cfg.Current)
	return report
}

// selectionSet is the selection part under a held-out split and the whole corpus
// under cross-validation, which holds data out fold by fold instead.
func selectionSet(dev, all []CaseStat, cfg Config) []CaseStat {
	if cfg.Limits.CVFolds > 0 {
		return all
	}
	return dev
}

// policyThreshold is the recommended threshold when there is one, else the reference one.
func policyThreshold(report Report, cfg Config) float64 {
	if report.Threshold != nil {
		return *report.Threshold
	}
	return cfg.Current
}

func withoutOperators(stats []CaseStat, skip map[string]bool) []CaseStat {
	if len(skip) == 0 {
		return stats
	}
	return pick(stats, func(c CaseStat) bool { return !skip[c.Operator] })
}

func scored(pos, neg []CaseStat) []Scored {
	out := make([]Scored, 0, len(pos)+len(neg))
	for _, stat := range pos {
		out = append(out, Scored{stat.Score(), 1})
	}
	for _, stat := range neg {
		out = append(out, Scored{stat.Score(), 0})
	}
	return out
}

func counts(all []Observation, stats, held []CaseStat) Counts {
	c := Counts{Observations: len(all), Golden: len(positives(stats)), Defects: len(defects(stats)), HeldOutPos: len(positives(held)), HeldOutNeg: len(defects(held))}
	for _, stat := range stats {
		c.Failed += stat.Failed
		if stat.Kind == KindNeutral {
			c.Neutral++
		}
		if stat.ParentID != "" {
			c.Pairs++
		}
	}
	return c
}

func sortedOperators(set map[string]bool) []string {
	var out []string
	for name, on := range set {
		if on {
			out = append(out, name)
		}
	}
	sort.Strings(out)
	return out
}
