package main

import (
	"bytes"
	"path/filepath"
	"testing"

	"github.com/SergioLacerda/strategist-skill/internal/telemetry"
	"github.com/SergioLacerda/strategist-skill/internal/testutil"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func labelOpts(root string) metricsLabelOptions {
	return metricsLabelOptions{
		Root: root, Mission: "m-1", Subject: telemetry.GroundTruthSubjectRoute,
		Label: telemetry.RouteLabelReversed, Kind: "user_revision", Ref: "gate-review-1",
	}
}

func TestMetricsLabelCmd_RecordsThenReportsReplay(t *testing.T) {
	dir := t.TempDir()
	testutil.MinimalRoot(t, dir)
	var out bytes.Buffer
	metricsLabelCmd.SetOut(&out)
	t.Cleanup(func() { metricsLabelCmd.SetOut(nil) })

	require.NoError(t, runMetricsLabel(metricsLabelCmd, labelOpts(dir)))
	assert.Contains(t, out.String(), "label recorded: mission=m-1")
	out.Reset()
	require.NoError(t, runMetricsLabel(metricsLabelCmd, labelOpts(dir)))
	assert.Contains(t, out.String(), "already recorded")

	labels, err := telemetry.ReadGroundTruthLabels(telemetry.GroundTruthLabelHistoryPath(dir), "")
	require.NoError(t, err)
	require.Len(t, labels, 1)
	assert.Equal(t, "gate-review-1", labels[0].Ref)
	assert.FileExists(t, filepath.Join(dir, "memory", "ground-truth-labels.jsonl"))
}

func TestMetricsLabelCmd_RejectsInvalidAndFeedsScoutMetrics(t *testing.T) {
	dir := t.TempDir()
	testutil.MinimalRoot(t, dir)
	var out bytes.Buffer
	metricsLabelCmd.SetOut(&out)
	t.Cleanup(func() { metricsLabelCmd.SetOut(nil) })

	bad := labelOpts(dir)
	bad.Label = telemetry.HandoffApplicationLabelApplied
	require.ErrorContains(t, runMetricsLabel(metricsLabelCmd, bad), "not allowed for subject")

	bad = labelOpts(dir)
	bad.Ref = ""
	require.ErrorContains(t, runMetricsLabel(metricsLabelCmd, bad), "ground_truth_ref")

	labels, err := telemetry.ReadGroundTruthLabels(telemetry.GroundTruthLabelHistoryPath(dir), "")
	require.NoError(t, err)
	assert.Empty(t, labels)
}

func TestMetricsLabelCmd_RejectsGateOutcomeSource(t *testing.T) {
	dir := t.TempDir()
	testutil.MinimalRoot(t, dir)
	gate := labelOpts(dir)
	gate.Subject = telemetry.GroundTruthSubjectGateOutcome
	gate.Label = telemetry.GateOutcomeAccepted
	gate.Ref = "approval_gate:event-1"
	if err := runMetricsLabel(metricsLabelCmd, gate); err == nil {
		t.Fatal("gate outcomes must be recorded through the human gate command")
	}
	_, err := telemetry.ReadGroundTruthLabels(telemetry.GroundTruthLabelHistoryPath(dir), "")
	require.NoError(t, err)
}

func TestMetricsLabelCmd_RootAndWriteErrors(t *testing.T) {

	dir := t.TempDir()
	testutil.MinimalRoot(t, dir)
	metricsLabelCmd.SetOut(errorWriter{})
	t.Cleanup(func() { metricsLabelCmd.SetOut(nil) })
	require.ErrorContains(t, runMetricsLabel(metricsLabelCmd, labelOpts(dir)), "write output")
	assert.True(t, isHumanStatusCommand(metricsLabelCmd))
}
