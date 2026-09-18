package install

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"testing"

	"github.com/SergioLacerda/strategist-skill/internal/domain"
	"github.com/stretchr/testify/require"
	"gopkg.in/yaml.v3"
)

func writeRankedRuntimeFixture(t *testing.T, root string, provider string, slot string, runtime domain.RankedRuntimeContract) {
	t.Helper()
	strategist := filepath.Join(root, ".strategist")
	require.NoError(t, os.MkdirAll(filepath.Join(strategist, "roles"), 0o755))
	require.NoError(t, os.MkdirAll(filepath.Join(strategist, "plugins"), 0o755))
	roleMap := []byte("discovery: ranger\nrefinement: archivist\nexecution: sniper\n")
	require.NoError(t, os.WriteFile(filepath.Join(strategist, "roles", "default.yaml"), roleMap, 0o644))
	lock := domain.PluginLockFile{
		Bindings: []domain.SlotBinding{{Slot: slot, InstalledInstanceID: provider, Mode: domain.SlotBindingModeRanked, Generation: 1, Status: "active"}},
	}
	lockRaw, err := yaml.Marshal(lock)
	require.NoError(t, err)
	require.NoError(t, os.WriteFile(filepath.Join(strategist, "plugins.lock"), lockRaw, 0o644))
	catalog := pluginCatalog{SchemaVersion: "strategist-plugin-catalog/v1", Providers: []pluginCatalogProvider{{
		ID: provider, RiskScore: "write_analysis", Ranked: true, CertificationDigest: "sha256:runtime-test", Runtime: runtime,
	}}}
	catalogRaw, err := yaml.Marshal(catalog)
	require.NoError(t, err)
	require.NoError(t, os.WriteFile(filepath.Join(strategist, "plugins", "catalog.yaml"), catalogRaw, 0o644))
}

func TestPrepareRankedProviderRuntimes_RangerNoneDoesNotCreateRuntime(t *testing.T) {
	dir := t.TempDir()
	writeRankedRuntimeFixture(t, dir, "brainstorming", "discovery", domain.RankedRuntimeContract{Kind: domain.RankedRuntimeNone})

	require.NoError(t, prepareRankedProviderRuntimes(context.Background(), filepath.Join(dir, ".strategist")))
	_, err := os.Stat(filepath.Join(dir, ".strategist", rankedRuntimeStatePath))
	require.ErrorIs(t, err, os.ErrNotExist)
}

func TestPrepareRankedProviderRuntimes_ArchivistBootstrapsOpenSpecRoot(t *testing.T) {
	dir := t.TempDir()
	writeRankedRuntimeFixture(t, dir, "openspec-propose", "refinement", domain.RankedRuntimeContract{
		Kind: domain.RankedRuntimeOpenSpecRoot, Root: ".strategist/openspec", Bootstrap: "openspec init --profile core --tools codex", Healthcheck: "openspec context --json",
	})
	original := runRankedRuntimeCommand
	t.Cleanup(func() { runRankedRuntimeCommand = original })
	runRankedRuntimeCommand = func(_ context.Context, commandDir string, name string, args ...string) ([]byte, error) {
		require.Equal(t, "openspec", name)
		if args[0] == "init" {
			require.Equal(t, []string{"init", "--profile", "core", "--tools", "codex"}, args)
			require.Equal(t, filepath.Join(dir, ".strategist"), commandDir)
			require.NoError(t, os.WriteFile(filepath.Join(commandDir, "openspec", "config.yaml"), []byte("schema: spec-driven\n"), 0o644))
		} else {
			require.Equal(t, filepath.Join(dir, ".strategist", "openspec"), commandDir)
			require.Equal(t, []string{"context", "--json"}, args)
		}
		return []byte(`{"root":{"path":"` + filepath.ToSlash(filepath.Join(dir, ".strategist")) + `"},"status":[]}`), nil
	}

	require.NoError(t, prepareRankedProviderRuntimes(context.Background(), filepath.Join(dir, ".strategist")))
	require.FileExists(t, filepath.Join(dir, ".strategist", "openspec", "config.yaml"))
	require.FileExists(t, filepath.Join(dir, ".strategist", rankedRuntimeStatePath))
	state, err := os.ReadFile(filepath.Join(dir, ".strategist", rankedRuntimeStatePath))
	require.NoError(t, err)
	require.Contains(t, string(state), "openspec-propose")
}

