package check

import (
	"bytes"
	"errors"
	"os"
	"path/filepath"
	"testing"
	"text/tabwriter"

	"github.com/SergioLacerda/strategist-skill/internal/domain"
	"github.com/SergioLacerda/strategist-skill/internal/runtime/env"
	"github.com/SergioLacerda/strategist-skill/internal/runtime/payload"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestRankedHostValidationBranchesArePortable(t *testing.T) {
	t.Parallel()
	root := t.TempDir()
	provider := "provider"
	validScript := "weapon-runtime/" + provider + "/openspec/openspec.mjs"
	state := domain.RankedRuntimeStateRuntime{Node: filepath.Join(root, "node"), Script: validScript}

	_, invalid := verifiedRankedBundle(root, provider, domain.RankedRuntimeStateRuntime{Node: "relative", Script: validScript})
	assert.Equal(t, domain.ReasonRankedRuntimeStateInvalid, invalid.ReasonCode)

	_, invalid = verifiedRankedBundle(root, provider, domain.RankedRuntimeStateRuntime{Node: state.Node, Script: "outside.mjs"})
	assert.Equal(t, domain.ReasonRankedRuntimeStateInvalid, invalid.ReasonCode)

	_, invalid = verifiedRankedBundle(root, provider, domain.RankedRuntimeStateRuntime{Node: state.Node, Script: validScript})
	assert.Equal(t, domain.ReasonRankedRuntimeStateInvalid, invalid.ReasonCode)

	scriptPath := filepath.Join(root, filepath.FromSlash(validScript))
	require.NoError(t, os.MkdirAll(filepath.Dir(scriptPath), 0o755))
	state.Components = []domain.RankedRuntimeStateComponent{{Name: domain.RankedRuntimeComponentOpenSpec, SHA256: "sha256:missing"}}
	require.NoError(t, os.Mkdir(scriptPath, 0o755))
	_, missing := verifiedRankedBundle(root, provider, state)
	assert.Equal(t, domain.ReasonRankedRuntimeBundleMissing, missing.ReasonCode)
	require.NoError(t, os.Remove(scriptPath))
	require.NoError(t, os.WriteFile(scriptPath, []byte("script"), 0o644))

	bundleDir := filepath.Join(root, "weapon-runtime", provider, "openspec")
	require.NoError(t, os.WriteFile(filepath.Join(bundleDir, "payload.txt"), []byte("payload"), 0o644))
	_, altered := verifiedRankedBundle(root, provider, state)
	assert.Equal(t, domain.ReasonRankedRuntimeBundleAltered, altered.ReasonCode)

	digest, _, err := payload.TreeDigest(os.DirFS(filepath.Join(root, "weapon-runtime", provider)), "openspec")
	require.NoError(t, err)
	state.Components[0].SHA256 = digest
	_, ready := verifiedRankedBundle(root, provider, state)
	assert.True(t, ready.Ready())

	runtimeRoot := t.TempDir()
	health := runHostNodeRankedRuntimeHealthcheck(root, runtimeRoot, provider, domain.WeaponRuntime{}, state)
	assert.Equal(t, domain.ReasonRankedRuntimeExecutableMissing, health.ReasonCode)
}

func TestRankedHostCommandFailureDistinguishesMissingExecutable(t *testing.T) {
	t.Parallel()
	missing := hostNodeCommandFailure("provider", "/runtime", &env.ExecutableNotFoundError{Name: "node", Err: errors.New("missing")})
	assert.Equal(t, domain.ReasonRankedRuntimeExecutableMissing, missing.ReasonCode)

	other := hostNodeCommandFailure("provider", "/runtime", errors.New("broken command"))
	assert.Equal(t, "ranked_runtime_healthcheck_failed", other.ReasonCode)
}

func TestRankedRuntimePathAndExecutableNormalization(t *testing.T) {
	t.Parallel()
	for _, provider := range []string{"", ".", "..", "nested/provider", `provider\escape`, "provider:bad"} {
		_, ok := rankedRuntimeDir(provider)
		assert.False(t, ok, provider)
	}
	for _, tc := range []struct {
		name    string
		runtime domain.WeaponRuntime
		want    string
	}{
		{name: "entrypoint", runtime: domain.WeaponRuntime{Kind: domain.RankedRuntimeExecutable, Entrypoint: "node --version"}, want: "node"},
		{name: "bootstrap", runtime: domain.WeaponRuntime{Bootstrap: "openspec init"}, want: "openspec"},
		{name: "default", runtime: domain.WeaponRuntime{}, want: "openspec"},
	} {
		t.Run(tc.name, func(t *testing.T) { assert.Equal(t, tc.want, runtimeExecutableName(tc.runtime)) })
	}
}

func TestCodexBootstrapAdvisoriesCoverDirectoryAndSeedStates(t *testing.T) {
	t.Parallel()
	root := t.TempDir()
	assert.Empty(t, codexBootstrapAdvisories(root))

	fileRoot := t.TempDir()
	require.NoError(t, os.WriteFile(filepath.Join(fileRoot, ".codex"), []byte("file"), 0o644))
	assert.Contains(t, codexBootstrapAdvisories(fileRoot)[0], "codex_bootstrap_unreadable")

	seedRoot := t.TempDir()
	require.NoError(t, os.Mkdir(filepath.Join(seedRoot, ".codex"), 0o755))
	assert.Contains(t, codexBootstrapAdvisories(seedRoot)[0], "codex_bootstrap_missing")
	require.NoError(t, os.WriteFile(filepath.Join(seedRoot, ".codex", "commands.md"), []byte("stale"), 0o644))
	assert.Contains(t, codexBootstrapAdvisories(seedRoot)[0], "codex_bootstrap_stale")
	require.NoError(t, os.WriteFile(filepath.Join(seedRoot, ".codex", "commands.md"), []byte(codexRuntimeSection+"\n"+codexGeneratedMarker), 0o644))
	assert.Empty(t, codexBootstrapAdvisories(seedRoot))
}

func TestCheckOutputRendersEmptyAndPopulatedTables(t *testing.T) {
	t.Parallel()
	var out bytes.Buffer
	w := tabwriter.NewWriter(&out, 0, 0, 3, ' ', 0)
	require.NoError(t, writeCheckReadinessSection(w, map[string]slotResolution{"discovery": {}}))
	require.NoError(t, writeCheckSlotsSection(w, map[string]string{"discovery": "provider"}, map[string]slotResolution{"discovery": {}}))
	require.NoError(t, w.Flush())
	assert.Contains(t, out.String(), "ROLE READINESS")
	assert.Contains(t, out.String(), "provider_not_resolved")
}
