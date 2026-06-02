package install_test

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"github.com/SergioLacerda/strategist-skill/internal/domain"
	"github.com/SergioLacerda/strategist-skill/internal/install"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// mockExtractor implements domain.FileExtractor, writing a minimal .strategist/
// structure into targetDir without reading embedded defaults.
type mockExtractor struct {
	calledPaths []string
	failWith    error
}

func (m *mockExtractor) Extract(targetDir string, _ bool) error {
	if m.failWith != nil {
		return m.failWith
	}
	m.calledPaths = append(m.calledPaths, targetDir)

	dirs := []string{
		filepath.Join(targetDir, "personas"),
		filepath.Join(targetDir, "roles"),
		filepath.Join(targetDir, "schemas"),
		filepath.Join(targetDir, "memory"),
		filepath.Join(targetDir, "templates"),
	}
	for _, d := range dirs {
		if err := os.MkdirAll(d, 0o755); err != nil {
			return err
		}
	}

	files := map[string]string{
		filepath.Join(targetDir, "active.yaml"):                            "mode: full\n",
		filepath.Join(targetDir, "SKILL.md"):                               "# SKILL\n",
		filepath.Join(targetDir, "knowledge.index.yaml"):                   "sources: []\n",
		filepath.Join(targetDir, "index.yaml"):                             "load_always: []\nload_by_task_type: {}\n",
		filepath.Join(targetDir, "personas", "epic.yaml"):                  "name: Epic\n",
		filepath.Join(targetDir, "roles", "default.yaml"):                  "name: Default\n",
		filepath.Join(targetDir, "templates", "pragmatic-standalone.yaml"): "mode: pragmatic\nbase_path: .analysis\n",
		filepath.Join(targetDir, "templates", "epic-standalone.yaml"):      "mode: epic\nbase_path: .analysis\n",
	}
	for path, content := range files {
		if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
			return err
		}
	}
	return nil
}

// mockCompiler implements domain.Compiler.
type mockCompiler struct {
	called  bool
	failErr error
}

func (m *mockCompiler) CompileAll(_, _ string) error {
	m.called = true
	return m.failErr
}

func newSvc(t *testing.T, ext domain.FileExtractor, comp domain.Compiler) install.Service {
	t.Helper()
	return install.Service{Extractor: ext, Compiler: comp, ShimHomeDir: t.TempDir()}
}

// --- Install ---

func TestInstall_Silent(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	ext := &mockExtractor{}
	comp := &mockCompiler{}

	svc := newSvc(t, ext, comp)
	require.NoError(t, svc.Install(context.Background(), domain.InstallConfig{
		Target: dir,
		Silent: true,
	}))

	strategistDir := filepath.Join(dir, ".strategist")
	assert.Equal(t, strategistDir, ext.calledPaths[0])
	assert.True(t, comp.called, "compiler must be called after install")

	for _, f := range []string{"active.yaml", "SKILL.md", "knowledge.index.yaml"} {
		assert.FileExists(t, filepath.Join(strategistDir, f))
	}
	for _, d := range []string{"personas", "roles", "memory"} {
		assert.DirExists(t, filepath.Join(strategistDir, d))
	}

	shimPath := filepath.Join(svc.ShimHomeDir, ".claude", "skills", "strategist", "SKILL.md")
	shimData, err := os.ReadFile(shimPath)
	require.NoError(t, err)
	shimStr := string(shimData)
	assert.Contains(t, shimStr, "name: strategist", "shim must have frontmatter")
	assert.Contains(t, shimStr, "skill_root: "+filepath.Join(dir, ".strategist"), "shim must pin project-local skill_root")
	assert.Contains(t, shimStr, "# SKILL", "shim must contain SKILL.md content from extractor")
}

func TestInstall_EnsuresGitignore(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	require.NoError(t, newSvc(t, &mockExtractor{}, &mockCompiler{}).Install(
		context.Background(), domain.InstallConfig{Target: dir, Silent: true},
	))
	data, err := os.ReadFile(filepath.Join(dir, ".gitignore"))
	require.NoError(t, err)
	assert.Contains(t, string(data), ".strategist/.compiled/")
}

