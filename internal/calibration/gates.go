package calibration

import (
	"fmt"
)

// GateStatus is the outcome of one fitness gate.
type GateStatus string

// Gate statuses. A gate that cannot be measured has not passed.
const (
	GatePassed      GateStatus = "passed"
	GateFailed      GateStatus = "failed"
	GateNotMeasured GateStatus = "not_measured"
)

// Gate names, in the order they are read.
const (
	GateSeparation = "separation"
	GateMargin     = "margin"
	GateMonotonic  = "monotonic_response"
	GateStability  = "stability"
	// GateLocalCoverage is read after the judge gates and only for a case that
	// delegates operators to the local checks.
	GateLocalCoverage = "local_coverage"
)

// Limits are the fitness-gate limits.
type Limits struct {
	MinAUC       float64 `json:"min_auc"`
	MonotonicTol float64 `json:"monotonic_tolerance"`
	MaxStdP95    float64 `json:"max_std_p95"`
	// NoiseFactor scales the measured repetition noise (the p95 standard deviation of
	// a case's score) into the monotonic tolerance: a step is an inversion only when
	// the score rises by more than max(MonotonicTol, NoiseFactor*noise), because a
	// smaller rise cannot be told from the judge's own variation. Zero means no
	// noise term, which is how records made before it existed were computed.
	NoiseFactor float64 `json:"noise_factor,omitempty"`
	// Rule names the gate rule these limits belong to. gates-1 is what the first
	// pilots ran under (no margin gate, zero inversions, independent cases);
	// gates-2 is pre-registered for the runs that follow: a margin gate, an
	// inversion rate under an exact upper bound, and cluster-aware samples.
	Rule string `json:"gate_rule,omitempty"`
	// MinMargin is the least gap between the low end of the golden scores (p10) and the
	// high end of the defect scores (p90). Zero disables the margin gate.
	MinMargin float64 `json:"min_margin,omitempty"`
	// MaxInversionRate bounds the share of ladder steps that invert, by its exact
	// one-sided upper bound on the effective number of steps. Zero means no inversion is allowed.
	MaxInversionRate float64 `json:"max_inversion_rate,omitempty"`
	// MinLocalCoverage is the least lower bound (95 percent, one-sided) of the rate at
	// which the local checks must catch each operator delegated to them.
	MinLocalCoverage float64 `json:"min_local_coverage,omitempty"`
	// CVFolds, when positive, replaces the single held-out split by a grouped
	// cross-validation over that many folds (gates-3).
	CVFolds int `json:"cv_folds,omitempty"`
	// BandNoiseFactor, when positive, requires the band of feasible thresholds to be
	// at least that many times the repetition noise wide (gates-3).
	BandNoiseFactor float64 `json:"band_noise_factor,omitempty"`
}

// Gate rules.
const (
	RuleGates1 = "gates-1"
	RuleGates2 = "gates-2"
	// RuleGates3 is gates-2 with the single held-out split replaced by a grouped
	// cross-validation judged on upper bounds, a threshold-band gate, and a
	// threshold placed at the middle of its band.
	RuleGates3 = "gates-3"
)

// DefaultLimits are the starting limits of the fitness gates.
func DefaultLimits() Limits {
	return Limits{MinAUC: 0.90, MonotonicTol: 0.05, MaxStdP95: 0.10, NoiseFactor: 2, Rule: RuleGates2, MinMargin: 0.05, MaxInversionRate: 0.20, MinLocalCoverage: 0.80}
}

// Gates1Limits are the limits the first pilots ran under, to reproduce a report
// under the rule it was first read with.
func Gates1Limits() Limits {
	return Limits{MinAUC: 0.90, MonotonicTol: 0.05, MaxStdP95: 0.10, NoiseFactor: 2, Rule: RuleGates1}
}

