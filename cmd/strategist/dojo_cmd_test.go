package main

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/SergioLacerda/strategist-skill/internal/domain"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestIsDojoScenarioEntry(t *testing.T) {
	dojoDir := t.TempDir()
	require.NoError(t, os.WriteFile(filepath.Join(dojoDir, "not-a-dir.txt"), []byte("x"), 0o644))
	require.NoError(t, os.MkdirAll(filepath.Join(dojoDir, ".last-run"), 0o755))
	require.NoError(t, os.MkdirAll(filepath.Join(dojoDir, "valid-scenario"), 0o755))
	require.NoError(t, os.WriteFile(filepath.Join(dojoDir, "valid-scenario", "criteria.yaml"), []byte("scenario: valid-scenario\n"), 0o644))

	entries, err := os.ReadDir(dojoDir)
	require.NoError(t, err)

	got := map[string]bool{}
	for _, e := range entries {
		got[e.Name()] = isDojoScenarioEntry(dojoDir, e)
	}
	assert.False(t, got["not-a-dir.txt"], "a regular file must never be a scenario entry")
	assert.False(t, got[".last-run"], "the run output directory must never be a scenario entry")
	assert.True(t, got["valid-scenario"], "a directory with criteria.yaml must be a scenario entry")
}

func TestDojoDescription_MissingCriteriaFile(t *testing.T) {
	dojoDir := t.TempDir()
	require.NoError(t, os.MkdirAll(filepath.Join(dojoDir, "empty-scenario"), 0o755))

	assert.Empty(t, dojoDescription(dojoDir, "empty-scenario"))
}

func TestDojoDescription_InvalidYAML(t *testing.T) {
	dojoDir := t.TempDir()
	scenarioDir := filepath.Join(dojoDir, "broken-scenario")
	require.NoError(t, os.MkdirAll(scenarioDir, 0o755))
	require.NoError(t, os.WriteFile(filepath.Join(scenarioDir, "criteria.yaml"), []byte("scenario: [unterminated\n"), 0o644))

	assert.Empty(t, dojoDescription(dojoDir, "broken-scenario"))
}

func TestDojoCheckCmd_AllPass(t *testing.T) {
	root := setupDojoScenario(t, "sample-scenario",
		"scenario: sample-scenario\nrun_dir: dojo/run\nfiles_created:\n  - path: todo/geral.md\n    must_contain: [KATA_RAPIDO]\n",
		"ideia: KATA_RAPIDO test\n",
	)

	orig := dojoRoot
	t.Cleanup(func() { dojoRoot = orig })
	dojoRoot = root

	out := captureStdout(t, func() {
		err := dojoCheckCmd.RunE(dojoCheckCmd, []string{"sample-scenario"})
		require.NoError(t, err)
	})
	assert.Contains(t, out, "PASS")
	assert.Contains(t, out, "KATA_RAPIDO")
}

func TestDojoCheckCmd_FileMissing(t *testing.T) {
	root := setupDojoScenario(t, "sample-scenario",
		"scenario: sample-scenario\nrun_dir: dojo/run\nfiles_created:\n  - path: todo/geral.md\n",
		"",
	)

	orig := dojoRoot
	t.Cleanup(func() { dojoRoot = orig })
	dojoRoot = root

	out := captureStdout(t, func() {
		err := dojoCheckCmd.RunE(dojoCheckCmd, []string{"sample-scenario"})
		require.Error(t, err)
		assert.Contains(t, err.Error(), "failed")
	})
	assert.Contains(t, out, "FAIL")
}

func TestDojoCheckCmd_MissingActiveYAML(t *testing.T) {
	orig := dojoRoot
	t.Cleanup(func() { dojoRoot = orig })
	dojoRoot = filepath.Join(t.TempDir(), "nonexistent")

	err := dojoCheckCmd.RunE(dojoCheckCmd, []string{"sample-scenario"})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "active.yaml")
}

