package main

import (
	"bytes"
	"os"
	"path/filepath"
	"testing"

	"github.com/SergioLacerda/strategist-skill/internal/telemetry"
	"github.com/SergioLacerda/strategist-skill/internal/testutil"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const recordClaimYAML = `claim:
  id: C-1
  statement: s
  agent: ranger
  correlation_key: k1
  claim_kind: assertion
  confidence_percent: 90
  evidence_ids: [E-1]
  evidence_classes: [explicit]
evidence:
  - {id: E-1, source_ref: docs/x.md, class: explicit, confidence: high}
`

func recordFixture(t *testing.T) (root string, out *bytes.Buffer) {
	t.Helper()
	root = t.TempDir()
	testutil.MinimalRoot(t, root)
	out = &bytes.Buffer{}
	metricsRecordCmd.SetOut(out)
	metricsGateOutcomeCmd.SetOut(out)
	t.Cleanup(func() { metricsRecordCmd.SetOut(nil); metricsGateOutcomeCmd.SetOut(nil) })
	return root, out
}

func TestMetricsRecord_ClaimMissingAndErrors(t *testing.T) {
	root, out := recordFixture(t)
	claimPath := filepath.Join(t.TempDir(), "claim.yaml")
	require.NoError(t, os.WriteFile(claimPath, []byte(recordClaimYAML), 0o600))
	opts := metricsRecordOptions{Root: root, Mission: "m-1", Agent: "ranger", ClaimFile: claimPath}

	require.NoError(t, runMetricsRecord(metricsRecordCmd, opts))
	assert.Contains(t, out.String(), "coverage_status: reported")
	require.NoError(t, runMetricsRecord(metricsRecordCmd, opts))

	bad := filepath.Join(t.TempDir(), "bad.yaml")
	require.NoError(t, os.WriteFile(bad, []byte("claim:\n  id: C-2\n  claim_kind: assertion\n  confidence_percent: 90\n"), 0o600))
	out.Reset()
	opts.ClaimFile = bad
	require.NoError(t, runMetricsRecord(metricsRecordCmd, opts))
	assert.Contains(t, out.String(), "coverage_status: rejected")

	out.Reset()
	require.NoError(t, runMetricsRecord(metricsRecordCmd, metricsRecordOptions{Root: root, Mission: "m-1", Agent: "scout", Missing: true, CorrelationKey: "k", Reason: "r"}))
	assert.Contains(t, out.String(), "missing-record recorded")

	review, err := telemetry.LoadConfidenceGateReview(root, "m-1")
	require.NoError(t, err)
	assert.Equal(t, 1, review.Metrics.MissingRecords)

	require.ErrorContains(t, runMetricsRecord(metricsRecordCmd, metricsRecordOptions{Root: root, Mission: "m", Agent: "nobody", Missing: true}), "unsupported agent")
	require.ErrorContains(t, runMetricsRecord(metricsRecordCmd, metricsRecordOptions{Root: root, Mission: "m", Agent: "scout", Missing: true}), "requires --correlation-key")
	require.ErrorContains(t, runMetricsRecord(metricsRecordCmd, metricsRecordOptions{Root: root, Mission: "m", Agent: "scout"}), "--claim-file is required")
	require.ErrorContains(t, runMetricsRecord(metricsRecordCmd, metricsRecordOptions{Root: root, Mission: "m", Agent: "scout", ClaimFile: filepath.Join(root, "absent.yaml")}), "read claim file")
	notYAML := filepath.Join(t.TempDir(), "n.yaml")
	require.NoError(t, os.WriteFile(notYAML, []byte("claim: [unterminated"), 0o600))
	require.ErrorContains(t, runMetricsRecord(metricsRecordCmd, metricsRecordOptions{Root: root, Mission: "m", Agent: "scout", ClaimFile: notYAML}), "parse claim file")
}

func TestMetricsGateOutcomeAndConfidenceMissionScope(t *testing.T) {
	root, out := recordFixture(t)
	opts := metricsGateOutcomeOptions{Root: root, Mission: "m-1", Outcome: "accepted", Ref: "gate-1"}
	require.NoError(t, runMetricsGateOutcome(metricsGateOutcomeCmd, opts))
	assert.Contains(t, out.String(), "gate outcome recorded")
	out.Reset()
	require.NoError(t, runMetricsGateOutcome(metricsGateOutcomeCmd, opts))
	assert.Contains(t, out.String(), "already recorded")
	opts.Outcome = "auto"
	require.Error(t, runMetricsGateOutcome(metricsGateOutcomeCmd, opts))

	var w bytes.Buffer
	require.NoError(t, printGateOutcome(&w, root, ""))
	assert.Empty(t, w.String())
	require.NoError(t, printGateOutcome(&w, root, "m-1"))
	require.NoError(t, printGateOutcome(&w, root, "m-2"))
	assert.Contains(t, w.String(), "gate_outcome: accepted")
	assert.Contains(t, w.String(), "gate_outcome: none")
	require.Error(t, printGateOutcome(errorWriter{}, root, "m-1"))
	assert.True(t, isHumanStatusCommand(metricsRecordCmd) && isHumanStatusCommand(metricsGateOutcomeCmd))
}

func TestMetricsRolloutCheck(t *testing.T) {
	var out bytes.Buffer
	metricsRolloutCmd.SetOut(&out)
	t.Cleanup(func() { metricsRolloutCmd.SetOut(nil) })

	require.NoError(t, runMetricsRollout(metricsRolloutCmd, metricsRolloutOptions{Enforcement: "advisory"}))
	assert.Contains(t, out.String(), "enforcement advisory admissible")
	require.ErrorContains(t, runMetricsRollout(metricsRolloutCmd, metricsRolloutOptions{Enforcement: "blocking"}), "stays advisory")
	require.ErrorContains(t, runMetricsRollout(metricsRolloutCmd, metricsRolloutOptions{Enforcement: "blocking", ReviewFile: filepath.Join(t.TempDir(), "absent.yaml")}), "observe review")

	review := filepath.Join(t.TempDir(), "review.yaml")
	body := "reviewed_by: r\nreviewed_at: 2026-09-20T12:00:00Z\ncompatibility_evidence_ref: e\nagent_denominators_reconciled: true\nsamples_per_agent: {scout: 3}\n"
	require.NoError(t, os.WriteFile(review, []byte(body), 0o600))
	out.Reset()
	require.NoError(t, runMetricsRollout(metricsRolloutCmd, metricsRolloutOptions{Enforcement: "blocking", ReviewFile: review}))
	assert.Contains(t, out.String(), "enforcement blocking admissible")

	metricsRolloutCmd.SetOut(errorWriter{})
	require.ErrorContains(t, runMetricsRollout(metricsRolloutCmd, metricsRolloutOptions{Enforcement: "advisory"}), "write output")
}
