package runbook

import "strings"

type scoredCandidate struct {
	runbook Runbook
	score   int
	matched []string
}

func scoreCandidates(candidates []Runbook, signals MissionSignals) []scoredCandidate {
	scored := make([]scoredCandidate, 0, len(candidates))
	for _, rb := range candidates {
		matched := matchAppliesWhen(rb.AppliesWhen, signals)
		scored = append(scored, scoredCandidate{runbook: rb, score: len(matched), matched: matched})
	}
	return scored
}

func matchAppliesWhen(appliesWhen []string, signals MissionSignals) []string {
	var matched []string
	for _, trigger := range appliesWhen {
		if trigger == "" {
			continue
		}
		if triggerMatchesAnySignal(trigger, signals) {
			matched = append(matched, trigger)
		}
	}
	return matched
}

// triggerMatchesAnySignal reports whether trigger (one applies_when entry)
// matches any of signals. It first consults the controlled signal
// vocabulary (signal_vocabulary.go): if trigger and a signal both resolve
// to the same CanonicalSignal, they match even when neither string is a
// substring of the other (e.g. trigger "CI test suite is red" and signal
// "flaky test" both resolve to SignalCITestFailure). When the vocabulary
// yields no shared canonical signal, it falls back to the original
// case-insensitive raw substring match, so free-text triggers/signals with
// no controlled-vocabulary coverage still behave exactly as before.
func triggerMatchesAnySignal(trigger string, signals MissionSignals) bool {
	lowerTrigger := strings.ToLower(trigger)
	triggerCanonical := canonicalSignalsIn(trigger)
	for _, signal := range signals {
		if signal == "" {
			continue
		}
		if len(triggerCanonical) > 0 && sharesCanonicalSignal(triggerCanonical, canonicalSignalsIn(signal)) {
			return true
		}
		if strings.Contains(lowerTrigger, strings.ToLower(signal)) {
			return true
		}
	}
	return false
}
