package main

import (
	"fmt"
	"path/filepath"
	"strings"

	missionadapter "github.com/SergioLacerda/strategist-skill/cmd/strategist/mission"
	"github.com/SergioLacerda/strategist-skill/internal/application"
	"github.com/SergioLacerda/strategist-skill/internal/domain"
	missionruntime "github.com/SergioLacerda/strategist-skill/internal/mission"
	"github.com/SergioLacerda/strategist-skill/internal/telemetry"
)

// completeArchivistInvocation publishes one embedded Archivist completion as
// the canonical refined package (ADR-0067). The completion is parsed and
// validated before the invocation journal commits to it, so a malformed
// result leaves the request completable; the publication itself is atomic and
// idempotent, which makes a retry after a crash safe.
func completeArchivistInvocation(store missionruntime.InvocationStore, input missionadapter.InvocationCompleteInput, record domain.MissionInvocationRecord) (domain.MissionInvocationOutcome, error) {
	missionID := record.Request.MissionID
	prepared, err := application.PrepareEmbeddedRefinement(application.EmbeddedRefinementRequest{
		MissionID: missionID, BasePath: input.BasePath, ProviderID: record.Request.Weapon.ID,
		PendingAnalysisPath: filepath.Join(input.BasePath, "pending", missionID+"-analysis.md"),
		RequestID:           record.Request.RequestID, Result: input.Completion.Result,
	}, archivistPorts(input.Root, missionID))
	if err != nil {
		return domain.MissionInvocationOutcome{}, wrapMissionError(err)
	}
	artifactPath, err := workspaceRelative(input.Root, filepath.Join(input.BasePath, "refined", missionID))
	if err != nil {
		return domain.MissionInvocationOutcome{}, err
	}
	digest := contentDigest([]byte(input.Completion.Result))
	if err := store.BeginProcessing(record.Request.RequestID, artifactPath, digest); err != nil {
		return domain.MissionInvocationOutcome{}, fmt.Errorf("begin mission invocation: %w", err)
	}
	if _, err := prepared.Publish(); err != nil {
		return domain.MissionInvocationOutcome{}, wrapMissionError(err)
	}
	if err := store.Complete(record.Request.RequestID, digest); err != nil {
		return domain.MissionInvocationOutcome{}, fmt.Errorf("complete mission invocation: %w", err)
	}
	return domain.MissionInvocationOutcome{
		RequestID: input.RequestID, MissionID: missionID, Status: "normalized", ArtifactPath: artifactPath,
		BindingDigest: record.Request.BindingDigest, SourceDigest: record.Request.SourceDigest,
	}, nil
}

// archivistPorts wires the Archivist confidence claim and the publication
// record to the same durable stores `mission normalize-openspec` uses.
func archivistPorts(root, missionID string) application.NormalizeOpenSpecPorts {
	return application.NormalizeOpenSpecPorts{
		RecordConfidence: func(claim domain.ConfidenceClaim, evidence []domain.Evidence) error {
			producer, err := telemetry.NewConfidenceProducerAdapter(telemetry.ConfidenceHistoryPath(root), telemetry.ConfidenceAgentArchivist, missionID)
			if err != nil {
				return fmt.Errorf("create Archivist confidence producer: %w", err)
			}
			if _, err := producer.RecordClaim(claim, evidence); err != nil {
				return fmt.Errorf("persist Archivist confidence: %w", err)
			}
			return nil
		},
		RecordPublication: func(publication application.NormalizeOpenSpecPublication) error {
			if err := telemetry.AppendRefinedPackagePublication(telemetry.RefinedPackagePublicationHistoryPath(root), telemetry.RefinedPackagePublicationRecord{
				MissionID: publication.MissionID, ProviderChangeID: publication.ProviderChangeID,
				SourceDigest: publication.SourceDigest, PackageDigest: publication.PackageDigest, PublishedAt: publication.PublishedAt,
			}); err != nil {
				return fmt.Errorf("record refined package publication: %w", err)
			}
			return nil
		},
	}
}

// workspaceRelative renders path relative to the workspace root, refusing a
// path that escapes it.
func workspaceRelative(strategistRoot, path string) (string, error) {
	relative, err := filepath.Rel(filepath.Dir(strategistRoot), path)
	if err != nil || filepath.IsAbs(relative) || relative == ".." || strings.HasPrefix(relative, ".."+string(filepath.Separator)) {
		return "", fmt.Errorf("role_invocation_failed: Archivist package path escapes workspace")
	}
	return filepath.ToSlash(relative), nil
}
