package main

import (
	"context"
	"fmt"
	"path/filepath"

	"github.com/SergioLacerda/strategist-skill/internal/feats/opportunity_attack"
	preparerunbook "github.com/SergioLacerda/strategist-skill/internal/tools/prepare_runbook"
)

// prepareRunbookOpportunity is the composition boundary for OA's optional
// confidence reading. The Tool remains deterministic and the manifest remains
// the only persisted output; canonical Runbook materialization is a later Gate
// decision.
func prepareRunbookOpportunity(root, basePath, missionID, digest, candidateID string) error {
	canonical, err := runbookCanonicalPath(root)
	if err != nil {
		canonical = "docs/runbooks"
	}
	runbookRoot := canonical
	if !filepath.IsAbs(runbookRoot) {
		runbookRoot = filepath.Join(filepath.Dir(root), runbookRoot)
	}
	consumer, consumerErr := runbookConsumer(root)
	if consumerErr != nil {
		consumer = nil
	}
	var evaluate preparerunbook.Evaluator
	if consumer != nil {
		evaluate = consumer.Evaluate
	}
	result, err := preparerunbook.Prepare(context.Background(), preparerunbook.Input{
		MissionID: missionID, CandidateID: candidateID, PackageDigest: digest,
		RefinedPath: filepath.Join(basePath, "refined", missionID), RunbookRoot: runbookRoot,
		Evaluate: evaluate,
	})
	if err != nil {
		return fmt.Errorf("prepare candidate: %w", err)
	}
	if _, err := opportunity_attack.AttachRunbookPreparation(root, missionID, digest, result); err != nil {
		return fmt.Errorf("attach candidate: %w", err)
	}
	return nil
}
