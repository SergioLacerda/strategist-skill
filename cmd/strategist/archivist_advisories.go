package main

import (
	"fmt"

	"github.com/SergioLacerda/strategist-skill/internal/mission/refinement"
	"github.com/SergioLacerda/strategist-skill/internal/telemetry"
)

func recordArchivistAdvisories(root string, input refinement.AdvisoryInput) error {
	if err := telemetry.RecordArchivistAdvisories(telemetry.ConfidenceHistoryPath(root), telemetry.ArchivistAdvisoryInput{
		MissionID: input.MissionID, PackageDigest: input.PackageDigest, Analysis: input.Analysis,
	}); err != nil {
		return fmt.Errorf("persist Archivist advisories: %w", err)
	}
	return nil
}
