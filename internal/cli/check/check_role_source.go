package check

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/SergioLacerda/strategist-skill/internal/domain"
	embedpkg "github.com/SergioLacerda/strategist-skill/internal/embed"
)

// validateRoleSourceParity verifies the installed Role directives and native
// skill manifests against the single embedded source boundary. The two files
// are checked as one projection so a valid Role contract cannot silently run
// beside a stale generated brief.
func validateRoleSourceParity(root string) []string {
	extractor := embedpkg.Extractor{}
	var errs []string
	for _, role := range domain.DefaultRoleRegistry().Roles() {
		roleID := role.ID
		artifact, err := extractor.RoleSourceArtifact(roleID)
		if err != nil {
			errs = append(errs, fmt.Sprintf("runtime_stale: embedded Role source %q unreadable: %v", roleID, err))
			continue
		}
		errs = append(errs, validateRoleSourceFile(root, artifact.RoleConfigPath, artifact.RoleConfigDigest)...)
		errs = append(errs, validateRoleSourceFile(root, artifact.SkillManifestPath, artifact.SkillManifestDigest)...)
	}
	return errs
}

func validateRoleSourceFile(root, relPath, expectedDigest string) []string {
	path := filepath.Join(root, filepath.FromSlash(relPath))
	raw, err := os.ReadFile(path) //nolint:gosec // path is derived from the selected Strategist root
	if err != nil {
		if os.IsNotExist(err) {
			return []string{fmt.Sprintf("runtime_missing: Role source file %q is missing — run strategist install", relPath)}
		}
		return []string{fmt.Sprintf("runtime_stale: read Role source file %q: %v", relPath, err)}
	}
	if domain.SHA256Hex(raw) == expectedDigest {
		return nil
	}
	return []string{fmt.Sprintf("runtime_stale: Role source file %q differs from the embedded source — run strategist install", relPath)}
}
