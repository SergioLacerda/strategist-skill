package install

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestResolveExternalSkillFailureModes(t *testing.T) {
	t.Parallel()
	_, err := resolveExternalSkill(t.TempDir())
	require.ErrorContains(t, err, "resolve external skill")

	noSidecar := t.TempDir()
	require.NoError(t, os.WriteFile(filepath.Join(noSidecar, "SKILL.md"), []byte("---\nname: bare\nmetadata:\n  version: \"1.0.0\"\n---\nbody\n"), 0o644))
	_, err = resolveExternalSkill(noSidecar)
	require.ErrorContains(t, err, "read strategist.yaml")

	scaffolded := t.TempDir()
	require.NoError(t, os.WriteFile(filepath.Join(scaffolded, "SKILL.md"), []byte("---\nname: ok-skill\nmetadata:\n  version: \"1.0.0\"\n---\nbody\n"), 0o644))
	scaffoldHostPackage(t, scaffolded, "ranger", "discovery")
	skill, err := resolveExternalSkill(scaffolded)
	require.NoError(t, err)
	assert.Equal(t, "ok-skill", skill.ID)
}

func TestVerifyUpstreamPin(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	require.NoError(t, os.WriteFile(filepath.Join(dir, "SKILL.md"), []byte("body"), 0o644))
	local, err := HashFileSHA256(filepath.Join(dir, "SKILL.md"))
	require.NoError(t, err)

	require.NoError(t, verifyUpstreamPin(dir, "x", externalSkillAdapter{}), "an absent pin is unverified, not an error")
	require.NoError(t, verifyUpstreamPin(dir, "x", externalSkillAdapter{UpstreamContentDigest: "sha256:other", LocalModifications: true}))
	require.NoError(t, verifyUpstreamPin(dir, "x", externalSkillAdapter{UpstreamContentDigest: local}))
	require.ErrorContains(t, verifyUpstreamPin(dir, "x", externalSkillAdapter{UpstreamContentDigest: "sha256:other"}), "upstream_pin_mismatch")
	require.Error(t, verifyUpstreamPin(t.TempDir(), "x", externalSkillAdapter{UpstreamContentDigest: local}))
}

func TestBuildCompiledRegistryFailureModes(t *testing.T) {
	t.Parallel()
	defaults := filepath.Join(repoRoot(), "internal", "embed", "defaults")

	missingPayload := pluginCatalog{Providers: []pluginCatalogProvider{{ID: "ghost", Version: "1.0.0", CompatibilitySource: "embedded"}}}
	_, err := buildCompiledRegistry(missingPayload, defaults)
	require.ErrorContains(t, err, "embedded_payload_missing")

	unknownRole := pluginCatalog{Providers: []pluginCatalogProvider{{
		ID: "native-ranked", Version: "1.0.0", CompatibilitySource: "native_role", Roles: []string{"no-such-role"},
		Ranked: true, CertificationDigest: "sha256:x",
	}}}
	_, err = buildCompiledRegistry(unknownRole, defaults)
	require.Error(t, err)
}
