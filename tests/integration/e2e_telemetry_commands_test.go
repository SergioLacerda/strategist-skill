//go:build integration

package integration_test

import (
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"testing"

	"github.com/SergioLacerda/strategist-skill/internal/testutil"
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

func writeRunbookProvider(t *testing.T, workspace, items string) string {
	t.Helper()

	status := `{"Provider":"treasure-chest-tmp","Version":"0.1.0","SchemaVersion":1,"Healthy":true}`
	search := `{"Envelope":{"Provider":"treasure-chest-tmp","Version":"0.1.0","SchemaVersion":1,"Freshness":"fresh","FallbackState":"none"},"Items":[` + items + `]}`
	source := fmt.Sprintf(`package main

import (
	"fmt"
	"os"
)

const statusResponse = %s
const searchResponse = %s

func main() {
	if len(os.Args) > 1 && os.Args[1] == "status" {
		fmt.Println(statusResponse)
		return
	}
	fmt.Println(searchResponse)
}
`, strconv.Quote(status), strconv.Quote(search))
	return testutil.BuildGoExecutable(t, filepath.Base(filepath.Join(workspace, "treasure-chest-tmp")), source)
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

// TestE2E_CLI_HandoffEvaluate_ThenMetricsHandoff runs `handoff evaluate` on a
// package that requires the challenge, with a passing challenges/acknowledgment
// pair, confirms it appends to .strategist/memory/handoff-challenges.jsonl, and
// then confirms `metrics handoff` picks up that recorded attempt — exercising
// internal/telemetry.AppendHandoffChallenge, ReadHandoffChallenges, and
// ComputeHandoffMetrics with a real, non-empty record, not just the
// empty-file path above.
func TestE2E_CLI_HandoffEvaluate_ThenMetricsHandoff(t *testing.T) {
	t.Parallel()

	workspace, strategistDir := installedTelemetryWorkspace(t)
	startMissionAtHandoff(t, workspace, "e2e-handoff-pass", requiredPackageFacts, implementationPackageTasks)

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

	evaluate := runStrategistCLI(t, workspace,
		"handoff", "evaluate",
		"--challenges", challengesPath,
		"--ack", ackPath,
		"--mission-id", "e2e-handoff-pass",
	)
	require.Equal(t, 0, evaluate.exitCode, evaluate.output())
	assert.Contains(t, evaluate.output(), "outcome: passed")
	assert.Contains(t, evaluate.output(), "passed: true")

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

// TestE2E_CLI_RunbookSelect_NoProviderItems covers the provider-mediated
// selector when the admitted Weapon returns no matching items.
func TestE2E_CLI_RunbookSelect_NoProviderItems(t *testing.T) {
	t.Parallel()

	workspace, _ := installedTelemetryWorkspace(t)
	provider := writeRunbookProvider(t, workspace, "")

	result := runStrategistCLI(t, workspace, "runbook", "select", "--provider-executable", provider, "--mission", "e2e-runbook-empty", "--signal", "timeout")
	require.Equal(t, 0, result.exitCode, result.output())
	assert.Contains(t, result.output(), "no runbook matched the given signals")
}

// TestE2E_CLI_RunbookSelect_MatchingProviderItem exercises the production
// JEWELER-mediated selection path with one Potion-backed runbook item.
func TestE2E_CLI_RunbookSelect_MatchingProviderItem(t *testing.T) {
	t.Parallel()

	workspace, _ := installedTelemetryWorkspace(t)
	provider := writeRunbookProvider(t, workspace, `{"ID":"fix-timeout","Kind":"potion","Score":1,"Trust":1,"Freshness":"fresh","Applicability":"accepted","Reason":"matched","Source":{"Path":"docs/runbooks/fix-timeout.md"}}`)

	result := runStrategistCLI(t, workspace, "runbook", "select", "--provider-executable", provider, "--mission", "e2e-runbook-match", "--signal", "timeout")
	require.Equal(t, 0, result.exitCode, result.output())
	assert.Contains(t, result.output(), "fix-timeout")
}
