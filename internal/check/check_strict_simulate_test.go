package check

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/SergioLacerda/strategist-skill/internal/compile"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// --- check --strict ---

func resetCheckFlags(t *testing.T) {
	t.Helper()
	origRoot, origStrict, origSimulate := checkRoot, checkStrict, checkSimulate
	t.Cleanup(func() {
		checkRoot, checkStrict, checkSimulate = origRoot, origStrict, origSimulate
	})
}

func compileMinimalCheckRoot(t *testing.T, dir string) {
	t.Helper()
	require.NoError(t, os.MkdirAll(filepath.Join(dir, "roles"), 0o755))
	require.NoError(t, os.WriteFile(filepath.Join(dir, "roles", "default.yaml"),
		[]byte("discovery: brainstorming\nrefinement: openspec-explore\nexecution: sdd-ask\n"), 0o644))
	require.NoError(t, os.WriteFile(filepath.Join(dir, "knowledge.index.yaml"), []byte("sources: []\n"), 0o644))
	require.NoError(t, os.WriteFile(filepath.Join(dir, "index.yaml"), []byte("load_always: []\nload_by_task_type: {}\n"), 0o644))

	kiPath := filepath.Join(dir, "knowledge.index.yaml")
	require.NoError(t, compile.Compiler{}.CompileAll(dir, kiPath))
}

func TestCheckCmd_Strict_Success(t *testing.T) {
	resetCheckFlags(t)
	dir := minimalCheckRoot(t)
	compileMinimalCheckRoot(t, dir)

	checkRoot = dir
	checkStrict = true

	err := checkCmd.RunE(checkCmd, nil)
	require.NoError(t, err, "check --strict must pass against a freshly compiled root")
}

func TestCheckCmd_Strict_MissingArtifact(t *testing.T) {
	resetCheckFlags(t)
	dir := minimalCheckRoot(t)
	compileMinimalCheckRoot(t, dir)
	require.NoError(t, os.Remove(filepath.Join(dir, ".compiled", ".config.gz")))

	checkRoot = dir
	checkStrict = true
	err := checkCmd.RunE(checkCmd, nil)
	require.Error(t, err, "check --strict must fail when a compiled artifact is missing")
	assert.Contains(t, err.Error(), "check=failed")

	// Plain (non-strict) check must still pass — strict only adds checks.
	checkStrict = false
	err = checkCmd.RunE(checkCmd, nil)
	require.NoError(t, err, "plain check must not be affected by a missing compiled artifact")
}

func TestCheckCmd_Strict_HashDrift(t *testing.T) {
	resetCheckFlags(t)
	dir := minimalCheckRoot(t)
	compileMinimalCheckRoot(t, dir)
	// Hand-edit a compiled artifact after the manifest was generated.
	require.NoError(t, os.WriteFile(filepath.Join(dir, ".compiled", ".config.gz"), []byte("tampered"), 0o644))

	checkRoot = dir
	checkStrict = true
	err := checkCmd.RunE(checkCmd, nil)
	require.Error(t, err, "check --strict must detect manifest hash drift")
	assert.Contains(t, err.Error(), "check=failed")
}

// --- check --simulate ---

func TestCheckCmd_Simulate_NoMutation(t *testing.T) {
	resetCheckFlags(t)
	dir := minimalCheckRoot(t)

	before, err := filepath.Glob(filepath.Join(dir, "**"))
	require.NoError(t, err)

	checkRoot = dir
	checkSimulate = true

	out := captureStdout(t, func() {
		err := checkCmd.RunE(checkCmd, nil)
		require.NoError(t, err)
	})
	assert.Contains(t, out, "READINESS")
	assert.Contains(t, out, "pipeline_route")
	assert.Contains(t, out, "decision_reason")

	after, err := filepath.Glob(filepath.Join(dir, "**"))
	require.NoError(t, err)
	assert.Equal(t, before, after, "--simulate must not create, modify, or delete any files")
}

func TestCheckCmd_Simulate_ReportsBlockers(t *testing.T) {
	resetCheckFlags(t)
	dir := minimalCheckRoot(t)
	require.NoError(t, os.Remove(filepath.Join(dir, "personas", "epic.yaml")))

	checkRoot = dir
	checkSimulate = true

	var runErr error
	out := captureStdout(t, func() {
		runErr = checkCmd.RunE(checkCmd, nil)
	})
	require.Error(t, runErr, "--simulate must still surface real blockers, not hide them")
	assert.Contains(t, out, "BLOCKERS")
}
