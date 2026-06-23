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

func writeCriteria(t *testing.T, dir string, content string) {
	t.Helper()
	require.NoError(t, os.WriteFile(filepath.Join(dir, "criteria.yaml"), []byte(content), 0o644))
}

func TestLoadCriteria_OK(t *testing.T) {
	dir := t.TempDir()
	writeCriteria(t, dir, `
scenario: quick-draw
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
	assert.Equal(t, "quick-draw", c.Scenario)
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

func TestCheckFiles_FileExists(t *testing.T) {
	base := t.TempDir()
	runDir := filepath.Join(base, "dojo", "run")
	require.NoError(t, os.MkdirAll(filepath.Join(runDir, "todo"), 0o755))
	require.NoError(t, os.WriteFile(filepath.Join(runDir, "todo", "geral.md"),
		[]byte("ideia: KATA_RAPIDO test idea\n"), 0o644))

	criteria := domain.DojoCriteria{
		RunDir: "dojo/run",
		FilesCreated: []domain.DojoFileCheck{
			{
				Path:             "todo/geral.md",
				RequiredSections: []string{"ideia:"},
				MustContain:      []string{"KATA_RAPIDO"},
			},
		},
	}
	items := dojo.CheckFiles(criteria, base)
	for _, item := range items {
		assert.True(t, item.Passed, "expected pass: %s — %s", item.Label, item.Detail)
	}
}

func TestCheckFiles_FileMissing(t *testing.T) {
	base := t.TempDir()
	criteria := domain.DojoCriteria{
		RunDir: "dojo/run",
		FilesCreated: []domain.DojoFileCheck{
			{Path: "todo/geral.md"},
		},
	}
	items := dojo.CheckFiles(criteria, base)
	require.Len(t, items, 1)
	assert.False(t, items[0].Passed)
	assert.Contains(t, items[0].Detail, "not found")
}

func TestCheckFiles_SectionMissing(t *testing.T) {
	base := t.TempDir()
	runDir := filepath.Join(base, "dojo", "run", "todo")
	require.NoError(t, os.MkdirAll(runDir, 0o755))
	require.NoError(t, os.WriteFile(filepath.Join(runDir, "geral.md"), []byte("no section here\n"), 0o644))

	criteria := domain.DojoCriteria{
		RunDir: "dojo/run",
		FilesCreated: []domain.DojoFileCheck{
			{Path: "todo/geral.md", RequiredSections: []string{"ideia:"}},
		},
	}
	items := dojo.CheckFiles(criteria, base)
	require.Len(t, items, 2)
	assert.True(t, items[0].Passed)
	assert.False(t, items[1].Passed)
}

func TestCheckFiles_MustContainFails(t *testing.T) {
	base := t.TempDir()
	runDir := filepath.Join(base, "dojo", "run", "todo")
	require.NoError(t, os.MkdirAll(runDir, 0o755))
	require.NoError(t, os.WriteFile(filepath.Join(runDir, "geral.md"), []byte("ideia: other content\n"), 0o644))

	criteria := domain.DojoCriteria{
		RunDir: "dojo/run",
		FilesCreated: []domain.DojoFileCheck{
			{Path: "todo/geral.md", MustContain: []string{"KATA_RAPIDO"}},
		},
	}
	items := dojo.CheckFiles(criteria, base)
	failed := false
	for _, it := range items {
		if !it.Passed {
			failed = true
			assert.Contains(t, it.Detail, "KATA_RAPIDO")
		}
	}
	assert.True(t, failed)
}

func TestCheckFiles_MustNotContainFails(t *testing.T) {
	base := t.TempDir()
	runDir := filepath.Join(base, "dojo", "run", "todo")
	require.NoError(t, os.MkdirAll(runDir, 0o755))
	require.NoError(t, os.WriteFile(filepath.Join(runDir, "geral.md"), []byte("FORBIDDEN_STRING present\n"), 0o644))

	criteria := domain.DojoCriteria{
		RunDir: "dojo/run",
		FilesCreated: []domain.DojoFileCheck{
			{Path: "todo/geral.md", MustNotContain: []string{"FORBIDDEN_STRING"}},
		},
	}
	items := dojo.CheckFiles(criteria, base)
	failed := false
	for _, it := range items {
		if !it.Passed {
			failed = true
		}
	}
	assert.True(t, failed)
}

func TestCheckFiles_EmptyFilesCreated(t *testing.T) {
	criteria := domain.DojoCriteria{RunDir: "dojo/run"}
	items := dojo.CheckFiles(criteria, t.TempDir())
	assert.Empty(t, items)
}

func TestCheckEmitLog_KeyPresent(t *testing.T) {
	logDir := t.TempDir()
	logPath := filepath.Join(logDir, "emit.log")
	require.NoError(t, os.WriteFile(logPath, []byte("ranger_start\nranger_done\napproval_prompt\n"), 0o644))

	criteria := domain.DojoCriteria{
		EmitLog: domain.DojoEmitLog{
			MustContain:    []string{"ranger_start", "approval_prompt"},
			MustNotContain: []string{"sniper_start"},
		},
	}
	items := dojo.CheckEmitLog(criteria, logPath, false)
	for _, it := range items {
		assert.True(t, it.Passed, "expected pass: %s — %s", it.Label, it.Detail)
	}
}

func TestCheckEmitLog_KeyMissing(t *testing.T) {
	logDir := t.TempDir()
	logPath := filepath.Join(logDir, "emit.log")
	require.NoError(t, os.WriteFile(logPath, []byte("ranger_start\n"), 0o644))

	criteria := domain.DojoCriteria{
		EmitLog: domain.DojoEmitLog{MustContain: []string{"approval_prompt"}},
	}
	items := dojo.CheckEmitLog(criteria, logPath, false)
	require.Len(t, items, 1)
	assert.False(t, items[0].Passed)
}

func TestCheckEmitLog_MustNotContainViolated(t *testing.T) {
	logDir := t.TempDir()
	logPath := filepath.Join(logDir, "emit.log")
	require.NoError(t, os.WriteFile(logPath, []byte("ranger_start\nsniper_start\n"), 0o644))

	criteria := domain.DojoCriteria{
		EmitLog: domain.DojoEmitLog{MustNotContain: []string{"sniper_start"}},
	}
	items := dojo.CheckEmitLog(criteria, logPath, false)
	require.Len(t, items, 1)
	assert.False(t, items[0].Passed)
}

func TestCheckEmitLog_LogMissing_FilesOnly(t *testing.T) {
	criteria := domain.DojoCriteria{
		EmitLog: domain.DojoEmitLog{MustContain: []string{"ranger_start"}},
	}
	items := dojo.CheckEmitLog(criteria, filepath.Join(t.TempDir(), "nonexistent.log"), true)
	assert.Empty(t, items)
}

func TestCheckEmitLog_LogMissing_NotFilesOnly(t *testing.T) {
	criteria := domain.DojoCriteria{
		EmitLog: domain.DojoEmitLog{MustContain: []string{"ranger_start"}},
	}
	items := dojo.CheckEmitLog(criteria, filepath.Join(t.TempDir(), "nonexistent.log"), false)
	require.Len(t, items, 1)
	assert.False(t, items[0].Passed)
	assert.Contains(t, items[0].Detail, "emit.log not found")
}

func TestCheckEmitLog_NoEmitCriteria(t *testing.T) {
	criteria := domain.DojoCriteria{}
	items := dojo.CheckEmitLog(criteria, filepath.Join(t.TempDir(), "emit.log"), false)
	assert.Empty(t, items)
}

func TestCheckEmitLog_ReadError(t *testing.T) {
	dir := t.TempDir()
	logPath := filepath.Join(dir, "emit.log")
	require.NoError(t, os.MkdirAll(logPath, 0o755)) // directory, not file — causes read error

	criteria := domain.DojoCriteria{
		EmitLog: domain.DojoEmitLog{MustContain: []string{"ranger_start"}},
	}
	items := dojo.CheckEmitLog(criteria, logPath, false)
	require.Len(t, items, 1)
	assert.False(t, items[0].Passed)
}

func TestCheckManifests_Pass(t *testing.T) {
	strategistDir := t.TempDir()
	providerDir := filepath.Join(strategistDir, "skills", "brainstorming")
	require.NoError(t, os.MkdirAll(providerDir, 0o755))
	require.NoError(t, os.WriteFile(filepath.Join(providerDir, "skill.yaml"),
		[]byte("canonical_role: ranger\nprovider_class: rankeado\n"), 0o644))

	criteria := domain.DojoCriteria{
		ManifestChecks: []domain.DojoManifestCheck{
			{
				Slot:             "discovery",
				ExpectedProvider: "brainstorming",
				ManifestExists:   true,
				FieldsPresent:    []string{"canonical_role", "provider_class"},
			},
		},
	}
	items := dojo.CheckManifests(criteria, strategistDir)
	for _, it := range items {
		assert.True(t, it.Passed, "expected pass: %s — %s", it.Label, it.Detail)
	}
}

func TestCheckManifests_ManifestMissing(t *testing.T) {
	criteria := domain.DojoCriteria{
		ManifestChecks: []domain.DojoManifestCheck{
			{Slot: "discovery", ExpectedProvider: "brainstorming", ManifestExists: true},
		},
	}
	items := dojo.CheckManifests(criteria, t.TempDir())
	require.Len(t, items, 1)
	assert.False(t, items[0].Passed)
}

func TestCheckManifests_ManifestExpectedAbsent_IsAbsent(t *testing.T) {
	criteria := domain.DojoCriteria{
		ManifestChecks: []domain.DojoManifestCheck{
			{Slot: "execution", ExpectedProvider: "sdd-ask", ManifestExists: false},
		},
	}
	items := dojo.CheckManifests(criteria, t.TempDir())
	require.Len(t, items, 1)
	assert.True(t, items[0].Passed)
}

func TestCheckManifests_FieldMissing(t *testing.T) {
	strategistDir := t.TempDir()
	providerDir := filepath.Join(strategistDir, "skills", "brainstorming")
	require.NoError(t, os.MkdirAll(providerDir, 0o755))
	require.NoError(t, os.WriteFile(filepath.Join(providerDir, "skill.yaml"),
		[]byte("id: brainstorming\n"), 0o644))

	criteria := domain.DojoCriteria{
		ManifestChecks: []domain.DojoManifestCheck{
			{
				Slot:             "discovery",
				ExpectedProvider: "brainstorming",
				ManifestExists:   true,
				FieldsPresent:    []string{"canonical_role"},
			},
		},
	}
	items := dojo.CheckManifests(criteria, strategistDir)
	failed := false
	for _, it := range items {
		if !it.Passed {
			failed = true
		}
	}
	assert.True(t, failed)
}

func TestCheckManifests_Empty(t *testing.T) {
	criteria := domain.DojoCriteria{}
	items := dojo.CheckManifests(criteria, t.TempDir())
	assert.Empty(t, items)
}

func TestRun_AllPass(t *testing.T) {
	base := t.TempDir()
	strategistDir := filepath.Join(base, ".strategist")
	require.NoError(t, os.MkdirAll(strategistDir, 0o755))

	runDir := filepath.Join(base, "dojo", "run", "todo")
	require.NoError(t, os.MkdirAll(runDir, 0o755))
	require.NoError(t, os.WriteFile(filepath.Join(runDir, "geral.md"),
		[]byte("ideia: KATA_RAPIDO\n"), 0o644))

	logDir := filepath.Join(base, "dojo", ".last-run", "quick-draw")
	require.NoError(t, os.MkdirAll(logDir, 0o755))
	require.NoError(t, os.WriteFile(filepath.Join(logDir, "emit.log"),
		[]byte("ranger_start\nranger_done\n"), 0o644))

	criteria := domain.DojoCriteria{
		Scenario: "quick-draw",
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

func TestCheckTiming_Pass(t *testing.T) {
	logDir := t.TempDir()
	logPath := filepath.Join(logDir, "emit.log")
	require.NoError(t, os.WriteFile(logPath,
		[]byte("ranger_start\ntotal_wall_time_ms=1200\napproval_prompt\n"), 0o644))

	criteria := domain.DojoCriteria{
		TimingCriteria: &domain.DojoTimingCriteria{MaxWallTimeMs: 30000},
	}
	items := dojo.CheckTiming(criteria, logPath)
	require.Len(t, items, 1)
	assert.True(t, items[0].Passed, "expected pass: %s", items[0].Detail)
}

func TestCheckTiming_Fail_ExceedsMax(t *testing.T) {
	logDir := t.TempDir()
	logPath := filepath.Join(logDir, "emit.log")
	require.NoError(t, os.WriteFile(logPath,
		[]byte("total_wall_time_ms=45000\n"), 0o644))

	criteria := domain.DojoCriteria{
		TimingCriteria: &domain.DojoTimingCriteria{MaxWallTimeMs: 30000},
	}
	items := dojo.CheckTiming(criteria, logPath)
	require.Len(t, items, 1)
	assert.False(t, items[0].Passed)
	assert.Contains(t, items[0].Detail, "45000")
}

func TestCheckTiming_NilCriteria_Skip(t *testing.T) {
	criteria := domain.DojoCriteria{}
	items := dojo.CheckTiming(criteria, t.TempDir()+"/emit.log")
	assert.Empty(t, items)
}

func TestCheckTiming_LogMissing(t *testing.T) {
	criteria := domain.DojoCriteria{
		TimingCriteria: &domain.DojoTimingCriteria{MaxWallTimeMs: 30000},
	}
	items := dojo.CheckTiming(criteria, filepath.Join(t.TempDir(), "nonexistent.log"))
	require.Len(t, items, 1)
	assert.False(t, items[0].Passed)
	assert.Contains(t, items[0].Detail, "emit.log not found")
}

func TestCheckTiming_FieldMissing(t *testing.T) {
	logDir := t.TempDir()
	logPath := filepath.Join(logDir, "emit.log")
	require.NoError(t, os.WriteFile(logPath, []byte("ranger_start\nranger_done\n"), 0o644))

	criteria := domain.DojoCriteria{
		TimingCriteria: &domain.DojoTimingCriteria{MaxWallTimeMs: 30000},
	}
	items := dojo.CheckTiming(criteria, logPath)
	require.Len(t, items, 1)
	assert.False(t, items[0].Passed)
	assert.Contains(t, items[0].Detail, "total_wall_time_ms not found")
}
