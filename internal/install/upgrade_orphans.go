package install

import (
	"fmt"
	"path/filepath"
	"sort"
	"strings"

	"github.com/SergioLacerda/strategist-skill/internal/domain"
	"github.com/SergioLacerda/strategist-skill/internal/runtimefs"
)

// orphanEntries reports every manifest-tracked path that is no longer part
// of the current embedded tree (embeddedHashes has no entry for it). An
// untouched file of the legacy id-only skills layout is reported as migratable.
func orphanEntries(strategistDir string, manifest domain.InstallManifest, manifestLoaded bool, embeddedHashes map[string]string) ([]UpgradePlanEntry, error) {
	if !manifestLoaded {
		return nil, nil
	}
	versioned := versionedSkillIDs(embeddedHashes)
	var orphans []UpgradePlanEntry
	for _, f := range manifest.Files {
		if _, stillEmbedded := embeddedHashes[f.Path]; stillEmbedded {
			continue
		}
		state, err := orphanState(strategistDir, f, versioned)
		if err != nil {
			return nil, err
		}
		orphans = append(orphans, UpgradePlanEntry{Path: f.Path, State: state})
	}
	return orphans, nil
}

// orphanState is UpgradeLegacyLayout for an untouched legacy-layout file of an
// id the versioned layout replaces, UpgradeOrphaned otherwise.
func orphanState(strategistDir string, file domain.InstallManifestFile, versioned map[string]bool) (domain.RuntimeFileUpgradeState, error) {
	if !isLegacyLayoutPath(file.Path, versioned) {
		return domain.UpgradeOrphaned, nil
	}
	runtimePath, err := runtimefs.SafeJoin(strategistDir, filepath.FromSlash(file.Path))
	if err != nil {
		return "", fmt.Errorf("upgrade: resolve %s: %w", file.Path, err)
	}
	currentHash, exists, err := runtimefs.ReadSHA256(runtimePath)
	if err != nil {
		return "", fmt.Errorf("upgrade: read %s: %w", file.Path, err)
	}
	if exists && currentHash == file.SHA256 {
		return domain.UpgradeLegacyLayout, nil
	}
	return domain.UpgradeOrphaned, nil
}

// versionedSkillIDs lists the ids that have a skills/<id>@<version>/ payload in
// the current embedded tree.
func versionedSkillIDs(embeddedHashes map[string]string) map[string]bool {
	ids := make(map[string]bool)
	for path := range embeddedHashes {
		parts := strings.SplitN(path, "/", 3)
		if len(parts) < 3 || parts[0] != "skills" {
			continue
		}
		if id, _, found := strings.Cut(parts[1], "@"); found {
			ids[id] = true
		}
	}
	return ids
}

// isLegacyLayoutPath reports whether path is a file of the pre-ADR-0061
// skills/<id>/ layout for an id that now has a versioned payload. The id-only
// skill.yaml at the directory root is the compat view the installer owns, so it
// is never part of the migration.
func isLegacyLayoutPath(path string, versioned map[string]bool) bool {
	parts := strings.SplitN(path, "/", 3)
	if len(parts) < 3 || parts[0] != "skills" || strings.Contains(parts[1], "@") || !versioned[parts[1]] {
		return false
	}
	return parts[2] != "skill.yaml"
}

// lockSlotsNeedingMigration lists the slots whose persisted Ranked binding has
// no weapon_version. A missing lock needs none.
func lockSlotsNeedingMigration(strategistDir string) ([]string, error) {
	lockFile, err := readPluginLockFile(strategistDir)
	if err != nil {
		return nil, fmt.Errorf("upgrade: read lock: %w", err)
	}
	var slots []string
	for _, binding := range lockFile.Bindings {
		if binding.EffectiveMode() == domain.SlotBindingModeRanked && binding.WeaponVersion == "" {
			slots = append(slots, binding.Slot)
		}
	}
	sort.Strings(slots)
	return slots, nil
}
