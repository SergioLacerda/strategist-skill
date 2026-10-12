package treasurecli

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func fixtureWithID(id, doc string) string {
	return strings.NewReplacer("{ID}", id, "{DOC}", doc).Replace(`schema_version: "1"
runbook_id: {ID}
runbook_type: analytical
source_doc: docs/runbooks/{DOC}.md
applies_when:
  - timeout
objective: fixture
`)
}

// runSelect runs the command and returns stdout, stderr and the error.
func runSelect(t *testing.T, root string, signals ...string) (string, string, error) {
	t.Helper()
	resetRunbookSelectFlags(t)
	setRunbookSignals(t, signals...)
	setRunbookRoot(t, root)
	setCmdFlag(t, runbookSelectCmd, flagFormat, outputFormatJSON)
	attachMissionRun(t, runbookSelectCmd)
	var runErr error
	var stdout string
	stderr := captureStderr(t, func() {
		stdout = captureStdout(t, func() {
			runErr = runbookSelectCmd.RunE(runbookSelectCmd, nil)
		})
	})
	return stdout, stderr, runErr
}

func removeRunbookDoc(t *testing.T, root, stem string) {
	t.Helper()
	require.NoError(t, os.Remove(filepath.Join(filepath.Dir(root), "docs", "runbooks", stem+".md")))
}

func TestRunbookSelect_InvalidSidecarIsSkippedAndReported(t *testing.T) {
	root := runbookSelectTestRoot(t, map[string]string{
		"broken-runbook": fixtureRunbookInvalidLevel,
		"fix-timeout":    fixtureRunbookA,
	})
	stdout, stderr, err := runSelect(t, root, "timeout")
	require.NoError(t, err)
	assert.Contains(t, stdout, `"runbook_id": "fix-timeout"`)
	assert.NotContains(t, stdout, "broken-runbook")
	assert.Contains(t, stderr, "[Strategist] runbook select: skipped docs/runbooks/broken-runbook.runbook.yaml code=sidecar_invalid")
	assert.Contains(t, stderr, "check_invalid")
	assert.Contains(t, stderr, "corpus loaded=1 skipped=1 warnings=0")
}

func TestRunbookSelect_UnreadableSidecarIsSkipped(t *testing.T) {
	if os.Getuid() == 0 {
		t.Skip("file modes are not enforced for root")
	}
	root := runbookSelectTestRoot(t, map[string]string{"fix-timeout": fixtureRunbookA, "fix-timeout-secondary": fixtureRunbookB})
	path := filepath.Join(filepath.Dir(root), "docs", "runbooks", "fix-timeout-secondary.runbook.yaml")
	require.NoError(t, os.Chmod(path, 0o000))
	t.Cleanup(func() { _ = os.Chmod(path, 0o600) }) //nolint:errcheck // best-effort restore of a temp file
	if _, err := os.ReadFile(path); err == nil {
		t.Skip("file modes are not enforced on this platform")
	}
	stdout, stderr, err := runSelect(t, root, "timeout")
	require.NoError(t, err)
	assert.Contains(t, stdout, `"runbook_id": "fix-timeout"`)
	assert.Contains(t, stderr, "code=read_failed")
}

func TestRunbookSelect_DuplicateIDKeepsTheStemMatch(t *testing.T) {
	root := runbookSelectTestRoot(t, map[string]string{
		"dup-id": fixtureWithID("dup-id", "dup-id"),
		"copy":   fixtureWithID("dup-id", "copy"),
	})
	stdout, stderr, err := runSelect(t, root, "timeout")
	require.NoError(t, err)
	assert.Contains(t, stdout, `"ref": "docs/runbooks/dup-id.md"`)
	assert.NotContains(t, stdout, "copy.md")
	assert.Contains(t, stderr, "skipped docs/runbooks/copy.runbook.yaml code=duplicate_runbook_id")
	assert.Contains(t, stderr, "docs/runbooks/dup-id.runbook.yaml")
}

