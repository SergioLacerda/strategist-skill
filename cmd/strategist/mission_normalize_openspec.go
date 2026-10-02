package main

import (
	"fmt"
	"path/filepath"

	missionadapter "github.com/SergioLacerda/strategist-skill/cmd/strategist/mission"
	"github.com/SergioLacerda/strategist-skill/internal/cliutil"
	"github.com/SergioLacerda/strategist-skill/internal/domain"
	"github.com/SergioLacerda/strategist-skill/internal/refinement"
	"github.com/SergioLacerda/strategist-skill/internal/telemetry"
)

func resolveNormalizePaths(opts missionadapter.NormalizeOptions) (string, string, string, error) {
	strategistRoot, basePath, err := cliutil.ResolveActiveBasePath(opts.Root)
	if err != nil {
		return "", "", "", fmt.Errorf("resolve active base path: %w", err)
	}
	projectRoot := filepath.Dir(strategistRoot)
	return basePath, resolvePath(opts.RuntimeRoot, filepath.Join(strategistRoot, "openspec"), projectRoot),
		resolvePath(opts.Pending, filepath.Join(basePath, "pending", opts.MissionID+"-analysis.md"), projectRoot), nil
}

func recordNormalizeConfidence(opts missionadapter.NormalizeOptions, claim domain.ConfidenceClaim, evidence []domain.Evidence) error {
	strategistRoot, _, err := cliutil.ResolveActiveBasePath(opts.Root)
	if err != nil {
		return fmt.Errorf("resolve active base path for confidence: %w", err)
	}
	producer, err := telemetry.NewConfidenceProducerAdapter(
		telemetry.ConfidenceHistoryPath(strategistRoot), telemetry.ConfidenceAgentArchivist, opts.MissionID,
	)
	if err != nil {
		return fmt.Errorf("create Archivist confidence producer: %w", err)
	}
	if _, err := producer.RecordClaim(claim, evidence); err != nil {
		return fmt.Errorf("persist Archivist confidence: %w", err)
	}
	return nil
}

func recordNormalizePublication(opts missionadapter.NormalizeOptions, publication refinement.PackagePublication) error {
	strategistRoot, _, err := cliutil.ResolveActiveBasePath(opts.Root)
	if err != nil {
		return fmt.Errorf("resolve active base path for publication: %w", err)
	}
	if err := telemetry.AppendRefinedPackagePublication(telemetry.RefinedPackagePublicationHistoryPath(strategistRoot), telemetry.RefinedPackagePublicationRecord{
		MissionID: publication.MissionID, ProviderChangeID: publication.ProviderChangeID,
		SourceDigest: publication.SourceDigest, PackageDigest: publication.PackageDigest, PublishedAt: publication.PublishedAt,
	}); err != nil {
		return fmt.Errorf("record refined package publication: %w", err)
	}
	return nil
}

func resolvePath(value, fallback, projectRoot string) string {
	if value == "" {
		return fallback
	}
	if filepath.IsAbs(value) {
		return value
	}
	return filepath.Join(projectRoot, value)
}

// resolveNormalizeGateLabel reads the mission's gate outcome label, which an
// amendment consults: a rejected analysis is not amended.
func resolveNormalizeGateLabel(opts missionadapter.NormalizeOptions) (string, error) {
	strategistRoot, _, err := cliutil.ResolveActiveBasePath(opts.Root)
	if err != nil {
		return "", fmt.Errorf("resolve active base path: %w", err)
	}
	label, err := telemetry.GateOutcomeFor(strategistRoot, opts.MissionID)
	if err != nil {
		return "", fmt.Errorf("gate outcome: %w", err)
	}
	return label, nil
}

func loadNormalizeMission(opts missionadapter.NormalizeOptions) (domain.MissionEngineStatus, error) {
	strategistRoot, _, err := cliutil.ResolveActiveBasePath(opts.Root)
	if err != nil {
		return domain.MissionEngineStatus{}, fmt.Errorf("resolve active base path: %w", err)
	}
	_, status, err := loadMission(strategistRoot, opts.MissionID)
	if err != nil {
		return domain.MissionEngineStatus{}, err
	}
	return status, nil
}
