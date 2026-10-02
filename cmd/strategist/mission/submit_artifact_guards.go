package mission

import (
	"fmt"
	"path/filepath"

	"github.com/SergioLacerda/strategist-skill/internal/domain"
	"github.com/SergioLacerda/strategist-skill/internal/handoff"
)

// validateSubmitArtifacts keeps lifecycle events behind the artifact boundary
// they claim to complete. The domain FSM knows nothing about filesystem
// artifacts, so this guard runs at the command boundary under the mission lock.
func validateSubmitArtifacts(basePath, missionID string, event domain.MissionEngineEvent) error {
	switch event {
	case domain.MissionEventDiscoveryDone:
		path := filepath.Join(basePath, "pending", missionID+"-analysis.md")
		if err := handoff.ValidateRangerArtifact(path, missionID); err != nil {
			return fmt.Errorf("discovery artifact validation failed: %w", err)
		}
	case domain.MissionEventRefinementDone, domain.MissionEventGateApproved,
		domain.MissionEventGateApprovedAnalysisOnly:
		refined := filepath.Join(basePath, "refined", missionID)
		if err := handoff.ValidateArchivistPackage(refined, missionID); err != nil {
			return fmt.Errorf("refined package validation failed: %w", err)
		}
	case domain.MissionEventBootstrapDone, domain.MissionEventIntakeDone,
		domain.MissionEventNoTasks, domain.MissionEventGateDenied,
		domain.MissionEventGateTimeout, domain.MissionEventGateRevision,
		domain.MissionEventHandoffSatisfied, domain.MissionEventHandoffFailed,
		domain.MissionEventHandoffNotApplicable, domain.MissionEventHandoffExhausted,
		domain.MissionEventSniperDone, domain.MissionEventRetryOK,
		domain.MissionEventSlotTransient, domain.MissionEventSlotPermanent,
		domain.MissionEventRefinementArtifactInvalid,
		domain.MissionEventADRCriterion, domain.MissionEventADRApproved,
		domain.MissionEventADRDeclined:
		// These events either do not claim an artifact boundary or are
		// validated by their owning handoff/execution guard.
		return nil
	}
	return nil
}
