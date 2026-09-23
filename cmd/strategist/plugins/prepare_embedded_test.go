package plugins

import (
	"bytes"
	"path/filepath"
	"testing"

	"github.com/SergioLacerda/strategist-skill/internal/domain"
	"github.com/SergioLacerda/strategist-skill/internal/install"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// missingCatalogOptions points every path at a fresh temp dir whose
// catalog.yaml was never written, so internal/install fails fast at its first
// read — deterministic coverage of both error branches without a real
// external-skills-source/ fixture.
func missingCatalogOptions(t *testing.T) install.PrepareEmbeddedOptions {
	t.Helper()
	dir := t.TempDir()
	return install.PrepareEmbeddedOptions{
		Source:       filepath.Join(dir, "external-skills-source"),
		DefaultsRoot: filepath.Join(dir, "defaults"),
		LockPath:     filepath.Join(dir, "lock.yaml"),
	}
}

func TestRunPrepareEmbedded_CheckModeMissingCatalogErrors(t *testing.T) {
	t.Parallel()
	err := RunPrepareEmbedded(&bytes.Buffer{}, missingCatalogOptions(t), true)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "prepare-embedded --check")
}

func TestRunPrepareEmbedded_WriteModeMissingCatalogErrors(t *testing.T) {
	t.Parallel()
	err := RunPrepareEmbedded(&bytes.Buffer{}, missingCatalogOptions(t), false)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "prepare-embedded:")
}

func TestPrintPrepareEmbeddedReport_PrintsRejectedAndIngested(t *testing.T) {
	t.Parallel()
	var out bytes.Buffer
	require.NoError(t, printPrepareEmbeddedReport(&out, install.PrepareEmbeddedReport{
		Rejected: []install.IngestionRejection{{ID: "bad-skill", Reason: "missing strategist.yaml"}},
		Ingested: []install.IngestedSkill{{ID: "good-skill", Package: domain.PluginPackage{Digest: "sha256:deadbeef"}}},
	}))
	assert.Contains(t, out.String(), "rejected bad-skill: missing strategist.yaml")
	assert.Contains(t, out.String(), "ingested good-skill (digest=sha256:deadbeef)")
}

func TestPrintPrepareEmbeddedReport_EmptyReportPrintsNothing(t *testing.T) {
	t.Parallel()
	var out bytes.Buffer
	require.NoError(t, printPrepareEmbeddedReport(&out, install.PrepareEmbeddedReport{}))
	assert.Empty(t, out.String())
}

func TestPrepareEmbeddedCmd_DispatchesThroughCobra(t *testing.T) {
	t.Parallel()
	opts := missingCatalogOptions(t)
	_, err := runCommand(t, "plugins", "prepare-embedded", "--check",
		"--source", opts.Source, "--defaults-root", opts.DefaultsRoot, "--lock", opts.LockPath)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "prepare-embedded --check")
}
