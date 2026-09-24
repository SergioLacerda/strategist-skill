package mission

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"testing"

	"github.com/SergioLacerda/strategist-skill/internal/domain"
	"github.com/SergioLacerda/strategist-skill/internal/embed"
	"github.com/SergioLacerda/strategist-skill/internal/initiative"
	"github.com/SergioLacerda/strategist-skill/internal/telemetry"
	"github.com/stretchr/testify/require"
)

func TestInitiativeRuntimeRoleEntryResultAndHandoffAreCorrelated(t *testing.T) {
	root := t.TempDir()
	runtime, err := NewDefaultInitiativeRuntime(root)
	require.NoError(t, err)

	advice, err := runtime.EnterRole(InitiativeRoleEntry{
		MissionID: "m-runtime", Role: "ranger", RunID: "run-1",
		Observed: initiative.Observation{State: initiative.ObservationKnown, Effort: initiative.EffortHigh, Model: "host-model", Provider: "host", LevelSource: "host"},
	})
	require.NoError(t, err)

	handoff, err := runtime.CompleteRole(advice, initiative.Result{
		AdviceID: advice.AdviceID, MissionID: advice.MissionID, Role: advice.Role, RunID: advice.RunID,
		GateIndependent: true,
		Checks:          []initiative.ObligationCheck{{ID: "inspect_evidence", Status: initiative.CheckSatisfied, EvidenceRefs: []initiative.EvidenceRef{{ID: "e-1", Class: "explicit"}}}},
		EvidenceRefs:    []initiative.EvidenceRef{{ID: "e-1", Class: "explicit"}},
		Outcomes:        []initiative.OutcomeCorrelation{{ID: "out-1", Status: "observed"}},
	}, "archivist")
	require.NoError(t, err)
	require.NoError(t, handoff.Validate())
	require.False(t, handoff.Assessment.Challenge)
	require.NoError(t, runtime.ConsumeHandoff(handoff))

	records, err := initiative.ReadRecords(initiative.LedgerPath(root))
	require.NoError(t, err)
	require.Len(t, records, 2)
	require.Equal(t, advice.AdviceID, records[1].AdviceID)
	_, err = os.Stat(filepath.Join(root, "memory", "initiative-events.jsonl"))
	require.NoError(t, err)
	events, err := os.ReadFile(filepath.Join(root, "memory", "initiative-events.jsonl"))
	require.NoError(t, err)
	require.Contains(t, string(events), advice.AdviceID)
	require.Contains(t, string(events), "strategist.initiative.recommended_effort")
	require.NotContains(t, string(events), `"strategist.effort"`)
	require.Contains(t, string(events), "strategist.initiative.result")
	_, err = os.Stat(filepath.Join(root, "memory", "role-levels.jsonl"))
	require.ErrorIs(t, err, os.ErrNotExist, "INITIATIVE must not write the LEVELING ledger")
}

func TestInitiativeRuntimePreservesUnknownObservation(t *testing.T) {
	root := t.TempDir()
	runtime, err := NewDefaultInitiativeRuntime(root)
	require.NoError(t, err)
	advice, err := runtime.EnterRole(InitiativeRoleEntry{
		MissionID: "m-unknown", Role: "sniper", RunID: "run-1",
		Observed: initiative.Observation{State: initiative.ObservationUnavailable},
	})
	require.NoError(t, err)
	require.Equal(t, initiative.AlignmentUnavailable, advice.Alignment)
	require.Empty(t, advice.Observed.Model)
	require.Empty(t, advice.Observed.Effort)
}

func TestDefaultInitiativeRuntimeUsesCanonicalPolicy(t *testing.T) {
	raw, err := (embed.Extractor{}).ReadFile("initiative.yaml")
	require.NoError(t, err)
	parsed, err := initiative.Parse(raw)
	require.NoError(t, err)
	require.Equal(t, initiative.DefaultPolicy(), parsed)

	runtime, err := NewDefaultInitiativeRuntime(t.TempDir())
	require.NoError(t, err)
	advice, err := runtime.EnterRole(InitiativeRoleEntry{MissionID: "canonical", Role: "ranger", RunID: "run"})
	require.NoError(t, err)
	require.Equal(t, parsed.Digest(), advice.PolicyDigest)
}

func TestDefaultInitiativeRuntimeRejectsPolicyMirrorDrift(t *testing.T) {
	root := t.TempDir()
	require.NoError(t, os.WriteFile(filepath.Join(root, "initiative.yaml"), []byte("version: drift\n"), 0o600))
	_, err := NewDefaultInitiativeRuntime(root)
	require.ErrorContains(t, err, "policy mirror drift")
}

func TestRoleLifecycleConsumesDeclaredHooks(t *testing.T) {
	lifecycle, err := NewDefaultRoleLifecycle(t.TempDir(), domain.DefaultRoleRegistry())
	require.NoError(t, err)
	advice, err := lifecycle.Enter(InitiativeRoleEntry{MissionID: "hooks", Role: "ranger", RunID: "run", Leveling: testLeveling("level-1", "ranger")})
	require.NoError(t, err)
	handoff, err := lifecycle.Complete(advice, validMissionResult(advice), "archivist")
	require.NoError(t, err)
	require.NoError(t, lifecycle.Consume(handoff))
}

