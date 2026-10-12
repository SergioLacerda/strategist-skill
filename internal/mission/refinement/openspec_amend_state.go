package refinement

import (
	"crypto/sha256"
	"fmt"

	"github.com/SergioLacerda/strategist-skill/internal/domain"
)

func (p *amendmentPlan) checkPersistedState() error {
	status := p.input.PersistedStatus
	if err := validatePersistedMission(status, p.input.MissionID); err != nil {
		return err
	}
	if err := validateGateEvidence(p.input.GateLabel, p.status); err != nil {
		return err
	}
	if !frontmatterMatchesState(p.status, status.State, p.previous["tasks.md"], status.HandoffNextAction) {
		return fmt.Errorf("openspec amend: persisted FSM state %q disagrees with mission_status %q", status.State, p.status)
	}
	p.disposition = amendmentDisposition(status)
	p.reason = p.input.Reason
	if p.reason == "" {
		p.reason = p.disposition
	}
	return nil
}

func validatePersistedMission(status domain.MissionEngineStatus, missionID string) error {
	switch {
	case status.MissionID == "":
		return fmt.Errorf("openspec amend: persisted FSM state is required")
	case status.MissionID != missionID:
		return fmt.Errorf("openspec amend: persisted FSM mission_id %q does not match %q", status.MissionID, missionID)
	default:
		return nil
	}
}

func validateGateEvidence(label, status string) error {
	if label == "rejected" {
		return fmt.Errorf("openspec amend: the mission's gate outcome is rejected; a rejected analysis is not amended")
	}
	if label != "" && !gateLabelMatchesStatus(label, status) {
		return fmt.Errorf("openspec amend: gate telemetry label %q disagrees with mission_status %q", label, status)
	}
	return nil
}

func amendmentDisposition(status domain.MissionEngineStatus) string {
	if status.State == domain.StateRefinement && status.HandoffNextAction == "reapprove_gate" {
		return "same_mission_repair"
	}
	return "same_mission_amendment"
}

func canonicalPackageDigest(files map[string][]byte) string {
	hash := sha256.New()
	for _, name := range canonicalFiles {
		content := files[name]
		_, _ = fmt.Fprintf(hash, "%s\x00%d\x00", name, len(content)) //nolint:errcheck // hash writes never fail
		_, _ = hash.Write(content)                                   //nolint:errcheck // hash writes never fail
	}
	return fmt.Sprintf("sha256:%x", hash.Sum(nil))
}

func gateLabelMatchesStatus(label, status string) bool {
	switch status {
	case "gate_analysis_accepted":
		return label == "accepted"
	case "gate_revision_requested":
		return label == "revision_requested"
	default:
		return false
	}
}

func frontmatterMatchesState(status string, state domain.MissionState, tasks []byte, nextAction string) bool {
	switch status {
	case "archivist_done", "gate_revision_requested":
		return state == domain.StateRefinement
	case "gate_pending":
		return state == domain.StateApprovalGate
	case "gate_analysis_accepted":
		return acceptedFrontmatterMatchesState(state, tasks, nextAction)
	default:
		return false
	}
}

func acceptedFrontmatterMatchesState(state domain.MissionState, tasks []byte, nextAction string) bool {
	if !documentationTargetMarker.Match(tasks) {
		return state == domain.StateDoneAnalysis
	}
	return state == domain.StateHandoffChallenge || (state == domain.StateRefinement && nextAction == "reapprove_gate")
}
