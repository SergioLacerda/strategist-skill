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

func TestRun_AllPass(t *testing.T) {
	base := t.TempDir()
	strategistDir := filepath.Join(base, ".strategist")
	require.NoError(t, os.MkdirAll(strategistDir, 0o755))

	runDir := filepath.Join(base, "dojo", "run", "todo")
	require.NoError(t, os.MkdirAll(runDir, 0o755))
	require.NoError(t, os.WriteFile(filepath.Join(runDir, "geral.md"),
		[]byte("ideia: KATA_RAPIDO\n"), 0o644))

	logDir := filepath.Join(base, "dojo", ".last-run", "sample-scenario")
	require.NoError(t, os.MkdirAll(logDir, 0o755))
	require.NoError(t, os.WriteFile(filepath.Join(logDir, "emit.log"),
		[]byte("ranger_start\nranger_done\n"), 0o644))

	criteria := domain.DojoCriteria{
		Scenario: "sample-scenario",
		RunDir:   "dojo/run",
		FilesCreated: []domain.DojoFileCheck{
			{Path: "todo/geral.md", MustContain: []string{"KATA_RAPIDO"}},
		},
		EmitLog: domain.DojoEmitLog{MustContain: []string{"ranger_start"}},
	}
	emitLogPath := filepath.Join(logDir, "emit.log")
	result := dojo.Run(criteria, base, strategistDir, emitLogPath, false)
	assert.True(t, result.Passed())
	assert.Equal(t, 0, result.FailCount())
}

func TestRun_FilesOnly_SkipsAllLogDependentChecks(t *testing.T) {
	base := t.TempDir()
	strategistDir := filepath.Join(base, ".strategist")
	require.NoError(t, os.MkdirAll(strategistDir, 0o755))

	runDir := filepath.Join(base, "dojo", "run", "docs")
	require.NoError(t, os.MkdirAll(runDir, 0o755))
	require.NoError(t, os.WriteFile(filepath.Join(runDir, "fixture.md"),
		[]byte("CRITICAL_HIT_CANARY\n"), 0o644))

	criteria := domain.DojoCriteria{
		Scenario: "critical-hit",
		RunDir:   "dojo/run",
		FilesCreated: []domain.DojoFileCheck{
			{Path: "docs/fixture.md", MustContain: []string{"CRITICAL_HIT_CANARY"}},
		},
		Pipeline: domain.DojoPipeline{
			SlotsInvoked:    []string{"execution"},
			SlotsNotInvoked: []string{"discovery", "refinement"},
		},
		EmitLog:        domain.DojoEmitLog{MustContain: []string{"sniper_start"}},
		TimingCriteria: &domain.DojoTimingCriteria{MaxWallTimeMs: 60000},
	}
	// No emit.log written at all — files-only must not require it for any check.
	emitLogPath := filepath.Join(base, "dojo", ".last-run", "critical-hit", "emit.log")
	result := dojo.Run(criteria, base, strategistDir, emitLogPath, true)
	assert.True(t, result.Passed(), "expected pass with --files-only: %+v", result.Items)
}

func TestDojoCheckResult_Passed(t *testing.T) {
	r := domain.DojoCheckResult{
		Scenario: "test",
		Items: []domain.DojoCheckItem{
			{Passed: true},
			{Passed: true},
		},
	}
	assert.True(t, r.Passed())
	assert.Equal(t, 0, r.FailCount())
}

func TestDojoCheckResult_Failed(t *testing.T) {
	r := domain.DojoCheckResult{
		Scenario: "test",
		Items: []domain.DojoCheckItem{
			{Passed: true},
			{Passed: false, Detail: "oops"},
		},
	}
	assert.False(t, r.Passed())
	assert.Equal(t, 1, r.FailCount())
}

func TestDojoCheckResult_Empty(t *testing.T) {
	r := domain.DojoCheckResult{Scenario: "test"}
	assert.True(t, r.Passed())
	assert.Equal(t, 0, r.FailCount())
}
