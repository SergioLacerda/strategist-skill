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
	t.Parallel()
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
			t.Parallel()
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
	t.Parallel()
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
			t.Parallel()
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
	t.Parallel()
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
			t.Parallel()
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
	t.Parallel()
	t.Run("produces all four artifacts", func(t *testing.T) {
		t.Parallel()
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
		t.Parallel()
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
		t.Parallel()
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
		t.Parallel()
		dir := t.TempDir()
		err := compile.Compiler{}.CompileAll(dir, filepath.Join(dir, "nonexistent.yaml"))
		require.Error(t, err)
		assert.ErrorContains(t, err, "index")
	})

	t.Run("fails if index.yaml missing (domain step)", func(t *testing.T) {
		t.Parallel()
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
	t.Parallel()
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
	t.Parallel()
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
	t.Parallel()
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
	t.Parallel()
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
	t.Parallel()
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
	t.Parallel()
	dir := t.TempDir()
	kiPath := filepath.Join(dir, "knowledge.index.yaml")
	require.NoError(t, os.WriteFile(kiPath, []byte("sources: [unclosed bracket"), 0o644))
	err := compile.Index(kiPath, filepath.Join(dir, ".compiled", ".index.gz"))
	require.Error(t, err)
}
