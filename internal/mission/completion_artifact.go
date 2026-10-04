package mission

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/SergioLacerda/strategist-skill/internal/provider"
)

// DiscoveryArtifactPaths confines the canonical pending artifact to the
// workspace containing the resolved .strategist root.
func DiscoveryArtifactPaths(root, basePath, missionID string) (string, string, error) {
	workspace := filepath.Dir(root)
	absolute := filepath.Join(basePath, "pending", missionID+"-analysis.md")
	relative, err := filepath.Rel(workspace, absolute)
	if err != nil || filepath.IsAbs(relative) || relative == ".." || strings.HasPrefix(relative, ".."+string(filepath.Separator)) {
		return "", "", fmt.Errorf("role_invocation_failed: discovery artifact path escapes workspace")
	}
	return filepath.ToSlash(relative), absolute, nil
}

// WriteDiscoveryArtifact creates the target directory and publishes one
// normalized artifact atomically.
func WriteDiscoveryArtifact(path string, content []byte) error {
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return fmt.Errorf("write normalized artifact: create directory: %w", err)
	}
	if err := WriteFileAtomic(path, content, 0o644); err != nil {
		return fmt.Errorf("write normalized artifact: %w", err)
	}
	return nil
}

// RequireArtifactTargetFree refuses to overwrite an artifact whose ownership
// cannot be proven to be the current request.
func RequireArtifactTargetFree(path, requestID string) error {
	existing, err := os.ReadFile(path) //nolint:gosec // caller constructs the path inside the workspace
	if os.IsNotExist(err) {
		return nil
	}
	if err != nil {
		return fmt.Errorf("inspect existing discovery artifact: %w", err)
	}
	provenance, err := provider.ParseDiscoveryProvenance(existing)
	switch {
	case err != nil:
		return fmt.Errorf("invocation_artifact_exists: %s has unreadable frontmatter and will not be overwritten: %w", filepath.Base(path), err)
	case provenance.Status != "ranger_pending":
		return fmt.Errorf("invocation_artifact_exists: %s is not a pending Ranger artifact and will not be overwritten", filepath.Base(path))
	case provenance.RequestID == "":
		return fmt.Errorf("invocation_artifact_exists: %s is pending but carries no request provenance; remove it to issue a new request", filepath.Base(path))
	default:
		return fmt.Errorf("invocation_artifact_exists: %s is owned by request %q, not %q", filepath.Base(path), provenance.RequestID, requestID)
	}
}

// ContentDigest returns the stable digest used by the invocation journal.
func ContentDigest(content []byte) string {
	sum := sha256.Sum256(content)
	return "sha256:" + hex.EncodeToString(sum[:])
}
