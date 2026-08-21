package main

import (
	"os"
	"path/filepath"
	"runtime"
	"testing"

	"github.com/SergioLacerda/strategist-skill/internal/telemetry"
	"github.com/SergioLacerda/strategist-skill/internal/testutil"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func setMetricsHandoffRoot(t *testing.T, value string) {
	t.Helper()
	require.NoError(t, metricsHandoffCmd.Flags().Set(flagRoot, value))
}

func TestMetricsHandoffCmd_EmptyMemoryPrintsZeroRatesAndExitsCleanly(t *testing.T) {
	dir := t.TempDir()
	testutil.MinimalRoot(t, dir)
	setMetricsHandoffRoot(t, dir)
	t.Cleanup(func() { setMetricsHandoffRoot(t, "") })

	// Acceptance check: no .strategist/memory/handoff-challenges.jsonl exists yet.
	_, err := os.Stat(filepath.Join(dir, "memory", "handoff-challenges.jsonl"))
	require.True(t, os.IsNotExist(err))

	out := captureStdout(t, func() {
		require.NoError(t, metricsHandoffCmd.RunE(metricsHandoffCmd, nil))
	})

	assert.Contains(t, out, "handoff_pass_rate: 0.00")
	assert.Contains(t, out, "first_attempt_pass_rate: 0.00")
	assert.Contains(t, out, "critical_constraint_recall: 0.00")
	assert.Contains(t, out, "decision_classification_accuracy: 0.00")
	assert.Contains(t, out, "scope_violation_rate: 0.00")
	assert.Contains(t, out, "handoff_repair_rate: 0.00")
	assert.Contains(t, out, "semantic_handoff_loss.recall: 0.00")
	assert.Contains(t, out, "semantic_handoff_loss.classification: 0.00")
	assert.Contains(t, out, "semantic_handoff_loss.application: 0.00")
	assert.Contains(t, out, "sample_size: 0")
}

func TestMetricsHandoffCmd_ReadsRecordedHistory(t *testing.T) {
	dir := t.TempDir()
	testutil.MinimalRoot(t, dir)
	setMetricsHandoffRoot(t, dir)
	t.Cleanup(func() { setMetricsHandoffRoot(t, "") })

	memDir := filepath.Join(dir, "memory")
	require.NoError(t, os.MkdirAll(memDir, 0o755))
	content := `{"mission_id":"m-1","transition":"archivist_to_sniper","attempt":1,"timestamp":"2026-08-03T00:00:00Z","status":"passed","passed":true}
`
	require.NoError(t, os.WriteFile(filepath.Join(memDir, "handoff-challenges.jsonl"), []byte(content), 0o644))

	out := captureStdout(t, func() {
		require.NoError(t, metricsHandoffCmd.RunE(metricsHandoffCmd, nil))
	})

	assert.Contains(t, out, "handoff_pass_rate: 1.00")
	assert.Contains(t, out, "sample_size: 1")
}

func TestMetricsHandoffCmd_NonexistentRootStillExitsCleanWithZeroRates(t *testing.T) {
	// ReadHandoffChallenges tolerates a missing file (and, transitively, a
	// missing root) the same way it tolerates an installed-but-empty
	// workspace — the acceptance check only requires zero-rate success, not
	// that this command double as a "workspace is installed" check.
	setMetricsHandoffRoot(t, filepath.Join(t.TempDir(), "nonexistent"))
	t.Cleanup(func() { setMetricsHandoffRoot(t, "") })

	out := captureStdout(t, func() {
		require.NoError(t, metricsHandoffCmd.RunE(metricsHandoffCmd, nil))
	})
	assert.Contains(t, out, "sample_size: 0")
}

func TestMetricsCmd_IsHumanStatusCommand(t *testing.T) {
	assert.True(t, isHumanStatusCommand(metricsHandoffCmd))
}

func TestPrintHandoffMetrics_WriteError(t *testing.T) {
	err := printHandoffMetrics(errorWriter{}, telemetry.HandoffMetrics{})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "write output")
}

func TestPrintRouteMetrics_WriteError(t *testing.T) {
	err := printRouteMetrics(errorWriter{}, telemetry.RouteMetrics{})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "write output")
}

func TestRunMetricsHandoff_WithMissionRunDoesNotError(t *testing.T) {
	dir := t.TempDir()
	testutil.MinimalRoot(t, dir)
	setMetricsHandoffRoot(t, dir)
	t.Cleanup(func() { setMetricsHandoffRoot(t, "") })
	attachMissionRun(t, metricsHandoffCmd)

	require.NoError(t, runMetricsHandoff(metricsHandoffCmd, metricsHandoffOptions{}))
}

func TestRunMetricsHandoff_ReadErrorPropagates(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("ENOTDIR read-error semantics differ on Windows")
	}
	dir := t.TempDir()
	blocker := filepath.Join(dir, "blocker")
	require.NoError(t, os.WriteFile(blocker, []byte("x"), 0o644))
	// blocker is a file, so telemetry.ReadHandoffChallenges' os.Open on
	// blocker/memory/handoff-challenges.jsonl fails with ENOTDIR — a real
	// error distinct from os.ErrNotExist, which resolveMetricsActionRoot's
	// explicit-root resolution never triggers on its own (filepath.Abs
	// doesn't validate existence).
	setMetricsHandoffRoot(t, blocker)
	t.Cleanup(func() { setMetricsHandoffRoot(t, "") })

	err := runMetricsHandoff(metricsHandoffCmd, metricsHandoffOptions{})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "metrics handoff")
}

func TestRunMetricsScout_WithMissionRunDoesNotError(t *testing.T) {
	dir := t.TempDir()
	testutil.MinimalRoot(t, dir)
	setMetricsScoutRoot(t, dir)
	t.Cleanup(func() { setMetricsScoutRoot(t, "") })
	attachMissionRun(t, metricsScoutCmd)

	require.NoError(t, runMetricsScout(metricsScoutCmd, metricsScoutOptions{}))
}
