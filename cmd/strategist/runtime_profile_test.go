package main

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"testing"

	"github.com/SergioLacerda/strategist-skill/internal/domain"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// --- resolveRuntimeProfile ---

func TestResolveRuntimeProfile_ValidPersona(t *testing.T) {
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
		[]byte(minimalPersonaYAML),
		0o644,
	))

	profile := resolveRuntimeProfile(dir)
	assert.Equal(t, "epic", profile.PersonaResolved)
	assert.Equal(t, "active_yaml", profile.Reason)
	assert.Equal(t, "local", profile.ProfileMode)
}

func TestResolveRuntimeProfile_MissingActiveYAML(t *testing.T) {
	t.Parallel()
	profile := resolveRuntimeProfile(t.TempDir())
	assert.Equal(t, "unknown", profile.PersonaResolved)
	assert.Equal(t, "active_yaml_missing", profile.Reason)
}

func TestResolveRuntimeProfile_MissingPersonaFile(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	require.NoError(t, os.WriteFile(
		filepath.Join(dir, "active.yaml"),
		[]byte("mode: epic\nbase_path: .analysis\n"),
		0o644,
	))
	profile := resolveRuntimeProfile(dir)
	assert.Equal(t, "unknown", profile.PersonaResolved)
	assert.Equal(t, "persona_file_missing", profile.Reason)
}

func TestResolveRuntimeProfile_MissingMode(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	require.NoError(t, os.WriteFile(
		filepath.Join(dir, "active.yaml"),
		[]byte("base_path: .analysis\n"),
		0o644,
	))
	profile := resolveRuntimeProfile(dir)
	assert.Equal(t, "unknown", profile.PersonaResolved)
	assert.Equal(t, "mode_missing", profile.Reason)
}

func TestResolveRuntimeProfile_PersonaMissingDiagnostics(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	require.NoError(t, os.MkdirAll(filepath.Join(dir, "personas"), 0o755))
	require.NoError(t, os.WriteFile(
		filepath.Join(dir, "active.yaml"),
		[]byte("mode: pragmatic\nbase_path: .analysis\n"),
		0o644,
	))
	require.NoError(t, os.WriteFile(
		filepath.Join(dir, "personas", "pragmatic.yaml"),
		[]byte("id: pragmatic\ntone_directive: test\n"),
		0o644,
	))
	profile := resolveRuntimeProfile(dir)
	assert.Equal(t, "unknown", profile.PersonaResolved)
	assert.Equal(t, "persona_diagnostics_missing", profile.Reason)
}

func TestRenderPersonaHeader(t *testing.T) {
	t.Parallel()
	tpl := "⚔️  [Strategist] pipeline=starting mission_id={id} persona={persona} output={output}\n"
	got := renderPersonaHeader(tpl, "mission-123", "epic", "default")
	assert.Equal(t, "⚔️  [Strategist] pipeline=starting mission_id=mission-123 persona=epic output=default", got)
}

// --- exitCodeFor ---

// --- exitCodeFor ---

func TestExitCodeFor(t *testing.T) {
	t.Parallel()
	assert.Equal(t, 2, exitCodeFor(domain.ErrPipelineBypassDetected))
	assert.Equal(t, 3, exitCodeFor(domain.ErrSourceStale))
	assert.Equal(t, 3, exitCodeFor(domain.ErrArtifactAbsent))
	assert.Equal(t, 3, exitCodeFor(domain.ErrManifestMissing))
	assert.Equal(t, 1, exitCodeFor(errors.New("some generic error")))
	assert.Equal(t, 2, exitCodeFor(fmt.Errorf("wrapped: %w", domain.ErrPipelineBypassDetected)))
}

// --- requireStrategistDir ---

// --- requireStrategistDir ---

func TestRequireStrategistDir_FileExists(t *testing.T) {
	oldWd, err := os.Getwd()
	require.NoError(t, err)
	t.Cleanup(func() { _ = os.Chdir(oldWd) })

	dir := t.TempDir()
	require.NoError(t, os.Chdir(dir))
	require.NoError(t, os.MkdirAll(filepath.Join(dir, ".strategist"), 0o755))
	require.NoError(t, os.WriteFile(filepath.Join(dir, ".strategist", "active.yaml"), []byte("mode: epic\n"), 0o644))

	assert.NoError(t, requireStrategistDir())
}

// --- addLine ---
