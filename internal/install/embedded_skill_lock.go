package install

import (
	"fmt"
	"os"
	"strings"

	"gopkg.in/yaml.v3"
)

// WriteCatalogAndMirrors writes the merged catalog, mirrors, and provenance lock.
func WriteCatalogAndMirrors(result IngestionResult, defaultsRoot, catalogPath, lockPath string) error {
	if err := writeCatalog(result.Catalog, catalogPath); err != nil {
		return err
	}
	if err := writeSkillMirrors(result.Catalog, result.Ingested, defaultsRoot); err != nil {
		return err
	}
	skills, err := withNormalizedSkillDigests(result.Catalog, result.Ingested)
	if err != nil {
		return err
	}
	return writeEmbeddedSkillLock(skills, lockPath)
}

func validateEmbeddedSkillLockBytes(raw []byte) error {
	var lock EmbeddedSkillLock
	if err := yaml.Unmarshal(raw, &lock); err != nil {
		return fmt.Errorf("validate embedded skill lock: parse: %w", err)
	}
	if err := validateEmbeddedSkillLockSchema(lock); err != nil {
		return err
	}
	for _, pkg := range lock.Packages {
		if err := validateEmbeddedSkillLockPackage(pkg); err != nil {
			return err
		}
	}
	return nil
}

func validateEmbeddedSkillLockSchema(lock EmbeddedSkillLock) error {
	if lock.SchemaVersion != "strategist-embedded-skill-lock/v1" {
		return fmt.Errorf("validate embedded skill lock: unsupported schema %q", lock.SchemaVersion)
	}
	return nil
}

func validateEmbeddedSkillLockPackage(pkg EmbeddedSkillLockNode) error {
	if incompleteLockProvenance(pkg) {
		return fmt.Errorf("validate embedded skill lock: incomplete provenance for %q", pkg.ID)
	}
	if lockDigestsMismatch(pkg) {
		return fmt.Errorf("validate embedded skill lock: digest mismatch for %q", pkg.ID)
	}
	if pkg.Transformation == "" || pkg.VerificationState == "" || pkg.OriginalDigestEvidence == "" || pkg.NormalizedDigestEvidence == "" {
		return fmt.Errorf("validate embedded skill lock: missing evidence state for %q", pkg.ID)
	}
	if !validLockEvidenceState(pkg.VerificationState) || !validLockEvidenceState(pkg.OriginalDigestEvidence) || !validLockEvidenceState(pkg.NormalizedDigestEvidence) {
		return fmt.Errorf("validate embedded skill lock: invalid evidence state for %q", pkg.ID)
	}
	return nil
}

func incompleteLockProvenance(pkg EmbeddedSkillLockNode) bool {
	return pkg.ID == "" || pkg.ContractVersion == "" || pkg.Digest == "" || pkg.OriginalDigest == "" || pkg.NormalizedDigest == ""
}

func lockDigestsMismatch(pkg EmbeddedSkillLockNode) bool {
	return !validLockDigest(pkg.Digest) || !validLockDigest(pkg.OriginalDigest) || !validLockDigest(pkg.NormalizedDigest) || pkg.OriginalDigest != pkg.Digest
}

func validLockDigest(digest string) bool {
	if !strings.HasPrefix(digest, "sha256:") || len(digest) != len("sha256:")+64 {
		return false
	}
	for _, char := range digest[len("sha256:"):] {
		if !isLowerHex(char) {
			return false
		}
	}
	return true
}

func isLowerHex(char rune) bool {
	return (char >= '0' && char <= '9') || (char >= 'a' && char <= 'f')
}

func validLockEvidenceState(state string) bool {
	switch state {
	case "declared", "verified", "unknown", "unsupported", "failed", "blocked":
		return true
	default:
		return false
	}
}

func writeCatalog(catalog pluginCatalog, path string) error {
	data, err := yaml.Marshal(catalog)
	if err != nil {
		return fmt.Errorf("marshal catalog: %w", err)
	}
	if err := os.WriteFile(path, data, 0o644); err != nil { //nolint:gosec // G306: generated catalog is not sensitive
		return fmt.Errorf("write %s: %w", path, err)
	}
	return nil
}

func writeSkillMirrors(catalog pluginCatalog, skills []IngestedSkill, defaultsRoot string) error {
	for _, skill := range skills {
		if err := writeSkillMirror(catalog, skill, defaultsRoot); err != nil {
			return err
		}
	}
	return nil
}

func writeEmbeddedSkillLock(skills []IngestedSkill, path string) error {
	data, err := yaml.Marshal(LockFromIngestedSkills(skills))
	if err != nil {
		return fmt.Errorf("marshal lock: %w", err)
	}
	if err := validateEmbeddedSkillLockBytes(data); err != nil {
		return err
	}
	if err := os.WriteFile(path, data, 0o644); err != nil { //nolint:gosec // G306: generated lock is not sensitive
		return fmt.Errorf("write %s: %w", path, err)
	}
	return nil
}
