package main

import (
	"os"
	"path/filepath"
	"runtime"
	"testing"

	"github.com/SergioLacerda/strategist-skill/internal/testutil"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func setMetricsScoutRoot(t *testing.T, value string) {
	t.Helper()
	require.NoError(t, metricsScoutCmd.Flags().Set(flagRoot, value))
}

func TestMetricsScoutCmd_EmptyMemoryPrintsZeroRatesAndExitsCleanly(t *testing.T) {
	dir := t.TempDir()
	testutil.MinimalRoot(t, dir)
	setMetricsScoutRoot(t, dir)
	t.Cleanup(func() { setMetricsScoutRoot(t, "") })

	// Acceptance check: no route-decisions.jsonl/outcomes.jsonl exist yet.
	_, err := os.Stat(filepath.Join(dir, "memory", "route-decisions.jsonl"))
	require.True(t, os.IsNotExist(err))

	out := captureStdout(t, func() {
		require.NoError(t, metricsScoutCmd.RunE(metricsScoutCmd, nil))
	})

	assert.Contains(t, out, "fallback_rate: 0.00")
	assert.Contains(t, out, "unnecessary_pipeline_rate: 0.00")
	assert.Contains(t, out, "sample_size: 0")
	assert.Contains(t, out, "full_pipeline_sample_size: 0")
}

func TestMetricsScoutCmd_ReadsRecordedHistory(t *testing.T) {
	dir := t.TempDir()
	testutil.MinimalRoot(t, dir)
	setMetricsScoutRoot(t, dir)
	t.Cleanup(func() { setMetricsScoutRoot(t, "") })

	memDir := filepath.Join(dir, "memory")
	require.NoError(t, os.MkdirAll(memDir, 0o755))
	decisionLine := `{"mission_id":"m-1","request_category":"general","selected_route":"full_pipeline","route_reason":"r","route_confidence":0.8,"evidence_state":"explicit","fallback_route":"full_pipeline","timestamp":"2026-08-03T00:00:00Z"}
`
	require.NoError(t, os.WriteFile(filepath.Join(memDir, "route-decisions.jsonl"), []byte(decisionLine), 0o644))

	out := captureStdout(t, func() {
		require.NoError(t, metricsScoutCmd.RunE(metricsScoutCmd, nil))
	})

	assert.Contains(t, out, "fallback_rate: 1.00")
	assert.Contains(t, out, "sample_size: 1")
}

func TestMetricsScoutCmd_IsHumanStatusCommand(t *testing.T) {
	assert.True(t, isHumanStatusCommand(metricsScoutCmd))
}

func TestRunMetricsScout_ReadRouteDecisionsErrorPropagates(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("ENOTDIR read-error semantics differ on Windows")
	}
	dir := t.TempDir()
	blocker := filepath.Join(dir, "blocker")
	require.NoError(t, os.WriteFile(blocker, []byte("x"), 0o644))
	setMetricsScoutRoot(t, blocker)
	t.Cleanup(func() { setMetricsScoutRoot(t, "") })

	err := runMetricsScout(metricsScoutCmd, metricsScoutOptions{})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "metrics scout")
}
