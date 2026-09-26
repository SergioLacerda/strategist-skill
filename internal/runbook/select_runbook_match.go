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
