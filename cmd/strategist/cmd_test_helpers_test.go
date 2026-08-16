package main

import (
	"bytes"
	"context"
	"os"
	"path/filepath"
	"testing"

	"github.com/SergioLacerda/strategist-skill/internal/telemetry"
	"github.com/spf13/cobra"
	"github.com/stretchr/testify/require"
)

// attachMissionRun wires a non-nil telemetry.MissionRun into cmd's context, so
// that RunE bodies gated on `telemetryRunFromCmd(cmd) != nil` (the SetSilent()
// branch present in most treasure-chest subcommands) get exercised. Returns
// the run so callers can inspect it (e.g. Snapshot()) if needed, and restores
// the command's original context on test cleanup.
func attachMissionRun(t *testing.T, cmd *cobra.Command) *telemetry.MissionRun {
	t.Helper()
	run := telemetry.NewMissionRun("test-mission")
	origCtx := cmd.Context()
	t.Cleanup(func() { cmd.SetContext(origCtx) })
	cmd.SetContext(telemetry.WithMissionRun(context.Background(), run))
	return run
}

// --- helpers ---

// captureStdout replaces os.Stdout with a pipe and returns whatever was written.

// captureStdout replaces os.Stdout with a pipe and returns whatever was written.
func captureStdout(t *testing.T, fn func()) string {
	t.Helper()
	r, w, err := os.Pipe()
	require.NoError(t, err)
	old := os.Stdout
	os.Stdout = w
	fn()
	require.NoError(t, w.Close())
	os.Stdout = old
	var buf bytes.Buffer
	_, err = buf.ReadFrom(r)
	require.NoError(t, err)
	return buf.String()
}

// captureStderr replaces os.Stderr with a pipe and returns whatever was written.

// captureStderr replaces os.Stderr with a pipe and returns whatever was written.
func captureStderr(t *testing.T, fn func()) string {
	t.Helper()
	r, w, err := os.Pipe()
	require.NoError(t, err)
	old := os.Stderr
	os.Stderr = w
	fn()
	require.NoError(t, w.Close())
	os.Stderr = old
	var buf bytes.Buffer
	_, err = buf.ReadFrom(r)
	require.NoError(t, err)
	return buf.String()
}

// --- version ---

// --- validate ---

// minimalValidateRoot creates a .strategist/-like tree suitable for validateCmd:
// active.yaml, personas/pragmatic.yaml, roles/default.yaml.
func minimalValidateRoot(t *testing.T) string {
	t.Helper()
	dir := t.TempDir()
	require.NoError(t, os.MkdirAll(filepath.Join(dir, "personas"), 0o755))
	require.NoError(t, os.MkdirAll(filepath.Join(dir, "roles"), 0o755))
	require.NoError(t, os.WriteFile(filepath.Join(dir, "active.yaml"),
		[]byte("mode: pragmatic\nbase_path: .analysis\nslots:\n  discovery: brainstorming\n"), 0o644))
	require.NoError(t, os.WriteFile(filepath.Join(dir, "personas", "pragmatic.yaml"),
		[]byte("id: pragmatic\ntone_directive: precise\nphase_labels:\n  discovery: analysis\n  refinement: refinement\n  execution: execution\ndiagnostics:\n  pipeline_header: \"[Strategist] pipeline=starting\"\n  bootstrap_origin: \"[Strategist] profile_path={path}\"\n"), 0o644))
	require.NoError(t, os.WriteFile(filepath.Join(dir, "roles", "default.yaml"),
		[]byte("discovery: brainstorming\nrefinement: archivist\nexecution: caveman\n"), 0o644))
	return dir
}

// --- dojo ---

func setupDojoScenario(t *testing.T, scenario, criteria, runContent string) string {
	t.Helper()
	dir := t.TempDir()

	strategistRoot := filepath.Join(dir, ".strategist")
	require.NoError(t, os.MkdirAll(strategistRoot, 0o755))
	require.NoError(t, os.WriteFile(filepath.Join(strategistRoot, "active.yaml"),
		[]byte("mode: epic\nbase_path: .analysis\nslots:\n  discovery: brainstorming\n"), 0o644))

	scenarioDir := filepath.Join(dir, ".analysis", "dojo", scenario)
	require.NoError(t, os.MkdirAll(scenarioDir, 0o755))
	require.NoError(t, os.WriteFile(filepath.Join(scenarioDir, "criteria.yaml"), []byte(criteria), 0o644))

	if runContent != "" {
		runDir := filepath.Join(dir, ".analysis", "dojo", "run", "todo")
		require.NoError(t, os.MkdirAll(runDir, 0o755))
		require.NoError(t, os.WriteFile(filepath.Join(runDir, "geral.md"), []byte(runContent), 0o644))
	}
	return strategistRoot
}

// --- runtime_profile ---

// minimalPersonaYAML is also used directly by internal/check's own duplicate
// (check_test_helpers_test.go) — kept here only for runtime_profile_test.go,
// which is not part of the check cluster.
const minimalPersonaYAML = `id: epic
tone_directive: test tone
phase_labels:
  discovery: Ranger
  refinement: Archivist
  execution: Sniper
diagnostics:
  pipeline_header: "[Strategist] pipeline=starting mission_id={id} persona=epic\n"
  bootstrap_origin: "[Strategist] profile_path={path} active_yaml={active} reason={reason}\n"
`
