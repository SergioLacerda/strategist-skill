package dojo

import (
	"bytes"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/SergioLacerda/strategist-skill/internal/cliutil"
	"github.com/SergioLacerda/strategist-skill/internal/domain"
	"github.com/spf13/cobra"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func testDeps() Dependencies {
	return Dependencies{ResolveRoots: func(root string) (string, string, error) {
		strategistRoot, basePath, err := cliutil.ResolveActiveBasePath(root)
		if err != nil {
			return "", "", fmt.Errorf("dojo: %w", err)
		}
		return strategistRoot, basePath, nil
	}}
}

// execute runs the real dojo command tree with args and returns its stdout.
func execute(t *testing.T, args ...string) (string, error) {
	t.Helper()
	cmd := New(testDeps())
	var out bytes.Buffer
	cmd.SetOut(&out)
	cmd.SetErr(&bytes.Buffer{})
	cmd.SetArgs(args)
	err := cmd.Execute()
	return out.String(), err
}

func setupDojoScenario(t *testing.T, scenario, criteria, runContent string) string {
	t.Helper()
	dir := t.TempDir()

	strategistRoot := filepath.Join(dir, ".strategist")
	require.NoError(t, os.MkdirAll(strategistRoot, 0o755))
	require.NoError(t, os.WriteFile(filepath.Join(strategistRoot, "active.yaml"),
		[]byte("mode: epic\nbase_path: .analysis\nslots:\n  discovery: brainstorming\n"), 0o644))

	scenarioDir := filepath.Join(dir, ".analysis", "dojo", scenario)
	require.NoError(t, os.MkdirAll(scenarioDir, 0o755))
	require.NoError(t, os.WriteFile(filepath.Join(scenarioDir, "criteria.yaml"), []byte(criteria), 0o644))

	if runContent != "" {
		runDir := filepath.Join(dir, ".analysis", "dojo", "run", "todo")
		require.NoError(t, os.MkdirAll(runDir, 0o755))
		require.NoError(t, os.WriteFile(filepath.Join(runDir, "geral.md"), []byte(runContent), 0o644))
	}
	return strategistRoot
}

func withClosedStdout(t *testing.T, fn func()) {
	t.Helper()
	f, err := os.CreateTemp(t.TempDir(), "closed-stdout-*")
	require.NoError(t, err)
	require.NoError(t, f.Close())
	old := os.Stdout
	os.Stdout = f
	t.Cleanup(func() { os.Stdout = old })
	fn()
}

func TestNew_KeepsCommandContract(t *testing.T) {
	cmd := New(Dependencies{})

	assert.Equal(t, "dojo", cmd.Use)
	check, _, err := cmd.Find([]string{"check"})
	require.NoError(t, err)
	assert.Equal(t, "check <scenario>", check.Use)
	assert.NotNil(t, check.Flags().Lookup("root"))
	assert.NotNil(t, check.Flags().Lookup("files-only"))
	list, _, err := cmd.Find([]string{"list"})
	require.NoError(t, err)
	assert.NotNil(t, list.Flags().Lookup("root"))
}

func TestRegister_AttachesDojo(t *testing.T) {
	root := &cobra.Command{Use: "strategist"}

	Register(root, Dependencies{})

	found, _, err := root.Find([]string{"dojo", "list"})
	require.NoError(t, err)
	assert.Equal(t, "list", found.Name())
}

func TestCommands_FailClosedWithoutResolver(t *testing.T) {
	for _, args := range [][]string{{"list"}, {"check", "sample-scenario"}} {
		cmd := New(Dependencies{})
		cmd.SetOut(&bytes.Buffer{})
		cmd.SetErr(&bytes.Buffer{})
		cmd.SetArgs(args)

		err := cmd.Execute()

		require.Error(t, err, args[0])
		assert.Contains(t, err.Error(), "root resolver is not configured", args[0])
	}
}

func TestIsScenarioEntry(t *testing.T) {
	dojoDir := t.TempDir()
	require.NoError(t, os.WriteFile(filepath.Join(dojoDir, "not-a-dir.txt"), []byte("x"), 0o644))
	require.NoError(t, os.MkdirAll(filepath.Join(dojoDir, ".last-run"), 0o755))
	require.NoError(t, os.MkdirAll(filepath.Join(dojoDir, "valid-scenario"), 0o755))
	require.NoError(t, os.WriteFile(filepath.Join(dojoDir, "valid-scenario", "criteria.yaml"), []byte("scenario: valid-scenario\n"), 0o644))

	entries, err := os.ReadDir(dojoDir)
	require.NoError(t, err)

	got := map[string]bool{}
	for _, e := range entries {
		got[e.Name()] = isScenarioEntry(dojoDir, e)
	}
	assert.False(t, got["not-a-dir.txt"], "a regular file must never be a scenario entry")
	assert.False(t, got[".last-run"], "the run output directory must never be a scenario entry")
	assert.True(t, got["valid-scenario"], "a directory with criteria.yaml must be a scenario entry")
}

func TestCheckCmd_AllPass(t *testing.T) {
	root := setupDojoScenario(t, "sample-scenario",
		"scenario: sample-scenario\nrun_dir: dojo/run\nfiles_created:\n  - path: todo/geral.md\n    must_contain: [KATA_RAPIDO]\n",
		"ideia: KATA_RAPIDO test\n",
	)

	out, err := execute(t, "check", "--root", root, "sample-scenario")

	require.NoError(t, err)
	assert.Contains(t, out, "PASS")
	assert.Contains(t, out, "KATA_RAPIDO")
}

func TestCheckCmd_FileMissing(t *testing.T) {
	root := setupDojoScenario(t, "sample-scenario",
		"scenario: sample-scenario\nrun_dir: dojo/run\nfiles_created:\n  - path: todo/geral.md\n",
		"",
	)

	out, err := execute(t, "check", "--root", root, "sample-scenario")

	require.Error(t, err)
	assert.Contains(t, err.Error(), "failed")
	assert.Contains(t, out, "FAIL")
}

func TestCheckCmd_MissingActiveYAML(t *testing.T) {
	_, err := execute(t, "check", "--root", filepath.Join(t.TempDir(), "nonexistent"), "sample-scenario")

	require.Error(t, err)
	assert.Contains(t, err.Error(), "active.yaml")
}

func TestCheckCmd_MissingCriteria(t *testing.T) {
	root := setupDojoScenario(t, "sample-scenario", "scenario: sample-scenario\n", "")

	_, err := execute(t, "check", "--root", root, "nonexistent-scenario")

	require.Error(t, err)
}

func TestCheckCmd_FilesOnlySkipsEmitLog(t *testing.T) {
	root := setupDojoScenario(t, "sample-scenario",
		"scenario: sample-scenario\nrun_dir: dojo/run\nfiles_created:\n  - path: todo/geral.md\nemit_log:\n  must_contain: [ranger_start]\n",
		"ideia: content\n",
	)

	out, err := execute(t, "check", "--root", root, "--files-only", "sample-scenario")

	require.NoError(t, err)
	assert.Contains(t, out, "PASS")
	assert.NotContains(t, out, "emit.log not found")
}

func TestCheckCmd_FlagsDoNotLeakBetweenInvocations(t *testing.T) {
	root := setupDojoScenario(t, "sample-scenario",
		"scenario: sample-scenario\nrun_dir: dojo/run\nfiles_created:\n  - path: todo/geral.md\nemit_log:\n  must_contain: [ranger_start]\n",
		"ideia: content\n",
	)

	_, err := execute(t, "check", "--root", root, "--files-only", "sample-scenario")
	require.NoError(t, err)

	// A fresh command must not inherit --files-only from the previous one.
	_, err = execute(t, "check", "--root", root, "sample-scenario")
	require.Error(t, err, "emit_log is required again once --files-only is not passed")
}

func TestCheckCmd_PersistsResultAndLesson(t *testing.T) {
	root := setupDojoScenario(t, "sample-scenario",
		"scenario: sample-scenario\nrun_dir: dojo/run\nfiles_created:\n  - path: todo/geral.md\n",
		"",
	)

	_, err := execute(t, "check", "--root", root, "sample-scenario")
	require.Error(t, err)

	basePath := filepath.Join(filepath.Dir(root), ".analysis")
	_, err = os.Stat(filepath.Join(basePath, "dojo", ".last-run", "sample-scenario", "result.json"))
	require.NoError(t, err, "expected result.json to be persisted")
	_, err = os.Stat(filepath.Join(basePath, "dojo", ".history.jsonl"))
	require.NoError(t, err, "expected .history.jsonl to be appended")
	_, err = os.Stat(filepath.Join(basePath, "dojo", ".last-run", "sample-scenario", "lesson.md"))
	assert.NoError(t, err, "expected lesson.md to be written for a failing run")
}

func TestCheckCmd_EmptyBasePath(t *testing.T) {
	dir := t.TempDir()
	strategistRoot := filepath.Join(dir, ".strategist")
	require.NoError(t, os.MkdirAll(strategistRoot, 0o755))
	require.NoError(t, os.WriteFile(filepath.Join(strategistRoot, "active.yaml"),
		[]byte("mode: epic\nslots:\n  discovery: brainstorming\n"), 0o644))

	_, err := execute(t, "check", "--root", strategistRoot, "sample-scenario")

	require.Error(t, err)
	assert.Contains(t, err.Error(), "base_path")
}

func TestListCmd_ListsScenarios(t *testing.T) {
	root := setupDojoScenario(t, "sample-scenario",
		"scenario: sample-scenario\ndescription: \"sample scenario test\"\n", "")

	// root is .strategist; project root is its parent; dojo dir is <project>/.analysis/dojo
	s2 := filepath.Join(filepath.Dir(root), ".analysis", "dojo", "ranger-weapons")
	require.NoError(t, os.MkdirAll(s2, 0o755))
	require.NoError(t, os.WriteFile(filepath.Join(s2, "criteria.yaml"),
		[]byte("scenario: ranger-weapons\ndescription: \"Ranger weapons test\"\n"), 0o644))

	out, err := execute(t, "list", "--root", root)

	require.NoError(t, err)
	assert.Contains(t, out, "sample-scenario")
	assert.Contains(t, out, "sample scenario test")
	assert.Contains(t, out, "ranger-weapons")
}

func TestListCmd_ExcludesRunDirectory(t *testing.T) {
	root := setupDojoScenario(t, "sample-scenario",
		"scenario: sample-scenario\ndescription: \"sample scenario test\"\n", "ideia: content\n")

	out, err := execute(t, "list", "--root", root)

	require.NoError(t, err)
	assert.Contains(t, out, "sample-scenario")
	for _, line := range strings.Split(out, "\n") {
		assert.False(t, strings.HasPrefix(line, "run\t") || line == "run",
			"dojo list must not list the run output directory: %q", line)
	}
}

func TestListCmd_EmptyDojo(t *testing.T) {
	dir := t.TempDir()
	strategistRoot := filepath.Join(dir, ".strategist")
	require.NoError(t, os.MkdirAll(strategistRoot, 0o755))
	require.NoError(t, os.WriteFile(filepath.Join(strategistRoot, "active.yaml"),
		[]byte("mode: epic\nbase_path: .analysis\nslots:\n  discovery: brainstorming\n"), 0o644))
	require.NoError(t, os.MkdirAll(filepath.Join(dir, ".analysis", "dojo"), 0o755))

	out, err := execute(t, "list", "--root", strategistRoot)

	require.NoError(t, err)
	assert.Empty(t, out)
}

func TestListCmd_MissingDojoDir(t *testing.T) {
	dir := t.TempDir()
	strategistRoot := filepath.Join(dir, ".strategist")
	require.NoError(t, os.MkdirAll(strategistRoot, 0o755))
	require.NoError(t, os.WriteFile(filepath.Join(strategistRoot, "active.yaml"),
		[]byte("mode: epic\nbase_path: .analysis\nslots:\n  discovery: brainstorming\n"), 0o644))

	_, err := execute(t, "list", "--root", strategistRoot)

	require.Error(t, err)
}

func TestListCmd_MissingActiveYAML(t *testing.T) {
	_, err := execute(t, "list", "--root", filepath.Join(t.TempDir(), "nonexistent"))

	require.Error(t, err)
	assert.Contains(t, err.Error(), "active.yaml")
}

func TestItemLine_Passed(t *testing.T) {
	t.Parallel()
	line := itemLine(domain.DojoCheckItem{Label: "file-exists", Passed: true})
	assert.Contains(t, line, "✓")
	assert.Contains(t, line, "file-exists")
}

func TestItemLine_FailedWithDetail(t *testing.T) {
	t.Parallel()
	line := itemLine(domain.DojoCheckItem{Label: "file-exists", Passed: false, Detail: "missing"})
	assert.Contains(t, line, "✗")
	assert.Contains(t, line, "missing")
}

func TestItemLine_FailedWithoutDetail(t *testing.T) {
	t.Parallel()
	line := itemLine(domain.DojoCheckItem{Label: "file-exists", Passed: false})
	assert.Contains(t, line, "FAIL")
}

// TestCheckCmd_PrintResultErrorOnClosedStdout covers runCheck's own
// "if err := printResult(...); err != nil { return err }" branch — with the
// default writer (os.Stdout) closed, the first Fprint fails and runCheck must
// propagate it unwrapped.
func TestCheckCmd_PrintResultErrorOnClosedStdout(t *testing.T) {
	root := setupDojoScenario(t, "sample-scenario",
		"scenario: sample-scenario\nrun_dir: dojo/run\nfiles_created:\n  - path: todo/geral.md\n    must_contain: [KATA_RAPIDO]\n",
		"ideia: KATA_RAPIDO test\n",
	)

	withClosedStdout(t, func() {
		cmd := New(testDeps())
		cmd.SetErr(&bytes.Buffer{})
		cmd.SetArgs([]string{"check", "--root", root, "sample-scenario"})
		err := cmd.Execute()
		require.Error(t, err)
		assert.Contains(t, err.Error(), "dojo: write result")
	})
}

// TestListCmd_FlushErrorOnClosedStdout covers runList's own Flush error branch.
// tabwriter buffers single-line Fprintf calls in memory, so the closed-stdout
// failure only surfaces at Flush().
func TestListCmd_FlushErrorOnClosedStdout(t *testing.T) {
	root := setupDojoScenario(t, "sample-scenario",
		"scenario: sample-scenario\ndescription: \"sample scenario test\"\n", "")

	withClosedStdout(t, func() {
		cmd := New(testDeps())
		cmd.SetErr(&bytes.Buffer{})
		cmd.SetArgs([]string{"list", "--root", root})
		err := cmd.Execute()
		require.Error(t, err)
		assert.Contains(t, err.Error(), "dojo list: flush")
	})
}

// TestPersistResult_WarnsOnPersistAndLessonErrors covers persistResult's two
// warning branches. Pre-occupying basePath/dojo with a file blocks both
// MkdirAll calls at once; a failed result is required so WriteLesson actually
// attempts its write.
func TestPersistResult_WarnsOnPersistAndLessonErrors(t *testing.T) {
	dir := t.TempDir()
	require.NoError(t, os.WriteFile(filepath.Join(dir, "dojo"), []byte("x"), 0o644))
	result := domain.DojoCheckResult{Scenario: "sample", Items: []domain.DojoCheckItem{{Passed: false}}}

	var errOut bytes.Buffer
	persistResult(&errOut, dir, result, time.Now(), time.Now())

	assert.Contains(t, errOut.String(), "failed to persist result")
	assert.Contains(t, errOut.String(), "failed to write lesson")
}

// TestPersistResult_PassedResultSkipsLesson covers the WriteLesson early-return
// branch (result.Passed()==true) separately from the persist-error branch.
func TestPersistResult_PassedResultSkipsLesson(t *testing.T) {
	dir := t.TempDir()
	require.NoError(t, os.WriteFile(filepath.Join(dir, "dojo"), []byte("x"), 0o644))
	result := domain.DojoCheckResult{Scenario: "sample", Items: []domain.DojoCheckItem{{Passed: true}}}

	var errOut bytes.Buffer
	persistResult(&errOut, dir, result, time.Now(), time.Now())

	assert.Contains(t, errOut.String(), "failed to persist result")
	assert.NotContains(t, errOut.String(), "failed to write lesson")
}

// TestPrintResult_ClosedWriterErrors covers printResult's terminal Flush error
// check with a writer that always fails.
func TestPrintResult_ClosedWriterErrors(t *testing.T) {
	result := domain.DojoCheckResult{
		Scenario: "sample",
		Items:    []domain.DojoCheckItem{{Passed: true, Label: "check-1"}},
	}

	require.Error(t, printResult(failingWriter{}, result))
}

type failingWriter struct{}

func (failingWriter) Write([]byte) (int, error) { return 0, fmt.Errorf("write failed") }
