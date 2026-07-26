//go:build spec

package spec_test

import (
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"

	"github.com/SergioLacerda/strategist-skill/internal/domain"
	"gopkg.in/yaml.v3"
)

type fixture struct {
	Scenario      string `yaml:"scenario"`
	ExpectedEvent string `yaml:"expected_event"`
}

func testDir(t *testing.T) string {
	t.Helper()
	_, file, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("runtime.Caller failed")
	}
	return filepath.Dir(file)
}

func repoRoot(t *testing.T) string {
	t.Helper()
	return filepath.Clean(filepath.Join(testDir(t), "..", ".."))
}

func readFile(t *testing.T, path string) string {
	t.Helper()
	b, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read %s: %v", path, err)
	}
	return string(b)
}

func normativeRuntimeFiles() []string {
	return domain.NormativeRuntimeDefaultPaths()
}

func readFixture(t *testing.T, path string) fixture {
	t.Helper()
	b, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read fixture %s: %v", path, err)
	}
	var f fixture
	if err := yaml.Unmarshal(b, &f); err != nil {
		t.Fatalf("parse fixture %s: %v", path, err)
	}
	return f
}

// assertNoToken fails if the file at path contains token.
func assertNoToken(t *testing.T, path, token string) {
	t.Helper()
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read %s: %v", path, err)
	}
	if strings.Contains(string(data), token) {
		t.Fatalf("%s must not contain %q", path, token)
	}
}

// relativeFileSet walks dir and returns the set of file paths relative to dir, using forward
// slashes regardless of OS.
func relativeFileSet(t *testing.T, dir string) map[string]bool {
	t.Helper()
	files := map[string]bool{}
	err := filepath.WalkDir(dir, func(path string, d os.DirEntry, err error) error {
		if err != nil || d.IsDir() {
			return err
		}
		rel, relErr := filepath.Rel(dir, path)
		if relErr != nil {
			return relErr
		}
		files[filepath.ToSlash(rel)] = true
		return nil
	})
	if err != nil {
		t.Fatalf("walk %s: %v", dir, err)
	}
	return files
}

// --- Provider discovery conformance tests ---

// providerBootstrapFiles lists all provider bootstrap surfaces that must declare
// Strategist runtime discovery semantics.
func providerBootstrapFiles(t *testing.T) []string {
	t.Helper()
	root := repoRoot(t)
	return []string{
		filepath.Join(root, ".codex", "commands.md"),
		filepath.Join(root, ".claude", "claude-instructions.md"),
		filepath.Join(root, ".antigravity", "antigravity-instructions.md"),
		filepath.Join(root, "AGENTS.md"),
		filepath.Join(root, "GEMINI.md"),
	}
}
