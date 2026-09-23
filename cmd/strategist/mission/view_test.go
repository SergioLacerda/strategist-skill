package mission_test

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"testing"

	mission "github.com/SergioLacerda/strategist-skill/cmd/strategist/mission"
	"github.com/SergioLacerda/strategist-skill/internal/cliutil"
	"github.com/SergioLacerda/strategist-skill/internal/domain"
	"github.com/SergioLacerda/strategist-skill/internal/leveling"
	"github.com/SergioLacerda/strategist-skill/internal/missionview"
	"github.com/SergioLacerda/strategist-skill/internal/telemetry"
	"github.com/SergioLacerda/strategist-skill/internal/testutil"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// setupViewRoot builds a hermetic .strategist/-shaped root at t.TempDir() and,
// when status.MissionID is non-empty, a matching mission-engine state file
// under missions/<id>.json so cmd/strategist/mission.RunView's Load dependency
// succeeds. Mirrors cmd/strategist/mission_report_usage_test.go's hermetic
// pattern (testutil.MinimalRoot + t.TempDir()). See
// .analysis/refined/20260921-evaluate-ux-role-confidence-leveling/tasks.md § 1.1.
func setupViewRoot(t *testing.T, status domain.MissionEngineStatus) (root string) {
	t.Helper()
	dir := t.TempDir()
	root = filepath.Join(dir, ".strategist")
	require.NoError(t, os.MkdirAll(root, 0o755))
	testutil.MinimalRoot(t, root)
	if status.MissionID != "" {
		writeMissionState(t, root, status)
	}
	return root
}

func writeMissionState(t *testing.T, root string, status domain.MissionEngineStatus) {
	t.Helper()
	dir := filepath.Join(root, "missions")
	require.NoError(t, os.MkdirAll(dir, 0o755))
	data, err := json.Marshal(status)
	require.NoError(t, err)
	require.NoError(t, os.WriteFile(filepath.Join(dir, status.MissionID+".json"), data, 0o644))
}

// testLoadMission mirrors cmd/strategist/mission_persistence.go's loadMission,
// which lives in package main and is not importable from this test package.
func testLoadMission(root, id string) (*domain.MissionEngine, domain.MissionEngineStatus, error) {
	data, err := os.ReadFile(filepath.Join(root, "missions", id+".json")) //nolint:gosec // test helper, path built from t.TempDir()
	if err != nil {
		return nil, domain.MissionEngineStatus{}, fmt.Errorf("mission %q not found: %w", id, err)
	}
	var status domain.MissionEngineStatus
	if err := json.Unmarshal(data, &status); err != nil {
		return nil, status, fmt.Errorf("invalid persisted state: %w", err)
	}
	engine, err := domain.RestoreMission(status)
	if err != nil {
		return nil, status, fmt.Errorf("restore mission state: %w", err)
	}
	return engine, status, nil
}

func testFilterLevels(records []leveling.Record, missionID string) []leveling.Record {
	out := records[:0:0]
	for _, r := range records {
		if r.MissionID == missionID {
			out = append(out, r)
		}
	}
	return out
}

func testRequireMissionID(id string) error {
	if id == "" {
		return errors.New("mission view: --mission-id is required")
	}
	return nil
}

func testDeps() mission.ViewDependencies {
	return mission.ViewDependencies{
		RootFlag:         cliutil.FlagRoot,
		Ledger:           "role-levels.jsonl",
		RequireMissionID: testRequireMissionID,
		ResolveBasePath:  cliutil.ResolveActiveBasePath,
		Load:             testLoadMission,
		FilterLevels:     testFilterLevels,
	}
}

func runView(t *testing.T, args ...string) (string, error) {
	t.Helper()
	cmd := mission.NewView(testDeps())
	var out bytes.Buffer
	cmd.SetOut(&out)
	cmd.SetArgs(args)
	err := cmd.Execute()
	return out.String(), err
}

// --- § 2.1: CLI coverage ---

func TestRunView_HumanOutput(t *testing.T) {
	root := setupViewRoot(t, domain.MissionEngineStatus{MissionID: "m-cli", Phase: domain.PhaseRefinement, State: domain.StateRefinement})
	ledgerPath := filepath.Join(root, "memory", "role-levels.jsonl")
	require.NoError(t, leveling.AppendRecord(ledgerPath, leveling.Record{MissionID: "m-cli", Level: leveling.Level{Role: "ranger", Model: "Sonnet", Effort: "High", Source: "host"}}))

	out, err := runView(t, "--root", root, "--mission-id", "m-cli")
	require.NoError(t, err)
	assert.Contains(t, out, "id: m-cli")
	assert.Contains(t, out, "phase: REFINEMENT")
	assert.Contains(t, out, "ranger: Sonnet-High")
}

