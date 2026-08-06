//go:build integration

package integration_test

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// This file covers docs/integration-coverage-gaps.md item C2: CLI-level
// integration scenarios for the mission-pipeline telemetry commands
// (`metrics handoff`, `metrics scout`, `handoff verify`, `runbook select`).
// Before this file, no tests/integration/*.go scenario ever invoked these
// subcommands, leaving internal/telemetry's mission-pipeline emission paths
// (mission_run.go, route_decision.go, route_metrics.go,
// scout_metrics_history.go, handoff_metrics.go,
// handoff_challenge_record.go) unreached under the `integration` build tag —
// see .analysis/refined/20260805-integration-coverage-mapping/analysis.md.

func installedTelemetryWorkspace(t *testing.T) (workspace, strategistDir string) {
	t.Helper()

	workspace = t.TempDir()
	strategistDir = filepath.Join(workspace, ".strategist")

	install := runStrategistCLI(t, workspace, "install", "--target", workspace, "--silent")
	require.Equal(t, 0, install.exitCode, install.output())

	writeHappyPathActiveYAML(t, strategistDir)

	compile := runStrategistCLI(t, workspace, "compile", "--root", strategistDir)
	require.Equal(t, 0, compile.exitCode, compile.output())

	return workspace, strategistDir
}

// TestE2E_CLI_MetricsHandoff_EmptyMemory covers `metrics handoff` against a
// freshly installed workspace with no handoff-challenges.jsonl yet — the
// documented "prints all rates as 0, not an error" path.
func TestE2E_CLI_MetricsHandoff_EmptyMemory(t *testing.T) {
	t.Parallel()

	workspace, _ := installedTelemetryWorkspace(t)

	result := runStrategistCLI(t, workspace, "metrics", "handoff")
	require.Equal(t, 0, result.exitCode, result.output())
	assert.Contains(t, result.output(), "handoff_pass_rate: 0.00")
	assert.Contains(t, result.output(), "sample_size: 0")
}

// TestE2E_CLI_HandoffVerify_ThenMetricsHandoff runs `handoff verify` against
// a passing challenges/acknowledgment pair, confirms it appends to
// .strategist/memory/handoff-challenges.jsonl, and then confirms `metrics
// handoff` picks up that recorded attempt — exercising
// internal/telemetry.AppendHandoffChallenge, ReadHandoffChallenges, and
// ComputeHandoffMetrics with a real, non-empty record, not just the
// empty-file path above.
func TestE2E_CLI_HandoffVerify_ThenMetricsHandoff(t *testing.T) {
	t.Parallel()

	workspace, strategistDir := installedTelemetryWorkspace(t)

	// Matches DefaultPolicy() (cmd/strategist/handoff_verify.go via
	// internal/handoff.DefaultPolicy) required types for archivist_to_sniper:
	// objective, boundary, classification, gate — same fixture shape proven
	// to pass in cmd/strategist/handoff_verify_test.go's
	// handoffVerifyChallengesYAML/handoffVerifyAckPassYAML.
	challengesPath := filepath.Join(workspace, "challenges.yaml")
	require.NoError(t, os.WriteFile(challengesPath, []byte(`
challenges:
  - id: HC-001
    type: objective
    source_refs: [G-001]
    critical: true
  - id: HC-002
    type: boundary
    source_refs: [X-001]
    critical: true
  - id: HC-003
    type: classification
    source_refs: [D-001, Q-001]
    critical: true
    expected_classification:
      D-001: approved_decision
      Q-001: unresolved_question
  - id: HC-004
    type: gate
    source_refs: [approval.required]
    critical: true
    expected_gate_allowed: false
`), 0o644))

	ackPath := filepath.Join(workspace, "ack.yaml")
	require.NoError(t, os.WriteFile(ackPath, []byte(`
challenge_refs: [HC-001, HC-002, HC-003, HC-004]
understood_refs: [G-001, X-001, D-001, Q-001, approval.required]
classifications:
  D-001: approved_decision
  Q-001: unresolved_question
gate_allowed: false
`), 0o644))

	verify := runStrategistCLI(t, workspace,
		"handoff", "verify",
		"--transition", "archivist_to_sniper",
		"--challenges", challengesPath,
		"--ack", ackPath,
		"--mission-id", "e2e-handoff-pass",
	)
	require.Equal(t, 0, verify.exitCode, verify.output())
	assert.Contains(t, verify.output(), "passed: true")

	historyPath := filepath.Join(strategistDir, "memory", "handoff-challenges.jsonl")
	history, err := os.ReadFile(historyPath)
	require.NoError(t, err)
	assert.Contains(t, string(history), `"mission_id":"e2e-handoff-pass"`)

	metrics := runStrategistCLI(t, workspace, "metrics", "handoff")
	require.Equal(t, 0, metrics.exitCode, metrics.output())
	assert.Contains(t, metrics.output(), "sample_size: 1")
}

// TestE2E_CLI_MetricsScout_EmptyMemory covers `metrics scout` against a
// freshly installed workspace with neither route-decisions.jsonl nor
// outcomes.jsonl yet — the documented "runs cleanly ... printing
// sample_size: 0" path (internal/telemetry's ReadRouteDecisions/
// ReadOutcomes missing-file branches, and ComputeRouteMetrics on an empty
// input).
func TestE2E_CLI_MetricsScout_EmptyMemory(t *testing.T) {
	t.Parallel()

	workspace, _ := installedTelemetryWorkspace(t)

	result := runStrategistCLI(t, workspace, "metrics", "scout")
	require.Equal(t, 0, result.exitCode, result.output())
	assert.Contains(t, result.output(), "fallback_rate: 0.00")
	assert.Contains(t, result.output(), "sample_size: 0")
}

// TestE2E_CLI_RunbookSelect_NoSidecars covers `runbook select` against a
// workspace with no docs/runbooks/*.runbook.yaml sidecars — the documented
// "prints an empty-result message (exit 0)" path.
func TestE2E_CLI_RunbookSelect_NoSidecars(t *testing.T) {
	t.Parallel()

	workspace, _ := installedTelemetryWorkspace(t)

	result := runStrategistCLI(t, workspace, "runbook", "select", "--signal", "timeout")
	require.Equal(t, 0, result.exitCode, result.output())
	assert.Contains(t, result.output(), "no docs/runbooks/*.runbook.yaml sidecars found")
}

// TestE2E_CLI_RunbookSelect_MatchingSidecar seeds one matching runbook
// sidecar and confirms `runbook select` selects it end to end through the
// real CLI binary — internal/runbook.Select plus the table-render path,
// neither of which any prior tests/integration scenario reached.
func TestE2E_CLI_RunbookSelect_MatchingSidecar(t *testing.T) {
	t.Parallel()

	workspace, _ := installedTelemetryWorkspace(t)

	runbooksDir := filepath.Join(workspace, "docs", "runbooks")
	require.NoError(t, os.MkdirAll(runbooksDir, 0o755))
	require.NoError(t, os.WriteFile(filepath.Join(runbooksDir, "fix-timeout.runbook.yaml"), []byte(`schema_version: "1"
runbook_id: fix-timeout
runbook_type: analytical
source_doc: docs/runbooks/fix-timeout.md
applies_when:
  - timeout
objective: diagnose timeout issues
checks:
  - id: check-logs
    level: mandatory
`), 0o644))

	result := runStrategistCLI(t, workspace, "runbook", "select", "--signal", "timeout")
	require.Equal(t, 0, result.exitCode, result.output())
	assert.Contains(t, result.output(), "fix-timeout")
}
