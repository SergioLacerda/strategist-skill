// Package criticalhit evaluates the deterministic Critical Hit trigger.
package criticalhit

import (
	"path/filepath"

	"github.com/SergioLacerda/strategist-skill/internal/domain"
)

// Mode selects which trigger_conditions set (see
// contracts/machine/critical-hit.yaml) EvaluateCriticalHit checks evidence against.
type Mode string

// Critical Hit modes, matching the plain_move/closure_move trigger_conditions
// sets in contracts/machine/critical-hit.yaml.
const (
	ModePlain   Mode = "plain"
	ModeClosure Mode = "closure"
)

const (
	criticalHitReasonConditionsNotMet = "conditions_not_met"
	criticalHitFallbackFullPipeline   = "full_pipeline"
)

// Evidence mirrors PipelineEvidence's shape: plain fields, no I/O,
// no interfaces. Presence/explicitness flags are pre-computed by the caller —
// this function never judges evidence content (e.g. whether a supplied evidence
// summary is genuine), only its declared presence.
type Evidence struct {
	Mode                                       Mode
	TaskType                                   string
	SourcePath                                 string
	TargetPath                                 string
	BasePath                                   string
	FileTypes                                  []string
	RiskLevel                                  string
	FileCount                                  int
	ExplicitCompletionClaim                    bool
	EvidenceSummaryPresent                     bool
	CompletionInferredFromCodeOnly             bool
	PartialImplementationWithDeclaredResiduals bool
}

// Decision reports whether the supplied evidence satisfies Critical
// Hit's plain-move or closure-move trigger conditions.
type Decision struct {
	Allowed       bool
	Mode          Mode
	Reason        string
	FallbackRoute string
}

// EvaluateEligibility is the Mechanism-backed deterministic policy boundary.
// It is deliberately pure so contextual Feat activation remains separate from
// path, evidence, and safety predicates.
func EvaluateEligibility(e Evidence) Decision {
	switch e.Mode {
	case ModePlain:
		return evaluatePlainMove(e)
	case ModeClosure:
		return evaluateClosureMove(e)
	default:
		return blockedCriticalHitDecision(e.Mode)
	}
}

// EvaluateCriticalHit runs Critical Hit through its deterministic activation
// flow. A mode must select one of the contract's declared trigger condition
// sets; unknown modes fail closed to the full pipeline.
func EvaluateCriticalHit(e Evidence) Decision {
	return EvaluateEligibility(e)
}

func evaluatePlainMove(e Evidence) Decision {
	analysisFolders := []string{
		filepath.Join(e.BasePath, "pending"),
		filepath.Join(e.BasePath, "refined"),
		filepath.Join(e.BasePath, "archived"),
	}

	ok := e.TaskType == "analysis_move" &&
		pathWithinAny(analysisFolders, e.SourcePath) &&
		pathWithinAny(analysisFolders, e.TargetPath) &&
		onlyFileType(e.FileTypes, ".md") &&
		e.RiskLevel == "low" &&
		e.FileCount <= 5 &&
		!e.ExplicitCompletionClaim

	if !ok {
		return blockedCriticalHitDecision(ModePlain)
	}
	return Decision{Allowed: true, Mode: ModePlain}
}

func evaluateClosureMove(e Evidence) Decision {
	analysisFolders := []string{
		filepath.Join(e.BasePath, "pending"),
		filepath.Join(e.BasePath, "refined"),
	}
	doneFolder := filepath.Join(e.BasePath, "done")

	ok := e.TaskType == "analysis_move" &&
		pathWithinAny(analysisFolders, e.SourcePath) &&
		domain.PathWithin(doneFolder, e.TargetPath) &&
		e.ExplicitCompletionClaim &&
		e.EvidenceSummaryPresent &&
		!e.CompletionInferredFromCodeOnly &&
		!e.PartialImplementationWithDeclaredResiduals

	if !ok {
		return blockedCriticalHitDecision(ModeClosure)
	}
	return Decision{Allowed: true, Mode: ModeClosure}
}

func blockedCriticalHitDecision(mode Mode) Decision {
	return Decision{
		Allowed:       false,
		Mode:          mode,
		Reason:        criticalHitReasonConditionsNotMet,
		FallbackRoute: criticalHitFallbackFullPipeline,
	}
}

func pathWithinAny(prefixes []string, path string) bool {
	for _, prefix := range prefixes {
		if domain.PathWithin(prefix, path) {
			return true
		}
	}
	return false
}

func onlyFileType(fileTypes []string, want string) bool {
	if len(fileTypes) == 0 {
		return false
	}
	for _, ft := range fileTypes {
		if ft != want {
			return false
		}
	}
	return true
}
