package calibration

import "fmt"

// PairLimits are the limits a dimension must meet to be called fit to judge.
type PairLimits struct {
	MinAUC     float64 `json:"min_auc"`
	MinMargin  float64 `json:"min_margin"`
	MaxStdP95  float64 `json:"max_std_p95"`
	MinParents int     `json:"min_parents"`
}

// DefaultPairLimits mirror the package gates, with a floor on independent parents.
func DefaultPairLimits() PairLimits {
	return PairLimits{MinAUC: 0.90, MinMargin: 0.05, MaxStdP95: 0.10, MinParents: 10}
}

// Dimension statuses.
const (
	DimensionFit                = "fit"
	DimensionNotFit             = "not_fit"
	DimensionInsufficientSample = "insufficient_sample"
)

// DimensionReport is the evidence about one dimension: can the provider tell a
// pair where the relation holds from one where it does not, by how much, and how
// steadily. A dimension that is not fit is not used to decide anything.
type DimensionReport struct {
	Dimension    string       `json:"dimension"`
	Status       string       `json:"status"`
	FailingGate  string       `json:"failing_gate,omitempty"`
	Parents      int          `json:"parents"`
	Positives    int          `json:"positives"`
	Negatives    int          `json:"negatives"`
	AUC          float64      `json:"auc"`
	HeldOutAUC   float64      `json:"held_out_auc"`
	Margin       float64      `json:"margin"`
	StdP95       float64      `json:"std_p95"`
	PositiveMean float64      `json:"positive_mean"`
	NegativeMean float64      `json:"negative_mean"`
	Sweep        []SweepPoint `json:"sweep,omitempty"`
}

// SweepPoint is an operating point of one dimension: the share of pairs where the
// relation does not hold that the dimension would still accept, and the share
// where it holds that it would reject, at a threshold on its score.
type SweepPoint struct {
	Threshold   float64 `json:"threshold"`
	FalseAccept Rate    `json:"false_accept"`
	FalseReject Rate    `json:"false_reject"`
}

// FamilyReport is the paired reading of one family against its faithful sibling.
type FamilyReport struct {
	Family string `json:"family"`
	// Dimension is the dimension the family breaks, or "" for a control.
	Dimension string `json:"dimension,omitempty"`
	Pairs     int    `json:"pairs"`
	// Responded is how many pairs scored lower than their faithful sibling by more
	// than PairEpsilon on the broken dimension; for a control, how many stayed within it.
	Responded Rate    `json:"responded"`
	MeanDelta float64 `json:"mean_delta"`
}

// PairReport is the full evaluation of a set of pair observations.
type PairReport struct {
	Limits     PairLimits        `json:"limits"`
	Dimensions []DimensionReport `json:"dimensions"`
	Families   []FamilyReport    `json:"families"`
	Calls      int               `json:"calls"`
	Failed     int               `json:"failed_calls"`
}

type pairStat struct {
	obs    PairObservation
	scores map[string][]float64
}

// EvaluatePairs computes the report from observations alone.
func EvaluatePairs(observations []PairObservation, limits PairLimits) PairReport {
	report := PairReport{Limits: limits}
	stats := pairStats(observations, &report)
	for _, dim := range PairDimensions() {
		report.Dimensions = append(report.Dimensions, dimensionReport(dim, stats, limits))
	}
	report.Families = familyReports(stats)
	return report
}

func pairStats(observations []PairObservation, report *PairReport) map[string]*pairStat {
	stats := map[string]*pairStat{}
	for _, o := range observations {
		report.Calls++
		if !o.Answered() {
			report.Failed++
			continue
		}
		s := stats[o.Pair]
		if s == nil {
			s = &pairStat{obs: o, scores: map[string][]float64{}}
			stats[o.Pair] = s
		}
		for dim, v := range o.Scores {
			s.scores[dim] = append(s.scores[dim], v)
		}
	}
	return stats
}

func (s *pairStat) mean(dim string) float64 { return Mean(s.scores[dim]) }

func dimensionReport(dim string, stats map[string]*pairStat, limits PairLimits) DimensionReport {
	r := DimensionReport{Dimension: dim}
	var devPos, devNeg, heldPos, heldNeg, spreads []float64
	parents := map[string]bool{}
	for _, s := range stats {
		holds, ok := s.obs.Oracle[dim]
		if !ok {
			continue
		}
		parents[s.obs.Parent] = true
		pick := map[bool]*[]float64{true: &devPos, false: &devNeg}
		if s.obs.HeldOut {
			pick = map[bool]*[]float64{true: &heldPos, false: &heldNeg}
		}
		*pick[holds] = append(*pick[holds], s.mean(dim))
		if len(s.scores[dim]) >= 2 {
			spreads = append(spreads, StdDev(s.scores[dim]))
		}
	}
	r.Parents, r.Positives, r.Negatives = len(parents), len(devPos)+len(heldPos), len(devNeg)+len(heldNeg)
	r.AUC, r.HeldOutAUC = AUC(devPos, devNeg), AUC(heldPos, heldNeg)
	r.Margin = Percentile(devPos, 0.10) - Percentile(devNeg, 0.90)
	r.StdP95 = Percentile(spreads, 0.95)
	r.PositiveMean, r.NegativeMean = Mean(devPos), Mean(devNeg)
	r.Sweep = dimensionSweep(devPos, devNeg)
	r.Status, r.FailingGate = dimensionStatus(r, devPos, devNeg, heldPos, heldNeg, limits)
	return r
}

func dimensionStatus(r DimensionReport, devPos, devNeg, heldPos, heldNeg []float64, l PairLimits) (string, string) {
	switch {
	case len(devPos) < l.MinParents || len(devNeg) < l.MinParents:
		return DimensionInsufficientSample, ""
	case r.AUC < l.MinAUC:
		return DimensionNotFit, GateSeparation
	case r.Margin < l.MinMargin:
		return DimensionNotFit, GateMargin
	case r.StdP95 > l.MaxStdP95:
		return DimensionNotFit, GateStability
	case len(heldPos) > 0 && len(heldNeg) > 0 && AUC(heldPos, heldNeg) < l.MinAUC:
		return DimensionNotFit, "held_out"
	}
	return DimensionFit, ""
}

func dimensionSweep(pos, neg []float64) []SweepPoint {
	var out []SweepPoint
	for i := 30; i <= 95; i += 5 {
		t := float64(i) / 100
		fa := countFloats(neg, func(v float64) bool { return v > t })
		fr := countFloats(pos, func(v float64) bool { return v <= t })
		out = append(out, SweepPoint{Threshold: t, FalseAccept: NewRate(fa, len(neg)), FalseReject: NewRate(fr, len(pos))})
	}
	return out
}

func countFloats(values []float64, hit func(float64) bool) int {
	n := 0
	for _, v := range values {
		if hit(v) {
			n++
		}
	}
	return n
}

// Describe is a one-line summary of a dimension for a result item.
func (d DimensionReport) Describe() string {
	return fmt.Sprintf("%s: AUC %.4f (held out %.4f), margin %.4f, p95 std %.3f, %d parents, positive mean %.2f, negative mean %.2f", d.Status, d.AUC, d.HeldOutAUC, d.Margin, d.StdP95, d.Parents, d.PositiveMean, d.NegativeMean)
}
