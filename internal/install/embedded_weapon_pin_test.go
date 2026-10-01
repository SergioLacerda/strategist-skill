package install

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/SergioLacerda/strategist-skill/internal/domain"
	"github.com/stretchr/testify/require"
)

func appendAdapter(t *testing.T, dir, extra string) {
	t.Helper()
	path := filepath.Join(dir, "strategist.yaml")
	raw, err := os.ReadFile(path)
	require.NoError(t, err)
	require.NoError(t, os.WriteFile(path, append(raw, []byte(extra)...), 0o600))
}

func ingestPinned(t *testing.T, extra string) IngestionResult {
	t.Helper()
	root := t.TempDir()
	dir := writeExternalSkill(t, root, "pinned", "ranger", "write_analysis", nil)
	appendAdapter(t, dir, extra)
	result, err := IngestExternalSkills(root, pluginCatalog{SchemaVersion: "v1"}, domain.TrustPolicy{})
	require.NoError(t, err)
	return result
}

func TestIngestRejectsAnUpstreamPinThatDiffersFromTheLocalBytes(t *testing.T) {
	t.Parallel()

	result := ingestPinned(t, "upstream_content_digest: sha256:"+"0123456789abcdef"[:1]+repeatHex()+"\n")

	require.Empty(t, result.Ingested)
	require.Len(t, result.Rejected, 1)
	require.Contains(t, result.Rejected[0].Reason, "upstream_pin_mismatch")
	require.Contains(t, result.Rejected[0].Reason, "local_modifications")
}

func TestIngestAcceptsADifferingPinWhenLocalModificationsAreDeclared(t *testing.T) {
	t.Parallel()

	result := ingestPinned(t, "upstream_content_digest: sha256:"+repeatHex()+"\nlocal_modifications: true\n")

	require.Empty(t, result.Rejected)
	require.Len(t, result.Ingested, 1)
}

func TestIngestAcceptsAPinThatMatchesTheLocalBytes(t *testing.T) {
	t.Parallel()

	root := t.TempDir()
	dir := writeExternalSkill(t, root, "pinned", "ranger", "write_analysis", nil)
	digest, err := HashFileSHA256(filepath.Join(dir, "SKILL.md"))
	require.NoError(t, err)
	appendAdapter(t, dir, "upstream_content_digest: "+digest+"\n")

	result, err := IngestExternalSkills(root, pluginCatalog{SchemaVersion: "v1"}, domain.TrustPolicy{})

	require.NoError(t, err)
	require.Empty(t, result.Rejected)
	require.Len(t, result.Ingested, 1)
}

func repeatHex() string {
	out := ""
	for range 63 {
		out += "a"
	}
	return out
}
