// Package customws builds a workspace holding a package added with the real
// `provider add`. It is a subpackage of testutil because it imports the provider
// package, which testutil must not (internal/compile tests import testutil).
package customws

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/SergioLacerda/strategist-skill/internal/domain"
	"github.com/SergioLacerda/strategist-skill/internal/embed"
	"github.com/SergioLacerda/strategist-skill/internal/provider"
	"github.com/stretchr/testify/require"
	"gopkg.in/yaml.v3"
)

// Instance is the installed instance id of the package
// Workspace adds.
const Instance = "fixture-provider@1.0.0"

// Workspace returns a .strategist root, extracted from the embedded
// defaults, into which the minimal fixture package was added with the real
// `provider add` (refinement slot), so plugins.lock, providers/ and the compiled
// runtime are exactly what the command produces. active.yaml names the package by
// its instance id, and the discovery slot carries a custom binding for
// brainstorming, so tests that walk every analysis slot have a binding for each.
func Workspace(t testing.TB) string {
	t.Helper()
	root := t.TempDir()
	require.NoError(t, embed.Extractor{}.Extract(root, true))
	require.NoError(t, os.WriteFile(filepath.Join(root, "active.yaml"), []byte("mode: epic\nbase_path: .analysis\nknowledge_index_path: knowledge.index.yaml\nslots:\n  discovery: brainstorming\n  refinement: "+Instance+"\n  execution: sniper\n"), 0o644))

	pkg := t.TempDir()
	for _, name := range []string{"package.yaml", "adapter.yaml", "skill.yaml"} {
		raw, err := embed.Extractor{}.ReadFile("plugins/fixtures/minimal-provider/" + name)
		require.NoError(t, err)
		if name == "adapter.yaml" {
			raw = append(raw, []byte("risk_score: write_analysis\n")...)
		}
		require.NoError(t, os.WriteFile(filepath.Join(pkg, name), raw, 0o644))
	}
	_, err := provider.Add(root, pkg, "refinement")
	require.NoError(t, err)
	addDiscoveryBinding(t, root)
	return root
}

func addDiscoveryBinding(t testing.TB, root string) {
	t.Helper()
	path := filepath.Join(root, "plugins.lock")
	raw, err := os.ReadFile(path) //nolint:gosec // G304: path is under the test temp root
	require.NoError(t, err)
	var lock domain.PluginLockFile
	require.NoError(t, yaml.Unmarshal(raw, &lock))
	lock.Bindings = append(lock.Bindings, domain.SlotBinding{SchemaVersion: "strategist-plugin-binding/v1", Slot: "discovery", InstalledInstanceID: "brainstorming", Generation: 1, Status: "active", Mode: domain.SlotBindingModeCustom})
	out, err := yaml.Marshal(lock)
	require.NoError(t, err)
	require.NoError(t, os.WriteFile(path, out, 0o644))
}
