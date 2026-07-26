package main

import (
	"bytes"
	"log/slog"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestValidateCmd_Success(t *testing.T) {
	dir := minimalValidateRoot(t)

	orig := validateRoot
	t.Cleanup(func() { validateRoot = orig })
	validateRoot = dir

	out := captureStdout(t, func() {
		err := validateCmd.RunE(validateCmd, nil)
		require.NoError(t, err)
	})
	assert.Contains(t, out, "validate OK")
	assert.Contains(t, out, dir)
}

func TestValidateCmd_EmitsStructuredTelemetry(t *testing.T) {
	root := minimalValidateRoot(t)

	orig := validateRoot
	t.Cleanup(func() { validateRoot = orig })
	validateRoot = root

	var buf bytes.Buffer
	h := slog.NewTextHandler(&buf, &slog.HandlerOptions{Level: slog.LevelInfo})
	prev := slog.Default()
	slog.SetDefault(slog.New(h))
	t.Cleanup(func() { slog.SetDefault(prev) })

	err := validateCmd.RunE(validateCmd, nil)
	require.NoError(t, err)

	out := buf.String()
	assert.Contains(t, out, "strategist.component=validate")
	assert.Contains(t, out, "strategist.runtime_mode=cli")
	assert.Contains(t, out, "strategist.output_profile=default")
	assert.Contains(t, out, "strategist.target="+root)
}

func TestValidateCmd_WithKnowledgeIndex(t *testing.T) {
	dir := minimalValidateRoot(t)
	require.NoError(t, os.WriteFile(filepath.Join(dir, "knowledge.index.yaml"),
		[]byte("schema_version: \"1\"\nsources: []\n"), 0o644))

	orig := validateRoot
	t.Cleanup(func() { validateRoot = orig })
	validateRoot = dir

	out := captureStdout(t, func() {
		err := validateCmd.RunE(validateCmd, nil)
		require.NoError(t, err)
	})
	assert.Contains(t, out, "validate OK")
}

func TestValidateCmd_MissingRoot(t *testing.T) {
	orig := validateRoot
	t.Cleanup(func() { validateRoot = orig })
	validateRoot = filepath.Join(t.TempDir(), "nonexistent")

	err := validateCmd.RunE(validateCmd, nil)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "validate")
}

func TestValidateCmd_MissingActiveYAML(t *testing.T) {
	dir := t.TempDir()
	require.NoError(t, os.MkdirAll(filepath.Join(dir, "personas"), 0o755))
	require.NoError(t, os.MkdirAll(filepath.Join(dir, "roles"), 0o755))

	orig := validateRoot
	t.Cleanup(func() { validateRoot = orig })
	validateRoot = dir

	err := validateCmd.RunE(validateCmd, nil)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "validate")
}

func TestValidateCmd_InvalidMode(t *testing.T) {
	dir := minimalValidateRoot(t)
	require.NoError(t, os.WriteFile(filepath.Join(dir, "active.yaml"),
		[]byte("mode: invalid_mode\nbase_path: .analysis\nslots:\n  discovery: brainstorming\n"), 0o644))

	orig := validateRoot
	t.Cleanup(func() { validateRoot = orig })
	validateRoot = dir

	err := validateCmd.RunE(validateCmd, nil)
	require.Error(t, err)
}

func TestValidateCmd_MissingSlot(t *testing.T) {
	dir := minimalValidateRoot(t)
	// overwrite roles/default.yaml without the required slots
	require.NoError(t, os.WriteFile(filepath.Join(dir, "roles", "default.yaml"),
		[]byte("discovery: brainstorming\n"), 0o644))

	orig := validateRoot
	t.Cleanup(func() { validateRoot = orig })
	validateRoot = dir

	err := validateCmd.RunE(validateCmd, nil)
	require.Error(t, err)
}

func TestValidateCmd_InvalidActiveYAML(t *testing.T) {
	dir := t.TempDir()
	require.NoError(t, os.MkdirAll(filepath.Join(dir, "personas"), 0o755))
	require.NoError(t, os.MkdirAll(filepath.Join(dir, "roles"), 0o755))
	require.NoError(t, os.WriteFile(filepath.Join(dir, "active.yaml"),
		[]byte(": invalid: yaml: content:\n"), 0o644))

	orig := validateRoot
	t.Cleanup(func() { validateRoot = orig })
	validateRoot = dir

	err := validateCmd.RunE(validateCmd, nil)
	require.Error(t, err)
}

