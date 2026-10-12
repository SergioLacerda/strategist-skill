package check

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	embedpkg "github.com/SergioLacerda/strategist-skill/internal/embed"
	"github.com/stretchr/testify/require"
)

func driftRoot(t *testing.T, mutate func(string) string) string {
	t.Helper()
	root := t.TempDir()
	raw, err := embedpkg.Extractor{}.ReadFile("plugins/catalog.yaml")
	require.NoError(t, err)
	content := string(raw)
	if mutate != nil {
		content = mutate(content)
	}
	require.NoError(t, os.MkdirAll(filepath.Join(root, "plugins"), 0o755))
	require.NoError(t, os.WriteFile(filepath.Join(root, "plugins", "catalog.yaml"), []byte(content), 0o600))
	return root
}

// shipped reads the catalog compiled into the binary, bypassing TestMain's stub.
func shipped() ([]byte, error) { return embedpkg.Extractor{}.ReadFile("plugins/catalog.yaml") }

func TestRegistryDriftAdvisoriesIsSilentWhenTheWorkspaceMatchesTheBinary(t *testing.T) {
	require.Empty(t, registryDriftAdvisoriesAgainst(driftRoot(t, nil), shipped))
}

func TestRegistryDriftAdvisoriesReportsADivergentRegistry(t *testing.T) {
	root := driftRoot(t, func(c string) string {
		at := strings.Index(c, "ranked_bindings:")
		return c[:at] + strings.Replace(c[at:], "generation: 1", "generation: 2", 1)
	})

	got := registryDriftAdvisoriesAgainst(root, shipped)

	require.Len(t, got, 1)
	require.Contains(t, got[0], "reason=compiled_registry_drift")
	require.Contains(t, got[0], "strategist upgrade")
}

func TestRegistryDriftAdvisoriesReportsAnUnparseableCatalog(t *testing.T) {
	root := driftRoot(t, func(string) string { return "not: [valid" })

	got := registryDriftAdvisoriesAgainst(root, shipped)

	require.Len(t, got, 1)
	require.Contains(t, got[0], "compiled_registry_unreadable")
}

func TestRegistryDriftAdvisoriesIgnoresAMissingCatalog(t *testing.T) {
	require.Empty(t, registryDriftAdvisoriesAgainst(t.TempDir(), shipped))
}
