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

func setMetricsFallbackRoot(t *testing.T, value string) {
	t.Helper()
	require.NoError(t, metricsFallbackCmd.Flags().Set(flagRoot, value))
}

func TestMetricsFallbackCmd_EmptyMemoryPrintsZeroRatesAndExitsCleanly(t *testing.T) {
	dir := t.TempDir()
	testutil.MinimalRoot(t, dir)
	setMetricsFallbackRoot(t, dir)
	t.Cleanup(func() { setMetricsFallbackRoot(t, "") })

	_, err := os.Stat(filepath.Join(dir, "memory", "fallback-decisions.jsonl"))
	require.True(t, os.IsNotExist(err))

	out := captureStdout(t, func() {
		require.NoError(t, metricsFallbackCmd.RunE(metricsFallbackCmd, nil))
	})

	assert.Contains(t, out, "auto_native_rate: 0.00")
	assert.Contains(t, out, "ask_confirmed_rate: 0.00")
	assert.Contains(t, out, "sample_size: 0")
}

func TestMetricsFallbackCmd_ReadsRecordedHistory(t *testing.T) {
	dir := t.TempDir()
	testutil.MinimalRoot(t, dir)
	setMetricsFallbackRoot(t, dir)
	t.Cleanup(func() { setMetricsFallbackRoot(t, "") })

	memDir := filepath.Join(dir, "memory")
	require.NoError(t, os.MkdirAll(memDir, 0o755))
	decisionLine := `{"mission_id":"m-1","slot":"execution","phase":"execution","policy":"native","outcome":"auto_native","configured_provider":"openspec-explore","effective_provider":"sniper","reason":"role_invocation_failed","degraded":true,"timestamp":"2026-08-20T00:00:00Z"}
`
	require.NoError(t, os.WriteFile(filepath.Join(memDir, "fallback-decisions.jsonl"), []byte(decisionLine), 0o644))

	out := captureStdout(t, func() {
		require.NoError(t, metricsFallbackCmd.RunE(metricsFallbackCmd, nil))
	})

	assert.Contains(t, out, "auto_native_rate: 1.00")
	assert.Contains(t, out, "sample_size: 1")
}

func TestMetricsFallbackCmd_IsHumanStatusCommand(t *testing.T) {
	assert.True(t, isHumanStatusCommand(metricsFallbackCmd))
}

func TestRunMetricsFallback_ReadFallbackDecisionsErrorPropagates(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("ENOTDIR read-error semantics differ on Windows")
	}
	dir := t.TempDir()
	blocker := filepath.Join(dir, "blocker")
	require.NoError(t, os.WriteFile(blocker, []byte("x"), 0o644))
	setMetricsFallbackRoot(t, blocker)
	t.Cleanup(func() { setMetricsFallbackRoot(t, "") })

	err := runMetricsFallback(metricsFallbackCmd, metricsFallbackOptions{})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "metrics fallback")
}
