package compile_test

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/SergioLacerda/strategist-skill/internal/compile"
)

// FuzzCompileConfig ensures Config never panics on arbitrary active.yaml content.
// Properties verified: no panic, no infinite loop, error is acceptable.
func FuzzCompileConfig(f *testing.F) {
	f.Add("mode: full\n")
	f.Add("mode: lightweight\nprovider: claude\n")
	f.Add("")
	f.Add(": invalid yaml: :")
	f.Add("mode: [unclosed")
	f.Add("\x00\xff\xfe")

	f.Fuzz(func(t *testing.T, activeContent string) {
		dir := t.TempDir()
		if err := os.MkdirAll(filepath.Join(dir, "personas"), 0o755); err != nil {
			t.Skip()
		}
		if err := os.MkdirAll(filepath.Join(dir, "roles"), 0o755); err != nil {
			t.Skip()
		}
		if err := os.WriteFile(filepath.Join(dir, "active.yaml"), []byte(activeContent), 0o644); err != nil {
			t.Skip()
		}
		_ = compile.Config(dir, filepath.Join(dir, ".compiled", ".config.gz"))
	})
}

// FuzzCompileIndex ensures Index never panics on arbitrary knowledge index content.
func FuzzCompileIndex(f *testing.F) {
	f.Add("sources: []\n")
	f.Add("sources:\n  - id: doc\n    tags: [a, b]\n")
	f.Add("")
	f.Add("sources: [invalid yaml: :")
	f.Add("\x00\xff")

	f.Fuzz(func(t *testing.T, content string) {
		dir := t.TempDir()
		kiPath := filepath.Join(dir, "knowledge.index.yaml")
		if err := os.WriteFile(kiPath, []byte(content), 0o644); err != nil {
			t.Skip()
		}
		_ = compile.Index(kiPath, filepath.Join(dir, ".compiled", ".index.gz"))
	})
}
