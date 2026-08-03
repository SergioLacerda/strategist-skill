package runbook

// StepResult is EvaluateStep's verdict for a single Check.
type StepResult struct {
	CheckID string
	Level   Level
	// Satisfied is true when the check holds — either because qualifying
	// evidence backs it, or because a justification explicitly excepts it.
	Satisfied bool
	// Excepted is true when Satisfied is true only because of a
	// justification, not because qualifying evidence was found.
	Excepted bool
}

// EvaluateStep evaluates one Check against the evidence gathered for it and
// an optional justification. A check is satisfied outright when at least
// one EvidenceRef qualifies (see EvidenceRef.qualifies); lacking that, an
// explicit justification still satisfies it, but as a recorded exception —
// mirrors runbook_v2.txt's own "Não pode avançar sem cumprir ou justificar
// exceção" framing, evaluated per-step here so ValidateCompletion can
// aggregate without re-deriving this logic.
func EvaluateStep(check Check, evidence []EvidenceRef, justification string) StepResult {
	satisfied, excepted := evaluateSatisfaction(hasQualifyingEvidence(evidence), justification)
	return StepResult{
		CheckID:   check.ID,
		Level:     check.Level,
		Satisfied: satisfied,
		Excepted:  excepted,
	}
}

func hasQualifyingEvidence(evidence []EvidenceRef) bool {
	for _, e := range evidence {
		if e.qualifies() {
			return true
		}
	}
	return false
}

// evaluateSatisfaction is the shared rule EvaluateStep and ValidateCompletion
// both apply: qualifying evidence (or, for ValidateCompletion's caller-
// supplied bool, a raw "satisfied" signal) satisfies a check outright; a
// justification satisfies it as an exception; neither leaves it unsatisfied.
func evaluateSatisfaction(rawSatisfied bool, justification string) (satisfied, excepted bool) {
	if rawSatisfied {
		return true, false
	}
	if justification != "" {
		return true, true
	}
	return false, false
}
