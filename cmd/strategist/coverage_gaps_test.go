package main

// Tests targeting specific uncovered branches to bring cmd/strategist above 90% coverage.

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// --- resolveRuntimeProfile edge cases ---

func TestResolveRuntimeProfile_InvalidActiveYAML(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	require.NoError(t, os.WriteFile(
		filepath.Join(dir, "active.yaml"),
		[]byte(": not: valid: yaml:\n"),
		0o644,
	))
	profile := resolveRuntimeProfile(dir)
	assert.Equal(t, "active_yaml_invalid_yaml", profile.Reason)
	assert.Equal(t, "unknown", profile.PersonaResolved)
}

func TestResolveRuntimeProfile_InvalidPersonaYAML(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	require.NoError(t, os.MkdirAll(filepath.Join(dir, "personas"), 0o755))
	require.NoError(t, os.WriteFile(
		filepath.Join(dir, "active.yaml"),
		[]byte("mode: epic\nbase_path: .analysis\n"),
		0o644,
	))
	require.NoError(t, os.WriteFile(
		filepath.Join(dir, "personas", "epic.yaml"),
		[]byte(": not: valid: yaml:\n"),
		0o644,
	))
	profile := resolveRuntimeProfile(dir)
	assert.Equal(t, "persona_invalid_yaml", profile.Reason)
	assert.Equal(t, "unknown", profile.PersonaResolved)
}

// --- validateActiveYAML: missing mode field ---

func TestValidateCmd_MissingMode(t *testing.T) {
	dir := t.TempDir()
	require.NoError(t, os.MkdirAll(filepath.Join(dir, "personas"), 0o755))
	require.NoError(t, os.MkdirAll(filepath.Join(dir, "roles"), 0o755))
	// active.yaml has the current required fields except mode.
	require.NoError(t, os.WriteFile(filepath.Join(dir, "active.yaml"),
		[]byte("base_path: .analysis\nslots:\n  discovery: brainstorming\n"), 0o644))

	orig := validateRoot
	t.Cleanup(func() { validateRoot = orig })
	validateRoot = dir

	err := validateCmd.RunE(validateCmd, nil)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "validate")
}

// --- validatePersonasDir: missing tone_directive ---

func TestValidateCmd_PersonaMissingToneDirective(t *testing.T) {
	dir := minimalValidateRoot(t)
	// Add a persona that has phase_labels but no tone_directive
	require.NoError(t, os.WriteFile(filepath.Join(dir, "personas", "notone.yaml"),
		[]byte("id: notone\nphase_labels:\n  discovery: analysis\n  refinement: refinement\n  execution: execution\n"), 0o644))

	orig := validateRoot
	t.Cleanup(func() { validateRoot = orig })
	validateRoot = dir

	err := validateCmd.RunE(validateCmd, nil)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "validate")
}

// --- resolveDojoRoots: empty root defaults to ".strategist" ---

func TestResolveDojoRoots_EmptyRootDefaultsToStrategist(t *testing.T) {
	// When root is empty, resolveDojoRoots sets strategistRoot = ".strategist".
	// Reading active.yaml from ".strategist/active.yaml" in a tmp CWD will fail
	// (file not found), but the assignment branch is exercised.
	oldWd, err := os.Getwd()
	require.NoError(t, err)
	t.Cleanup(func() { _ = os.Chdir(oldWd) })
	require.NoError(t, os.Chdir(t.TempDir()))

	_, _, err = resolveDojoRoots("")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "active.yaml")
}

// --- resolveDojoRoots: invalid YAML in active.yaml ---

func TestResolveDojoRoots_InvalidActiveYAML(t *testing.T) {
	dir := t.TempDir()
	strategistRoot := filepath.Join(dir, ".strategist")
	require.NoError(t, os.MkdirAll(strategistRoot, 0o755))
	require.NoError(t, os.WriteFile(filepath.Join(strategistRoot, "active.yaml"),
		[]byte(": not: valid: yaml:\n"), 0o644))

	_, _, err := resolveDojoRoots(strategistRoot)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "parse active.yaml")
}

// --- printInstallCompleteBanner: wizard=true mode ---

func TestPrintInstallCompleteBanner_WizardMode(t *testing.T) {
	out := captureStdout(t, func() {
		printInstallCompleteBanner("/some/target", true, false)
	})
	assert.Contains(t, out, "wizard")
	assert.Contains(t, out, "/some/target")
	assert.NotContains(t, out, "partial")
}

func TestPrintInstallCompleteBanner_Partial(t *testing.T) {
	out := captureStdout(t, func() {
		printInstallCompleteBanner("/some/target", false, true)
	})
	assert.Contains(t, out, "partial")
	assert.Contains(t, out, "strategist compile")
	assert.Contains(t, out, "--strict-compile")
}

func TestInstallIsPartial(t *testing.T) {
	dir := t.TempDir()
	assert.True(t, installIsPartial(dir), "no manifest present must be reported as partial")

	manifestPath := filepath.Join(dir, ".strategist", ".compiled", ".manifest.gz")
	require.NoError(t, os.MkdirAll(filepath.Dir(manifestPath), 0o755))
	require.NoError(t, os.WriteFile(manifestPath, []byte("x"), 0o644))
	assert.False(t, installIsPartial(dir), "manifest present must not be reported as partial")
}

// --- resolveInstallTarget: walk-up finds existing .strategist/ ---

func TestResolveInstallTarget_ExistingStrategistDir(t *testing.T) {
	root := t.TempDir()
	require.NoError(t, os.MkdirAll(filepath.Join(root, ".strategist"), 0o755))

	subdir := filepath.Join(root, "sub")
	require.NoError(t, os.MkdirAll(subdir, 0o755))

	oldWd, err := os.Getwd()
	require.NoError(t, err)
	t.Cleanup(func() { _ = os.Chdir(oldWd) })
	require.NoError(t, os.Chdir(subdir))

	target, err := resolveInstallTarget("", false)
	require.NoError(t, err)
	assert.Equal(t, root, target)
}

// --- compile: CompileAll failure path ---

func TestCompileCmd_CompileAllError(t *testing.T) {
	if os.Getuid() == 0 {
		t.Skip("permission tests do not apply when running as root")
	}
	dir := t.TempDir()

	// Create a minimal .strategist dir but make .compiled/ unwritable so CompileAll fails.
	compiledDir := filepath.Join(dir, ".compiled")
	require.NoError(t, os.MkdirAll(compiledDir, 0o555))
	t.Cleanup(func() { _ = os.Chmod(compiledDir, 0o755) })

	// Also create active.yaml so the dir looks valid.
	require.NoError(t, os.WriteFile(filepath.Join(dir, "active.yaml"),
		[]byte("mode: epic\nbase_path: .analysis\n"), 0o644))

	orig := compileRoot
	t.Cleanup(func() { compileRoot = orig })
	compileRoot = dir

	err := compileCmd.RunE(compileCmd, nil)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "compile")
}
