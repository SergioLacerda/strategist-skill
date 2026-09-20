package main

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/SergioLacerda/strategist-skill/internal/testutil"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func setMetricsConfidenceRoot(t *testing.T, value string) {
	t.Helper()
	require.NoError(t, metricsConfidenceCmd.Flags().Set(flagRoot, value))
}

func TestMetricsConfidenceCmd_EmptyMemoryReportsNoSample(t *testing.T) {
	dir := t.TempDir()
	testutil.MinimalRoot(t, dir)
	setMetricsConfidenceRoot(t, dir)
	t.Cleanup(func() { setMetricsConfidenceRoot(t, "") })

	out := captureStdout(t, func() {
		require.NoError(t, metricsConfidenceCmd.RunE(metricsConfidenceCmd, nil))
	})
	assert.Contains(t, out, "policy_version: v1")
	assert.Contains(t, out, "sample_size: 0")
	assert.Contains(t, out, "calibration_status: no_sample")
	assert.NotContains(t, out, "calibrated: 0%")
}

func TestMetricsConfidenceCmd_ReadsSeededGroundTruthHistory(t *testing.T) {
	dir := t.TempDir()
	testutil.MinimalRoot(t, dir)
	setMetricsConfidenceRoot(t, dir)
	t.Cleanup(func() { setMetricsConfidenceRoot(t, "") })

	path := filepath.Join(dir, "memory", "confidence-records.jsonl")
	records := "" +
		`{"claim_id":"q-1","agent":"ranger","claim_kind":"question","confidence_level":"low","confidence_percent":40,"question_preserved":true}` + "\n" +
		`{"claim_id":"a-1","agent":"archivist","claim_kind":"assertion","confidence_level":"high","confidence_percent":90,"evidence_provided":true,"ground_truth_ref":"review-1","ground_truth_kind":"user_revision","corrected":true}` + "\n"
	require.NoError(t, os.MkdirAll(filepath.Dir(path), 0o755))
	require.NoError(t, os.WriteFile(path, []byte(records), 0o644))

	out := captureStdout(t, func() {
		require.NoError(t, metricsConfidenceCmd.RunE(metricsConfidenceCmd, nil))
	})
	assert.Contains(t, out, "confidence_distribution.low: 1")
	assert.Contains(t, out, "confidence_distribution.high: 1")
	assert.Contains(t, out, "question_preservation_rate: 1.00")
	assert.Contains(t, out, "ground_truth_sample_size: 1")
	assert.Contains(t, out, "calibration_status: observed")
	assert.Contains(t, out, "agent.ranger.sample_size: 1")
	assert.Contains(t, out, "agent.archivist.sample_size: 1")
	assert.Contains(t, out, "review_required: true")
}

func TestMetricsConfidenceCmdIsHumanStatusCommand(t *testing.T) {
	assert.True(t, isHumanStatusCommand(metricsConfidenceCmd))
}
