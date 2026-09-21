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

func TestEvaluateTreasureChestBoundaryRejectsUnsupportedStage(t *testing.T) {
	result := EvaluateTreasureChestBoundary(TreasureChestBoundary{
		Stage:     BoundaryStage("remote_cache"),
		Ownership: Ownership{Host: "strategist", Skill: "treasure-chest"},
		Evidence:  certifiedBoundaryEvidence(),
	})

	require.Equal(t, BoundaryBlocked, result.Status)
	require.Equal(t, "stage_unsupported", result.Reason)
	require.False(t, result.InvocationReady)
	require.True(t, result.LegacyDataPreserved)
}

func TestEvaluateTreasureChestBoundaryDeclaringExternalStageNeverActivatesIt(t *testing.T) {
	evidence := certifiedBoundaryEvidence()
	evidence.Delegate, evidence.Probe, evidence.Health = StateUnknown, StateUnknown, StateUnknown
	result := EvaluateTreasureChestBoundary(TreasureChestBoundary{
		Stage:     StageExternalAdapter,
		Ownership: Ownership{Host: "strategist", Skill: "treasure-chest"},
		Evidence:  evidence,
	})

	require.Equal(t, BoundaryDegraded, result.Status)
	require.Equal(t, "delegate_unwired", result.Reason)
	require.False(t, result.InvocationReady)

	ready := EvaluateTreasureChestBoundary(TreasureChestBoundary{
		Stage:     StageExternalAdapter,
		Ownership: Ownership{Host: "strategist", Skill: "treasure-chest"},
		Evidence:  certifiedBoundaryEvidence(),
	})
	require.Equal(t, BoundaryReady, ready.Status)
	require.True(t, ready.InvocationReady)
}

// TestEvaluateTreasureChestBoundaryEvidenceMatrix walks every evidence
// dimension through every non-certified state so no dimension can silently
// activate an invocation path.
func TestEvaluateTreasureChestBoundaryEvidenceMatrix(t *testing.T) {
	dimensions := map[string]func(*EvidenceVector, EvidenceState){
		"package":    func(e *EvidenceVector, s EvidenceState) { e.Package = s },
		"identity":   func(e *EvidenceVector, s EvidenceState) { e.Identity = s },
		"provenance": func(e *EvidenceVector, s EvidenceState) { e.Provenance = s },
		"api":        func(e *EvidenceVector, s EvidenceState) { e.API = s },
		"delegate":   func(e *EvidenceVector, s EvidenceState) { e.Delegate = s },
		"probe":      func(e *EvidenceVector, s EvidenceState) { e.Probe = s },
		"health":     func(e *EvidenceVector, s EvidenceState) { e.Health = s },
	}
	unknownReason := map[string]string{
		"package": "package_unverified", "identity": "identity_unverified", "provenance": "provenance_unverified",
		"api": "api_unverified", "delegate": "delegate_unwired", "probe": "probe_unverified", "health": "health_unverified",
	}
	blockedStates := []EvidenceState{StateBlocked, StateUnsupported, StateUnauthorized}
	degradedStates := []EvidenceState{StateFailed, StateStale, StateTimeout, StateMalformed, StateUnavailable, StateTeardownFailed}

	for name, set := range dimensions {
		evaluate := func(state EvidenceState) BoundaryResult {
			evidence := certifiedBoundaryEvidence()
			set(&evidence, state)
			return EvaluateTreasureChestBoundary(TreasureChestBoundary{
				Stage:     StageInRepo,
				Ownership: Ownership{Host: "strategist", Skill: "treasure-chest"},
				Evidence:  evidence,
			})
		}

		unknown := evaluate(StateUnknown)
		wantStatus := BoundaryUnverified
		if name == "delegate" {
			wantStatus = BoundaryDegraded
		}
		require.Equal(t, wantStatus, unknown.Status, name)
		require.Equal(t, unknownReason[name], unknown.Reason, name)
		require.False(t, unknown.InvocationReady, name)

		for _, state := range blockedStates {
			result := evaluate(state)
			require.Equal(t, BoundaryBlocked, result.Status, "%s/%s", name, state)
			require.Equal(t, name+"_"+string(state), result.Reason)
			require.False(t, result.InvocationReady, "%s/%s", name, state)
		}
		for _, state := range degradedStates {
			result := evaluate(state)
			require.Equal(t, BoundaryDegraded, result.Status, "%s/%s", name, state)
			require.Equal(t, name+"_"+string(state), result.Reason)
			require.False(t, result.InvocationReady, "%s/%s", name, state)
		}
		require.True(t, evaluate(StateCertified).InvocationReady, "all-certified must be ready (%s)", name)
	}
}

func TestEvaluateMigrationReadyAndEachIncompleteCause(t *testing.T) {
	complete := MigrationRecord{
		FromStage: StageInRepo, ToStage: StageExternalAdapter,
		SourceIdentity: "sha256:old", TargetIdentity: "sha256:new",
		CommandParity: true, APICompatible: true, ProvenanceVerified: true, MigrationComplete: true,
		RollbackRef: "sha256:old",
	}
	ready := EvaluateMigration(complete)
	require.Equal(t, BoundaryReady, ready.Status)
	require.Equal(t, "migration_ready:in_repo_staging_to_external_adapter", ready.Reason)
	require.True(t, ready.LegacyDataPreserved)

	for name, mutate := range map[string]func(*MigrationRecord){
		"same stage":         func(r *MigrationRecord) { r.ToStage = r.FromStage },
		"missing source":     func(r *MigrationRecord) { r.SourceIdentity = "" },
		"missing target":     func(r *MigrationRecord) { r.TargetIdentity = "" },
		"missing rollback":   func(r *MigrationRecord) { r.RollbackRef = "" },
		"no command parity":  func(r *MigrationRecord) { r.CommandParity = false },
		"api incompatible":   func(r *MigrationRecord) { r.APICompatible = false },
		"provenance missing": func(r *MigrationRecord) { r.ProvenanceVerified = false },
		"migration pending":  func(r *MigrationRecord) { r.MigrationComplete = false },
	} {
		record := complete
		mutate(&record)
		result := EvaluateMigration(record)
		require.Equal(t, BoundaryBlocked, result.Status, name)
		require.Contains(t, result.Reason, "migration_", name)
		require.True(t, result.LegacyDataPreserved, name)
	}
}