func TestValidateCmd_MissingRequiredField(t *testing.T) {
	dir := minimalValidateRoot(t)
	// active.yaml missing slots
	require.NoError(t, os.WriteFile(filepath.Join(dir, "active.yaml"),
		[]byte("mode: pragmatic\nbase_path: .analysis\n"), 0o644))

	orig := validateRoot
	t.Cleanup(func() { validateRoot = orig })
	validateRoot = dir

	err := validateCmd.RunE(validateCmd, nil)
	require.Error(t, err)
}

func TestValidateCmd_InvalidPersonaYAML(t *testing.T) {
	dir := minimalValidateRoot(t)
	require.NoError(t, os.WriteFile(filepath.Join(dir, "personas", "bad.yaml"),
		[]byte(": not: valid: yaml:\n"), 0o644))

	orig := validateRoot
	t.Cleanup(func() { validateRoot = orig })
	validateRoot = dir

	err := validateCmd.RunE(validateCmd, nil)
	require.Error(t, err)
}

func TestValidateCmd_PersonaMissingField(t *testing.T) {
	dir := minimalValidateRoot(t)
	// persona without phase_labels
	require.NoError(t, os.WriteFile(filepath.Join(dir, "personas", "minimal.yaml"),
		[]byte("id: minimal\ntone_directive: brief\n"), 0o644))

	orig := validateRoot
	t.Cleanup(func() { validateRoot = orig })
	validateRoot = dir

	err := validateCmd.RunE(validateCmd, nil)
	require.Error(t, err)
}

func TestValidateCmd_InvalidRoleYAML(t *testing.T) {
	dir := minimalValidateRoot(t)
	require.NoError(t, os.WriteFile(filepath.Join(dir, "roles", "bad.yaml"),
		[]byte(": not: valid: yaml:\n"), 0o644))

	orig := validateRoot
	t.Cleanup(func() { validateRoot = orig })
	validateRoot = dir

	err := validateCmd.RunE(validateCmd, nil)
	require.Error(t, err)
}

func TestValidateCmd_RoleFileInvalidSlotValue(t *testing.T) {
	dir := minimalValidateRoot(t)
	require.NoError(t, os.WriteFile(filepath.Join(dir, "roles", "sniper.yaml"),
		[]byte("role: sniper\nslot: bogus\n"), 0o644))

	orig := validateRoot
	t.Cleanup(func() { validateRoot = orig })
	validateRoot = dir

	err := validateCmd.RunE(validateCmd, nil)
	require.Error(t, err, "validate must reject a role file declaring an unknown slot")
}

func TestValidateCmd_RoleFileMissingSlot(t *testing.T) {
	dir := minimalValidateRoot(t)
	require.NoError(t, os.WriteFile(filepath.Join(dir, "roles", "sniper.yaml"),
		[]byte("role: sniper\n"), 0o644))

	orig := validateRoot
	t.Cleanup(func() { validateRoot = orig })
	validateRoot = dir

	err := validateCmd.RunE(validateCmd, nil)
	require.Error(t, err, "validate must reject a role definition missing slot")
}

func TestValidateCmd_RoleSlotMapMissingEntry(t *testing.T) {
	dir := minimalValidateRoot(t)
	require.NoError(t, os.WriteFile(filepath.Join(dir, "roles", "default.yaml"),
		[]byte("discovery: brainstorming\n"), 0o644))

	orig := validateRoot
	t.Cleanup(func() { validateRoot = orig })
	validateRoot = dir

	err := validateCmd.RunE(validateCmd, nil)
	require.Error(t, err, "validate must reject a slot map missing refinement/execution")
}

func TestValidateCmd_InvalidKnowledgeIndex(t *testing.T) {
	dir := minimalValidateRoot(t)
	require.NoError(t, os.WriteFile(filepath.Join(dir, "knowledge.index.yaml"),
		[]byte(": not: valid: yaml:\n"), 0o644))

	orig := validateRoot
	t.Cleanup(func() { validateRoot = orig })
	validateRoot = dir

	err := validateCmd.RunE(validateCmd, nil)
	require.Error(t, err)
}

func TestValidateCmd_DefaultRoot(t *testing.T) {
	// When validateRoot is empty, auto-discovery walks up from CWD.
	// In an empty temp dir (no .strategist/), it returns a "runtime not found" error.
	orig := validateRoot
	t.Cleanup(func() { validateRoot = orig })
	validateRoot = ""

	oldWd, err := os.Getwd()
	require.NoError(t, err)
	t.Cleanup(func() { _ = os.Chdir(oldWd) })
	require.NoError(t, os.Chdir(t.TempDir()))

	err = validateCmd.RunE(validateCmd, nil)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "runtime not found")
}

// TestCompileCmd_PrintsCompletion verifies the success message path.
