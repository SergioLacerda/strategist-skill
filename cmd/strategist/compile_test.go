package main

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/SergioLacerda/strategist-skill/internal/testutil"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// --- compile ---

func TestCompileCmd_Success(t *testing.T) {
	dir := t.TempDir()
	testutil.MinimalRoot(t, dir)

	orig := compileRoot
	t.Cleanup(func() { compileRoot = orig })
	compileRoot = dir

	err := compileCmd.RunE(compileCmd, nil)
	require.NoError(t, err)
	assert.FileExists(t, filepath.Join(dir, ".compiled", ".manifest.gz"))
	assert.FileExists(t, filepath.Join(dir, "agent-protocol.md"), "compile must generate agent-protocol.md via RefreshAgentAwareness")
}

func TestCompileCmd_DefaultRoot(t *testing.T) {
	// When compileRoot is empty it defaults to ".strategist"; that dir doesn't
	// exist here so we get an error — but the "if compileRoot == """ branch is covered.
	orig := compileRoot
	t.Cleanup(func() { compileRoot = orig })
	compileRoot = ""

	// Change to a temp dir so ".strategist" definitely doesn't exist.
	oldWd, err := os.Getwd()
	require.NoError(t, err)
	t.Cleanup(func() { _ = os.Chdir(oldWd) })
	require.NoError(t, os.Chdir(t.TempDir()))

	err = compileCmd.RunE(compileCmd, nil)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "not_installed")
	// After the run, compileRoot must be the default value.
	assert.Equal(t, ".strategist", compileRoot)
}

// --- check-stale ---

// TestCompileCmd_PrintsCompletion verifies the success message path.
func TestCompileCmd_PrintsCompletion(t *testing.T) {
	dir := t.TempDir()
	testutil.MinimalRoot(t, dir)

	orig := compileRoot
	t.Cleanup(func() { compileRoot = orig })
	compileRoot = dir

	out := captureStdout(t, func() {
		err := compileCmd.RunE(compileCmd, nil)
		require.NoError(t, err)
	})
	assert.Contains(t, out, "compile complete")
}

// TestInstallCmd_PrintsCompletion verifies the success message (install completes).
