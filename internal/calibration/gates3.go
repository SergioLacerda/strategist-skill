package calibration

import "fmt"

// GateThresholdBand is read for gates-3 after a threshold is feasible.
const GateThresholdBand = "threshold_band"

// sweepStep is the spacing of the candidate thresholds.
const sweepStep = 0.01

// Gates3Limits are the limits of the pre-registered rule: gates-2 plus a
// five-fold cross-validation and a band at least twice the repetition noise wide.
func Gates3Limits() Limits {
	limits := DefaultLimits()
	limits.Rule, limits.CVFolds, limits.BandNoiseFactor = RuleGates3, 5, 2
	return limits
}

// widestBand is the first index and the length of the longest run of consecutive
// feasible thresholds, 0 and 0 when none is feasible.
func widestBand(rows []SweepRow) (int, int) {
	bestStart, bestLen, start := 0, 0, 0
	for i, row := range rows {
		if !row.Feasible {
			start = i + 1
			continue
		}
		if length := i - start + 1; length > bestLen {
			bestStart, bestLen = start, length
		}
	}
	return bestStart, bestLen
}

// RecommendCentered places the threshold at the middle of the widest feasible
// band instead of at its edge, so a small shift of the judge's scores does not
// leave the band. It reports false when no threshold is feasible.
func RecommendCentered(rows []SweepRow) (float64, bool) {
	start, length := widestBand(rows)
	if length == 0 {
		return 0, false
	}
	return rows[start+length/2].Threshold, true
}

func repeatedCases(stats []CaseStat) int {
	n := 0
	for _, stat := range stats {
		if stat.Answered() >= 2 {
			n++
		}
	}
	return n
}

// bandGate requires the feasible band to be wider than the judge's own run-to-run
// noise: a band narrower than that moves with the next run, and so does the threshold.
func bandGate(rows []SweepRow, sel []CaseStat, limits Limits) Gate {
	if repeatedCases(sel) == 0 {
		return unmeasured(GateThresholdBand, 0, "needs at least two repetitions to know the noise")
	}
	_, length := widestBand(rows)
	width, noise := float64(length)*sweepStep, noiseP95(sel)
	need := limits.BandNoiseFactor * noise
	return decide(GateThresholdBand, width, need, width >= need,
		fmt.Sprintf("feasible thresholds span %.2f, the repetition noise is %.4f (p95 standard deviation) and the band must be at least %.1f times it", width, noise, limits.BandNoiseFactor))
}

// KnownGateRule reports whether rule names a gate rule; the empty rule is the default (gates-2).
func KnownGateRule(rule string) bool {
	return rule == "" || rule == RuleGates1 || rule == RuleGates2 || rule == RuleGates3
}