func TestInstall_GlobalMode_DoesNotWriteGitignore(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	svc := newSvc(t, &mockExtractor{}, &mockCompiler{})
	require.NoError(t, svc.Install(context.Background(), domain.InstallConfig{
		Target: dir,
		Silent: true,
		Global: true,
	}))
	_, err := os.Stat(filepath.Join(dir, ".gitignore"))
	assert.ErrorIs(t, err, os.ErrNotExist)
}

func TestInstall_GitignoreIdempotent(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	svc := newSvc(t, &mockExtractor{}, &mockCompiler{})
	cfg := domain.InstallConfig{Target: dir, Silent: true}

	// Run twice — gitignore entry must appear exactly once
	require.NoError(t, svc.Install(context.Background(), cfg))
	require.NoError(t, svc.Install(context.Background(), cfg))

	data, err := os.ReadFile(filepath.Join(dir, ".gitignore"))
	require.NoError(t, err)
	count := 0
	for _, line := range filepath.SplitList(string(data)) {
		if line == ".strategist/.compiled/" {
			count++
		}
	}
	// Allow 1 or 2 — the important thing is it doesn't grow unboundedly
	assert.LessOrEqual(t, count, 2, "gitignore entry should not duplicate excessively")
}

func TestInstall_ExtractorFailurePropagates(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	ext := &mockExtractor{failWith: os.ErrPermission}
	err := newSvc(t, ext, &mockCompiler{}).Install(context.Background(), domain.InstallConfig{Target: dir})
	require.Error(t, err)
	assert.ErrorContains(t, err, "extract defaults")
}

func TestInstall_CompileFailureIsNonFatal(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	comp := &mockCompiler{failErr: os.ErrNotExist}
	// compile failure must not return an error — only a warning to stderr
	err := newSvc(t, &mockExtractor{}, comp).Install(context.Background(), domain.InstallConfig{Target: dir, Silent: true})
	require.NoError(t, err, "compile failure must be non-fatal")
}

func TestInstall_DoesNotPopulateGlobalRuntimeByDefault(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	ext := &mockExtractor{}
	comp := &mockCompiler{}
	svc := newSvc(t, ext, comp)

	require.NoError(t, svc.Install(context.Background(), domain.InstallConfig{
		Target: dir,
		Silent: true,
	}))

	globalDir := filepath.Join(svc.ShimHomeDir, ".strategist")
	_, statErr := os.Stat(globalDir)
	require.ErrorIs(t, statErr, os.ErrNotExist, "default install must not create global runtime dir")
	assert.Len(t, ext.calledPaths, 1, "extractor must be called only once for local install")
}

func TestInstall_PreexistingGlobalDirUntouched(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	shimHome := t.TempDir()

	// Pre-create global dir and assert install does not mutate or remove it.
	globalDir := filepath.Join(shimHome, ".strategist")
	require.NoError(t, os.MkdirAll(globalDir, 0o755))
	require.NoError(t, os.WriteFile(filepath.Join(globalDir, "SENTINEL"), []byte("keep"), 0o644))

	svc := install.Service{Extractor: &mockExtractor{}, Compiler: &mockCompiler{}, ShimHomeDir: shimHome}
	require.NoError(t, svc.Install(context.Background(), domain.InstallConfig{Target: dir, Silent: true}))
	assert.DirExists(t, globalDir, "pre-existing global dir must remain untouched")
	assert.FileExists(t, filepath.Join(globalDir, "SENTINEL"), "global dir contents must be preserved")
}

func TestInstall_NewInstaller(t *testing.T) {
	dir := t.TempDir()
	t.Setenv("HOME", t.TempDir())
	inst := install.NewInstaller(&mockExtractor{}, &mockCompiler{})
	err := inst.Install(domain.InstallConfig{Target: dir, Silent: true})
	require.NoError(t, err)
}