func TestRunbookSelect_DuplicateIDWithoutStemMatchSkipsAll(t *testing.T) {
	root := runbookSelectTestRoot(t, map[string]string{
		"one": fixtureWithID("dup-id", "one"),
		"two": fixtureWithID("dup-id", "two"),
	})
	stdout, stderr, err := runSelect(t, root, "timeout")
	require.NoError(t, err)
	assert.NotContains(t, stdout, "dup-id")
	assert.Contains(t, stderr, "skipped docs/runbooks/one.runbook.yaml code=duplicate_runbook_id")
	assert.Contains(t, stderr, "skipped docs/runbooks/two.runbook.yaml code=duplicate_runbook_id")
}

func TestRunbookSelect_DanglingSourceDocIsSkipped(t *testing.T) {
	root := runbookSelectTestRoot(t, map[string]string{"fix-timeout": fixtureRunbookA, "fix-timeout-secondary": fixtureRunbookB})
	removeRunbookDoc(t, root, "fix-timeout-secondary")
	stdout, stderr, err := runSelect(t, root, "timeout")
	require.NoError(t, err)
	assert.NotContains(t, stdout, "fix-timeout-secondary")
	assert.Contains(t, stderr, "code=source_doc_missing")
}

func TestRunbookSelect_SourceDocNotMarkdownWarnsAndStaysSelectable(t *testing.T) {
	selfRef := strings.Replace(fixtureRunbookA, "source_doc: docs/runbooks/fix-timeout.md", "source_doc: docs/runbooks/fix-timeout.runbook.yaml", 1)
	root := runbookSelectTestRoot(t, map[string]string{"fix-timeout": selfRef})
	stdout, stderr, err := runSelect(t, root, "timeout")
	require.NoError(t, err)
	assert.Contains(t, stdout, `"runbook_id": "fix-timeout"`)
	assert.Contains(t, stderr, "warning docs/runbooks/fix-timeout.runbook.yaml code=source_doc_not_markdown")
	assert.Contains(t, stderr, "corpus loaded=1 skipped=0 warnings=1")
}

func TestRunbookSelect_StemMismatchWarns(t *testing.T) {
	root := runbookSelectTestRoot(t, map[string]string{"renamed": fixtureRunbookA})
	removeRunbookDoc(t, root, "renamed")
	require.NoError(t, os.WriteFile(filepath.Join(filepath.Dir(root), "docs", "runbooks", "fix-timeout.md"), []byte("# d\n"), 0o600))
	stdout, stderr, err := runSelect(t, root, "timeout")
	require.NoError(t, err)
	assert.Contains(t, stdout, `"runbook_id": "fix-timeout"`)
	assert.Contains(t, stderr, "code=runbook_id_stem_mismatch")
}

func TestRunbookSelect_ValidCorpusProducesNoDiagnostics(t *testing.T) {
	root := runbookSelectTestRoot(t, map[string]string{"fix-timeout": fixtureRunbookA})
	_, stderr, err := runSelect(t, root, "timeout")
	require.NoError(t, err)
	assert.NotContains(t, stderr, "skipped")
	assert.NotContains(t, stderr, "corpus loaded=")
}

func TestRunbookSelect_AllSidecarsSkippedSaysSoInsteadOfNoSidecars(t *testing.T) {
	root := runbookSelectTestRoot(t, map[string]string{"broken-runbook": fixtureRunbookInvalidLevel})
	resetRunbookSelectFlags(t)
	setRunbookSignals(t, "timeout")
	setRunbookRoot(t, root)
	attachMissionRun(t, runbookSelectCmd)
	var out string
	_ = captureStderr(t, func() {
		out = captureStdout(t, func() {
			require.NoError(t, runbookSelectCmd.RunE(runbookSelectCmd, nil))
		})
	})
	assert.Contains(t, out, "no usable")
	assert.NotContains(t, out, "no docs/runbooks/*.runbook.yaml sidecars found")
}

func TestRunbookSelect_StrictRendersThenFails(t *testing.T) {
	root := runbookSelectTestRoot(t, map[string]string{"fix-timeout": fixtureRunbookA, "broken-runbook": fixtureRunbookInvalidLevel})
	resetRunbookSelectFlags(t)
	setRunbookSignals(t, "timeout")
	setRunbookRoot(t, root)
	setCmdFlag(t, runbookSelectCmd, flagFormat, outputFormatJSON)
	setCmdFlag(t, runbookSelectCmd, "strict", "true")
	attachMissionRun(t, runbookSelectCmd)
	var out string
	var err error
	_ = captureStderr(t, func() {
		out = captureStdout(t, func() { err = runbookSelectCmd.RunE(runbookSelectCmd, nil) })
	})
	assert.Contains(t, out, `"runbook_id": "fix-timeout"`, "the selection is rendered before the strict failure")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "strict: skipped=1 warnings=0")
}

