package calibration

import "fmt"

// Policy is the explicit risk policy of a calibration: the limits false approval
// (on defect negatives) and false rejection (on golden positives) must stay under.
type Policy struct {
	MaxFalseApproval  float64 `json:"max_false_approval"`
	MaxFalseRejection float64 `json:"max_false_rejection"`
	// ClusterRho is the assumed intra-parent correlation of defect cases. Mutants
	// of one parent are not independent trials, so the false approval bounds use
	// the sample size reduced by 1 + (m-1)*rho. Zero means independence, which is
	// how the first two gate rules treated them.
	ClusterRho float64 `json:"cluster_rho,omitempty"`
}

// DefaultPolicy is the starting risk policy.
func DefaultPolicy() Policy {
	return Policy{MaxFalseApproval: 0.05, MaxFalseRejection: 0.15, ClusterRho: DefaultClusterRho}
}

// DefaultClusterRho is the pre-registered intra-parent correlation (gate rule gates-2).
const DefaultClusterRho = 0.25

// SweepRow is the dev-set result at one candidate threshold.
type SweepRow struct {
	Threshold      float64 `json:"threshold"`
	FalseApproval  Rate    `json:"false_approval"`
	FalseRejection Rate    `json:"false_rejection"`
	Feasible       bool    `json:"feasible"`
}

// Candidate thresholds: 0.50 to 0.99 in steps of 0.01.
func thresholds() []float64 {
	out := make([]float64, 0, 50)
	for i := 50; i <= 99; i++ {
		out = append(out, float64(i)/100)
	}
	return out
}

// Sweep evaluates every candidate threshold on the selection cases. A threshold
// is feasible when the upper confidence bound of both rates is within the policy.
func Sweep(dev []CaseStat, policy Policy) []SweepRow {
	pos, neg := positives(dev), defects(dev)
	rows := make([]SweepRow, 0, 50)
	for _, t := range thresholds() {
		fa, fr := falseApproval(neg, t, policy.ClusterRho), falseRejection(pos, t)
		feasible := len(neg) > 0 && len(pos) > 0 && fa.Upper <= policy.MaxFalseApproval && fr.Upper <= policy.MaxFalseRejection
		rows = append(rows, SweepRow{Threshold: t, FalseApproval: fa, FalseRejection: fr, Feasible: feasible})
	}
	return rows
}

// Recommend picks, among the feasible thresholds, the one with the lowest false
// rejection; a tie goes to the higher threshold, the safer side.
func Recommend(rows []SweepRow) (float64, bool) {
	best := -1
	for i, row := range rows {
		if !row.Feasible {
			continue
		}
		if best < 0 || row.FalseRejection.Value <= rows[best].FalseRejection.Value {
			best = i
		}
	}
	if best < 0 {
		return 0, false
	}
	return rows[best].Threshold, true
}

// SampleDeficits names the shortfall of a selection set against the policy: how
// many more independent cases are needed before the limit can be certified at all.
// The defect negatives count at their effective size, because mutants of one
// parent are not independent; the golden positives are one per parent.
func SampleDeficits(dev []CaseStat, policy Policy) []string {
	var out []string
	neg := defects(dev)
	effective := EffectiveSample(len(neg), parentCount(neg), policy.ClusterRho)
	if need := MinSample(policy.MaxFalseApproval); effective < float64(need) {
		out = append(out, fmt.Sprintf("defect negatives: have %d from %d parents (effective %.0f at correlation %.2f), need an effective %d to certify the limit", len(neg), parentCount(neg), effective, policy.ClusterRho, need))
	}
	if need, have := MinSample(policy.MaxFalseRejection), len(positives(dev)); have < need {
		out = append(out, sampleNote("golden positives (one per parent)", have, need))
	}
	return out
}
