package main

import (
	"path/filepath"
	"testing"

	"github.com/SergioLacerda/strategist-skill/internal/domain"
	"github.com/SergioLacerda/strategist-skill/internal/install"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// nonexistentPrepareEmbeddedOptions points every path at a fresh temp dir
// whose catalog.yaml was never written, so ingestForOptions (internal/install)
// fails fast at its first os.ReadFile — cheap, deterministic coverage of the
// error paths in both runPluginsPrepareEmbedded branches without needing a
// real ORKA-shaped external-skills-source/ fixture.
func nonexistentPrepareEmbeddedOptions(t *testing.T) install.PrepareEmbeddedOptions {
	t.Helper()
	dir := t.TempDir()
	return install.PrepareEmbeddedOptions{
		Source:       filepath.Join(dir, "external-skills-source"),
		DefaultsRoot: filepath.Join(dir, "defaults"),
		LockPath:     filepath.Join(dir, "lock.yaml"),
	}
}

func TestRunPluginsPrepareEmbedded_CheckModeMissingCatalogErrors(t *testing.T) {
	opts := nonexistentPrepareEmbeddedOptions(t)

	out := captureStdout(t, func() {
		err := runPluginsPrepareEmbedded(opts, true)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "prepare-embedded --check")
	})
	_ = out
}

func TestRunPluginsPrepareEmbedded_WriteModeMissingCatalogErrors(t *testing.T) {
	opts := nonexistentPrepareEmbeddedOptions(t)

	err := runPluginsPrepareEmbedded(opts, false)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "prepare-embedded:")
}

func TestPrintPrepareEmbeddedReport_PrintsRejectedAndIngested(t *testing.T) {
	report := install.PrepareEmbeddedReport{
		Rejected: []install.IngestionRejection{
			{ID: "bad-skill", Reason: "missing strategist.yaml"},
		},
		Ingested: []install.IngestedSkill{
			{ID: "good-skill", Package: domain.PluginPackage{Digest: "sha256:deadbeef"}},
		},
	}

	out := captureStdout(t, func() {
		printPrepareEmbeddedReport(report)
	})
	assert.Contains(t, out, "rejected bad-skill: missing strategist.yaml")
	assert.Contains(t, out, "ingested good-skill (digest=sha256:deadbeef)")
}

func TestPrintPrepareEmbeddedReport_EmptyReportPrintsNothing(t *testing.T) {
	out := captureStdout(t, func() {
		printPrepareEmbeddedReport(install.PrepareEmbeddedReport{})
	})
	assert.Empty(t, out)
}

// TestPluginsPrepareEmbeddedCmd_RunEInvokesRunPluginsPrepareEmbedded exercises
// the init()-wired RunE closure directly (not runPluginsPrepareEmbedded
// itself), covering the closure body that only cobra dispatch reaches.
func TestPluginsPrepareEmbeddedCmd_RunEInvokesRunPluginsPrepareEmbedded(t *testing.T) {
	dir := t.TempDir()
	require.NoError(t, pluginsPrepareEmbeddedCmd.Flags().Set("source", filepath.Join(dir, "external-skills-source")))
	require.NoError(t, pluginsPrepareEmbeddedCmd.Flags().Set("defaults-root", filepath.Join(dir, "defaults")))
	require.NoError(t, pluginsPrepareEmbeddedCmd.Flags().Set("lock", filepath.Join(dir, "lock.yaml")))
	require.NoError(t, pluginsPrepareEmbeddedCmd.Flags().Set("check", "true"))
	t.Cleanup(func() {
		_ = pluginsPrepareEmbeddedCmd.Flags().Set("source", "external-skills-source")
		_ = pluginsPrepareEmbeddedCmd.Flags().Set("defaults-root", filepath.Join("internal", "embed", "defaults"))
		_ = pluginsPrepareEmbeddedCmd.Flags().Set("lock", install.EmbeddedSkillLockFileName)
		_ = pluginsPrepareEmbeddedCmd.Flags().Set("check", "false")
	})

	_ = captureStdout(t, func() {
		err := pluginsPrepareEmbeddedCmd.RunE(pluginsPrepareEmbeddedCmd, nil)
		require.Error(t, err)
	})
}

// TestPluginsPrepareEmbeddedCmd_IsRegistered is a smoke test confirming the
// subcommand is wired under "plugins" (init() side effect).
func TestPluginsPrepareEmbeddedCmd_IsRegistered(t *testing.T) {
	found := false
	for _, c := range pluginsCmd.Commands() {
		if c.Use == "prepare-embedded" {
			found = true
		}
	}
	assert.True(t, found, "expected prepare-embedded to be registered under plugins")
}
