strategist 
.analysis/todo/critique_tests.md 
.analysis/todo/critique_tests2.md
avaliar criticas gerais sobre testes em GOLang e adaptar ao nosso cenario, ou seja, desconsiderar testes em outras linguagens, vamos incorporar as melhores praticas sugeridas.
Bash Read strategist skill configuration
IN
cat /home/sergio/.claude/skills/strategist/skill.yaml 2>/dev/null || cat /home/sergio/.strategist/skill.yaml 2>/dev/null || ls /home/sergio/.strategist/ 2>/dev/null

OUT
Exit code 2
Bash List strategist skill files
IN
ls /home/sergio/.claude/skills/strategist/ 2>/dev/null && ls ~/.strategist/ 2>/dev/null

OUT
Exit code 2
SKILL.md
Bash Read strategist SKILL.md
IN
cat ~/.strategist/SKILL.md

OUT
Exit code 1
cat: /home/sergio/.strategist/SKILL.md: No such file or directory
Bash Find strategist skill files
IN
ls ~/.strategist/ 2>/dev/null; find ~/.claude/skills -name "*.md" -o -name "*.yaml" 2>/dev/null | head -20

OUT
/home/sergio/.claude/skills/brainstorming/spec-document-reviewer-prompt.md
/home/sergio/.claude/skills/brainstorming/visual-companion.md
/home/sergio/.claude/skills/brainstorming/SKILL.md
/home/sergio/.claude/skills/openspec-apply-change/SKILL.md
/home/sergio/.claude/skills/openspec-archive-change/SKILL.md
/home/sergio/.claude/skills/openspec-explore/SKILL.md
/home/sergio/.claude/skills/openspec-propose/SKILL.md
/home/sergio/.claude/skills/strategist/SKILL.md
Read SKILL.md
Bash Check strategist binary availability
IN
which strategist 2>/dev/null && strategist --help 2>/dev/null | head -30

OUT
/home/sergio/.local/bin/strategist
Strategist — install, compile, and manage the Strategist skill for Claude agents.

Usage:
  strategist [command]

Available Commands:
  check-stale Check if a compiled artifact is stale (exit 0=fresh, exit 1=stale)
  compile     Compile all skill artifacts from a .strategist/ root
  completion  Generate the autocompletion script for the specified shell
  help        Help about any command
  install     Install the Strategist skill into a target repository
  version     Print the strategist version

Flags:
  -h, --help   help for strategist

Use "strategist [command] --help" for more information about a command.
The strategist skill is an agent shim — I'll read the critique files and act on the task directly.

Read critique_tests.md
Read critique_tests2.md
critique_tests.md is Python-specific — discarding it as instructed. Now let me survey the current Go tests against the world-class patterns in critique_tests2.md.

Bash List all Go test files
IN
find /home/sergio/dev/strategist-skill -name "*_test.go" | sort

OUT
/home/sergio/dev/strategist-skill/cmd/strategist/cmd_test.go
/home/sergio/dev/strategist-skill/internal/compile/compile_test.go
/home/sergio/dev/strategist-skill/internal/compile/helpers_test.go
/home/sergio/dev/strategist-skill/internal/domain/domain_test.go
/home/sergio/dev/strategist-skill/internal/embed/embed_test.go
/home/sergio/dev/strategist-skill/internal/embed/extractfs_test.go
/home/sergio/dev/strategist-skill/internal/install/active_yaml_test.go
/home/sergio/dev/strategist-skill/internal/install/installer_whitebox_test.go
/home/sergio/dev/strategist-skill/internal/install/install_test.go
/home/sergio/dev/strategist-skill/internal/install/wizard_test.go
/home/sergio/dev/strategist-skill/internal/stale/stale_test.go
/home/sergio/dev/strategist-skill/tests/compile_test.go
/home/sergio/dev/strategist-skill/tests/fixtures_test.go
/home/sergio/dev/strategist-skill/tests/install_test.go
/home/sergio/dev/strategist-skill/tests/stale_test.go
Bash List internal and cmd packages
IN
ls /home/sergio/dev/strategist-skill/internal/ 2>/dev/null && ls /home/sergio/dev/strategist-skill/cmd/ 2>/dev/null

OUT
compile
domain
embed
install
stale
strategist
Bash Sample existing test files for pattern audit
IN
for f in \
  /home/sergio/dev/strategist-skill/internal/compile/compile_test.go \
  /home/sergio/dev/strategist-skill/internal/stale/stale_test.go \
  /home/sergio/dev/strategist-skill/internal/embed/embed_test.go \
  /home/sergio/dev/strategist-skill/cmd/strategist/cmd_test.go \
  /home/sergio/dev/strategist-skill/tests/compile_test.go \
  /home/sergio/dev/strategist-skill/tests/install_test.go; do
  echo "=== $f ==="
  head -60 "$f"
  echo
done

OUT
=== /home/sergio/dev/strategist-skill/internal/compile/compile_test.go ===
package compile_test

