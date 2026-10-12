package handoff

import (
	"crypto/sha256"
	"fmt"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestRangerArtifactDigestReturnsContentDigest(t *testing.T) {
	path := filepath.Join(t.TempDir(), "analysis.md")
	content := []byte("normalized Ranger artifact\n")
	require.NoError(t, os.WriteFile(path, content, 0o600))

	got, err := RangerArtifactDigest(path)
	require.NoError(t, err)
	want := fmt.Sprintf("sha256:%x", sha256.Sum256(content))
	require.Equal(t, want, got)
}

func TestRangerArtifactDigestRejectsMissingArtifact(t *testing.T) {
	_, err := RangerArtifactDigest(filepath.Join(t.TempDir(), "missing.md"))
	require.ErrorContains(t, err, "handoff_artifact_invalid: read Ranger artifact")
}
