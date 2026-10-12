// Package installtest holds the test fixtures shared by the install packages'
// tests: the embedded defaults extractor, the prebuilt OpenSpec runtime bundle
// and the Ranked runtime workspace fixture. It is imported by tests only.
package installtest

import (
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"testing"

	"github.com/SergioLacerda/strategist-skill/internal/domain"
	"github.com/SergioLacerda/strategist-skill/internal/install/plugincatalog"
	"github.com/SergioLacerda/strategist-skill/internal/runtime/payload"
	"github.com/stretchr/testify/require"
	"gopkg.in/yaml.v3"
)

func init() {
	// Production startup registers the embedded FS from cmd/strategist. Package
	// tests cannot import internal/embed because its white-box tests exercise
	// install; the checked-in defaults tree is the equivalent source.
	payload.RegisterOpenSpec(os.DirFS(EmbedDefaults()))
}

// RepoRoot returns this repository's root directory, resolved from the source
// location of this file so that tests at any package depth find the same tree.
func RepoRoot() string {
	_, thisFile, _, _ := runtime.Caller(0)
	// thisFile is <root>/internal/install/installtest/installtest.go.
	return filepath.Dir(filepath.Dir(filepath.Dir(filepath.Dir(thisFile))))
}

// EmbedDefaults returns the on-disk embedded defaults tree of this repository.
func EmbedDefaults() string { return filepath.Join(RepoRoot(), "internal", "embed", "defaults") }

// OpenSpecProposePayloadDir is the versioned skills/ directory of the embedded
// openspec-propose Weapon (ADR-0061 Decision 11).
const OpenSpecProposePayloadDir = "openspec-propose@1.0"

// DefaultsExtractor serves files from the embedded defaults tree on disk.
type DefaultsExtractor struct{}

// Extract is a no-op: the fixture only reads.
func (DefaultsExtractor) Extract(_ string, _ bool) error { return nil }

// ReadFile reads relPath from the embedded defaults tree.
func (DefaultsExtractor) ReadFile(relPath string) ([]byte, error) {
	path := filepath.Join(EmbedDefaults(), filepath.FromSlash(relPath))
	data, err := os.ReadFile(path) //nolint:gosec // test fixture path is repo-local
	if err != nil {
		return nil, fmt.Errorf("read defaults fixture %s: %w", relPath, err)
	}
	return data, nil
}

// SkipIfPermissionTestUnsupported skips chmod-based permission tests where they
// cannot be reliable: on Windows and when running as root.
func SkipIfPermissionTestUnsupported(t *testing.T) {
	t.Helper()
	if runtime.GOOS == "windows" {
		t.Skip("chmod-based permission tests are not reliable on windows")
	}
	if os.Getuid() == 0 {
		t.Skip("permission tests do not apply when running as root")
	}
}

// CopyOpenSpecRuntimeFixture supplies the same prebuilt bundle that the
// production extractor embeds, without making install tests import embed.
func CopyOpenSpecRuntimeFixture(targetDir string) error {
	source := filepath.Join(EmbedDefaults(), "skills", OpenSpecProposePayloadDir, "runtime")
	destination := filepath.Join(targetDir, "skills", OpenSpecProposePayloadDir, "runtime")
	return filepath.Walk(source, func(path string, info os.FileInfo, walkErr error) error {
		return copyRuntimeFixtureEntry(source, destination, path, info, walkErr)
	})
}

func copyRuntimeFixtureEntry(source, destination, path string, info os.FileInfo, walkErr error) error {
	if walkErr != nil {
		return walkErr
	}
	rel, err := filepath.Rel(source, path)
	if err != nil {
		return fmt.Errorf("relativize %s: %w", path, err)
	}
	out := filepath.Join(destination, rel)
	if info.IsDir() {
		return os.MkdirAll(out, 0o755)
	}
	data, err := os.ReadFile(path) //nolint:gosec // test fixture path is repo-local
	if err != nil {
		return fmt.Errorf("read %s: %w", path, err)
	}
	return os.WriteFile(out, data, 0o644)
}

// WriteRankedRuntimeFixture lays out a workspace whose single Ranked binding
// for slot resolves to provider with the given runtime.
func WriteRankedRuntimeFixture(t *testing.T, root string, provider string, slot string, runtime domain.WeaponRuntime) {
	t.Helper()
	strategist := filepath.Join(root, ".strategist")
	require.NoError(t, CopyOpenSpecRuntimeFixture(strategist))
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
	catalog := plugincatalog.PluginCatalog{SchemaVersion: "strategist-plugin-catalog/v2", Providers: []plugincatalog.Provider{{
		ID: provider, RiskScore: "write_analysis", Ranked: true, CertificationDigest: "sha256:runtime-test", Runtime: runtime,
	}}}
	catalogRaw, err := yaml.Marshal(catalog)
	require.NoError(t, err)
	require.NoError(t, os.WriteFile(filepath.Join(strategist, "plugins", "catalog.yaml"), catalogRaw, 0o644))
}
