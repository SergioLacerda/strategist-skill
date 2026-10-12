package check

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/SergioLacerda/strategist-skill/internal/domain"
	"github.com/SergioLacerda/strategist-skill/internal/testutil/customws"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gopkg.in/yaml.v3"
)

func TestCheckAcceptsAPackageAddedByTheRealProviderAdd(t *testing.T) {
	root := customws.Workspace(t)

	res, errMsg := resolveSlotProvider(root, "refinement", customws.Instance)

	require.Empty(t, errMsg)
	assert.Equal(t, slotResolutionSkillProvider, res.kind)
}

// Evidence that a real provider add produced, then tampered with on disk,
// fails closed with reinstall guidance and is never repaired.
func TestCheckRejectsTamperedOrIncompleteCustomEvidence(t *testing.T) {
	lockCases := map[string]func(*domain.PluginLockFile){
		"missing binding digest": func(l *domain.PluginLockFile) { l.Bindings[refinementIndex(l)].BindingDigest = "" },
		"missing weapon digest":  func(l *domain.PluginLockFile) { l.Bindings[refinementIndex(l)].WeaponDigest = "" },
		"tampered runtime":       func(l *domain.PluginLockFile) { l.Bindings[refinementIndex(l)].RuntimeKind = "executable" },
		"tampered connector":     func(l *domain.PluginLockFile) { l.Bindings[refinementIndex(l)].ConnectorID = "host" },
		"tampered package node": func(l *domain.PluginLockFile) {
			for i := range l.Lock.Nodes {
				if l.Lock.Nodes[i].Kind == string(domain.PluginResourcePackage) {
					l.Lock.Nodes[i].Digest = "sha256:tampered"
				}
			}
		},
	}
	for name, mutate := range lockCases {
		root := customws.Workspace(t)
		path := filepath.Join(root, "plugins.lock")
		raw, err := os.ReadFile(path)
		require.NoError(t, err)
		var lock domain.PluginLockFile
		require.NoError(t, yaml.Unmarshal(raw, &lock))
		mutate(&lock)
		tampered, err := yaml.Marshal(lock)
		require.NoError(t, err)
		require.NoError(t, os.WriteFile(path, tampered, 0o644))

		_, errMsg := resolveSlotProvider(root, "refinement", customws.Instance)

		assert.Contains(t, errMsg, "custom_binding_invalid", name)
		assert.Contains(t, errMsg, "re-add the package", name)
		after, readErr := os.ReadFile(path)
		require.NoError(t, readErr)
		assert.Equal(t, tampered, after, "%s: check never rewrites the lock", name)
	}
}

func TestCheckRejectsAnInstalledAdapterThatNoLongerMatchesTheLock(t *testing.T) {
	root := customws.Workspace(t)
	adapter := filepath.Join(root, "providers", customws.Instance, "adapter.yaml")
	raw, err := os.ReadFile(adapter)
	require.NoError(t, err)
	require.NoError(t, os.WriteFile(adapter, append(raw, []byte("# edited after install\n")...), 0o644))

	_, errMsg := resolveSlotProvider(root, "refinement", customws.Instance)

	assert.Contains(t, errMsg, "adapter digest mismatch")
	assert.Contains(t, errMsg, "re-add the package")
}

func TestCheckRejectsAnInstalledPackageWhoseIdentityDiffersFromTheBinding(t *testing.T) {
	root := customws.Workspace(t)
	pkg := filepath.Join(root, "providers", customws.Instance, "package.yaml")
	raw, err := os.ReadFile(pkg)
	require.NoError(t, err)
	var manifest map[string]any
	require.NoError(t, yaml.Unmarshal(raw, &manifest))
	manifest["version"] = "9.9.9"
	edited, err := yaml.Marshal(manifest)
	require.NoError(t, err)
	require.NoError(t, os.WriteFile(pkg, edited, 0o644))

	_, errMsg := resolveSlotProvider(root, "refinement", customws.Instance)

	assert.Contains(t, errMsg, "identifies")
}

func refinementIndex(lock *domain.PluginLockFile) int {
	for i, binding := range lock.Bindings {
		if binding.Slot == "refinement" {
			return i
		}
	}
	panic("no refinement binding")
}
