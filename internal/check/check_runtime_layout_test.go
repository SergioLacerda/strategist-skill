package check

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	"github.com/SergioLacerda/strategist-skill/internal/domain"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func writeLayoutManifest(t *testing.T, root string, generation int) {
	t.Helper()
	raw, err := json.Marshal(domain.InstallManifest{Schema: "strategist.install-manifest.v1", RuntimeLayoutGeneration: generation})
	require.NoError(t, err)
	require.NoError(t, os.WriteFile(filepath.Join(root, domain.InstallManifestRelPath), raw, 0o644))
}

// A runtime laid out by a newer binary is reported, never blocked: the binary
// trusts its own layout, and the warning is distinct from the blocking, hash-based
// runtime_newer_than_binary.
func TestLayoutSkewAdvisoryWhenTheRuntimeIsNewerThanTheBinary(t *testing.T) {
	root := t.TempDir()
	writeLayoutManifest(t, root, domain.RuntimeLayoutGeneration+1)

	advisories := layoutSkewAdvisories(root)

	require.Len(t, advisories, 1)
	assert.Contains(t, advisories[0], "reason=runtime_layout_newer_than_binary")
	assert.Contains(t, advisories[0], "runtime_generation=2")
	assert.NotContains(t, advisories[0], "reason=runtime_newer_than_binary", "a different reason code from the blocking hash-based check")
}

func TestLayoutSkewAdvisoryIsSilentWhenTheLayoutMatchesOrIsOlder(t *testing.T) {
	root := t.TempDir()
	for _, generation := range []int{domain.RuntimeLayoutGeneration, domain.RuntimeLayoutGeneration - 1, 0} {
		writeLayoutManifest(t, root, generation)
		assert.Empty(t, layoutSkewAdvisories(root), "generation %d", generation)
	}
}

func TestLayoutSkewAdvisoryTreatsAnAbsentFieldOrManifestAsGenerationZero(t *testing.T) {
	root := t.TempDir()
	assert.Empty(t, layoutSkewAdvisories(root), "no manifest at all")

	require.NoError(t, os.WriteFile(filepath.Join(root, domain.InstallManifestRelPath), []byte(`{"schema":"strategist.install-manifest.v1"}`), 0o644))
	assert.Empty(t, layoutSkewAdvisories(root), "a manifest that predates the marker")

	require.NoError(t, os.WriteFile(filepath.Join(root, domain.InstallManifestRelPath), []byte("not json"), 0o644))
	assert.Empty(t, layoutSkewAdvisories(root), "an unreadable manifest is reported elsewhere, never as skew")
}

// The advisory never flips the preflight status or the exit code.
func TestCheckCmd_JSON_LayoutSkewIsAdvisoryOnly(t *testing.T) {
	resetCheckFlags(t)
	dir := minimalCheckRoot(t)
	writeLayoutManifest(t, dir, domain.RuntimeLayoutGeneration+1)
	checkRoot = dir
	checkJSON = true

	out := captureStdout(t, func() {
		require.NoError(t, checkCmd.RunE(checkCmd, nil))
	})

	var result domain.PreflightResult
	require.NoError(t, json.Unmarshal([]byte(out), &result))
	assert.Equal(t, "ready", result.Status)
	require.NotEmpty(t, result.Warnings)
	assert.Contains(t, result.Warnings[len(result.Warnings)-1], "reason=runtime_layout_newer_than_binary")
}

// A permission requested by the catalog entry (or an adapter) reaches the grant
// readiness without any compat view, and is blocked until a grant exists.
func TestGrantReadinessSeesPermissionsRequestedByTheCatalogEntry(t *testing.T) {
	root := t.TempDir()
	require.NoError(t, os.MkdirAll(filepath.Join(root, "plugins"), 0o755))
	require.NoError(t, os.WriteFile(filepath.Join(root, "plugins", "catalog.yaml"), []byte("schema_version: strategist-plugin-catalog/v2\nproviders:\n  - id: asks\n    risk_score: write_analysis\n    requested_permissions: [docs.write]\n"), 0o644))

	requested := requestedPermissions(root, "asks")

	require.Equal(t, []domain.PluginPermission{"docs.write"}, requested)
	check := skillProviderPermissionGrantReadinessFor(root, "sha256:1111111111111111111111111111111111111111111111111111111111111111", requested)
	assert.Equal(t, domain.ReadinessBlocked, check.Status)
	assert.Equal(t, "permission_grant_missing", check.ReasonCode)
	assert.Empty(t, requestedPermissions(root, "unknown"), "an unresolved provider requests nothing here")
}
