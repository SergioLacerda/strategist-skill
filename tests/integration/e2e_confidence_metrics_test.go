//go:build integration

package integration_test

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const e2eClaimYAML = `claim:
  id: C-1
  statement: the gate is advisory
  agent: ranger
  correlation_key: e2e-c1
  claim_kind: assertion
  confidence_percent: 90
  evidence_ids: [E-1]
  evidence_classes: [explicit]
evidence:
  - id: E-1
    source_ref: docs/adr/0050-confidence-governance-contract.md
    class: explicit
    confidence: high
`

func TestE2E_CLI_ConfidenceMetricsCommands(t *testing.T) {
	t.Parallel()

	workspace := t.TempDir()
	strategistDir := filepath.Join(workspace, ".strategist")
	install := runStrategistCLI(t, workspace, "install", "--target", workspace, "--silent")
	require.Equal(t, 0, install.exitCode, install.output())

	claimFile := filepath.Join(workspace, "claim.yaml")
	require.NoError(t, os.WriteFile(claimFile, []byte(e2eClaimYAML), 0o600))
	metrics := func(args ...string) cliResult {
		return runStrategistCLI(t, workspace, append([]string{"metrics"}, append(args, "--root", strategistDir)...)...)
	}

	record := metrics("record", "--mission", "m-1", "--agent", "ranger", "--claim-file", claimFile)
	require.Equal(t, 0, record.exitCode, record.output())
	assert.Contains(t, record.output(), "coverage_status: reported")
	replay := metrics("record", "--mission", "m-1", "--agent", "ranger", "--claim-file", claimFile)
	require.Equal(t, 0, replay.exitCode, replay.output())

	missing := metrics("record", "--mission", "m-1", "--agent", "scout", "--missing",
		"--correlation-key", "scout-route", "--reason", "scout_summary_not_supplied")
	require.Equal(t, 0, missing.exitCode, missing.output())
	assert.Contains(t, missing.output(), "missing-record recorded")

	history, err := os.ReadFile(filepath.Join(strategistDir, "memory", "confidence-records.jsonl"))
	require.NoError(t, err)
	assert.Equal(t, 2, strings.Count(strings.TrimSpace(string(history)), "\n")+1, "replay must not add a line")

	review := metrics("confidence", "--mission", "m-1")
	require.Equal(t, 0, review.exitCode, review.output())
	assert.Contains(t, review.output(), "sample_size: 1")
	assert.Contains(t, review.output(), "missing_records: 1")
	assert.Contains(t, review.output(), "review_required: true")
	assert.Contains(t, review.output(), "gate_outcome: none")

	outcome := metrics("gate-outcome", "--mission", "m-1", "--outcome", "accepted", "--ref", "gate-event-1")
	require.Equal(t, 0, outcome.exitCode, outcome.output())
	label := metrics("label", "--mission", "m-1", "--subject", "handoff_application",
		"--label", "applied", "--kind", "downstream_verification", "--ref", "verify-1")
	require.Equal(t, 0, label.exitCode, label.output())

	after := metrics("confidence", "--mission", "m-1")
	assert.Contains(t, after.output(), "gate_outcome: accepted")
	assert.Contains(t, metrics("scout").output(), "calibration_status: no_sample")
	handoff := metrics("handoff")
	require.Equal(t, 0, handoff.exitCode, handoff.output())
	assert.Contains(t, handoff.output(), "application_sample_size: 1")

	labels, err := os.ReadFile(filepath.Join(strategistDir, "memory", "ground-truth-labels.jsonl"))
	require.NoError(t, err)
	assert.Contains(t, string(labels), `"subject":"gate_outcome"`)
}
