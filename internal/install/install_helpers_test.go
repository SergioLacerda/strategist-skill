package install_test

import (
	"fmt"
	"os"
	"path/filepath"
	"testing"

	"github.com/SergioLacerda/strategist-skill/internal/domain"
	"github.com/SergioLacerda/strategist-skill/internal/install"
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