func TestRunbookSelect_StrictOnCleanCorpusSucceeds(t *testing.T) {
	root := runbookSelectTestRoot(t, map[string]string{"fix-timeout": fixtureRunbookA})
	resetRunbookSelectFlags(t)
	setRunbookSignals(t, "timeout")
	setRunbookRoot(t, root)
	setCmdFlag(t, runbookSelectCmd, "strict", "true")
	attachMissionRun(t, runbookSelectCmd)
	_ = captureStdout(t, func() { require.NoError(t, runbookSelectCmd.RunE(runbookSelectCmd, nil)) })
}

// TestCommittedRunbookCorpusHasNoDiagnostics is the commit-time gate: the
// runtime tolerates a broken sidecar, the repository does not.
func TestCommittedRunbookCorpusHasNoDiagnostics(t *testing.T) {
	repoRoot, err := filepath.Abs(filepath.Join("..", ".."))
	require.NoError(t, err)
	corpus, err := loadRunbookSidecars(filepath.Join(t.TempDir(), ".strategist"), repoRoot)
	require.NoError(t, err)
	require.NotEmpty(t, corpus.Candidates, "the committed corpus must not be empty")
	assert.Empty(t, corpus.Diagnostics, "committed runbook sidecars must have no skip or warning diagnostics")
}

func TestRunbookSelect_UnreadableChestConfigDegradesToNoTrustWithAWarning(t *testing.T) {
	root := runbookSelectTestRoot(t, map[string]string{"fix-timeout": fixtureRunbookA})
	require.NoError(t, os.MkdirAll(root, 0o755))
	require.NoError(t, os.WriteFile(filepath.Join(root, "treasure-chests.yaml"), []byte("chests: [unterminated"), 0o600))
	stdout, stderr, err := runSelect(t, root, "timeout")
	require.NoError(t, err, "trust is optional metadata: a broken chest config must not stop the selection")
	assert.Contains(t, stdout, `"runbook_id": "fix-timeout"`)
	assert.Contains(t, stderr, "warning treasure-chests.yaml code=trust_unavailable")
	assert.Contains(t, stderr, "corpus loaded=1 skipped=0 warnings=1")
}

func TestRunbookSelect_StrictFailsOnTrustUnavailable(t *testing.T) {
	root := runbookSelectTestRoot(t, map[string]string{"fix-timeout": fixtureRunbookA})
	require.NoError(t, os.MkdirAll(root, 0o755))
	require.NoError(t, os.WriteFile(filepath.Join(root, "treasure-chests.yaml"), []byte("chests: [unterminated"), 0o600))
	resetRunbookSelectFlags(t)
	setRunbookSignals(t, "timeout")
	setRunbookRoot(t, root)
	setCmdFlag(t, runbookSelectCmd, "strict", "true")
	attachMissionRun(t, runbookSelectCmd)
	var err error
	_ = captureStderr(t, func() {
		_ = captureStdout(t, func() { err = runbookSelectCmd.RunE(runbookSelectCmd, nil) })
	})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "strict: skipped=0 warnings=1")
}

// configureRunbookRoot writes active.yaml with runbooks.canonical_path and
// returns the configured directory inside the project root.
func configureRunbookRoot(t *testing.T, strategistRoot, canonical string) string {
	t.Helper()
	require.NoError(t, os.MkdirAll(strategistRoot, 0o755))
	yaml := "mode: epic\nbase_path: .analysis\nrunbooks:\n  canonical_path: " + canonical + "\n"
	require.NoError(t, os.WriteFile(filepath.Join(strategistRoot, "active.yaml"), []byte(yaml), 0o600))
	dir := filepath.Join(filepath.Dir(strategistRoot), filepath.FromSlash(canonical))
	require.NoError(t, os.MkdirAll(dir, 0o755))
	return dir
}