func TestPrepareRankedProviderRuntimes_RemovesEmptyLegacyNestedRoot(t *testing.T) {
	dir := t.TempDir()
	writeRankedRuntimeFixture(t, dir, "openspec-propose", "refinement", domain.RankedRuntimeContract{
		Kind: domain.RankedRuntimeOpenSpecRoot, Root: ".strategist/openspec", Bootstrap: "openspec init --profile core --tools codex", Healthcheck: "openspec context --json",
	})
	root := filepath.Join(dir, ".strategist", "openspec")
	require.NoError(t, os.MkdirAll(filepath.Join(root, "openspec", "specs"), 0o755))
	require.NoError(t, os.WriteFile(filepath.Join(root, "openspec", "config.yaml"), []byte("schema: spec-driven\n"), 0o644))
	require.NoError(t, os.WriteFile(filepath.Join(root, "openspec", "specs", ".gitkeep"), nil, 0o644))
	original := runRankedRuntimeCommand
	t.Cleanup(func() { runRankedRuntimeCommand = original })
	runRankedRuntimeCommand = func(_ context.Context, commandDir string, _ string, args ...string) ([]byte, error) {
		if args[0] == "init" {
			require.NoError(t, os.WriteFile(filepath.Join(commandDir, "openspec", "config.yaml"), []byte("schema: spec-driven\n"), 0o644))
		}
		return []byte(`{"root":{"path":"` + filepath.ToSlash(filepath.Join(dir, ".strategist")) + `"},"status":[]}`), nil
	}

	require.NoError(t, prepareRankedProviderRuntimes(context.Background(), filepath.Join(dir, ".strategist")))
	_, err := os.Stat(filepath.Join(root, "openspec"))
	require.ErrorIs(t, err, os.ErrNotExist)
}

func TestPrepareRankedProviderRuntimes_RejectsOpenSpecSemanticRootMismatch(t *testing.T) {
	dir := t.TempDir()
	writeRankedRuntimeFixture(t, dir, "openspec-propose", "refinement", domain.RankedRuntimeContract{
		Kind: domain.RankedRuntimeOpenSpecRoot, Root: ".strategist/openspec", Bootstrap: "openspec init --profile core --tools codex", Healthcheck: "openspec context --json",
	})
	original := runRankedRuntimeCommand
	t.Cleanup(func() { runRankedRuntimeCommand = original })
	runRankedRuntimeCommand = func(_ context.Context, _ string, _ string, args ...string) ([]byte, error) {
		if args[0] == "init" {
			require.NoError(t, os.WriteFile(filepath.Join(dir, ".strategist", "openspec", "config.yaml"), []byte("schema: spec-driven\n"), 0o644))
		}
		return []byte(`{"root":{"path":"` + filepath.ToSlash(filepath.Join(dir, ".strategist", "openspec")) + `"}}`), nil
	}

	err := prepareRankedProviderRuntimes(context.Background(), filepath.Join(dir, ".strategist"))
	require.Error(t, err)
	require.ErrorContains(t, err, "semantic root mismatch")
}

