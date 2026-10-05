package calibration

import (
	"crypto/sha256"
	"fmt"
)

// CrossValidation is the grouped k-fold reading of the whole selection procedure:
// each fold's threshold is chosen on the other folds and judged on the fold, and
// the pooled out-of-fold rates stand for what a threshold chosen this way does on
// packages it has not seen. Every package, with all its mutants, is in one fold.
type CrossValidation struct {
	Folds      int       `json:"folds"`
	Thresholds []float64 `json:"fold_thresholds"`
	// Infeasible counts folds whose training part had no feasible threshold.
	Infeasible     int  `json:"infeasible_folds,omitempty"`
	FalseApproval  Rate `json:"false_approval"`
	FalseRejection Rate `json:"false_rejection"`
}

// FoldOf is the fold of a parent: a pure function of its id, so a package and its
// mutants always fall together and the assignment never changes between runs.
func FoldOf(parent string, folds int) int {
	if folds <= 0 {
		return 0
	}
	sum := sha256.Sum256([]byte("fold|" + parent))
	return (int(sum[0])<<24 | int(sum[1])<<16 | int(sum[2])<<8 | int(sum[3])) % folds //nolint:gosec // the modulo bounds the value before conversion
}

func foldKey(c CaseStat) string {
	if c.ParentID != "" {
		return c.ParentID
	}
	return c.ID
}

func foldSplit(stats []CaseStat, fold, folds int) (train, test []CaseStat) {
	for _, stat := range stats {
		if FoldOf(foldKey(stat), folds) == fold {
			test = append(test, stat)
		} else {
			train = append(train, stat)
		}
	}
	return train, test
}

type foldTally struct{ faK, faN, frK, frN int }

func (t *foldTally) add(test []CaseStat, threshold float64) {
	neg, pos := defects(test), positives(test)
	t.faK, t.faN = t.faK+countApproved(neg, threshold), t.faN+len(neg)
	t.frK, t.frN = t.frK+len(pos)-countApproved(pos, threshold), t.frN+len(pos)
}

// CrossValidate runs the selection procedure `choose` inside every fold.
func CrossValidate(stats []CaseStat, policy Policy, folds int, choose func([]SweepRow) (float64, bool)) CrossValidation {
	out := CrossValidation{Folds: folds}
	var tally foldTally
	for fold := 0; fold < folds; fold++ {
		train, test := foldSplit(stats, fold, folds)
		threshold, ok := choose(Sweep(train, policy))
		if !ok {
			out.Infeasible++
			continue
		}
		out.Thresholds = append(out.Thresholds, threshold)
		tally.add(test, threshold)
	}
	neg := defects(stats)
	out.FalseApproval = NewClusterRate(tally.faK, tally.faN, EffectiveSample(tally.faN, parentCount(neg), policy.ClusterRho))
	out.FalseRejection = NewRate(tally.frK, tally.frN)
	return out
}

// Confirmed reports whether the out-of-fold upper bounds are inside the policy and
// every fold found a threshold.
func (cv CrossValidation) Confirmed(policy Policy) bool {
	return cv.Infeasible == 0 && cv.FalseApproval.Upper <= policy.MaxFalseApproval && cv.FalseRejection.Upper <= policy.MaxFalseRejection
}

// Note is the one line that says why a cross-validation did not confirm.
func (cv CrossValidation) Note(policy Policy) string {
	return fmt.Sprintf("cross-validation over %d folds: false approval upper bound %.3f (limit %.3f), false rejection upper bound %.3f (limit %.3f), %d folds without a feasible threshold",
		cv.Folds, cv.FalseApproval.Upper, policy.MaxFalseApproval, cv.FalseRejection.Upper, policy.MaxFalseRejection, cv.Infeasible)
}
