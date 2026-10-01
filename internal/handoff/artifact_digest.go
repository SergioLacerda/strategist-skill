package handoff

import (
	"crypto/sha256"
	"fmt"
	"os"
)

// RangerArtifactDigest returns the content identity of the normalized Ranger
// artifact. Unlike PackageDigest it includes the complete artifact because the
// entire discovery handoff is the evidence being approved.
func RangerArtifactDigest(path string) (string, error) {
	content, err := os.ReadFile(path) //nolint:gosec // path is resolved by the mission boundary
	if err != nil {
		return "", fmt.Errorf("handoff_artifact_invalid: read Ranger artifact: %w", err)
	}
	return fmt.Sprintf("sha256:%x", sha256.Sum256(content)), nil
}