func TestRoleLifecycleRequiresLevelingBeforeInitiative(t *testing.T) {
	lifecycle, err := NewDefaultRoleLifecycle(t.TempDir(), domain.DefaultRoleRegistry())
	require.NoError(t, err)
	_, err = lifecycle.Enter(InitiativeRoleEntry{MissionID: "missing-level", Role: "ranger", RunID: "run"})
	require.ErrorContains(t, err, "LEVELING resolution is required")
}

func TestRoleLifecycleReevaluationRequiresNewLevelingEvent(t *testing.T) {
	lifecycle, err := NewDefaultRoleLifecycle(t.TempDir(), domain.DefaultRoleRegistry())
	require.NoError(t, err)
	first, err := lifecycle.Enter(InitiativeRoleEntry{MissionID: "reeval-level", Role: "ranger", RunID: "run", Leveling: testLeveling("level-1", "ranger")})
	require.NoError(t, err)
	_, err = lifecycle.Reevaluate(InitiativeRoleEntry{MissionID: "reeval-level", Role: "ranger", RunID: "run", Leveling: testLeveling("level-1", "ranger")}, initiative.TriggerScopeChanged)
	require.ErrorContains(t, err, "new LEVELING event")
	second, err := lifecycle.Reevaluate(InitiativeRoleEntry{MissionID: "reeval-level", Role: "ranger", RunID: "run", Leveling: testLeveling("level-2", "ranger")}, initiative.TriggerScopeChanged)
	require.NoError(t, err)
	require.Equal(t, first.AdviceID, second.Supersedes)
}

func TestRoleLifecycleKeepsRolesWithoutHooksAsLegacyNoOp(t *testing.T) {
	registry, err := domain.NewRoleRegistry([]domain.Role{{ID: "legacy", Phase: 1}})
	require.NoError(t, err)
	lifecycle, err := NewDefaultRoleLifecycle(t.TempDir(), registry)
	require.NoError(t, err)
	advice, err := lifecycle.Enter(InitiativeRoleEntry{MissionID: "legacy", Role: "legacy", RunID: "run"})
	require.NoError(t, err)
	require.Empty(t, advice.AdviceID)
}

func TestInitiativeHandoffRejectsInvalidTransitionAndTamperedReasons(t *testing.T) {
	runtime, err := NewDefaultInitiativeRuntime(t.TempDir())
	require.NoError(t, err)
	advice, err := runtime.EnterRole(InitiativeRoleEntry{MissionID: "integrity", Role: "ranger", RunID: "run"})
	require.NoError(t, err)
	handoff, err := runtime.CompleteRole(advice, initiative.Result{
		AdviceID: advice.AdviceID, MissionID: advice.MissionID, Role: advice.Role, RunID: advice.RunID,
		GateIndependent: true,
		Checks:          []initiative.ObligationCheck{{ID: "inspect_evidence", Status: initiative.CheckPartial}},
		EvidenceRefs:    []initiative.EvidenceRef{{ID: "e-1", Class: "explicit"}},
	}, "archivist")
	require.NoError(t, err)
	handoff.Assessment.Reasons = []string{"tampered"}
	require.ErrorContains(t, handoff.Validate(), "assessment does not match")
	_, err = runtime.CompleteRole(advice, validMissionResult(advice), "sniper")
	require.ErrorContains(t, err, "transition")
}

func TestInitiativeRuntimeSurfacesEventSinkFailure(t *testing.T) {
	runtime, err := NewInitiativeRuntimeWithSink(t.TempDir(), initiative.DefaultPolicy(), failingInitiativeSink{})
	require.NoError(t, err)
	_, err = runtime.EnterRole(InitiativeRoleEntry{MissionID: "sink", Role: "ranger", RunID: "run"})
	require.ErrorIs(t, err, errInitiativeSink)
}

var errInitiativeSink = errors.New("event sink unavailable")

type failingInitiativeSink struct{}

func (failingInitiativeSink) Emit(context.Context, telemetry.Event) error { return errInitiativeSink }

func validMissionResult(advice initiative.Advice) initiative.Result {
	return initiative.Result{
		AdviceID: advice.AdviceID, MissionID: advice.MissionID, Role: advice.Role, RunID: advice.RunID,
		GateIndependent: true,
		Checks:          []initiative.ObligationCheck{{ID: "inspect_evidence", Status: initiative.CheckSatisfied, EvidenceRefs: []initiative.EvidenceRef{{ID: "e-1", Class: "explicit"}}}},
		EvidenceRefs:    []initiative.EvidenceRef{{ID: "e-1", Class: "explicit"}},
	}
}

func testLeveling(eventID, role string) *initiative.LevelingResolution {
	return &initiative.LevelingResolution{
		EventID: eventID, Role: role, State: initiative.ObservationKnown,
		Model: "host-model", Provider: "host", Effort: initiative.EffortHigh,
		Capability: "reasoning", LevelSource: "host",
	}
}
