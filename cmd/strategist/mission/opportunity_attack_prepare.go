package mission

import (
	"fmt"
	"path/filepath"

	"github.com/SergioLacerda/strategist-skill/internal/domain"
	"github.com/SergioLacerda/strategist-skill/internal/feats/opportunity_attack"
	"github.com/SergioLacerda/strategist-skill/internal/mission/handoff"
)

func ensureOpportunityAttack(deps LifecycleDependencies, root, basePath, missionID string, event domain.MissionEngineEvent) error {
	if event != domain.MissionEventRefinementDone {
		return nil
	}
	refined := filepath.Join(basePath, "refined", missionID)
	digest, err := handoff.PackageDigest(refined)
	if err != nil {
		return fmt.Errorf("mission submit: opportunity attack package digest: %w", err)
	}
	manifest, err := opportunity_attack.Ensure(opportunity_attack.Input{
		RefinedPath: refined, StrategistRoot: root, MissionID: missionID, PackageDigest: digest,
		ChestIndex: loadOpportunityChestIndex,
	})
	if err != nil {
		return fmt.Errorf("mission submit: opportunity attack: %w", err)
	}
	return prepareRunbookOpportunity(deps, root, basePath, missionID, manifest)
}

func prepareRunbookOpportunity(deps LifecycleDependencies, root, basePath, missionID string, manifest opportunity_attack.Manifest) error {
	if deps.PrepareRunbook == nil {
		return nil
	}
	for _, candidate := range manifest.Candidates {
		if candidate.Kind != opportunity_attack.KindRunbook || !candidate.Warranted {
			continue
		}
		if err := deps.PrepareRunbook(root, basePath, missionID, manifest.PackageDigest, candidate.ID); err != nil {
			return fmt.Errorf("mission submit: prepare Runbook opportunity: %w", err)
		}
		break
	}
	return nil
}
