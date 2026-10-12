package check

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/SergioLacerda/strategist-skill/internal/stale"
	"github.com/SergioLacerda/strategist-skill/internal/testutil"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// --- check-stale ---

func TestCheckStaleCmd_FreshArtifact(t *testing.T) {
	_, artifactPath := freshArtifactDir(t)
	err := checkStaleCmd.RunE(checkStaleCmd, []string{artifactPath})
	require.NoError(t, err) // fresh → isStale=false → no os.Exit
}

func TestCheckStaleCmd_FreshArtifactJSON(t *testing.T) {
	_, artifactPath := freshArtifactDir(t)
	origJSON, origQuiet := checkStaleJSON, checkStaleQuiet
	t.Cleanup(func() {
		checkStaleJSON = origJSON
		checkStaleQuiet = origQuiet
	})
	checkStaleJSON = true
	checkStaleQuiet = false

	out := captureStdout(t, func() {
		err := checkStaleCmd.RunE(checkStaleCmd, []string{artifactPath})
		require.NoError(t, err)
	})

	var result stale.Result
	require.NoError(t, json.Unmarshal([]byte(strings.TrimSpace(out)), &result))
	assert.False(t, result.Stale)
	assert.Equal(t, stale.ReasonFresh, result.Reason)
	assert.Equal(t, artifactPath, result.ArtifactPath)
}

func TestPrintCheckStaleResult_DefaultAndQuiet(t *testing.T) {
	result := stale.Result{
		Stale:        true,
		Reason:       stale.ReasonMissingSource,
		ArtifactPath: "artifact.gz",
		SourcePath:   "active.yaml",
	}
	origJSON, origQuiet := checkStaleJSON, checkStaleQuiet
	t.Cleanup(func() {
		checkStaleJSON = origJSON
		checkStaleQuiet = origQuiet
	})

	checkStaleJSON = false
	checkStaleQuiet = false
	out := captureStdout(t, func() {
		require.NoError(t, printCheckStaleResult(result))
	})
	assert.Contains(t, out, "stale: missing_source path=active.yaml")

	checkStaleQuiet = true
	out = captureStdout(t, func() {
		require.NoError(t, printCheckStaleResult(result))
	})
	assert.Empty(t, out)
}

func TestPrintCheckStaleResult_NoSourcePathUsesArtifactFallback(t *testing.T) {
	result := stale.Result{
		Stale:        true,
		Reason:       stale.ReasonMissingSource,
		ArtifactPath: "artifact.gz",
	}
	origJSON, origQuiet := checkStaleJSON, checkStaleQuiet
	t.Cleanup(func() {
		checkStaleJSON = origJSON
		checkStaleQuiet = origQuiet
	})
	checkStaleJSON = false
	checkStaleQuiet = false

	out := captureStdout(t, func() {
		require.NoError(t, printCheckStaleResult(result))
	})
	assert.Contains(t, out, "stale: missing_source artifact=artifact.gz")
}

func TestCheckStaleCmd_WithMissionRunDoesNotError(t *testing.T) {
	_, artifactPath := freshArtifactDir(t)
	attachMissionRun(t, checkStaleCmd)

	err := checkStaleCmd.RunE(checkStaleCmd, []string{artifactPath})
	require.NoError(t, err)
}

func TestCheckStaleCmd_CorruptArtifact(t *testing.T) {
	dir := t.TempDir()
	art := filepath.Join(dir, "artifact.gz")
	require.NoError(t, os.WriteFile(art, []byte("not gzip"), 0o644))
	testutil.WriteGzJSON(t, filepath.Join(dir, ".manifest.gz"), map[string]any{})

	err := checkStaleCmd.RunE(checkStaleCmd, []string{art})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "check-stale")
}

// --- install ---