func writeRunbookIn(t *testing.T, dir, name, id, sourceDoc string) {
	t.Helper()
	content := strings.NewReplacer("{ID}", id, "{DOC}", sourceDoc).Replace(`schema_version: "1"
runbook_id: {ID}
runbook_type: analytical
source_doc: {DOC}
applies_when:
  - timeout
objective: fixture
`)
	require.NoError(t, os.WriteFile(filepath.Join(dir, name+".runbook.yaml"), []byte(content), 0o600))
	require.NoError(t, os.WriteFile(filepath.Join(filepath.Dir(sourceDocAbs(dir, sourceDoc)), name+".md"), []byte("# "+name+"\n"), 0o600))
}

// sourceDocAbs resolves a project-relative source_doc against the project root
// that contains dir (dir is <project>/<some/root>).
func sourceDocAbs(dir, sourceDoc string) string {
	project := dir
	for _, part := range strings.Split(filepath.ToSlash(filepath.Dir(sourceDoc)), "/") {
		if part != "" && part != "." {
			project = filepath.Dir(project)
		}
	}
	return filepath.Join(project, filepath.FromSlash(sourceDoc))
}

func TestRunbookSelect_ReadsTheConfiguredRootAndTheLegacyRoot(t *testing.T) {
	root := runbookSelectTestRoot(t, map[string]string{"legacy-one": fixtureWithID("legacy-one", "legacy-one")})
	configured := configureRunbookRoot(t, root, ".analysis/runbooks")
	writeRunbookIn(t, configured, "configured-one", "configured-one", ".analysis/runbooks/configured-one.md")
	stdout, stderr, err := runSelect(t, root, "timeout")
	require.NoError(t, err)
	assert.Contains(t, stdout, `"runbook_id": "configured-one"`)
	assert.Contains(t, stdout, `"runbook_id": "legacy-one"`)
	assert.NotContains(t, stderr, "skipped")
}

func TestRunbookSelect_ConfiguredRootWinsOverLegacyForTheSameID(t *testing.T) {
	root := runbookSelectTestRoot(t, map[string]string{"shared": fixtureWithID("shared", "shared")})
	configured := configureRunbookRoot(t, root, ".analysis/runbooks")
	writeRunbookIn(t, configured, "shared", "shared", ".analysis/runbooks/shared.md")
	stdout, stderr, err := runSelect(t, root, "timeout")
	require.NoError(t, err)
	assert.Contains(t, stdout, `"ref": ".analysis/runbooks/shared.md"`)
	assert.NotContains(t, stdout, "docs/runbooks/shared.md")
	assert.Contains(t, stderr, "skipped docs/runbooks/shared.runbook.yaml code=duplicate_runbook_id")
}

func TestRunbookSelect_AConfiguredRootWithNoFilesStillSelectsTheLegacyCorpus(t *testing.T) {
	root := runbookSelectTestRoot(t, map[string]string{"legacy-one": fixtureWithID("legacy-one", "legacy-one")})
	configureRunbookRoot(t, root, ".analysis/runbooks")
	stdout, _, err := runSelect(t, root, "timeout")
	require.NoError(t, err)
	assert.Contains(t, stdout, `"runbook_id": "legacy-one"`)
}

func TestRunbookSelect_ConfiguredRootEqualToTheLegacyRootIsNotLoadedTwice(t *testing.T) {
	root := runbookSelectTestRoot(t, map[string]string{"only": fixtureWithID("only", "only")})
	configureRunbookRoot(t, root, "docs/runbooks")
	stdout, stderr, err := runSelect(t, root, "timeout")
	require.NoError(t, err)
	assert.Contains(t, stdout, `"runbook_id": "only"`)
	assert.NotContains(t, stderr, "duplicate_runbook_id")
}

func TestRunbookSelect_MalformedActiveYAMLWarnsAndUsesTheLegacyRoot(t *testing.T) {
	root := runbookSelectTestRoot(t, map[string]string{"legacy-one": fixtureWithID("legacy-one", "legacy-one")})
	require.NoError(t, os.MkdirAll(root, 0o755))
	require.NoError(t, os.WriteFile(filepath.Join(root, "active.yaml"), []byte("runbooks: [unterminated"), 0o600))
	stdout, stderr, err := runSelect(t, root, "timeout")
	require.NoError(t, err)
	assert.Contains(t, stdout, `"runbook_id": "legacy-one"`)
	assert.Contains(t, stderr, "code=runbooks_root_unavailable")
}