func TestRunView_JSONOutput(t *testing.T) {
	root := setupViewRoot(t, domain.MissionEngineStatus{MissionID: "m-json-cli", Phase: domain.PhaseBootstrap, State: domain.StateInit})

	out, err := runView(t, "--root", root, "--mission-id", "m-json-cli", "--json")
	require.NoError(t, err)

	var v missionview.View
	require.NoError(t, json.Unmarshal([]byte(out), &v))
	assert.Equal(t, missionview.SchemaVersion, v.Schema)
	assert.Equal(t, "m-json-cli", v.MissionID)
}

func TestRunView_MissingMissionIDErrors(t *testing.T) {
	root := setupViewRoot(t, domain.MissionEngineStatus{})
	_, err := runView(t, "--root", root)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "--mission-id")
}

func TestRunView_UnknownMissionErrors(t *testing.T) {
	root := setupViewRoot(t, domain.MissionEngineStatus{})
	_, err := runView(t, "--root", root, "--mission-id", "does-not-exist")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "mission view")
}

func TestRunView_CorruptMissionStateErrors(t *testing.T) {
	root := setupViewRoot(t, domain.MissionEngineStatus{})
	require.NoError(t, os.MkdirAll(filepath.Join(root, "missions"), 0o755))
	require.NoError(t, os.WriteFile(filepath.Join(root, "missions", "m-corrupt.json"), []byte("{not valid json"), 0o644))

	_, err := runView(t, "--root", root, "--mission-id", "m-corrupt")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "mission view")
}

// --- § 4: named hermetic scenarios ---

// TestRunView_FirstExecutionShowsExplicitAbsence covers the "first run" case:
// no LEVELING/confidence records exist yet, and every secondary section must
// render an explicit absence state, never a fabricated zero value.
func TestRunView_FirstExecutionShowsExplicitAbsence(t *testing.T) {
	root := setupViewRoot(t, domain.MissionEngineStatus{MissionID: "m-first", Phase: domain.PhaseBootstrap, State: domain.StateInit})

	out, err := runView(t, "--root", root, "--mission-id", "m-first", "--json")
	require.NoError(t, err)

	var v missionview.View
	require.NoError(t, json.Unmarshal([]byte(out), &v))
	require.NotNil(t, v.Confidence.Review)
	assert.Equal(t, "no_sample", v.Confidence.Review.Metrics.CalibrationStatus)
	assert.Equal(t, 0, v.Confidence.Review.Metrics.SampleSize)
	assert.Equal(t, missionview.NotApplicable, v.ApprovalGate.Availability)
	assert.Equal(t, missionview.NoSample, v.Confidence.Availability, "no calibrated sample is reported explicitly, not as available")
	assert.Equal(t, missionview.Unknown, v.Leveling.Availability, "no role-level record for the mission yet: unknown, not available")
	for _, role := range v.Leveling.Roles {
		assert.Nilf(t, role.Effective, "role %s should have no effective record on a first run", role.Role)
	}
}

// roleByID finds one role's LevelingRole entry by role ID, since
// missionview always lists every registry role (present or not).
func roleByID(t *testing.T, roles []missionview.LevelingRole, id string) missionview.LevelingRole {
	t.Helper()
	for _, r := range roles {
		if r.Role == id {
			return r
		}
	}
	t.Fatalf("role %q not found in LEVELING roles %+v", id, roles)
	return missionview.LevelingRole{}
}

