package tests_test

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

// mockExtractor implements domain.FileExtractor by writing a minimal .strategist/
// structure into targetDir — no real filesystem reads from embedded defaults.
type mockExtractor struct {
	calledWith string
}

func (m *mockExtractor) Extract(targetDir string) error {
	m.calledWith = targetDir

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
		filepath.Join(targetDir, "active.yaml"):                              "mode: full\n",
		filepath.Join(targetDir, "SKILL.md"):                                 "# SKILL\n",
		filepath.Join(targetDir, "knowledge.index.yaml"):                     "sources: []\n",
		filepath.Join(targetDir, "index.yaml"):                               "load_always: []\nload_by_task_type: {}\n",
		filepath.Join(targetDir, "personas", "epic.yaml"):                    "name: Epic\n",
		filepath.Join(targetDir, "roles", "default.yaml"):                    "name: Default\n",
		filepath.Join(targetDir, "templates", "pragmatic-standalone.yaml"):   "mode: pragmatic\nbase_path: .analysis\n",
	}
	for path, content := range files {
		if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
			return err
		}
	}
	return nil
}

// mockCompiler implements domain.Compiler and records whether it was called.
type mockCompiler struct {
	called bool
}

func (m *mockCompiler) CompileAll(_, _ string) error {
	m.called = true
	return nil
}

func TestInstallSilent_ProducesExpectedStructure(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	extractor := &mockExtractor{}
	compiler := &mockCompiler{}

	svc := install.Service{
		Extractor: extractor,
		Compiler:  compiler,
	}

	cfg := domain.InstallConfig{
		Target: dir,
		Silent: true,
	}

	err := svc.Install(context.Background(), cfg)
	require.NoError(t, err)

	strategistDir := filepath.Join(dir, ".strategist")

	expectedFiles := []string{
		filepath.Join(strategistDir, "active.yaml"),
		filepath.Join(strategistDir, "SKILL.md"),
		filepath.Join(strategistDir, "knowledge.index.yaml"),
	}
	for _, p := range expectedFiles {
		assert.FileExists(t, p, "expected file: %s", p)
	}

	expectedDirs := []string{
		filepath.Join(strategistDir, "personas"),
		filepath.Join(strategistDir, "roles"),
		filepath.Join(strategistDir, "memory"),
	}
	for _, p := range expectedDirs {
		assert.DirExists(t, p, "expected dir: %s", p)
	}

	assert.Equal(t, strategistDir, extractor.calledWith, "extractor called with correct target")
	assert.True(t, compiler.called, "compiler should be called after install")
}

func TestInstallSilent_EnsuresGitignore(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	extractor := &mockExtractor{}
	compiler := &mockCompiler{}

	svc := install.Service{Extractor: extractor, Compiler: compiler}
	cfg := domain.InstallConfig{Target: dir, Silent: true}

	require.NoError(t, svc.Install(context.Background(), cfg))

	gitignore, err := os.ReadFile(filepath.Join(dir, ".gitignore"))
	require.NoError(t, err)
	assert.Contains(t, string(gitignore), ".strategist/.compiled/")
}