func TestDojoCheckCmd_MissingCriteria(t *testing.T) {
	root := setupDojoScenario(t, "sample-scenario", "scenario: sample-scenario\n", "")
	orig := dojoRoot
	t.Cleanup(func() { dojoRoot = orig })
	dojoRoot = root

	err := dojoCheckCmd.RunE(dojoCheckCmd, []string{"nonexistent-scenario"})
	require.Error(t, err)
}

func TestDojoCheckCmd_FilesOnlySkipsEmitLog(t *testing.T) {
	root := setupDojoScenario(t, "sample-scenario",
		"scenario: sample-scenario\nrun_dir: dojo/run\nfiles_created:\n  - path: todo/geral.md\nemit_log:\n  must_contain: [ranger_start]\n",
		"ideia: content\n",
	)

	orig := dojoRoot
	origFilesOnly := dojoFilesOnly
	t.Cleanup(func() { dojoRoot = orig; dojoFilesOnly = origFilesOnly })
	dojoRoot = root
	dojoFilesOnly = true

	out := captureStdout(t, func() {
		err := dojoCheckCmd.RunE(dojoCheckCmd, []string{"sample-scenario"})
		require.NoError(t, err)
	})
	assert.Contains(t, out, "PASS")
	assert.NotContains(t, out, "emit.log not found")
}

func TestDojoCheckCmd_PersistsResultAndLesson(t *testing.T) {
	root := setupDojoScenario(t, "sample-scenario",
		"scenario: sample-scenario\nrun_dir: dojo/run\nfiles_created:\n  - path: todo/geral.md\n",
		"",
	)

	orig := dojoRoot
	t.Cleanup(func() { dojoRoot = orig })
	dojoRoot = root

	_ = captureStdout(t, func() {
		err := dojoCheckCmd.RunE(dojoCheckCmd, []string{"sample-scenario"})
		require.Error(t, err)
	})

	basePath := filepath.Join(filepath.Dir(root), ".analysis")
	_, err := os.Stat(filepath.Join(basePath, "dojo", ".last-run", "sample-scenario", "result.json"))
	require.NoError(t, err, "expected result.json to be persisted")
	_, err = os.Stat(filepath.Join(basePath, "dojo", ".history.jsonl"))
	require.NoError(t, err, "expected .history.jsonl to be appended")
	_, err = os.Stat(filepath.Join(basePath, "dojo", ".last-run", "sample-scenario", "lesson.md"))
	assert.NoError(t, err, "expected lesson.md to be written for a failing run")
}

func TestDojoCheckCmd_EmptyBasePath(t *testing.T) {
	dir := t.TempDir()
	strategistRoot := filepath.Join(dir, ".strategist")
	require.NoError(t, os.MkdirAll(strategistRoot, 0o755))
	require.NoError(t, os.WriteFile(filepath.Join(strategistRoot, "active.yaml"),
		[]byte("mode: epic\nslots:\n  discovery: brainstorming\n"), 0o644))

	orig := dojoRoot
	t.Cleanup(func() { dojoRoot = orig })
	dojoRoot = strategistRoot

	err := dojoCheckCmd.RunE(dojoCheckCmd, []string{"sample-scenario"})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "base_path")
}

func TestDojoListCmd_ListsScenarios(t *testing.T) {
	root := setupDojoScenario(t, "sample-scenario",
		"scenario: sample-scenario\ndescription: \"sample scenario test\"\n", "")

	// root is .strategist; project root is its parent; dojo dir is <project>/.analysis/dojo
	projectRoot := filepath.Dir(root)
	s2 := filepath.Join(projectRoot, ".analysis", "dojo", "ranger-weapons")
	require.NoError(t, os.MkdirAll(s2, 0o755))
	require.NoError(t, os.WriteFile(filepath.Join(s2, "criteria.yaml"),
		[]byte("scenario: ranger-weapons\ndescription: \"Ranger weapons test\"\n"), 0o644))

	orig := dojoRoot
	t.Cleanup(func() { dojoRoot = orig })
	dojoRoot = root

	out := captureStdout(t, func() {
		err := dojoListCmd.RunE(dojoListCmd, nil)
		require.NoError(t, err)
	})
	assert.Contains(t, out, "sample-scenario")
}

