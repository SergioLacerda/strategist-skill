package calibration

import (
	"fmt"
	"sort"
)

// CoverageRow is, for one defect operator, how many of its cases each protection
// catches at the reference threshold: the judge, the local checks, and the policy
// as a whole (the judge not approving, or any local check flagging).
type CoverageRow struct {
	Operator string `json:"operator"`
	Cases    int    `json:"cases"`
	Judge    Rate   `json:"judge_caught"`
	Local    Rate   `json:"local_caught"`
	Policy   Rate   `json:"policy_caught"`
	// Delegated is true when the operator is declared local scope: its protection is
	// the local checks', and the judge gates do not read it.
	Delegated bool `json:"delegated_to_local,omitempty"`
}

// PolicyLevel is the measure of the whole decision, judge and local checks
// together, at one threshold. It exists only when the local checks were evaluated.
type PolicyLevel struct {
	Threshold      float64       `json:"threshold"`
	FalseApproval  Rate          `json:"false_approval"`
	FalseRejection Rate          `json:"false_rejection"`
	GoldenFlagged  Rate          `json:"golden_flagged_by_local"`
	Coverage       []CoverageRow `json:"coverage"`
}

func approvedByPolicy(c CaseStat, threshold float64) bool {
	return c.ApprovedAt(threshold) && len(c.Flags) == 0
}

// evaluatedLocally keeps the cases whose local checks were evaluated.
func evaluatedLocally(stats []CaseStat) []CaseStat {
	return pick(stats, func(c CaseStat) bool { return c.Local })
}

// policyLevel measures the whole decision over every counted case, delegated
// operators included. It returns nil when no case carries local flags.
func policyLevel(stats []CaseStat, threshold float64, cfg Config) *PolicyLevel {
	local := evaluatedLocally(stats)
	if len(local) == 0 {
		return nil
	}
	pos, neg := positives(local), defects(local)
	falseAcc := countWhere(neg, func(c CaseStat) bool { return approvedByPolicy(c, threshold) })
	falseRej := countWhere(pos, func(c CaseStat) bool { return !approvedByPolicy(c, threshold) })
	flagged := countWhere(pos, func(c CaseStat) bool { return len(c.Flags) > 0 })
	return &PolicyLevel{
		Threshold:      threshold,
		FalseApproval:  NewClusterRate(falseAcc, len(neg), EffectiveSample(len(neg), parentCount(neg), cfg.Policy.ClusterRho)),
		FalseRejection: NewRate(falseRej, len(pos)),
		GoldenFlagged:  NewRate(flagged, len(pos)),
		Coverage:       coverage(neg, threshold, cfg.LocalScope),
	}
}

func countWhere(cases []CaseStat, hit func(CaseStat) bool) int {
	n := 0
	for _, c := range cases {
		if hit(c) {
			n++
		}
	}
	return n
}

func coverage(neg []CaseStat, threshold float64, scope map[string]bool) []CoverageRow {
	type tally struct{ n, judge, local, policy int }
	byOp := map[string]*tally{}
	for _, c := range neg {
		t := byOp[c.Operator]
		if t == nil {
			t = &tally{}
			byOp[c.Operator] = t
		}
		t.n++
		judge, local := !c.ApprovedAt(threshold), len(c.Flags) > 0
		t.judge += b2i(judge)
		t.local += b2i(local)
		t.policy += b2i(judge || local)
	}
	rows := make([]CoverageRow, 0, len(byOp))
	for op, t := range byOp {
		rows = append(rows, CoverageRow{Operator: op, Cases: t.n, Judge: NewRate(t.judge, t.n), Local: NewRate(t.local, t.n), Policy: NewRate(t.policy, t.n), Delegated: scope[op]})
	}
	sort.Slice(rows, func(i, j int) bool { return rows[i].Operator < rows[j].Operator })
	return rows
}

func b2i(v bool) int {
	if v {
		return 1
	}
	return 0
}

// LocalCoverageGate checks that every operator declared local scope is caught by
// the local checks: the lower bound of its detection rate must reach the limit.
// An operator cannot be both unseen by the judge and unprotected by the local
// checks and still be called calibrated.
func LocalCoverageGate(rows []CoverageRow, scope map[string]bool, minLower float64) (Gate, bool) {
	if len(scope) == 0 {
		return Gate{}, false
	}
	measured, worst, failing := delegatedCoverage(rows, scope, minLower)
	if measured < len(scope) {
		return unmeasured(GateLocalCoverage, minLower, "a delegated operator has no case with local checks evaluated"), true
	}
	detail := fmt.Sprintf("%d delegated operators", measured)
	if len(failing) > 0 {
		detail += "; not covered by the local checks: " + fmt.Sprint(failing)
	}
	return decide(GateLocalCoverage, worst, minLower, len(failing) == 0, detail), true
}

// delegatedCoverage returns how many delegated operators have cases, the worst lower
// bound of their local detection rate and the ones below the limit.
func delegatedCoverage(rows []CoverageRow, scope map[string]bool, minLower float64) (int, float64, []string) {
	measured, worst, failing := 0, 1.0, []string{}
	for _, row := range rows {
		if !scope[row.Operator] {
			continue
		}
		measured++
		lower := lowerBound(row.Local.K, row.Local.N, OneSidedAlpha)
		worst = min(worst, lower)
		if lower < minLower {
			failing = append(failing, fmt.Sprintf("%s (%d of %d, lower bound %.3f)", row.Operator, row.Local.K, row.Local.N, lower))
		}
	}
	return measured, worst, failing
}
