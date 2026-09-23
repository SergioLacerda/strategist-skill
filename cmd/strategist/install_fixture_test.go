package main

import (
	"os"
	"path/filepath"
	"testing"

	installadapter "github.com/SergioLacerda/strategist-skill/cmd/strategist/install"
	"github.com/SergioLacerda/strategist-skill/internal/compile"
	"github.com/SergioLacerda/strategist-skill/internal/domain"
	embedpkg "github.com/SergioLacerda/strategist-skill/internal/embed"
	internalinstall "github.com/SergioLacerda/strategist-skill/internal/install"
	"github.com/spf13/cobra"
	"github.com/stretchr/testify/require"
)

func newInstallTestCommand(t *testing.T, extractor domain.FileExtractor, flags map[string]string) *cobra.Command {
	t.Helper()
	cmd := installadapter.New(installTestDependencies(extractor))
	for name, value := range flags {
		require.NoError(t, cmd.Flags().Set(name, value))
	}
	return cmd
}

func installTestDependencies(extractor domain.FileExtractor) installadapter.Dependencies {
	deps := installDependencies()
	if extractor == nil {
		return deps
	}
	deps.ServiceFactory = func(shimHome string) installadapter.Installer {
		svc := internalinstall.Service{
			Extractor:          extractor,
			Compiler:           compile.Compiler{},
			ShimHomeDir:        shimHome,
			AwarenessRefresher: refreshAgentAwarenessFromEmbed,
			Version:            Version,
		}
		if lister, ok := extractor.(domain.FileLister); ok {
			svc.Lister = lister
		}
		return svc
	}
	return deps
}

// minimalInstallExtractor creates the minimum .strategist/ layout a silent
// install needs, instead of extracting the full embedded default tree
// (~170 files/~3MB, plus the ranked OpenSpec runtime bootstrap it triggers
// for the default refinement provider — see the analysis this fixture is
// built from: .analysis/refined/20260921-slow-test-packages-diagnosis/).
//
// Its epic-standalone template deliberately omits `slots:` (unlike the real
// embedded one), so activateSilentRoleProviderBindings resolves every slot
// to "" and never promotes a provider to a Ranked binding — no plugins.lock
// entry is written, so prepareRankedProviderRuntimes finds nothing to
// bootstrap. This mirrors internal/install's own minimalExtractor test
// double (internal/install/installer_whitebox_helpers_test.go), proven
// across that package's ~350 tests to avoid the ranked-bootstrap cost this
// way; only tests that explicitly opt into a ranked binding there pay it.
//
// Use only for cmd/strategist tests that don't care about installed file
// content/count (a printed banner, a resolved target, a specific error
// path) — never for a test asserting something about the real default tree
// itself (see upgrade_test.go's installedTempDir, which intentionally does
// NOT use a fixture for exactly that reason).
type minimalInstallExtractor struct{}

var (
	_ domain.FileExtractor = minimalInstallExtractor{}
	_ domain.FileLister    = minimalInstallExtractor{}
)

// minimalEpicStandaloneYAML intentionally has no `slots:` key — see the
// type doc comment above for why that's what keeps this fixture cheap.
const minimalEpicStandaloneYAML = "mode: epic\nbase_path: .analysis\n"

// minimalInstallExtractorContent is the single source of truth for this
// fixture: Extract, ReadFile, and AllPaths all derive from it, so the
// three-way merge path (which calls ReadFile for every AllPaths entry) never
// sees a path it can't also read. leveling.yaml is resolved separately
// (real embedded content — see realEmbeddedLevelingYAML) since it must
// parse as leveling.Policy, a schema this fixture doesn't hand-roll.
func minimalInstallExtractorContent() (map[string]string, error) {
	leveling, err := realEmbeddedLevelingYAML()
	if err != nil {
		return nil, err
	}
	return map[string]string{
		"SKILL.md":                            "# SKILL\n",
		"knowledge.index.yaml":                "sources: []\n",
		"treasure-chests.yaml":                "chests: []\n",
		"index.yaml":                          "load_always: []\nload_by_task_type: {}\n",
		"templates/pragmatic-standalone.yaml": "mode: pragmatic\nbase_path: .analysis\n",
		"templates/epic-standalone.yaml":      minimalEpicStandaloneYAML,
		"plugins/catalog.yaml":                "schema_version: strategist-plugin-catalog/v1\nproviders: []\n",
		"leveling.yaml":                       string(leveling),
	}, nil
}

// realEmbeddedLevelingYAML reads the real leveling.yaml (not a synthetic
// stand-in): it's parsed into leveling.Policy, a real schema this fixture
// doesn't attempt to hand-roll.
func realEmbeddedLevelingYAML() ([]byte, error) {
	return embedpkg.Extractor{}.ReadFile("leveling.yaml")
}

func (minimalInstallExtractor) AllPaths() ([]string, error) {
	content, err := minimalInstallExtractorContent()
	if err != nil {
		return nil, err
	}
	paths := make([]string, 0, len(content))
	for rel := range content {
		paths = append(paths, rel)
	}
	return paths, nil
}

func (minimalInstallExtractor) Extract(targetDir string, _ bool) error {
	for _, dir := range []string{"personas", "roles", "templates", "memory", "plugins"} {
		if err := os.MkdirAll(filepath.Join(targetDir, dir), 0o755); err != nil {
			return err
		}
	}
	content, err := minimalInstallExtractorContent()
	if err != nil {
		return err
	}
	for rel, data := range content {
		if err := os.WriteFile(filepath.Join(targetDir, rel), []byte(data), 0o644); err != nil {
			return err
		}
	}
	return nil
}

func (minimalInstallExtractor) ReadFile(relPath string) ([]byte, error) {
	content, err := minimalInstallExtractorContent()
	if err != nil {
		return nil, err
	}
	if data, ok := content[relPath]; ok {
		return []byte(data), nil
	}
	// Every other normative path (role/skill manifests, etc.) — the install
	// code validates these exist and are readable, not their exact content,
	// for a plain silent install with no ranked binding. Mirrors
	// internal/install's own minimalExtractor test double.
	for _, file := range domain.NormativeRuntimeDefaultFiles() {
		if relPath == file.Path {
			return []byte(relPath + "\n"), nil
		}
	}
	return nil, &os.PathError{Op: "open", Path: relPath, Err: os.ErrNotExist}
}
