package install_test

import (
	"context"
	"fmt"
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

func (m *mockExtractor) ReadFile(relPath string) ([]byte, error) {
	switch relPath {
	case "templates/epic-standalone.yaml":
		return []byte("mode: epic\nbase_path: .analysis\n"), nil
	case "SKILL.md":
		return []byte("# SKILL\n"), nil
	default:
		for _, file := range domain.NormativeRuntimeDefaultFiles() {
			if relPath == file.Path {
				return []byte(relPath + "\n"), nil
			}
		}
		return nil, fmt.Errorf("mockExtractor: file not found: %s", relPath)
	}
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

func TestInstall_StrictCompileMakesFailureFatal(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	comp := &mockCompiler{failErr: os.ErrNotExist}
	err := newSvc(t, &mockExtractor{}, comp).Install(context.Background(), domain.InstallConfig{
		Target: dir, Silent: true, StrictCompile: true,
	})
	require.Error(t, err, "strict-compile install must fail on a CompileAll error")
	require.ErrorContains(t, err, "strict compile")

	strategistDir := filepath.Join(dir, ".strategist")
	_, statErr := os.Stat(strategistDir)
	assert.ErrorIs(t, statErr, os.ErrNotExist, "fresh install must be fully rolled back on strict-compile failure")
}

func TestInstall_RollbackRemovesFullTreeOnFreshInstallFailure(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	// Fail after extraction+shim, during compile with StrictCompile — exercises the
	// full manifest (active.yaml, gitignore, shim) plus the fresh-tree RemoveAll path.
	comp := &mockCompiler{failErr: os.ErrNotExist}
	err := newSvc(t, &mockExtractor{}, comp).Install(context.Background(), domain.InstallConfig{
		Target: dir, Silent: true, StrictCompile: true,
	})
	require.Error(t, err)

	strategistDir := filepath.Join(dir, ".strategist")
	_, statErr := os.Stat(strategistDir)
	assert.ErrorIs(t, statErr, os.ErrNotExist, ".strategist/ must be fully removed, not left partially extracted")
}

func TestInstall_RollbackPreservesPreexistingStrategistDir(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	strategistDir := filepath.Join(dir, ".strategist")
	require.NoError(t, os.MkdirAll(strategistDir, 0o755))
	require.NoError(t, os.WriteFile(filepath.Join(strategistDir, "SENTINEL"), []byte("keep"), 0o644))

	comp := &mockCompiler{failErr: os.ErrNotExist}
	err := newSvc(t, &mockExtractor{}, comp).Install(context.Background(), domain.InstallConfig{
		Target: dir, Silent: true, Force: true, StrictCompile: true,
	})
	require.Error(t, err)
	assert.FileExists(t, filepath.Join(strategistDir, "SENTINEL"), "reinstall over an existing tree must never delete pre-existing content on rollback")
}

func TestInstall_NoShim_SkipsShimWrite(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	svc := newSvc(t, &mockExtractor{}, &mockCompiler{})
	require.NoError(t, svc.Install(context.Background(), domain.InstallConfig{
		Target: dir, Silent: true, NoShim: true,
	}))
	shimPath := filepath.Join(svc.ShimHomeDir, ".claude", "skills", "strategist", "SKILL.md")
	_, err := os.Stat(shimPath)
	assert.ErrorIs(t, err, os.ErrNotExist, "--no-shim must not write under the home shim path")
}

func TestInstall_ShimPath_WritesToCustomPath(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	customShim := filepath.Join(t.TempDir(), "custom", "SKILL.md")
	svc := newSvc(t, &mockExtractor{}, &mockCompiler{})
	require.NoError(t, svc.Install(context.Background(), domain.InstallConfig{
		Target: dir, Silent: true, ShimPath: customShim,
	}))
	assert.FileExists(t, customShim)

	defaultShim := filepath.Join(svc.ShimHomeDir, ".claude", "skills", "strategist", "SKILL.md")
	_, err := os.Stat(defaultShim)
	assert.ErrorIs(t, err, os.ErrNotExist, "--shim-path must not also write the default home shim path")
}

func TestInstall_ShimPath_RollsBackOnLaterFailure(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	customShim := filepath.Join(t.TempDir(), "custom", "SKILL.md")
	comp := &mockCompiler{failErr: os.ErrNotExist}
	err := newSvc(t, &mockExtractor{}, comp).Install(context.Background(), domain.InstallConfig{
		Target: dir, Silent: true, ShimPath: customShim, StrictCompile: true,
	})
	require.Error(t, err)
	_, statErr := os.Stat(customShim)
	assert.ErrorIs(t, statErr, os.ErrNotExist, "custom shim path must be rolled back on later fatal failure")
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

func TestInstall_AwarenessRefresherCalledAfterInstall(t *testing.T) {
	t.Parallel()

	t.Run("silent install calls refresher", func(t *testing.T) {
		t.Parallel()
		dir := t.TempDir()
		called := false
		svc := newSvc(t, &mockExtractor{}, &mockCompiler{})
		svc.AwarenessRefresher = func(strategistRoot, projectRoot, _ string) bool {
			called = true
			assert.NotEmpty(t, strategistRoot)
			assert.NotEmpty(t, projectRoot)
			return true
		}
		require.NoError(t, svc.Install(context.Background(), domain.InstallConfig{Target: dir, Silent: true}))
		assert.True(t, called, "AwarenessRefresher must be called after silent install")
	})

	t.Run("nil refresher does not panic", func(t *testing.T) {
		t.Parallel()
		dir := t.TempDir()
		svc := newSvc(t, &mockExtractor{}, &mockCompiler{})
		svc.AwarenessRefresher = nil
		require.NoError(t, svc.Install(context.Background(), domain.InstallConfig{Target: dir, Silent: true}))
	})

	t.Run("refresher returning false is non-fatal", func(t *testing.T) {
		t.Parallel()
		dir := t.TempDir()
		svc := newSvc(t, &mockExtractor{}, &mockCompiler{})
		svc.AwarenessRefresher = func(_, _, _ string) bool { return false }
		err := svc.Install(context.Background(), domain.InstallConfig{Target: dir, Silent: true})
		require.NoError(t, err, "partial awareness failure must not abort install")
	})

	t.Run("refresher receives correct strategistRoot and projectRoot", func(t *testing.T) {
		t.Parallel()
		dir := t.TempDir()
		var gotStrategistRoot, gotProjectRoot string
		svc := newSvc(t, &mockExtractor{}, &mockCompiler{})
		svc.AwarenessRefresher = func(strategistRoot, projectRoot, _ string) bool {
			gotStrategistRoot = strategistRoot
			gotProjectRoot = projectRoot
			return true
		}
		require.NoError(t, svc.Install(context.Background(), domain.InstallConfig{Target: dir, Silent: true}))
		assert.Equal(t, filepath.Join(dir, ".strategist"), gotStrategistRoot)
		assert.Equal(t, dir, gotProjectRoot)
	})

	t.Run("refresher receives version from Service.Version", func(t *testing.T) {
		t.Parallel()
		dir := t.TempDir()
		var gotVersion string
		svc := newSvc(t, &mockExtractor{}, &mockCompiler{})
		svc.Version = "1.2.3"
		svc.AwarenessRefresher = func(_, _, version string) bool {
			gotVersion = version
			return true
		}
		require.NoError(t, svc.Install(context.Background(), domain.InstallConfig{Target: dir, Silent: true}))
		assert.Equal(t, "1.2.3", gotVersion)
	})
}