func TestPrepareRankedProviderRuntimes_RejectsMalformedHealthcheck(t *testing.T) {
	dir := t.TempDir()
	writeRankedRuntimeFixture(t, dir, "openspec-propose", "refinement", domain.RankedRuntimeContract{
		Kind: domain.RankedRuntimeOpenSpecRoot, Root: ".strategist/openspec", Bootstrap: "openspec init", Healthcheck: "openspec context --json",
	})
	original := runRankedRuntimeCommand
	t.Cleanup(func() { runRankedRuntimeCommand = original })
	setRankedRuntimeCommandForTest(t, func(_ context.Context, commandDir, _ string, args []string) ([]byte, error) {
		if args[0] == "init" {
			require.NoError(t, os.WriteFile(filepath.Join(commandDir, "openspec", "config.yaml"), []byte("schema: spec-driven\n"), 0o644))
		}
		return []byte("not-json"), nil
	})

	err := prepareRankedProviderRuntimes(context.Background(), filepath.Join(dir, ".strategist"))
	require.ErrorContains(t, err, "parse OpenSpec context")
}

func TestPrepareRankedProviderRuntimes_RejectsFailedHealthcheck(t *testing.T) {
	dir := t.TempDir()
	writeRankedRuntimeFixture(t, dir, "openspec-propose", "refinement", domain.RankedRuntimeContract{
		Kind: domain.RankedRuntimeOpenSpecRoot, Root: ".strategist/openspec", Bootstrap: "openspec init", Healthcheck: "openspec context --json",
	})
	original := runRankedRuntimeCommand
	t.Cleanup(func() { runRankedRuntimeCommand = original })
	setRankedRuntimeCommandForTest(t, func(_ context.Context, commandDir, _ string, args []string) ([]byte, error) {
		if args[0] == "init" {
			require.NoError(t, os.WriteFile(filepath.Join(commandDir, "openspec", "config.yaml"), []byte("schema: spec-driven\n"), 0o644))
			return nil, nil
		}
		return nil, errors.New("provider unavailable")
	})

	err := prepareRankedProviderRuntimes(context.Background(), filepath.Join(dir, ".strategist"))
	require.ErrorContains(t, err, "healthcheck failed")
}

func TestPrepareRankedProviderRuntimes_RejectsUnexpectedLegacyContent(t *testing.T) {
	dir := t.TempDir()
	writeRankedRuntimeFixture(t, dir, "openspec-propose", "refinement", domain.RankedRuntimeContract{
		Kind: domain.RankedRuntimeOpenSpecRoot, Root: ".strategist/openspec", Bootstrap: "openspec init", Healthcheck: "openspec context --json",
	})
	root := filepath.Join(dir, ".strategist", "openspec")
	require.NoError(t, os.MkdirAll(filepath.Join(root, "openspec"), 0o755))
	require.NoError(t, os.WriteFile(filepath.Join(root, "openspec", "config.yaml"), []byte("schema: spec-driven\n"), 0o644))
	require.NoError(t, os.WriteFile(filepath.Join(root, "openspec", "user-note.md"), []byte("keep me\n"), 0o644))
	original := runRankedRuntimeCommand
	t.Cleanup(func() { runRankedRuntimeCommand = original })
	setRankedRuntimeCommandForTest(t, func(_ context.Context, commandDir, _ string, args []string) ([]byte, error) {
		if args[0] == "init" {
			require.NoError(t, os.WriteFile(filepath.Join(commandDir, "openspec", "config.yaml"), []byte("schema: spec-driven\n"), 0o644))
		}
		return []byte(`{"root":{"path":"` + filepath.ToSlash(filepath.Join(dir, ".strategist")) + `"}}`), nil
	})

	err := prepareRankedProviderRuntimes(context.Background(), filepath.Join(dir, ".strategist"))
	require.ErrorContains(t, err, "contains user content")
}

func setRankedRuntimeCommandForTest(t *testing.T, fn func(context.Context, string, string, []string) ([]byte, error)) {
	t.Helper()
	runRankedRuntimeCommand = func(ctx context.Context, commandDir, name string, args ...string) ([]byte, error) {
		return fn(ctx, commandDir, name, args)
	}
}
