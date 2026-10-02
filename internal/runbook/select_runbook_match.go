package runbook

import (
	"regexp"
	"strings"
	"unicode"
	"unicode/utf8"
)

// placeholderGroup matches a value placeholder such as <discovery|refinement|execution>.
var placeholderGroup = regexp.MustCompile(`<[^<>]*>`)

type scoredCandidate struct {
	runbook Runbook
	score   int
	matched []string
}

func scoreCandidates(candidates []Runbook, signals MissionSignals) []scoredCandidate {
	scored := make([]scoredCandidate, 0, len(candidates))
	for _, rb := range candidates {
		matched := matchAppliesWhen(rb.AppliesWhen, signals)
		matched = append(matched, matchDeclaredSignals(rb.Signals, signals)...)
		scored = append(scored, scoredCandidate{runbook: rb, score: len(matched), matched: matched})
	}
	return scored
}

// matchDeclaredSignals reports which of a candidate's declared Signals
// (runbook.go, an optional controlled-vocabulary shortcut validated by
// ValidateRunbook) are present in signals, resolved through the same
// canonical-vocabulary machinery triggerMatchesAnySignal uses for
// applies_when prose. It supplements matchAppliesWhen's result rather than
// replacing it: a candidate with no declared Signals contributes nothing
// here, so scoreCandidates' behavior is unchanged for every sidecar that
// predates this field. Each match is prefixed "signal:" so it reads
// distinctly from a matched applies_when entry in Select's Reason string.
func matchDeclaredSignals(declared []string, signals MissionSignals) []string {
	var matched []string
	for _, d := range declared {
		if declaredSignalMatches(d, signals) {
			matched = append(matched, "signal:"+d)
		}
	}
	return matched
}

func declaredSignalMatches(declared string, signals MissionSignals) bool {
	canonical := CanonicalSignal(declared)
	for _, signal := range signals {
		if signal == "" {
			continue
		}
		if canonicalSignalsIn(signal)[canonical] {
			return true
		}
	}
	return false
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
	triggerCanonical := canonicalSignalsIn(trigger)
	for _, signal := range signals {
		if signal == "" {
			continue
		}
		if canonicalTriggerMatches(triggerCanonical, signal) {
			return true
		}
		if rawTriggerMatches(trigger, signal) {
			return true
		}
	}
	return false
}

func canonicalTriggerMatches(triggerCanonical map[CanonicalSignal]bool, signal string) bool {
	return len(triggerCanonical) > 0 && sharesCanonicalSignal(triggerCanonical, canonicalSignalsIn(signal))
}

// rawTriggerMatches is the free-text fallback: the signal must appear in the trigger as
// whole words, after placeholder groups such as <discovery|refinement|execution> are
// dropped. Those groups list the values a field may take; a mission signal that equals
// one of them says nothing about the situation the trigger describes.
func rawTriggerMatches(trigger, signal string) bool {
	signal = strings.ToLower(strings.TrimSpace(signal))
	if signal == "" {
		return false
	}
	text := placeholderGroup.ReplaceAllString(strings.ToLower(trigger), " ")
	return containsAtWordBoundary(text, signal)
}

// containsAtWordBoundary reports whether term occurs in text with no letter, digit or
// underscore directly touching either end. A term that starts or ends with punctuation
// (for example an error=token) needs no boundary on that side.
func containsAtWordBoundary(text, term string) bool {
	for from := 0; from < len(text); {
		i := strings.Index(text[from:], term)
		if i < 0 {
			return false
		}
		start := from + i
		end := start + len(term)
		if boundaryBefore(text, start, term) && boundaryAfter(text, end, term) {
			return true
		}
		from = start + 1
	}
	return false
}

func boundaryBefore(text string, start int, term string) bool {
	first, _ := utf8.DecodeRuneInString(term)
	if !isWordRune(first) || start == 0 {
		return true
	}
	prev, _ := utf8.DecodeLastRuneInString(text[:start])
	return !isWordRune(prev)
}

func boundaryAfter(text string, end int, term string) bool {
	last, _ := utf8.DecodeLastRuneInString(term)
	if !isWordRune(last) || end >= len(text) {
		return true
	}
	next, _ := utf8.DecodeRuneInString(text[end:])
	return !isWordRune(next)
}

func isWordRune(r rune) bool {
	return unicode.IsLetter(r) || unicode.IsDigit(r) || r == '_'
}
