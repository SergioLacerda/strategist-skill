// Package embed provides the embedded default Strategist skill files and the extractor to install them.
package embed

import (
	"crypto/sha256"
	"embed"
	"fmt"
	"io/fs"
	"sort"
	"strings"

	"github.com/SergioLacerda/strategist-skill/internal/domain"
)

//go:embed all:defaults
var defaultsFS embed.FS

// Extractor implements domain.FileExtractor using the embedded defaults.
type Extractor struct{}

// unsafePayloadSegment reports a Weapon id or version that is empty or could
// escape the flat skills/<id>@<version>/ directory.
func unsafePayloadSegment(segment string) bool {
	return strings.TrimSpace(segment) == "" || strings.ContainsAny(segment, "/\\") || segment == "." || segment == ".."
}

// ReadEmbeddedWeaponPayload returns the canonical embedded SKILL.md bytes of one
// Weapon version (skills/<id>@<version>/) and their raw SHA-256 digest. It never consults a filesystem path or host skill
// loader at runtime.
func (e Extractor) ReadEmbeddedWeaponPayload(weaponID, version string) ([]byte, string, error) {
	if unsafePayloadSegment(weaponID) || strings.Contains(weaponID, "@") {
		return nil, "", fmt.Errorf("embed: invalid Weapon id %q", weaponID)
	}
	if unsafePayloadSegment(version) {
		return nil, "", fmt.Errorf("embed: invalid Weapon version %q for %q", version, weaponID)
	}
	data, err := e.ReadFile(strings.Join([]string{"skills", domain.WeaponPayloadDirName(weaponID, version), "SKILL.md"}, "/"))
	if err != nil {
		return nil, "", err
	}
	sum := sha256.Sum256(data)
	return data, fmt.Sprintf("sha256:%x", sum), nil
}

// ReadEmbeddedNativeRolePayload returns the canonical SKILL.md bytes for a
// native Role provider and their raw SHA-256 digest. Native Roles are compiled
// as Ranked embedded Weapons, but their payload authority lives under
// internal_skills/<role>/ rather than the versioned external skill mirror.
func (e Extractor) ReadEmbeddedNativeRolePayload(roleID string) ([]byte, string, error) {
	if unsafePayloadSegment(roleID) || strings.Contains(roleID, "@") {
		return nil, "", fmt.Errorf("embed: invalid native Role id %q", roleID)
	}
	data, err := e.ReadFile(strings.Join([]string{"internal_skills", roleID, "SKILL.md"}, "/"))
	if err != nil {
		return nil, "", err
	}
	sum := sha256.Sum256(data)
	return data, fmt.Sprintf("sha256:%x", sum), nil
}

// LevelingPolicyRequired marks the production embedded extractor as requiring
// a valid LEVELING policy identity during install and upgrade. Test doubles
// may intentionally omit the policy when exercising unrelated installer
// behavior, but production extraction must fail closed.
func (e Extractor) LevelingPolicyRequired() bool { return true }

// Extract copies all embedded defaults into targetDir, preserving the directory
// structure but stripping the leading "defaults/" path prefix.
//
// When force is false (merge mode), files that already exist on disk and whose
// content differs from the embedded default are skipped — the user's customizations
// are preserved. Files that match the embedded default are overwritten (idempotent).
// When force is true, all files are overwritten unconditionally.
func (e Extractor) Extract(targetDir string, force bool) error {
	return extractFS(defaultsFS, "defaults", targetDir, force)
}

// ReadFile reads a single file from the embedded default FS without touching disk.
// relPath is relative to the defaults root (e.g. "templates/epic-standalone.yaml").
func (e Extractor) ReadFile(relPath string) ([]byte, error) {
	data, err := fs.ReadFile(defaultsFS, "defaults/"+relPath)
	if err != nil {
		return nil, fmt.Errorf("embed: read %s: %w", relPath, err)
	}
	return data, nil
}

// AllPaths returns every regular embedded default file's path relative to
// the defaults root (e.g. "templates/epic-standalone.yaml"), sorted.
// Implements domain.FileLister — used by `strategist upgrade` to enumerate
// the full tree, since ReadFile alone only supports reading one
// already-known path.
func (e Extractor) AllPaths() ([]string, error) {
	var paths []string
	err := fs.WalkDir(defaultsFS, "defaults", func(path string, d fs.DirEntry, walkErr error) error {
		rel, ok, relErr := embeddedRelPath(path, "defaults", walkErr)
		if relErr != nil || !ok {
			return relErr
		}
		if d.IsDir() {
			return nil
		}
		paths = append(paths, rel)
		return nil
	})
	if err != nil {
		return nil, fmt.Errorf("embed: list defaults: %w", err)
	}
	sort.Strings(paths)
	return paths, nil
}

// DefaultsFS returns the embedded defaults tree rooted at defaults/, for
// callers that read embedded files in place (for example the runtime payload).
func DefaultsFS() fs.FS {
	sub, err := fs.Sub(defaultsFS, "defaults")
	if err != nil {
		// defaults is a compile-time embed root; Sub only fails on an invalid path.
		panic(fmt.Sprintf("embed: defaults root: %v", err))
	}
	return sub
}