import (
	"compress/gzip"
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	"github.com/SergioLacerda/strategist-skill/internal/compile"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// readGzJSON decompresses a gzipped JSON artifact into v.
func readGzJSON(t *testing.T, path string, v any) {
	t.Helper()
	f, err := os.Open(path)
	require.NoError(t, err, "open artifact %s", path)
	defer f.Close() //nolint:errcheck
	gz, err := gzip.NewReader(f)
	require.NoError(t, err, "gzip reader")
	defer gz.Close() //nolint:errcheck
	require.NoError(t, json.NewDecoder(gz).Decode(v), "json decode")
}

// minimalRoot creates a minimal .strategist/-like directory in dir.
func minimalRoot(t *testing.T, dir string) {
	t.Helper()
	require.NoError(t, os.MkdirAll(filepath.Join(dir, "personas"), 0o755))
	require.NoError(t, os.MkdirAll(filepath.Join(dir, "roles"), 0o755))
	require.NoError(t, os.WriteFile(filepath.Join(dir, "active.yaml"), []byte("mode: full\n"), 0o644))
	require.NoError(t, os.WriteFile(filepath.Join(dir, "personas", "epic.yaml"), []byte("name: Epic\n"), 0o644))
	require.NoError(t, os.WriteFile(filepath.Join(dir, "roles", "default.yaml"), []byte("name: Default\n"), 0o644))
}

// --- CompileConfig ---

func TestCompileConfig(t *testing.T) {
	tests := []struct {
		name    string
		setup   func(t *testing.T, dir string)
		wantErr bool
		check   func(t *testing.T, artifact map[string]any)
	}{
		{
			name:  "minimal valid root produces artifact",
			setup: minimalRoot,
			check: func(t *testing.T, a map[string]any) {
				assert.Equal(t, "strategist-compiled-config/1.0", a["schema"])
				assert.NotNil(t, a["compiled_at"])
				assert.NotNil(t, a["sources"])
				assert.NotNil(t, a["active"])
				personas, ok := a["personas"].(map[string]any)
				require.True(t, ok)
				assert.Contains(t, personas, "epic")
				roles, ok := a["roles"].(map[string]any)
				require.True(t, ok)
				assert.Contains(t, roles, "default")
			},

=== /home/sergio/dev/strategist-skill/internal/stale/stale_test.go ===
package stale_test

import (
	"compress/gzip"
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/SergioLacerda/strategist-skill/internal/stale"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func writeGzJSON(t *testing.T, path string, v any) {
	t.Helper()
	require.NoError(t, os.MkdirAll(filepath.Dir(path), 0o755))
	f, err := os.Create(path)
	require.NoError(t, err)
	gz := gzip.NewWriter(f)
	require.NoError(t, json.NewEncoder(gz).Encode(v))
	require.NoError(t, gz.Close())
	require.NoError(t, f.Close())
}

func TestIsStale(t *testing.T) {
	checker := stale.Checker{}

	tests := []struct {
		name      string
		setup     func(t *testing.T, dir string) string
		wantStale bool
		wantErr   bool
	}{
		{
			name: "absent artifact is stale",
			setup: func(t *testing.T, dir string) string {
				t.Helper()
				return filepath.Join(dir, "absent.gz")
			},
			wantStale: true,
		},
		{
			name: "missing manifest is stale",
			setup: func(t *testing.T, dir string) string {
				t.Helper()
				art := filepath.Join(dir, ".config.gz")
				writeGzJSON(t, art, map[string]any{"sources": map[string]int64{}})
				return art
			},
			wantStale: true,
		},
		{
			name: "fresh artifact with empty sources is not stale",
			setup: func(t *testing.T, dir string) string {
				t.Helper()
				art := filepath.Join(dir, ".config.gz")
				writeGzJSON(t, art, map[string]any{"sources": map[string]int64{}})
				writeGzJSON(t, filepath.Join(dir, ".manifest.gz"), map[string]any{})

=== /home/sergio/dev/strategist-skill/internal/embed/embed_test.go ===
package embed_test

import (
	"os"
	"path/filepath"
	"testing"

	embedpkg "github.com/SergioLacerda/strategist-skill/internal/embed"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestExtractor_Extract_ReadOnlyTarget(t *testing.T) {
	if os.Getuid() == 0 {
		t.Skip("permission tests do not apply when running as root")
	}
	dir := t.TempDir()
	// Make target directory read-only so MkdirAll / WriteFile inside it fails
	require.NoError(t, os.Chmod(dir, 0o444))
	t.Cleanup(func() { _ = os.Chmod(dir, 0o755) })

	err := embedpkg.Extractor{}.Extract(dir)
	require.Error(t, err)
}

func TestExtractor_Extract(t *testing.T) {
	t.Run("extracts defaults into target directory", func(t *testing.T) {
		dir := t.TempDir()
		ext := embedpkg.Extractor{}
		require.NoError(t, ext.Extract(dir))

		// Core files that must always be extracted
		expectedFiles := []string{
			"SKILL.md",
			"knowledge.index.yaml",
			"skill.yaml",
			"index.yaml",
		}
		for _, f := range expectedFiles {
			assert.FileExists(t, filepath.Join(dir, f), "expected embedded file: %s", f)
		}

		// Core directories
		expectedDirs := []string{
			"personas",
			"roles",
			"schemas",
			"templates",
		}
		for _, d := range expectedDirs {
			assert.DirExists(t, filepath.Join(dir, d), "expected embedded dir: %s", d)
		}
	})

	t.Run("extract is idempotent", func(t *testing.T) {
		dir := t.TempDir()
		ext := embedpkg.Extractor{}
		require.NoError(t, ext.Extract(dir))
		require.NoError(t, ext.Extract(dir), "second extract should not fail")
	})

=== /home/sergio/dev/strategist-skill/cmd/strategist/cmd_test.go ===
package main

// Tests for all Cobra command RunE/Run functions.
// Each test targets a specific command to maximise coverage without
// triggering os.Exit (which would kill the test process).

import (
	"bytes"
	"compress/gzip"
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// --- helpers ---

// minimalStrategistRoot builds the minimal layout that CompileAll requires.
func minimalStrategistRoot(t *testing.T, dir string) string {
	t.Helper()
	require.NoError(t, os.MkdirAll(filepath.Join(dir, "personas"), 0o755))
	require.NoError(t, os.MkdirAll(filepath.Join(dir, "roles"), 0o755))
	require.NoError(t, os.WriteFile(filepath.Join(dir, "active.yaml"), []byte("mode: full\n"), 0o644))
	require.NoError(t, os.WriteFile(filepath.Join(dir, "knowledge.index.yaml"), []byte("sources: []\n"), 0o644))
	require.NoError(t, os.WriteFile(filepath.Join(dir, "index.yaml"), []byte("load_always: []\nload_by_task_type: {}\n"), 0o644))
	return dir
}

// writeGzJSON writes v as gzip-compressed JSON to path.
func writeGzJSON(t *testing.T, path string, v any) {
	t.Helper()
	require.NoError(t, os.MkdirAll(filepath.Dir(path), 0o755))
	f, err := os.Create(path)
	require.NoError(t, err)
	defer f.Close() //nolint:errcheck
	gz := gzip.NewWriter(f)
	require.NoError(t, json.NewEncoder(gz).Encode(v))
	require.NoError(t, gz.Close())
}

// freshArtifactDir creates an artifact + manifest pair with no sources
// (= always considered fresh by IsStale).
func freshArtifactDir(t *testing.T) (dir, artifactPath string) {
	t.Helper()
	dir = t.TempDir()
	artifactPath = filepath.Join(dir, "artifact.gz")
	writeGzJSON(t, artifactPath, map[string]any{"sources": map[string]int64{}})
	writeGzJSON(t, filepath.Join(dir, ".manifest.gz"), map[string]any{"generated_at": 0})
	return dir, artifactPath
}

// captureStdout replaces os.Stdout with a pipe and returns whatever was written.
func captureStdout(t *testing.T, fn func()) string {
	t.Helper()
	r, w, err := os.Pipe()
	require.NoError(t, err)
	old := os.Stdout

=== /home/sergio/dev/strategist-skill/tests/compile_test.go ===
package tests_test

import (
	"compress/gzip"
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	"github.com/SergioLacerda/strategist-skill/internal/compile"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// readGzJSON decompresses a gzipped JSON file into v.
func readGzJSON(t *testing.T, path string, v interface{}) {
	t.Helper()
	f, err := os.Open(path)
	require.NoError(t, err, "open artifact")
	defer f.Close() //nolint:errcheck
	gz, err := gzip.NewReader(f)
	require.NoError(t, err, "gzip reader")
	defer gz.Close() //nolint:errcheck
	require.NoError(t, json.NewDecoder(gz).Decode(v), "json decode")
}

// minimalStrategistRoot creates a minimal .strategist/-like directory
// with active.yaml, personas/, and roles/ inside dir.
func minimalStrategistRoot(t *testing.T, dir string) {
	t.Helper()
	require.NoError(t, os.MkdirAll(filepath.Join(dir, "personas"), 0o755))
	require.NoError(t, os.MkdirAll(filepath.Join(dir, "roles"), 0o755))
	require.NoError(t, os.WriteFile(filepath.Join(dir, "active.yaml"), []byte("mode: full\n"), 0o644))
	require.NoError(t, os.WriteFile(filepath.Join(dir, "personas", "epic.yaml"), []byte("name: Epic\n"), 0o644))
	require.NoError(t, os.WriteFile(filepath.Join(dir, "roles", "default.yaml"), []byte("name: Default\n"), 0o644))
}

func TestCompileConfig(t *testing.T) {
	dir := t.TempDir()
	minimalStrategistRoot(t, dir)
	out := filepath.Join(dir, ".compiled", ".config.gz")

	require.NoError(t, compile.Config(dir, out))
	require.FileExists(t, out)

	var artifact map[string]interface{}
	readGzJSON(t, out, &artifact)

	assert.Equal(t, "strategist-compiled-config/1.0", artifact["schema"])
	assert.NotNil(t, artifact["compiled_at"])
	assert.NotNil(t, artifact["sources"])
	assert.NotNil(t, artifact["active"])
	assert.NotNil(t, artifact["personas"])
	assert.NotNil(t, artifact["roles"])
}

func TestCompileDomain_EmptyIndex(t *testing.T) {
	dir := t.TempDir()
	indexYAML := "load_always: []\nload_by_task_type: {}\n"
	require.NoError(t, os.WriteFile(filepath.Join(dir, "index.yaml"), []byte(indexYAML), 0o644))

=== /home/sergio/dev/strategist-skill/tests/install_test.go ===
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
Let me check the remaining test files and look for t.Parallel() usage and testdata/ directories.

Bash Check for t.Parallel, testdata dirs, and go-cmp usage
IN
grep -rn "t\.Parallel" /home/sergio/dev/strategist-skill --include="*_test.go" | head -20; echo "---"; find /home/sergio/dev/strategist-skill -name "testdata" -type d; echo "---"; grep -rn "go-cmp\|google/go-cmp" /home/sergio/dev/strategist-skill/go.mod /home/sergio/dev/strategist-skill/go.sum 2>/dev/null | head -5

OUT
---
---
Bash Read install and wizard test files
IN
cat /home/sergio/dev/strategist-skill/internal/install/install_test.go
cat /home/sergio/dev/strategist-skill/internal/install/wizard_test.go

OUT
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
	calledWith string
	failWith   error
}

func (m *mockExtractor) Extract(targetDir string) error {
	if m.failWith != nil {
		return m.failWith
	}
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
		filepath.Join(targetDir, "active.yaml"):                            "mode: full\n",
		filepath.Join(targetDir, "SKILL.md"):                               "# SKILL\n",
		filepath.Join(targetDir, "knowledge.index.yaml"):                   "sources: []\n",
		filepath.Join(targetDir, "index.yaml"):                             "load_always: []\nload_by_task_type: {}\n",
		filepath.Join(targetDir, "personas", "epic.yaml"):                  "name: Epic\n",
		filepath.Join(targetDir, "roles", "default.yaml"):                  "name: Default\n",
		filepath.Join(targetDir, "templates", "pragmatic-standalone.yaml"): "mode: pragmatic\nbase_path: .analysis\n",
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

func newSvc(ext domain.FileExtractor, comp domain.Compiler) install.Service {
	return install.Service{Extractor: ext, Compiler: comp}
}

// --- Install ---

func TestInstall_Silent(t *testing.T) {
	dir := t.TempDir()
	ext := &mockExtractor{}
	comp := &mockCompiler{}

	require.NoError(t, newSvc(ext, comp).Install(context.Background(), domain.InstallConfig{
		Target: dir,
		Silent: true,
	}))

	strategistDir := filepath.Join(dir, ".strategist")
	assert.Equal(t, strategistDir, ext.calledWith)
	assert.True(t, comp.called, "compiler must be called after install")

	for _, f := range []string{"active.yaml", "SKILL.md", "knowledge.index.yaml"} {
		assert.FileExists(t, filepath.Join(strategistDir, f))
	}
	for _, d := range []string{"personas", "roles", "memory"} {
		assert.DirExists(t, filepath.Join(strategistDir, d))
	}
}

func TestInstall_EnsuresGitignore(t *testing.T) {
	dir := t.TempDir()
	require.NoError(t, newSvc(&mockExtractor{}, &mockCompiler{}).Install(
		context.Background(), domain.InstallConfig{Target: dir, Silent: true},
	))
	data, err := os.ReadFile(filepath.Join(dir, ".gitignore"))
	require.NoError(t, err)
	assert.Contains(t, string(data), ".strategist/.compiled/")
}

func TestInstall_GitignoreIdempotent(t *testing.T) {
	dir := t.TempDir()
	svc := newSvc(&mockExtractor{}, &mockCompiler{})
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
	dir := t.TempDir()
	ext := &mockExtractor{failWith: os.ErrPermission}
	err := newSvc(ext, &mockCompiler{}).Install(context.Background(), domain.InstallConfig{Target: dir})
	require.Error(t, err)
	assert.ErrorContains(t, err, "extract defaults")
}

func TestInstall_CompileFailureIsNonFatal(t *testing.T) {
	dir := t.TempDir()
	comp := &mockCompiler{failErr: os.ErrNotExist}
	// compile failure must not return an error — only a warning to stderr
	err := newSvc(&mockExtractor{}, comp).Install(context.Background(), domain.InstallConfig{Target: dir, Silent: true})
	require.NoError(t, err, "compile failure must be non-fatal")
}

func TestInstall_NewInstaller(t *testing.T) {
	dir := t.TempDir()
	inst := install.NewInstaller(&mockExtractor{}, &mockCompiler{})
	err := inst.Install(domain.InstallConfig{Target: dir, Silent: true})
	require.NoError(t, err)
}
package install

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestRunWizard(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		wantMode string
		wantBase string
		wantProv string
	}{
		{
			name:     "all defaults (empty lines)",
			input:    "\n\n\n",
			wantMode: "full",
			wantBase: ".",
			wantProv: "",
		},
		{
			name:     "custom values",
			input:    "lightweight\n/workspace\nclaude\n",
			wantMode: "lightweight",
			wantBase: "/workspace",
			wantProv: "claude",
		},
		{
			name:     "mode override, base default, no provider",
			input:    "minimal\n\n\n",
			wantMode: "minimal",
			wantBase: ".",
			wantProv: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			wc, err := runWizard(strings.NewReader(tt.input))
			require.NoError(t, err)
			assert.Equal(t, tt.wantMode, wc.Mode)
			assert.Equal(t, tt.wantBase, wc.BasePath)
			assert.Equal(t, tt.wantProv, wc.Provider)
		})
	}
}
Bash Read remaining test files
IN
cat /home/sergio/dev/strategist-skill/internal/domain/domain_test.go
cat /home/sergio/dev/strategist-skill/internal/compile/helpers_test.go
cat /home/sergio/dev/strategist-skill/internal/embed/extractfs_test.go

OUT
package domain_test

import (
	"encoding/json"
	"testing"

	"github.com/SergioLacerda/strategist-skill/internal/domain"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// --- Error sentinels ---

func TestErrors_NotNil(t *testing.T) {
	require.Error(t, domain.ErrArtifactAbsent)
	require.Error(t, domain.ErrManifestMissing)
	require.Error(t, domain.ErrSourceStale)
}

func TestErrors_Messages(t *testing.T) {
	assert.Equal(t, "artifact does not exist", domain.ErrArtifactAbsent.Error())
	assert.Equal(t, "manifest not found", domain.ErrManifestMissing.Error())
	assert.Equal(t, "source file modified after artifact", domain.ErrSourceStale.Error())
}

func TestErrors_AreDistinct(t *testing.T) {
	errs := []error{domain.ErrArtifactAbsent, domain.ErrManifestMissing, domain.ErrSourceStale}
	for i, a := range errs {
		for j, b := range errs {
			if i == j {
				assert.ErrorIs(t, a, b)
			} else {
				assert.NotErrorIs(t, a, b, "errors[%d] and errors[%d] must be distinct", i, j)
			}
		}
	}
}

// --- Struct JSON tags ---

func TestCompiledConfig_JSONTags(t *testing.T) {
	cc := domain.CompiledConfig{
		Schema:     "strategist-compiled-config/1.0",
		CompiledAt: "123",
		Sources:    map[string]int64{"/a": 100},
		Active:     map[string]any{"mode": "full"},
		Personas:   map[string]any{"epic": "yes"},
		Roles:      map[string]any{"default": "yes"},
	}
	data, err := json.Marshal(cc)
	require.NoError(t, err)
	s := string(data)
	assert.Contains(t, s, `"schema"`)
	assert.Contains(t, s, `"compiled_at"`)
	assert.Contains(t, s, `"sources"`)
	assert.Contains(t, s, `"active"`)
	assert.Contains(t, s, `"personas"`)
	assert.Contains(t, s, `"roles"`)
}

func TestCompiledDomain_JSONTags(t *testing.T) {
	cd := domain.CompiledDomain{Schema: "d/1.0", CompiledAt: "t", Sources: nil, Domain: nil}
	data, err := json.Marshal(cd)
	require.NoError(t, err)
	assert.Contains(t, string(data), `"schema"`)
}

func TestCompiledIndex_JSONTags(t *testing.T) {
	ci := domain.CompiledIndex{Schema: "i/1.0"}
	data, err := json.Marshal(ci)
	require.NoError(t, err)
	assert.Contains(t, string(data), `"schema"`)
}

func TestCompiledManifest_JSONTags(t *testing.T) {
	cm := domain.CompiledManifest{
		Schema:    "m/1.0",
		Artifacts: map[string]string{".config.gz": "sha256:abc"},
	}
	data, err := json.Marshal(cm)
	require.NoError(t, err)
	assert.Contains(t, string(data), `"artifacts"`)
}

func TestInstallConfig_Fields(t *testing.T) {
	cfg := domain.InstallConfig{Target: "/tmp/x", Silent: true, Wizard: false}
	assert.Equal(t, "/tmp/x", cfg.Target)
	assert.True(t, cfg.Silent)
	assert.False(t, cfg.Wizard)
}

func TestWizardConfig_Fields(t *testing.T) {
	wc := domain.WizardConfig{Mode: "minimal", BasePath: ".", Provider: "openai"}
	assert.Equal(t, "minimal", wc.Mode)
	assert.Equal(t, ".", wc.BasePath)
	assert.Equal(t, "openai", wc.Provider)
	wc.RolesConfig = map[string]any{"k": "v"}
	assert.Len(t, wc.RolesConfig, 1)
}
package compile

// Whitebox tests for unexported helpers — kept minimal, covering error paths
// that cannot be triggered through the public API.

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestMtime_MissingFile(t *testing.T) {
	result := mtime(filepath.Join(t.TempDir(), "nonexistent.txt"))
	assert.Equal(t, int64(0), result, "mtime of missing file must return 0")
}

func TestSHA256Artifact_MissingFile(t *testing.T) {
	result := sha256Artifact(filepath.Join(t.TempDir(), "nonexistent.gz"))
	assert.Equal(t, "unavailable", result)
}

func TestWriteGzJSON_NonSerializable(t *testing.T) {
	dir := t.TempDir()
	out := filepath.Join(dir, "out.gz")
	// Channels cannot be JSON-encoded — triggers the json.Encode error path
	err := writeGzJSON(out, make(chan int))
	require.Error(t, err)
}

func TestCompileYAMLDir_UnreadableDir(t *testing.T) {
	if os.Getuid() == 0 {
		t.Skip("permission tests do not apply when running as root")
	}
	dir := t.TempDir()
	subdir := filepath.Join(dir, "locked")
	require.NoError(t, os.Mkdir(subdir, 0o755))
	require.NoError(t, os.WriteFile(filepath.Join(subdir, "file.yaml"), []byte("x: 1\n"), 0o644))
	require.NoError(t, os.Chmod(subdir, 0o000))
	t.Cleanup(func() { _ = os.Chmod(subdir, 0o755) })

	sources := map[string]int64{}
	_, err := compileYAMLDir(subdir, sources)
	require.Error(t, err)
}
package embed

// Whitebox tests for extractFS — covers all error paths using an
// in-memory fs.FS constructed with testing/fstest.

import (
	"io/fs"
	"os"
	"path/filepath"
	"testing"
	"testing/fstest"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestExtractFS_Basic(t *testing.T) {
	memFS := fstest.MapFS{
		"root/file.yaml": {Data: []byte("x: 1\n")},
		"root/sub/a.md":  {Data: []byte("# A")},
	}
	dir := t.TempDir()
	require.NoError(t, extractFS(memFS, "root", dir))
	assert.FileExists(t, filepath.Join(dir, "file.yaml"))
	assert.FileExists(t, filepath.Join(dir, "sub", "a.md"))
}

func TestExtractFS_MkdirError(t *testing.T) {
	if os.Getuid() == 0 {
		t.Skip("permission tests do not apply when running as root")
	}
	memFS := fstest.MapFS{
		"root/sub/file.yaml": {Data: []byte("x: 1\n")},
	}
	dir := t.TempDir()
	require.NoError(t, os.Chmod(dir, 0o444))
	t.Cleanup(func() { _ = os.Chmod(dir, 0o755) })

	err := extractFS(memFS, "root", dir)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "embed:")
}

func TestExtractFS_WriteError(t *testing.T) {
	if os.Getuid() == 0 {
		t.Skip("permission tests do not apply when running as root")
	}
	memFS := fstest.MapFS{
		"root/file.yaml": {Data: []byte("x: 1\n")},
	}
	dir := t.TempDir()
	// Pre-create file.yaml as a directory so WriteFile fails with EISDIR
	require.NoError(t, os.Mkdir(filepath.Join(dir, "file.yaml"), 0o755))

	err := extractFS(memFS, "root", dir)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "embed: write")
}

func TestExtractFS_EmptyRoot(t *testing.T) {
	memFS := fstest.MapFS{
		"root": {Mode: os.ModeDir},
	}
	dir := t.TempDir()
	require.NoError(t, extractFS(memFS, "root", dir))
}

// errFS is an fs.FS that wraps a MapFS but fails ReadDir for a specific path,
// causing fs.WalkDir to deliver an error to the walk callback.
type errFS struct {
	fstest.MapFS
	failPath string
}

func (e errFS) ReadDir(name string) ([]fs.DirEntry, error) {
	if name == e.failPath {
		return nil, os.ErrPermission
	}
	return e.MapFS.ReadDir(name)
}

func TestExtractFS_WalkCallbackError(t *testing.T) {
	// errFS makes WalkDir deliver an error for "root/broken" to the callback.
	mem := errFS{
		MapFS: fstest.MapFS{
			"root/broken": {Mode: os.ModeDir},
		},
		failPath: "root/broken",
	}
	dir := t.TempDir()
	err := extractFS(mem, "root", dir)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "embed: walk")
}
Now I have a complete audit. Let me check the full cmd test and stale test to assess t.Parallel() gaps.

Bash Read stale and integration test files
IN
cat /home/sergio/dev/strategist-skill/internal/stale/stale_test.go
echo "---"
cat /home/sergio/dev/strategist-skill/tests/stale_test.go
echo "---"
cat /home/sergio/dev/strategist-skill/tests/fixtures_test.go

OUT
package stale_test

import (
	"compress/gzip"
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/SergioLacerda/strategist-skill/internal/stale"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func writeGzJSON(t *testing.T, path string, v any) {
	t.Helper()
	require.NoError(t, os.MkdirAll(filepath.Dir(path), 0o755))
	f, err := os.Create(path)
	require.NoError(t, err)
	gz := gzip.NewWriter(f)
	require.NoError(t, json.NewEncoder(gz).Encode(v))
	require.NoError(t, gz.Close())
	require.NoError(t, f.Close())
}

func TestIsStale(t *testing.T) {
	checker := stale.Checker{}

	tests := []struct {
		name      string
		setup     func(t *testing.T, dir string) string
		wantStale bool
		wantErr   bool
	}{
		{
			name: "absent artifact is stale",
			setup: func(t *testing.T, dir string) string {
				t.Helper()
				return filepath.Join(dir, "absent.gz")
			},
			wantStale: true,
		},
		{
			name: "missing manifest is stale",
			setup: func(t *testing.T, dir string) string {
				t.Helper()
				art := filepath.Join(dir, ".config.gz")
				writeGzJSON(t, art, map[string]any{"sources": map[string]int64{}})
				return art
			},
			wantStale: true,
		},
		{
			name: "fresh artifact with empty sources is not stale",
			setup: func(t *testing.T, dir string) string {
				t.Helper()
				art := filepath.Join(dir, ".config.gz")
				writeGzJSON(t, art, map[string]any{"sources": map[string]int64{}})
				writeGzJSON(t, filepath.Join(dir, ".manifest.gz"), map[string]any{})
				return art
			},
			wantStale: false,
		},
		{
			name: "fresh artifact with nil sources is not stale",
			setup: func(t *testing.T, dir string) string {
				t.Helper()
				art := filepath.Join(dir, ".config.gz")
				// No "sources" key at all — should be treated as empty
				writeGzJSON(t, art, map[string]any{})
				writeGzJSON(t, filepath.Join(dir, ".manifest.gz"), map[string]any{})
				return art
			},
			wantStale: false,
		},
		{
			name: "source newer than recorded mtime is stale",
			setup: func(t *testing.T, dir string) string {
				t.Helper()
				src := filepath.Join(dir, "active.yaml")
				require.NoError(t, os.WriteFile(src, []byte("mode: full"), 0o644))
				past := time.Now().Add(-1 * time.Hour)
				require.NoError(t, os.Chtimes(src, past, past))
				now := time.Now()
				require.NoError(t, os.Chtimes(src, now, now))
				art := filepath.Join(dir, ".config.gz")
				writeGzJSON(t, art, map[string]any{
					"sources": map[string]int64{src: past.Unix()},
				})
				writeGzJSON(t, filepath.Join(dir, ".manifest.gz"), map[string]any{})
				return art
			},
			wantStale: true,
		},
		{
			name: "source gone after compile is stale",
			setup: func(t *testing.T, dir string) string {
				t.Helper()
				src := filepath.Join(dir, "gone.yaml")
				art := filepath.Join(dir, ".config.gz")
				writeGzJSON(t, art, map[string]any{
					"sources": map[string]int64{src: time.Now().Unix()},
				})
				writeGzJSON(t, filepath.Join(dir, ".manifest.gz"), map[string]any{})
				return art
			},
			wantStale: true,
		},
		{
			name: "corrupt gzip artifact returns error",
			setup: func(t *testing.T, dir string) string {
				t.Helper()
				art := filepath.Join(dir, ".config.gz")
				require.NoError(t, os.WriteFile(art, []byte("not gzip data"), 0o644))
				writeGzJSON(t, filepath.Join(dir, ".manifest.gz"), map[string]any{})
				return art
			},
			wantStale: false,
			wantErr:   true,
		},
		{
			name: "valid gzip with invalid JSON returns error",
			setup: func(t *testing.T, dir string) string {
				t.Helper()
				art := filepath.Join(dir, ".config.gz")
				// Write valid gzip wrapping invalid JSON bytes
				f, err := os.Create(art)
				require.NoError(t, err)
				gz := gzip.NewWriter(f)
				_, err = gz.Write([]byte("{not valid json"))
				require.NoError(t, err)
				require.NoError(t, gz.Close())
				require.NoError(t, f.Close())
				writeGzJSON(t, filepath.Join(dir, ".manifest.gz"), map[string]any{})
				return art
			},
			wantStale: false,
			wantErr:   true,
		},
		{
			name: "artifact in unreadable directory returns error",
			setup: func(t *testing.T, dir string) string {
				t.Helper()
				if os.Getuid() == 0 {
					t.Skip("permission tests do not apply when running as root")
				}
				subdir := filepath.Join(dir, "locked")
				require.NoError(t, os.Mkdir(subdir, 0o755))
				art := filepath.Join(subdir, ".config.gz")
				writeGzJSON(t, art, map[string]any{"sources": map[string]int64{}})
				writeGzJSON(t, filepath.Join(subdir, ".manifest.gz"), map[string]any{})
				// Remove execute permission so stat of files inside fails with EACCES
				require.NoError(t, os.Chmod(subdir, 0o000))
				t.Cleanup(func() { _ = os.Chmod(subdir, 0o755) })
				return art
			},
			wantStale: false,
			wantErr:   true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			dir := t.TempDir()
			artifactPath := tt.setup(t, dir)
			got, err := checker.IsStale(artifactPath)
			if tt.wantErr {
				require.Error(t, err)
				return
			}
			require.NoError(t, err)
			assert.Equal(t, tt.wantStale, got)
		})
	}
}
---
package tests_test

import (
	"compress/gzip"
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/SergioLacerda/strategist-skill/internal/stale"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func writeGzJSON(t *testing.T, path string, v interface{}) {
	t.Helper()
	require.NoError(t, os.MkdirAll(filepath.Dir(path), 0o755))
	f, err := os.Create(path)
	require.NoError(t, err)
	gz := gzip.NewWriter(f)
	require.NoError(t, json.NewEncoder(gz).Encode(v))
	require.NoError(t, gz.Close())
	require.NoError(t, f.Close())
}

func TestIsStale(t *testing.T) {
	checker := stale.Checker{}

	tests := []struct {
		name     string
		setup    func(t *testing.T, dir string) string // returns artifact path
		wantStale bool
	}{
		{
			name: "absent artifact is stale",
			setup: func(t *testing.T, dir string) string {
				t.Helper()
				return filepath.Join(dir, "absent.gz")
			},
			wantStale: true,
		},
		{
			name: "missing manifest is stale",
			setup: func(t *testing.T, dir string) string {
				t.Helper()
				art := filepath.Join(dir, ".config.gz")
				writeGzJSON(t, art, map[string]interface{}{"sources": map[string]int64{}})
				// no .manifest.gz written
				return art
			},
			wantStale: true,
		},
		{
			name: "fresh artifact with empty sources is not stale",
			setup: func(t *testing.T, dir string) string {
				t.Helper()
				art := filepath.Join(dir, ".config.gz")
				writeGzJSON(t, art, map[string]interface{}{"sources": map[string]int64{}})
				writeGzJSON(t, filepath.Join(dir, ".manifest.gz"), map[string]interface{}{})
				return art
			},
			wantStale: false,
		},
		{
			name: "source newer than recorded mtime is stale",
			setup: func(t *testing.T, dir string) string {
				t.Helper()
				src := filepath.Join(dir, "active.yaml")
				require.NoError(t, os.WriteFile(src, []byte("mode: full"), 0o644))

				// Record an mtime in the past
				past := time.Now().Add(-1 * time.Hour)
				require.NoError(t, os.Chtimes(src, past, past))

				// Then touch the file to update its mtime
				now := time.Now()
				require.NoError(t, os.Chtimes(src, now, now))

				art := filepath.Join(dir, ".config.gz")
				writeGzJSON(t, art, map[string]interface{}{
					"sources": map[string]int64{src: past.Unix()},
				})
				writeGzJSON(t, filepath.Join(dir, ".manifest.gz"), map[string]interface{}{})
				return art
			},
			wantStale: true,
		},
		{
			name: "source gone after compile is stale",
			setup: func(t *testing.T, dir string) string {
				t.Helper()
				src := filepath.Join(dir, "gone.yaml")
				art := filepath.Join(dir, ".config.gz")
				writeGzJSON(t, art, map[string]interface{}{
					"sources": map[string]int64{src: time.Now().Unix()},
				})
				writeGzJSON(t, filepath.Join(dir, ".manifest.gz"), map[string]interface{}{})
				// src never created
				return art
			},
			wantStale: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			dir := t.TempDir()
			artifactPath := tt.setup(t, dir)
			got, err := checker.IsStale(artifactPath)
			require.NoError(t, err)
			assert.Equal(t, tt.wantStale, got)
		})
	}
}
---
package tests_test

import (
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gopkg.in/yaml.v3"
)

// fixture represents a single YAML test fixture file.
type fixture struct {
	Scenario      string `yaml:"scenario"`
	Description   string `yaml:"description"`
	ExpectedEvent string `yaml:"expected_event"`
}

// expectedEventPattern matches [Strategist] phase=<word> status=<word> ...
// The fixture only specifies key=value fragments; we validate the format is parseable.
var validEventFragment = regexp.MustCompile(`^\w+=\S+`)

func TestFixtures_ValidFormat(t *testing.T) {
	files, err := filepath.Glob(filepath.Join("fixtures", "*.yaml"))
	require.NoError(t, err)
	require.NotEmpty(t, files, "no fixtures found in tests/fixtures/")

	for _, f := range files {
		f := f // capture
		name := strings.TrimSuffix(filepath.Base(f), ".yaml")
		t.Run(name, func(t *testing.T) {
			data, err := os.ReadFile(f)
			require.NoError(t, err)

			var fix fixture
			require.NoError(t, yaml.Unmarshal(data, &fix), "fixture must be valid YAML")

			assert.NotEmpty(t, fix.Scenario, "fixture must have a scenario field")
			assert.NotEmpty(t, fix.ExpectedEvent, "fixture must have an expected_event field")

			// Validate that every token in expected_event matches key=value format
			for _, token := range strings.Fields(fix.ExpectedEvent) {
				assert.True(t, validEventFragment.MatchString(token),
					"expected_event token %q must match key=value", token)
			}
		})
	}
}
Let me read the full cmd test to check for global state before planning parallelism.

Read cmd_test.go
Bash Read full compile test
IN
cat /home/sergio/dev/strategist-skill/internal/compile/compile_test.go

OUT
package compile_test

import (
	"compress/gzip"
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	"github.com/SergioLacerda/strategist-skill/internal/compile"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// readGzJSON decompresses a gzipped JSON artifact into v.
func readGzJSON(t *testing.T, path string, v any) {
	t.Helper()
	f, err := os.Open(path)
	require.NoError(t, err, "open artifact %s", path)
	defer f.Close() //nolint:errcheck
	gz, err := gzip.NewReader(f)
	require.NoError(t, err, "gzip reader")
	defer gz.Close() //nolint:errcheck
	require.NoError(t, json.NewDecoder(gz).Decode(v), "json decode")
}

// minimalRoot creates a minimal .strategist/-like directory in dir.
func minimalRoot(t *testing.T, dir string) {
	t.Helper()
	require.NoError(t, os.MkdirAll(filepath.Join(dir, "personas"), 0o755))
	require.NoError(t, os.MkdirAll(filepath.Join(dir, "roles"), 0o755))
	require.NoError(t, os.WriteFile(filepath.Join(dir, "active.yaml"), []byte("mode: full\n"), 0o644))
	require.NoError(t, os.WriteFile(filepath.Join(dir, "personas", "epic.yaml"), []byte("name: Epic\n"), 0o644))
	require.NoError(t, os.WriteFile(filepath.Join(dir, "roles", "default.yaml"), []byte("name: Default\n"), 0o644))
}

// --- CompileConfig ---

func TestCompileConfig(t *testing.T) {
	tests := []struct {
		name    string
		setup   func(t *testing.T, dir string)
		wantErr bool
		check   func(t *testing.T, artifact map[string]any)
	}{
		{
			name:  "minimal valid root produces artifact",
			setup: minimalRoot,
			check: func(t *testing.T, a map[string]any) {
				assert.Equal(t, "strategist-compiled-config/1.0", a["schema"])
				assert.NotNil(t, a["compiled_at"])
				assert.NotNil(t, a["sources"])
				assert.NotNil(t, a["active"])
				personas, ok := a["personas"].(map[string]any)
				require.True(t, ok)
				assert.Contains(t, personas, "epic")
				roles, ok := a["roles"].(map[string]any)
				require.True(t, ok)
				assert.Contains(t, roles, "default")
			},
		},
		{
			name: "missing active.yaml returns error",
			setup: func(t *testing.T, _ string) {
				t.Helper()
				// no files
			},
			wantErr: true,
		},
		{
			name: "empty personas and roles dirs are valid",
			setup: func(t *testing.T, dir string) {
				t.Helper()
				require.NoError(t, os.MkdirAll(filepath.Join(dir, "personas"), 0o755))
				require.NoError(t, os.MkdirAll(filepath.Join(dir, "roles"), 0o755))
				require.NoError(t, os.WriteFile(filepath.Join(dir, "active.yaml"), []byte("mode: lite\n"), 0o644))
			},
			check: func(t *testing.T, a map[string]any) {
				assert.Equal(t, "strategist-compiled-config/1.0", a["schema"])
			},
		},
		{
			name: "non-yaml files in personas dir are ignored",
			setup: func(t *testing.T, dir string) {
				t.Helper()
				require.NoError(t, os.MkdirAll(filepath.Join(dir, "personas"), 0o755))
				require.NoError(t, os.MkdirAll(filepath.Join(dir, "roles"), 0o755))
				require.NoError(t, os.WriteFile(filepath.Join(dir, "active.yaml"), []byte("mode: full\n"), 0o644))
				require.NoError(t, os.WriteFile(filepath.Join(dir, "personas", "README.md"), []byte("# readme"), 0o644))
			},
			check: func(t *testing.T, a map[string]any) {
				personas, ok := a["personas"].(map[string]any)
				require.True(t, ok)
				assert.NotContains(t, personas, "README")
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			dir := t.TempDir()
			tt.setup(t, dir)
			out := filepath.Join(dir, ".compiled", ".config.gz")
			err := compile.Config(dir, out)
			if tt.wantErr {
				require.Error(t, err)
				return
			}
			require.NoError(t, err)
			require.FileExists(t, out)
			var artifact map[string]any
			readGzJSON(t, out, &artifact)
			if tt.check != nil {
				tt.check(t, artifact)
			}
		})
	}
}

// --- CompileDomain ---

func TestCompileDomain(t *testing.T) {
	tests := []struct {
		name    string
		setup   func(t *testing.T, dir string)
		wantErr bool
		check   func(t *testing.T, artifact map[string]any)
	}{
		{
			name: "empty load_always and load_by_task_type",
			setup: func(t *testing.T, dir string) {
				t.Helper()
				require.NoError(t, os.WriteFile(
					filepath.Join(dir, "index.yaml"),
					[]byte("load_always: []\nload_by_task_type: {}\n"),
					0o644,
				))
			},
			check: func(t *testing.T, a map[string]any) {
				assert.Equal(t, "strategist-compiled-domain/1.0", a["schema"])
				assert.NotNil(t, a["load_always"])
				assert.NotNil(t, a["load_by_task_type"])
			},
		},
		{
			name: "load_always with existing file",
			setup: func(t *testing.T, dir string) {
				t.Helper()
				require.NoError(t, os.WriteFile(filepath.Join(dir, "roles.yaml"), []byte("roles: true\n"), 0o644))
				require.NoError(t, os.WriteFile(
					filepath.Join(dir, "index.yaml"),
					[]byte("load_always:\n  - roles.yaml\nload_by_task_type: {}\n"),
					0o644,
				))
			},
			check: func(t *testing.T, a map[string]any) {
				la, ok := a["load_always"].(map[string]any)
				require.True(t, ok)
				assert.Contains(t, la, "roles.yaml")
			},
		},
		{
			name: "missing file in load_always is skipped",
			setup: func(t *testing.T, dir string) {
				t.Helper()
				require.NoError(t, os.WriteFile(
					filepath.Join(dir, "index.yaml"),
					[]byte("load_always:\n  - missing.yaml\nload_by_task_type: {}\n"),
					0o644,
				))
			},
			check: func(t *testing.T, a map[string]any) {
				la, ok := a["load_always"].(map[string]any)
				require.True(t, ok)
				assert.NotContains(t, la, "missing.yaml")
			},
		},
		{
			name: "load_by_task_type with task types",
			setup: func(t *testing.T, dir string) {
				t.Helper()
				require.NoError(t, os.WriteFile(filepath.Join(dir, "arch.yaml"), []byte("arch: true\n"), 0o644))
				require.NoError(t, os.WriteFile(
					filepath.Join(dir, "index.yaml"),
					[]byte("load_always: []\nload_by_task_type:\n  analysis:\n    - arch.yaml\n"),
					0o644,
				))
			},
			check: func(t *testing.T, a map[string]any) {
				lbtt := a["load_by_task_type"].(map[string]any)
				assert.Contains(t, lbtt, "analysis")
			},
		},
		{
			name: "missing index.yaml returns error",
			setup: func(t *testing.T, _ string) {
				t.Helper()
			},
			wantErr: true,
		},
		{
			name: "invalid YAML in index.yaml returns error",
			setup: func(t *testing.T, dir string) {
				t.Helper()
				require.NoError(t, os.WriteFile(
					filepath.Join(dir, "index.yaml"),
					[]byte("load_always: [invalid yaml: : :\n"),
					0o644,
				))
			},
			wantErr: true,
		},
		{
			name: "load_by_task_type with missing file is skipped",
			setup: func(t *testing.T, dir string) {
				t.Helper()
				require.NoError(t, os.WriteFile(
					filepath.Join(dir, "index.yaml"),
					[]byte("load_always: []\nload_by_task_type:\n  analysis:\n    - nonexistent.yaml\n"),
					0o644,
				))
			},
			check: func(t *testing.T, a map[string]any) {
				lbtt := a["load_by_task_type"].(map[string]any)
				analysis := lbtt["analysis"].(map[string]any)
				assert.Empty(t, analysis)
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			dir := t.TempDir()
			tt.setup(t, dir)
			out := filepath.Join(dir, ".compiled", ".domain.gz")
			err := compile.Domain(dir, out)
			if tt.wantErr {
				require.Error(t, err)
				return
			}
			require.NoError(t, err)
			require.FileExists(t, out)
			var artifact map[string]any
			readGzJSON(t, out, &artifact)
			if tt.check != nil {
				tt.check(t, artifact)
			}
		})
	}
}

// --- CompileIndex ---

func TestCompileIndex(t *testing.T) {
	tests := []struct {
		name    string
		content string
		wantErr bool
		check   func(t *testing.T, artifact map[string]any)
	}{
		{
			name:    "empty sources",
			content: "sources: []\n",
			check: func(t *testing.T, a map[string]any) {
				assert.Equal(t, "strategist-compiled-index/1.0", a["schema"])
				tags := a["tags"].(map[string]any)
				assert.Empty(t, tags)
			},
		},
		{
			name: "single source with tags builds inverted index",
			content: `sources:
  - id: arch-doc
    tags: [architecture, system-design]
`,
			check: func(t *testing.T, a map[string]any) {
				tags := a["tags"].(map[string]any)
				assert.Contains(t, tags, "architecture")
				assert.Contains(t, tags, "system-design")
				archIDs := tags["architecture"].([]any)
				assert.Contains(t, archIDs, "arch-doc")
			},
		},
		{
			name: "multiple sources sharing a tag",
			content: `sources:
  - id: doc-a
    tags: [shared]
  - id: doc-b
    tags: [shared, unique]
`,
			check: func(t *testing.T, a map[string]any) {
				tags := a["tags"].(map[string]any)
				sharedIDs := tags["shared"].([]any)
				assert.Len(t, sharedIDs, 2)
				assert.Contains(t, sharedIDs, "doc-a")
				assert.Contains(t, sharedIDs, "doc-b")
			},
		},
		{
			name:    "missing knowledge index file returns error",
			content: "", // will use a non-existent path
			wantErr: true,
		},
		{
			name:    "invalid YAML in knowledge index returns error",
			content: "sources: [invalid: yaml: content",
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			dir := t.TempDir()
			out := filepath.Join(dir, ".compiled", ".index.gz")

			var kiPath string
			if tt.wantErr {
				kiPath = filepath.Join(dir, "nonexistent.yaml")
			} else {
				kiPath = filepath.Join(dir, "knowledge.index.yaml")
				require.NoError(t, os.WriteFile(kiPath, []byte(tt.content), 0o644))
			}

			err := compile.Index(kiPath, out)
			if tt.wantErr {
				require.Error(t, err)
				return
			}
			require.NoError(t, err)
			require.FileExists(t, out)
			var artifact map[string]any
			readGzJSON(t, out, &artifact)
			if tt.check != nil {
				tt.check(t, artifact)
			}
		})
	}
}

// --- CompileAll ---

func TestCompileAll(t *testing.T) {
	t.Run("produces all four artifacts", func(t *testing.T) {
		dir := t.TempDir()
		minimalRoot(t, dir)
		require.NoError(t, os.WriteFile(
			filepath.Join(dir, "index.yaml"),
			[]byte("load_always: []\nload_by_task_type: {}\n"),
			0o644,
		))
		kiPath := filepath.Join(dir, "knowledge.index.yaml")
		require.NoError(t, os.WriteFile(kiPath, []byte("sources: []\n"), 0o644))

		c := compile.Compiler{}
		require.NoError(t, c.CompileAll(dir, kiPath))

		compiledDir := filepath.Join(dir, ".compiled")
		for _, name := range []string{".config.gz", ".domain.gz", ".index.gz", ".manifest.gz"} {
			assert.FileExists(t, filepath.Join(compiledDir, name))
		}
	})

	t.Run("manifest contains sha256 for all artifacts", func(t *testing.T) {
		dir := t.TempDir()
		minimalRoot(t, dir)
		require.NoError(t, os.WriteFile(
			filepath.Join(dir, "index.yaml"),
			[]byte("load_always: []\nload_by_task_type: {}\n"),
			0o644,
		))
		kiPath := filepath.Join(dir, "knowledge.index.yaml")
		require.NoError(t, os.WriteFile(kiPath, []byte("sources: []\n"), 0o644))

		require.NoError(t, compile.Compiler{}.CompileAll(dir, kiPath))

		var manifest map[string]any
		readGzJSON(t, filepath.Join(dir, ".compiled", ".manifest.gz"), &manifest)
		artifacts := manifest["artifacts"].(map[string]any)
		for _, name := range []string{".config.gz", ".domain.gz", ".index.gz"} {
			sha, ok := artifacts[name].(string)
			require.True(t, ok, "artifact %s missing from manifest", name)
			assert.Contains(t, sha, "sha256:", "artifact %s should have sha256 prefix", name)
		}
	})

	t.Run("fails if active.yaml missing — manifest not written", func(t *testing.T) {
		dir := t.TempDir()
		require.NoError(t, os.WriteFile(
			filepath.Join(dir, "index.yaml"),
			[]byte("load_always: []\nload_by_task_type: {}\n"),
			0o644,
		))
		kiPath := filepath.Join(dir, "knowledge.index.yaml")
		require.NoError(t, os.WriteFile(kiPath, []byte("sources: []\n"), 0o644))

		err := compile.Compiler{}.CompileAll(dir, kiPath)
		require.Error(t, err)
		assert.NoFileExists(t, filepath.Join(dir, ".compiled", ".manifest.gz"))
	})

	t.Run("fails if knowledge index missing", func(t *testing.T) {
		dir := t.TempDir()
		err := compile.Compiler{}.CompileAll(dir, filepath.Join(dir, "nonexistent.yaml"))
		require.Error(t, err)
		assert.ErrorContains(t, err, "index")
	})

	t.Run("fails if index.yaml missing (domain step)", func(t *testing.T) {
		dir := t.TempDir()
		// ki exists, but no index.yaml for domain compile
		kiPath := filepath.Join(dir, "knowledge.index.yaml")
		require.NoError(t, os.WriteFile(kiPath, []byte("sources: []\n"), 0o644))

		err := compile.Compiler{}.CompileAll(dir, kiPath)
		require.Error(t, err)
		assert.ErrorContains(t, err, "domain")
	})
}

// --- writeGzJSON error paths ---

func TestWriteGzJSON_ReadOnlyDir(t *testing.T) {
	if os.Getuid() == 0 {
		t.Skip("permission tests do not apply when running as root")
	}
	dir := t.TempDir()
	// Make dir read-only so creating a file inside fails
	require.NoError(t, os.Chmod(dir, 0o444))
	t.Cleanup(func() { _ = os.Chmod(dir, 0o755) })

	err := compile.Config(dir, filepath.Join(dir, "output.gz"))
	require.Error(t, err)
}

// --- loadYAMLFile / yaml parse error ---

func TestCompileConfig_InvalidActiveYAML(t *testing.T) {
	dir := t.TempDir()
	require.NoError(t, os.MkdirAll(filepath.Join(dir, "personas"), 0o755))
	require.NoError(t, os.MkdirAll(filepath.Join(dir, "roles"), 0o755))
	// Write invalid YAML
	require.NoError(t, os.WriteFile(filepath.Join(dir, "active.yaml"), []byte("mode: [unclosed"), 0o644))
	err := compile.Config(dir, filepath.Join(dir, ".compiled", ".config.gz"))
	require.Error(t, err)
	assert.ErrorContains(t, err, "compile config")
}

func TestCompileConfig_InvalidPersonaYAML(t *testing.T) {
	dir := t.TempDir()
	require.NoError(t, os.MkdirAll(filepath.Join(dir, "personas"), 0o755))
	require.NoError(t, os.MkdirAll(filepath.Join(dir, "roles"), 0o755))
	require.NoError(t, os.WriteFile(filepath.Join(dir, "active.yaml"), []byte("mode: full\n"), 0o644))
	require.NoError(t, os.WriteFile(filepath.Join(dir, "personas", "bad.yaml"), []byte(": invalid\n  yaml: here"), 0o644))
	err := compile.Config(dir, filepath.Join(dir, ".compiled", ".config.gz"))
	require.Error(t, err)
}

// --- writeGzJSON: output path is a directory (os.Create fails) ---

func TestWriteGzJSON_OutputIsDirectory(t *testing.T) {
	dir := t.TempDir()
	minimalRoot(t, dir)
	// Create a directory where the output file should go — os.Create fails
	outPath := filepath.Join(dir, ".compiled")
	require.NoError(t, os.MkdirAll(outPath, 0o755))
	// outPath itself is a directory, so CompileConfig(dir, outPath) → os.Create(outPath) fails
	err := compile.Config(dir, outPath)
	require.Error(t, err)
}

// --- compilePaths: non-ErrNotExist error from a bad YAML file ---

func TestCompileDomain_InvalidYAMLInLoadAlways(t *testing.T) {
	dir := t.TempDir()
	// Write a file that exists but has invalid YAML
	require.NoError(t, os.WriteFile(filepath.Join(dir, "bad.yaml"), []byte(": invalid\n  yaml:"), 0o644))
	require.NoError(t, os.WriteFile(
		filepath.Join(dir, "index.yaml"),
		[]byte("load_always:\n  - bad.yaml\nload_by_task_type: {}\n"),
		0o644,
	))
	err := compile.Domain(dir, filepath.Join(dir, ".compiled", ".domain.gz"))
	require.Error(t, err)
}

// --- CompileIndex: invalid YAML parse ---

func TestCompileIndex_InvalidYAML(t *testing.T) {
	dir := t.TempDir()
	kiPath := filepath.Join(dir, "knowledge.index.yaml")
	require.NoError(t, os.WriteFile(kiPath, []byte("sources: [unclosed bracket"), 0o644))
	err := compile.Index(kiPath, filepath.Join(dir, ".compiled", ".index.gz"))
	require.Error(t, err)
}
Read active_yaml_test.go
Read installer_whitebox_test.go
Full picture complete. Let me check the module name then implement all improvements.

Bash Check module name
IN
head -5 /home/sergio/dev/strategist-skill/go.mod

OUT
module github.com/SergioLacerda/strategist-skill

go 1.26.3

require (
Good — Go 1.26, so no tt := tt loop-capture needed. Now I'll implement all improvements in parallel batches:

Batch 1: t.Parallel() additions across all pure unit test files
Batch 2: New architecture test + fuzz test
Let me start with batch 1, grouping independent file edits.

Update Todos

Add t.Parallel() to internal/domain, internal/stale, internal/compile, internal/embed tests

Add t.Parallel() to internal/install tests (install_test, wizard_test, active_yaml_test, installer_whitebox_test)

Add t.Parallel() to tests/ integration package (stale_test, compile_test, install_test, fixtures_test)

Create internal/domain/architecture_test.go (domain isolation test)

Create internal/compile/fuzz_test.go (YAML parser fuzz test)

Run go test ./... to verify all changes compile and pass
Now adding t.Parallel() to the first batch of internal package tests:

Edit domain_test.go
Added 1 line
Edit domain_test.go
Added 1 line
Edit domain_test.go
Added 1 line
Edit domain_test.go
Added 1 line
Edit domain_test.go
Added 1 line
Edit domain_test.go
Added 1 line
Edit domain_test.go
Added 1 line
Edit domain_test.go
Added 1 line
Edit domain_test.go
Added 1 line
Now the stale test — adding parallel to the table loop and top-level:

Edit stale_test.go
Added 1 line
Edit stale_test.go
Added 1 line
Now the compile tests — table-driven loops and standalone tests:

Edit compile_test.go
Added 1 line
Edit compile_test.go
Added 1 line
Edit compile_test.go
Added 1 line
Edit compile_test.go
Added 1 line
Edit compile_test.go
Added 1 line
Edit compile_test.go
Added 1 line
Now TestCompileAll subtests and all the standalone compile tests:

Edit compile_test.go
Added 2 lines
Edit compile_test.go
Added 1 line
Edit compile_test.go
Added 1 line
Edit compile_test.go
Added 1 line
Edit compile_test.go
Added 1 line
Now the standalone compile tests:

Edit compile_test.go
Added 1 line
Edit compile_test.go
Added 1 line
Edit compile_test.go
Added 1 line
Edit compile_test.go
Added 1 line
Edit compile_test.go
Added 1 line
Edit compile_test.go
Added 1 line
Now the compile helpers and embed tests in parallel:

Edit helpers_test.go
Added 1 line
Edit helpers_test.go
Added 1 line
Edit helpers_test.go
Added 1 line
Edit helpers_test.go
Added 1 line
Now embed tests:

Edit embed_test.go
Edit failed
Read embed_test.go
Edit embed_test.go
Added 1 line
Edit embed_test.go
Added 2 lines
Read embed_test.go (lines 51-70)
Read embed_test.go (lines 58-97)
Edit embed_test.go
Added 1 line
Edit embed_test.go
Added 1 line
Now the extractfs whitebox tests:

Read extractfs_test.go
Edit extractfs_test.go
Added 1 line
Edit extractfs_test.go
Added 1 line
Edit extractfs_test.go
Added 1 line
Edit extractfs_test.go
Added 1 line
Edit extractfs_test.go
Added 1 line
Update Todos

Add t.Parallel() to internal/domain, internal/stale, internal/compile, internal/embed tests

Add t.Parallel() to internal/install tests (install_test, wizard_test, active_yaml_test, installer_whitebox_test)

Add t.Parallel() to tests/ integration package (stale_test, compile_test, install_test, fixtures_test)

Create internal/domain/architecture_test.go (domain isolation test)

Create internal/compile/fuzz_test.go (YAML parser fuzz test)

Run go test ./... to verify all changes compile and pass
Now the install package tests:

Edit install_test.go
Added 1 line
Edit install_test.go
Added 1 line
