package main

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/SergioLacerda/strategist-skill/internal/domain"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestCheckCmd_Success(t *testing.T) {
	dir := minimalCheckRoot(t)

	orig := checkRoot
	t.Cleanup(func() { checkRoot = orig })
	checkRoot = dir

	out := captureStdout(t, func() {
		err := checkCmd.RunE(checkCmd, nil)
		require.NoError(t, err)
	})
	assert.Contains(t, out, "STRATEGIST :: check")
	assert.Contains(t, out, "ok")
	assert.Contains(t, out, "brainstorming")
	assert.Contains(t, out, "openspec-explore")
	assert.Contains(t, out, "sdd-ask")
	assert.Contains(t, out, "epic")
	assert.NotContains(t, out, "DELEGATION")
	assert.NotContains(t, out, "delegation_capability")
}

func TestCheckCmd_PersonaMissing(t *testing.T) {
	dir := minimalCheckRoot(t)
	require.NoError(t, os.Remove(filepath.Join(dir, "personas", "epic.yaml")))

	orig := checkRoot
	t.Cleanup(func() { checkRoot = orig })
	checkRoot = dir

	err := checkCmd.RunE(checkCmd, nil)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "check=failed")
}

func TestCheckCmd_RuntimeNormativeFileStale(t *testing.T) {
	dir := minimalCheckRoot(t)
	require.NoError(t, os.WriteFile(filepath.Join(dir, "SKILL.md"), []byte("stale runtime\n"), 0o644))

	orig := checkRoot
	t.Cleanup(func() { checkRoot = orig })
	checkRoot = dir

	var runErr error
	stderr := captureStderr(t, func() {
		runErr = checkCmd.RunE(checkCmd, nil)
	})
	require.Error(t, runErr)
	assert.Contains(t, runErr.Error(), "check=failed")
	assert.Contains(t, stderr, "runtime_stale_unknown_manifest")
	assert.Contains(t, stderr, "SKILL.md")
}

func TestCheckCmd_RuntimeNormativeFileAutoRepairable(t *testing.T) {
	dir := minimalCheckRoot(t)
	stale := []byte("previous default\n")
	require.NoError(t, os.WriteFile(filepath.Join(dir, "SKILL.md"), stale, 0o644))
	writeInstallManifestForTest(t, dir, "SKILL.md", domain.SHA256Hex(stale))

	orig := checkRoot
	t.Cleanup(func() { checkRoot = orig })
	checkRoot = dir

	var runErr error
	stderr := captureStderr(t, func() {
		runErr = checkCmd.RunE(checkCmd, nil)
	})
	require.Error(t, runErr)
	assert.Contains(t, runErr.Error(), "check=failed")
	assert.Contains(t, stderr, "runtime_stale_auto_repairable")
	assert.Contains(t, stderr, "run strategist install")
}

func TestCheckCmd_RuntimeNormativeFileConflict(t *testing.T) {
	dir := minimalCheckRoot(t)
	require.NoError(t, os.WriteFile(filepath.Join(dir, "SKILL.md"), []byte("local edit\n"), 0o644))
	writeInstallManifestForTest(t, dir, "SKILL.md", domain.SHA256Hex([]byte("previous default\n")))

	orig := checkRoot
	t.Cleanup(func() { checkRoot = orig })
	checkRoot = dir

	var runErr error
	stderr := captureStderr(t, func() {
		runErr = checkCmd.RunE(checkCmd, nil)
	})
	require.Error(t, runErr)
	assert.Contains(t, runErr.Error(), "check=failed")
	assert.Contains(t, stderr, "runtime_stale_conflict")
	assert.Contains(t, stderr, "strategist install --force")
}

func TestCheckCmd_PersonaMissingDiagnosticsField(t *testing.T) {
	dir := minimalCheckRoot(t)
	// Overwrite persona without diagnostics.
	require.NoError(t, os.WriteFile(
		filepath.Join(dir, "personas", "epic.yaml"),
		[]byte("id: epic\ntone_directive: test\nphase_labels:\n  discovery: Ranger\n  refinement: Archivist\n  execution: Sniper\n"),
		0o644,
	))

	orig := checkRoot
	t.Cleanup(func() { checkRoot = orig })
	checkRoot = dir

	err := checkCmd.RunE(checkCmd, nil)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "check=failed")
}

