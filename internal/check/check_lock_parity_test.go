package check

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/SergioLacerda/strategist-skill/internal/domain"
	"github.com/SergioLacerda/strategist-skill/internal/tools/resolver"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gopkg.in/yaml.v3"
)

// pluginLockSchemaVersion mirrors the resolver's lock schema (unexported,
// internal/tools/resolver/resolver.go), which VerifyLockDigest requires an exact
// match against.
const pluginLockSchemaVersion = "strategist-plugin-lock/v1"

func writePluginLockFixture(t *testing.T, root, refinementID string) {
	t.Helper()
	content := "schema_version: strategist-plugin-lock-file/v1\n" +
		"bindings:\n" +
		"  - slot: discovery\n" +
		"    installed_instance_id: ranger\n" +
		"  - slot: refinement\n" +
		"    installed_instance_id: " + refinementID + "\n"
	require.NoError(t, os.WriteFile(filepath.Join(root, "plugins.lock"), []byte(content), 0o644))
}

func TestCheckPluginLockParityReportsDivergence(t *testing.T) {
	t.Parallel()
	root := t.TempDir()
	writePluginLockFixture(t, root, "archivist")

	errs := checkPluginLockParity(root, map[string]string{
		"discovery":  "ranger",
		"refinement": "openspec-propose",
		"execution":  "sniper",
	})

	require.Len(t, errs, 1)
	assert.Contains(t, errs[0], "slot refinement")
	assert.Contains(t, errs[0], "openspec-propose")
	assert.Contains(t, errs[0], "archivist")
}

func TestCheckPluginLockParityNoLockFileReturnsNil(t *testing.T) {
	t.Parallel()
	root := t.TempDir() // no plugins.lock written

	errs := checkPluginLockParity(root, map[string]string{
		"discovery":  "ranger",
		"refinement": "archivist",
		"execution":  "sniper",
	})

	assert.Nil(t, errs)
}

func TestCheckPluginLockParityMalformedLockFileReturnsNil(t *testing.T) {
	t.Parallel()
	root := t.TempDir()
	require.NoError(t, os.WriteFile(filepath.Join(root, "plugins.lock"), []byte("bindings: [unterminated"), 0o644))

	errs := checkPluginLockParity(root, map[string]string{
		"discovery":  "ranger",
		"refinement": "archivist",
		"execution":  "sniper",
	})

	assert.Nil(t, errs)
}

func TestCheckPluginLockParityAcceptsMatchingBindings(t *testing.T) {
	t.Parallel()
	root := t.TempDir()
	writePluginLockFixture(t, root, "archivist")

	errs := checkPluginLockParity(root, map[string]string{
		"discovery":  "ranger",
		"refinement": "archivist",
		"execution":  "sniper",
	})

	assert.Empty(t, errs)
}

// A custom package's binding is reconciled by naming its instance in active.yaml,
// not by re-running install or compile, which cannot know the operator's choice.
func TestCheckPluginLockParityForACustomPackageNamesTheInstance(t *testing.T) {
	t.Parallel()
	root := t.TempDir()
	content := "schema_version: strategist-plugin-lock-file/v1\nbindings:\n  - slot: refinement\n    installed_instance_id: fixture-provider@1.0.0\n    mode: custom\n"
	require.NoError(t, os.WriteFile(filepath.Join(root, "plugins.lock"), []byte(content), 0o644))

	errs := checkPluginLockParity(root, map[string]string{"refinement": "openspec-propose"})

	require.Len(t, errs, 1)
	assert.Contains(t, errs[0], "slots.refinement")
	assert.Contains(t, errs[0], "fixture-provider@1.0.0")
	assert.NotContains(t, errs[0], "re-run `strategist install` or `strategist compile`")
}

// A plugins.lock without a resolved lock: block (the common case — only the
// Custom pipeline's provider migration populates it) is not treated as
// corruption; the digest check is silently skipped.
func TestCheckPluginLockParitySkipsDigestCheckWhenLockBlockAbsent(t *testing.T) {
	t.Parallel()
	root := t.TempDir()
	writePluginLockFixture(t, root, "archivist")

	errs := checkPluginLockParity(root, map[string]string{
		"discovery": "ranger", "refinement": "archivist", "execution": "sniper",
	})
	assert.Empty(t, errs)
}

func TestCheckPluginLockParityAcceptsConsistentLockDigest(t *testing.T) {
	t.Parallel()
	root := t.TempDir()
	nodes := []domain.PluginLockNode{{ID: "brainstorming", Kind: "adapter_contract", Digest: "sha256:aaa"}}
	digest := resolver.DigestLockNodes(nodes)
	writePluginLockWithLockBlock(t, root, domain.PluginLock{SchemaVersion: pluginLockSchemaVersion, GraphDigest: digest, Nodes: nodes})

	errs := checkPluginLockParity(root, map[string]string{})
	assert.Empty(t, errs, "a lock whose graph_digest matches its nodes must not be reported")
}

func TestCheckPluginLockParityReportsHandEditedLockDigest(t *testing.T) {
	t.Parallel()
	root := t.TempDir()
	nodes := []domain.PluginLockNode{{ID: "brainstorming", Kind: "adapter_contract", Digest: "sha256:aaa"}}
	// graph_digest deliberately does not match a fresh digest of nodes,
	// simulating a hand edit of one without recomputing the other.
	writePluginLockWithLockBlock(t, root, domain.PluginLock{SchemaVersion: pluginLockSchemaVersion, GraphDigest: "sha256:stale", Nodes: nodes})

	errs := checkPluginLockParity(root, map[string]string{})
	require.Len(t, errs, 1)
	assert.Contains(t, errs[0], "lock_graph_digest_mismatch")
	assert.Contains(t, errs[0], "hand-edited or corrupted")
}

func writePluginLockWithLockBlock(t *testing.T, root string, lock domain.PluginLock) {
	t.Helper()
	file := domain.PluginLockFile{SchemaVersion: "strategist-plugin-lock-file/v1", Lock: lock}
	raw, err := yaml.Marshal(file)
	require.NoError(t, err)
	require.NoError(t, os.WriteFile(filepath.Join(root, "plugins.lock"), raw, 0o644))
}

func TestCheckPluginLockParityKeepsTheReconcileHintForOtherBindings(t *testing.T) {
	t.Parallel()
	root := t.TempDir()
	content := "schema_version: strategist-plugin-lock-file/v1\nbindings:\n  - slot: refinement\n    installed_instance_id: archivist\n    mode: ranked\n"
	require.NoError(t, os.WriteFile(filepath.Join(root, "plugins.lock"), []byte(content), 0o644))

	errs := checkPluginLockParity(root, map[string]string{"refinement": "openspec-propose"})

	require.Len(t, errs, 1)
	assert.Contains(t, errs[0], "re-run `strategist install` or `strategist compile`")
}
