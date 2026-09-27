package plugins

import (
	"bytes"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func writeResolvedDigestCatalog(t *testing.T) string {
	t.Helper()
	path := filepath.Join(t.TempDir(), "catalog.yaml")
	body := "schema_version: strategist-plugin-catalog/v2\nproviders:\n" +
		"  - id: brainstorming\n    risk_score: write_analysis\n    upstream_content_digest: sha256:74edf03ea6d24ef53db48677b93558d14a979bdf052ca3f57ecdca0c66791608\n" +
		"  - id: sniper\n    risk_score: controlled\n"
	require.NoError(t, os.WriteFile(path, []byte(body), 0o644))
	return path
}

func TestRunResolvedDigest_TrueCopyMatches(t *testing.T) {
	t.Parallel()
	var out bytes.Buffer
	err := RunResolvedDigest(&out, ResolvedDigestOptions{
		Catalog: writeResolvedDigestCatalog(t), Provider: "brainstorming",
		Digest: "sha256:74edf03ea6d24ef53db48677b93558d14a979bdf052ca3f57ecdca0c66791608",
	})
	require.NoError(t, err)
	assert.Contains(t, out.String(), "status=match")
}

func TestRunResolvedDigest_AlteredCopyReportsMismatchAndFails(t *testing.T) {
	t.Parallel()
	var out bytes.Buffer
	err := RunResolvedDigest(&out, ResolvedDigestOptions{
		Catalog: writeResolvedDigestCatalog(t), Provider: "brainstorming", Digest: "sha256:0000000000000000000000000000000000000000000000000000000000000000",
	})
	require.ErrorContains(t, err, "mismatch")
	assert.Contains(t, out.String(), "status=mismatch")
}

func TestRunResolvedDigest_UnavailablePinReportsUnknownNeverMatch(t *testing.T) {
	t.Parallel()
	var out bytes.Buffer
	err := RunResolvedDigest(&out, ResolvedDigestOptions{Catalog: writeResolvedDigestCatalog(t), Provider: "sniper", Digest: "sha256:anything"})
	require.NoError(t, err)
	assert.Contains(t, out.String(), "status=pin_unavailable")
}

func TestRunResolvedDigest_HashesFileWhenGiven(t *testing.T) {
	t.Parallel()
	file := filepath.Join(t.TempDir(), "SKILL.md")
	require.NoError(t, os.WriteFile(file, []byte("hello"), 0o644))
	var out bytes.Buffer
	err := RunResolvedDigest(&out, ResolvedDigestOptions{Catalog: writeResolvedDigestCatalog(t), Provider: "brainstorming", File: file})
	require.ErrorContains(t, err, "mismatch")
	assert.Contains(t, out.String(), "resolved=sha256:2cf24dba5fb0a30e26e83b2ac5b9e29e1b161e5c1fa7425e73043362938b9824")
}

func TestRunResolvedDigest_RejectsBothDigestAndFile(t *testing.T) {
	t.Parallel()
	var out bytes.Buffer
	err := RunResolvedDigest(&out, ResolvedDigestOptions{Catalog: writeResolvedDigestCatalog(t), Provider: "brainstorming", Digest: "sha256:x", File: "some.md"})
	require.ErrorContains(t, err, "mutually exclusive")
}
