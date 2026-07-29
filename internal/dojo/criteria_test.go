package dojo_test

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/SergioLacerda/strategist-skill/internal/dojo"
	"github.com/SergioLacerda/strategist-skill/internal/domain"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestValidateCriteria_OK(t *testing.T) {
	c := domain.DojoCriteria{
		Scenario: "sample-scenario",
		RunDir:   "dojo/run",
		FilesCreated: []domain.DojoFileCheck{
			{Path: "todo/geral.md"},
		},
	}
	assert.NoError(t, dojo.ValidateCriteria(c))
}

func TestValidateCriteria_MissingScenario(t *testing.T) {
	c := domain.DojoCriteria{RunDir: "dojo/run"}
	err := dojo.ValidateCriteria(c)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "scenario is required")
}

func TestValidateCriteria_MissingRunDir(t *testing.T) {
	c := domain.DojoCriteria{Scenario: "sample-scenario"}
	err := dojo.ValidateCriteria(c)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "run_dir is required")
}

func TestValidateCriteria_PathTraversal(t *testing.T) {
	c := domain.DojoCriteria{
		Scenario: "sample-scenario",
		RunDir:   "dojo/run",
		FilesCreated: []domain.DojoFileCheck{
			{Path: "../../outside.md"},
		},
	}
	err := dojo.ValidateCriteria(c)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "escapes its root")
}

func TestValidateCriteria_AbsolutePathRejected(t *testing.T) {
	c := domain.DojoCriteria{
		Scenario: "sample-scenario",
		RunDir:   "dojo/run",
		FilesCreated: []domain.DojoFileCheck{
			{Path: "/etc/passwd"},
		},
	}
	err := dojo.ValidateCriteria(c)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "escapes its root")
}

func TestValidateCriteria_EmptyFilePath(t *testing.T) {
	c := domain.DojoCriteria{
		Scenario:     "sample-scenario",
		RunDir:       "dojo/run",
		FilesCreated: []domain.DojoFileCheck{{Path: ""}},
	}
	err := dojo.ValidateCriteria(c)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "path is required")
}

func TestValidateCriteria_ManifestMissingProvider(t *testing.T) {
	c := domain.DojoCriteria{
		Scenario: "sample-scenario",
		RunDir:   "dojo/run",
		ManifestChecks: []domain.DojoManifestCheck{
			{Slot: "discovery"},
		},
	}
	err := dojo.ValidateCriteria(c)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "expected_provider is required")
}

func TestValidateCriteria_ManifestEmptyField(t *testing.T) {
	c := domain.DojoCriteria{
		Scenario: "sample-scenario",
		RunDir:   "dojo/run",
		ManifestChecks: []domain.DojoManifestCheck{
			{ExpectedProvider: "brainstorming", FieldsPresent: []string{""}},
		},
	}
	err := dojo.ValidateCriteria(c)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "fields_present entries must be non-empty")
}

func TestValidateCriteria_TimingNonPositive(t *testing.T) {
	c := domain.DojoCriteria{
		Scenario:       "sample-scenario",
		RunDir:         "dojo/run",
		TimingCriteria: &domain.DojoTimingCriteria{MaxWallTimeMs: 0},
	}
	err := dojo.ValidateCriteria(c)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "max_wall_time_ms must be positive")
}

func TestLoadCriteria_InvalidIsRejected(t *testing.T) {
	dir := t.TempDir()
	writeCriteria(t, dir, "scenario: \"\"\nrun_dir: dojo/run\n")
	_, err := dojo.LoadCriteria(dir)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "criteria invalid")
}

func TestLoadCriteria_OK(t *testing.T) {
	dir := t.TempDir()
	writeCriteria(t, dir, `
scenario: sample-scenario
description: "test"
run_dir: dojo/run
auto_stop_at_gate: true
files_created:
  - path: "todo/geral.md"
    required_sections: ["ideia:"]
    must_contain: ["KATA_RAPIDO"]
emit_log:
  must_contain: [ranger_start]
  must_not_contain: [sniper_start]
`)
	c, err := dojo.LoadCriteria(dir)
	require.NoError(t, err)
	assert.Equal(t, "sample-scenario", c.Scenario)
	assert.Equal(t, "dojo/run", c.RunDir)
	assert.True(t, c.AutoStopAtGate)
	assert.Len(t, c.FilesCreated, 1)
	assert.Equal(t, []string{"KATA_RAPIDO"}, c.FilesCreated[0].MustContain)
	assert.Equal(t, []string{"ranger_start"}, c.EmitLog.MustContain)
	assert.Equal(t, []string{"sniper_start"}, c.EmitLog.MustNotContain)
}

func TestLoadCriteria_Missing(t *testing.T) {
	_, err := dojo.LoadCriteria(t.TempDir())
	require.Error(t, err)
	assert.Contains(t, err.Error(), "criteria.yaml")
}

func TestLoadCriteria_InvalidYAML(t *testing.T) {
	dir := t.TempDir()
	writeCriteria(t, dir, ": not: valid:\n")
	_, err := dojo.LoadCriteria(dir)
	require.Error(t, err)
}

func writeCriteria(t *testing.T, dir string, content string) {
	t.Helper()
	require.NoError(t, os.WriteFile(filepath.Join(dir, "criteria.yaml"), []byte(content), 0o644))
}
