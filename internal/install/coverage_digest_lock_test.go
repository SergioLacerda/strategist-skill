package install

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/SergioLacerda/strategist-skill/internal/domain"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gopkg.in/yaml.v3"
)

func TestNormalizedSkillDigestIsStableAndIgnoresTheSourceManifest(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	require.NoError(t, os.MkdirAll(filepath.Join(dir, "sub"), 0o755))
	require.NoError(t, os.WriteFile(filepath.Join(dir, "SKILL.md"), []byte("body"), 0o644))
	require.NoError(t, os.WriteFile(filepath.Join(dir, "sub", "ref.md"), []byte("ref"), 0o644))
	require.NoError(t, os.WriteFile(filepath.Join(dir, "skill.yaml"), []byte("source manifest"), 0o644))

	first, err := normalizedSkillDigest(dir, []byte("m"))
	require.NoError(t, err)
	require.NoError(t, os.WriteFile(filepath.Join(dir, "skill.yaml"), []byte("changed"), 0o644))
	second, err := normalizedSkillDigest(dir, []byte("m"))
	require.NoError(t, err)
	assert.Equal(t, first, second)
	other, err := normalizedSkillDigest(dir, []byte("other manifest"))
	require.NoError(t, err)
	assert.NotEqual(t, first, other)

	empty, err := normalizedSkillDigest("", []byte("m"))
	require.NoError(t, err)
	assert.Contains(t, empty, "sha256:")
}

func TestNormalizedSkillDigestErrors(t *testing.T) {
	t.Parallel()
	_, err := normalizedSkillDigest(filepath.Join(t.TempDir(), "absent"), nil)
	require.ErrorContains(t, err, "walk")

	catalog := pluginCatalog{}
	_, err = withNormalizedSkillDigests(catalog, []IngestedSkill{{ID: "ghost"}})
	require.ErrorContains(t, err, "generate normalized package evidence for ghost")
}

func validLockNode() EmbeddedSkillLockNode {
	digest := "sha256:" + strings.Repeat("a", 64)
	return EmbeddedSkillLockNode{
		ID: "x", Version: "1", Origin: string(domain.WeaponOriginEmbedded), RuntimeKind: string(domain.RankedRuntimeEmbedded), Digest: digest, OriginalDigest: digest,
		NormalizedDigest: digest, ContractVersion: "1", Transformation: "none", VerificationState: "verified",
		OriginalDigestEvidence: "verified", NormalizedDigestEvidence: "declared",
	}
}

func lockBytes(t *testing.T, schema string, nodes ...EmbeddedSkillLockNode) []byte {
	t.Helper()
	raw, err := yaml.Marshal(EmbeddedSkillLock{SchemaVersion: schema, Packages: nodes})
	require.NoError(t, err)
	return raw
}

func TestValidateEmbeddedSkillLockBytes(t *testing.T) {
	t.Parallel()
	require.NoError(t, validateEmbeddedSkillLockBytes(lockBytes(t, embeddedSkillLockSchemaVersion, validLockNode())))
	require.ErrorContains(t, validateEmbeddedSkillLockBytes([]byte("packages: [unclosed")), "parse")
	require.ErrorContains(t, validateEmbeddedSkillLockBytes(lockBytes(t, "old/v0")), "unsupported schema")

	mutate := func(fn func(*EmbeddedSkillLockNode)) []byte {
		node := validLockNode()
		fn(&node)
		return lockBytes(t, embeddedSkillLockSchemaVersion, node)
	}
	cases := map[string]func(*EmbeddedSkillLockNode){
		"incomplete provenance": func(n *EmbeddedSkillLockNode) { n.ID = "" },
		"digest mismatch":       func(n *EmbeddedSkillLockNode) { n.OriginalDigest = "sha256:" + strings.Repeat("b", 64) },
		"bad digest shape":      func(n *EmbeddedSkillLockNode) { n.Digest, n.OriginalDigest = "sha256:XYZ", "sha256:XYZ" },
		"uppercase hex":         func(n *EmbeddedSkillLockNode) { n.NormalizedDigest = "sha256:" + strings.Repeat("A", 64) },
		"missing evidence":      func(n *EmbeddedSkillLockNode) { n.Transformation = "" },
		"invalid evidence":      func(n *EmbeddedSkillLockNode) { n.VerificationState = "maybe" },
		"bad origin":            func(n *EmbeddedSkillLockNode) { n.Origin = "nowhere" },
		"bad runtime":           func(n *EmbeddedSkillLockNode) { n.RuntimeKind = "nowhere" },
	}
	for name, fn := range cases {
		require.Error(t, validateEmbeddedSkillLockBytes(mutate(fn)), name)
	}
}

func TestWriteCatalogAndEmbeddedLockFailureModes(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	require.ErrorContains(t, writeCatalog(pluginCatalog{}, filepath.Join(dir, "absent", "catalog.yaml")), "write")
	require.ErrorContains(t, writeEmbeddedSkillLock(nil, filepath.Join(dir, "absent", "lock.yaml")), "write")
	require.NoError(t, writeCatalog(pluginCatalog{}, filepath.Join(dir, "catalog.yaml")))
	require.NoError(t, writeEmbeddedSkillLock(nil, filepath.Join(dir, "lock.yaml")))
	require.Error(t, WriteCatalogAndMirrors(IngestionResult{}, dir, filepath.Join(dir, "absent", "c.yaml"), filepath.Join(dir, "l.yaml")))
	require.NoError(t, WriteCatalogAndMirrors(IngestionResult{}, dir, filepath.Join(dir, "c.yaml"), filepath.Join(dir, "l.yaml")))
	assert.FileExists(t, filepath.Join(dir, "l.yaml"))
}