func TestCheckCmd_EmptyMode(t *testing.T) {
	dir := minimalCheckRoot(t)
	require.NoError(t, os.WriteFile(
		filepath.Join(dir, "active.yaml"),
		[]byte("base_path: .analysis\nslots:\n  discovery: brainstorming\n  refinement: openspec-explore\n  execution: sdd-ask\n"),
		0o644,
	))

	orig := checkRoot
	t.Cleanup(func() { checkRoot = orig })
	checkRoot = dir

	err := checkCmd.RunE(checkCmd, nil)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "check=failed")
}

func TestCheckCmd_MissingActiveYAML(t *testing.T) {
	orig := checkRoot
	t.Cleanup(func() { checkRoot = orig })
	checkRoot = filepath.Join(t.TempDir(), "nonexistent")

	err := checkCmd.RunE(checkCmd, nil)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "active_yaml_not_found")
}

func TestCheckCmd_ProviderNotInstalled(t *testing.T) {
	dir := t.TempDir()
	require.NoError(t, os.WriteFile(
		filepath.Join(dir, "active.yaml"),
		[]byte("mode: epic\nbase_path: .analysis\nslots:\n  discovery: missing-provider\n  refinement: openspec-explore\n  execution: sdd-ask\n"),
		0o644,
	))
	writeMinimalIdentityFiles(t, dir)

	orig := checkRoot
	t.Cleanup(func() { checkRoot = orig })
	checkRoot = dir

	err := checkCmd.RunE(checkCmd, nil)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "check=failed")
}

func TestCheckCmd_WrongRiskScore(t *testing.T) {
	dir := minimalCheckRoot(t)
	// overwrite brainstorming with wrong risk_score
	require.NoError(t, os.WriteFile(
		filepath.Join(dir, "skills", "brainstorming", "skill.yaml"),
		[]byte("id: brainstorming\nrisk_score: controlled\n"),
		0o644,
	))

	orig := checkRoot
	t.Cleanup(func() { checkRoot = orig })
	checkRoot = dir

	err := checkCmd.RunE(checkCmd, nil)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "check=failed")
}

func TestCheckCmd_NativeRole_Sniper(t *testing.T) {
	dir := t.TempDir()
	// Install skill providers for discovery and refinement.
	for _, p := range []struct {
		name      string
		riskScore string
	}{
		{"brainstorming", "write_analysis"},
		{"openspec-explore", "write_analysis"},
	} {
		provDir := filepath.Join(dir, "skills", p.name)
		require.NoError(t, os.MkdirAll(provDir, 0o755))
		require.NoError(t, os.WriteFile(
			filepath.Join(provDir, "skill.yaml"),
			[]byte("id: "+p.name+"\nrisk_score: "+p.riskScore+"\n"),
			0o644,
		))
	}
	// Install sniper as a native role (no skills/sniper/skill.yaml).
	require.NoError(t, os.MkdirAll(filepath.Join(dir, "roles"), 0o755))
	require.NoError(t, os.WriteFile(
		filepath.Join(dir, "roles", "sniper.yaml"),
		[]byte("role: sniper\nslot: execution\n"),
		0o644,
	))
	require.NoError(t, os.MkdirAll(filepath.Join(dir, "personas"), 0o755))
	require.NoError(t, os.WriteFile(
		filepath.Join(dir, "personas", "epic.yaml"),
		[]byte(minimalPersonaYAML),
		0o644,
	))
	require.NoError(t, os.WriteFile(
		filepath.Join(dir, "active.yaml"),
		[]byte("mode: epic\nbase_path: .analysis\nslots:\n  discovery: brainstorming\n  refinement: openspec-explore\n  execution: sniper\n"),
		0o644,
	))
	writeMinimalIdentityFiles(t, dir)

	orig := checkRoot
	t.Cleanup(func() { checkRoot = orig })
	checkRoot = dir

	out := captureStdout(t, func() {
		err := checkCmd.RunE(checkCmd, nil)
		require.NoError(t, err)
	})
	assert.Contains(t, out, "STRATEGIST :: check")
	assert.Contains(t, out, "sniper")
}

