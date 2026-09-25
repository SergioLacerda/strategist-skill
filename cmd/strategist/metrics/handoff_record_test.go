package metrics

import (
	"os"
	"strings"
	"testing"

	"github.com/SergioLacerda/strategist-skill/internal/telemetry"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func runHandoffRecord(t *testing.T, root string, args ...string) (string, error) {
	t.Helper()
	cmd := NewHandoffRecord(testDependencies())
	out := &strings.Builder{}
	cmd.SetOut(out)
	cmd.SetArgs(append([]string{"--root", root}, args...))
	err := cmd.Execute()
	return out.String(), err
}

func handoffFile(t *testing.T, root string) string {
	t.Helper()
	raw, err := os.ReadFile(telemetry.HandoffMetricsPath(root)) //nolint:gosec // test-owned temp path
	require.NoError(t, err)
	return strings.TrimSpace(string(raw))
}

func TestHandoffRecordWritesOnlyWhatWasReported(t *testing.T) {
	root := testRoot(t)
	out, err := runHandoffRecord(t, root, "--mission", "m-1", "--reopens", "1", "--model", "sonnet", "--effort", "low", "--level-source", "host")
	require.NoError(t, err)
	assert.Contains(t, out, "handoff metrics recorded: mission=m-1")
	assert.JSONEq(t, `{"mission_id":"m-1","discovery_tokens":null,"brief_tokens":null,"brief_compression_ratio":null,"refinement_reopens":1,"evidence_coverage_ratio":null,"model":"sonnet","effort":"low","level_source":"host"}`, handoffFile(t, root))
}

// The two ratios have no definition in the contracts, so the command never
// derives them from the token counts.
func TestHandoffRecordNeverDerivesTheRatios(t *testing.T) {
	root := testRoot(t)
	_, err := runHandoffRecord(t, root, "--mission", "m-1", "--discovery-tokens", "1000", "--brief-tokens", "250")
	require.NoError(t, err)
	line := handoffFile(t, root)
	assert.Contains(t, line, `"discovery_tokens":1000`)
	assert.Contains(t, line, `"brief_tokens":250`)
	assert.Contains(t, line, `"brief_compression_ratio":null`)

	_, err = runHandoffRecord(t, root, "--mission", "m-2", "--brief-compression-ratio", "0.25", "--evidence-coverage-ratio", "0.8")
	require.NoError(t, err)
	assert.Contains(t, handoffFile(t, root), `"brief_compression_ratio":0.25,"refinement_reopens":0,"evidence_coverage_ratio":0.8`)
}

func TestHandoffRecordIsIdempotentAndSaysSo(t *testing.T) {
	root := testRoot(t)
	_, err := runHandoffRecord(t, root, "--mission", "m-1")
	require.NoError(t, err)
	out, err := runHandoffRecord(t, root, "--mission", "m-1", "--reopens", "5")
	require.NoError(t, err)
	assert.Contains(t, out, "already recorded for this mission; nothing written")
	assert.Len(t, strings.Split(handoffFile(t, root), "\n"), 1)
}

func TestHandoffRecordRejectsInvalidInputWithoutWriting(t *testing.T) {
	root := testRoot(t)
	_, err := runHandoffRecord(t, root, "--mission", "m-1", "--discovery-tokens", "-5")
	require.ErrorContains(t, err, "discovery_tokens must be >= 0")
	_, err = runHandoffRecord(t, root)
	require.Error(t, err, "--mission is required")
	_, statErr := os.Stat(telemetry.HandoffMetricsPath(root))
	assert.True(t, os.IsNotExist(statErr))
}