// TestRunView_RunFlagScopesConfidenceAndLeveling is the KF-005 regression
// guard: --run must scope both confidence and LEVELING consistently, not just
// LEVELING (the original evaluation's KF-005 finding).
func TestRunView_RunFlagScopesConfidenceAndLeveling(t *testing.T) {
	missionID := "m-run-scope"
	root := setupViewRoot(t, domain.MissionEngineStatus{MissionID: missionID, Phase: domain.PhaseRefinement, State: domain.StateRefinement})

	ledgerPath := filepath.Join(root, "memory", "role-levels.jsonl")
	require.NoError(t, leveling.AppendRecord(ledgerPath, leveling.Record{MissionID: missionID, Run: "run-1", Level: leveling.Level{Role: "ranger", Model: "Old"}}))
	require.NoError(t, leveling.AppendRecord(ledgerPath, leveling.Record{MissionID: missionID, Run: "run-2", Level: leveling.Level{Role: "ranger", Model: "New"}}))

	confidencePath := telemetry.ConfidenceHistoryPath(root)
	claimRun1 := domain.ConfidenceClaim{ID: "c-run-1", Statement: "run 1 claim", ClaimKind: domain.ClaimKindQuestion, ConfidencePercent: 40}
	claimRun2 := domain.ConfidenceClaim{ID: "c-run-2", Statement: "run 2 claim", ClaimKind: domain.ClaimKindQuestion, ConfidencePercent: 45}
	_, err := telemetry.AppendConfidenceObservationForRun(confidencePath, "archivist", missionID, "run-1", "2026-09-22T00:00:00Z", claimRun1, nil)
	require.NoError(t, err)
	_, err = telemetry.AppendConfidenceObservationForRun(confidencePath, "archivist", missionID, "run-2", "2026-09-22T00:01:00Z", claimRun2, nil)
	require.NoError(t, err)

	out, err := runView(t, "--root", root, "--mission-id", missionID, "--run", "run-2", "--json")
	require.NoError(t, err)

	var v missionview.View
	require.NoError(t, json.Unmarshal([]byte(out), &v))
	ranger := roleByID(t, v.Leveling.Roles, "ranger")
	require.NotNil(t, ranger.Effective)
	assert.Equal(t, "New", ranger.Effective.Model)

	require.NotNil(t, v.Confidence.Review)
	assert.Equal(t, 1, v.Confidence.Review.Metrics.SampleSize, "confidence must be scoped to run-2 only, not both runs")
}

// TestRunView_NoConfidenceHistoryShowsNoSample covers the "missing confidence"
// and "no_sample" scenarios named in the original acceptance criteria: with no
// confidence-records.jsonl file at all, ReadConfidenceRecordsWithDiagnostics
// fails open to an empty, non-error result (internal/telemetry/confidence_history_read.go),
// so both named scenarios collapse to the same observable outcome here —
// calibration_status: no_sample, never a fabricated percentage or an
// "unavailable" section for a merely-empty (not malformed/unreadable) history.
func TestRunView_NoConfidenceHistoryShowsNoSample(t *testing.T) {
	root := setupViewRoot(t, domain.MissionEngineStatus{MissionID: "m-no-sample", Phase: domain.PhaseRefinement, State: domain.StateRefinement})

	out, err := runView(t, "--root", root, "--mission-id", "m-no-sample", "--json")
	require.NoError(t, err)

	var v missionview.View
	require.NoError(t, json.Unmarshal([]byte(out), &v))
	assert.Equal(t, missionview.NoSample, v.Confidence.Availability, "an empty history is a valid review with no sample, not a read failure (unavailable)")
	require.NotNil(t, v.Confidence.Review)
	assert.Equal(t, "no_sample", v.Confidence.Review.Metrics.CalibrationStatus)
}

// TestRunView_LevelingFallbackRendersProvenance is the KF-006 regression
// guard: a LEVELING record recorded with generic provider fallback must
// surface fallback_used/fallback_reason in both JSON and human output, not
// just level_source.
func TestRunView_LevelingFallbackRendersProvenance(t *testing.T) {
	missionID := "m-fallback"
	root := setupViewRoot(t, domain.MissionEngineStatus{MissionID: missionID, Phase: domain.PhaseRefinement, State: domain.StateRefinement})
	ledgerPath := filepath.Join(root, "memory", "role-levels.jsonl")
	require.NoError(t, leveling.AppendRecord(ledgerPath, leveling.Record{
		MissionID: missionID,
		Level: leveling.Level{
			Role: "sniper", Model: "", Effort: "High", Source: "policy",
			Provider: "generic", FallbackUsed: true, FallbackReason: "unrecognized_provider",
		},
	}))

	humanOut, err := runView(t, "--root", root, "--mission-id", missionID)
	require.NoError(t, err)
	assert.Contains(t, humanOut, "fallback_used=true")
	assert.Contains(t, humanOut, "fallback_reason=unrecognized_provider")

	jsonOut, err := runView(t, "--root", root, "--mission-id", missionID, "--json")
	require.NoError(t, err)
	var v missionview.View
	require.NoError(t, json.Unmarshal([]byte(jsonOut), &v))
	sniper := roleByID(t, v.Leveling.Roles, "sniper")
	require.NotNil(t, sniper.Effective)
	assert.True(t, sniper.Effective.FallbackUsed)
	assert.Equal(t, "unrecognized_provider", sniper.Effective.FallbackReason)
}