// ConfigForRule is the default configuration of a gate rule: its limits and its
// assumption about the independence of mutants of one parent.
func ConfigForRule(rule string) Config {
	cfg := DefaultConfig()
	switch rule {
	case RuleGates1:
		cfg.Limits = Gates1Limits()
		cfg.Policy.ClusterRho = 0
	case RuleGates3:
		cfg.Limits = Gates3Limits()
	}
	return cfg
}

// Gate is the result of one fitness gate.
type Gate struct {
	Name   string     `json:"name"`
	Status GateStatus `json:"status"`
	Value  float64    `json:"value"`
	Limit  float64    `json:"limit"`
	Detail string     `json:"detail,omitempty"`
}

func decide(name string, value, limit float64, pass bool, detail string) Gate {
	status := GateFailed
	if pass {
		status = GatePassed
	}
	return Gate{Name: name, Status: status, Value: value, Limit: limit, Detail: detail}
}

func unmeasured(name string, limit float64, detail string) Gate {
	return Gate{Name: name, Status: GateNotMeasured, Limit: limit, Detail: detail}
}

func separationGate(dev []CaseStat, limits Limits) Gate {
	pos, neg := positives(dev), defects(dev)
	if len(pos) == 0 || len(neg) == 0 {
		return unmeasured(GateSeparation, limits.MinAUC, "needs golden positives and defect negatives")
	}
	auc := AUC(scoresOf(pos), scoresOf(neg))
	return decide(GateSeparation, auc, limits.MinAUC, auc >= limits.MinAUC,
		fmt.Sprintf("auc %.4f (limit %.4f, difference %+.4f) over %d golden parents and %d defects from %d parents", auc, limits.MinAUC, auc-limits.MinAUC, len(pos), len(neg), parentCount(neg)))
}

// marginGate requires a real gap between the golden scores and the defect scores,
// because a judge can separate by a hair and still be useless at any threshold.
func marginGate(dev []CaseStat, limits Limits) Gate {
	pos, neg := positives(dev), defects(dev)
	if len(pos) == 0 || len(neg) == 0 {
		return unmeasured(GateMargin, limits.MinMargin, "needs golden positives and defect negatives")
	}
	low, high := Percentile(scoresOf(pos), 0.10), Percentile(scoresOf(neg), 0.90)
	margin := low - high
	return decide(GateMargin, margin, limits.MinMargin, margin >= limits.MinMargin, fmt.Sprintf("p10 of golden %.4f minus p90 of defects %.4f", low, high))
}

// noiseP95 is the p95 standard deviation of a case's score across repetitions, 0 when none repeats.
func noiseP95(dev []CaseStat) float64 {
	var spreads []float64
	for _, stat := range dev {
		if stat.Answered() >= 2 {
			spreads = append(spreads, StdDev(stat.Scores()))
		}
	}
	return Percentile(spreads, 0.95)
}

func stabilityGate(dev []CaseStat, limits Limits) Gate {
	repeated := 0
	for _, stat := range dev {
		if stat.Answered() >= 2 {
			repeated++
		}
	}
	if repeated == 0 {
		return unmeasured(GateStability, limits.MaxStdP95, "needs at least two repetitions")
	}
	p95 := noiseP95(dev)
	return decide(GateStability, p95, limits.MaxStdP95, p95 <= limits.MaxStdP95, fmt.Sprintf("p95 standard deviation over %d cases", repeated))
}

// FitnessGates runs the gates on the selection (non held-out) cases.
func FitnessGates(dev []CaseStat, limits Limits) []Gate {
	gates := []Gate{separationGate(dev, limits)}
	if limits.MinMargin > 0 {
		gates = append(gates, marginGate(dev, limits))
	}
	return append(gates, monotonicGate(dev, limits), stabilityGate(dev, limits))
}

// FirstOpenGate is the first gate that did not pass, or "" when all passed.
func FirstOpenGate(gates []Gate) string {
	for _, gate := range gates {
		if gate.Status != GatePassed {
			return gate.Name
		}
	}
	return ""
}
