package domain

import "path/filepath"

// CriticalHitMode selects which trigger_conditions set (see
// contracts/machine/critical-hit.yaml) EvaluateCriticalHit checks evidence against.
type CriticalHitMode string

// Critical Hit modes, matching the plain_move/closure_move trigger_conditions
// sets in contracts/machine/critical-hit.yaml.
const (
	CriticalHitModePlain   CriticalHitMode = "plain"
	CriticalHitModeClosure CriticalHitMode = "closure"
)

const (
	criticalHitReasonConditionsNotMet = "conditions_not_met"
	criticalHitFallbackFullPipeline   = "full_pipeline"
)

// CriticalHitEvidence mirrors PipelineEvidence's shape: plain fields, no I/O,
// no interfaces. Presence/explicitness flags are pre-computed by the caller —
// this function never judges evidence content (e.g. whether a supplied evidence
// summary is genuine), only its declared presence.
type CriticalHitEvidence struct {
	Mode                                       CriticalHitMode
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

// CriticalHitDecision reports whether the supplied evidence satisfies Critical
// Hit's plain-move or closure-move trigger conditions.
type CriticalHitDecision struct {
	Allowed       bool
	Mode          CriticalHitMode
	Reason        string
	FallbackRoute string
}

// EvaluateCriticalHit runs Critical Hit through its deterministic activation
// flow. A mode must select one of the contract's declared trigger condition
// sets; unknown modes fail closed to the full pipeline.
func EvaluateCriticalHit(e CriticalHitEvidence) CriticalHitDecision {
	switch e.Mode {
	case CriticalHitModePlain:
		return evaluatePlainMove(e)
	case CriticalHitModeClosure:
		return evaluateClosureMove(e)
	default:
		return blockedCriticalHitDecision(e.Mode)
	}
}

func evaluatePlainMove(e CriticalHitEvidence) CriticalHitDecision {
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
		return blockedCriticalHitDecision(CriticalHitModePlain)
	}
	return CriticalHitDecision{Allowed: true, Mode: CriticalHitModePlain}
}

func evaluateClosureMove(e CriticalHitEvidence) CriticalHitDecision {
	analysisFolders := []string{
		filepath.Join(e.BasePath, "pending"),
		filepath.Join(e.BasePath, "refined"),
	}
	doneFolder := filepath.Join(e.BasePath, "done")

	ok := e.TaskType == "analysis_move" &&
		pathWithinAny(analysisFolders, e.SourcePath) &&
		slotWritePathAllowed(doneFolder, e.TargetPath) &&
		e.ExplicitCompletionClaim &&
		e.EvidenceSummaryPresent &&
		!e.CompletionInferredFromCodeOnly &&
		!e.PartialImplementationWithDeclaredResiduals

	if !ok {
		return blockedCriticalHitDecision(CriticalHitModeClosure)
	}
	return CriticalHitDecision{Allowed: true, Mode: CriticalHitModeClosure}
}

func blockedCriticalHitDecision(mode CriticalHitMode) CriticalHitDecision {
	return CriticalHitDecision{
		Allowed:       false,
		Mode:          mode,
		Reason:        criticalHitReasonConditionsNotMet,
		FallbackRoute: criticalHitFallbackFullPipeline,
	}
}

func pathWithinAny(prefixes []string, path string) bool {
	for _, prefix := range prefixes {
		if slotWritePathAllowed(prefix, path) {
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
