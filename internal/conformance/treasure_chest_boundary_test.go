package conformance

import (
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

func TestEvaluateTreasureChestBoundaryFailsClosedWithoutDelegateOrProbe(t *testing.T) {
	result := EvaluateTreasureChestBoundary(TreasureChestBoundary{
		Stage:     StageInRepo,
		Ownership: Ownership{Host: "strategist", Skill: "treasure-chest"},
		Evidence: EvidenceVector{
			Package: StateCertified, Identity: StateCertified, Provenance: StateCertified,
			API: StateCertified, Delegate: StateUnknown, Probe: StateUnknown, Health: StateUnknown,
		},
	})

	require.Equal(t, BoundaryDegraded, result.Status)
	require.Equal(t, "delegate_unwired", result.Reason)
	require.False(t, result.InvocationReady)
}

func TestEvaluateTreasureChestBoundaryRejectsOptimisticExternalReadiness(t *testing.T) {
	result := EvaluateTreasureChestBoundary(TreasureChestBoundary{
		Stage:     StageExternalAdapter,
		Ownership: Ownership{Host: "strategist", Skill: "treasure-chest"},
		Evidence: EvidenceVector{
			Package: StateCertified, Identity: StateCertified, Provenance: StateCertified,
			API: StateCertified, Delegate: StateCertified, Probe: StateUnknown, Health: StateCertified,
		},
	})

	require.Equal(t, BoundaryUnverified, result.Status)
	require.Equal(t, "probe_unverified", result.Reason)
	require.False(t, result.InvocationReady)
}

func TestEvaluateTreasureChestBoundaryRejectsOwnershipConflict(t *testing.T) {
	result := EvaluateTreasureChestBoundary(TreasureChestBoundary{
		Stage:     StageInRepo,
		Ownership: Ownership{Host: "strategist", Skill: "strategist"},
		Evidence:  certifiedBoundaryEvidence(),
	})

	require.Equal(t, BoundaryBlocked, result.Status)
	require.Equal(t, "ownership_conflict", result.Reason)
}

func TestClassifyTreasureChestImportersRequiresReviewedExceptions(t *testing.T) {
	now := time.Date(2026, time.September, 19, 0, 0, 0, 0, time.UTC)
	entries := ClassifyTreasureChestImporters([]string{
		"github.com/SergioLacerda/strategist-skill/cmd/strategist",
		"github.com/SergioLacerda/strategist-skill/internal/eval",
		"github.com/SergioLacerda/strategist-skill/internal/newcaller",
	}, []ImportException{
		{Importer: "github.com/SergioLacerda/strategist-skill/cmd/strategist", Owner: "eval", Reason: "fixture generation", Scope: "eval harvest", ReviewBy: now.AddDate(0, 1, 0)},
		{Importer: "github.com/SergioLacerda/strategist-skill/internal/eval", Owner: "eval", Reason: "deterministic validation", Scope: "eval harness", ReviewBy: now.AddDate(0, 1, 0)},
	}, now)

	require.Equal(t, ImportReviewedException, entries[0].Disposition)
	require.Equal(t, ImportReviewedException, entries[1].Disposition)
	require.Equal(t, ImportViolation, entries[2].Disposition)
	require.Equal(t, "unreviewed_direct_import", entries[2].Reason)
}

func TestClassifyTreasureChestImportersExpiresExceptionAndAcceptsDelegate(t *testing.T) {
	now := time.Date(2026, time.September, 19, 0, 0, 0, 0, time.UTC)
	entries := ClassifyTreasureChestImporters([]string{
		"github.com/SergioLacerda/strategist-skill/internal/jewelcrafter",
		"github.com/SergioLacerda/strategist-skill/cmd/strategist",
	}, []ImportException{
		{Importer: "github.com/SergioLacerda/strategist-skill/internal/jewelcrafter", Mediated: true},
		{Importer: "github.com/SergioLacerda/strategist-skill/cmd/strategist", Owner: "eval", Reason: "fixture generation", Scope: "eval harvest", ReviewBy: now.Add(-time.Second)},
	}, now)

	require.Equal(t, ImportMediated, entries[0].Disposition)
	require.Equal(t, ImportViolation, entries[1].Disposition)
	require.Equal(t, "exception_expired", entries[1].Reason)
}

func TestEvaluateMigrationPreservesLegacyDataOnIncompleteEvidence(t *testing.T) {
	result := EvaluateMigration(MigrationRecord{
		FromStage: StageInRepo, ToStage: StageExternalAdapter,
		SourceIdentity: "sha256:old", TargetIdentity: "sha256:new",
		CommandParity: true, APICompatible: true, ProvenanceVerified: true,
		RollbackRef: "sha256:old",
	})

	require.Equal(t, BoundaryBlocked, result.Status)
	require.Equal(t, "migration_evidence_incomplete", result.Reason)
	require.True(t, result.LegacyDataPreserved)
}

func certifiedBoundaryEvidence() EvidenceVector {
	return EvidenceVector{
		Package: StateCertified, Identity: StateCertified, Provenance: StateCertified,
		API: StateCertified, Delegate: StateCertified, Probe: StateCertified, Health: StateCertified,
	}
}
