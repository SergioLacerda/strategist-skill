package install

import (
	"context"
	"crypto/sha256"
	"fmt"
	"os"
	"path/filepath"
	"testing"

	"github.com/SergioLacerda/strategist-skill/internal/domain"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type runtimeDefaultsExtractor struct {
	files map[string][]byte
}

func newRuntimeDefaultsExtractor(overrides map[string]string) runtimeDefaultsExtractor {
	files := map[string][]byte{
		"templates/epic-standalone.yaml": []byte("mode: epic\nbase_path: .analysis\n"),
		"knowledge.index.yaml":           []byte("sources: []\n"),
		"treasure-chests.yaml":           []byte("chests: []\n"),
		"index.yaml":                     []byte("load_always: []\nload_by_task_type: {}\n"),
		"personas/epic.yaml":             []byte("id: epic\ntone_directive: test\nphase_labels:\n  discovery: Ranger\n  refinement: Archivist\n  execution: Sniper\ndiagnostics:\n  pipeline_header: test\n  bootstrap_origin: test\n"),
		"roles/default.yaml":             []byte("discovery: brainstorming\nrefinement: openspec-explore\nexecution: sniper\n"),
	}
	for _, file := range domain.NormativeRuntimeDefaultFiles() {
		files[file.Path] = []byte(file.Path + " v1\n")
	}
	for path, content := range overrides {
		files[path] = []byte(content)
	}
	return runtimeDefaultsExtractor{files: files}
}

func (e runtimeDefaultsExtractor) Extract(targetDir string, force bool) error {
	for rel, data := range e.files {
		if err := e.writeFile(targetDir, rel, data, force); err != nil {
			return err
		}
	}
	return nil
}

func (e runtimeDefaultsExtractor) writeFile(targetDir, rel string, data []byte, force bool) error {
	dst := filepath.Join(targetDir, filepath.FromSlash(rel))
	if err := os.MkdirAll(filepath.Dir(dst), 0o755); err != nil {
		return err
	}
	if !force && testFileDiffers(dst, data) {
		return nil
	}
	return os.WriteFile(dst, data, 0o644)
}

func testFileDiffers(path string, data []byte) bool {
	existing, err := os.ReadFile(path)
	return err == nil && sha256.Sum256(existing) != sha256.Sum256(data)
}

func (e runtimeDefaultsExtractor) ReadFile(relPath string) ([]byte, error) {
	data, ok := e.files[relPath]
	if !ok {
		return nil, fmt.Errorf("runtimeDefaultsExtractor: missing %s", relPath)
	}
	return data, nil
}

func runtimeDefaultService(ext runtimeDefaultsExtractor) Service {
	return Service{
		Extractor: ext,
		Compiler:  nopCompiler{},
		Version:   "test",
	}
}

func TestInstall_AutoUpgradesOldNormativeDefault(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	oldExtractor := newRuntimeDefaultsExtractor(nil)
	newExtractor := newRuntimeDefaultsExtractor(map[string]string{
		"contracts/machine/preflight.yaml": "contracts/machine/preflight.yaml v2\n",
	})

	require.NoError(t, runtimeDefaultService(oldExtractor).Install(context.Background(), domain.InstallConfig{
		Target: dir,
		Silent: true,
		NoShim: true,
	}))
	require.NoError(t, runtimeDefaultService(newExtractor).Install(context.Background(), domain.InstallConfig{
		Target: dir,
		Silent: true,
		NoShim: true,
	}))

	got, err := os.ReadFile(filepath.Join(dir, ".strategist", "contracts", "machine", "preflight.yaml"))
	require.NoError(t, err)
	assert.Equal(t, "contracts/machine/preflight.yaml v2\n", string(got))
	assert.FileExists(t, filepath.Join(dir, ".strategist", domain.InstallManifestRelPath))
}

func TestInstall_BlocksLocalNormativeEdit(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	oldExtractor := newRuntimeDefaultsExtractor(nil)
	newExtractor := newRuntimeDefaultsExtractor(map[string]string{
		"contracts/machine/preflight.yaml": "contracts/machine/preflight.yaml v2\n",
	})

	require.NoError(t, runtimeDefaultService(oldExtractor).Install(context.Background(), domain.InstallConfig{
		Target: dir,
		Silent: true,
		NoShim: true,
	}))
	preflight := filepath.Join(dir, ".strategist", "contracts", "machine", "preflight.yaml")
	require.NoError(t, os.WriteFile(preflight, []byte("local edit\n"), 0o644))

	err := runtimeDefaultService(newExtractor).Install(context.Background(), domain.InstallConfig{
		Target: dir,
		Silent: true,
		NoShim: true,
	})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "runtime_stale_conflict")

	got, readErr := os.ReadFile(preflight)
	require.NoError(t, readErr)
	assert.Equal(t, "local edit\n", string(got))
}

func TestInstall_PreservesUserOwnedActiveYAML(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	oldExtractor := newRuntimeDefaultsExtractor(nil)
	newExtractor := newRuntimeDefaultsExtractor(map[string]string{
		"contracts/machine/preflight.yaml": "contracts/machine/preflight.yaml v2\n",
	})

	require.NoError(t, runtimeDefaultService(oldExtractor).Install(context.Background(), domain.InstallConfig{
		Target: dir,
		Silent: true,
		NoShim: true,
	}))
	active := filepath.Join(dir, ".strategist", "active.yaml")
	require.NoError(t, os.WriteFile(active, []byte("mode: custom\nbase_path: .custom\n"), 0o644))

	require.NoError(t, runtimeDefaultService(newExtractor).Install(context.Background(), domain.InstallConfig{
		Target: dir,
		Silent: true,
		NoShim: true,
	}))
	got, err := os.ReadFile(active)
	require.NoError(t, err)
	assert.Equal(t, "mode: custom\nbase_path: .custom\n", string(got))
}

func TestInstall_BlocksUnknownManifestForStaleNormativeFile(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	strategistDir := filepath.Join(dir, ".strategist")
	require.NoError(t, os.MkdirAll(filepath.Join(strategistDir, "contracts", "machine"), 0o755))
	require.NoError(t, os.WriteFile(
		filepath.Join(strategistDir, "contracts", "machine", "preflight.yaml"),
		[]byte("old unknown default\n"),
		0o644,
	))

	err := runtimeDefaultService(newRuntimeDefaultsExtractor(map[string]string{
		"contracts/machine/preflight.yaml": "new default\n",
	})).Install(context.Background(), domain.InstallConfig{
		Target: dir,
		Silent: true,
		NoShim: true,
	})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "runtime_stale_unknown_manifest")
}
