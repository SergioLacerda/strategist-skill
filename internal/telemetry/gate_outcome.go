package telemetry

import (
	"fmt"
	"strings"
	"time"

	"github.com/SergioLacerda/strategist-skill/internal/domain"
)

// RecordGateOutcome is the only automatic ground-truth producer. The Approval
// Gate is the immutable point of human action, so a human gate outcome is the
// sole automatic label source. Confidence an agent fills between handoffs is a
// claim and is never written here: the label kind is fixed to user_revision
// and the reference must name the gate event that recorded the human decision.
func RecordGateOutcome(strategistRoot, missionID, outcome, gateRef string) (appended bool, err error) {
	gateRef = strings.TrimSpace(gateRef)
	if gateRef == "" {
		return false, fmt.Errorf("gate outcome: gate event reference is required")
	}
	if !strings.HasPrefix(gateRef, "approval_gate:") {
		gateRef = "approval_gate:" + gateRef
	}
	label := GroundTruthLabel{
		MissionID: missionID,
		Subject:   GroundTruthSubjectGateOutcome,
		Label:     outcome,
		Kind:      domain.GroundTruthUserRevision,
		Ref:       gateRef,
		Timestamp: time.Now().UTC().Format(time.RFC3339),
	}
	appended, err = AppendGroundTruthLabel(GroundTruthLabelHistoryPath(strategistRoot), label)
	if err != nil {
		return false, fmt.Errorf("gate outcome: %w", err)
	}
	return appended, nil
}

// GateOutcomeFor returns the recorded human gate outcome for a mission, or ""
// when the gate has not been answered (or no label exists).
func GateOutcomeFor(strategistRoot, missionID string) (string, error) {
	labels, err := ReadGroundTruthLabels(GroundTruthLabelHistoryPath(strategistRoot), GroundTruthSubjectGateOutcome)
	if err != nil {
		return "", fmt.Errorf("gate outcome: %w", err)
	}
	for _, l := range labels {
		if l.MissionID == missionID {
			return l.Label, nil
		}
	}
	return "", nil
}
