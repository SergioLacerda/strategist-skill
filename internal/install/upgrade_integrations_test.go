package install

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/SergioLacerda/strategist-skill/internal/integration"
	"github.com/SergioLacerda/strategist-skill/internal/integration/config"
	"github.com/stretchr/testify/require"
)

// An operator's integration decision lives next to the other operator-owned
// state in the .strategist root. The upgrade only touches the embedded tree and
// manifest-tracked orphans, so the file must survive an install and a forced
// upgrade untouched (AC-16: an upgrade preserves the existing decision).
func TestUpgradePreservesTheOperatorIntegrationDecision(t *testing.T) {
	dir := t.TempDir()
	svc := upgradeTestService(map[string][]byte{"a.yaml": []byte("a")})

	file := config.File{SchemaVersion: config.SchemaVersion, Providers: map[string]config.Provider{"jev": {
		Enabled: true, Endpoint: "https://api.typesafe.ai/v1/systemone", CredentialRef: "dotenv:.env#TYPESAFE_API_KEY", Model: "jev-1.13.0",
		AllowedCapabilities: []integration.Capability{integration.CapHandoffValidate},
		DataPolicy:          config.DataPolicy{AllowedFields: []string{"objective"}, MaxStateBytes: 4096},
	}}}
	path := filepath.Join(dir, config.FileName)
	require.NoError(t, config.Save(path, file))
	before, err := os.ReadFile(path)
	require.NoError(t, err)

	plan, err := svc.PlanUpgrade(dir)
	require.NoError(t, err)
	for _, entry := range plan.Entries {
		require.NotEqual(t, config.FileName, entry.Path, "the upgrade does not even plan the operator file")
	}
	_, err = svc.ApplyUpgrade(dir, plan, true)
	require.NoError(t, err)

	after, err := os.ReadFile(path)
	require.NoError(t, err)
	require.Equal(t, string(before), string(after))
	loaded, found, err := config.Load(path)
	require.NoError(t, err)
	require.True(t, found)
	require.Equal(t, file, loaded)
}
