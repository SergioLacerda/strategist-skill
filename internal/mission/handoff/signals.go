package handoff

import (
	"fmt"
	"path/filepath"
	"strings"
)

// SignalProvenance names where one true signal came from, so an outcome record
// can show why a challenge was required or skipped.
type SignalProvenance struct {
	Signal string `json:"signal"`
	Source string `json:"source"`
	Detail string `json:"detail,omitempty"`
}

// ExtractedSignals is the single validated-package derivation of RiskSignals.
type ExtractedSignals struct {
	Signals    RiskSignals
	Provenance []SignalProvenance
}

// ExtractRiskSignals derives the RiskSignals of a validated Archivist package.
// It is the only production source of those signals: task classifications come
// from tasks.md, the other facts from the typed block, and every true signal
// records its provenance. riskLevel is the optional coarse intake label; it can
// only make the result stricter and never overrides a concrete fact. A package
// that declares itself informational while carrying any require fact is
// contradictory and rejected.
func ExtractRiskSignals(refined, missionID, riskLevel string) (ExtractedSignals, error) {
	if err := ValidateArchivistPackage(refined, missionID); err != nil {
		return ExtractedSignals{}, fmt.Errorf("validate Archivist package: %w", err)
	}
	facts, err := readPolicyFacts(refined)
	if err != nil {
		return ExtractedSignals{}, err
	}
	tasks, err := readArchivistTasks(refined)
	if err != nil {
		return ExtractedSignals{}, err
	}
	extracted := factSignals(facts)
	if task, ok := firstImplementationHandoff(tasks); ok {
		extracted.add(&extracted.Signals.ImplementationHandoffPresent, PredicateImplementationHandoffPresent, "tasks.md", task)
	}
	if err := extracted.settleInformational(facts); err != nil {
		return ExtractedSignals{}, err
	}
	if err := extracted.applyCoarseRisk(riskLevel); err != nil {
		return ExtractedSignals{}, err
	}
	return extracted, nil
}

func readPolicyFacts(refined string) (PolicyFacts, error) {
	content, err := readArtifact(filepath.Join(refined, "analysis.md"))
	if err != nil {
		return PolicyFacts{}, fmt.Errorf("handoff_artifact_invalid: read Archivist analysis: %w", err)
	}
	frontmatter, _, err := parseFrontmatter(content)
	if err != nil {
		return PolicyFacts{}, fmt.Errorf("handoff_artifact_invalid: Archivist analysis: %w", err)
	}
	return ParsePolicyFacts(frontmatter)
}

func (e *ExtractedSignals) add(flag *bool, predicate PolicyPredicate, source, detail string) {
	*flag = true
	e.Provenance = append(e.Provenance, SignalProvenance{Signal: string(predicate), Source: source, Detail: detail})
}

func factSignals(facts PolicyFacts) ExtractedSignals {
	var e ExtractedSignals
	source := "analysis.md#" + PolicyFactsKey
	if len(facts.MandatoryConstraints) > 0 {
		e.add(&e.Signals.MandatoryConstraintsPresent, PredicateMandatoryConstraintsPresent, source, fmt.Sprintf("%d mandatory constraint(s)", len(facts.MandatoryConstraints)))
	}
	if len(facts.UnresolvedQuestions) > 0 {
		e.add(&e.Signals.UnresolvedQuestionsPresent, PredicateUnresolvedQuestionsPresent, source, fmt.Sprintf("%d unresolved question(s)", len(facts.UnresolvedQuestions)))
	}
	if len(facts.ForbiddenScope) > 0 {
		e.add(&e.Signals.ForbiddenScopePresent, PredicateForbiddenScopePresent, source, fmt.Sprintf("%d forbidden scope entr(ies)", len(facts.ForbiddenScope)))
	}
	if *facts.DestructiveOperationPossible {
		e.add(&e.Signals.DestructiveOperationPossible, PredicateDestructiveOperationPossible, source, "declared")
	}
	if *facts.SecuritySensitiveTask {
		e.add(&e.Signals.SecuritySensitiveTask, PredicateSecuritySensitiveTask, source, "declared")
	}
	return e
}

// settleInformational accepts a declared informational_only only when no
// require fact exists; otherwise the package contradicts itself.
func (e *ExtractedSignals) settleInformational(facts PolicyFacts) error {
	if !*facts.InformationalOnly {
		return nil
	}
	if len(e.Provenance) > 0 {
		return fmt.Errorf("handoff_policy_facts_contradictory: informational_only is declared but %s holds (%s)", e.Provenance[0].Signal, e.Provenance[0].Source)
	}
	e.Signals.InformationalOnly = true
	return nil
}

// applyCoarseRisk lets the intake risk_level add require signals. It never
// removes one and never makes a package informational.
func (e *ExtractedSignals) applyCoarseRisk(riskLevel string) error {
	source := "risk_level=" + riskLevel + " (supplemental, stricter only)"
	switch riskLevel {
	case "", "low":
		return nil
	case "medium":
		e.add(&e.Signals.MandatoryConstraintsPresent, PredicateMandatoryConstraintsPresent, "risk_level", source)
	case "high":
		e.add(&e.Signals.DestructiveOperationPossible, PredicateDestructiveOperationPossible, "risk_level", source)
		e.add(&e.Signals.SecuritySensitiveTask, PredicateSecuritySensitiveTask, "risk_level", source)
	default:
		return fmt.Errorf("handoff_risk_level_unknown: %q is not low, medium or high", riskLevel)
	}
	e.Signals.InformationalOnly = false
	return nil
}

// firstImplementationHandoff returns the id of the first task classified
// implementation_handoff, in either accepted classification spelling.
func firstImplementationHandoff(tasks []byte) (string, bool) {
	for _, line := range strings.Split(string(tasks), "\n") {
		line = strings.TrimSpace(line)
		if !strings.HasPrefix(line, "- [") {
			continue
		}
		if strings.Contains(line, "[implementation_handoff]") || strings.Contains(line, "[task_type: implementation_handoff]") {
			return taskID(line), true
		}
	}
	return "", false
}

func taskID(line string) string {
	fields := strings.Fields(line)
	for _, field := range fields[2:] {
		if len(field) > 0 && field[0] >= '0' && field[0] <= '9' {
			return field
		}
	}
	return "unnumbered"
}
