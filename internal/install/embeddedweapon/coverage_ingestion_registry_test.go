package embeddedweapon

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/SergioLacerda/strategist-skill/internal/install/installtest"
	"github.com/SergioLacerda/strategist-skill/internal/install/plugincatalog"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestResolveExternalSkillFailureModes(t *testing.T) {
	t.Parallel()
	_, err := ResolveExternalSkill(t.TempDir())
	require.ErrorContains(t, err, "resolve external skill")

	noSidecar := t.TempDir()
	require.NoError(t, os.WriteFile(filepath.Join(noSidecar, "SKILL.md"), []byte("---\nname: bare\nmetadata:\n  version: \"1.0.0\"\n---\nbody\n"), 0o644))
	_, err = ResolveExternalSkill(noSidecar)
	require.ErrorContains(t, err, "read strategist.yaml")

	scaffolded := t.TempDir()
	require.NoError(t, os.WriteFile(filepath.Join(scaffolded, "SKILL.md"), []byte("---\nname: ok-skill\nmetadata:\n  version: \"1.0.0\"\n---\nbody\n"), 0o644))
	installtest.ScaffoldHostPackage(t, scaffolded, "ranger", "discovery")
	skill, err := ResolveExternalSkill(scaffolded)
	require.NoError(t, err)
	assert.Equal(t, "ok-skill", skill.ID)
}

func TestVerifyUpstreamPin(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	require.NoError(t, os.WriteFile(filepath.Join(dir, "SKILL.md"), []byte("body"), 0o644))
	local, err := plugincatalog.HashFileSHA256(filepath.Join(dir, "SKILL.md"))
	require.NoError(t, err)

	require.NoError(t, VerifyUpstreamPin(dir, "x", plugincatalog.ExternalSkillAdapter{}), "an absent pin is unverified, not an error")
	require.NoError(t, VerifyUpstreamPin(dir, "x", plugincatalog.ExternalSkillAdapter{UpstreamContentDigest: "sha256:other", LocalModifications: true}))
	require.NoError(t, VerifyUpstreamPin(dir, "x", plugincatalog.ExternalSkillAdapter{UpstreamContentDigest: local}))
	require.ErrorContains(t, VerifyUpstreamPin(dir, "x", plugincatalog.ExternalSkillAdapter{UpstreamContentDigest: "sha256:other"}), "upstream_pin_mismatch")
	require.Error(t, VerifyUpstreamPin(t.TempDir(), "x", plugincatalog.ExternalSkillAdapter{UpstreamContentDigest: local}))
}

func TestBuildCompiledRegistryFailureModes(t *testing.T) {
	t.Parallel()
	defaults := filepath.Join(RepoRoot(), "internal", "embed", "defaults")

	missingPayload := plugincatalog.PluginCatalog{Providers: []plugincatalog.Provider{{ID: "ghost", Version: "1.0.0", CompatibilitySource: "embedded"}}}
	_, err := BuildCompiledRegistry(missingPayload, defaults)
	require.ErrorContains(t, err, "embedded_payload_missing")

	unknownRole := plugincatalog.PluginCatalog{Providers: []plugincatalog.Provider{{
		ID: "native-ranked", Version: "1.0.0", CompatibilitySource: "native_role", Roles: []string{"no-such-role"},
		Ranked: true, CertificationDigest: "sha256:x",
	}}}
	_, err = BuildCompiledRegistry(unknownRole, defaults)
	require.Error(t, err)
}