func TestDojoListCmd_ExcludesRunDirectory(t *testing.T) {
	root := setupDojoScenario(t, "sample-scenario",
		"scenario: sample-scenario\ndescription: \"sample scenario test\"\n", "ideia: content\n")

	orig := dojoRoot
	t.Cleanup(func() { dojoRoot = orig })
	dojoRoot = root

	out := captureStdout(t, func() {
		err := dojoListCmd.RunE(dojoListCmd, nil)
		require.NoError(t, err)
	})
	assert.Contains(t, out, "sample-scenario")
	for _, line := range strings.Split(out, "\n") {
		assert.False(t, strings.HasPrefix(line, "run\t") || line == "run",
			"dojo list must not list the run output directory: %q", line)
	}
}

func TestDojoListCmd_EmptyDojo(t *testing.T) {
	dir := t.TempDir()
	strategistRoot := filepath.Join(dir, ".strategist")
	require.NoError(t, os.MkdirAll(strategistRoot, 0o755))
	require.NoError(t, os.WriteFile(filepath.Join(strategistRoot, "active.yaml"),
		[]byte("mode: epic\nbase_path: .analysis\nslots:\n  discovery: brainstorming\n"), 0o644))
	// base_path resolves to <dir>/.analysis (parent of strategistRoot is dir)
	require.NoError(t, os.MkdirAll(filepath.Join(dir, ".analysis", "dojo"), 0o755))

	orig := dojoRoot
	t.Cleanup(func() { dojoRoot = orig })
	dojoRoot = strategistRoot

	out := captureStdout(t, func() {
		err := dojoListCmd.RunE(dojoListCmd, nil)
		require.NoError(t, err)
	})
	assert.Empty(t, out)
}

func TestDojoListCmd_MissingDojoDir(t *testing.T) {
	dir := t.TempDir()
	strategistRoot := filepath.Join(dir, ".strategist")
	require.NoError(t, os.MkdirAll(strategistRoot, 0o755))
	require.NoError(t, os.WriteFile(filepath.Join(strategistRoot, "active.yaml"),
		[]byte("mode: epic\nbase_path: .analysis\nslots:\n  discovery: brainstorming\n"), 0o644))

	orig := dojoRoot
	t.Cleanup(func() { dojoRoot = orig })
	dojoRoot = strategistRoot

	err := dojoListCmd.RunE(dojoListCmd, nil)
	require.Error(t, err)
}

func TestDojoListCmd_MissingActiveYAML(t *testing.T) {
	orig := dojoRoot
	t.Cleanup(func() { dojoRoot = orig })
	dojoRoot = filepath.Join(t.TempDir(), "nonexistent")

	err := dojoListCmd.RunE(dojoListCmd, nil)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "active.yaml")
}

// --- check ---

// --- dojoItemLine ---

func TestDojoItemLine_Passed(t *testing.T) {
	t.Parallel()
	item := domain.DojoCheckItem{Label: "file-exists", Passed: true}
	line := dojoItemLine(item)
	assert.Contains(t, line, "✓")
	assert.Contains(t, line, "file-exists")
}

func TestDojoItemLine_FailedWithDetail(t *testing.T) {
	t.Parallel()
	item := domain.DojoCheckItem{Label: "file-exists", Passed: false, Detail: "missing"}
	line := dojoItemLine(item)
	assert.Contains(t, line, "✗")
	assert.Contains(t, line, "missing")
}

func TestDojoItemLine_FailedWithoutDetail(t *testing.T) {
	t.Parallel()
	item := domain.DojoCheckItem{Label: "file-exists", Passed: false}
	line := dojoItemLine(item)
	assert.Contains(t, line, "FAIL")
}
