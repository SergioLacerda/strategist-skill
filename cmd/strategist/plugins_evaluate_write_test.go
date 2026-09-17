package main

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// No t.Parallel() in this file: captureStdout swaps the global os.Stdout,
// which is not safe across concurrently running tests.

func TestRunPluginsEvaluateWrite_AllowsWriteUnderBasePath(t *testing.T) {
	dir := minimalValidateRoot(t)

	var runErr error
	out := captureStdout(t, func() {
		runErr = runPluginsEvaluateWrite(pluginsEvaluateWriteOptions{
			Root:   dir,
			Target: ".analysis/refined/example/tasks.md",
		})
	})
	require.NoError(t, runErr)
	assert.Contains(t, out, "write=allowed")
	assert.Contains(t, out, "permission=analysis.write")
}

func TestRunPluginsEvaluateWrite_DeniesWriteUnderDocs(t *testing.T) {
	dir := minimalValidateRoot(t)

	var runErr error
	out := captureStdout(t, func() {
		runErr = runPluginsEvaluateWrite(pluginsEvaluateWriteOptions{
			Root:   dir,
			Target: "docs/example.md",
		})
	})
	require.Error(t, runErr)
	assert.Contains(t, out, "write=denied")
	assert.Contains(t, out, "permission=docs.write")
	assert.Contains(t, runErr.Error(), "denied")
}

func TestRunPluginsEvaluateWrite_DeniesWriteToSourceFile(t *testing.T) {
	dir := minimalValidateRoot(t)

	var runErr error
	out := captureStdout(t, func() {
		runErr = runPluginsEvaluateWrite(pluginsEvaluateWriteOptions{
			Root:   dir,
			Target: "internal/foo/bar.go",
		})
	})
	require.Error(t, runErr)
	assert.Contains(t, out, "write=denied")
	assert.Contains(t, out, "permission=source.write")
}

func TestRunPluginsEvaluateWrite_MissingTargetErrors(t *testing.T) {
	dir := minimalValidateRoot(t)

	err := runPluginsEvaluateWrite(pluginsEvaluateWriteOptions{Root: dir})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "--target")
}

func TestRunPluginsEvaluateWrite_RootResolutionErrorPropagates(t *testing.T) {
	err := runPluginsEvaluateWrite(pluginsEvaluateWriteOptions{
		Root:   t.TempDir(), // no active.yaml written
		Target: "docs/example.md",
	})
	require.Error(t, err)
	assert.True(t, strings.Contains(err.Error(), "active.yaml") || strings.Contains(err.Error(), "plugins evaluate-write"))
}

// TestReadActiveBasePath_EmptyRootErrors covers readActiveBasePath's
// runtimefs.SafeJoin error branch (SafeJoin rejects an empty root outright).
func TestReadActiveBasePath_EmptyRootErrors(t *testing.T) {
	t.Parallel()
	_, err := readActiveBasePath("")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "resolve active.yaml path")
}

// TestReadActiveBasePath_InvalidYAMLErrors covers readActiveBasePath's
// yaml.Unmarshal error branch.
func TestReadActiveBasePath_InvalidYAMLErrors(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	require.NoError(t, os.WriteFile(filepath.Join(dir, "active.yaml"), []byte(": not: valid: yaml:\n"), 0o644))

	_, err := readActiveBasePath(dir)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "parse active.yaml")
}

// TestReadActiveBasePath_EmptyBasePathErrors covers readActiveBasePath's
// "if cfg.BasePath == \"\"" branch.
func TestReadActiveBasePath_EmptyBasePathErrors(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	require.NoError(t, os.WriteFile(filepath.Join(dir, "active.yaml"), []byte("mode: epic\n"), 0o644))

	_, err := readActiveBasePath(dir)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "base_path is empty")
}

// TestPluginsEvaluateWriteCmd_RunEInvokesRunPluginsEvaluateWrite exercises
// the init()-wired RunE closure directly (every other test in this file
// calls runPluginsEvaluateWrite directly, never through cobra dispatch).
func TestPluginsEvaluateWriteCmd_RunEInvokesRunPluginsEvaluateWrite(t *testing.T) {
	dir := minimalValidateRoot(t)
	require.NoError(t, pluginsEvaluateWriteCmd.Flags().Set(flagRoot, dir))
	require.NoError(t, pluginsEvaluateWriteCmd.Flags().Set("target", ".analysis/refined/example/tasks.md"))
	t.Cleanup(func() {
		_ = pluginsEvaluateWriteCmd.Flags().Set(flagRoot, "")
		_ = pluginsEvaluateWriteCmd.Flags().Set("target", "")
	})

	out := captureStdout(t, func() {
		require.NoError(t, pluginsEvaluateWriteCmd.RunE(pluginsEvaluateWriteCmd, nil))
	})
	assert.Contains(t, out, "write=allowed")
}

func TestPluginsEvaluateWriteCmd_IsRegistered(t *testing.T) {
	t.Parallel()
	found := false
	for _, c := range pluginsCmd.Commands() {
		if c.Use == "evaluate-write" {
			found = true
		}
	}
	assert.True(t, found, "expected evaluate-write to be registered under plugins")
}
