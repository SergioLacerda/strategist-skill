package provider

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"
)

func fixturePath(t *testing.T) string {
	t.Helper()
	return filepath.Join("..", "embed", "defaults", "plugins", "fixtures", "minimal-provider")
}

func TestValidateMinimalFixtureSeparatesStaticAndLiveEvidence(t *testing.T) {
	report, err := Validate(fixturePath(t), "refinement")
	require.NoError(t, err)
	require.True(t, report.Validated)
	require.Equal(t, "fixture-provider", report.ProviderID)
	require.Equal(t, "ready", string(report.Readiness.Descriptor.Status))
	require.Equal(t, "unknown", string(report.LiveInvocation.Status))
	require.Equal(t, "live_probe_not_run", report.LiveInvocation.ReasonCode)
}

func TestValidateRejectsRemoteSourceWithoutMutation(t *testing.T) {
	root := t.TempDir()
	sentinel := filepath.Join(root, "active.yaml")
	require.NoError(t, os.WriteFile(sentinel, []byte("native\n"), 0o644))
	report, err := Validate("https://example.invalid/provider.git", "")
	require.Error(t, err)
	require.False(t, report.Validated)
	require.Equal(t, "remote_source_deferred", report.Reasons[0].Code)
	raw, readErr := os.ReadFile(sentinel)
	require.NoError(t, readErr)
	require.Equal(t, "native\n", string(raw))
}

func TestAddAllowsStaticDiscoveryBindingWithoutLiveInvocation(t *testing.T) {
	root := t.TempDir()
	require.NoError(t, copySource(filepath.Join("..", "embed", "defaults"), root))
	require.NoError(t, os.WriteFile(filepath.Join(root, "active.yaml"), []byte("mode: epic\nbase_path: .analysis\nslots:\n  discovery: fixture-provider\n  refinement: archivist\n  execution: sniper\n"), 0o644))
	fixture := copyFixture(t)
	require.NoError(t, os.WriteFile(filepath.Join(fixture, adapterManifestName), []byte("schema_version: strategist-plugin-adapter/v1\nid: fixture-provider\nadapter_revision: 1.0.0\nplugin_api_range: \"=1\"\nsupported_slots: [discovery]\nsupported_roles: [ranger]\nentrypoints: [host.prompt]\npackage_constraint: fixture-provider@1\nrequested_permissions: [workspace.read]\n"), 0o644))
	require.NoError(t, os.WriteFile(filepath.Join(fixture, legacyManifestName), []byte("id: fixture-provider\ncanonical_role: ranger\nsupported_slots: [discovery]\n"), 0o644))
	result, err := Add(root, fixture, "discovery")
	require.NoError(t, err)
	require.Equal(t, "complete", result.TransactionState)
	require.Equal(t, "unknown", string(result.Report.LiveInvocation.Status))
	require.Equal(t, "live_probe_not_run", result.Report.LiveInvocation.ReasonCode)
	_, statErr := os.Stat(filepath.Join(root, providerDirName, "fixture-provider@1.0.0"))
	require.NoError(t, statErr)
}

func TestValidateRejectsLegacyAuthorityConflict(t *testing.T) {
	dir := copyFixture(t)
	require.NoError(t, os.WriteFile(filepath.Join(dir, legacyManifestName), []byte("id: other-provider\ncanonical_role: archivist\n"), 0o644))

	report, err := Validate(dir, "refinement")
	require.Error(t, err)
	require.Contains(t, reasonCodes(report.Reasons), "legacy_authority_conflict")
}

func TestAddRollsBackWhenCompileFails(t *testing.T) {
	root := t.TempDir()
	require.NoError(t, os.WriteFile(filepath.Join(root, "active.yaml"), []byte("mode: custom\nbase_path: .analysis\nslots:\n  discovery: ranger\n  refinement: archivist\n  execution: sniper\n"), 0o644))

	result, err := Add(root, fixturePath(t), "refinement")
	require.Error(t, err)
	require.Equal(t, "rolled_back", result.TransactionState)
	_, statErr := os.Stat(filepath.Join(root, providerDirName, "fixture-provider@1.0.0"))
	require.ErrorIs(t, statErr, os.ErrNotExist)
	_, lockErr := os.Stat(filepath.Join(root, "plugins.lock"))
	require.ErrorIs(t, lockErr, os.ErrNotExist)
	raw, readErr := os.ReadFile(filepath.Join(root, transactionFileName))
	require.NoError(t, readErr)
	require.Contains(t, string(raw), "rolled_back")
}

func TestAddCommitsBindingAndCompilesGovernedRuntime(t *testing.T) {
	root := t.TempDir()
	require.NoError(t, copySource(filepath.Join("..", "embed", "defaults"), root))
	require.NoError(t, os.WriteFile(filepath.Join(root, "active.yaml"), []byte("mode: epic\nbase_path: .analysis\nknowledge_index_path: knowledge.index.yaml\nslots:\n  discovery: ranger\n  refinement: archivist\n  execution: sniper\n"), 0o644))

	result, err := Add(root, fixturePath(t), "refinement")
	require.NoError(t, err)
	require.Equal(t, "complete", result.TransactionState)
	require.Equal(t, int64(1), result.BindingGeneration)
	replay, replayErr := Add(root, fixturePath(t), "refinement")
	require.NoError(t, replayErr)
	require.Equal(t, result.BindingGeneration, replay.BindingGeneration)
	require.Equal(t, result.InstanceID, replay.InstanceID)
	lockRaw, readErr := os.ReadFile(filepath.Join(root, "plugins.lock"))
	require.NoError(t, readErr)
	require.Contains(t, string(lockRaw), "fixture-provider@1.0.0")
	_, compileErr := os.Stat(filepath.Join(root, ".compiled", ".manifest.gz"))
	require.NoError(t, compileErr)
}

func copyFixture(t *testing.T) string {
	t.Helper()
	target := filepath.Join(t.TempDir(), "fixture-provider")
	require.NoError(t, copySource(fixturePath(t), target))
	return target
}

func TestFailedAddPreservesActiveBindingAndExistingLock(t *testing.T) {
	root := t.TempDir()
	active := []byte("mode: custom\nbase_path: .analysis\nslots:\n  discovery: ranger\n  refinement: archivist\n  execution: sniper\n")
	lock := []byte("schema_version: strategist-plugin-lock/v1\n")
	require.NoError(t, os.WriteFile(filepath.Join(root, "active.yaml"), active, 0o644))
	require.NoError(t, os.WriteFile(filepath.Join(root, "plugins.lock"), lock, 0o644))

	_, err := Add(root, fixturePath(t), "refinement")
	require.Error(t, err)
	gotActive, readErr := os.ReadFile(filepath.Join(root, "active.yaml"))
	require.NoError(t, readErr)
	require.Equal(t, active, gotActive)
	gotLock, readErr := os.ReadFile(filepath.Join(root, "plugins.lock"))
	require.NoError(t, readErr)
	require.Contains(t, string(gotLock), "schema_version: strategist-plugin-lock/v1")
	require.NotContains(t, string(gotLock), "fixture-provider")
}
