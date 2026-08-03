package telemetry

// HandoffMetrics reports the handoff governance metrics computable from
// ChallengeRecord history, per
// .analysis/refined/20260803-handoff-challenge-extensions/design.md § Item
// 3 (same shape as RouteMetrics, applied to Handoff Challenges instead of
// Scout routing decisions).
type HandoffMetrics struct {
	// HandoffPassRate is the share of all recorded attempts that passed.
	HandoffPassRate float64
	// FirstAttemptPassRate is the share of attempt==1 records that passed —
	// how often a handoff succeeds with no repair loop.
	FirstAttemptPassRate float64
	// CriticalConstraintRecall is an attempt-level proxy for recall: the
	// share of attempts with zero MissingRefs. See SemanticHandoffLoss's
	// doc comment for why this is an attempt-level approximation rather
	// than a true item-level fraction.
	CriticalConstraintRecall float64
	// DecisionClassificationAccuracy is the share of attempts with zero
	// MisclassifiedRefs.
	DecisionClassificationAccuracy float64
	// ScopeViolationRate is the share of attempts whose MissingChallenges
	// includes "boundary" — the boundary challenge type was required but
	// never satisfied for that attempt.
	ScopeViolationRate float64
	// HandoffRepairRate is, among missions whose first attempt failed, the
	// share that have a later attempt on record (the return_to_archivist
	// repair loop was actually exercised, not just triggered).
	HandoffRepairRate float64
	SemanticLoss      SemanticHandoffLoss
	SampleSize        int
}

// SemanticHandoffLoss follows quiz.txt's own definition:
//
//	recall         = 1 - (correctly_recalled / critical_items)
//	classification = 1 - (correctly_classified / critical_items)
//	application    = 1 - (correctly_applied / critical_items)
//
// A ChallengeRecord carries no per-item critical_items count, only
// per-attempt pass/fail signals (MissingRefs, MisclassifiedRefs), so Recall
// and Classification here are attempt-level proxies — 1 minus
// CriticalConstraintRecall / DecisionClassificationAccuracy above — rather
// than true item-level fractions. Application requires evidence of whether
// Sniper's actual downstream work correctly incorporated an acknowledged
// constraint, which is outside anything a Handoff Challenge record alone
// can show; it is always 0 until such a signal exists, the same "no ground
// truth yet" posture RouteMetrics takes for its four reversal-dependent
// metrics.
type SemanticHandoffLoss struct {
	Recall         float64
	Classification float64
	Application    float64
}

// ComputeHandoffMetrics aggregates records into HandoffMetrics. Every rate
// is 0 when its sample is empty, not NaN, so callers (including `strategist
// metrics handoff` against a workspace with no handoff-challenges.jsonl
// yet) can render "0.00" instead of special-casing an empty history.
func ComputeHandoffMetrics(records []ChallengeRecord) HandoffMetrics {
	recall := safeRate(countWithNoMissingRefs(records), len(records))
	classification := safeRate(countWithNoMisclassifiedRefs(records), len(records))
	firstAttempts, passedFirstAttempts := countFirstAttempts(records)

	return HandoffMetrics{
		HandoffPassRate:                safeRate(countPassed(records), len(records)),
		FirstAttemptPassRate:           safeRate(passedFirstAttempts, firstAttempts),
		CriticalConstraintRecall:       recall,
		DecisionClassificationAccuracy: classification,
		ScopeViolationRate:             safeRate(countBoundaryMissing(records), len(records)),
		HandoffRepairRate:              computeRepairRate(records),
		SemanticLoss:                   semanticHandoffLoss(recall, classification, len(records)),
		SampleSize:                     len(records),
	}
}

// semanticHandoffLoss turns recall/classification accuracy into loss
// (1 - accuracy), except on an empty sample: accuracy is 0/0 = 0 there by
// safeRate's convention, and 1 - 0 would misreport "100% loss" for a
// sample that does not exist. Empty means 0 loss, same as every other
// HandoffMetrics rate.
func semanticHandoffLoss(recall, classification float64, sampleSize int) SemanticHandoffLoss {
	if sampleSize == 0 {
		return SemanticHandoffLoss{}
	}
	return SemanticHandoffLoss{Recall: 1 - recall, Classification: 1 - classification}
}

func countPassed(records []ChallengeRecord) int {
	n := 0
	for _, r := range records {
		if r.Passed {
			n++
		}
	}
	return n
}

func countWithNoMissingRefs(records []ChallengeRecord) int {
	n := 0
	for _, r := range records {
		if len(r.MissingRefs) == 0 {
			n++
		}
	}
	return n
}

func countWithNoMisclassifiedRefs(records []ChallengeRecord) int {
	n := 0
	for _, r := range records {
		if len(r.MisclassifiedRefs) == 0 {
			n++
		}
	}
	return n
}

func countBoundaryMissing(records []ChallengeRecord) int {
	n := 0
	for _, r := range records {
		if containsString(r.MissingChallenges, "boundary") {
			n++
		}
	}
	return n
}

func containsString(values []string, target string) bool {
	for _, v := range values {
		if v == target {
			return true
		}
	}
	return false
}

func countFirstAttempts(records []ChallengeRecord) (total, passed int) {
	for _, r := range records {
		if r.Attempt != 1 {
			continue
		}
		total++
		if r.Passed {
			passed++
		}
	}
	return total, passed
}

// computeRepairRate reports, among missions whose attempt==1 record
// failed, the share that have a later passing attempt on record.
func computeRepairRate(records []ChallengeRecord) float64 {
	byMission := groupChallengeRecordsByMission(records)
	failedFirst, repaired := 0, 0
	for _, attempts := range byMission {
		if !missionFirstAttemptFailed(attempts) {
			continue
		}
		failedFirst++
		if missionHasLaterPass(attempts) {
			repaired++
		}
	}
	return safeRate(repaired, failedFirst)
}

func groupChallengeRecordsByMission(records []ChallengeRecord) map[string][]ChallengeRecord {
	byMission := make(map[string][]ChallengeRecord)
	for _, r := range records {
		byMission[r.MissionID] = append(byMission[r.MissionID], r)
	}
	return byMission
}

func missionFirstAttemptFailed(attempts []ChallengeRecord) bool {
	for _, a := range attempts {
		if a.Attempt == 1 {
			return !a.Passed
		}
	}
	return false
}

func missionHasLaterPass(attempts []ChallengeRecord) bool {
	for _, a := range attempts {
		if a.Attempt > 1 && a.Passed {
			return true
		}
	}
	return false
}
