package domain

import "fmt"

// Pipeline bypass reasons and mission route identifiers.
const (
	PipelineBypassDetectedReason = "pipeline_bypass_detected"
	MissionRouteMain             = "main"
	MissionRouteDirectExecute    = "direct_execute"
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
	DirectGateApproved bool
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
	if e.Route == MissionRouteDirectExecute {
		return evaluateDirectExecuteBypass(e)
	}
	return evaluateMainRouteBypass(e)
}

func evaluateMainRouteBypass(e PipelineEvidence) PipelineBypassDecision {
	if !e.DiscoveryPresent {
		return discoveryBypassDecision(e)
	}
	if !e.RefinementPresent || !e.TasksPresent {
		return refinementBypassDecision(e)
	}
	if !e.GatePresented || !e.GateApproved {
		return approvalGateBypassDecision(e)
	}
	return PipelineBypassDecision{Allowed: true}
}

func discoveryBypassDecision(e PipelineEvidence) PipelineBypassDecision {
	return blockedBypassDecision(
		e,
		"ranger",
		fmt.Sprintf("%s/refined/%s/analysis.md", e.BasePath, e.MissionID),
		fmt.Sprintf("re-invoke the mission through the full pipeline so Ranger and Archivist can produce %s/refined/%s/analysis.md", e.BasePath, e.MissionID),
	)
}

func refinementBypassDecision(e PipelineEvidence) PipelineBypassDecision {
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

func approvalGateBypassDecision(e PipelineEvidence) PipelineBypassDecision {
	return blockedBypassDecision(
		e,
		"approval_gate",
		"approval_gate:approved",
		fmt.Sprintf("present the approval gate for mission %s and wait for explicit user approval before execution", e.MissionID),
	)
}

func evaluateDirectExecuteBypass(e PipelineEvidence) PipelineBypassDecision {
	if !e.DirectGateApproved {
		return blockedBypassDecision(
			e,
			"direct_gate",
			fmt.Sprintf("direct_gate:approved (mission %s)", e.MissionID),
			fmt.Sprintf("present the Critical Hit gate for mission %s and wait for user confirmation before writing", e.MissionID),
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
