package calibration

import "fmt"

// judgeCriterion applies the pre-registered rule (criteria-1): enough golden parents, the
// separation limit, an out-of-fold false-flag rate within the false rejection limit, and a
// detection lower bound of at least MinCriterionDetection. A hint is advisory, so the
// false-flag limit reads the observed rate: an upper bound at 15 percent would need zero
// flagged parents out of 21, which a threshold read from the data cannot promise (about one
// parent in n+1 falls below the lowest of the others). The upper bound is reported alongside.
func judgeCriterion(res CriterionResult, policy Policy, limits Limits) (string, string) {
	if need := MinSample(policy.MaxFalseRejection); res.Golden < need {
		return CriterionNotMeasured, fmt.Sprintf("%d golden parents, need at least %d", res.Golden, need)
	}
	lower := detectionLower(res.Detection)
	switch {
	case res.AUC < limits.MinAUC:
		return CriterionUnreliable, fmt.Sprintf("auc %.3f below %.2f", res.AUC, limits.MinAUC)
	case res.FalseFlag.Value > policy.MaxFalseRejection:
		return CriterionUnreliable, fmt.Sprintf("golden parents flagged out of fold: %.3f above %.2f (upper bound %.3f)", res.FalseFlag.Value, policy.MaxFalseRejection, res.FalseFlag.Upper)
	case lower < MinCriterionDetection:
		return CriterionUnreliable, fmt.Sprintf("detection lower bound %.3f below %.2f", lower, MinCriterionDetection)
	}
	return CriterionReliable, ""
}

// detectionLower is the one-sided lower bound of the detection rate on its effective sample (the
// scaled count is rounded down, the conservative side for a lower bound).
func detectionLower(r Rate) float64 {
	n, k := r.N, r.K
	if r.Effective > 0 && r.Effective < r.N {
		k, n = scaledK(r), r.Effective
	}
	if n == 0 {
		return 0
	}
	return lowerBound(k, n, OneSidedAlpha)
}

func scaledK(r Rate) int {
	return min(int(float64(r.K)*float64(r.Effective)/float64(r.N)), r.Effective)
}
