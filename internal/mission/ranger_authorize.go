package mission

import (
	"fmt"
	"path/filepath"

	"github.com/SergioLacerda/strategist-skill/internal/handoff"
)

// AuthorizeRangerToArchivist verifies and consumes the current outcome before
// Archivist provider invocation. Consumption is exclusive and therefore also
// rejects replayed or cross-process reuse.
func AuthorizeRangerToArchivist(strategistRoot, basePath, missionID string) (handoff.Outcome, error) {
	artifactPath := filepath.Join(basePath, "pending", missionID+"-analysis.md")
	facts, err := handoff.ReadRangerPolicyFacts(artifactPath, missionID)
	if err != nil {
		return handoff.Outcome{}, fmt.Errorf("authorize Ranger-to-Archivist handoff: %w", err)
	}
	policy, err := handoff.RangerToArchivistPolicyForFacts(facts)
	if err != nil {
		return handoff.Outcome{}, fmt.Errorf("authorize Ranger-to-Archivist handoff: %w", err)
	}
	digest, err := handoff.RangerArtifactDigest(artifactPath)
	if err != nil {
		return handoff.Outcome{}, fmt.Errorf("authorize Ranger-to-Archivist handoff: %w", err)
	}
	store := handoff.NewOutcomeStore(strategistRoot)
	outcome, err := store.AuthorizeExecution(handoff.ExecutionCheck{
		MissionID: missionID, Transition: handoff.TransitionRangerToArchivist,
		ArtifactDigest: digest, PolicyID: handoff.PolicyIdentity(policy), SkipAuthorized: !policy.Enabled,
	})
	if err != nil {
		return handoff.Outcome{}, fmt.Errorf("authorize Ranger-to-Archivist handoff: %w", err)
	}
	if err := store.Consume(outcome); err != nil {
		return handoff.Outcome{}, fmt.Errorf("authorize Ranger-to-Archivist handoff: %w", err)
	}
	return outcome, nil
}
