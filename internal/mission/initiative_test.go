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

func TestRoleLifecycleRequiresLevelingBeforeInitiative(t *testing.T) {
	lifecycle, err := NewDefaultRoleLifecycle(t.TempDir(), domain.DefaultRoleRegistry())
	require.NoError(t, err)
	_, err = lifecycle.Enter(InitiativeRoleEntry{MissionID: "missing-level", Role: "scout", RunID: "run"})
	require.ErrorContains(t, err, "LEVELING resolution is required")
}

func TestRoleLifecycleDoesNotExposeRetiredBoundaryHooks(t *testing.T) {
	lifecycle, err := NewDefaultRoleLifecycle(t.TempDir(), domain.DefaultRoleRegistry())
	require.NoError(t, err)
	advice, err := lifecycle.Enter(InitiativeRoleEntry{MissionID: "ranger", Role: "ranger", RunID: "run", Leveling: testLeveling("level-1", "ranger")})
	require.NoError(t, err)
	require.Empty(t, advice.AdviceID)
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

func testLeveling(eventID, role string) *initiative.LevelingResolution {
	return &initiative.LevelingResolution{
		EventID: eventID, Role: role, State: initiative.ObservationKnown,
		Model: "host-model", Provider: "host", Effort: initiative.EffortHigh,
		Capability: "reasoning", LevelSource: "host",
	}
}