func TestCheckCmd_NativeRole_InvalidRoleDefinition(t *testing.T) {
	dir := t.TempDir()
	for _, p := range []struct {
		name      string
		riskScore string
	}{
		{"brainstorming", "write_analysis"},
		{"openspec-explore", "write_analysis"},
	} {
		provDir := filepath.Join(dir, "skills", p.name)
		require.NoError(t, os.MkdirAll(provDir, 0o755))
		require.NoError(t, os.WriteFile(
			filepath.Join(provDir, "skill.yaml"),
			[]byte("id: "+p.name+"\nrisk_score: "+p.riskScore+"\n"),
			0o644,
		))
	}
	// Native role missing the required `slot` field.
	require.NoError(t, os.MkdirAll(filepath.Join(dir, "roles"), 0o755))
	require.NoError(t, os.WriteFile(
		filepath.Join(dir, "roles", "sniper.yaml"),
		[]byte("role: sniper\n"),
		0o644,
	))
	require.NoError(t, os.MkdirAll(filepath.Join(dir, "personas"), 0o755))
	require.NoError(t, os.WriteFile(
		filepath.Join(dir, "personas", "epic.yaml"),
		[]byte(minimalPersonaYAML),
		0o644,
	))
	require.NoError(t, os.WriteFile(
		filepath.Join(dir, "active.yaml"),
		[]byte("mode: epic\nbase_path: .analysis\nslots:\n  discovery: brainstorming\n  refinement: openspec-explore\n  execution: sniper\n"),
		0o644,
	))
	writeMinimalIdentityFiles(t, dir)

	orig := checkRoot
	t.Cleanup(func() { checkRoot = orig })
	checkRoot = dir

	err := checkCmd.RunE(checkCmd, nil)
	require.Error(t, err, "check must reject a native role definition missing the required slot field")
	assert.Contains(t, err.Error(), "check=failed")
}

func TestCheckCmd_NativeRole_SlotMismatch(t *testing.T) {
	dir := t.TempDir()
	for _, p := range []struct {
		name      string
		riskScore string
	}{
		{"brainstorming", "write_analysis"},
		{"openspec-explore", "write_analysis"},
	} {
		provDir := filepath.Join(dir, "skills", p.name)
		require.NoError(t, os.MkdirAll(provDir, 0o755))
		require.NoError(t, os.WriteFile(
			filepath.Join(provDir, "skill.yaml"),
			[]byte("id: "+p.name+"\nrisk_score: "+p.riskScore+"\n"),
			0o644,
		))
	}
	// Role declares slot=discovery but active.yaml puts it in execution.
	require.NoError(t, os.MkdirAll(filepath.Join(dir, "roles"), 0o755))
	require.NoError(t, os.WriteFile(
		filepath.Join(dir, "roles", "wrong-role.yaml"),
		[]byte("role: wrong-role\nslot: discovery\n"),
		0o644,
	))
	require.NoError(t, os.WriteFile(
		filepath.Join(dir, "active.yaml"),
		[]byte("mode: epic\nbase_path: .analysis\nslots:\n  discovery: brainstorming\n  refinement: openspec-explore\n  execution: wrong-role\n"),
		0o644,
	))
	writeMinimalIdentityFiles(t, dir)

	orig := checkRoot
	t.Cleanup(func() { checkRoot = orig })
	checkRoot = dir

	err := checkCmd.RunE(checkCmd, nil)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "check=failed")
}

func TestCheckCmd_DefaultRoot(t *testing.T) {
	orig := checkRoot
	t.Cleanup(func() { checkRoot = orig })
	checkRoot = ""

	oldWd, err := os.Getwd()
	require.NoError(t, err)
	t.Cleanup(func() { _ = os.Chdir(oldWd) })
	require.NoError(t, os.Chdir(t.TempDir()))

	err = checkCmd.RunE(checkCmd, nil)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "runtime_not_found")
}

// --- resolveRuntimeProfile ---
