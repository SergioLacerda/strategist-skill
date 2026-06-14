package domain

import "fmt"

// Pipeline bypass reasons and mission route identifiers.
const (
	PipelineBypassDetectedReason = "pipeline_bypass_detected"
	MissionRouteMain             = "main"
	MissionRouteQuickDraw        = "quick_draw"
)

// PipelineEvidence captures the mission state used to detect pipeline bypass attempts.
type PipelineEvidence struct {
	Route              string
	BasePath           string
	MissionID          string
	AttemptedAction    string
	DiscoveryPresent   bool
	RefinementPresent  bool
	TasksPresent       bool
	GatePresented      bool
	GateApproved       bool
	QuickDrawPresented bool
	QuickDrawApproved  bool
}

// PipelineBypassDecision reports whether an attempted action is allowed to proceed
// outside the standard mission pipeline.
type PipelineBypassDecision struct {
	Allowed         bool
	Reason          string
	AttemptedAction string
	ExpectedPhase   string
	MissingEvidence string
	ResumeHint      string
}

func (d PipelineBypassDecision) Error() string {
	return fmt.Sprintf(
		"%s attempted_action=%s expected_phase=%s missing_evidence=%s resume_hint=%s",
		d.Reason, d.AttemptedAction, d.ExpectedPhase, d.MissingEvidence, d.ResumeHint,
	)
}

// EvaluatePipelineBypass determines whether the given mission evidence indicates
// a pipeline bypass and, if so, returns the expected phase and resume hint.
func EvaluatePipelineBypass(e PipelineEvidence) PipelineBypassDecision {
	e = normalizePipelineEvidence(e)
	if e.Route == MissionRouteQuickDraw {
		return evaluateQuickDrawBypass(e)
	}
	if !e.DiscoveryPresent {
		return blockedBypassDecision(
			e,
			"ranger",
			fmt.Sprintf("%s/pending/%s-discovery.md", e.BasePath, e.MissionID),
			fmt.Sprintf("re-invoke the mission through the full pipeline so Ranger can produce %s/pending/%s-discovery.md", e.BasePath, e.MissionID),
		)
	}
	if !e.RefinementPresent || !e.TasksPresent {
		missing := fmt.Sprintf("%s/refined/%s/tasks.md", e.BasePath, e.MissionID)
		if !e.RefinementPresent {
			missing = fmt.Sprintf("%s/refined/%s/", e.BasePath, e.MissionID)
		}
		return blockedBypassDecision(
			e,
			"archivist",
			missing,
			fmt.Sprintf("resume at Archivist by generating the refined plan under %s/refined/%s/", e.BasePath, e.MissionID),
		)
	}
	if !e.GatePresented || !e.GateApproved {
		return blockedBypassDecision(
			e,
			"approval_gate",
			"approval_gate:approved",
			fmt.Sprintf("present the approval gate for mission %s and wait for explicit user approval before execution", e.MissionID),
		)
	}
	return PipelineBypassDecision{Allowed: true}
}

func evaluateQuickDrawBypass(e PipelineEvidence) PipelineBypassDecision {
	if !e.QuickDrawPresented || !e.QuickDrawApproved {
		return blockedBypassDecision(
			e,
			"quick_draw_gate",
			fmt.Sprintf("%s/pending/%s-quick-draw.md + quick_draw_gate:approved", e.BasePath, e.MissionID),
			fmt.Sprintf("route this prompt through quick draw and wait for gate approval before writing %s/todo/", e.BasePath),
		)
	}
	return PipelineBypassDecision{Allowed: true}
}

func blockedBypassDecision(e PipelineEvidence, expectedPhase, missingEvidence, resumeHint string) PipelineBypassDecision {
	return PipelineBypassDecision{
		Allowed:         false,
		Reason:          PipelineBypassDetectedReason,
		AttemptedAction: e.AttemptedAction,
		ExpectedPhase:   expectedPhase,
		MissingEvidence: missingEvidence,
		ResumeHint:      resumeHint,
	}
}

func normalizePipelineEvidence(e PipelineEvidence) PipelineEvidence {
	if e.Route == "" {
		e.Route = MissionRouteMain
	}
	if e.BasePath == "" {
		e.BasePath = ".analysis"
	}
	if e.AttemptedAction == "" {
		e.AttemptedAction = "direct repository mutation"
	}
	return e
}
